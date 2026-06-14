package db

import (
	"path/filepath"
	"testing"
)

func TestNotificationSchemaFreshDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fresh.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	ok, err := NotificationTablesExist()
	if err != nil || !ok {
		t.Fatalf("expected notification tables, ok=%v err=%v", ok, err)
	}

	for _, column := range []string{
		"enabled",
		"notify_container_actions",
		"notify_security_events",
		"notify_admin_actions",
		"updated_at",
		"delivery_user_set",
	} {
		if !tableHasColumn("notification_settings", column) {
			t.Fatalf("missing notification_settings column %q", column)
		}
	}

	for _, column := range []string{
		"channel_type",
		"name",
		"enabled",
		"config",
		"created_at",
		"updated_at",
	} {
		if !tableHasColumn("notification_channels", column) {
			t.Fatalf("missing notification_channels column %q", column)
		}
	}

	var settingsID int
	if err := DB.QueryRow(`SELECT id FROM notification_settings WHERE id = 1`).Scan(&settingsID); err != nil {
		t.Fatalf("expected default settings row: %v", err)
	}

	if err := UpsertNotificationChannel(ChannelTypeDiscord, "Discord", true, map[string]string{
		"webhook_url": "https://discord.com/api/webhooks/test/token",
	}); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}

	var channelType, config string
	if err := DB.QueryRow(`SELECT channel_type, config FROM notification_channels WHERE channel_type = ?`, ChannelTypeDiscord).
		Scan(&channelType, &config); err != nil {
		t.Fatalf("read channel: %v", err)
	}
	if channelType != ChannelTypeDiscord || config == "" {
		t.Fatalf("unexpected channel row: type=%q config=%q", channelType, config)
	}
}

func TestNotificationSchemaLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "legacy.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	_, err := DB.Exec(`
		UPDATE notification_settings
		SET slack_webhook_url = ?, teams_webhook_url = ?
		WHERE id = 1`,
		"https://hooks.slack.com/services/legacy",
		"https://outlook.office.com/webhook/legacy",
	)
	if err != nil {
		t.Fatalf("seed legacy urls: %v", err)
	}

	if err := MigrateNotificationChannelsFromLegacy(); err != nil {
		t.Fatalf("migrate legacy: %v", err)
	}

	var channelCount int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM notification_channels`).Scan(&channelCount); err != nil {
		t.Fatalf("count channels: %v", err)
	}
	if channelCount != 2 {
		t.Fatalf("expected 2 migrated channels, got %d", channelCount)
	}

	var slackURL, teamsURL string
	if err := DB.QueryRow(`SELECT slack_webhook_url, teams_webhook_url FROM notification_settings WHERE id = 1`).
		Scan(&slackURL, &teamsURL); err != nil {
		t.Fatalf("read legacy columns: %v", err)
	}
	if slackURL != "" || teamsURL != "" {
		t.Fatalf("expected legacy webhook columns cleared, got slack=%q teams=%q", slackURL, teamsURL)
	}

	// Second run should be a no-op.
	if err := MigrateNotificationChannelsFromLegacy(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if err := DB.QueryRow(`SELECT COUNT(*) FROM notification_channels`).Scan(&channelCount); err != nil {
		t.Fatalf("count channels after second migrate: %v", err)
	}
	if channelCount != 2 {
		t.Fatalf("expected channel count unchanged, got %d", channelCount)
	}
}

func TestNotificationSchemaUpgradeFromOldSettingsTable(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "old.db")

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	_, err := DB.Exec(`DROP TABLE notification_channels`)
	if err != nil {
		t.Fatalf("drop channels table: %v", err)
	}

	if err := migrateNotificationSchema(); err != nil {
		t.Fatalf("remigrate schema: %v", err)
	}

	ok, err := NotificationTablesExist()
	if err != nil || !ok {
		t.Fatalf("expected channels table recreated, ok=%v err=%v", ok, err)
	}
}
