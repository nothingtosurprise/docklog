package controllers

import (
	"net/http"
	"strings"

	"docklog/models"
	"docklog/services"

	"github.com/labstack/echo/v4"
)

type AuditLogger func(userID int, username, action, resource, status, message string)

type SessionUser struct {
	ID       int
	Username string
}

type SessionResolver func(c echo.Context) (SessionUser, error)

type NotificationController struct {
	service        *services.NotificationService
	auditLogger    AuditLogger
	sessionResolver SessionResolver
}

func NewNotificationController(service *services.NotificationService, audit AuditLogger, session SessionResolver) *NotificationController {
	return &NotificationController{
		service:         service,
		auditLogger:     audit,
		sessionResolver: session,
	}
}

func (nc *NotificationController) RegisterRoutes(admin *echo.Group) {
	admin.GET("/notifications", nc.GetSettings)
	admin.PUT("/notifications", nc.UpdateSettings)
	admin.POST("/notifications/test", nc.TestSettings)
}

func (nc *NotificationController) GetSettings(c echo.Context) error {
	response, err := nc.service.GetPublicSettings()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load notification settings"})
	}
	return c.JSON(http.StatusOK, response)
}

func (nc *NotificationController) UpdateSettings(c echo.Context) error {
	user, err := nc.sessionResolver(c)
	if err != nil {
		return err
	}

	var update models.NotificationsUpdateRequest
	if err := c.Bind(&update); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	response, err := nc.service.UpdateSettings(update)
	if err != nil {
		if isInternalNotificationError(err) {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save notification settings"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if nc.auditLogger != nil {
		nc.auditLogger(user.ID, user.Username, "UPDATE_NOTIFICATIONS", "notifications", "Success", "Notification settings updated")
	}
	return c.JSON(http.StatusOK, response)
}

func (nc *NotificationController) TestSettings(c echo.Context) error {
	var req models.NotificationTestRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := nc.service.TestNotification(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Test notification sent"})
}

func isInternalNotificationError(err error) bool {
	msg := err.Error()
	return strings.HasPrefix(msg, "save preferences:") ||
		strings.HasPrefix(msg, "load channels:") ||
		strings.HasPrefix(msg, "remove channel:") ||
		strings.HasPrefix(msg, "save channel:") ||
		strings.HasPrefix(msg, "apply settings:")
}
