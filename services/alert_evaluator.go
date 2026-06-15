package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"docklog/models"
)

func parseLogAlertConfig(raw string) (models.LogAlertConfig, error) {
	var cfg models.LogAlertConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	if cfg.MatchCount <= 0 {
		cfg.MatchCount = 1
	}
	if cfg.WindowSeconds <= 0 {
		cfg.WindowSeconds = 120
	}
	return cfg, nil
}

func parseEventAlertConfig(raw string) (models.EventAlertConfig, error) {
	var cfg models.EventAlertConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	if cfg.MinOccurrences <= 0 {
		cfg.MinOccurrences = 1
	}
	if cfg.WindowSeconds <= 0 {
		cfg.WindowSeconds = 300
	}
	return cfg, nil
}

func parseMetricAlertConfig(raw string) (models.MetricAlertConfig, error) {
	var cfg models.MetricAlertConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	if cfg.DurationMinutes <= 0 {
		cfg.DurationMinutes = 5
	}
	if cfg.Operator == "" {
		cfg.Operator = "gt"
	}
	return cfg, nil
}

func parseAlertScope(raw string) (models.AlertScope, error) {
	var scope models.AlertScope
	if strings.TrimSpace(raw) == "" {
		scope.Type = models.AlertScopeAll
		return scope, nil
	}
	if err := json.Unmarshal([]byte(raw), &scope); err != nil {
		return scope, err
	}
	if scope.Type == "" {
		scope.Type = models.AlertScopeAll
	}
	return scope, nil
}

func containerMatchesScope(name, image string, labels map[string]string, scope models.AlertScope) bool {
	switch scope.Type {
	case "", models.AlertScopeAll:
		return true
	case models.AlertScopeNames:
		for _, candidate := range scope.Containers {
			if strings.EqualFold(strings.TrimPrefix(candidate, "/"), strings.TrimPrefix(name, "/")) {
				return true
			}
		}
		return false
	case models.AlertScopePatterns:
		for _, pattern := range scope.Patterns {
			if pattern == "" {
				continue
			}
			if matched, _ := regexp.MatchString(wildcardToRegex(pattern), name); matched {
				return true
			}
		}
		return false
	case models.AlertScopeLabels:
		if len(scope.Labels) == 0 {
			return true
		}
		for key, want := range scope.Labels {
			if labels[key] != want {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func wildcardToRegex(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "^$"
	}
	if strings.Contains(pattern, "^") || strings.Contains(pattern, "$") {
		return pattern
	}
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")
	escaped = strings.ReplaceAll(escaped, "\\?", ".")
	return "^" + escaped + "$"
}

func logLineMatches(cfg models.LogAlertConfig, line string) bool {
	if len(cfg.Patterns) == 0 {
		return false
	}
	target := line
	if !cfg.CaseSensitive {
		target = strings.ToLower(line)
	}

	matchedInclude := false
	for _, pattern := range cfg.Patterns {
		if strings.TrimSpace(pattern.Pattern) == "" {
			continue
		}
		expr := pattern.Pattern
		if !cfg.CaseSensitive {
			expr = strings.ToLower(expr)
		}
		hit := false
		if pattern.Regex {
			re, err := regexp.Compile(expr)
			if err != nil {
				continue
			}
			hit = re.MatchString(target)
		} else {
			hit = strings.Contains(target, expr)
		}
		if !hit {
			continue
		}
		if pattern.Exclude {
			return false
		}
		matchedInclude = true
	}
	return matchedInclude
}

func eventTypeMatches(cfg models.EventAlertConfig, eventType string) bool {
	eventType = strings.ToLower(strings.TrimSpace(eventType))
	for _, item := range cfg.Events {
		if strings.EqualFold(strings.TrimSpace(item), eventType) {
			return true
		}
	}
	return false
}

func metricThresholdMet(cfg models.MetricAlertConfig, value float64) bool {
	switch cfg.Operator {
	case "gte", ">=":
		return value >= cfg.Threshold
	case "gt", ">":
		return value > cfg.Threshold
	default:
		return value > cfg.Threshold
	}
}

func validateAlertRuleUpsert(input models.AlertRuleUpsert) error {
	input.RuleKey = strings.TrimSpace(input.RuleKey)
	input.Name = strings.TrimSpace(input.Name)
	if input.RuleKey == "" {
		return fmt.Errorf("rule id is required")
	}
	if input.Name == "" {
		return fmt.Errorf("name is required")
	}
	switch input.Severity {
	case models.AlertSeverityInfo, models.AlertSeverityWarning, models.AlertSeverityCritical:
	default:
		return fmt.Errorf("severity must be info, warning, or critical")
	}
	switch input.SourceType {
	case models.AlertSourceLogs, models.AlertSourceEvents, models.AlertSourceMetrics:
	default:
		return fmt.Errorf("source type must be logs, events, or metrics")
	}
	if input.Enabled && len(input.ChannelIDs) == 0 {
		return fmt.Errorf("select at least one notification destination")
	}
	if input.CooldownMinutes < 0 {
		return fmt.Errorf("cooldown minutes cannot be negative")
	}
	if input.MaxPerHour < 0 {
		return fmt.Errorf("max alerts per hour cannot be negative")
	}
	if input.GroupWindowMinutes < 0 {
		return fmt.Errorf("group window minutes cannot be negative")
	}

	switch input.SourceType {
	case models.AlertSourceLogs:
		cfg, err := decodeLogConfig(input.Config)
		if err != nil {
			return err
		}
		if len(cfg.Patterns) == 0 {
			return fmt.Errorf("add at least one log pattern")
		}
	case models.AlertSourceEvents:
		cfg, err := decodeEventConfig(input.Config)
		if err != nil {
			return err
		}
		if len(cfg.Events) == 0 {
			return fmt.Errorf("select at least one docker event")
		}
	case models.AlertSourceMetrics:
		cfg, err := decodeMetricConfig(input.Config)
		if err != nil {
			return err
		}
		switch cfg.Metric {
		case "cpu", "memory":
		default:
			return fmt.Errorf("metric must be cpu or memory")
		}
		if cfg.Threshold <= 0 {
			return fmt.Errorf("threshold must be greater than zero")
		}
	}
	return nil
}

func decodeLogConfig(raw interface{}) (models.LogAlertConfig, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return models.LogAlertConfig{}, err
	}
	return parseLogAlertConfig(string(data))
}

func decodeEventConfig(raw interface{}) (models.EventAlertConfig, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return models.EventAlertConfig{}, err
	}
	return parseEventAlertConfig(string(data))
}

func decodeMetricConfig(raw interface{}) (models.MetricAlertConfig, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return models.MetricAlertConfig{}, err
	}
	return parseMetricAlertConfig(string(data))
}

func encodeConfig(raw interface{}) (string, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func encodeScope(scope models.AlertScope) (string, error) {
	if scope.Type == "" {
		scope.Type = models.AlertScopeAll
	}
	data, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ruleToPublic(rule models.AlertRule) models.AlertRulePublic {
	scope, _ := parseAlertScope(rule.ScopeJSON)
	channelIDs := parseChannelIDsJSON(rule.ChannelIDsJSON)
	var config interface{}
	switch rule.SourceType {
	case models.AlertSourceLogs:
		cfg, _ := parseLogAlertConfig(rule.ConfigJSON)
		config = cfg
	case models.AlertSourceEvents:
		cfg, _ := parseEventAlertConfig(rule.ConfigJSON)
		config = cfg
	case models.AlertSourceMetrics:
		cfg, _ := parseMetricAlertConfig(rule.ConfigJSON)
		config = cfg
	}
	return models.AlertRulePublic{
		AlertRule:  rule,
		Config:     config,
		Scope:      scope,
		ChannelIDs: channelIDs,
	}
}

func parseChannelIDsJSON(raw string) []int64 {
	var ids []int64
	_ = json.Unmarshal([]byte(raw), &ids)
	return ids
}

func mergeAlertRuleUpsert(existing models.AlertRule, input models.AlertRuleUpsert) models.AlertRuleUpsert {
	if strings.TrimSpace(input.RuleKey) == "" {
		input.RuleKey = existing.RuleKey
	}
	if strings.TrimSpace(input.Name) == "" {
		input.Name = existing.Name
	}
	if strings.TrimSpace(input.Description) == "" {
		input.Description = existing.Description
	}
	if strings.TrimSpace(input.Severity) == "" {
		input.Severity = existing.Severity
	}
	if strings.TrimSpace(input.SourceType) == "" {
		input.SourceType = existing.SourceType
	}
	if len(input.ChannelIDs) == 0 {
		input.ChannelIDs = parseChannelIDsJSON(existing.ChannelIDsJSON)
	}
	if input.Config == nil {
		switch existing.SourceType {
		case models.AlertSourceLogs:
			cfg, _ := parseLogAlertConfig(existing.ConfigJSON)
			input.Config = cfg
		case models.AlertSourceEvents:
			cfg, _ := parseEventAlertConfig(existing.ConfigJSON)
			input.Config = cfg
		case models.AlertSourceMetrics:
			cfg, _ := parseMetricAlertConfig(existing.ConfigJSON)
			input.Config = cfg
		}
	}
	if input.Scope.Type == "" && len(input.Scope.Containers) == 0 && len(input.Scope.Patterns) == 0 && len(input.Scope.Labels) == 0 {
		scope, _ := parseAlertScope(existing.ScopeJSON)
		input.Scope = scope
	}
	if input.CooldownMinutes == 0 {
		input.CooldownMinutes = existing.CooldownMinutes
	}
	if input.MaxPerHour == 0 {
		input.MaxPerHour = existing.MaxPerHour
	}
	if input.GroupWindowMinutes == 0 {
		input.GroupWindowMinutes = existing.GroupWindowMinutes
	}
	return input
}
