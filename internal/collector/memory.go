package collector

import (
	"fmt"

	"github.com/eyzaun/godash/internal/models"
	"github.com/shirou/gopsutil/v3/mem"
)

// MemoryCollector handles memory metrics collection
type MemoryCollector struct {
	// Add any state needed for memory collection
}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{}
}

// GetMemoryMetrics collects current memory metrics
func (mc *MemoryCollector) GetMemoryMetrics() (*models.MemoryMetrics, error) {
	// Get virtual memory stats
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	// Get swap memory stats
	swapStat, err := mem.SwapMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get swap memory stats: %w", err)
	}

	metrics := &models.MemoryMetrics{
		Total:       vmStat.Total,
		Used:        vmStat.Used,
		Available:   vmStat.Available,
		Free:        vmStat.Free,
		Cached:      vmStat.Cached,
		Buffers:     vmStat.Buffers,
		Percent:     vmStat.UsedPercent,
		SwapTotal:   swapStat.Total,
		SwapUsed:    swapStat.Used,
		SwapPercent: swapStat.UsedPercent,
	}

	// Add comprehensive logging for debugging
	fmt.Printf("🔍 Memory Metrics Collected - Used: %.2f%% (%.1f GB / %.1f GB), Available: %.1f GB, Swap: %.2f%% (%.1f GB / %.1f GB)\n",
		metrics.Percent, float64(metrics.Used)/(1024*1024*1024), float64(metrics.Total)/(1024*1024*1024),
		float64(metrics.Available)/(1024*1024*1024), metrics.SwapPercent,
		float64(metrics.SwapUsed)/(1024*1024*1024), float64(metrics.SwapTotal)/(1024*1024*1024))

	return metrics, nil
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
