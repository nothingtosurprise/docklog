package services

import (
	"sync"
	"time"
)

type suppressionKey struct {
	ruleKey   string
	container string
	message   string
}

type alertSuppressor struct {
	mu sync.Mutex

	cooldownUntil map[suppressionKey]time.Time
	groupUntil    map[suppressionKey]time.Time
	hourCounts    map[string][]time.Time
}

func newAlertSuppressor() *alertSuppressor {
	return &alertSuppressor{
		cooldownUntil: make(map[suppressionKey]time.Time),
		groupUntil:    make(map[suppressionKey]time.Time),
		hourCounts:    make(map[string][]time.Time),
	}
}

func (s *alertSuppressor) allow(ruleKey, container, message string, cooldownMin, maxPerHour, groupWindowMin int, recovery bool) (bool, string) {
	if recovery {
		return true, ""
	}

	now := time.Now()
	key := suppressionKey{ruleKey: ruleKey, container: container, message: message}
	groupKey := suppressionKey{ruleKey: ruleKey, container: container, message: ""}

	s.mu.Lock()
	defer s.mu.Unlock()

	if cooldownMin > 0 {
		if until, ok := s.cooldownUntil[key]; ok && now.Before(until) {
			return false, "cooldown"
		}
	}

	if groupWindowMin > 0 {
		if until, ok := s.groupUntil[groupKey]; ok && now.Before(until) {
			return false, "grouped"
		}
	}

	if maxPerHour > 0 {
		bucketKey := ruleKey + "|" + container
		times := s.pruneHourCounts(s.hourCounts[bucketKey], now)
		if len(times) >= maxPerHour {
			s.hourCounts[bucketKey] = times
			return false, "hourly_limit"
		}
	}

	if cooldownMin > 0 {
		s.cooldownUntil[key] = now.Add(time.Duration(cooldownMin) * time.Minute)
	}
	if groupWindowMin > 0 {
		s.groupUntil[groupKey] = now.Add(time.Duration(groupWindowMin) * time.Minute)
	}
	if maxPerHour > 0 {
		bucketKey := ruleKey + "|" + container
		times := s.pruneHourCounts(s.hourCounts[bucketKey], now)
		times = append(times, now)
		s.hourCounts[bucketKey] = times
	}

	return true, ""
}

func (s *alertSuppressor) clearRecovery(ruleKey, container string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := suppressionKey{ruleKey: ruleKey, container: container}
	for key := range s.cooldownUntil {
		if key.ruleKey == prefix.ruleKey && key.container == prefix.container {
			delete(s.cooldownUntil, key)
		}
	}
	delete(s.groupUntil, suppressionKey{ruleKey: ruleKey, container: container, message: ""})
}

func (s *alertSuppressor) pruneHourCounts(times []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-time.Hour)
	kept := make([]time.Time, 0, len(times))
	for _, ts := range times {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	return kept
}
