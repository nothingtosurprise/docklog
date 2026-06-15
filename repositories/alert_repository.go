package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"docklog/db"
	"docklog/models"
)

type AlertRepository struct{}

func NewAlertRepository() *AlertRepository {
	return &AlertRepository{}
}

func (r *AlertRepository) ListRules() ([]models.AlertRule, error) {
	rows, err := db.DB.Query(`
		SELECT id, rule_key, name, description, severity, enabled, source_type,
			config, scope, channel_ids, cooldown_minutes, max_per_hour,
			group_window_minutes, recovery_enabled, is_template, created_at, updated_at
		FROM alert_rules
		ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]models.AlertRule, 0)
	for rows.Next() {
		rule, err := scanAlertRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *AlertRepository) ListEnabledRules() ([]models.AlertRule, error) {
	rows, err := db.DB.Query(`
		SELECT id, rule_key, name, description, severity, enabled, source_type,
			config, scope, channel_ids, cooldown_minutes, max_per_hour,
			group_window_minutes, recovery_enabled, is_template, created_at, updated_at
		FROM alert_rules
		WHERE enabled = 1
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]models.AlertRule, 0)
	for rows.Next() {
		rule, err := scanAlertRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *AlertRepository) GetRule(id int64) (models.AlertRule, error) {
	row := db.DB.QueryRow(`
		SELECT id, rule_key, name, description, severity, enabled, source_type,
			config, scope, channel_ids, cooldown_minutes, max_per_hour,
			group_window_minutes, recovery_enabled, is_template, created_at, updated_at
		FROM alert_rules WHERE id = ?`, id)
	return scanAlertRuleRow(row)
}

func (r *AlertRepository) GetRuleByKey(ruleKey string) (models.AlertRule, error) {
	row := db.DB.QueryRow(`
		SELECT id, rule_key, name, description, severity, enabled, source_type,
			config, scope, channel_ids, cooldown_minutes, max_per_hour,
			group_window_minutes, recovery_enabled, is_template, created_at, updated_at
		FROM alert_rules WHERE rule_key = ?`, ruleKey)
	return scanAlertRuleRow(row)
}

func (r *AlertRepository) UpsertRule(rule models.AlertRule) (int64, error) {
	if rule.ID > 0 {
		_, err := db.DB.Exec(`
			UPDATE alert_rules SET
				rule_key = ?, name = ?, description = ?, severity = ?, enabled = ?,
				source_type = ?, config = ?, scope = ?, channel_ids = ?,
				cooldown_minutes = ?, max_per_hour = ?, group_window_minutes = ?,
				recovery_enabled = ?, is_template = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			rule.RuleKey, rule.Name, rule.Description, rule.Severity, rule.Enabled,
			rule.SourceType, rule.ConfigJSON, rule.ScopeJSON, rule.ChannelIDsJSON,
			rule.CooldownMinutes, rule.MaxPerHour, rule.GroupWindowMinutes,
			rule.RecoveryEnabled, rule.IsTemplate, rule.ID,
		)
		return rule.ID, err
	}

	result, err := db.DB.Exec(`
		INSERT INTO alert_rules (
			rule_key, name, description, severity, enabled, source_type,
			config, scope, channel_ids, cooldown_minutes, max_per_hour,
			group_window_minutes, recovery_enabled, is_template
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.RuleKey, rule.Name, rule.Description, rule.Severity, rule.Enabled,
		rule.SourceType, rule.ConfigJSON, rule.ScopeJSON, rule.ChannelIDsJSON,
		rule.CooldownMinutes, rule.MaxPerHour, rule.GroupWindowMinutes,
		rule.RecoveryEnabled, rule.IsTemplate,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *AlertRepository) DeleteRule(id int64) error {
	_, err := db.DB.Exec(`DELETE FROM alert_rules WHERE id = ?`, id)
	return err
}

func (r *AlertRepository) CountRules() (int, error) {
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM alert_rules`).Scan(&count)
	return count, err
}

func (r *AlertRepository) InsertHistory(entry models.AlertHistoryEntry) (int64, error) {
	metadata := entry.MetadataJSON
	if metadata == "" {
		metadata = "{}"
	}
	result, err := db.DB.Exec(`
		INSERT INTO alert_history (
			rule_key, rule_name, severity, container, host, source, message, status, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.RuleKey, entry.RuleName, entry.Severity, entry.Container, entry.Host,
		entry.Source, entry.Message, entry.Status, metadata,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *AlertRepository) ListHistory(limit int) ([]models.AlertHistoryEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.DB.Query(`
		SELECT id, rule_key, rule_name, severity, container, host, source, message, status, metadata, created_at
		FROM alert_history
		ORDER BY created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.AlertHistoryEntry, 0)
	for rows.Next() {
		var item models.AlertHistoryEntry
		if err := rows.Scan(
			&item.ID, &item.RuleKey, &item.RuleName, &item.Severity, &item.Container,
			&item.Host, &item.Source, &item.Message, &item.Status, &item.MetadataJSON, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(item.MetadataJSON), &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *AlertRepository) InsertDelivery(entry models.AlertDeliveryEntry) (int64, error) {
	result, err := db.DB.Exec(`
		INSERT INTO alert_deliveries (alert_history_id, channel_id, channel_type, status, attempts, last_error)
		VALUES (?, ?, ?, ?, ?, ?)`,
		entry.AlertHistoryID, entry.ChannelID, entry.ChannelType, entry.Status, entry.Attempts, entry.LastError,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *AlertRepository) UpdateDelivery(id int64, status, lastError string, attempts int) error {
	sentAt := ""
	if status == models.DeliveryStatusSent {
		sentAt = "CURRENT_TIMESTAMP"
	}
	if sentAt != "" {
		_, err := db.DB.Exec(`
			UPDATE alert_deliveries
			SET status = ?, attempts = ?, last_error = ?, sent_at = CURRENT_TIMESTAMP
			WHERE id = ?`, status, attempts, lastError, id)
		return err
	}
	_, err := db.DB.Exec(`
		UPDATE alert_deliveries
		SET status = ?, attempts = ?, last_error = ?
		WHERE id = ?`, status, attempts, lastError, id)
	return err
}

func (r *AlertRepository) ListDeliveriesForHistory(historyID int64) ([]models.AlertDeliveryEntry, error) {
	rows, err := db.DB.Query(`
		SELECT id, alert_history_id, channel_id, channel_type, status, attempts, last_error,
			COALESCE(sent_at, ''), created_at
		FROM alert_deliveries
		WHERE alert_history_id = ?
		ORDER BY id ASC`, historyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.AlertDeliveryEntry, 0)
	for rows.Next() {
		var item models.AlertDeliveryEntry
		if err := rows.Scan(
			&item.ID, &item.AlertHistoryID, &item.ChannelID, &item.ChannelType,
			&item.Status, &item.Attempts, &item.LastError, &item.SentAt, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanAlertRule(rows *sql.Rows) (models.AlertRule, error) {
	var rule models.AlertRule
	err := rows.Scan(
		&rule.ID, &rule.RuleKey, &rule.Name, &rule.Description, &rule.Severity, &rule.Enabled,
		&rule.SourceType, &rule.ConfigJSON, &rule.ScopeJSON, &rule.ChannelIDsJSON,
		&rule.CooldownMinutes, &rule.MaxPerHour, &rule.GroupWindowMinutes,
		&rule.RecoveryEnabled, &rule.IsTemplate, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
}

func scanAlertRuleRow(row *sql.Row) (models.AlertRule, error) {
	var rule models.AlertRule
	err := row.Scan(
		&rule.ID, &rule.RuleKey, &rule.Name, &rule.Description, &rule.Severity, &rule.Enabled,
		&rule.SourceType, &rule.ConfigJSON, &rule.ScopeJSON, &rule.ChannelIDsJSON,
		&rule.CooldownMinutes, &rule.MaxPerHour, &rule.GroupWindowMinutes,
		&rule.RecoveryEnabled, &rule.IsTemplate, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return rule, fmt.Errorf("alert rule not found")
	}
	return rule, err
}

func ParseChannelIDs(raw string) []int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var ids []int64
	_ = json.Unmarshal([]byte(raw), &ids)
	return ids
}

func EncodeChannelIDs(ids []int64) string {
	if len(ids) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(ids)
	return string(data)
}
