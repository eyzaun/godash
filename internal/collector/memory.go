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
