package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// SystemMetrics represents the complete system metrics at a point in time
type SystemMetrics struct {
	CPU       CPUMetrics     `json:"cpu"`
	Memory    MemoryMetrics  `json:"memory"`
	Disk      DiskMetrics    `json:"disk"`
	Network   NetworkMetrics `json:"network"`
	Timestamp time.Time      `json:"timestamp"`
	Hostname  string         `json:"hostname"`
	Uptime    time.Duration  `json:"uptime"`
}

// CPUMetrics represents CPU usage information
type CPUMetrics struct {
	Usage     float64   `json:"usage_percent"` // Overall CPU usage percentage
	Cores     int       `json:"cores"`         // Number of CPU cores
	CoreUsage []float64 `json:"core_usage"`    // Per-core usage percentages
	LoadAvg   []float64 `json:"load_average"`  // 1, 5, 15 minute load averages
	Frequency float64   `json:"frequency_mhz"` // CPU frequency in MHz
}

// MemoryMetrics represents memory usage information
type MemoryMetrics struct {
	Total       uint64  `json:"total_bytes"`      // Total physical memory
	Used        uint64  `json:"used_bytes"`       // Used memory
	Available   uint64  `json:"available_bytes"`  // Available memory
	Free        uint64  `json:"free_bytes"`       // Free memory
	Cached      uint64  `json:"cached_bytes"`     // Cached memory
	Buffers     uint64  `json:"buffers_bytes"`    // Buffer memory
	Percent     float64 `json:"usage_percent"`    // Memory usage percentage
	SwapTotal   uint64  `json:"swap_total_bytes"` // Total swap space
	SwapUsed    uint64  `json:"swap_used_bytes"`  // Used swap space
	SwapPercent float64 `json:"swap_percent"`     // Swap usage percentage
}

// DiskMetrics represents disk usage information (SPEED FIELDS ADDED)
type DiskMetrics struct {
	Total      uint64          `json:"total_bytes"`   // Total disk space
	Used       uint64          `json:"used_bytes"`    // Used disk space
	Free       uint64          `json:"free_bytes"`    // Free disk space
	Percent    float64         `json:"usage_percent"` // Disk usage percentage
	Partitions []PartitionInfo `json:"partitions"`    // Individual partition info
	IOStats    DiskIOStats     `json:"io_stats"`      // Disk I/O statistics
	// NEW: Real-time speed metrics
	ReadSpeed  float64 `json:"read_speed_mbps"`  // Current read speed in MB/s
	WriteSpeed float64 `json:"write_speed_mbps"` // Current write speed in MB/s
}

// PartitionInfo represents individual partition information
type PartitionInfo struct {
	Device     string  `json:"device"`        // Device name (e.g., /dev/sda1)
	Mountpoint string  `json:"mountpoint"`    // Mount point (e.g., /)
	Fstype     string  `json:"fstype"`        // Filesystem type (e.g., ext4)
	Total      uint64  `json:"total_bytes"`   // Total space
	Used       uint64  `json:"used_bytes"`    // Used space
	Free       uint64  `json:"free_bytes"`    // Free space
	Percent    float64 `json:"usage_percent"` // Usage percentage
}

// DiskIOStats represents disk I/O statistics (ENHANCED)
type DiskIOStats struct {
	ReadBytes  uint64 `json:"read_bytes"`    // Bytes read
	WriteBytes uint64 `json:"write_bytes"`   // Bytes written
	ReadOps    uint64 `json:"read_ops"`      // Read operations
	WriteOps   uint64 `json:"write_ops"`     // Write operations
	ReadTime   uint64 `json:"read_time_ms"`  // Time spent reading (ms)
	WriteTime  uint64 `json:"write_time_ms"` // Time spent writing (ms)
	// NEW: For speed calculation tracking
	LastReadBytes  uint64    `json:"-"` // Previous read bytes (not exported)
	LastWriteBytes uint64    `json:"-"` // Previous write bytes (not exported)
	LastUpdateTime time.Time `json:"-"` // Last update time (not exported)
}

// NetworkMetrics represents network usage information (SPEED FIELDS ADDED)
type NetworkMetrics struct {
	Interfaces    []NetworkInterface `json:"interfaces"`       // Network interfaces
	TotalSent     uint64             `json:"total_sent_bytes"` // Total bytes sent
	TotalReceived uint64             `json:"total_recv_bytes"` // Total bytes received
	// NEW: Real-time speed metrics
	UploadSpeed   float64 `json:"upload_speed_mbps"`   // Current upload speed in Mbps
	DownloadSpeed float64 `json:"download_speed_mbps"` // Current download speed in Mbps
	// NEW: For speed calculation tracking
	LastTotalSent     uint64    `json:"-"` // Previous total sent (not exported)
	LastTotalReceived uint64    `json:"-"` // Previous total received (not exported)
	LastUpdateTime    time.Time `json:"-"` // Last update time (not exported)
}

// NetworkInterface represents individual network interface information
type NetworkInterface struct {
	Name        string `json:"name"`             // Interface name (e.g., eth0)
	BytesSent   uint64 `json:"bytes_sent"`       // Bytes sent
	BytesRecv   uint64 `json:"bytes_received"`   // Bytes received
	PacketsSent uint64 `json:"packets_sent"`     // Packets sent
	PacketsRecv uint64 `json:"packets_received"` // Packets received
	Errors      uint64 `json:"errors"`           // Error count
	Drops       uint64 `json:"drops"`            // Dropped packets
}

// ProcessInfo represents individual process information
type ProcessInfo struct {
	PID         int32   `json:"pid"`          // Process ID
	Name        string  `json:"name"`         // Process name
	CPUPercent  float64 `json:"cpu_percent"`  // CPU usage percentage
	MemoryBytes uint64  `json:"memory_bytes"` // Memory usage in bytes
	Status      string  `json:"status"`       // Process status
}

// SystemInfo represents basic system information
type SystemInfo struct {
	Hostname        string    `json:"hostname"`
	Platform        string    `json:"platform"`
	PlatformFamily  string    `json:"platform_family"`
	PlatformVersion string    `json:"platform_version"`
	KernelVersion   string    `json:"kernel_version"`
	KernelArch      string    `json:"kernel_arch"`
	HostID          string    `json:"host_id"`
	BootTime        time.Time `json:"boot_time"`
	Processes       uint64    `json:"processes"`
}

// MetricsSnapshot represents a complete system snapshot
type MetricsSnapshot struct {
	SystemMetrics SystemMetrics `json:"metrics"`
	SystemInfo    SystemInfo    `json:"system_info"`
	TopProcesses  []ProcessInfo `json:"top_processes"`
	Timestamp     time.Time     `json:"timestamp"`
}

// NEW: Speed calculation helper methods

// CalculateDiskSpeed calculates disk read/write speed in MB/s
func (dm *DiskMetrics) CalculateDiskSpeed(lastStats DiskIOStats, timeDiff time.Duration) {
	if timeDiff.Seconds() <= 0 {
		dm.ReadSpeed = 0
		dm.WriteSpeed = 0
		return
	}

	// Calculate bytes per second, then convert to MB/s
	readBytesPerSec := float64(dm.IOStats.ReadBytes-lastStats.ReadBytes) / timeDiff.Seconds()
	writeBytesPerSec := float64(dm.IOStats.WriteBytes-lastStats.WriteBytes) / timeDiff.Seconds()

	// Convert to MB/s (1 MB = 1024*1024 bytes)
	dm.ReadSpeed = readBytesPerSec / (1024 * 1024)
	dm.WriteSpeed = writeBytesPerSec / (1024 * 1024)

	// Ensure non-negative values
	if dm.ReadSpeed < 0 {
		dm.ReadSpeed = 0
	}
	if dm.WriteSpeed < 0 {
		dm.WriteSpeed = 0
	}
}

// CalculateNetworkSpeed calculates network upload/download speed in Mbps
func (nm *NetworkMetrics) CalculateNetworkSpeed(lastSent, lastReceived uint64, timeDiff time.Duration) {
	if timeDiff.Seconds() <= 0 {
		nm.UploadSpeed = 0
		nm.DownloadSpeed = 0
		return
	}

	// Calculate bytes per second
	uploadBytesPerSec := float64(nm.TotalSent-lastSent) / timeDiff.Seconds()
	downloadBytesPerSec := float64(nm.TotalReceived-lastReceived) / timeDiff.Seconds()

	// Convert to Mbps (1 Mbps = 125,000 bytes/sec)
	nm.UploadSpeed = (uploadBytesPerSec * 8) / (1024 * 1024)     // bits per second to Mbps
	nm.DownloadSpeed = (downloadBytesPerSec * 8) / (1024 * 1024) // bits per second to Mbps

	// Ensure non-negative values
	if nm.UploadSpeed < 0 {
		nm.UploadSpeed = 0
	}
	if nm.DownloadSpeed < 0 {
		nm.DownloadSpeed = 0
	}
}

// String returns a formatted string representation of SystemMetrics
func (sm *SystemMetrics) String() string {
	data, _ := json.MarshalIndent(sm, "", "  ")
	return string(data)
}

// ToJSON converts SystemMetrics to JSON string
func (sm *SystemMetrics) ToJSON() (string, error) {
	data, err := json.Marshal(sm)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON creates SystemMetrics from JSON string
func (sm *SystemMetrics) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), sm)
}

// GetMemoryUsagePercent calculates memory usage percentage
func (mm *MemoryMetrics) GetMemoryUsagePercent() float64 {
	if mm.Total == 0 {
		return 0
	}
	return float64(mm.Used) / float64(mm.Total) * 100
}

// GetDiskUsagePercent calculates disk usage percentage
func (dm *DiskMetrics) GetDiskUsagePercent() float64 {
	if dm.Total == 0 {
		return 0
	}
	return float64(dm.Used) / float64(dm.Total) * 100
}

// IsHighUsage checks if any metric is above the threshold
func (sm *SystemMetrics) IsHighUsage(cpuThreshold, memThreshold, diskThreshold float64) bool {
	return sm.CPU.Usage > cpuThreshold ||
		sm.Memory.Percent > memThreshold ||
		sm.Disk.Percent > diskThreshold
}

// GetAverageLoadUsage returns the average CPU load usage
func (cm *CPUMetrics) GetAverageLoadUsage() float64 {
	if len(cm.LoadAvg) == 0 {
		return 0
	}
	return cm.LoadAvg[0] // 1-minute load average
}

// NEW: Speed helper methods

// GetFormattedDiskSpeed returns formatted disk speed strings
func (dm *DiskMetrics) GetFormattedDiskSpeed() (string, string) {
	readSpeed := fmt.Sprintf("%.1f MB/s", dm.ReadSpeed)
	writeSpeed := fmt.Sprintf("%.1f MB/s", dm.WriteSpeed)
	return readSpeed, writeSpeed
}

// GetFormattedNetworkSpeed returns formatted network speed strings
func (nm *NetworkMetrics) GetFormattedNetworkSpeed() (string, string) {
	uploadSpeed := fmt.Sprintf("%.1f Mbps", nm.UploadSpeed)
	downloadSpeed := fmt.Sprintf("%.1f Mbps", nm.DownloadSpeed)
	return uploadSpeed, downloadSpeed
}

// IsDiskIOHigh checks if disk I/O is unusually high
func (dm *DiskMetrics) IsDiskIOHigh() bool {
	// Consider high if combined speed > 100 MB/s
	return (dm.ReadSpeed + dm.WriteSpeed) > 100.0
}

// IsNetworkTrafficHigh checks if network traffic is unusually high
func (nm *NetworkMetrics) IsNetworkTrafficHigh() bool {
	// Consider high if combined speed > 100 Mbps
	return (nm.UploadSpeed + nm.DownloadSpeed) > 100.0
}
