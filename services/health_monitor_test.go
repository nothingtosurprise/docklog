package services

import (
	"docklog/models"
	"testing"
)

func TestHealthStatusFromSummary(t *testing.T) {
	cases := []struct {
		status string
		want   string
		ok     bool
	}{
		{"Up 2 hours (healthy)", "healthy", true},
		{"Up 5 minutes (unhealthy)", "unhealthy", true},
		{"Up 10 seconds (health: starting)", "starting", true},
		{"Up 2 hours", "", false},
	}
	for _, tc := range cases {
		got, ok := healthStatusFromSummary(map[string]interface{}{"Status": tc.status})
		if ok != tc.ok || got != tc.want {
			t.Fatalf("status %q => (%q, %v), want (%q, %v)", tc.status, got, ok, tc.want, tc.ok)
		}
	}
}

func TestEventMatchesHealthAlerts(t *testing.T) {
	events := models.NotificationChannelEvents{NotifyHealthEvents: true}
	if !eventMatchesChannel(events, models.AuditNotificationEvent{Action: "health_check_failed", Status: "Error"}) {
		t.Fatal("expected health failure to match")
	}
	if !eventMatchesChannel(events, models.AuditNotificationEvent{Action: "health_check_recovered", Status: "Success"}) {
		t.Fatal("expected health recovery to match")
	}

	events.NotifyHealthEvents = false
	if eventMatchesChannel(events, models.AuditNotificationEvent{Action: "health_check_failed", Status: "Error"}) {
		t.Fatal("health alerts disabled should not match")
	}
}
