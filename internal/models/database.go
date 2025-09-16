package models

import (
	"time"
)

// BaseModel contains common fields for all models
type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Metric represents a single metrics entry in the database (SPEED FIELDS ADDED)
type Metric struct {
	BaseModel

	// Basic fields
	Hostname  string    `json:"hostname" gorm:"not null;index"`
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`

	// CPU metrics - DETAILED
	CPUUsage     float64 `json:"cpu_usage"`
	CPUCores     int     `json:"cpu_cores"`
	CPUFrequency float64 `json:"cpu_frequency_mhz"`
	CPULoadAvg1  float64 `json:"cpu_load_avg_1"`
	CPULoadAvg5  float64 `json:"cpu_load_avg_5"`
	CPULoadAvg15 float64 `json:"cpu_load_avg_15"`

	// Memory metrics - DETAILED
	MemoryPercent     float64 `json:"memory_percent"`
	MemoryTotal       uint64  `json:"memory_total_bytes"`
	MemoryUsed        uint64  `json:"memory_used_bytes"`
	MemoryAvailable   uint64  `json:"memory_available_bytes"`
	MemoryFree        uint64  `json:"memory_free_bytes"`
	MemoryCached      uint64  `json:"memory_cached_bytes"`
	MemoryBuffers     uint64  `json:"memory_buffers_bytes"`
	MemorySwapTotal   uint64  `json:"memory_swap_total_bytes"`
	MemorySwapUsed    uint64  `json:"memory_swap_used_bytes"`
	MemorySwapPercent float64 `json:"memory_swap_percent"`

	// Disk metrics - DETAILED (SPEED FIELDS ADDED)
	DiskPercent    float64 `json:"disk_percent"`
	DiskTotal      uint64  `json:"disk_total_bytes"`
	DiskUsed       uint64  `json:"disk_used_bytes"`
	DiskFree       uint64  `json:"disk_free_bytes"`
	DiskReadBytes  uint64  `json:"disk_read_bytes"`
	DiskWriteBytes uint64  `json:"disk_write_bytes"`
	DiskReadOps    uint64  `json:"disk_read_ops"`
	DiskWriteOps   uint64  `json:"disk_write_ops"`
	// Disk I/O Speed fields
	DiskReadSpeed  float64 `json:"disk_read_speed_mbps"`  // MB/s
	DiskWriteSpeed float64 `json:"disk_write_speed_mbps"` // MB/s

	// Network metrics - DETAILED (SPEED FIELDS ADDED)
	NetworkTotalSent     uint64 `json:"network_total_sent_bytes"`
	NetworkTotalReceived uint64 `json:"network_total_recv_bytes"`
	NetworkPacketsSent   uint64 `json:"network_packets_sent"`
	NetworkPacketsRecv   uint64 `json:"network_packets_recv"`
	NetworkErrors        uint64 `json:"network_errors"`
	NetworkDrops         uint64 `json:"network_drops"`
	// Network Speed fields
	NetworkUploadSpeed   float64 `json:"network_upload_speed_mbps"`   // Mbps
	NetworkDownloadSpeed float64 `json:"network_download_speed_mbps"` // Mbps

	// System info
	Platform        string        `json:"platform"`
	PlatformVersion string        `json:"platform_version"`
	KernelArch      string        `json:"kernel_arch"`
	Uptime          time.Duration `json:"uptime_seconds"`
	ProcessCount    uint64        `json:"process_count"`
}

// TableName specifies the table name for Metric model
func (Metric) TableName() string {
	return "metrics"
}

// ConvertSystemMetricsToDBMetric converts SystemMetrics to database Metric (SPEED SUPPORT ADDED)
func ConvertSystemMetricsToDBMetric(sm *SystemMetrics) *Metric {
	metric := &Metric{
		Hostname:  sm.Hostname,
		Timestamp: sm.Timestamp,

		// CPU metrics
		CPUUsage:     sm.CPU.Usage,
		CPUCores:     sm.CPU.Cores,
		CPUFrequency: sm.CPU.Frequency,

		// Memory metrics
		MemoryPercent:     sm.Memory.Percent,
		MemoryTotal:       sm.Memory.Total,
		MemoryUsed:        sm.Memory.Used,
		MemoryAvailable:   sm.Memory.Available,
		MemoryFree:        sm.Memory.Free,
		MemoryCached:      sm.Memory.Cached,
		MemoryBuffers:     sm.Memory.Buffers,
		MemorySwapTotal:   sm.Memory.SwapTotal,
		MemorySwapUsed:    sm.Memory.SwapUsed,
		MemorySwapPercent: sm.Memory.SwapPercent,

		// Disk metrics (SPEED FIELDS ADDED)
		DiskPercent:    sm.Disk.Percent,
		DiskTotal:      sm.Disk.Total,
		DiskUsed:       sm.Disk.Used,
		DiskFree:       sm.Disk.Free,
		DiskReadBytes:  sm.Disk.IOStats.ReadBytes,
		DiskWriteBytes: sm.Disk.IOStats.WriteBytes,
		DiskReadOps:    sm.Disk.IOStats.ReadOps,
		DiskWriteOps:   sm.Disk.IOStats.WriteOps,
		// Disk speeds
		DiskReadSpeed:  sm.Disk.ReadSpeed,
		DiskWriteSpeed: sm.Disk.WriteSpeed,

		// Network metrics (SPEED FIELDS ADDED)
		NetworkTotalSent:     sm.Network.TotalSent,
		NetworkTotalReceived: sm.Network.TotalReceived,
		// Network speeds
		NetworkUploadSpeed:   sm.Network.UploadSpeed,
		NetworkDownloadSpeed: sm.Network.DownloadSpeed,

		// System info
		Platform: "Unknown", // Will be filled by system info
		Uptime:   sm.Uptime,
	}

	// CPU Load averages
	if len(sm.CPU.LoadAvg) >= 3 {
		metric.CPULoadAvg1 = sm.CPU.LoadAvg[0]
		metric.CPULoadAvg5 = sm.CPU.LoadAvg[1]
		metric.CPULoadAvg15 = sm.CPU.LoadAvg[2]
	}

	// Network aggregation
	var totalPacketsSent, totalPacketsRecv, totalErrors, totalDrops uint64
	for _, iface := range sm.Network.Interfaces {
		totalPacketsSent += iface.PacketsSent
		totalPacketsRecv += iface.PacketsRecv
		totalErrors += iface.Errors
		totalDrops += iface.Drops
	}
	metric.NetworkPacketsSent = totalPacketsSent
	metric.NetworkPacketsRecv = totalPacketsRecv
	metric.NetworkErrors = totalErrors
	metric.NetworkDrops = totalDrops

	return metric
}

// DBSystemInfo represents system information in the database
type DBSystemInfo struct {
	BaseModel

	Hostname        string    `json:"hostname" gorm:"not null;uniqueIndex"`
	Platform        string    `json:"platform"`
	PlatformFamily  string    `json:"platform_family"`
	PlatformVersion string    `json:"platform_version"`
	KernelVersion   string    `json:"kernel_version"`
	KernelArch      string    `json:"kernel_arch"`
	HostID          string    `json:"host_id"`
	BootTime        time.Time `json:"boot_time"`
	ProcessCount    uint64    `json:"process_count"`
	LastSeen        time.Time `json:"last_seen" gorm:"index"`
}

// TableName specifies the table name for DBSystemInfo model
func (DBSystemInfo) TableName() string {
	return "system_info"
}

// Alert represents alert configurations in the database
type Alert struct {
	BaseModel

	Name        string  `json:"name" gorm:"not null;uniqueIndex"`
	MetricType  string  `json:"metric_type" gorm:"index"`
	Condition   string  `json:"condition"`
	Threshold   float64 `json:"threshold"`
	Duration    int     `json:"duration"`
	Severity    string  `json:"severity" gorm:"index"`
	IsActive    bool    `json:"is_active" gorm:"default:true;index"`
	Description string  `json:"description"`

	// Notification settings
	EmailEnabled    bool   `json:"email_enabled" gorm:"default:false"`
	EmailRecipients string `json:"email_recipients"`
	WebhookEnabled  bool   `json:"webhook_enabled" gorm:"default:false"`
	WebhookURL      string `json:"webhook_url"`

	// Alert statistics
	TriggeredCount int       `json:"triggered_count" gorm:"default:0"`
	LastTriggered  time.Time `json:"last_triggered"`
}

// TableName specifies the table name for Alert model
func (Alert) TableName() string {
	return "alerts"
}

// AlertHistory represents triggered alerts history
type AlertHistory struct {
	BaseModel

	AlertID     uint      `json:"alert_id" gorm:"index"`
	Alert       Alert     `json:"alert" gorm:"foreignKey:AlertID"`
	Hostname    string    `json:"hostname" gorm:"index"`
	MetricValue float64   `json:"metric_value"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity" gorm:"index"`
	Message     string    `json:"message"`
	Resolved    bool      `json:"resolved" gorm:"default:false;index"`
	ResolvedAt  time.Time `json:"resolved_at"`

	// Notification status
	EmailSent   bool `json:"email_sent" gorm:"default:false"`
	WebhookSent bool `json:"webhook_sent" gorm:"default:false"`
}

// TableName specifies the table name for AlertHistory model
func (AlertHistory) TableName() string {
	return "alert_history"
}

// Additional models for repository responses

// AverageMetrics represents average resource usage over time (SPEED SUPPORT ADDED)
type AverageMetrics struct {
	Hostname       string        `json:"hostname,omitempty"`
	Duration       time.Duration `json:"duration"`
	From           time.Time     `json:"from"`
	To             time.Time     `json:"to"`
	AvgCPUUsage    float64       `json:"avg_cpu_usage"`
	AvgMemoryUsage float64       `json:"avg_memory_usage"`
	AvgDiskUsage   float64       `json:"avg_disk_usage"`
	MaxCPUUsage    float64       `json:"max_cpu_usage"`
	MaxMemoryUsage float64       `json:"max_memory_usage"`
	MaxDiskUsage   float64       `json:"max_disk_usage"`
	MinCPUUsage    float64       `json:"min_cpu_usage"`
	MinMemoryUsage float64       `json:"min_memory_usage"`
	MinDiskUsage   float64       `json:"min_disk_usage"`
	SampleCount    int64         `json:"sample_count"`
	// Speed averages
	AvgDiskReadSpeed   float64 `json:"avg_disk_read_speed"`
	AvgDiskWriteSpeed  float64 `json:"avg_disk_write_speed"`
	AvgNetworkUpload   float64 `json:"avg_network_upload"`
	AvgNetworkDownload float64 `json:"avg_network_download"`
	MaxDiskReadSpeed   float64 `json:"max_disk_read_speed"`
	MaxDiskWriteSpeed  float64 `json:"max_disk_write_speed"`
	MaxNetworkUpload   float64 `json:"max_network_upload"`
	MaxNetworkDownload float64 `json:"max_network_download"`
}

// MetricsSummary provides summary statistics for a time period (SPEED SUPPORT ADDED)
type MetricsSummary struct {
	From           time.Time     `json:"from"`
	To             time.Time     `json:"to"`
	Duration       time.Duration `json:"duration"`
	TotalRecords   int64         `json:"total_records"`
	UniqueHosts    int64         `json:"unique_hosts"`
	AvgCPUUsage    float64       `json:"avg_cpu_usage"`
	AvgMemoryUsage float64       `json:"avg_memory_usage"`
	AvgDiskUsage   float64       `json:"avg_disk_usage"`
	// Speed summaries
	AvgDiskReadSpeed   float64 `json:"avg_disk_read_speed"`
	AvgDiskWriteSpeed  float64 `json:"avg_disk_write_speed"`
	AvgNetworkUpload   float64 `json:"avg_network_upload"`
	AvgNetworkDownload float64 `json:"avg_network_download"`
}

// HostUsage represents resource usage for a specific host
type HostUsage struct {
	Hostname       string    `json:"hostname"`
	AvgCPUUsage    float64   `json:"avg_cpu_usage"`
	AvgMemoryUsage float64   `json:"avg_memory_usage"`
	AvgDiskUsage   float64   `json:"avg_disk_usage"`
	LastSeen       time.Time `json:"last_seen"`
}

// UsageTrend represents usage trends over time
type UsageTrend struct {
	Hour           time.Time `json:"hour"`
	AvgCPUUsage    float64   `json:"avg_cpu_usage"`
	AvgMemoryUsage float64   `json:"avg_memory_usage"`
	AvgDiskUsage   float64   `json:"avg_disk_usage"`
	SampleCount    int64     `json:"sample_count"`
}

// SystemStatus represents current system status
type SystemStatus struct {
	Hostname      string    `json:"hostname"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskPercent   float64   `json:"disk_percent"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"` // online, warning, offline
}

// GetAllModels returns all models for auto-migration
// Note: Removed unused GetAllModels helper.
