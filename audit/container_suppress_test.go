package audit

import (
	"testing"
	"time"
)

func TestSuppressContainerEvent(t *testing.T) {
	id := "ed4f4a86a9380123456789abcd"
	SuppressContainerEvent(id, "restart", time.Second)

	if !ShouldSuppressContainerEvent(id, "restart") {
		t.Fatal("expected restart to be suppressed")
	}
	if !ShouldSuppressContainerEvent(id, "start") {
		t.Fatal("expected start to be suppressed after restart")
	}
	if !ShouldSuppressContainerEvent(id, "stop") {
		t.Fatal("expected stop to be suppressed after restart")
	}
	if ShouldSuppressContainerEvent(id, "remove") {
		t.Fatal("remove should not be suppressed")
	}
}
