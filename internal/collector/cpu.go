package collector

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/eyzaun/godash/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
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
	logicalCount, err := cpu.Counts(true) // logical cores
	if err != nil {
		return nil, fmt.Errorf("failed to get logical CPU count: %w", err)
	}

	// Get CPU usage percentage - use advanced method for better accuracy
	usage, err := c.getCPUUsageAdvanced()
	if err != nil {
		// Fallback to simple method if advanced fails
		usage, err = c.getCPUUsage()
		if err != nil {
			return nil, fmt.Errorf("failed to get CPU usage: %w", err)
		}
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

	// Get CPU temperature
	temperature, err := c.GetCPUTemperature()
	if err != nil {
		// Temperature might not be available on all systems, use 0 as default
		temperature = 0
	}

	return &models.CPUMetrics{
		Usage:       usage,
		Cores:       logicalCount,
		CoreUsage:   perCoreUsage,
		LoadAvg:     loadAvg,
		Frequency:   frequency,
		Temperature: temperature,
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

// GetCPUTemperature gets CPU temperature (platform-specific)
func (c *CPUCollector) GetCPUTemperature() (float64, error) {
	// Try to get CPU temperature from different sources

	// Windows: Use WMI
	if runtime.GOOS == "windows" {
		return c.getCPUTemperatureWindows()
	}

	// Linux: Use thermal zones
	if runtime.GOOS == "linux" {
		return c.getCPUTemperatureLinux()
	}

	// macOS: Use sysctl
	if runtime.GOOS == "darwin" {
		return c.getCPUTemperatureDarwin()
	}

	return 0, fmt.Errorf("CPU temperature monitoring not supported on %s", runtime.GOOS)
}

// getCPUTemperatureWindows gets temperature on Windows using WMI
func (c *CPUCollector) getCPUTemperatureWindows() (float64, error) {
	// Use gopsutil to get temperature from thermal sensors
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return 0, fmt.Errorf("failed to get temperature sensors: %w", err)
	}

	// Find CPU temperature sensor
	for _, temp := range temps {
		// Look for common CPU temperature sensor names
		sensorName := strings.ToLower(temp.SensorKey)
		if strings.Contains(sensorName, "cpu") ||
			strings.Contains(sensorName, "core") ||
			strings.Contains(sensorName, "package") ||
			strings.Contains(sensorName, "processor") {
			if temp.Temperature > 0 && temp.Temperature < 120 { // Reasonable range
				return temp.Temperature, nil
			}
		}
	}

	return 0, fmt.Errorf("no CPU temperature sensor found")
}

// getCPUTemperatureLinux gets temperature on Linux
func (c *CPUCollector) getCPUTemperatureLinux() (float64, error) {
	// Try thermal zones first
	temps, err := host.SensorsTemperatures()
	if err == nil {
		for _, temp := range temps {
			sensorName := strings.ToLower(temp.SensorKey)
			if strings.Contains(sensorName, "cpu") ||
				strings.Contains(sensorName, "core") ||
				strings.Contains(sensorName, "package") {
				if temp.Temperature > 0 && temp.Temperature < 120 {
					return temp.Temperature, nil
				}
			}
		}
	}

	// Fallback: try reading from /sys/class/thermal/
	thermalFiles := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
		"/sys/class/thermal/thermal_zone2/temp",
	}

	for _, file := range thermalFiles {
		if data, err := os.ReadFile(file); err == nil {
			if temp, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
				// Temperature is in millidegrees Celsius
				tempC := temp / 1000.0
				if tempC > 0 && tempC < 120 {
					return tempC, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("no CPU temperature found")
}

// getCPUTemperatureDarwin gets temperature on macOS
func (c *CPUCollector) getCPUTemperatureDarwin() (float64, error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return 0, fmt.Errorf("failed to get temperature sensors: %w", err)
	}

	for _, temp := range temps {
		sensorName := strings.ToLower(temp.SensorKey)
		if strings.Contains(sensorName, "cpu") || strings.Contains(sensorName, "core") {
			if temp.Temperature > 0 && temp.Temperature < 120 {
				return temp.Temperature, nil
			}
		}
	}

	return 0, fmt.Errorf("no CPU temperature sensor found")
}

// Reset resets the collector state
func (c *CPUCollector) Reset() {
	c.lastSample = nil
	c.sampleCount = 0
}
