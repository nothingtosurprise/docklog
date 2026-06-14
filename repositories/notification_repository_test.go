package repositories

import (
	"docklog/db"
	"docklog/models"
	"path/filepath"
	"testing"
)

func setupNotificationTestDB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "repo-test.db")
	if err := db.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
}

func TestNotificationRepositoryPreferencesRoundTrip(t *testing.T) {
	setupNotificationTestDB(t)
	repo := NewNotificationRepository()

	prefs := models.NotificationPreferences{
		Enabled:                true,
		NotifyContainerActions: true,
		NotifySecurityEvents:   false,
		NotifyAdminActions:     true,
	}
	if err := repo.SavePreferences(prefs); err != nil {
		t.Fatalf("save prefs: %v", err)
	}

	loaded, err := repo.LoadPreferences()
	if err != nil {
		t.Fatalf("load prefs: %v", err)
	}
	if !loaded.Enabled || loaded.NotifySecurityEvents || !loaded.NotifyContainerActions {
		t.Fatalf("unexpected prefs: %+v", loaded)
	}
}

func TestNotificationRepositoryChannelCRUD(t *testing.T) {
	setupNotificationTestDB(t)
	repo := NewNotificationRepository()

	if err := repo.UpsertChannel(models.ChannelTypeSlack, "Slack", true, map[string]string{
		"webhook_url": "https://hooks.slack.com/services/test",
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	channels, err := repo.ListChannels()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(channels) != 1 || channels[0].ChannelType != models.ChannelTypeSlack {
		t.Fatalf("unexpected channels: %+v", channels)
	}

	count, err := repo.CountChannels()
	if err != nil || count != 1 {
		t.Fatalf("expected count 1, got %d err=%v", count, err)
	}

	if err := repo.DeleteChannel(models.ChannelTypeSlack); err != nil {
		t.Fatalf("delete: %v", err)
	}
	count, err = repo.CountChannels()
	if err != nil || count != 0 {
		t.Fatalf("expected count 0 after delete, got %d err=%v", count, err)
	}
}

func TestNotificationRepositoryLegacyMigration(t *testing.T) {
	setupNotificationTestDB(t)

	_, err := db.DB.Exec(`
		UPDATE notification_settings
		SET slack_webhook_url = ?, teams_webhook_url = ?
		WHERE id = 1`,
		"https://hooks.slack.com/services/legacy",
		"https://outlook.office.com/webhook/legacy",
	)
	if err != nil {
		t.Fatalf("seed legacy: %v", err)
	}

	repo := NewNotificationRepository()
	if err := repo.MigrateLegacyChannels(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	channels, err := repo.ListChannels()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("expected 2 migrated channels, got %d", len(channels))
	}
}
