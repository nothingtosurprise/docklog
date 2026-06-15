package stats

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

const (
	systemSampleInterval = time.Second
	cpuSmoothingAlpha    = 0.35
)

type procCPUStat struct {
	total uint64
	idle  uint64
}

func procStatPath() string {
	if root := strings.TrimSpace(os.Getenv("HOST_PROC")); root != "" {
		return filepath.Join(root, "stat")
	}
	return "/proc/stat"
}

func readProcCPUStat() (procCPUStat, error) {
	file, err := os.Open(procStatPath())
	if err != nil {
		return procCPUStat{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return procCPUStat{}, scanner.Err()
	}
	return parseProcCPUStatLine(scanner.Text())
}

func parseProcCPUStatLine(line string) (procCPUStat, error) {
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return procCPUStat{}, os.ErrInvalid
	}

	var total uint64
	for i := 1; i < len(fields); i++ {
		value, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return procCPUStat{}, err
		}
		total += value
	}

	idle, err := strconv.ParseUint(fields[4], 10, 64)
	if err != nil {
		return procCPUStat{}, err
	}

	return procCPUStat{total: total, idle: idle}, nil
}

func sampleCPUPercent(interval time.Duration) (float64, error) {
	start, err := readProcCPUStat()
	if err != nil {
		return sampleCPUPercentGopsutil(interval)
	}

	time.Sleep(interval)

	end, err := readProcCPUStat()
	if err != nil {
		return sampleCPUPercentGopsutil(interval)
	}

	totalDelta := end.total - start.total
	if totalDelta == 0 {
		return 0, nil
	}

	idleDelta := end.idle - start.idle
	busy := (1.0 - float64(idleDelta)/float64(totalDelta)) * 100.0
	if busy < 0 {
		return 0, nil
	}
	if busy > 100 {
		return 100, nil
	}
	return busy, nil
}

func sampleCPUPercentGopsutil(interval time.Duration) (float64, error) {
	values, err := cpu.Percent(interval, false)
	if err != nil || len(values) == 0 {
		return 0, err
	}
	return values[0], nil
}

func smoothCPU(sample float64) float64 {
	if cpuSmoothed == 0 {
		cpuSmoothed = sample
		return sample
	}
	cpuSmoothed = cpuSmoothingAlpha*sample + (1-cpuSmoothingAlpha)*cpuSmoothed
	return cpuSmoothed
}
