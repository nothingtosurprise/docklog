package services

import (
	"testing"
	"time"

	"docklog/models"
)

func TestLogLineMatches(t *testing.T) {
	cfg := modelsLogConfig(t, []LogPatternInput{
		{Pattern: "ERROR"},
		{Pattern: "ignore", Exclude: true},
	}, false)
	if !logLineMatches(cfg, "something ERROR happened") {
		t.Fatal("expected ERROR match")
	}
	if logLineMatches(cfg, "ignore this ERROR") {
		t.Fatal("expected exclude to win")
	}
}

func TestAlertSuppressorCooldown(t *testing.T) {
	s := newAlertSuppressor()
	if ok, _ := s.allow("cpu", "api", "high", 1, 0, 0, false); !ok {
		t.Fatal("first alert should pass")
	}
	if ok, reason := s.allow("cpu", "api", "high", 1, 0, 0, false); ok || reason != "cooldown" {
		t.Fatalf("expected cooldown, got ok=%v reason=%q", ok, reason)
	}
}

func TestEventTypeMatches(t *testing.T) {
	cfg := models.EventAlertConfig{Events: []string{"restart", "oom"}}
	if !eventTypeMatches(cfg, "restart") {
		t.Fatal("expected restart")
	}
	if eventTypeMatches(cfg, "start") {
		t.Fatal("start should not match")
	}
}

func TestNormalizeK8sAlertEvent(t *testing.T) {
	if got := normalizeK8sAlertEvent("CrashLoopBackOff"); got != "crash_loop_backoff" {
		t.Fatalf("expected crash_loop_backoff, got %q", got)
	}
	if got := normalizeK8sAlertEvent("ImagePullBackOff"); got != "image_pull_backoff" {
		t.Fatalf("expected image_pull_backoff, got %q", got)
	}
	if got := normalizeK8sAlertEvent("Started"); got != "" {
		t.Fatalf("expected empty for non-alert reason, got %q", got)
	}
}

type LogPatternInput struct {
	Pattern string
	Exclude bool
	Regex   bool
}

func modelsLogConfig(t *testing.T, patterns []LogPatternInput, caseSensitive bool) models.LogAlertConfig {
	t.Helper()
	cfg := models.LogAlertConfig{MatchCount: 1, WindowSeconds: 60, CaseSensitive: caseSensitive}
	for _, item := range patterns {
		cfg.Patterns = append(cfg.Patterns, models.LogPattern{
			Pattern: item.Pattern, Exclude: item.Exclude, Regex: item.Regex,
		})
	}
	return cfg
}

func TestValidateAlertRuleUpsertDisabledWithoutDestinations(t *testing.T) {
	input := models.AlertRuleUpsert{
		RuleKey: "high-cpu", Name: "High CPU", Severity: models.AlertSeverityWarning,
		Enabled: false, SourceType: models.AlertSourceMetrics,
		Config: models.MetricAlertConfig{Metric: "cpu", Threshold: 90, DurationMinutes: 5},
		Scope: models.AlertScope{Type: models.AlertScopeAll},
	}
	if err := validateAlertRuleUpsert(input); err != nil {
		t.Fatalf("disabled rule without destinations should validate: %v", err)
	}
}

func TestMergeAlertRuleUpsertPreservesChannels(t *testing.T) {
	existing := models.AlertRule{
		RuleKey: "restart-loop", Name: "Restart loop", Severity: models.AlertSeverityWarning,
		SourceType: models.AlertSourceEvents, ChannelIDsJSON: `[3]`,
		ConfigJSON: `{"events":["restart"],"min_occurrences":3,"window_seconds":300}`,
		ScopeJSON: `{"type":"all"}`, CooldownMinutes: 10, MaxPerHour: 12, GroupWindowMinutes: 5,
	}
	merged := mergeAlertRuleUpsert(existing, models.AlertRuleUpsert{Enabled: true})
	if len(merged.ChannelIDs) != 1 || merged.ChannelIDs[0] != 3 {
		t.Fatalf("expected preserved channel ids, got %v", merged.ChannelIDs)
	}
}

func TestSlidingCounterWindow(t *testing.T) {
	counter := newSlidingCounter(2 * time.Second)
	now := time.Now()
	if counter.add(now) != 1 {
		t.Fatal("expected 1")
	}
	if counter.add(now.Add(time.Second)) != 2 {
		t.Fatal("expected 2")
	}
	if counter.count(now.Add(3 * time.Second)) != 0 {
		t.Fatal("expected expired window")
	}
}
