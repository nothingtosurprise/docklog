package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"docklog/models"
)

// ProcessK8sEvent evaluates enabled Kubernetes warning event alert rules.
func (e *AlertEngine) ProcessK8sEvent(event models.K8sEvent) {
	if e == nil {
		return
	}
	eventType := normalizeK8sAlertEvent(event.Reason)
	if eventType == "" {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(event.Type), "Warning") {
		return
	}

	namespace := strings.TrimSpace(event.Namespace)
	resourceName := strings.TrimSpace(event.InvolvedName)
	if namespace == "" || resourceName == "" {
		return
	}

	resourceKey := namespace + "/" + resourceName
	rules := e.rulesBySource(models.AlertSourceK8sEvents)
	now := time.Now()
	tracker := e.k8sEventTracker()

	for _, rule := range rules {
		scope, err := parseAlertScope(rule.ScopeJSON)
		if err != nil || !k8sResourceMatchesScope(namespace, resourceName, scope) {
			continue
		}
		cfg, err := parseEventAlertConfig(rule.ConfigJSON)
		if err != nil || !eventTypeMatches(cfg, eventType) {
			continue
		}

		count := tracker.add(rule.RuleKey, resourceKey, time.Duration(cfg.WindowSeconds)*time.Second, now)
		if count < cfg.MinOccurrences {
			continue
		}
		tracker.reset(rule.RuleKey, resourceKey)

		e.Emit(rule, models.AlertNotification{
			RuleID: rule.RuleKey, Severity: rule.Severity, Container: resourceKey,
			Source: models.AlertSourceK8sEvents,
			Message: k8sEventAlertMessage(rule.Name, eventType, event.Reason, count),
			Metadata: map[string]interface{}{
				"event":          eventType,
				"reason":         event.Reason,
				"message":        event.Message,
				"namespace":      namespace,
				"involved_kind":  event.InvolvedKind,
				"involved_name":  resourceName,
				"occurrences":    count,
				"event_count":    event.Count,
			},
			URL: k8sEventAlertURL(namespace, event.InvolvedKind, resourceName),
		})
	}
}

func normalizeK8sAlertEvent(reason string) string {
	switch strings.TrimSpace(reason) {
	case "CrashLoopBackOff":
		return "crash_loop_backoff"
	case "ImagePullBackOff", "ErrImagePull":
		return "image_pull_backoff"
	case "FailedScheduling":
		return "failed_scheduling"
	case "OOMKilled":
		return "oom_killed"
	case "BackOff":
		return "backoff"
	case "Failed":
		return "failed"
	case "FailedMount":
		return "failed_mount"
	case "Evicted":
		return "evicted"
	case "Unhealthy", "FailedKillPod", "FailedPreStopHook", "FailedPostStartHook":
		return "unhealthy"
	default:
		return ""
	}
}

func k8sResourceMatchesScope(namespace, resourceName string, scope models.AlertScope) bool {
	resourceKey := namespace + "/" + resourceName
	return containerMatchesScope(resourceKey, "", nil, scope) ||
		containerMatchesScope(resourceName, "", nil, scope) ||
		containerMatchesScope(namespace, "", nil, scope)
}

func k8sEventAlertMessage(ruleName, eventType, reason string, count int) string {
	if count > 1 {
		return fmt.Sprintf("%s: %s (%s) occurred %d times", ruleName, reason, eventType, count)
	}
	return fmt.Sprintf("%s: %s (%s)", ruleName, reason, eventType)
}

func k8sEventAlertURL(namespace, kind, name string) string {
	if strings.EqualFold(kind, "Pod") {
		return "/kubernetes/pods/" + namespace + "/" + name
	}
	return "/kubernetes?tab=events&namespace=" + namespace
}

func (e *AlertEngine) k8sEventTracker() *occurrenceTracker {
	if e.sharedK8sEventTracker == nil {
		e.sharedK8sEventTracker = newOccurrenceTracker()
	}
	return e.sharedK8sEventTracker
}

func k8sEventDedupKey(event models.K8sEvent) string {
	uid := strings.TrimSpace(event.UID)
	if uid == "" {
		uid = event.Name
	}
	rv := strings.TrimSpace(event.ResourceVersion)
	return event.Namespace + "/" + uid + "@" + rv + ":" + strconv.FormatInt(event.LastTimestamp, 10)
}
