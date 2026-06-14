package db

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"
)

const (
	NotificationSettingsRowID = 1
)

// Notification channel types stored in notification_channels.channel_type.
const (
	ChannelTypeSlack   = "slack"
	ChannelTypeTeams   = "teams"
	ChannelTypeDiscord = "discord"
	ChannelTypeEmail   = "email"
)

func notificationSchemaSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS notification_settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		enabled BOOLEAN NOT NULL DEFAULT 0,
		notify_container_actions BOOLEAN NOT NULL DEFAULT 1,
		notify_security_events BOOLEAN NOT NULL DEFAULT 1,
		notify_admin_actions BOOLEAN NOT NULL DEFAULT 1,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		slack_webhook_url TEXT NOT NULL DEFAULT '',
		teams_webhook_url TEXT NOT NULL DEFAULT ''
	);
	INSERT OR IGNORE INTO notification_settings (id) VALUES (1);
	CREATE TABLE IF NOT EXISTS notification_channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_type TEXT NOT NULL,
		name TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		config TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(channel_type)
	);
	CREATE INDEX IF NOT EXISTS idx_notification_channels_type ON notification_channels(channel_type);
	CREATE INDEX IF NOT EXISTS idx_notification_channels_enabled ON notification_channels(enabled);
	`
}

func migrateNotificationSchema() error {
	if _, err := DB.Exec(notificationSchemaSQL()); err != nil {
		return err
	}

	legacyColumns := []struct {
		table  string
		column string
		def    string
	}{
		{"notification_settings", "slack_webhook_url", "TEXT NOT NULL DEFAULT ''"},
		{"notification_settings", "teams_webhook_url", "TEXT NOT NULL DEFAULT ''"},
		{"notification_settings", "updated_at", "DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP"},
		{"notification_settings", "delivery_user_set", "BOOLEAN NOT NULL DEFAULT 0"},
	}

	for _, item := range legacyColumns {
		if tableHasColumn(item.table, item.column) {
			continue
		}
		_, err := DB.Exec("ALTER TABLE " + item.table + " ADD COLUMN " + item.column + " " + item.def)
		if err != nil {
			log.Printf("Notification migration note (safe to ignore): %v", err)
		}
	}

	_, err := DB.Exec(`INSERT OR IGNORE INTO notification_settings (id) VALUES (1)`)
	return err
}

func tableHasColumn(tableName, columnName string) bool {
	switch tableName {
	case "notification_settings", "notification_channels":
	default:
		return false
	}

	rows, err := DB.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false
		}
		if name == columnName {
			return true
		}
	}
	return false
}

// MigrateNotificationChannelsFromLegacy copies deprecated webhook columns on
// notification_settings into notification_channels when the channels table is empty.
func MigrateNotificationChannelsFromLegacy() error {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM notification_channels`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if !tableHasColumn("notification_settings", "slack_webhook_url") {
		return nil
	}

	var slackURL, teamsURL sql.NullString
	err := DB.QueryRow(`
		SELECT slack_webhook_url, teams_webhook_url
		FROM notification_settings WHERE id = 1`,
	).Scan(&slackURL, &teamsURL)
	if err != nil {
		return err
	}

	migrated := false
	if url := strings.TrimSpace(slackURL.String); url != "" {
		if err := UpsertNotificationChannel(ChannelTypeSlack, "Slack", true, map[string]string{"webhook_url": url}); err != nil {
			return err
		}
		migrated = true
	}
	if url := strings.TrimSpace(teamsURL.String); url != "" {
		if err := UpsertNotificationChannel(ChannelTypeTeams, "Microsoft Teams", true, map[string]string{"webhook_url": url}); err != nil {
			return err
		}
		migrated = true
	}

	if migrated {
		_, err = DB.Exec(`
			UPDATE notification_settings
			SET slack_webhook_url = '', teams_webhook_url = '', updated_at = CURRENT_TIMESTAMP
			WHERE id = 1`)
		if err != nil {
			return err
		}
	}

	return nil
}

func UpsertNotificationChannel(channelType, name string, enabled bool, config map[string]string) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		INSERT INTO notification_channels (channel_type, name, enabled, config, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_type) DO UPDATE SET
			name = excluded.name,
			enabled = excluded.enabled,
			config = excluded.config,
			updated_at = CURRENT_TIMESTAMP`,
		channelType, name, enabled, string(configJSON),
	)
	return err
}

// NotificationTablesExist reports whether notification tables are present.
func NotificationTablesExist() (bool, error) {
	var settingsCount, channelsCount int
	if err := DB.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'notification_settings'`,
	).Scan(&settingsCount); err != nil {
		return false, err
	}
	if err := DB.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'notification_channels'`,
	).Scan(&channelsCount); err != nil {
		return false, err
	}
	return settingsCount > 0 && channelsCount > 0, nil
}
