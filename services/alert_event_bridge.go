package services

import (
	"strconv"
	"strings"
	"time"

	"docklog/models"
)

// ProcessDockerEventForAlerts evaluates enabled event alert rules.
func (e *AlertEngine) ProcessDockerEvent(containerID, name, image string, labels map[string]string, eventType string, exitCode int) {
	if e == nil {
		return
	}
	eventType = normalizeDockerAlertEvent(eventType, exitCode)
	if eventType == "" {
		return
	}

	rules := e.rulesBySource(models.AlertSourceEvents)
	now := time.Now()
	tracker := e.eventTracker()

	for _, rule := range rules {
		scope, err := parseAlertScope(rule.ScopeJSON)
		if err != nil || !containerMatchesScope(name, image, labels, scope) {
			continue
		}
		cfg, err := parseEventAlertConfig(rule.ConfigJSON)
		if err != nil || !eventTypeMatches(cfg, eventType) {
			continue
		}

		count := tracker.add(rule.RuleKey, name, time.Duration(cfg.WindowSeconds)*time.Second, now)
		if count < cfg.MinOccurrences {
			continue
		}
		tracker.reset(rule.RuleKey, name)

		e.Emit(rule, models.AlertNotification{
			RuleID: rule.RuleKey, Severity: rule.Severity, Container: name,
			Source: models.AlertSourceEvents,
			Message: eventAlertMessage(rule.Name, eventType, count),
			Metadata: map[string]interface{}{"event": eventType, "occurrences": count, "exit_code": exitCode},
			URL: "/containers/" + containerID,
		})

		if rule.RecoveryEnabled && isRecoverableEvent(eventType) {
			e.trackRecovery(rule, name)
		}
	}
}

func normalizeDockerAlertEvent(eventType string, exitCode int) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "start":
		return "start"
	case "stop":
		return "stop"
	case "restart":
		return "restart"
	case "die":
		if exitCode != 0 {
			return "exit_nonzero"
		}
		return "stop"
	case "oom":
		return "oom"
	case "unhealthy", "health_check_failed":
		return "unhealthy"
	case "healthy", "health_check_recovered":
		return "healthy"
	default:
		return ""
	}
}

func isRecoverableEvent(eventType string) bool {
	switch eventType {
	case "unhealthy", "exit_nonzero", "oom":
		return true
	default:
		return false
	}
}

func eventAlertMessage(ruleName, eventType string, count int) string {
	if count > 1 {
		return ruleName + ": " + eventType + " occurred " + strconv.Itoa(count) + " times"
	}
	return ruleName + ": container " + eventType
}

func (e *AlertEngine) eventTracker() *occurrenceTracker {
	if e.sharedEventTracker == nil {
		e.sharedEventTracker = newOccurrenceTracker()
	}
	return e.sharedEventTracker
}

func (e *AlertEngine) trackRecovery(rule models.AlertRule, container string) {
	key := rule.RuleKey + "|" + container
	e.recoveryMu.Lock()
	e.recoveryState[key] = rule
	e.recoveryMu.Unlock()
}

func (e *AlertEngine) EmitRecovery(containerID, name, image string, labels map[string]string, recoveredEvent string) {
	e.recoveryMu.Lock()
	defer e.recoveryMu.Unlock()
	for key, rule := range e.recoveryState {
		if !strings.HasSuffix(key, "|"+name) {
			continue
		}
		cfg, err := parseEventAlertConfig(rule.ConfigJSON)
		if err != nil {
			continue
		}
		if !eventTypeMatches(cfg, recoveredEvent) {
			continue
		}
		delete(e.recoveryState, key)
		e.suppressor.clearRecovery(rule.RuleKey, name)
		e.Emit(rule, models.AlertNotification{
			RuleID: rule.RuleKey, Severity: rule.Severity, Container: name,
			Source: models.AlertSourceEvents, Recovery: true,
			Message: rule.Name + " recovered on " + name,
			URL: "/containers/" + containerID,
		})
	}
}
