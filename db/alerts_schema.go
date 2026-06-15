package db

import "log"

func migrateAlertsSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS alert_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_key TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		severity TEXT NOT NULL DEFAULT 'warning',
		enabled BOOLEAN NOT NULL DEFAULT 0,
		source_type TEXT NOT NULL,
		config TEXT NOT NULL DEFAULT '{}',
		scope TEXT NOT NULL DEFAULT '{"type":"all"}',
		channel_ids TEXT NOT NULL DEFAULT '[]',
		cooldown_minutes INTEGER NOT NULL DEFAULT 15,
		max_per_hour INTEGER NOT NULL DEFAULT 20,
		group_window_minutes INTEGER NOT NULL DEFAULT 5,
		recovery_enabled BOOLEAN NOT NULL DEFAULT 0,
		is_template BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);
	CREATE INDEX IF NOT EXISTS idx_alert_rules_source ON alert_rules(source_type);
	CREATE TABLE IF NOT EXISTS alert_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_key TEXT NOT NULL,
		rule_name TEXT NOT NULL DEFAULT '',
		severity TEXT NOT NULL,
		container TEXT NOT NULL DEFAULT '',
		host TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL,
		message TEXT NOT NULL,
		status TEXT NOT NULL,
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_alert_history_time ON alert_history(created_at);
	CREATE INDEX IF NOT EXISTS idx_alert_history_rule ON alert_history(rule_key);
	CREATE TABLE IF NOT EXISTS alert_deliveries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alert_history_id INTEGER NOT NULL,
		channel_id INTEGER NOT NULL,
		channel_type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		attempts INTEGER NOT NULL DEFAULT 0,
		last_error TEXT NOT NULL DEFAULT '',
		sent_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (alert_history_id) REFERENCES alert_history(id)
	);
	CREATE INDEX IF NOT EXISTS idx_alert_deliveries_history ON alert_deliveries(alert_history_id);
	`
	if _, err := DB.Exec(schema); err != nil {
		return err
	}
	return nil
}

func AlertsTablesExist() (bool, error) {
	var count int
	if err := DB.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'alert_rules'`,
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func logAlertsMigrationNote(err error) {
	if err != nil {
		log.Printf("Alerts migration note (safe to ignore): %v", err)
	}
}
