package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"docklog/db"
	"docklog/models"
	"docklog/repositories"
	"docklog/services"

	"github.com/labstack/echo/v4"
)

func setupNotificationController(t *testing.T) (*echo.Echo, *NotificationController) {
	t.Helper()
	dir := t.TempDir()
	if err := db.InitDB(filepath.Join(dir, "controller-test.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}

	svc := services.NewNotificationService(repositories.NewNotificationRepository())
	audited := false
	controller := NewNotificationController(
		svc,
		func(userID int, username, action, resource, status, message string) {
			audited = true
			if action != "UPDATE_NOTIFICATIONS" {
				t.Fatalf("unexpected audit action: %s", action)
			}
		},
		func(c echo.Context) (SessionUser, error) {
			return SessionUser{ID: 1, Username: "admin"}, nil
		},
	)

	e := echo.New()
	admin := e.Group("/api/admin")
	controller.RegisterRoutes(admin)
	t.Cleanup(func() { _ = audited })
	return e, controller
}

func TestNotificationControllerGetSettings(t *testing.T) {
	e, _ := setupNotificationController(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/notifications", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response models.NotificationsPublicResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.ChannelTypes) < 4 {
		t.Fatalf("expected channel catalog in response")
	}
}

func TestNotificationControllerUpdateSettings(t *testing.T) {
	e, _ := setupNotificationController(t)

	body := `{
		"enabled": true,
		"channels": [{
			"type": "slack",
			"enabled": true,
			"config": { "webhook_url": "https://hooks.slack.com/services/test-channel/placeholder" },
			"events": {
				"notify_container_actions": true,
				"notify_security_events": true,
				"notify_admin_actions": false
			}
		}]
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/notifications", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response models.NotificationsPublicResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Enabled || len(response.Channels) != 1 || !response.Channels[0].Configured {
		t.Fatalf("unexpected saved response: %+v", response)
	}
}

func TestNotificationControllerUpdateSettingsValidationError(t *testing.T) {
	e, _ := setupNotificationController(t)

	body := `{
		"enabled": true,
		"channels": [{
			"type": "slack",
			"enabled": true,
			"config": { "webhook_url": "http://insecure.example/hook" },
			"events": {
				"notify_container_actions": true,
				"notify_security_events": true,
				"notify_admin_actions": true
			}
		}]
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/notifications", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNotificationControllerTestSettingsMissingChannel(t *testing.T) {
	e, _ := setupNotificationController(t)

	body := `{"target":"all"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/notifications/test", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
