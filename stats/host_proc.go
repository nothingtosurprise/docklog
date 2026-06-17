package stats

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func hostProcRoot() string {
	if root := strings.TrimSpace(os.Getenv("HOST_PROC")); root != "" {
		return root
	}
	if hostProcLooksValid("/host/proc") {
		return "/host/proc"
	}
	return ""
}

func hostProcLooksValid(root string) bool {
	if root == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "meminfo")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "stat")); err != nil {
		return false
	}
	return true
}

func systemMemory() (total, used uint64) {
	if root := hostProcRoot(); root != "" {
		if total, used, ok := memoryFromProc(filepath.Join(root, "meminfo")); ok {
			return total, used
		}
	}
	if total, used, ok := k8sHostMemory(); ok {
		return total, used
	}

	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0
	}

	used = v.Used
	if v.Available > 0 && v.Total > v.Available {
		used = v.Total - v.Available
	}
	return v.Total, used
}

func systemLogicalCPUs() int {
	if root := hostProcRoot(); root != "" {
		if count, ok := logicalCPUsFromProc(filepath.Join(root, "cpuinfo")); ok && count > 0 {
			return count
		}
		if count, ok := logicalCPUsFromStat(filepath.Join(root, "stat")); ok && count > 0 {
			return count
		}
	}
	if count, ok := k8sHostLogicalCPUs(); ok {
		return count
	}

	cores, err := cpu.Counts(true)
	if err != nil || cores == 0 {
		return runtime.NumCPU()
	}
	return cores
}

func memoryFromProc(meminfoPath string) (total, used uint64, ok bool) {
	file, err := os.Open(meminfoPath)
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()

	var memTotalKB, memAvailableKB uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			memTotalKB = parseProcMeminfoValue(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			memAvailableKB = parseProcMeminfoValue(line)
		}
		if memTotalKB > 0 && memAvailableKB > 0 {
			break
		}
	}
	if err := scanner.Err(); err != nil || memTotalKB == 0 {
		return 0, 0, false
	}

	total = memTotalKB * 1024
	if memAvailableKB > 0 && memTotalKB >= memAvailableKB {
		used = (memTotalKB - memAvailableKB) * 1024
		return total, used, true
	}

	// Older kernels without MemAvailable: caller falls back to gopsutil.
	return 0, 0, false
}

func parseProcMeminfoValue(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	value, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func logicalCPUsFromProc(cpuinfoPath string) (int, bool) {
	file, err := os.Open(cpuinfoPath)
	if err != nil {
		return 0, false
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "processor") {
			count++
		}
	}
	if err := scanner.Err(); err != nil || count == 0 {
		return 0, false
	}
	return count, true
}

func logicalCPUsFromStat(statPath string) (int, bool) {
	file, err := os.Open(statPath)
	if err != nil {
		return 0, false
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if name == "cpu" {
			continue
		}
		if strings.HasPrefix(name, "cpu") {
			if _, err := strconv.Atoi(strings.TrimPrefix(name, "cpu")); err == nil {
				count++
			}
		}
	}
	if err := scanner.Err(); err != nil || count == 0 {
		return 0, false
	}
	return count, true
}
