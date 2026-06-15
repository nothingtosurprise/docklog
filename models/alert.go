package models

import "time"

const (
	AlertSeverityInfo     = "info"
	AlertSeverityWarning  = "warning"
	AlertSeverityCritical = "critical"

	AlertSourceLogs    = "logs"
	AlertSourceEvents  = "events"
	AlertSourceMetrics = "metrics"

	AlertScopeAll      = "all"
	AlertScopeNames    = "names"
	AlertScopePatterns = "patterns"
	AlertScopeLabels   = "labels"

	AlertStatusFired      = "fired"
	AlertStatusSuppressed = "suppressed"
	AlertStatusRecovered  = "recovered"
	AlertStatusDelivered  = "delivered"
	AlertStatusFailed     = "failed"

	DeliveryStatusPending = "pending"
	DeliveryStatusSent    = "sent"
	DeliveryStatusFailed  = "failed"
)

type LogPattern struct {
	Pattern string `json:"pattern"`
	Exclude bool   `json:"exclude,omitempty"`
	Regex   bool   `json:"regex,omitempty"`
}

type LogAlertConfig struct {
	Patterns      []LogPattern `json:"patterns"`
	MatchCount    int          `json:"match_count"`
	WindowSeconds int          `json:"window_seconds"`
	CaseSensitive bool         `json:"case_sensitive"`
}

type EventAlertConfig struct {
	Events         []string `json:"events"`
	MinOccurrences int      `json:"min_occurrences"`
	WindowSeconds  int      `json:"window_seconds"`
}

type MetricAlertConfig struct {
	Metric          string  `json:"metric"`
	Operator        string  `json:"operator"`
	Threshold       float64 `json:"threshold"`
	DurationMinutes int     `json:"duration_minutes"`
}

type AlertScope struct {
	Type       string            `json:"type"`
	Containers []string          `json:"containers,omitempty"`
	Patterns   []string          `json:"patterns,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type AlertRule struct {
	ID                 int64  `json:"id"`
	RuleKey            string `json:"rule_id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Severity           string `json:"severity"`
	Enabled            bool   `json:"enabled"`
	SourceType         string `json:"source_type"`
	ConfigJSON         string `json:"-"`
	ScopeJSON          string `json:"-"`
	ChannelIDsJSON     string `json:"-"`
	CooldownMinutes    int    `json:"cooldown_minutes"`
	MaxPerHour         int    `json:"max_per_hour"`
	GroupWindowMinutes int    `json:"group_window_minutes"`
	RecoveryEnabled    bool   `json:"recovery_enabled"`
	IsTemplate         bool   `json:"is_template"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type AlertRulePublic struct {
	AlertRule
	Config     interface{} `json:"config"`
	Scope      AlertScope  `json:"scope"`
	ChannelIDs []int64     `json:"channel_ids"`
}

type AlertRuleUpsert struct {
	RuleKey            string      `json:"rule_id"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	Severity           string      `json:"severity"`
	Enabled            bool        `json:"enabled"`
	SourceType         string      `json:"source_type"`
	Config             interface{} `json:"config"`
	Scope              AlertScope  `json:"scope"`
	ChannelIDs         []int64     `json:"channel_ids"`
	CooldownMinutes    int         `json:"cooldown_minutes"`
	MaxPerHour         int         `json:"max_per_hour"`
	GroupWindowMinutes int         `json:"group_window_minutes"`
	RecoveryEnabled    bool        `json:"recovery_enabled"`
}

type AlertNotification struct {
	RuleID    string                 `json:"ruleId"`
	Severity  string                 `json:"severity"`
	Container string                 `json:"container"`
	Host      string                 `json:"host"`
	Source    string                 `json:"source"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	URL       string                 `json:"url"`
	Recovery  bool                   `json:"recovery,omitempty"`
}

type AlertHistoryEntry struct {
	ID           int64  `json:"id"`
	RuleKey      string `json:"rule_id"`
	RuleName     string `json:"rule_name,omitempty"`
	Severity     string `json:"severity"`
	Container    string `json:"container"`
	Host         string `json:"host"`
	Source       string `json:"source"`
	Message      string `json:"message"`
	Status       string `json:"status"`
	MetadataJSON string `json:"-"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    string `json:"timestamp"`
}

type AlertDeliveryEntry struct {
	ID             int64  `json:"id"`
	AlertHistoryID int64  `json:"alert_history_id"`
	ChannelID      int64  `json:"channel_id"`
	ChannelType    string `json:"channel_type"`
	Status         string `json:"status"`
	Attempts       int    `json:"attempts"`
	LastError      string `json:"last_error,omitempty"`
	SentAt         string `json:"sent_at,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type AlertTemplate struct {
	RuleKey     string `json:"rule_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	SourceType  string `json:"source_type"`
}

type AlertsPublicResponse struct {
	Rules     []AlertRulePublic  `json:"rules"`
	Templates []AlertTemplate    `json:"templates"`
	History   []AlertHistoryEntry `json:"history,omitempty"`
}

type AlertTestRequest struct {
	RuleID int64 `json:"rule_id"`
}
