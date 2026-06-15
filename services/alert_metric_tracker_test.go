package services

import (
	"testing"
	"time"
)

func TestMetricBreachTrackerFiresAfterDuration(t *testing.T) {
	tracker := newMetricBreachTracker()
	now := time.Now()
	duration := 5 * time.Minute

	if fire, _ := tracker.observe("high-cpu", "api", true, duration, now); fire {
		t.Fatal("should not fire on first breach sample")
	}
	if fire, _ := tracker.observe("high-cpu", "api", true, duration, now.Add(4*time.Minute)); fire {
		t.Fatal("should not fire before duration elapses")
	}
	if fire, _ := tracker.observe("high-cpu", "api", true, duration, now.Add(5*time.Minute)); !fire {
		t.Fatal("expected fire after duration")
	}
}

func TestMetricBreachTrackerRecoveryState(t *testing.T) {
	tracker := newMetricBreachTracker()
	now := time.Now()
	duration := time.Minute

	tracker.observe("high-cpu", "api", true, duration, now)
	if fire, _ := tracker.observe("high-cpu", "api", true, duration, now.Add(duration)); !fire {
		t.Fatal("expected fire")
	}
	tracker.markFired("high-cpu", "api")

	if tracker.consumeRecovery("high-cpu", "api") != true {
		t.Fatal("expected pending recovery")
	}
	if tracker.consumeRecovery("high-cpu", "api") != false {
		t.Fatal("recovery should only fire once")
	}
}

func TestMetricBreachTrackerClearsBreachWhenRecovered(t *testing.T) {
	tracker := newMetricBreachTracker()
	now := time.Now()
	duration := 5 * time.Minute

	tracker.observe("high-cpu", "api", true, duration, now)
	if fire, _ := tracker.observe("high-cpu", "api", false, duration, now.Add(time.Minute)); fire {
		t.Fatal("clearing breach should not fire")
	}
	if fire, _ := tracker.observe("high-cpu", "api", true, duration, now.Add(2*time.Minute)); fire {
		t.Fatal("breach timer should restart after recovery sample")
	}
}
