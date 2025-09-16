package collector

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/eyzaun/godash/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	wmi "github.com/yusufpapurcu/wmi"
)

// CPUCollector handles CPU metrics collection
type CPUCollector struct {
	sampleCount int
}

// NewCPUCollector creates a new CPU collector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		sampleCount: 0,
	}
}

// GetCPUMetrics collects CPU usage metrics
func (c *CPUCollector) GetCPUMetrics() (*models.CPUMetrics, error) {
	// Get CPU usage percentages
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	var totalUsage float64
	if len(percentages) > 0 {
		totalUsage = percentages[0]
	}

	// Get per-core usage
	corePercentages, err := cpu.Percent(0, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get per-core CPU usage: %w", err)
	}

	// Get CPU count
	count, err := cpu.Counts(true)
	if err != nil {
		count = 1 // fallback
	}

	// Get CPU frequency
	freq := float64(0)
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		freq = cpuInfo[0].Mhz
	}

	// Get load averages (Unix systems only)
	loadAvg := []float64{0, 0, 0}
	if runtime.GOOS != "windows" {
		if avg, err := load.Avg(); err == nil {
			loadAvg = []float64{avg.Load1, avg.Load5, avg.Load15}
		}
	}

	// Get CPU temperature (if available)
	temp, err := c.GetCPUTemperature()
	if err != nil {
		temp = 0 // fallback
	}

	metrics := &models.CPUMetrics{
		Usage:       totalUsage,
		Cores:       count,
		CoreUsage:   corePercentages,
		LoadAvg:     loadAvg,
		Frequency:   freq,
		Temperature: temp,
	}

	// Add comprehensive logging for debugging
	fmt.Printf("🔍 CPU Metrics Collected - Usage: %.2f%%, Cores: %d, Frequency: %.0f MHz, Temp: %.1f°C, LoadAvg: [%.2f, %.2f, %.2f]\n",
		metrics.Usage, metrics.Cores, metrics.Frequency, metrics.Temperature,
		metrics.LoadAvg[0], metrics.LoadAvg[1], metrics.LoadAvg[2])

	return metrics, nil
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
	// 1) Try WMI: MSAcpi_ThermalZoneTemperature (values in tenths of Kelvin)
	type thermalZone struct {
		CurrentTemperature uint32
		InstanceName       string
	}

	var zones []thermalZone
	// Query from root\WMI namespace
	if err := wmi.QueryNamespace("SELECT CurrentTemperature, InstanceName FROM MSAcpi_ThermalZoneTemperature", &zones, "root\\WMI"); err == nil {
		best := -1.0
		for _, z := range zones {
			if z.CurrentTemperature == 0 {
				continue
			}
			// Convert to Celsius
			celsius := (float64(z.CurrentTemperature) / 10.0) - 273.15
			// Filter obviously invalid values
			if celsius > 10 && celsius < 120 {
				if celsius > best {
					best = celsius
				}
			}
		}
		if best > 0 {
			return best, nil
		}
	}

	// 2) Fallback: gopsutil SensorsTemperatures (may not work on many Windows setups)
	temps, err := host.SensorsTemperatures()
	if err == nil {
		for _, temp := range temps {
			sensorName := strings.ToLower(temp.SensorKey)
			if strings.Contains(sensorName, "cpu") || strings.Contains(sensorName, "core") || strings.Contains(sensorName, "package") || strings.Contains(sensorName, "processor") {
				if temp.Temperature > 10 && temp.Temperature < 120 {
					return temp.Temperature, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("no CPU temperature sensor found via WMI or sensors")
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
	c.sampleCount = 0
}
