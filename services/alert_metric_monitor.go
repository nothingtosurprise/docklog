package services

import (
	"context"
	"fmt"
	"time"

	"docklog/containers"
	"docklog/db"
	"docklog/dockerutil"
	"docklog/models"

	"github.com/moby/moby/client"
)

// StartMetricAlertEvaluator checks container stats against metric alert rules.
func StartMetricAlertEvaluator(cli *client.Client, engine *AlertEngine) {
	if cli == nil || engine == nil {
		return
	}
	go runMetricAlertEvaluator(cli, engine)
}

func runMetricAlertEvaluator(cli *client.Client, engine *AlertEngine) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		_ = engine.Reload()
		evaluateMetricRules(cli, engine)
	}
}

func evaluateMetricRules(cli *client.Client, engine *AlertEngine) {
	rules := engine.rulesBySource(models.AlertSourceMetrics)
	if len(rules) == 0 {
		return
	}

	ctx := context.Background()
	res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: false})
	if err != nil {
		return
	}

	now := time.Now()
	for _, summary := range dockerutil.ExtractContainers(res) {
		id := containerSummaryID(summary)
		name := containerSummaryName(summary)
		image, _ := summary["Image"].(string)
		if id == "" || containers.IsExcludedContainer(name, image) {
			continue
		}
		labels := containerSummaryLabels(summary)

		cpu, memPct, ok := latestContainerMetric(cli, ctx, id)
		if !ok {
			continue
		}

		for _, rule := range rules {
			scope, err := parseAlertScope(rule.ScopeJSON)
			if err != nil || !containerMatchesScope(name, image, labels, scope) {
				continue
			}
			cfg, err := parseMetricAlertConfig(rule.ConfigJSON)
			if err != nil {
				continue
			}

			value := cpu
			if cfg.Metric == "memory" {
				value = memPct
			}
			breached := metricThresholdMet(cfg, value)
			fire, _ := engine.metricTracker.observe(
				rule.RuleKey, name, breached,
				time.Duration(cfg.DurationMinutes)*time.Minute, now,
			)
			if fire {
				engine.Emit(rule, models.AlertNotification{
					RuleID: rule.RuleKey, Severity: rule.Severity, Container: name,
					Source: models.AlertSourceMetrics,
					Message: fmt.Sprintf("%s: %s usage %.1f%% exceeded %.1f%% for %d minutes",
						rule.Name, cfg.Metric, value, cfg.Threshold, cfg.DurationMinutes),
					Metadata: map[string]interface{}{cfg.Metric: value, "threshold": cfg.Threshold},
					URL: "/containers/" + id,
				})
				if rule.RecoveryEnabled {
					engine.metricTracker.markFired(rule.RuleKey, name)
				}
				continue
			}

			if !breached && rule.RecoveryEnabled && engine.metricTracker.consumeRecovery(rule.RuleKey, name) {
				engine.suppressor.clearRecovery(rule.RuleKey, name)
				engine.Emit(rule, models.AlertNotification{
					RuleID: rule.RuleKey, Severity: rule.Severity, Container: name,
					Source: models.AlertSourceMetrics, Recovery: true,
					Message: fmt.Sprintf("%s recovered on %s: %s at %.1f%% (threshold %.1f%%)",
						rule.Name, name, cfg.Metric, value, cfg.Threshold),
					Metadata: map[string]interface{}{cfg.Metric: value, "threshold": cfg.Threshold},
					URL: "/containers/" + id,
				})
			}
		}
	}
}

func latestContainerMetric(cli *client.Client, ctx context.Context, containerID string) (cpu float64, memoryPercent float64, ok bool) {
	var cpuVal float64
	var memBytes int64
	err := db.DB.QueryRow(`
		SELECT cpu, memory FROM stats
		WHERE container_id = ?
		ORDER BY timestamp DESC
		LIMIT 1`, containerID).Scan(&cpuVal, &memBytes)
	if err != nil {
		return 0, 0, false
	}

	memPct := 0.0
	inspect, err := cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err == nil && inspect.Container.HostConfig != nil && inspect.Container.HostConfig.Memory > 0 {
		memPct = (float64(memBytes) / float64(inspect.Container.HostConfig.Memory)) * 100
	}
	return cpuVal, memPct, true
}
