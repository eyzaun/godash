package collector

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/eyzaun/godash/internal/models"
)

// CPUCollector handles CPU metrics collection
type CPUCollector struct {
	lastSample  *cpuSample
	sampleCount int
}

// cpuSample represents a CPU sample for calculating usage
type cpuSample struct {
	timestamp time.Time
	cpuTimes  []cpu.TimesStat
}

// NewCPUCollector creates a new CPU collector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		sampleCount: 0,
	}
}

// GetCPUMetrics collects CPU usage metrics
func (c *CPUCollector) GetCPUMetrics() (*models.CPUMetrics, error) {
	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	// Get CPU count
	logicalCount, err := cpu.Counts(true)  // logical cores
	if err != nil {
		return nil, fmt.Errorf("failed to get logical CPU count: %w", err)
	}

	// physicalCount, err := cpu.Counts(false) // physical cores
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get physical CPU count: %w", err)
	// }

	// Get CPU usage percentage
	usage, err := c.getCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	// Get per-core usage
	perCoreUsage, err := c.getPerCoreUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get per-core usage: %w", err)
	}

	// Get load average (Unix systems only)
	loadAvg, err := c.getLoadAverage()
	if err != nil {
		// Load average might not be available on all systems
		loadAvg = []float64{0, 0, 0}
	}

	// Get CPU frequency
	frequency := float64(0)
	if len(cpuInfo) > 0 {
		frequency = cpuInfo[0].Mhz
	}

	return &models.CPUMetrics{
		Usage:     usage,
		Cores:     logicalCount,
		CoreUsage: perCoreUsage,
		LoadAvg:   loadAvg,
		Frequency: frequency,
	}, nil
}

// getCPUUsage calculates overall CPU usage percentage
func (c *CPUCollector) getCPUUsage() (float64, error) {
	// Use gopsutil's built-in CPU percent with timeout for reliability
	percentages, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU percentage: %w", err)
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU percentage data received")
	}

	return percentages[0], nil
}

// getPerCoreUsage gets CPU usage for each core
func (c *CPUCollector) getPerCoreUsage() ([]float64, error) {
	// Get per-CPU usage with shorter timeout for better responsiveness
	percentages, err := cpu.Percent(500*time.Millisecond, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get per-core CPU percentage: %w", err)
	}

	return percentages, nil
}

// getCPUUsageAdvanced calculates CPU usage using CPU times (more precise)
func (c *CPUCollector) getCPUUsageAdvanced() (float64, error) {
	// Get current CPU times
	currentTimes, err := cpu.Times(false)
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU times: %w", err)
	}

	if len(currentTimes) == 0 {
		return 0, fmt.Errorf("no CPU times data received")
	}

	currentSample := &cpuSample{
		timestamp: time.Now(),
		cpuTimes:  currentTimes,
	}

	// If this is the first sample, store it and return 0
	if c.lastSample == nil {
		c.lastSample = currentSample
		return 0, nil
	}

	// Calculate CPU usage between samples
	usage := c.calculateCPUUsage(c.lastSample.cpuTimes[0], currentSample.cpuTimes[0])
	
	// Update last sample
	c.lastSample = currentSample
	c.sampleCount++

	return usage, nil
}

// calculateCPUUsage calculates CPU usage percentage between two CPU time samples
func (c *CPUCollector) calculateCPUUsage(prev, curr cpu.TimesStat) float64 {
	// Calculate total time differences
	prevTotal := prev.User + prev.System + prev.Nice + prev.Iowait + prev.Irq + prev.Softirq + prev.Steal + prev.Idle
	currTotal := curr.User + curr.System + curr.Nice + curr.Iowait + curr.Irq + curr.Softirq + curr.Steal + curr.Idle

	// Calculate idle time differences
	prevIdle := prev.Idle + prev.Iowait
	currIdle := curr.Idle + curr.Iowait

	// Calculate differences
	totalDiff := currTotal - prevTotal
	idleDiff := currIdle - prevIdle

	// Avoid division by zero
	if totalDiff == 0 {
		return 0
	}

	// Calculate usage percentage
	usage := (totalDiff - idleDiff) / totalDiff * 100

	// Ensure usage is within valid range
	if usage < 0 {
		usage = 0
	} else if usage > 100 {
		usage = 100
	}

	return usage
}

// getLoadAverage gets system load average (Unix systems only)
func (c *CPUCollector) getLoadAverage() ([]float64, error) {
	// Skip load average on Windows
	if runtime.GOOS == "windows" {
		return []float64{0, 0, 0}, nil
	}

	avg, err := load.Avg()
	if err != nil {
		return nil, fmt.Errorf("failed to get load average: %w", err)
	}

	return []float64{avg.Load1, avg.Load5, avg.Load15}, nil
}

// GetCPUTemperature gets CPU temperature (if available)
func (c *CPUCollector) GetCPUTemperature() (float64, error) {
	// This is platform-specific and might not be available
	// For now, return 0 as temperature monitoring requires platform-specific implementations
	return 0, fmt.Errorf("CPU temperature monitoring not implemented")
}

// GetCPUInfo gets detailed CPU information
func (c *CPUCollector) GetCPUInfo() ([]cpu.InfoStat, error) {
	return cpu.Info()
}

// Reset resets the collector state
func (c *CPUCollector) Reset() {
	c.lastSample = nil
	c.sampleCount = 0
}

// GetSampleCount returns the number of samples collected
func (c *CPUCollector) GetSampleCount() int {
	return c.sampleCount
}

// IsReady checks if the collector has enough samples for accurate measurements
func (c *CPUCollector) IsReady() bool {
	return c.sampleCount > 0
}