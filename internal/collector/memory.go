package collector

import (
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/eyzaun/godash/internal/models"
)

// MemoryCollector handles memory metrics collection
type MemoryCollector struct {
	// Add any state needed for memory collection
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{}
}

// GetMemoryMetrics collects memory usage metrics
func (m *MemoryCollector) GetMemoryMetrics() (*models.MemoryMetrics, error) {
	// Get virtual memory statistics
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	// Get swap memory statistics
	swapStat, err := mem.SwapMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get swap memory stats: %w", err)
	}

	// Calculate memory usage percentage
	usagePercent := float64(vmStat.Used) / float64(vmStat.Total) * 100
	if vmStat.Total == 0 {
		usagePercent = 0
	}

	// Calculate swap usage percentage
	swapPercent := float64(0)
	if swapStat.Total > 0 {
		swapPercent = float64(swapStat.Used) / float64(swapStat.Total) * 100
	}

	return &models.MemoryMetrics{
		Total:       vmStat.Total,
		Used:        vmStat.Used,
		Available:   vmStat.Available,
		Free:        vmStat.Free,
		Cached:      vmStat.Cached,
		Buffers:     vmStat.Buffers,
		Percent:     usagePercent,
		SwapTotal:   swapStat.Total,
		SwapUsed:    swapStat.Used,
		SwapPercent: swapPercent,
	}, nil
}

// GetGoMemoryStats gets Go runtime memory statistics
func (m *MemoryCollector) GetGoMemoryStats() (*runtime.MemStats, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return &memStats, nil
}

// GetDetailedMemoryInfo gets detailed memory information
func (m *MemoryCollector) GetDetailedMemoryInfo() (*DetailedMemoryInfo, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	swapStat, err := mem.SwapMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get swap memory stats: %w", err)
	}

	goStats, err := m.GetGoMemoryStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get Go memory stats: %w", err)
	}

	return &DetailedMemoryInfo{
		Virtual: *vmStat,
		Swap:    *swapStat,
		Go:      *goStats,
	}, nil
}

// DetailedMemoryInfo contains comprehensive memory information
type DetailedMemoryInfo struct {
	Virtual mem.VirtualMemoryStat `json:"virtual"`
	Swap    mem.SwapMemoryStat    `json:"swap"`
	Go      runtime.MemStats      `json:"go_runtime"`
}

// GetMemoryPressure calculates memory pressure level
func (m *MemoryCollector) GetMemoryPressure() (MemoryPressureLevel, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return PressureUnknown, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	usagePercent := float64(vmStat.Used) / float64(vmStat.Total) * 100

	switch {
	case usagePercent < 50:
		return PressureLow, nil
	case usagePercent < 75:
		return PressureModerate, nil
	case usagePercent < 90:
		return PressureHigh, nil
	default:
		return PressureCritical, nil
	}
}

// MemoryPressureLevel represents different levels of memory pressure
type MemoryPressureLevel int

const (
	PressureUnknown MemoryPressureLevel = iota
	PressureLow
	PressureModerate
	PressureHigh
	PressureCritical
)

// String returns string representation of memory pressure level
func (mpl MemoryPressureLevel) String() string {
	switch mpl {
	case PressureLow:
		return "low"
	case PressureModerate:
		return "moderate"
	case PressureHigh:
		return "high"
	case PressureCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// GetMemoryTrend analyzes memory usage trend
func (m *MemoryCollector) GetMemoryTrend(samples []models.MemoryMetrics) MemoryTrend {
	if len(samples) < 2 {
		return TrendStable
	}

	// Calculate average change over the samples
	var totalChange float64
	for i := 1; i < len(samples); i++ {
		change := samples[i].Percent - samples[i-1].Percent
		totalChange += change
	}

	avgChange := totalChange / float64(len(samples)-1)

	switch {
	case avgChange > 2:
		return TrendIncreasing
	case avgChange < -2:
		return TrendDecreasing
	default:
		return TrendStable
	}
}

// MemoryTrend represents memory usage trend
type MemoryTrend int

const (
	TrendStable MemoryTrend = iota
	TrendIncreasing
	TrendDecreasing
)

// String returns string representation of memory trend
func (mt MemoryTrend) String() string {
	switch mt {
	case TrendIncreasing:
		return "increasing"
	case TrendDecreasing:
		return "decreasing"
	default:
		return "stable"
	}
}

// FormatBytes formats bytes into human readable format
func (m *MemoryCollector) FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// GetMemoryEfficiency calculates memory efficiency metrics
func (m *MemoryCollector) GetMemoryEfficiency() (*MemoryEfficiency, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	goStats, err := m.GetGoMemoryStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get Go memory stats: %w", err)
	}

	return &MemoryEfficiency{
		CacheHitRatio:    calculateCacheHitRatio(vmStat),
		MemoryFragmentation: calculateFragmentation(goStats),
		BufferEfficiency: calculateBufferEfficiency(vmStat),
	}, nil
}

// MemoryEfficiency contains memory efficiency metrics
type MemoryEfficiency struct {
	CacheHitRatio       float64 `json:"cache_hit_ratio"`
	MemoryFragmentation float64 `json:"memory_fragmentation"`
	BufferEfficiency    float64 `json:"buffer_efficiency"`
}

// calculateCacheHitRatio calculates cache hit ratio (simplified)
func calculateCacheHitRatio(vmStat *mem.VirtualMemoryStat) float64 {
	if vmStat.Total == 0 {
		return 0
	}
	// Simplified calculation based on cached memory
	return float64(vmStat.Cached) / float64(vmStat.Total) * 100
}

// calculateFragmentation calculates memory fragmentation (simplified)
func calculateFragmentation(goStats *runtime.MemStats) float64 {
	if goStats.HeapSys == 0 {
		return 0
	}
	// Calculate fragmentation as unused heap space percentage
	fragmentation := float64(goStats.HeapSys-goStats.HeapInuse) / float64(goStats.HeapSys) * 100
	return fragmentation
}

// calculateBufferEfficiency calculates buffer efficiency
func calculateBufferEfficiency(vmStat *mem.VirtualMemoryStat) float64 {
	if vmStat.Total == 0 {
		return 0
	}
	// Buffer efficiency as percentage of total memory
	return float64(vmStat.Buffers) / float64(vmStat.Total) * 100
}

// IsMemoryHealthy checks if memory usage is within healthy limits
func (m *MemoryCollector) IsMemoryHealthy() (bool, string, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return false, "", fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	usagePercent := float64(vmStat.Used) / float64(vmStat.Total) * 100

	switch {
	case usagePercent > 95:
		return false, "Critical: Memory usage above 95%", nil
	case usagePercent > 85:
		return false, "Warning: Memory usage above 85%", nil
	case vmStat.Available < 100*1024*1024: // Less than 100MB available
		return false, "Warning: Less than 100MB available memory", nil
	default:
		return true, "Healthy", nil
	}
}