package server

import (
	"fmt"
	"strings"

	"docklog/audit"
	"docklog/db"
	"docklog/models"
	"docklog/services"

	"github.com/moby/moby/client"
)

type AuditLogger func(userID int, username, action, resource, status, message string)

type Deps struct {
	Docker        *client.Client
	Notifications *services.NotificationService
	AuditLogger   AuditLogger
}

func (s *Server) audit(userID int, username, action, resource, status, message string) {
	if s.deps.AuditLogger != nil {
		s.deps.AuditLogger(userID, username, action, resource, status, message)
		return
	}
	audit.Log(userID, username, action, resource, status, message)
}

func (s *Server) auditActor(claims *models.UserClaims) string {
	if claims == nil {
		return "system"
	}
	if name := strings.TrimSpace(claims.Username); name != "" {
		return name
	}
	var name string
	if err := db.DB.QueryRow("SELECT username FROM users WHERE id = ?", claims.ID).Scan(&name); err == nil {
		if name = strings.TrimSpace(name); name != "" {
			return name
		}
	}
	return "system"
}

func containerActionDetail(action, actor string) string {
	verb := map[string]string{
		"start":   "Started",
		"stop":    "Stopped",
		"restart": "Restarted",
		"remove":  "Removed",
	}[action]
	if verb == "" {
		verb = "Action"
	}
	return fmt.Sprintf("%s by %s via DockLog", verb, actor)
}
