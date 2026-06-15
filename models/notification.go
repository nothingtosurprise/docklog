package models

const (
	ChannelTypeSlack   = "slack"
	ChannelTypeTeams   = "teams"
	ChannelTypeDiscord = "discord"
	ChannelTypeCustom  = "custom"
	ChannelTypeEmail   = "email"
)

type NotificationPreferences struct {
	Enabled bool `json:"enabled"`
	// Legacy global event flags remain in DB for migration; event routing is per-channel.
	NotifyContainerActions bool `json:"notify_container_actions,omitempty"`
	NotifySecurityEvents   bool `json:"notify_security_events,omitempty"`
	NotifyAdminActions     bool `json:"notify_admin_actions,omitempty"`
}

type NotificationChannelEvents struct {
	NotifyContainerActions bool `json:"notify_container_actions"`
	NotifySecurityEvents   bool `json:"notify_security_events"`
	NotifyAdminActions     bool `json:"notify_admin_actions"`
	NotifyHealthEvents     bool `json:"notify_health_events"`
	NotifyAlertEvents      bool `json:"notify_alert_events"`
	NotifyVersionUpdates   bool `json:"notify_version_updates"`
}

func (e NotificationChannelEvents) AnyEnabled() bool {
	return e.NotifyContainerActions || e.NotifySecurityEvents || e.NotifyAdminActions || e.NotifyHealthEvents || e.NotifyAlertEvents || e.NotifyVersionUpdates
}

type NotificationChannel struct {
	ID          int64  `json:"id"`
	ChannelType string `json:"type"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	ConfigJSON  string `json:"-"`
}

type NotificationChannelField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Secret      bool   `json:"secret"`
	Placeholder string `json:"placeholder"`
}

type NotificationEventTypeInfo struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type NotificationChannelTypeInfo struct {
	Type         string                     `json:"type"`
	Label        string                     `json:"label"`
	Description  string                     `json:"description"`
	Available    bool                       `json:"available"`
	ConfigFields []NotificationChannelField `json:"config_fields"`
}

type NotificationChannelPublic struct {
	ID           int64                     `json:"id"`
	Type         string                    `json:"type"`
	Name         string                    `json:"name"`
	Enabled      bool                      `json:"enabled"`
	Configured   bool                      `json:"configured"`
	ConfigMasked map[string]string         `json:"config_masked"`
	Events       NotificationChannelEvents `json:"events"`
}

type NotificationsPublicResponse struct {
	NotificationPreferences
	ChannelTypes []NotificationChannelTypeInfo `json:"channel_types"`
	EventTypes   []NotificationEventTypeInfo   `json:"event_types"`
	Channels     []NotificationChannelPublic   `json:"channels"`
}

type NotificationChannelUpdate struct {
	Type    string                    `json:"type"`
	Enabled bool                      `json:"enabled"`
	Config  map[string]string         `json:"config"`
	Events  NotificationChannelEvents `json:"events"`
	Clear   bool                      `json:"clear"`
}

type NotificationsUpdateRequest struct {
	Enabled  bool                        `json:"enabled"`
	Channels []NotificationChannelUpdate `json:"channels"`
}

type AuditNotificationEvent struct {
	UserID   int
	Username string
	Action   string
	Resource string
	Status   string
	Message  string
}

type NotificationTestRequest struct {
	Target    string `json:"target"`
	ChannelID int64  `json:"channel_id"`
}
