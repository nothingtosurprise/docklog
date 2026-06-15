package services

import (
	"fmt"
	"strconv"
	"time"

	"docklog/models"
)

type AlertDeliveryResult struct {
	ChannelID   int64
	ChannelType string
	Attempts    int
	Err         error
}

func (s *NotificationService) DispatchAlert(alert models.AlertNotification, channelIDs []int64) []AlertDeliveryResult {
	if s == nil {
		return []AlertDeliveryResult{{Err: fmt.Errorf("notification service unavailable")}}
	}
	if err := s.Reload(); err != nil {
		return []AlertDeliveryResult{{Err: err}}
	}
	if !s.deliveryEnabled() {
		return []AlertDeliveryResult{{Err: fmt.Errorf("notification delivery is disabled")}}
	}

	targets := s.channelsByIDs(channelIDs)
	results := make([]AlertDeliveryResult, 0, len(targets))
	for _, channel := range targets {
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil || !s.channelIsConfigured(channel.ChannelType, config) {
			results = append(results, AlertDeliveryResult{
				ChannelID: channel.ID, ChannelType: channel.ChannelType,
				Err: fmt.Errorf("channel is not configured"),
			})
			continue
		}
		if !eventsFromConfig(config).NotifyAlertEvents {
			results = append(results, AlertDeliveryResult{
				ChannelID: channel.ID, ChannelType: channel.ChannelType,
				Err: fmt.Errorf("intelligent alerts are disabled for this channel"),
			})
			continue
		}
		attempts, err := s.deliverAlertWithRetry(channel.ChannelType, config, alert)
		results = append(results, AlertDeliveryResult{
			ChannelID: channel.ID, ChannelType: channel.ChannelType,
			Attempts: attempts, Err: err,
		})
	}
	return results
}

func (s *NotificationService) channelsByIDs(channelIDs []int64) []models.NotificationChannel {
	if len(channelIDs) == 0 {
		return nil
	}
	want := map[int64]bool{}
	for _, id := range channelIDs {
		want[id] = true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.NotificationChannel, 0, len(channelIDs))
	for _, channel := range s.channels {
		if !want[channel.ID] {
			continue
		}
		config, err := parseChannelConfig(channel.ConfigJSON)
		if err != nil || !s.channelIsConfigured(channel.ChannelType, config) {
			continue
		}
		out = append(out, channel)
	}
	return out
}

func (s *NotificationService) deliverAlertWithRetry(channelType string, config map[string]string, alert models.AlertNotification) (int, error) {
	backoffs := []time.Duration{0, 2 * time.Second, 5 * time.Second}
	var lastErr error
	for attempt, wait := range backoffs {
		if wait > 0 {
			time.Sleep(wait)
		}
		if err := s.deliverAlert(channelType, config, alert); err != nil {
			lastErr = err
			continue
		}
		return attempt + 1, nil
	}
	return len(backoffs), lastErr
}

func (s *NotificationService) deliverAlert(channelType string, config map[string]string, alert models.AlertNotification) error {
	webhookURL := config["webhook_url"]
	switch channelType {
	case models.ChannelTypeSlack:
		return s.postWebhook(webhookURL, buildSlackAlertPayload(alert))
	case models.ChannelTypeTeams:
		return s.postWebhook(webhookURL, buildTeamsAlertPayload(alert))
	case models.ChannelTypeDiscord:
		return s.postWebhook(webhookURL, buildDiscordAlertPayload(alert))
	case models.ChannelTypeCustom:
		return s.postWebhook(webhookURL, buildCustomAlertPayload(alert))
	default:
		return fmt.Errorf("channel type %q is not available yet", channelType)
	}
}

func alertSeverityColor(severity string) string {
	switch severity {
	case models.AlertSeverityCritical:
		return "EF4444"
	case models.AlertSeverityWarning:
		return "F59E0B"
	default:
		return "0891B2"
	}
}

func alertTitle(alert models.AlertNotification) string {
	prefix := "Alert"
	if alert.Recovery {
		prefix = "Recovered"
	}
	return fmt.Sprintf("%s: %s", prefix, alert.Message)
}

func buildSlackAlertPayload(alert models.AlertNotification) map[string]interface{} {
	title := alertTitle(alert)
	color := alertSeverityColor(alert.Severity)
	text := fmt.Sprintf("*Rule:* `%s`\n*Severity:* %s\n*Container:* `%s`\n*Host:* %s\n*Source:* %s\n%s",
		alert.RuleID, alert.Severity, alert.Container, alert.Host, alert.Source, alert.Timestamp.Format(time.RFC3339))
	return map[string]interface{}{
		"text": fmt.Sprintf("DockLog Alert: %s", title),
		"attachments": []map[string]interface{}{{
			"color": color, "title": title, "text": text, "footer": "DockLog",
			"ts": alert.Timestamp.Unix(), "mrkdwn_in": []string{"text", "title"},
		}},
	}
}

func buildTeamsAlertPayload(alert models.AlertNotification) map[string]interface{} {
	title := alertTitle(alert)
	return map[string]interface{}{
		"@type": "MessageCard", "@context": "https://schema.org/extensions",
		"summary": title, "themeColor": alertSeverityColor(alert.Severity), "title": "DockLog Alert",
		"sections": []map[string]interface{}{{
			"activityTitle": title,
			"facts": []map[string]string{
				{"name": "Rule", "value": alert.RuleID},
				{"name": "Severity", "value": alert.Severity},
				{"name": "Container", "value": alert.Container},
				{"name": "Host", "value": alert.Host},
				{"name": "Source", "value": alert.Source},
				{"name": "Time", "value": alert.Timestamp.Format(time.RFC3339)},
			},
		}},
	}
}

func buildDiscordAlertPayload(alert models.AlertNotification) map[string]interface{} {
	title := alertTitle(alert)
	colorInt, _ := strconv.ParseInt(alertSeverityColor(alert.Severity), 16, 64)
	desc := fmt.Sprintf("Rule: %s\nSeverity: %s\nContainer: %s\nHost: %s\nSource: %s",
		alert.RuleID, alert.Severity, alert.Container, alert.Host, alert.Source)
	return map[string]interface{}{
		"content": fmt.Sprintf("DockLog Alert: %s", title),
		"embeds": []map[string]interface{}{{
			"title": title, "description": desc, "color": colorInt,
		}},
	}
}

func buildCustomAlertPayload(alert models.AlertNotification) map[string]interface{} {
	return map[string]interface{}{
		"type":      "alert",
		"title":     alertTitle(alert),
		"ruleId":    alert.RuleID,
		"severity":  alert.Severity,
		"container": alert.Container,
		"host":      alert.Host,
		"source":    alert.Source,
		"message":   alert.Message,
		"recovery":  alert.Recovery,
		"timestamp": alert.Timestamp.UTC().Format(time.RFC3339),
	}
}
