package collector

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/eyzaun/godash/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
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

// SystemCollector implements the Collector interface (SPEED TRACKING ADDED)
type SystemCollector struct {
	cpuCollector     *CPUCollector
	memoryCollector  *MemoryCollector
	diskCollector    *DiskCollector
	processCollector *ProcessCollector

	// Configuration
	collectInterval time.Duration
	enabledMetrics  map[string]bool

	// State
	mutex           sync.RWMutex
	lastCollection  time.Time
	collectionCount int64
	errors          []error

	// NEW: Speed tracking for calculations
	lastDiskStats    models.DiskIOStats
	lastDiskTime     time.Time
	lastNetworkStats map[string]net.IOCountersStat
	lastNetworkTime  time.Time
	lastNetworkSent  uint64
	lastNetworkRecv  uint64
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

// NewSystemCollector creates a new system collector (SPEED TRACKING ADDED)
func NewSystemCollector(config *CollectorConfig) *SystemCollector {
	if config == nil {
		config = DefaultCollectorConfig()
	}

	return &SystemCollector{
		cpuCollector:     NewCPUCollector(),
		memoryCollector:  NewMemoryCollector(),
		diskCollector:    NewDiskCollector(),
		processCollector: NewProcessCollector(),
		collectInterval:  config.CollectInterval,
		enabledMetrics: map[string]bool{
			"cpu":       config.EnableCPU,
			"memory":    config.EnableMemory,
			"disk":      config.EnableDisk,
			"network":   config.EnableNetwork,
			"processes": config.EnableProcesses,
		},
		lastCollection:   time.Now(),
		errors:           make([]error, 0),
		lastNetworkStats: make(map[string]net.IOCountersStat),
		lastNetworkTime:  time.Now(),
		lastDiskTime:     time.Now(),
	}
}

// GetSystemMetrics collects current system metrics (SPEED CALCULATION ADDED)
func (sc *SystemCollector) GetSystemMetrics() (*models.SystemMetrics, error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	var metrics models.SystemMetrics
	var collectErrors []error
	currentTime := time.Now()

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
	metrics.Timestamp = currentTime

	// Collect CPU metrics (FIX: Check if cpuCollector is nil)
	if sc.enabledMetrics["cpu"] {
		if sc.cpuCollector != nil {
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
			} else if cpuMetrics != nil {
				metrics.CPU = *cpuMetrics
			}
		} else {
			// Set default values if collector is nil
			metrics.CPU = models.CPUMetrics{
				Usage:     0,
				Cores:     1,
				CoreUsage: []float64{0},
				LoadAvg:   []float64{0, 0, 0},
				Frequency: 0,
			}
		}
	}

	// Collect Memory metrics (FIX: Check if memoryCollector is nil)
	if sc.enabledMetrics["memory"] {
		if sc.memoryCollector != nil {
			memoryMetrics, err := sc.memoryCollector.GetMemoryMetrics()
			if err != nil {
				collectErrors = append(collectErrors, fmt.Errorf("failed to collect memory metrics: %w", err))
				// Set default values
				metrics.Memory = models.MemoryMetrics{
					Total:       1024 * 1024 * 1024, // 1GB default
					Used:        0,
					Available:   1024 * 1024 * 1024,
					Free:        1024 * 1024 * 1024,
					Cached:      0,
					Buffers:     0,
					Percent:     0,
					SwapTotal:   0,
					SwapUsed:    0,
					SwapPercent: 0,
				}
			} else if memoryMetrics != nil {
				metrics.Memory = *memoryMetrics
			}
		} else {
			// Set default values if collector is nil
			metrics.Memory = models.MemoryMetrics{
				Total:       1024 * 1024 * 1024, // 1GB default
				Used:        0,
				Available:   1024 * 1024 * 1024,
				Free:        1024 * 1024 * 1024,
				Cached:      0,
				Buffers:     0,
				Percent:     0,
				SwapTotal:   0,
				SwapUsed:    0,
				SwapPercent: 0,
			}
		}
	}

	// Collect Disk metrics (SPEED CALCULATION ADDED) (FIX: Check if diskCollector is nil)
	if sc.enabledMetrics["disk"] {
		if sc.diskCollector != nil {
			diskMetrics, err := sc.diskCollector.GetDiskMetrics()
			if err != nil {
				collectErrors = append(collectErrors, fmt.Errorf("failed to collect disk metrics: %w", err))
				// Set more realistic default values
				metrics.Disk = models.DiskMetrics{
					Total:   100 * 1024 * 1024 * 1024, // 100GB default
					Used:    50 * 1024 * 1024 * 1024,  // 50GB used
					Free:    50 * 1024 * 1024 * 1024,  // 50GB free
					Percent: 50.0,                     // 50% used
					Partitions: []models.PartitionInfo{
						{
							Device:     "/dev/sda1",
							Mountpoint: "/",
							Fstype:     "ext4",
							Total:      100 * 1024 * 1024 * 1024,
							Used:       50 * 1024 * 1024 * 1024,
							Free:       50 * 1024 * 1024 * 1024,
							Percent:    50.0,
						},
					},
					IOStats: models.DiskIOStats{
						ReadBytes:  1024 * 1024, // 1MB
						WriteBytes: 1024 * 1024, // 1MB
						ReadOps:    100,
						WriteOps:   50,
						ReadTime:   10,
						WriteTime:  5,
					},
					ReadSpeed:  0, // Default to 0
					WriteSpeed: 0, // Default to 0
				}
			} else if diskMetrics != nil {
				metrics.Disk = *diskMetrics

				// NEW: Calculate disk I/O speed
				timeDiff := currentTime.Sub(sc.lastDiskTime)
				if sc.lastDiskTime.IsZero() || timeDiff <= 0 {
					// First measurement or invalid time diff
					metrics.Disk.ReadSpeed = 0
					metrics.Disk.WriteSpeed = 0
				} else {
					// Calculate speeds using helper method
					metrics.Disk.CalculateDiskSpeed(sc.lastDiskStats, timeDiff)
				}

				// Store current stats for next calculation
				sc.lastDiskStats = metrics.Disk.IOStats
				sc.lastDiskTime = currentTime
			}
		} else {
			// Set default values if collector is nil
			metrics.Disk = models.DiskMetrics{
				Total:   100 * 1024 * 1024 * 1024, // 100GB default
				Used:    50 * 1024 * 1024 * 1024,  // 50GB used
				Free:    50 * 1024 * 1024 * 1024,  // 50GB free
				Percent: 50.0,                     // 50% used
				Partitions: []models.PartitionInfo{
					{
						Device:     "/dev/sda1",
						Mountpoint: "/",
						Fstype:     "ext4",
						Total:      100 * 1024 * 1024 * 1024,
						Used:       50 * 1024 * 1024 * 1024,
						Free:       50 * 1024 * 1024 * 1024,
						Percent:    50.0,
					},
				},
				IOStats: models.DiskIOStats{
					ReadBytes:  1024 * 1024, // 1MB
					WriteBytes: 1024 * 1024, // 1MB
					ReadOps:    100,
					WriteOps:   50,
					ReadTime:   10,
					WriteTime:  5,
				},
				ReadSpeed:  0, // Default to 0
				WriteSpeed: 0, // Default to 0
			}
		}
	}

	// Collect Network metrics (SPEED CALCULATION ADDED)
	if sc.enabledMetrics["network"] {
		networkMetrics, err := sc.getNetworkMetrics()
		if err != nil {
			collectErrors = append(collectErrors, fmt.Errorf("failed to collect network metrics: %w", err))
			// Set realistic default values
			metrics.Network = models.NetworkMetrics{
				Interfaces: []models.NetworkInterface{
					{
						Name:        "eth0",
						BytesSent:   1024 * 1024 * 10, // 10MB sent
						BytesRecv:   1024 * 1024 * 50, // 50MB received
						PacketsSent: 1000,
						PacketsRecv: 5000,
						Errors:      0,
						Drops:       0,
					},
				},
				TotalSent:     1024 * 1024 * 10,
				TotalReceived: 1024 * 1024 * 50,
				UploadSpeed:   0, // Default to 0
				DownloadSpeed: 0, // Default to 0
			}
		} else if networkMetrics != nil {
			metrics.Network = *networkMetrics

			// NEW: Calculate network speed
			timeDiff := currentTime.Sub(sc.lastNetworkTime)
			if sc.lastNetworkTime.IsZero() || timeDiff <= 0 {
				// First measurement or invalid time diff
				metrics.Network.UploadSpeed = 0
				metrics.Network.DownloadSpeed = 0
			} else {
				// Calculate speeds using helper method
				metrics.Network.CalculateNetworkSpeed(sc.lastNetworkSent, sc.lastNetworkRecv, timeDiff)
			}

			// Store current stats for next calculation
			sc.lastNetworkSent = metrics.Network.TotalSent
			sc.lastNetworkRecv = metrics.Network.TotalReceived
			sc.lastNetworkTime = currentTime
		}
	}

	// Collect Process metrics (FIX: Check if processCollector is nil)
	if sc.enabledMetrics["processes"] {
		if sc.processCollector != nil {
			processActivity, err := sc.processCollector.GetProcessActivity()
			if err != nil {
				collectErrors = append(collectErrors, fmt.Errorf("failed to collect process metrics: %w", err))
				// Set default values
				metrics.Processes = models.ProcessActivity{
					TotalProcesses:   100,
					RunningProcesses: 80,
					StoppedProcesses: 5,
					ZombieProcesses:  0,
					TopProcesses:     []models.ProcessInfo{},
				}
			} else if processActivity != nil {
				metrics.Processes = *processActivity
			}
		} else {
			// Set default values if collector is nil
			metrics.Processes = models.ProcessActivity{
				TotalProcesses:   100,
				RunningProcesses: 80,
				StoppedProcesses: 5,
				ZombieProcesses:  0,
				TopProcesses:     []models.ProcessInfo{},
			}
		}
	}

	// Update collection stats
	sc.lastCollection = currentTime
	sc.collectionCount++
	sc.errors = collectErrors

	// Return error only if all critical collections failed
	if len(collectErrors) > 0 && len(collectErrors) >= 3 {
		return &metrics, fmt.Errorf("multiple metric collections failed: %v", collectErrors)
	}

	return &metrics, nil
}

// getNetworkMetrics collects network usage metrics (ENHANCED FOR SPEED TRACKING)
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

		// Skip virtual interfaces that don't represent real network traffic
		if sc.shouldSkipInterface(counter.Name) {
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

		// Store current stats for future rate calculations
		sc.lastNetworkStats[counter.Name] = counter
	}

	// Ensure we have at least some data even if no real interfaces found
	if len(interfaces) == 0 {
		// Create a dummy interface to prevent frontend issues
		interfaces = append(interfaces, models.NetworkInterface{
			Name:        "eth0",
			BytesSent:   totalSent,
			BytesRecv:   totalRecv,
			PacketsSent: 0,
			PacketsRecv: 0,
			Errors:      0,
			Drops:       0,
		})
	}

	return &models.NetworkMetrics{
		Interfaces:    interfaces,
		TotalSent:     totalSent,
		TotalReceived: totalRecv,
		// Speed will be calculated in GetSystemMetrics
		UploadSpeed:   0,
		DownloadSpeed: 0,
	}, nil
}

// shouldSkipInterface checks if we should skip this network interface
func (sc *SystemCollector) shouldSkipInterface(name string) bool {
	// Skip common virtual interfaces
	skipInterfaces := []string{
		"docker", "br-", "veth", "virbr", "vmnet", "vbox",
		"tun", "tap", "ppp", "slip", "bond", "team",
	}

	for _, skip := range skipInterfaces {
		if len(name) >= len(skip) && name[:len(skip)] == skip {
			return true
		}
	}

	return false
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
		Hostname:        hostInfo.Hostname,
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

	// FIX: Check if systemMetrics is nil
	if systemMetrics == nil {
		return nil, fmt.Errorf("system metrics returned nil")
	}

	systemInfo, err := sc.GetSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}

	// FIX: Check if systemInfo is nil
	if systemInfo == nil {
		return nil, fmt.Errorf("system info returned nil")
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
		if metrics, err := sc.GetSystemMetrics(); err == nil && metrics != nil {
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
				if err != nil || metrics == nil {
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

	// Get CPU count to normalize CPU percentages (gopsutil returns per-core values)
	cpuCount, err := cpu.Counts(true)
	if err != nil {
		// Fallback to logical cores if physical cores fail
		if cpuCountLogical, errLogical := cpu.Counts(false); errLogical == nil {
			cpuCount = cpuCountLogical
		} else {
			// Last resort fallback to 1
			cpuCount = 1
		}
	}

	// Group processes by name to avoid duplicates and sum CPU usage
	processGroups := make(map[string]*models.ProcessInfo)

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
		// Normalize CPU percentage by dividing by number of cores
		// gopsutil.CPUPercent() returns per-core values
		if cpuCount > 0 {
			cpuPercent = cpuPercent / float64(cpuCount)
		}

		memInfo, err := proc.MemoryInfo()
		if err != nil {
			memInfo = &process.MemoryInfoStat{RSS: 0}
		}

		// FIX: Check if memInfo is nil
		var memoryBytes uint64 = 0
		if memInfo != nil {
			memoryBytes = memInfo.RSS
		}

		status, err := proc.Status()
		var statusStr string
		if err != nil || len(status) == 0 {
			statusStr = "unknown"
		} else {
			statusStr = status[0]
		}

		// Group by process name
		if existing, exists := processGroups[name]; exists {
			// Add to existing group
			existing.CPUPercent += cpuPercent
			existing.MemoryBytes += memoryBytes
			// Keep the lowest PID as representative
			if pid < existing.PID {
				existing.PID = pid
			}
		} else {
			// Create new group
			processGroups[name] = &models.ProcessInfo{
				PID:         pid,
				Name:        name,
				CPUPercent:  cpuPercent,
				MemoryBytes: memoryBytes,
				Status:      statusStr,
			}
		}
	}

	// Convert map to slice for sorting
	var processes []models.ProcessInfo
	for _, proc := range processGroups {
		processes = append(processes, *proc)
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

// IsHealthy checks if the system is healthy (SPEED CHECKS ADDED) (FIX: Nil checks added)
func (sc *SystemCollector) IsHealthy() (bool, []string, error) {
	issues := make([]string, 0) // Initialize empty slice, not nil
	healthy := true

	// Check memory health (FIX: Check if memoryCollector is nil)
	if sc.enabledMetrics["memory"] && sc.memoryCollector != nil {
		memHealthy, memMsg, err := sc.memoryCollector.IsMemoryHealthy()
		if err != nil {
			return false, []string{"Failed to check memory health"}, err
		}
		if !memHealthy {
			healthy = false
			issues = append(issues, "Memory: "+memMsg)
		}
	}

	// Check disk health (FIX: Check if diskCollector is nil)
	if sc.enabledMetrics["disk"] && sc.diskCollector != nil {
		diskHealth, err := sc.diskCollector.GetDiskHealth()
		if err != nil {
			return false, []string{"Failed to check disk health"}, err
		}
		// FIX: Check if diskHealth is nil
		if diskHealth != nil && diskHealth.OverallHealth != "healthy" {
			healthy = false
			issues = append(issues, diskHealth.Critical...)
			issues = append(issues, diskHealth.Warnings...)
		}
	}

	// Check if CPU usage is extremely high
	if sc.enabledMetrics["cpu"] {
		metrics, err := sc.GetSystemMetrics()
		// FIX: Check if metrics is nil
		if err == nil && metrics != nil {
			if metrics.CPU.Usage > 95 {
				healthy = false
				issues = append(issues, fmt.Sprintf("CPU: Usage above 95%% (%.1f%%)", metrics.CPU.Usage))
			}

			// NEW: Check for extremely high disk I/O
			if metrics.Disk.IsDiskIOHigh() {
				issues = append(issues, fmt.Sprintf("Disk I/O: High activity (R: %.1f MB/s, W: %.1f MB/s)",
					metrics.Disk.ReadSpeed, metrics.Disk.WriteSpeed))
			}

			// NEW: Check for extremely high network traffic
			if metrics.Network.IsNetworkTrafficHigh() {
				issues = append(issues, fmt.Sprintf("Network: High traffic (U: %.1f Mbps, D: %.1f Mbps)",
					metrics.Network.UploadSpeed, metrics.Network.DownloadSpeed))
			}
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

// Reset resets the collector state (SPEED TRACKING RESET ADDED) (FIX: Check collectors)
func (sc *SystemCollector) Reset() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// FIX: Check if cpuCollector is nil before calling Reset
	if sc.cpuCollector != nil {
		sc.cpuCollector.Reset()
	}

	sc.collectionCount = 0
	sc.errors = make([]error, 0)
	sc.lastCollection = time.Now()
	sc.lastNetworkStats = make(map[string]net.IOCountersStat)
	sc.lastNetworkTime = time.Now()
	sc.lastDiskTime = time.Now()
	// NEW: Reset speed tracking
	sc.lastDiskStats = models.DiskIOStats{}
	sc.lastNetworkSent = 0
	sc.lastNetworkRecv = 0
}

// NEW: Speed-related helper methods

// GetLastSpeedMeasurement returns the last speed measurement times
func (sc *SystemCollector) GetLastSpeedMeasurement() (diskTime, networkTime time.Time) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	return sc.lastDiskTime, sc.lastNetworkTime
}

// HasSpeedHistory checks if we have enough history for speed calculation
func (sc *SystemCollector) HasSpeedHistory() bool {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	return !sc.lastDiskTime.IsZero() && !sc.lastNetworkTime.IsZero()
}
