package server

import (
	"net/http"
	"strconv"

	"docklog/db"
	"docklog/stats"

	"github.com/labstack/echo/v4"
)

func (s *Server) registerSystemRoutes(r *echo.Group) {
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
		defer rows.Close()
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
