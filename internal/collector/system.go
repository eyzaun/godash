package collector

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/eyzaun/godash/internal/models"
)

// Collector interface defines the contract for system metrics collection
type Collector interface {
	// GetSystemMetrics collects current system metrics
	GetSystemMetrics() (*models.SystemMetrics, error)
	
	// GetSystemInfo collects system information
	GetSystemInfo() (*models.SystemInfo, error)
	
	// GetMetricsSnapshot collects a complete system snapshot
	GetMetricsSnapshot() (*models.MetricsSnapshot, error)
	
	// StartCollection starts continuous metrics collection
	StartCollection(ctx context.Context, interval time.Duration) <-chan *models.SystemMetrics
	
	// GetTopProcesses gets top processes by CPU or memory usage
	GetTopProcesses(count int, sortBy string) ([]models.ProcessInfo, error)
	
	// IsHealthy checks if the system is healthy
	IsHealthy() (bool, []string, error)
}

// SystemCollector implements the Collector interface
type SystemCollector struct {
	cpuCollector    *CPUCollector
	memoryCollector *MemoryCollector
	diskCollector   *DiskCollector
	
	// Configuration
	collectInterval time.Duration
	enabledMetrics  map[string]bool
	
	// State
	mutex           sync.RWMutex
	lastCollection  time.Time
	collectionCount int64
	errors          []error
}

// CollectorConfig holds configuration for the system collector
type CollectorConfig struct {
	CollectInterval time.Duration `json:"collect_interval"`
	EnableCPU       bool          `json:"enable_cpu"`
	EnableMemory    bool          `json:"enable_memory"`
	EnableDisk      bool          `json:"enable_disk"`
	EnableNetwork   bool          `json:"enable_network"`
	EnableProcesses bool          `json:"enable_processes"`
}

// DefaultCollectorConfig returns default collector configuration
func DefaultCollectorConfig() *CollectorConfig {
	return &CollectorConfig{
		CollectInterval: 30 * time.Second,
		EnableCPU:       true,
		EnableMemory:    true,
		EnableDisk:      true,
		EnableNetwork:   true,
		EnableProcesses: true,
	}
}

// NewSystemCollector creates a new system collector
func NewSystemCollector(config *CollectorConfig) *SystemCollector {
	if config == nil {
		config = DefaultCollectorConfig()
	}

	return &SystemCollector{
		cpuCollector:    NewCPUCollector(),
		memoryCollector: NewMemoryCollector(),
		diskCollector:   NewDiskCollector(),
		collectInterval: config.CollectInterval,
		enabledMetrics: map[string]bool{
			"cpu":       config.EnableCPU,
			"memory":    config.EnableMemory,
			"disk":      config.EnableDisk,
			"network":   config.EnableNetwork,
			"processes": config.EnableProcesses,
		},
		lastCollection: time.Now(),
		errors:         make([]error, 0),
	}
}

// GetSystemMetrics collects current system metrics
func (sc *SystemCollector) GetSystemMetrics() (*models.SystemMetrics, error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	var metrics models.SystemMetrics
	var collectErrors []error

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
		collectErrors = append(collectErrors, fmt.Errorf("failed to get hostname: %w", err))
	}

	// Get uptime
	uptime, err := host.Uptime()
	if err != nil {
		uptime = 0
		collectErrors = append(collectErrors, fmt.Errorf("failed to get uptime: %w", err))
	}

	// Set basic info
	metrics.Hostname = hostname
	metrics.Uptime = time.Duration(uptime) * time.Second
	metrics.Timestamp = time.Now()

	// Collect CPU metrics
	if sc.enabledMetrics["cpu"] {
		cpuMetrics, err := sc.cpuCollector.GetCPUMetrics()
		if err != nil {
			collectErrors = append(collectErrors, fmt.Errorf("failed to collect CPU metrics: %w", err))
			// Set default values to avoid nil pointer issues
			metrics.CPU = models.CPUMetrics{
				Usage:     0,
				Cores:     1,
				CoreUsage: []float64{0},
				LoadAvg:   []float64{0, 0, 0},
				Frequency: 0,
			}
		} else {
			metrics.CPU = *cpuMetrics
		}
	}

	// Collect Memory metrics
	if sc.enabledMetrics["memory"] {
		memoryMetrics, err := sc.memoryCollector.GetMemoryMetrics()
		if err != nil {
			collectErrors = append(collectErrors, fmt.Errorf("failed to collect memory metrics: %w", err))
			// Set default values
			metrics.Memory = models.MemoryMetrics{
				Total:       1,
				Used:        0,
				Available:   1,
				Free:        1,
				Cached:      0,
				Buffers:     0,
				Percent:     0,
				SwapTotal:   0,
				SwapUsed:    0,
				SwapPercent: 0,
			}
		} else {
			metrics.Memory = *memoryMetrics
		}
	}

	// Collect Disk metrics
	if sc.enabledMetrics["disk"] {
		diskMetrics, err := sc.diskCollector.GetDiskMetrics()
		if err != nil {
			collectErrors = append(collectErrors, fmt.Errorf("failed to collect disk metrics: %w", err))
			// Set default values
			metrics.Disk = models.DiskMetrics{
				Total:      1,
				Used:       0,
				Free:       1,
				Percent:    0,
				Partitions: []models.PartitionInfo{
					{
						Device:     "default",
						Mountpoint: "/",
						Fstype:     "unknown",
						Total:      1,
						Used:       0,
						Free:       1,
						Percent:    0,
					},
				},
				IOStats: models.DiskIOStats{},
			}
		} else {
			metrics.Disk = *diskMetrics
		}
	}

	// Collect Network metrics
	if sc.enabledMetrics["network"] {
		networkMetrics, err := sc.getNetworkMetrics()
		if err != nil {
			collectErrors = append(collectErrors, fmt.Errorf("failed to collect network metrics: %w", err))
			// Set default values
			metrics.Network = models.NetworkMetrics{
				Interfaces:    []models.NetworkInterface{},
				TotalSent:     0,
				TotalReceived: 0,
			}
		} else {
			metrics.Network = *networkMetrics
		}
	}

	// Update collection stats
	sc.lastCollection = time.Now()
	sc.collectionCount++
	sc.errors = collectErrors

	// Return error only if all critical collections failed
	if len(collectErrors) > 0 && len(collectErrors) >= 3 {
		return &metrics, fmt.Errorf("multiple metric collections failed: %v", collectErrors)
	}

	return &metrics, nil
}

// getNetworkMetrics collects network usage metrics
func (sc *SystemCollector) getNetworkMetrics() (*models.NetworkMetrics, error) {
	// Get network I/O counters
	netCounters, err := net.IOCounters(true) // per interface
	if err != nil {
		return nil, fmt.Errorf("failed to get network counters: %w", err)
	}

	var totalSent, totalRecv uint64
	var interfaces []models.NetworkInterface

	for _, counter := range netCounters {
		// Skip loopback interfaces
		if counter.Name == "lo" || counter.Name == "lo0" {
			continue
		}

		totalSent += counter.BytesSent
		totalRecv += counter.BytesRecv

		interfaces = append(interfaces, models.NetworkInterface{
			Name:        counter.Name,
			BytesSent:   counter.BytesSent,
			BytesRecv:   counter.BytesRecv,
			PacketsSent: counter.PacketsSent,
			PacketsRecv: counter.PacketsRecv,
			Errors:      counter.Errin + counter.Errout,
			Drops:       counter.Dropin + counter.Dropout,
		})
	}

	return &models.NetworkMetrics{
		Interfaces:    interfaces,
		TotalSent:     totalSent,
		TotalReceived: totalRecv,
	}, nil
}

// GetSystemInfo collects system information
func (sc *SystemCollector) GetSystemInfo() (*models.SystemInfo, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	// Get process count
	processes, err := process.Pids()
	if err != nil {
		processes = []int32{} // Continue with empty process list
	}

	return &models.SystemInfo{
		Platform:        hostInfo.Platform,
		PlatformFamily:  hostInfo.PlatformFamily,
		PlatformVersion: hostInfo.PlatformVersion,
		KernelVersion:   hostInfo.KernelVersion,
		KernelArch:      hostInfo.KernelArch,
		HostID:          hostInfo.HostID,
		BootTime:        time.Unix(int64(hostInfo.BootTime), 0),
		Processes:       uint64(len(processes)),
	}, nil
}

// GetMetricsSnapshot collects a complete system snapshot
func (sc *SystemCollector) GetMetricsSnapshot() (*models.MetricsSnapshot, error) {
	systemMetrics, err := sc.GetSystemMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get system metrics: %w", err)
	}

	systemInfo, err := sc.GetSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	topProcesses, err := sc.GetTopProcesses(10, "cpu")
	if err != nil {
		// Continue with empty process list if this fails
		topProcesses = []models.ProcessInfo{}
	}

	return &models.MetricsSnapshot{
		SystemMetrics: *systemMetrics,
		SystemInfo:    *systemInfo,
		TopProcesses:  topProcesses,
		Timestamp:     time.Now(),
	}, nil
}

// StartCollection starts continuous metrics collection
func (sc *SystemCollector) StartCollection(ctx context.Context, interval time.Duration) <-chan *models.SystemMetrics {
	if interval == 0 {
		interval = sc.collectInterval
	}

	ch := make(chan *models.SystemMetrics, 10) // Buffered channel
	
	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Send initial metrics immediately
		if metrics, err := sc.GetSystemMetrics(); err == nil {
			select {
			case ch <- metrics:
			case <-ctx.Done():
				return
			}
		}

		// Send metrics at intervals
		for {
			select {
			case <-ticker.C:
				metrics, err := sc.GetSystemMetrics()
				if err != nil {
					// Log error but continue collection
					continue
				}
				
				select {
				case ch <- metrics:
				case <-ctx.Done():
					return
				default:
					// Channel is full, skip this update
				}
				
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// GetTopProcesses gets top processes by CPU or memory usage
func (sc *SystemCollector) GetTopProcesses(count int, sortBy string) ([]models.ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("failed to get process PIDs: %w", err)
	}

	var processes []models.ProcessInfo

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue // Skip processes we can't access
		}

		name, err := proc.Name()
		if err != nil {
			name = "unknown"
		}

		cpuPercent, err := proc.CPUPercent()
		if err != nil {
			cpuPercent = 0
		}

		memInfo, err := proc.MemoryInfo()
		if err != nil {
			memInfo = &process.MemoryInfoStat{RSS: 0}
		}

			   status, err := proc.Status()
			   var statusStr string
			   if err != nil || len(status) == 0 {
					   statusStr = "unknown"
			   } else {
					   statusStr = status[0]
			   }

			   processes = append(processes, models.ProcessInfo{
					   PID:         pid,
					   Name:        name,
					   CPUPercent:  cpuPercent,
					   MemoryBytes: memInfo.RSS,
					   Status:      statusStr,
			   })

		// Limit the number of processes to check for performance
		if len(processes) > count*3 {
			break
		}
	}

	// Sort processes based on sortBy parameter
	sc.sortProcesses(processes, sortBy)

	// Return top N processes
	if len(processes) > count {
		processes = processes[:count]
	}

	return processes, nil
}

// sortProcesses sorts processes by the specified criteria
func (sc *SystemCollector) sortProcesses(processes []models.ProcessInfo, sortBy string) {
	switch sortBy {
	case "memory":
		// Sort by memory usage (descending)
		for i := 0; i < len(processes)-1; i++ {
			for j := i + 1; j < len(processes); j++ {
				if processes[i].MemoryBytes < processes[j].MemoryBytes {
					processes[i], processes[j] = processes[j], processes[i]
				}
			}
		}
	default: // "cpu" or anything else defaults to CPU
		// Sort by CPU usage (descending)
		for i := 0; i < len(processes)-1; i++ {
			for j := i + 1; j < len(processes); j++ {
				if processes[i].CPUPercent < processes[j].CPUPercent {
					processes[i], processes[j] = processes[j], processes[i]
				}
			}
		}
	}
}

// IsHealthy checks if the system is healthy
func (sc *SystemCollector) IsHealthy() (bool, []string, error) {
	issues := make([]string, 0) // Initialize empty slice, not nil
	healthy := true

	// Check memory health
	if sc.enabledMetrics["memory"] {
		memHealthy, memMsg, err := sc.memoryCollector.IsMemoryHealthy()
		if err != nil {
			return false, []string{"Failed to check memory health"}, err
		}
		if !memHealthy {
			healthy = false
			issues = append(issues, "Memory: "+memMsg)
		}
	}

	// Check disk health
	if sc.enabledMetrics["disk"] {
		diskHealth, err := sc.diskCollector.GetDiskHealth()
		if err != nil {
			return false, []string{"Failed to check disk health"}, err
		}
		if diskHealth.OverallHealth != "healthy" {
			healthy = false
			issues = append(issues, diskHealth.Critical...)
			issues = append(issues, diskHealth.Warnings...)
		}
	}

	// Check if CPU usage is extremely high
	if sc.enabledMetrics["cpu"] {
		metrics, err := sc.GetSystemMetrics()
		if err == nil && metrics.CPU.Usage > 95 {
			healthy = false
			issues = append(issues, fmt.Sprintf("CPU: Usage above 95%% (%.1f%%)", metrics.CPU.Usage))
		}
	}

	return healthy, issues, nil
}

// GetCollectionStats returns collection statistics
func (sc *SystemCollector) GetCollectionStats() map[string]interface{} {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	return map[string]interface{}{
		"last_collection":  sc.lastCollection,
		"collection_count": sc.collectionCount,
		"error_count":      len(sc.errors),
		"enabled_metrics":  sc.enabledMetrics,
	}
}

// GetLastErrors returns the last collection errors
func (sc *SystemCollector) GetLastErrors() []error {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	// Return a copy of the errors slice
	errors := make([]error, len(sc.errors))
	copy(errors, sc.errors)
	return errors
}

// SetEnabledMetrics sets which metrics to collect
func (sc *SystemCollector) SetEnabledMetrics(metrics map[string]bool) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	for key, value := range metrics {
		sc.enabledMetrics[key] = value
	}
}

// Reset resets the collector state
func (sc *SystemCollector) Reset() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.cpuCollector.Reset()
	sc.collectionCount = 0
	sc.errors = make([]error, 0)
	sc.lastCollection = time.Now()
}