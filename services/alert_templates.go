package services

import "docklog/models"

var defaultAlertTemplates = []models.AlertTemplate{
	{RuleKey: "container-unhealthy", Name: "Container unhealthy", Description: "Docker HEALTHCHECK reported unhealthy", Severity: models.AlertSeverityCritical, SourceType: models.AlertSourceEvents},
	{RuleKey: "oom-killed", Name: "OOM killed", Description: "Container was killed by the OOM killer", Severity: models.AlertSeverityCritical, SourceType: models.AlertSourceEvents},
	{RuleKey: "restart-loop", Name: "Container restart loop", Description: "Repeated restarts within a short window", Severity: models.AlertSeverityWarning, SourceType: models.AlertSourceEvents},
	{RuleKey: "high-cpu", Name: "High CPU usage", Description: "CPU usage above 90% for 5 minutes", Severity: models.AlertSeverityCritical, SourceType: models.AlertSourceMetrics},
	{RuleKey: "high-memory", Name: "High memory usage", Description: "Memory usage above 80% for 10 minutes", Severity: models.AlertSeverityWarning, SourceType: models.AlertSourceMetrics},
	{RuleKey: "error-log-spike", Name: "Error log spike", Description: "10 ERROR log lines within 2 minutes", Severity: models.AlertSeverityWarning, SourceType: models.AlertSourceLogs},
}

func defaultTemplateRules() []models.AlertRule {
	return []models.AlertRule{
		{
			RuleKey: "container-unhealthy", Name: "Container unhealthy",
			Description: "Docker HEALTHCHECK reported unhealthy",
			Severity: models.AlertSeverityCritical, Enabled: false, SourceType: models.AlertSourceEvents,
			ConfigJSON: `{"events":["unhealthy"],"min_occurrences":1,"window_seconds":60}`,
			ScopeJSON: `{"type":"all"}`, ChannelIDsJSON: "[]",
			CooldownMinutes: 15, MaxPerHour: 20, GroupWindowMinutes: 5, RecoveryEnabled: true, IsTemplate: true,
		},
		{
			RuleKey: "oom-killed", Name: "OOM killed",
			Description: "Container was killed by the OOM killer",
			Severity: models.AlertSeverityCritical, Enabled: false, SourceType: models.AlertSourceEvents,
			ConfigJSON: `{"events":["oom"],"min_occurrences":1,"window_seconds":60}`,
			ScopeJSON: `{"type":"all"}`, ChannelIDsJSON: "[]",
			CooldownMinutes: 15, MaxPerHour: 20, GroupWindowMinutes: 5, IsTemplate: true,
		},
		{
			RuleKey: "restart-loop", Name: "Container restart loop",
			Description: "Repeated restarts within a short window",
			Severity: models.AlertSeverityWarning, Enabled: false, SourceType: models.AlertSourceEvents,
			ConfigJSON: `{"events":["restart"],"min_occurrences":3,"window_seconds":300}`,
			ScopeJSON: `{"type":"all"}`, ChannelIDsJSON: "[]",
			CooldownMinutes: 10, MaxPerHour: 12, GroupWindowMinutes: 5, IsTemplate: true,
		},
		{
			RuleKey: "high-cpu", Name: "High CPU usage",
			Description: "CPU usage above 90% for 5 minutes",
			Severity: models.AlertSeverityCritical, Enabled: false, SourceType: models.AlertSourceMetrics,
			ConfigJSON: `{"metric":"cpu","operator":"gt","threshold":90,"duration_minutes":5}`,
			ScopeJSON: `{"type":"all"}`, ChannelIDsJSON: "[]",
			CooldownMinutes: 15, MaxPerHour: 10, GroupWindowMinutes: 5, RecoveryEnabled: true, IsTemplate: true,
		},
		{
			RuleKey: "high-memory", Name: "High memory usage",
			Description: "Memory usage above 80% for 10 minutes",
			Severity: models.AlertSeverityWarning, Enabled: false, SourceType: models.AlertSourceMetrics,
			ConfigJSON: `{"metric":"memory","operator":"gt","threshold":80,"duration_minutes":10}`,
			ScopeJSON: `{"type":"all"}`, ChannelIDsJSON: "[]",
			CooldownMinutes: 15, MaxPerHour: 10, GroupWindowMinutes: 5, RecoveryEnabled: true, IsTemplate: true,
		},
		{
			RuleKey: "error-log-spike", Name: "Error log spike",
			Description: "10 ERROR log lines within 2 minutes",
			Severity: models.AlertSeverityWarning, Enabled: false, SourceType: models.AlertSourceLogs,
			ConfigJSON: `{"patterns":[{"pattern":"ERROR"},{"pattern":"FATAL"},{"pattern":"panic","regex":false}],"match_count":10,"window_seconds":120,"case_sensitive":false}`,
			ScopeJSON: `{"type":"all"}`, ChannelIDsJSON: "[]",
			CooldownMinutes: 10, MaxPerHour: 12, GroupWindowMinutes: 5, IsTemplate: true,
		},
	}
}
