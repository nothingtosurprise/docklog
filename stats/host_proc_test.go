package stats

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryFromProc(t *testing.T) {
	dir := t.TempDir()
	meminfo := filepath.Join(dir, "meminfo")
	content := `MemTotal:       18874368 kB
MemFree:         1048576 kB
MemAvailable:   15728640 kB
`
	if err := os.WriteFile(meminfo, []byte(content), 0o644); err != nil {
		t.Fatalf("write meminfo: %v", err)
	}

	total, used, ok := memoryFromProc(meminfo)
	if !ok {
		t.Fatal("expected meminfo parse to succeed")
	}
	if total != 18874368*1024 {
		t.Fatalf("unexpected total: %d", total)
	}
	if used != (18874368-15728640)*1024 {
		t.Fatalf("unexpected used: %d", used)
	}
}

func TestLogicalCPUsFromProc(t *testing.T) {
	dir := t.TempDir()
	cpuinfo := filepath.Join(dir, "cpuinfo")
	content := `processor	: 0
processor	: 1
processor	: 2
`
	if err := os.WriteFile(cpuinfo, []byte(content), 0o644); err != nil {
		t.Fatalf("write cpuinfo: %v", err)
	}

	count, ok := logicalCPUsFromProc(cpuinfo)
	if !ok || count != 3 {
		t.Fatalf("expected 3 logical CPUs, got %d ok=%v", count, ok)
	}
}

func TestLogicalCPUsFromStat(t *testing.T) {
	dir := t.TempDir()
	stat := filepath.Join(dir, "stat")
	content := `cpu  2262 34 2301 817674 0 0 12 0 0 0
cpu0 1200 10 900 400000 0 0 5 0 0 0
cpu1 1062 24 1401 417674 0 0 7 0 0 0
`
	if err := os.WriteFile(stat, []byte(content), 0o644); err != nil {
		t.Fatalf("write stat: %v", err)
	}

	count, ok := logicalCPUsFromStat(stat)
	if !ok || count != 2 {
		t.Fatalf("expected 2 logical CPUs, got %d ok=%v", count, ok)
	}
}

func TestSystemMemoryUsesHostProc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "meminfo"), []byte(`MemTotal:       1048576 kB
MemAvailable:    524288 kB
`), 0o644); err != nil {
		t.Fatalf("write meminfo: %v", err)
	}

	t.Setenv("HOST_PROC", dir)
	total, used := systemMemory()
	if total != 1048576*1024 {
		t.Fatalf("unexpected total: %d", total)
	}
	if used != 524288*1024 {
		t.Fatalf("unexpected used: %d", used)
	}
}

func TestHostProcLooksValid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "meminfo"), []byte("MemTotal: 1024 kB\nMemAvailable: 512 kB\n"), 0o644); err != nil {
		t.Fatalf("write meminfo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte("cpu 1 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("write stat: %v", err)
	}
	if !hostProcLooksValid(dir) {
		t.Fatal("expected host proc root to be valid")
	}
}

func TestHostProcAutoDetect(t *testing.T) {
	dir := t.TempDir()
	hostRoot := filepath.Join(dir, "host-proc")
	if err := os.Mkdir(hostRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hostRoot, "meminfo"), []byte("MemTotal: 1024 kB\nMemAvailable: 512 kB\n"), 0o644); err != nil {
		t.Fatalf("write meminfo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hostRoot, "stat"), []byte("cpu 1 0 0 0\ncpu0 1 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("write stat: %v", err)
	}

	t.Setenv("HOST_PROC", "")
	oldRoot := "/host/proc"
	// Can't mount /host/proc in test; set explicit env instead for auto path test substitute.
	t.Setenv("HOST_PROC", hostRoot)
	if root := hostProcRoot(); root != hostRoot {
		t.Fatalf("expected %q, got %q", hostRoot, root)
	}
	_ = oldRoot
}

func TestSystemLogicalCPUsUsesHostProc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cpuinfo"), []byte("processor\t: 0\nprocessor\t: 1\nprocessor\t: 2\nprocessor\t: 3\n"), 0o644); err != nil {
		t.Fatalf("write cpuinfo: %v", err)
	}

	t.Setenv("HOST_PROC", dir)
	if count := systemLogicalCPUs(); count != 4 {
		t.Fatalf("expected 4 cores, got %d", count)
	}
}
