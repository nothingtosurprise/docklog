package repositories

import (
	"docklog/db"
	"docklog/models"
)

type NotificationRepository struct{}

func NewNotificationRepository() *NotificationRepository {
	return &NotificationRepository{}
}

func (r *NotificationRepository) LoadPreferences() (models.NotificationPreferences, error) {
	var prefs models.NotificationPreferences
	err := db.DB.QueryRow(`
		SELECT enabled, notify_container_actions, notify_security_events, notify_admin_actions
		FROM notification_settings WHERE id = 1`,
	).Scan(&prefs.Enabled, &prefs.NotifyContainerActions, &prefs.NotifySecurityEvents, &prefs.NotifyAdminActions)
	return prefs, err
}

func (r *NotificationRepository) SaveEnabled(enabled bool) error {
	_, err := db.DB.Exec(`
		UPDATE notification_settings
		SET enabled = ?, delivery_user_set = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`, enabled)
	return err
}

func (r *NotificationRepository) ArmDelivery() error {
	_, err := db.DB.Exec(`
		UPDATE notification_settings
		SET enabled = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`)
	return err
}

func (r *NotificationRepository) DeliveryUserSet() (bool, error) {
	var userSet bool
	err := db.DB.QueryRow(`
		SELECT COALESCE(delivery_user_set, 0)
		FROM notification_settings WHERE id = 1`,
	).Scan(&userSet)
	return userSet, err
}

func (r *NotificationRepository) SavePreferences(prefs models.NotificationPreferences) error {
	_, err := db.DB.Exec(`
		UPDATE notification_settings SET
			enabled = ?, notify_container_actions = ?, notify_security_events = ?,
			notify_admin_actions = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1`,
		prefs.Enabled, prefs.NotifyContainerActions, prefs.NotifySecurityEvents, prefs.NotifyAdminActions,
	)
	return err
}

func (r *NotificationRepository) ListChannels() ([]models.NotificationChannel, error) {
	rows, err := db.DB.Query(`
		SELECT id, channel_type, name, enabled, config
		FROM notification_channels ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := make([]models.NotificationChannel, 0)
	for rows.Next() {
		var channel models.NotificationChannel
		if err := rows.Scan(&channel.ID, &channel.ChannelType, &channel.Name, &channel.Enabled, &channel.ConfigJSON); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func (r *NotificationRepository) UpsertChannel(channelType, name string, enabled bool, config map[string]string) error {
	return db.UpsertNotificationChannel(channelType, name, enabled, config)
}

func (r *NotificationRepository) DeleteChannel(channelType string) error {
	_, err := db.DB.Exec(`DELETE FROM notification_channels WHERE channel_type = ?`, channelType)
	return err
}

func (r *NotificationRepository) CountChannels() (int, error) {
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM notification_channels`).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MigrateLegacyChannels() error {
	return db.MigrateNotificationChannelsFromLegacy()
}
