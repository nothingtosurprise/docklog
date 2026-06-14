package services

import (
	"docklog/db"
	"docklog/models"
	"docklog/repositories"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestShouldNotifyAudit(t *testing.T) {
	svc := NewNotificationService(repositories.NewNotificationRepository())
	svc.mu.Lock()
	svc.prefs = models.NotificationPreferences{Enabled: true}
	svc.channels = []models.NotificationChannel{
		{
			ChannelType: models.ChannelTypeSlack,
			Enabled:     true,
			ConfigJSON: `{"webhook_url":"https://hooks.slack.com/services/test","notify_container_actions":"true","notify_security_events":"true","notify_admin_actions":"false"}`,
		},
	}
	svc.mu.Unlock()

	event := models.AuditNotificationEvent{Action: "restart", Status: "Success"}
	if !svc.shouldNotify(event) {
		t.Fatal("expected container success to notify")
	}
	if svc.shouldNotify(models.AuditNotificationEvent{Action: "ping", Status: "Success"}) {
		t.Fatal("unexpected action should not notify")
	}

	svc.mu.Lock()
	svc.channels[0].ConfigJSON = `{"webhook_url":"https://hooks.slack.com/services/test","notify_container_actions":"false","notify_security_events":"true","notify_admin_actions":"false"}`
	svc.mu.Unlock()
	if svc.shouldNotify(models.AuditNotificationEvent{Action: "stop", Status: "Success"}) {
		t.Fatal("container actions disabled for slack channel")
	}
	if !svc.shouldNotify(models.AuditNotificationEvent{Action: "stop", Status: "Forbidden"}) {
		t.Fatal("security events should still notify on slack channel")
	}
}

func TestPerChannelEventRouting(t *testing.T) {
	events := models.NotificationChannelEvents{
		NotifyContainerActions: false,
		NotifySecurityEvents:   false,
		NotifyAdminActions:     true,
	}
	if !eventMatchesChannel(events, models.AuditNotificationEvent{Action: "reset_password", Status: "Success"}) {
		t.Fatal("expected admin action to match")
	}
	if eventMatchesChannel(events, models.AuditNotificationEvent{Action: "restart", Status: "Success"}) {
		t.Fatal("container action should not match")
	}
}

func TestParseChannelConfigBoolJSON(t *testing.T) {
	config, err := parseChannelConfig(`{"webhook_url":"https://hooks.slack.com/services/test","notify_container_actions":true}`)
	if err != nil {
		t.Fatalf("parse bool json: %v", err)
	}
	if config["notify_container_actions"] != "true" {
		t.Fatalf("expected string true, got %q", config["notify_container_actions"])
	}
}

func TestDispatchAuditEventContainerRestart(t *testing.T) {
	dir := t.TempDir()
	if err := db.InitDB(filepath.Join(dir, "dispatch.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}

	var received atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := repositories.NewNotificationRepository()
	if err := repo.SaveEnabled(true); err != nil {
		t.Fatalf("save enabled: %v", err)
	}
	if err := repo.UpsertChannel(models.ChannelTypeSlack, "Slack", true, map[string]string{
		"webhook_url":              server.URL,
		"notify_container_actions": "true",
	}); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}

	svc := NewNotificationService(repo)
	if err := svc.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	svc.DispatchAuditEvent(models.AuditNotificationEvent{
		Username: "alice",
		Action:   "restart",
		Resource: "api-server",
		Status:   "Success",
		Message:  "Restarted by alice via DockLog",
	})

	time.Sleep(100 * time.Millisecond)
	if !received.Load() {
		t.Fatal("expected webhook delivery for container restart")
	}
}

func TestArmDeliveryIfConfigured(t *testing.T) {
	dir := t.TempDir()
	if err := db.InitDB(filepath.Join(dir, "arm.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}

	repo := repositories.NewNotificationRepository()
	if err := repo.UpsertChannel(models.ChannelTypeTeams, "Microsoft Teams", true, map[string]string{
		"webhook_url":              "https://example.webhook.office.com/webhook",
		"notify_container_actions": "true",
		"notify_security_events":   "true",
		"notify_admin_actions":     "false",
	}); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}

	svc := NewNotificationService(repo)
	svc.Initialize()

	prefs, err := repo.LoadPreferences()
	if err != nil {
		t.Fatalf("load prefs: %v", err)
	}
	if !prefs.Enabled {
		t.Fatal("expected delivery to auto-enable for configured channel")
	}
}

func TestTestNotificationRequiresDeliveryEnabled(t *testing.T) {
	dir := t.TempDir()
	if err := db.InitDB(filepath.Join(dir, "test.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := repositories.NewNotificationRepository()
	if err := repo.UpsertChannel(models.ChannelTypeSlack, "Slack", true, map[string]string{
		"webhook_url":              server.URL,
		"notify_container_actions": "true",
	}); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}

	svc := NewNotificationService(repo)
	if err := svc.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	err := svc.TestNotification(models.NotificationTestRequest{Target: "slack"})
	if err == nil || !strings.Contains(err.Error(), "Delivery") {
		t.Fatalf("expected delivery error, got %v", err)
	}
}

func TestTestNotificationRequiresContainerEvents(t *testing.T) {
	dir := t.TempDir()
	if err := db.InitDB(filepath.Join(dir, "test.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := repositories.NewNotificationRepository()
	if err := repo.SaveEnabled(true); err != nil {
		t.Fatalf("save enabled: %v", err)
	}
	if err := repo.UpsertChannel(models.ChannelTypeSlack, "Slack", true, map[string]string{
		"webhook_url":              server.URL,
		"notify_container_actions": "false",
		"notify_security_events":   "true",
		"notify_admin_actions":     "false",
	}); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}

	svc := NewNotificationService(repo)
	if err := svc.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	err := svc.TestNotification(models.NotificationTestRequest{Target: "slack"})
	if err == nil || !strings.Contains(err.Error(), "Container actions") {
		t.Fatalf("expected container actions error, got %v", err)
	}
}

func TestSlackWebhookDelivery(t *testing.T) {
	svc := NewNotificationService(repositories.NewNotificationRepository())
	var received atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if payload["text"] == nil {
			t.Fatal("expected slack text field")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := svc.deliver(models.ChannelTypeSlack, map[string]string{"webhook_url": server.URL}, models.AuditNotificationEvent{
		Username: "admin", Action: "restart", Resource: "api-server",
		Status: "Success", Message: "Action executed successfully.",
	})
	if err != nil {
		t.Fatalf("delivery failed: %v", err)
	}
	if !received.Load() {
		t.Fatal("expected slack webhook to be called")
	}
}

func TestTeamsWebhookDelivery(t *testing.T) {
	svc := NewNotificationService(repositories.NewNotificationRepository())
	var received atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if payload["@type"] != "MessageCard" {
			t.Fatalf("expected MessageCard, got %v", payload["@type"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := svc.deliver(models.ChannelTypeTeams, map[string]string{"webhook_url": server.URL}, models.AuditNotificationEvent{
		Username: "dev", Action: "remove", Resource: "db",
		Status: "Forbidden", Message: "Access denied",
	})
	if err != nil {
		t.Fatalf("delivery failed: %v", err)
	}
	if !received.Load() {
		t.Fatal("expected teams webhook to be called")
	}
}

func TestDiscordWebhookDelivery(t *testing.T) {
	svc := NewNotificationService(repositories.NewNotificationRepository())
	var received atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := svc.deliver(models.ChannelTypeDiscord, map[string]string{"webhook_url": server.URL}, models.AuditNotificationEvent{
		Username: "admin", Action: "TEST", Resource: "notifications",
		Status: "Success", Message: "Test message",
	})
	if err != nil {
		t.Fatalf("delivery failed: %v", err)
	}
	if !received.Load() {
		t.Fatal("expected discord webhook to be called")
	}
}

func TestValidateDeliveryReady(t *testing.T) {
	svc := NewNotificationService(repositories.NewNotificationRepository())
	svc.mu.Lock()
	svc.prefs = models.NotificationPreferences{Enabled: true}
	svc.channels = []models.NotificationChannel{
		{ChannelType: models.ChannelTypeSlack, Enabled: true, ConfigJSON: `{"webhook_url":"https://hooks.slack.com/services/test"}`},
	}
	svc.mu.Unlock()

	if err := svc.validateDeliveryReady(true); err != nil {
		t.Fatalf("expected valid delivery, got %v", err)
	}

	svc.mu.Lock()
	svc.channels = []models.NotificationChannel{
		{ChannelType: models.ChannelTypeSlack, Enabled: true, ConfigJSON: `{}`},
	}
	svc.mu.Unlock()
	if err := svc.validateDeliveryReady(true); err == nil {
		t.Fatal("expected error when no webhook configured")
	}
}

func TestMergeChannelUpdate(t *testing.T) {
	svc := NewNotificationService(repositories.NewNotificationRepository())
	existing := &models.NotificationChannel{
		ChannelType: models.ChannelTypeSlack,
		ConfigJSON:  `{"webhook_url":"https://hooks.slack.com/services/old"}`,
	}

	config, shouldDelete, err := svc.mergeChannelUpdate(existing, models.NotificationChannelUpdate{
		Type: models.ChannelTypeSlack, Enabled: true,
		Events: models.NotificationChannelEvents{
			NotifyContainerActions: true,
			NotifySecurityEvents:   true,
			NotifyAdminActions:     false,
		},
	})
	if err != nil || shouldDelete || config["webhook_url"] != "https://hooks.slack.com/services/old" {
		t.Fatalf("expected to keep existing config, got %+v delete=%v err=%v", config, shouldDelete, err)
	}
	if config["notify_admin_actions"] != "false" {
		t.Fatalf("expected per-channel events in config, got %+v", config)
	}

	_, shouldDelete, err = svc.mergeChannelUpdate(existing, models.NotificationChannelUpdate{
		Type: models.ChannelTypeSlack, Clear: true,
	})
	if err != nil || !shouldDelete {
		t.Fatalf("expected delete, got delete=%v err=%v", shouldDelete, err)
	}

	_, _, err = svc.mergeChannelUpdate(nil, models.NotificationChannelUpdate{
		Type: models.ChannelTypeSlack,
		Config: map[string]string{"webhook_url": "https://hooks.slack.com/services/..."},
		Events: models.NotificationChannelEvents{
			NotifyContainerActions: true,
			NotifySecurityEvents:   true,
			NotifyAdminActions:     true,
		},
	})
	if err == nil {
		t.Fatal("expected placeholder webhook error")
	}

	_, _, err = svc.mergeChannelUpdate(nil, models.NotificationChannelUpdate{
		Type: models.ChannelTypeSlack,
		Config: map[string]string{"webhook_url": "http://insecure.example/hook"},
		Events: models.NotificationChannelEvents{
			NotifyContainerActions: true,
			NotifySecurityEvents:   true,
			NotifyAdminActions:     true,
		},
	})
	if err == nil {
		t.Fatal("expected invalid webhook error")
	}
}
