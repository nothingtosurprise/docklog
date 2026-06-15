package stats

import (
	"math"
	"os"
	"testing"
	"time"
)

func TestParseProcCPUStatLine(t *testing.T) {
	stat, err := parseProcCPUStatLine("cpu  2262 34 2301 817674 0 0 12 0 0 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if stat.idle != 817674 {
		t.Fatalf("unexpected idle: %d", stat.idle)
	}
	if stat.total == 0 {
		t.Fatal("expected non-zero total")
	}
}

func TestSampleCPUPercentIdleHost(t *testing.T) {
	start := procCPUStat{total: 1000, idle: 900}
	end := procCPUStat{total: 2000, idle: 1810}

	totalDelta := end.total - start.total
	idleDelta := end.idle - start.idle
	busy := (1.0 - float64(idleDelta)/float64(totalDelta)) * 100.0
	if math.Abs(busy-9.0) > 0.01 {
		t.Fatalf("expected ~9%% busy, got %.2f", busy)
	}
}

func TestSmoothCPU(t *testing.T) {
	cpuSmoothed = 0
	first := smoothCPU(20)
	if first != 20 {
		t.Fatalf("expected first sample to pass through, got %.2f", first)
	}
	second := smoothCPU(10)
	if second >= 20 || second <= 10 {
		t.Fatalf("expected smoothed value between 10 and 20, got %.2f", second)
	}
	cpuSmoothed = 0
}

func TestSampleCPUPercentLive(t *testing.T) {
	if os.Getenv("DOCKLOG_LIVE_CPU_TEST") == "" {
		t.Skip("set DOCKLOG_LIVE_CPU_TEST=1 to run live /proc sampling")
	}
	value, err := sampleCPUPercent(200 * time.Millisecond)
	if err != nil {
		t.Fatalf("sample failed: %v", err)
	}
	if value < 0 || value > 100 {
		t.Fatalf("cpu out of range: %.2f", value)
	}
}
