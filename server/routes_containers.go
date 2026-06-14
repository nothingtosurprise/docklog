package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"docklog/access"
	"docklog/audit"
	"docklog/config"
	"docklog/containers"
	"docklog/db"
	"docklog/dockerutil"
	"docklog/middleware"
	"docklog/models"
	"docklog/stats"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/moby/moby/client"
)

func (s *Server) registerContainerRoutes(r *echo.Group) {
	cli := s.deps.Docker

	r.GET("/containers", func(c echo.Context) error {
		token := c.Get("user").(*jwt.Token)
		user := token.Claims.(*models.UserClaims)

		var dbIsAdmin bool
		db.DB.QueryRow("SELECT COALESCE(is_admin, 0) FROM users WHERE id = ?", user.ID).Scan(&dbIsAdmin)

		res, err := cli.ContainerList(context.Background(), client.ContainerListOptions{All: true})
		if err != nil {
			return err
		}

		containerList := dockerutil.ExtractContainers(res)

		var patterns []string
		if !dbIsAdmin {
			patterns = access.GetAuthorizedPatterns(user.ID)
		}
		log.Printf("User %d (DB Admin: %v) authorized patterns: %v", user.ID, dbIsAdmin, patterns)

		var list []models.Container
		for _, ctr := range containerList {
			name := "unknown"
			names, _ := ctr["Names"].([]interface{})
			if len(names) > 0 {
				name = names[0].(string)[1:]
			}

			image, _ := ctr["Image"].(string)
			state, _ := ctr["State"].(string)

			id, ok := ctr["ID"].(string)
			if !ok {
				id, _ = ctr["Id"].(string)
			}

			if id == "" {
				continue
			}

			if containers.IsExcludedContainer(name, image) {
				continue
			}

			shortID := id
			if len(id) > 12 {
				shortID = id[:12]
			}

			visible := dbIsAdmin
			if !visible {
				for _, p := range patterns {
					if matched, _ := regexp.MatchString(p, name); matched {
						visible = true
						break
					}
				}
			}

			if visible {
				createdVal, _ := ctr["Created"].(float64)
				statusVal, _ := ctr["Status"].(string)

				inspect, _ := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
				cpuLimit := 0.0
				memLimit := int64(0)
				if inspect.Container.HostConfig != nil {
					if inspect.Container.HostConfig.NanoCPUs > 0 {
						cpuLimit = float64(inspect.Container.HostConfig.NanoCPUs) / 1e9
					}
					memLimit = inspect.Container.HostConfig.Memory
				}

				var lastCPU float64
				var lastMem int64
				db.DB.QueryRow("SELECT cpu, memory FROM stats WHERE container_id = ? ORDER BY timestamp DESC LIMIT 1", id).Scan(&lastCPU, &lastMem)

				list = append(list, models.Container{
					ID:       shortID,
					Name:     name,
					Image:    image,
					State:    state,
					Created:  int64(createdVal),
					Status:   statusVal,
					CPULimit: cpuLimit,
					MemLimit: memLimit,
					CPU:      lastCPU,
					Memory:   lastMem,
				})
			}
		}
		return c.JSON(http.StatusOK, list)
	})

	r.GET("/containers/:id/inspect", func(c echo.Context) error {
		id := c.Param("id")
		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		var dbIsAdmin bool
		db.DB.QueryRow("SELECT COALESCE(is_admin, 0) FROM users WHERE id = ?", userClaims.ID).Scan(&dbIsAdmin)

		container, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}

		if !dbIsAdmin {
			containerName := strings.TrimPrefix(container.Container.Name, "/")
			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, containerName); matched {
					authorized = true
					break
				}
			}
			if !authorized {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Access Denied"})
			}
		}

		containerName := strings.TrimPrefix(container.Container.Name, "/")
		containerImage := ""
		if container.Container.Config != nil {
			containerImage = container.Container.Config.Image
		}

		if containers.IsExcludedContainer(containerName, containerImage) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}

		if container.Container.Config != nil && len(container.Container.Config.Env) > 0 {
			container.Container.Config.Env = containers.SanitizeContainerEnv(container.Container.Config.Env)
		}

		return c.JSON(http.StatusOK, container)
	})

	r.POST("/containers/:id/action", func(c echo.Context) error {
		id := c.Param("id")
		action := c.FormValue("action")
		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		if action != "start" && action != "stop" && action != "restart" && action != "remove" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid action specified."})
		}

		if !middleware.ContainerActionEnvAllowed(action) {
			detail := "This action is disabled on this server."
			actor := s.auditActor(userClaims)
			s.audit(userClaims.ID, actor, action, id, "Forbidden", detail)
			return c.JSON(http.StatusForbidden, map[string]string{"error": detail})
		}

		target, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Target container not found."})
		}
		targetImage := ""
		if target.Container.Config != nil {
			targetImage = target.Container.Config.Image
		}
		if containers.InspectContainerExcluded(target.Container.Name, targetImage) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Target container not found."})
		}
		targetName := strings.TrimPrefix(target.Container.Name, "/")

		if !config.AuthDisabled {
			can, err := middleware.StaffHasContainerActionPermission(action, userClaims.ID)
			if err != nil || !can {
				detail := "This action is not permitted for this account."
				actor := s.auditActor(userClaims)
				s.audit(userClaims.ID, actor, action, targetName, "Forbidden", detail)
				return c.JSON(http.StatusForbidden, map[string]string{"error": detail})
			}
		}

		var dbIsAdmin bool
		db.DB.QueryRow("SELECT COALESCE(is_admin, 0) FROM users WHERE id = ?", userClaims.ID).Scan(&dbIsAdmin)
		if !dbIsAdmin {
			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, targetName); matched {
					authorized = true
					break
				}
			}

			if !authorized {
				actor := s.auditActor(userClaims)
				s.audit(userClaims.ID, actor, action, targetName, "Forbidden", "Security Restriction: Regex level rights missing.")
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Security Restriction: You are not authorized to interact with this container resource."})
			}
		}

		actor := s.auditActor(userClaims)
		ctx := context.Background()
		timeout := 60
		switch action {
		case "start":
			_, err = cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
		case "stop":
			_, err = cli.ContainerStop(ctx, id, client.ContainerStopOptions{Timeout: &timeout})
		case "restart":
			_, err = cli.ContainerRestart(ctx, id, client.ContainerRestartOptions{Timeout: &timeout})
		case "remove":
			_, err = cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
		}

		if err != nil {
			s.audit(userClaims.ID, actor, action, targetName, "Error", "System Error: "+err.Error())
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "System Error: " + err.Error()})
		}

		audit.SuppressContainerEvent(id, action, 8*time.Second)
		s.audit(userClaims.ID, actor, action, targetName, "Success", containerActionDetail(action, actor))
		return c.NoContent(http.StatusOK)
	})

	r.GET("/containers/:id/logs/download", func(c echo.Context) error {
		id := c.Param("id")
		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		container, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		downloadImage := ""
		if container.Container.Config != nil {
			downloadImage = container.Container.Config.Image
		}
		if containers.InspectContainerExcluded(container.Container.Name, downloadImage) {
			return c.NoContent(http.StatusNotFound)
		}
		containerName := strings.TrimPrefix(container.Container.Name, "/")

		if !userClaims.IsAdmin {
			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, containerName); matched {
					authorized = true
					break
				}
			}
			if !authorized {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission Denied: Download restricted for this resource."})
			}
		}

		options := client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Follow:     false,
		}

		out, err := cli.ContainerLogs(context.Background(), id, options)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch logs: " + err.Error()})
		}
		defer out.Close()

		s.audit(userClaims.ID, userClaims.Username, "DOWNLOAD_LOGS", id, "Success", "Full log archive exported")

		c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+id+"_full.log")
		c.Response().Header().Set(echo.HeaderContentType, "text/plain")
		c.Response().WriteHeader(http.StatusOK)

		_, err = io.Copy(c.Response().Writer, out)
		return err
	})

	r.GET("/containers/:id/logs", func(c echo.Context) error {
		id := c.Param("id")
		untilStr := c.QueryParam("until")

		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		container, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}
		logsImage := ""
		if container.Container.Config != nil {
			logsImage = container.Container.Config.Image
		}
		if containers.InspectContainerExcluded(container.Container.Name, logsImage) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}

		if !userClaims.IsAdmin {
			containerName := strings.TrimPrefix(container.Container.Name, "/")
			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, containerName); matched {
					authorized = true
					break
				}
			}
			if !authorized {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Access Denied"})
			}
		}

		options := client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Follow:     false,
		}

		out, err := cli.ContainerLogs(context.Background(), id, options)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer out.Close()

		var output bytes.Buffer
		if container.Container.Config.Tty {
			io.Copy(&output, out)
		} else {
			stdcopy.StdCopy(&output, &output, out)
		}

		allLines := strings.Split(output.String(), "\n")
		var logs []string

		if untilStr == "" {
			for _, line := range allLines {
				if line != "" {
					logs = append(logs, line)
				}
			}
			if len(logs) > 100 {
				logs = logs[len(logs)-100:]
			}
		} else {
			var untilTime time.Time
			untilTime, err = time.Parse(time.RFC3339Nano, untilStr)
			if err != nil {
				untilTime, err = time.Parse(time.RFC3339, untilStr)
				if err != nil {
					if unix, err := strconv.ParseInt(untilStr, 10, 64); err == nil {
						untilTime = time.Unix(unix, 0)
					}
				}
			}

			var filtered []string
			for _, line := range allLines {
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, " ", 2)
				if len(parts) > 0 {
					ts, err := time.Parse(time.RFC3339Nano, parts[0])
					if err != nil {
						ts, err = time.Parse(time.RFC3339, parts[0])
					}

					if err == nil {
						if !ts.After(untilTime) {
							filtered = append(filtered, line)
						}
					}
				}
			}

			if len(filtered) > 100 {
				logs = filtered[len(filtered)-100:]
			} else {
				logs = filtered
			}
		}

		log.Printf("[API] Found %d lines for %s (until: %s)", len(logs), id, untilStr)
		return c.JSON(http.StatusOK, logs)
	})

	r.GET("/containers/:id/logs/count", func(c echo.Context) error {
		id := c.Param("id")
		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		container, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}
		countImage := ""
		if container.Container.Config != nil {
			countImage = container.Container.Config.Image
		}
		if containers.InspectContainerExcluded(container.Container.Name, countImage) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}

		if !userClaims.IsAdmin {
			containerName := strings.TrimPrefix(container.Container.Name, "/")
			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, containerName); matched {
					authorized = true
					break
				}
			}
			if !authorized {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Access Denied"})
			}
		}

		options := client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     false,
		}

		out, err := cli.ContainerLogs(context.Background(), id, options)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer out.Close()

		var output bytes.Buffer
		if container.Container.Config.Tty {
			io.Copy(&output, out)
		} else {
			stdcopy.StdCopy(&output, &output, out)
		}

		count := strings.Count(output.String(), "\n")
		return c.JSON(http.StatusOK, map[string]int{"total": count})
	})

	r.GET("/containers/:id/stats", func(c echo.Context) error {
		id := c.Param("id")
		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		container, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		statsImage := ""
		if container.Container.Config != nil {
			statsImage = container.Container.Config.Image
		}
		if containers.InspectContainerExcluded(container.Container.Name, statsImage) {
			return c.NoContent(http.StatusNotFound)
		}
		containerName := strings.TrimPrefix(container.Container.Name, "/")

		if !userClaims.IsAdmin {
			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, containerName); matched {
					authorized = true
					break
				}
			}
			if !authorized {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized resource access."})
			}
		}

		containerStats, err := cli.ContainerStats(context.Background(), id, client.ContainerStatsOptions{Stream: true})
		if err != nil {
			return err
		}
		defer containerStats.Body.Close()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)

		enc := json.NewEncoder(c.Response())
		dec := json.NewDecoder(containerStats.Body)
		for {
			var data interface{}
			if err := dec.Decode(&data); err != nil {
				break
			}
			if err := enc.Encode(data); err != nil {
				break
			}
			c.Response().Flush()
		}
		return nil
	})

	r.GET("/containers/:id/stats-now", func(c echo.Context) error {
		id := c.Param("id")
		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		container, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		nowImage := ""
		if container.Container.Config != nil {
			nowImage = container.Container.Config.Image
		}
		if containers.InspectContainerExcluded(container.Container.Name, nowImage) {
			return c.NoContent(http.StatusNotFound)
		}
		containerName := strings.TrimPrefix(container.Container.Name, "/")

		if !userClaims.IsAdmin {
			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, containerName); matched {
					authorized = true
					break
				}
			}
			if !authorized {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized resource access."})
			}
		}

		s1, err := cli.ContainerStats(context.Background(), id, client.ContainerStatsOptions{Stream: false})
		if err != nil {
			return err
		}
		var v1 struct {
			CPUStats struct {
				CPUUsage struct {
					TotalUsage uint64 `json:"total_usage"`
				} `json:"cpu_usage"`
				SystemUsage uint64 `json:"system_cpu_usage"`
			} `json:"cpu_stats"`
		}
		json.NewDecoder(s1.Body).Decode(&v1)
		s1.Body.Close()

		time.Sleep(500 * time.Millisecond)

		s2, err := cli.ContainerStats(context.Background(), id, client.ContainerStatsOptions{Stream: false})
		if err != nil {
			return err
		}
		defer s2.Body.Close()

		var v2 struct {
			CPUStats struct {
				CPUUsage struct {
					TotalUsage uint64 `json:"total_usage"`
				} `json:"cpu_usage"`
				SystemUsage uint64 `json:"system_cpu_usage"`
				OnlineCPUs  uint32 `json:"online_cpus"`
			} `json:"cpu_stats"`
			MemoryStats struct {
				Usage uint64            `json:"usage"`
				Stats map[string]uint64 `json:"stats"`
			} `json:"memory_stats"`
		}
		if err := json.NewDecoder(s2.Body).Decode(&v2); err != nil {
			return err
		}

		cpuDelta := float64(v2.CPUStats.CPUUsage.TotalUsage) - float64(v1.CPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(v2.CPUStats.SystemUsage) - float64(v1.CPUStats.SystemUsage)

		onlineCPUs := float64(v2.CPUStats.OnlineCPUs)
		if onlineCPUs == 0 {
			onlineCPUs = float64(runtime.NumCPU())
		}

		cpuPercent := 0.0
		if systemDelta > 0 && cpuDelta > 0 {
			cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
		}
		memUsed := v2.MemoryStats.Usage - (v2.MemoryStats.Stats["cache"])

		return c.JSON(http.StatusOK, map[string]interface{}{
			"cpu":    cpuPercent,
			"memory": memUsed,
		})
	})

	r.GET("/containers/:id/history", func(c echo.Context) error {
		id := c.Param("id")
		token := c.Get("user").(*jwt.Token)
		userClaims := token.Claims.(*models.UserClaims)

		container, err := cli.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}
		historyImage := ""
		if container.Container.Config != nil {
			historyImage = container.Container.Config.Image
		}
		if containers.InspectContainerExcluded(container.Container.Name, historyImage) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Container not found"})
		}

		longID := container.Container.ID
		if longID == "" {
			longID = id
		}

		if !userClaims.IsAdmin {
			containerName := strings.TrimPrefix(container.Container.Name, "/")

			patterns := access.GetAuthorizedPatterns(userClaims.ID)
			authorized := false
			for _, p := range patterns {
				if matched, _ := regexp.MatchString(p, containerName); matched {
					authorized = true
					break
				}
			}
			if !authorized {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Unauthorized resource access."})
			}
		}

		duration := c.QueryParam("duration")
		from := c.QueryParam("from")
		to := c.QueryParam("to")

		query := "SELECT cpu, memory, timestamp FROM stats WHERE container_id = ? "
		var args []interface{}
		args = append(args, longID)

		if from != "" && to != "" {
			query += "AND timestamp BETWEEN ? AND ? "
			args = append(args, from, to)
		} else if duration == "1h" {
			query += "AND timestamp >= datetime('now', '-1 hour') "
		} else if duration == "24h" {
			query += "AND timestamp >= datetime('now', '-24 hours') "
		}

		query += "ORDER BY timestamp DESC LIMIT 200"

		rows, err := db.DB.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			var cpu float64
			var memory int64
			var timestamp string
			rows.Scan(&cpu, &memory, &timestamp)
			results = append(results, map[string]interface{}{
				"cpu":       cpu,
				"memory":    memory,
				"timestamp": timestamp,
			})
		}
		return c.JSON(http.StatusOK, results)
	})

	r.GET("/system/history", func(c echo.Context) error {
		daysStr := c.QueryParam("days")
		days := 30
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}

		rows, err := db.DB.Query("SELECT cpu, memory, timestamp FROM system_stats WHERE timestamp > datetime('now', '-' || ? || ' days') ORDER BY timestamp DESC", days)
		if err != nil {
			return err
		}
		var history []map[string]interface{}
		for rows.Next() {
			var cpu float64
			var mem int64
			var ts string
			rows.Scan(&cpu, &mem, &ts)
			history = append(history, map[string]interface{}{"cpu": cpu, "memory": mem, "timestamp": ts})
		}
		return c.JSON(http.StatusOK, history)
	})

	r.GET("/system/stats", func(c echo.Context) error {
		data, ok := stats.LatestSystemStats()
		if !ok {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Stats not ready"})
		}
		return c.JSON(http.StatusOK, data)
	})
}
