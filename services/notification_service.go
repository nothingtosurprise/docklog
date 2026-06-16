package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"docklog/config"
	"docklog/models"
	"docklog/repositories"
)

type NotificationService struct {
	repo       *repositories.NotificationRepository
	httpClient *http.Client

	mu       sync.RWMutex
	prefs    models.NotificationPreferences
	channels []models.NotificationChannel
}

func NewNotificationService(repo *repositories.NotificationRepository) *NotificationService {
	return &NotificationService{
		repo:       repo,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

var channelCatalog = []models.NotificationChannelTypeInfo{
	{
		Type: models.ChannelTypeSlack, Label: "Slack", Description: "Incoming webhook", Available: true,
		ConfigFields: []models.NotificationChannelField{
			{Key: "webhook_url", Label: "Webhook URL", Secret: true, Placeholder: "https://hooks.slack.com/services/..."},
		},
	},
	{
		Type: models.ChannelTypeTeams, Label: "Microsoft Teams", Description: "Incoming webhook", Available: true,
		ConfigFields: []models.NotificationChannelField{
			{Key: "webhook_url", Label: "Webhook URL", Secret: true, Placeholder: "https://outlook.office.com/webhook/..."},
		},
	},
	{
		Type: models.ChannelTypeDiscord, Label: "Discord", Description: "Webhook URL", Available: true,
		ConfigFields: []models.NotificationChannelField{
			{Key: "webhook_url", Label: "Webhook URL", Secret: true, Placeholder: "https://discord.com/api/webhooks/..."},
		},
	},
	{
		Type: models.ChannelTypeCustom, Label: "Custom webhook", Description: "HTTPS endpoint (JSON payload)", Available: true,
		ConfigFields: []models.NotificationChannelField{
			{Key: "display_name", Label: "Display name", Placeholder: "PagerDuty, n8n, Zapier, etc."},
			{Key: "webhook_url", Label: "Webhook URL", Secret: true, Placeholder: "https://your-service.example/hooks/docklog"},
		},
	},
	{
		Type: models.ChannelTypeEmail, Label: "Email", Description: "SMTP delivery (coming soon)", Available: false,
		ConfigFields: []models.NotificationChannelField{
			{Key: "smtp_host", Label: "SMTP host", Placeholder: "smtp.example.com"},
			{Key: "smtp_port", Label: "SMTP port", Placeholder: "587"},
			{Key: "smtp_user", Label: "SMTP username", Placeholder: "notifications@example.com"},
			{Key: "smtp_password", Label: "SMTP password", Secret: true},
			{Key: "from_address", Label: "From address", Placeholder: "docklog@example.com"},
			{Key: "to_addresses", Label: "Recipients", Placeholder: "ops@example.com"},
		},
	},
}

const (
	configKeyNotifyContainer = "notify_container_actions"
	configKeyNotifySecurity  = "notify_security_events"
	configKeyNotifyAdmin     = "notify_admin_actions"
	configKeyNotifyHealth    = "notify_health_events"
	configKeyNotifyAlerts    = "notify_alert_events"
	configKeyNotifyVersion   = "notify_version_updates"
)

func workloadActionsLabel() string {
	if config.DockerEnabled() && config.KubernetesEnabled() {
		return "Workload actions"
	}
	if config.KubernetesEnabled() {
		return "Pod actions"
	}
	return "Container actions"
}

func notificationEventCatalog() []models.NotificationEventTypeInfo {
	var actionLabel, actionDesc string
	if config.DockerEnabled() && config.KubernetesEnabled() {
		actionLabel = "Workload actions"
		actionDesc = "Successful start, stop, restart, and delete on containers and pods (DockLog UI, docker CLI, or kubectl)"
	} else if config.KubernetesEnabled() {
		actionLabel = "Pod actions"
		actionDesc = "Successful start, stop, restart, and delete (DockLog UI or kubectl)"
	} else {
		actionLabel = "Container actions"
		actionDesc = "Successful start, stop, restart, and delete (DockLog UI or docker CLI)"
	}

	var securityDesc string
	if config.DockerEnabled() && config.KubernetesEnabled() {
		securityDesc = "Blocked or failed container and pod actions"
	} else if config.KubernetesEnabled() {
		securityDesc = "Blocked or failed pod actions"
	} else {
		securityDesc = "Blocked or failed container actions"
	}

	var alertDesc string
	if config.DockerEnabled() && config.KubernetesEnabled() {
		alertDesc = "Rule-based alerts from logs, Docker/Kubernetes events, and metrics"
	} else if config.KubernetesEnabled() {
		alertDesc = "Rule-based alerts from logs, Kubernetes events, and metrics"
	} else {
		alertDesc = "Rule-based alerts from logs, Docker events, and metrics"
	}

	catalog := []models.NotificationEventTypeInfo{
		{
			Key: configKeyNotifyContainer, Label: actionLabel, Description: actionDesc,
		},
		{
			Key: configKeyNotifySecurity, Label: "Security events", Description: securityDesc,
		},
		{
			Key: configKeyNotifyAdmin, Label: "Admin actions",
			Description: "Password reset and log export",
		},
	}
	if config.DockerEnabled() {
		catalog = append(catalog, models.NotificationEventTypeInfo{
			Key: configKeyNotifyHealth, Label: "Health check alerts",
			Description: "Docker HEALTHCHECK failures and recovery",
		})
	}
	catalog = append(catalog,
		models.NotificationEventTypeInfo{
			Key: configKeyNotifyAlerts, Label: "Intelligent alerts", Description: alertDesc,
		},
		models.NotificationEventTypeInfo{
			Key: configKeyNotifyVersion, Label: "Version updates",
			Description: "A newer DockLog image version is available on Docker Hub",
		},
	)
	return catalog
}

func (s *NotificationService) Initialize() {
	if err := s.repo.MigrateLegacyChannels(); err != nil {
		log.Printf("Notifications: legacy migration failed: %v", err)
	}
	if err := s.Reload(); err != nil {
		log.Printf("Notifications: failed to load settings: %v", err)
		return
	}
	if err := s.armDeliveryIfConfigured(); err != nil {
		log.Printf("Notifications: delivery arm check failed: %v", err)
	}
	if err := s.Reload(); err != nil {
		log.Printf("Notifications: failed to reload settings: %v", err)
		return
	}
	prefs := s.preferences()
	if prefs.Enabled && len(s.activeChannels()) > 0 {
		log.Printf("Notifications: enabled (%d active channel(s))", len(s.activeChannels()))
	} else if s.hasRoutableChannels() && !prefs.Enabled {
		log.Printf("Notifications: channels are configured but Delivery is off; enable it in Admin -> Notifications")
	}
}

func (s *NotificationService) Reload() error {
	prefs, err := s.repo.LoadPreferences()
	if err != nil {
		return err
	}
	channels, err := s.repo.ListChannels()
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.prefs = prefs
	s.channels = channels
	s.mu.Unlock()
	return nil
}

func (s *NotificationService) GetPublicSettings() (models.NotificationsPublicResponse, error) {
	prefs, err := s.repo.LoadPreferences()
	if err != nil {
		return models.NotificationsPublicResponse{}, err
	}
	channels, err := s.repo.ListChannels()
	if err != nil {
		return models.NotificationsPublicResponse{}, err
	}
	return s.buildPublicResponse(prefs, channels), nil
}

func (s *NotificationService) UpdateSettings(req models.NotificationsUpdateRequest) (models.NotificationsPublicResponse, error) {
	if err := s.repo.SaveEnabled(req.Enabled); err != nil {
		return models.NotificationsPublicResponse{}, fmt.Errorf("save preferences: %w", err)
	}

	existingChannels, err := s.repo.ListChannels()
	if err != nil {
		return models.NotificationsPublicResponse{}, fmt.Errorf("load channels: %w", err)
	}
	existingByType := map[string]models.NotificationChannel{}
	for _, channel := range existingChannels {
		existingByType[channel.ChannelType] = channel
	}

	for _, channelUpdate := range req.Channels {
		var existing *models.NotificationChannel
		if record, ok := existingByType[channelUpdate.Type]; ok {
			copy := record
			existing = &copy
		}
		config, shouldDelete, err := s.mergeChannelUpdate(existing, channelUpdate)
		if err != nil {
			return models.NotificationsPublicResponse{}, err
		}
		if shouldDelete {
			if err := s.repo.DeleteChannel(channelUpdate.Type); err != nil {
				return models.NotificationsPublicResponse{}, fmt.Errorf("remove channel: %w", err)
			}
			continue
		}
		info, _ := s.channelTypeInfo(channelUpdate.Type)
		if channelUpdate.Enabled && s.channelIsConfigured(channelUpdate.Type, config) && !eventsFromConfig(config).AnyEnabled() {
			return models.NotificationsPublicResponse{}, fmt.Errorf("%s: select at least one event type", info.Label)
		}
		if err := s.repo.UpsertChannel(channelUpdate.Type, channelDisplayName(channelUpdate.Type, info, config), channelUpdate.Enabled, config); err != nil {
			return models.NotificationsPublicResponse{}, fmt.Errorf("save channel: %w", err)
		}
	}

	if err := s.Reload(); err != nil {
		return models.NotificationsPublicResponse{}, fmt.Errorf("apply settings: %w", err)
	}
	if err := s.validateDeliveryReady(req.Enabled); err != nil {
		return models.NotificationsPublicResponse{}, err
	}
	if err := s.validateChannelEventPreferences(req.Enabled); err != nil {
		return models.NotificationsPublicResponse{}, err
	}
	return s.GetPublicSettings()
}

func (s *NotificationService) TestNotification(req models.NotificationTestRequest) error {
	if err := s.Reload(); err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	if !s.deliveryEnabled() {
		return fmt.Errorf("turn on Delivery (Enable notifications) before testing live alerts")
	}

	channels, err := s.repo.ListChannels()
	if err != nil {
		return fmt.Errorf("load channels: %w", err)
	}

	testEvent := models.AuditNotificationEvent{
		Username: "admin",
		Action:   "restart",
		Resource: "example-container",
		Status:   "Success",
		Message:  "Restarted by admin via DockLog (test notification).",
	}

	target := strings.ToLower(strings.TrimSpace(req.Target))
	if target == "" {
		target = "all"
	}

	targets := s.resolveTestTargets(channels, target, req.ChannelID)
	ready := make([]models.NotificationChannel, 0, len(targets))
	for _, channel := range targets {
		if !channel.Enabled {
			continue
		}
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil || !s.channelIsConfigured(channel.ChannelType, config) {
			continue
		}
		ready = append(ready, channel)
	}
	if len(ready) == 0 {
		return fmt.Errorf("no configured notification channels to test. Save a webhook URL first")
	}

	var errs []string
	for _, channel := range ready {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid config", channel.ChannelType))
			continue
		}
		if !s.channelIsConfigured(channel.ChannelType, config) {
			info, _ := s.channelTypeInfo(channel.ChannelType)
			errs = append(errs, fmt.Sprintf("%s is not configured", info.Label))
			continue
		}
		if !eventMatchesChannel(eventsFromConfig(config), testEvent) {
			info, _ := s.channelTypeInfo(channel.ChannelType)
			errs = append(errs, fmt.Sprintf("%s: enable %s for start/stop/restart alerts", info.Label, workloadActionsLabel()))
			continue
		}
		if err := s.deliver(channel.ChannelType, config, testEvent); err != nil {
			info, _ := s.channelTypeInfo(channel.ChannelType)
			errs = append(errs, fmt.Sprintf("%s: %s", info.Label, err.Error()))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *NotificationService) DispatchAuditEvent(event models.AuditNotificationEvent) {
	if err := s.Reload(); err != nil {
		log.Printf("Notifications: reload failed: %v", err)
		return
	}
	if !s.deliveryEnabled() {
		config.Debugf("Notifications: delivery off; skipped %s on %s", event.Action, event.Resource)
		return
	}
	for _, channel := range s.activeChannels() {
		ch := channel
		go func() {
			config, err := parseChannelConfig(ch.ConfigJSON)
			if err != nil {
				log.Printf("%s notification failed: %v", ch.ChannelType, err)
				return
			}
			if !s.channelIsConfigured(ch.ChannelType, config) {
				return
			}
			if !eventMatchesChannel(eventsFromConfig(config), event) {
				return
			}
			if err := s.deliver(ch.ChannelType, config, event); err != nil {
				log.Printf("%s notification failed: %v", ch.ChannelType, err)
			}
		}()
	}
}

func (s *NotificationService) CLIConfigLines() []string {
	prefs, err := s.repo.LoadPreferences()
	if err != nil {
		return []string{"  notifications          unavailable"}
	}
	channels, err := s.repo.ListChannels()
	if err != nil {
		return []string{"  notifications          unavailable"}
	}

	lines := []string{fmt.Sprintf("  notifications_enabled  %t", prefs.Enabled)}
	active := 0
	for _, channel := range channels {
		config, _ := parseChannelConfig(channel.ConfigJSON)
		if s.channelIsConfigured(channel.ChannelType, config) {
			lines = append(lines, fmt.Sprintf("  %-22s configured", channel.ChannelType+":"))
			if channel.Enabled {
				active++
			}
		}
	}
	lines = append(lines, fmt.Sprintf("  active_channels        %d", active))
	return lines
}

func (s *NotificationService) validateDeliveryReady(enabled bool) error {
	if !enabled {
		return nil
	}
	for _, channel := range s.activeChannels() {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil {
			continue
		}
		if s.channelIsConfigured(channel.ChannelType, config) {
			return nil
		}
	}
	return fmt.Errorf("enable at least one notification channel with a webhook URL before turning notifications on")
}

func (s *NotificationService) validateChannelEventPreferences(enabled bool) error {
	if !enabled {
		return nil
	}
	for _, channel := range s.activeChannels() {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil || !s.channelIsConfigured(channel.ChannelType, config) {
			continue
		}
		if eventsFromConfig(config).AnyEnabled() {
			return nil
		}
	}
	return fmt.Errorf("enable at least one event type on an active notification channel")
}

func (s *NotificationService) preferences() models.NotificationPreferences {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.prefs
}

func (s *NotificationService) deliveryEnabled() bool {
	return s.preferences().Enabled
}

func (s *NotificationService) hasRoutableChannels() bool {
	for _, channel := range s.activeChannels() {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil || !s.channelIsConfigured(channel.ChannelType, config) {
			continue
		}
		if eventsFromConfig(config).AnyEnabled() {
			return true
		}
	}
	return false
}

func (s *NotificationService) armDeliveryIfConfigured() error {
	if s.deliveryEnabled() {
		return nil
	}
	userSet, err := s.repo.DeliveryUserSet()
	if err != nil {
		return err
	}
	if userSet || !s.hasRoutableChannels() {
		return nil
	}
	if err := s.repo.ArmDelivery(); err != nil {
		return err
	}
	log.Printf("Notifications: auto-enabled Delivery for configured channel(s)")
	return nil
}

func (s *NotificationService) activeChannels() []models.NotificationChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	active := make([]models.NotificationChannel, 0, len(s.channels))
	for _, channel := range s.channels {
		if channel.Enabled {
			active = append(active, channel)
		}
	}
	return active
}

func (s *NotificationService) ConfiguredChannelIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]int64, 0, len(s.channels))
	for _, channel := range s.channels {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil || !s.channelIsConfigured(channel.ChannelType, config) {
			continue
		}
		ids = append(ids, channel.ID)
	}
	return ids
}

func eventMatchesChannel(events models.NotificationChannelEvents, event models.AuditNotificationEvent) bool {
	action := strings.ToLower(strings.TrimSpace(event.Action))
	status := strings.TrimSpace(event.Status)
	containerActions := map[string]bool{"start": true, "stop": true, "restart": true, "remove": true}

	if containerActions[action] {
		if status == "Success" {
			return events.NotifyContainerActions
		}
		if status == "Forbidden" || status == "Error" {
			return events.NotifySecurityEvents
		}
		return false
	}
	if status == "Forbidden" {
		return events.NotifySecurityEvents
	}
	adminActions := map[string]bool{"reset_password": true, "download_logs": true}
	if adminActions[strings.ToLower(action)] {
		return events.NotifyAdminActions
	}
	if action == "version_update" {
		return events.NotifyVersionUpdates
	}
	healthActions := map[string]bool{"health_check_failed": true, "health_check_recovered": true}
	if healthActions[action] {
		return events.NotifyHealthEvents
	}
	return false
}

func (s *NotificationService) shouldNotify(event models.AuditNotificationEvent) bool {
	if !s.preferences().Enabled {
		return false
	}
	for _, channel := range s.activeChannels() {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil || !s.channelIsConfigured(channel.ChannelType, config) {
			continue
		}
		if eventMatchesChannel(eventsFromConfig(config), event) {
			return true
		}
	}
	return false
}

func (s *NotificationService) buildPublicResponse(prefs models.NotificationPreferences, channels []models.NotificationChannel) models.NotificationsPublicResponse {
	publicChannels := make([]models.NotificationChannelPublic, 0, len(channels))
	for _, channel := range channels {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil {
			continue
		}
		publicChannels = append(publicChannels, models.NotificationChannelPublic{
			ID: channel.ID, Type: channel.ChannelType, Name: channel.Name,
			Enabled: channel.Enabled, Configured: s.channelIsConfigured(channel.ChannelType, config),
			ConfigMasked: s.maskChannelConfig(channel.ChannelType, config),
			Events:       eventsFromConfig(config),
		})
	}
	return models.NotificationsPublicResponse{
		NotificationPreferences: models.NotificationPreferences{Enabled: prefs.Enabled},
		ChannelTypes:            channelCatalog,
		EventTypes:              notificationEventCatalog(),
		Channels:                publicChannels,
	}
}

func (s *NotificationService) channelTypeInfo(channelType string) (models.NotificationChannelTypeInfo, bool) {
	for _, item := range channelCatalog {
		if item.Type == channelType {
			return item, true
		}
	}
	return models.NotificationChannelTypeInfo{}, false
}

func (s *NotificationService) mergeChannelUpdate(existing *models.NotificationChannel, update models.NotificationChannelUpdate) (map[string]string, bool, error) {
	info, ok := s.channelTypeInfo(update.Type)
	if !ok {
		return nil, false, fmt.Errorf("unknown channel type %q", update.Type)
	}
	if !info.Available {
		return nil, false, fmt.Errorf("%s is not available yet", info.Label)
	}

	current := map[string]string{}
	if existing != nil {
		parsed, err := parseChannelConfig(existing.ConfigJSON)
		if err != nil {
			return nil, false, err
		}
		current = parsed
	}
	if update.Clear {
		return map[string]string{}, true, nil
	}

	next := map[string]string{}
	for key, value := range current {
		next[key] = value
	}
	for key, value := range update.Config {
		if value = strings.TrimSpace(value); value != "" {
			next[key] = value
		}
	}
	if err := s.validateChannelConfig(update.Type, next); err != nil {
		return nil, false, err
	}
	applyEventsToConfig(next, update.Events)
	if !s.channelIsConfigured(update.Type, next) {
		return nil, true, nil
	}
	return next, false, nil
}

func (s *NotificationService) validateChannelConfig(channelType string, config map[string]string) error {
	info, ok := s.channelTypeInfo(channelType)
	if !ok {
		return fmt.Errorf("unknown channel type %q", channelType)
	}
	if !info.Available {
		return fmt.Errorf("%s is not available yet", info.Label)
	}
	for _, field := range info.ConfigFields {
		if field.Key == "webhook_url" {
			if value := strings.TrimSpace(config[field.Key]); value != "" {
				if err := validateWebhookURLForChannel(channelType, value); err != nil {
					return fmt.Errorf("%s: %w", info.Label, err)
				}
			}
		}
	}
	return nil
}

func (s *NotificationService) channelIsConfigured(channelType string, config map[string]string) bool {
	if channelType == models.ChannelTypeCustom {
		return strings.TrimSpace(config["webhook_url"]) != ""
	}
	info, ok := s.channelTypeInfo(channelType)
	if !ok {
		return false
	}
	for _, field := range info.ConfigFields {
		if strings.TrimSpace(config[field.Key]) != "" {
			return true
		}
	}
	return false
}

func (s *NotificationService) maskChannelConfig(channelType string, config map[string]string) map[string]string {
	info, ok := s.channelTypeInfo(channelType)
	if !ok {
		return map[string]string{}
	}
	masked := map[string]string{}
	for _, field := range info.ConfigFields {
		value := strings.TrimSpace(config[field.Key])
		if value == "" {
			continue
		}
		if field.Secret {
			masked[field.Key] = maskSecretValue(value)
		} else {
			masked[field.Key] = value
		}
	}
	return masked
}

func (s *NotificationService) resolveTestTargets(channels []models.NotificationChannel, target string, channelID int64) []models.NotificationChannel {
	if channelID > 0 {
		for _, channel := range channels {
			if channel.ID == channelID {
				return []models.NotificationChannel{channel}
			}
		}
		return nil
	}
	targets := make([]models.NotificationChannel, 0)
	for _, channel := range channels {
		if target != "all" && channel.ChannelType != target {
			continue
		}
		targets = append(targets, channel)
	}
	return targets
}

func (s *NotificationService) deliver(channelType string, config map[string]string, event models.AuditNotificationEvent) error {
	webhookURL := strings.TrimSpace(config["webhook_url"])
	switch channelType {
	case models.ChannelTypeSlack:
		if webhookURL == "" {
			return fmt.Errorf("webhook URL is not configured")
		}
		return s.postWebhook(webhookURL, buildSlackPayload(event))
	case models.ChannelTypeTeams:
		if webhookURL == "" {
			return fmt.Errorf("webhook URL is not configured")
		}
		return s.postWebhook(webhookURL, buildTeamsPayload(event))
	case models.ChannelTypeDiscord:
		if webhookURL == "" {
			return fmt.Errorf("webhook URL is not configured")
		}
		return s.postWebhook(webhookURL, buildDiscordPayload(event))
	case models.ChannelTypeCustom:
		if webhookURL == "" {
			return fmt.Errorf("webhook URL is not configured")
		}
		return s.postWebhook(webhookURL, buildCustomPayload(event))
	default:
		return fmt.Errorf("channel type %q is not available yet", channelType)
	}
}

func (s *NotificationService) postWebhook(url string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func parseChannelConfig(configJSON string) (map[string]string, error) {
	config := map[string]string{}
	if strings.TrimSpace(configJSON) == "" {
		return config, nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &raw); err != nil {
		return nil, err
	}
	for key, value := range raw {
		switch typed := value.(type) {
		case string:
			config[key] = typed
		case bool:
			config[key] = boolConfigValue(typed)
		case float64:
			config[key] = strconv.FormatInt(int64(typed), 10)
		default:
			config[key] = fmt.Sprintf("%v", typed)
		}
	}
	return config, nil
}

func defaultChannelEvents() models.NotificationChannelEvents {
	return models.NotificationChannelEvents{
		NotifyContainerActions: true,
		NotifySecurityEvents:   false,
		NotifyAdminActions:     false,
		NotifyHealthEvents:     false,
		NotifyAlertEvents:      false,
		NotifyVersionUpdates:   false,
	}
}

func eventsFromConfig(config map[string]string) models.NotificationChannelEvents {
	events := defaultChannelEvents()
	if config == nil {
		return events
	}
	if value, ok := config[configKeyNotifyContainer]; ok {
		events.NotifyContainerActions = parseBoolConfig(value)
	}
	if value, ok := config[configKeyNotifySecurity]; ok {
		events.NotifySecurityEvents = parseBoolConfig(value)
	}
	if value, ok := config[configKeyNotifyAdmin]; ok {
		events.NotifyAdminActions = parseBoolConfig(value)
	}
	if value, ok := config[configKeyNotifyHealth]; ok {
		events.NotifyHealthEvents = parseBoolConfig(value)
	}
	if value, ok := config[configKeyNotifyAlerts]; ok {
		events.NotifyAlertEvents = parseBoolConfig(value)
	}
	if value, ok := config[configKeyNotifyVersion]; ok {
		events.NotifyVersionUpdates = parseBoolConfig(value)
	}
	return events
}

func applyEventsToConfig(config map[string]string, events models.NotificationChannelEvents) {
	config[configKeyNotifyContainer] = boolConfigValue(events.NotifyContainerActions)
	config[configKeyNotifySecurity] = boolConfigValue(events.NotifySecurityEvents)
	config[configKeyNotifyAdmin] = boolConfigValue(events.NotifyAdminActions)
	config[configKeyNotifyHealth] = boolConfigValue(events.NotifyHealthEvents)
	config[configKeyNotifyAlerts] = boolConfigValue(events.NotifyAlertEvents)
	config[configKeyNotifyVersion] = boolConfigValue(events.NotifyVersionUpdates)
}

func parseBoolConfig(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func boolConfigValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func maskSecretValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= 16 {
		return "••••••••"
	}
	return raw[:12] + "..." + raw[len(raw)-4:]
}

func validateWebhookURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "https://") {
		return fmt.Errorf("webhook URL must use https")
	}
	if len(raw) > 2048 {
		return fmt.Errorf("webhook URL is too long")
	}
	if strings.HasSuffix(raw, "...") || strings.Contains(raw, "/...") {
		return fmt.Errorf("webhook URL looks like a placeholder")
	}
	return nil
}

func validateWebhookURLForChannel(channelType, raw string) error {
	if err := validateWebhookURL(raw); err != nil {
		return err
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("webhook URL is invalid")
	}
	host := strings.ToLower(u.Hostname())
	switch channelType {
	case models.ChannelTypeSlack:
		if host != "hooks.slack.com" && host != "hooks.slack-gov.com" {
			return fmt.Errorf("expected a Slack hooks URL (hooks.slack.com)")
		}
	case models.ChannelTypeTeams:
		if host != "outlook.office.com" && !strings.HasSuffix(host, ".webhook.office.com") {
			return fmt.Errorf("expected a Microsoft Teams / Office 365 webhook URL")
		}
	case models.ChannelTypeDiscord:
		if host != "discord.com" && host != "discordapp.com" {
			return fmt.Errorf("expected a Discord webhook URL (discord.com)")
		}
	}
	return nil
}

func displayUsername(username string) string {
	if strings.TrimSpace(username) == "" {
		return "system"
	}
	return username
}

func containerActionVerb(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "start":
		return "started"
	case "stop":
		return "stopped"
	case "restart":
		return "restarted"
	case "remove":
		return "removed"
	default:
		return strings.ToLower(strings.TrimSpace(action))
	}
}

func isContainerAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "start", "stop", "restart", "remove":
		return true
	default:
		return false
	}
}

func notificationPresentation(event models.AuditNotificationEvent) (title, color string) {
	user := displayUsername(event.Username)
	action := strings.ToLower(strings.TrimSpace(event.Action))
	switch action {
	case "health_check_failed":
		title = fmt.Sprintf("Health check failed: %s", event.Resource)
	case "health_check_recovered":
		title = fmt.Sprintf("Health check recovered: %s", event.Resource)
	default:
		if isContainerAction(action) {
			title = fmt.Sprintf("%s %s %s", user, containerActionVerb(action), event.Resource)
		} else {
			title = fmt.Sprintf("%s: %s", event.Action, event.Resource)
		}
	}
	switch event.Status {
	case "Success":
		color = "10B981"
		if !isContainerAction(action) && action != "health_check_recovered" {
			title = fmt.Sprintf("%s by %s", title, user)
		}
	case "Forbidden":
		color = "F59E0B"
		if action == "health_check_failed" || action == "health_check_recovered" {
			title = fmt.Sprintf("Blocked: %s", title)
		} else {
			title = fmt.Sprintf("Blocked: %s (%s)", title, user)
		}
	case "Error":
		color = "EF4444"
		if action == "health_check_failed" {
			// title already set, e.g. "Health check failed: api-server"
		} else {
			title = fmt.Sprintf("Failed: %s (%s)", title, user)
		}
	default:
		color = "0891B2"
	}
	return title, color
}

func buildSlackPayload(event models.AuditNotificationEvent) map[string]interface{} {
	title, color := notificationPresentation(event)
	text := fmt.Sprintf("*%s*\nUser: *%s*\nAction: `%s`\nResource: `%s`\nStatus: *%s*\n%s",
		title, displayUsername(event.Username), event.Action, event.Resource, event.Status, event.Message)
	return map[string]interface{}{
		"text": fmt.Sprintf("DockLog: %s", title),
		"attachments": []map[string]interface{}{{
			"color": color, "title": title, "text": text, "footer": "DockLog",
			"ts": time.Now().Unix(), "mrkdwn_in": []string{"text", "title"},
		}},
	}
}

func buildTeamsPayload(event models.AuditNotificationEvent) map[string]interface{} {
	title, themeColor := notificationPresentation(event)
	return map[string]interface{}{
		"@type": "MessageCard", "@context": "https://schema.org/extensions",
		"summary": title, "themeColor": themeColor, "title": "DockLog",
		"sections": []map[string]interface{}{{
			"activityTitle": title,
			"facts": []map[string]string{
				{"name": "User", "value": displayUsername(event.Username)},
				{"name": "Action", "value": event.Action},
				{"name": "Resource", "value": event.Resource},
				{"name": "Status", "value": event.Status},
				{"name": "Details", "value": event.Message},
			},
		}},
	}
}

func buildDiscordPayload(event models.AuditNotificationEvent) map[string]interface{} {
	title, color := notificationPresentation(event)
	colorInt, _ := strconv.ParseInt(color, 16, 64)
	desc := fmt.Sprintf("User: %s\nAction: %s\nResource: %s\nStatus: %s\n%s",
		displayUsername(event.Username), event.Action, event.Resource, event.Status, event.Message)
	return map[string]interface{}{
		"content": fmt.Sprintf("DockLog: %s", title),
		"embeds": []map[string]interface{}{{
			"title": title, "description": desc, "color": colorInt,
		}},
	}
}

func channelDisplayName(channelType string, info models.NotificationChannelTypeInfo, config map[string]string) string {
	if channelType == models.ChannelTypeCustom {
		if name := strings.TrimSpace(config["display_name"]); name != "" {
			return name
		}
	}
	return info.Label
}

func buildCustomPayload(event models.AuditNotificationEvent) map[string]interface{} {
	title, _ := notificationPresentation(event)
	return map[string]interface{}{
		"type":      "audit",
		"title":     title,
		"user":      displayUsername(event.Username),
		"action":    event.Action,
		"resource":  event.Resource,
		"status":    event.Status,
		"message":   event.Message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"source":    "docklog",
	}
}
