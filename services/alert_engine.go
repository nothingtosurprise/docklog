package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"docklog/config"
	"docklog/models"
	"docklog/repositories"
)

const alertQueueSize = 256

type queuedAlert struct {
	rule      models.AlertRule
	notification models.AlertNotification
}

type AlertEngine struct {
	repo          *repositories.AlertRepository
	notifications *NotificationService

	mu    sync.RWMutex
	rules []models.AlertRule

	suppressor         *alertSuppressor
	queue              chan queuedAlert
	sharedEventTracker *occurrenceTracker
	recoveryMu         sync.Mutex
	recoveryState      map[string]models.AlertRule
	metricTracker      *metricBreachTracker

	host string
}

func NewAlertEngine(repo *repositories.AlertRepository, notifications *NotificationService) *AlertEngine {
	host, _ := os.Hostname()
	if host == "" {
		host = "docker-host"
	}
	engine := &AlertEngine{
		repo:          repo,
		notifications: notifications,
		suppressor:    newAlertSuppressor(),
		queue:         make(chan queuedAlert, alertQueueSize),
		recoveryState: make(map[string]models.AlertRule),
		metricTracker: newMetricBreachTracker(),
		host:          host,
	}
	go engine.runDeliveryWorkers(2)
	return engine
}

func (e *AlertEngine) Initialize() error {
	count, err := e.repo.CountRules()
	if err != nil {
		return err
	}
	if count == 0 {
		for _, rule := range defaultTemplateRules() {
			if _, err := e.repo.UpsertRule(rule); err != nil {
				return err
			}
		}
	}
	return e.Reload()
}

func (e *AlertEngine) Reload() error {
	rules, err := e.repo.ListRules()
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.rules = rules
	e.mu.Unlock()
	return nil
}

func (e *AlertEngine) enabledRules() []models.AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	enabled := make([]models.AlertRule, 0, len(e.rules))
	for _, rule := range e.rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled
}

func (e *AlertEngine) rulesBySource(source string) []models.AlertRule {
	enabled := e.enabledRules()
	out := make([]models.AlertRule, 0)
	for _, rule := range enabled {
		if rule.SourceType == source {
			out = append(out, rule)
		}
	}
	return out
}

func (e *AlertEngine) GetRuleByID(id int64) (models.AlertRule, error) {
	return e.repo.GetRule(id)
}

func (e *AlertEngine) GetPublic(limit int) (models.AlertsPublicResponse, error) {
	rules, err := e.repo.ListRules()
	if err != nil {
		return models.AlertsPublicResponse{}, err
	}
	public := make([]models.AlertRulePublic, 0, len(rules))
	for _, rule := range rules {
		public = append(public, ruleToPublic(rule))
	}
	history, err := e.repo.ListHistory(limit)
	if err != nil {
		return models.AlertsPublicResponse{}, err
	}
	return models.AlertsPublicResponse{
		Rules:     public,
		Templates: defaultAlertTemplates,
		History:   history,
	}, nil
}

func (e *AlertEngine) UpsertRule(input models.AlertRuleUpsert, existingID int64) (models.AlertRulePublic, error) {
	if existingID > 0 {
		existing, err := e.repo.GetRule(existingID)
		if err != nil {
			return models.AlertRulePublic{}, fmt.Errorf("alert rule not found")
		}
		input = mergeAlertRuleUpsert(existing, input)
	}
	if input.Enabled && len(input.ChannelIDs) == 0 && e.notifications != nil {
		input.ChannelIDs = e.notifications.ConfiguredChannelIDs()
	}
	if err := validateAlertRuleUpsert(input); err != nil {
		return models.AlertRulePublic{}, err
	}
	configJSON, err := encodeConfig(input.Config)
	if err != nil {
		return models.AlertRulePublic{}, err
	}
	scopeJSON, err := encodeScope(input.Scope)
	if err != nil {
		return models.AlertRulePublic{}, err
	}
	channelJSON, _ := json.Marshal(input.ChannelIDs)

	rule := models.AlertRule{
		ID: existingID, RuleKey: input.RuleKey, Name: input.Name, Description: input.Description,
		Severity: input.Severity, Enabled: input.Enabled, SourceType: input.SourceType,
		ConfigJSON: configJSON, ScopeJSON: scopeJSON, ChannelIDsJSON: string(channelJSON),
		CooldownMinutes: defaultInt(input.CooldownMinutes, 15),
		MaxPerHour: defaultInt(input.MaxPerHour, 20),
		GroupWindowMinutes: defaultInt(input.GroupWindowMinutes, 5),
		RecoveryEnabled: input.RecoveryEnabled,
	}
	id, err := e.repo.UpsertRule(rule)
	if err != nil {
		return models.AlertRulePublic{}, err
	}
	rule.ID = id
	_ = e.Reload()
	saved, err := e.repo.GetRule(id)
	if err != nil {
		return models.AlertRulePublic{}, err
	}
	return ruleToPublic(saved), nil
}

func (e *AlertEngine) DeleteRule(id int64) error {
	if err := e.repo.DeleteRule(id); err != nil {
		return err
	}
	return e.Reload()
}

func (e *AlertEngine) CreateFromTemplate(ruleKey string, channelIDs []int64) (models.AlertRulePublic, error) {
	for _, template := range defaultTemplateRules() {
		if template.RuleKey != ruleKey {
			continue
		}
		existing, err := e.repo.GetRuleByKey(ruleKey)
		if err == nil && existing.ID > 0 {
			var cfg interface{}
			_ = json.Unmarshal([]byte(existing.ConfigJSON), &cfg)
			scope, _ := parseAlertScope(existing.ScopeJSON)
			return e.UpsertRule(models.AlertRuleUpsert{
				RuleKey: existing.RuleKey, Name: existing.Name, Description: existing.Description,
				Severity: existing.Severity, Enabled: true, SourceType: existing.SourceType,
				Config: cfg, Scope: scope, ChannelIDs: channelIDs,
				CooldownMinutes: existing.CooldownMinutes, MaxPerHour: existing.MaxPerHour,
				GroupWindowMinutes: existing.GroupWindowMinutes, RecoveryEnabled: existing.RecoveryEnabled,
			}, existing.ID)
		}
		template.Enabled = true
		template.ChannelIDsJSON = repositories.EncodeChannelIDs(channelIDs)
		id, err := e.repo.UpsertRule(template)
		if err != nil {
			return models.AlertRulePublic{}, err
		}
		_ = e.Reload()
		saved, err := e.repo.GetRule(id)
		if err != nil {
			return models.AlertRulePublic{}, err
		}
		return ruleToPublic(saved), nil
	}
	return models.AlertRulePublic{}, fmt.Errorf("unknown template %q", ruleKey)
}

func (e *AlertEngine) TestRule(ruleID int64) error {
	rule, err := e.repo.GetRule(ruleID)
	if err != nil {
		return err
	}
	channelIDs := parseChannelIDsJSON(rule.ChannelIDsJSON)
	if len(channelIDs) == 0 {
		return fmt.Errorf("rule has no notification destinations")
	}
	alert := models.AlertNotification{
		RuleID: rule.RuleKey, Severity: rule.Severity, Container: "sample-container",
		Host: e.host, Source: rule.SourceType,
		Message: fmt.Sprintf("Test alert for rule %s", rule.Name),
		Timestamp: time.Now().UTC(), URL: "/containers/sample-container/logs",
		Metadata: map[string]interface{}{"test": true},
	}
	return e.deliverNow(rule, alert, channelIDs)
}

func (e *AlertEngine) Emit(rule models.AlertRule, alert models.AlertNotification) {
	if !rule.Enabled {
		return
	}
	select {
	case e.queue <- queuedAlert{rule: rule, notification: alert}:
	default:
		log.Printf("Alerts: queue full, dropped %s on %s", rule.RuleKey, alert.Container)
	}
}

func (e *AlertEngine) tryFire(rule models.AlertRule, alert models.AlertNotification) {
	if alert.Host == "" {
		alert.Host = e.host
	}
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now().UTC()
	}

	allowed, reason := e.suppressor.allow(
		rule.RuleKey, alert.Container, alert.Message,
		rule.CooldownMinutes, rule.MaxPerHour, rule.GroupWindowMinutes, alert.Recovery,
	)
	status := models.AlertStatusFired
	if !allowed {
		status = models.AlertStatusSuppressed
	}

	metadataJSON, _ := json.Marshal(alert.Metadata)
	historyID, err := e.repo.InsertHistory(models.AlertHistoryEntry{
		RuleKey: rule.RuleKey, RuleName: rule.Name, Severity: alert.Severity,
		Container: alert.Container, Host: alert.Host, Source: alert.Source,
		Message: alert.Message, Status: status, MetadataJSON: string(metadataJSON),
	})
	if err != nil {
		log.Printf("Alerts: history insert failed: %v", err)
		return
	}
	if !allowed {
		config.Debugf("Alerts: suppressed %s (%s) reason=%s", rule.RuleKey, alert.Container, reason)
		return
	}

	channelIDs := parseChannelIDsJSON(rule.ChannelIDsJSON)
	if len(channelIDs) == 0 {
		return
	}
	if err := e.deliverWithHistory(historyID, rule, alert, channelIDs); err != nil {
		log.Printf("Alerts: delivery failed for %s: %v", rule.RuleKey, err)
	}
}

func (e *AlertEngine) deliverWithHistory(historyID int64, rule models.AlertRule, alert models.AlertNotification, channelIDs []int64) error {
	if e.notifications == nil {
		return fmt.Errorf("notification service unavailable")
	}
	results := e.notifications.DispatchAlert(alert, channelIDs)
	var firstErr error
	for _, result := range results {
		status := models.DeliveryStatusSent
		if result.Err != nil {
			status = models.DeliveryStatusFailed
			if firstErr == nil {
				firstErr = result.Err
			}
		}
		deliveryID, err := e.repo.InsertDelivery(models.AlertDeliveryEntry{
			AlertHistoryID: historyID,
			ChannelID:      result.ChannelID,
			ChannelType:    result.ChannelType,
			Status:         models.DeliveryStatusPending,
		})
		if err != nil {
			continue
		}
		lastError := ""
		if result.Err != nil {
			lastError = result.Err.Error()
		}
		_ = e.repo.UpdateDelivery(deliveryID, status, lastError, result.Attempts)
	}
	return firstErr
}

func (e *AlertEngine) deliverNow(rule models.AlertRule, alert models.AlertNotification, channelIDs []int64) error {
	historyID, err := e.repo.InsertHistory(models.AlertHistoryEntry{
		RuleKey: rule.RuleKey, RuleName: rule.Name, Severity: alert.Severity,
		Container: alert.Container, Host: alert.Host, Source: alert.Source,
		Message: alert.Message, Status: models.AlertStatusFired,
	})
	if err != nil {
		return err
	}
	return e.deliverWithHistory(historyID, rule, alert, channelIDs)
}

func (e *AlertEngine) runDeliveryWorkers(count int) {
	for i := 0; i < count; i++ {
		go func() {
			for item := range e.queue {
				e.tryFire(item.rule, item.notification)
			}
		}()
	}
}

func (e *AlertEngine) ListDeliveries(historyID int64) ([]models.AlertDeliveryEntry, error) {
	return e.repo.ListDeliveriesForHistory(historyID)
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
