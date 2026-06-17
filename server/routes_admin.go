package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"docklog/controllers"
	"docklog/db"
	"docklog/middleware"
	"docklog/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) registerAdminRoutes(r *echo.Group) {
	admin := r.Group("/admin")
	admin.Use(s.adminMiddleware())

	admin.GET("/users", func(c echo.Context) error {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}
		limit := 10
		offset := (page - 1) * limit

		var total int
		db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)

		rows, err := db.DB.Query("SELECT id, username, is_admin, can_start, can_stop, can_restart, can_delete, can_shell, is_restricted_access, allowed_containers, is_active FROM users LIMIT ? OFFSET ?", limit, offset)
		if err != nil {
			log.Printf("Failed to query users: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch users: " + err.Error()})
		}
		defer rows.Close()
		users := make([]map[string]interface{}, 0)
		for rows.Next() {
			var id int
			var username, allowedContainers string
			var isAdmin, canStart, canStop, canRestart, canDelete, canShell, isRestricted, isActive bool
			if err := rows.Scan(&id, &username, &isAdmin, &canStart, &canStop, &canRestart, &canDelete, &canShell, &isRestricted, &allowedContainers, &isActive); err != nil {
				log.Printf("Failed to scan user row: %v", err)
				continue
			}
			users = append(users, map[string]interface{}{
				"id":                   id,
				"username":             username,
				"is_admin":             isAdmin,
				"can_start":            canStart,
				"can_stop":             canStop,
				"can_restart":          canRestart,
				"can_delete":           canDelete,
				"can_shell":            canShell,
				"is_restricted_access": isRestricted,
				"allowed_containers":   allowedContainers,
				"is_active":            isActive,
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"users": users,
			"total": total,
			"page":  page,
			"pages": (total + limit - 1) / limit,
		})
	})

	admin.PUT("/users/:id/active", func(c echo.Context) error {
		id := c.Param("id")
		isActive := c.FormValue("is_active") == "true"
		_, err := db.DB.Exec("UPDATE users SET is_active = ? WHERE id = ? AND is_admin = 0", isActive, id)
		if err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	admin.POST("/users", func(c echo.Context) error {
		username := db.TrimUsername(c.FormValue("username"))
		password := c.FormValue("password")
		if username == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Username is required"})
		}

		taken, err := db.UsernameTaken(username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to validate username"})
		}
		if taken {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "User already exists"})
		}

		if !middleware.IsPasswordStrongEnough(password) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Password must be at least %d characters", middleware.MinPasswordLength)})
		}
		canStart, canStop, canRestart, canDelete, canShell := middleware.ClampStaffActionPermissions(
			c.FormValue("can_start") == "true",
			c.FormValue("can_stop") == "true",
			c.FormValue("can_restart") == "true",
			c.FormValue("can_delete") == "true",
			c.FormValue("can_shell") == "true",
		)
		isRestricted := c.FormValue("is_restricted_access") == "true"
		allowedContainers := c.FormValue("allowed_containers")
		if allowedContainers == "" {
			allowedContainers = ".*"
		}

		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password is too long or invalid. Please use a shorter password."})
		}
		_, err = db.DB.Exec("INSERT INTO users (username, password, is_admin, can_start, can_stop, can_restart, can_delete, can_shell, is_restricted_access, allowed_containers, password_changed, is_active) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			username, string(h), false, canStart, canStop, canRestart, canDelete, canShell, isRestricted, allowedContainers, true, true)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "User already exists"})
		}
		return c.NoContent(http.StatusCreated)
	})

	admin.PUT("/users/:id/permissions", func(c echo.Context) error {
		id := c.Param("id")
		canStart, canStop, canRestart, canDelete, canShell := middleware.ClampStaffActionPermissions(
			c.FormValue("can_start") == "true",
			c.FormValue("can_stop") == "true",
			c.FormValue("can_restart") == "true",
			c.FormValue("can_delete") == "true",
			c.FormValue("can_shell") == "true",
		)
		isRestricted := c.FormValue("is_restricted_access") == "true"
		allowedContainers := c.FormValue("allowed_containers")

		_, err := db.DB.Exec("UPDATE users SET can_start = ?, can_stop = ?, can_restart = ?, can_delete = ?, can_shell = ?, is_restricted_access = ?, allowed_containers = ? WHERE id = ?", canStart, canStop, canRestart, canDelete, canShell, isRestricted, allowedContainers, id)
		if err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	admin.PUT("/users/:id/password", func(c echo.Context) error {
		id := c.Param("id")
		newPassword := c.FormValue("password")
		if !middleware.IsPasswordStrongEnough(newPassword) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Password must be at least %d characters", middleware.MinPasswordLength)})
		}

		h, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password is too long or invalid. Please use a shorter password."})
		}
		_, err = db.DB.Exec("UPDATE users SET password = ?, password_changed = 1, password_version = COALESCE(password_version, 1) + 1 WHERE id = ?", string(h), id)
		if err != nil {
			return err
		}

		token := c.Get("user").(*jwt.Token)
		claims := token.Claims.(*models.UserClaims)
		s.audit(claims.ID, claims.Username, "RESET_PASSWORD", "User:"+id, "Success", "Administrator reset user password")

		return c.NoContent(http.StatusOK)
	})

	admin.DELETE("/users/:id", func(c echo.Context) error {
		id := c.Param("id")
		_, err := db.DB.Exec("DELETE FROM users WHERE id = ? AND is_admin = 0", id)
		if err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	admin.GET("/audit", func(c echo.Context) error {
		from := strings.TrimSpace(c.QueryParam("from"))
		to := strings.TrimSpace(c.QueryParam("to"))
		search := strings.TrimSpace(c.QueryParam("q"))

		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		if limit == 0 {
			limit, _ = strconv.Atoi(c.QueryParam("size"))
		}
		limit = normalizeAuditPageSize(limit)
		offset := (page - 1) * limit

		conditions := make([]string, 0, 3)
		args := make([]interface{}, 0, 6)
		if from != "" && to != "" {
			conditions = append(conditions, "timestamp BETWEEN ? AND ?")
			args = append(args, from, to)
		}
		if search != "" {
			like := "%" + search + "%"
			conditions = append(conditions, "(username LIKE ? OR action LIKE ? OR resource LIKE ? OR message LIKE ?)")
			args = append(args, like, like, like, like)
		}

		whereClause := ""
		if len(conditions) > 0 {
			whereClause = " WHERE " + strings.Join(conditions, " AND ")
		}

		var total int
		countQuery := "SELECT COUNT(*) FROM audit_logs" + whereClause
		if err := db.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to count audit logs: " + err.Error()})
		}

		query := "SELECT id, user_id, username, action, resource, status, message, timestamp FROM audit_logs" +
			whereClause + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
		queryArgs := append(append([]interface{}{}, args...), limit, offset)
		rows, err := db.DB.Query(query, queryArgs...)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch audit logs: " + err.Error()})
		}
		defer rows.Close()

		logs := make([]map[string]interface{}, 0)
		for rows.Next() {
			var id, userID int
			var username, action, resource, status, message, timestamp string
			if err := rows.Scan(&id, &userID, &username, &action, &resource, &status, &message, &timestamp); err != nil {
				continue
			}
			logs = append(logs, map[string]interface{}{
				"id":        id,
				"user_id":   userID,
				"username":  username,
				"action":    action,
				"resource":  resource,
				"status":    status,
				"message":   message,
				"timestamp": timestamp,
			})
		}

		pages := 0
		if total > 0 {
			pages = (total + limit - 1) / limit
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"logs":  logs,
			"total": total,
			"page":  page,
			"pages": pages,
			"limit": limit,
		})
	})

	s.registerNotificationRoutes(admin)
	s.registerAlertRoutes(admin)
}

func (s *Server) registerAlertRoutes(admin *echo.Group) {
	if s.deps.Alerts == nil {
		return
	}
	controller := controllers.NewAlertController(
		s.deps.Alerts,
		s.auditNotificationSettingsUpdated,
		s.resolveNotificationSessionUser,
	)
	controller.RegisterRoutes(admin)
}

func (s *Server) registerNotificationRoutes(admin *echo.Group) {
	if s.deps.Notifications == nil {
		return
	}

	controller := controllers.NewNotificationController(
		s.deps.Notifications,
		s.auditNotificationSettingsUpdated,
		s.resolveNotificationSessionUser,
	)
	controller.RegisterRoutes(admin)
}

func (s *Server) auditNotificationSettingsUpdated(userID int, username, action, resource, status, message string) {
	s.audit(userID, username, action, resource, status, message)
}

func (s *Server) resolveNotificationSessionUser(c echo.Context) (controllers.SessionUser, error) {
	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*models.UserClaims)
	return controllers.SessionUser{ID: claims.ID, Username: claims.Username}, nil
}

func normalizeAuditPageSize(limit int) int {
	switch limit {
	case 10, 25, 50, 100:
		return limit
	default:
		return 10
	}
}
