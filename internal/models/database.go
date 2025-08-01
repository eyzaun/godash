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

// Metric represents a single metrics entry in the database
type Metric struct {
	BaseModel

	// Basic fields
	Hostname  string    `json:"hostname" gorm:"not null;index"`
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`

	// Simple metrics - starting with basic ones
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
}

// TableName specifies the table name for Metric model
func (Metric) TableName() string {
	return "metrics"
}

// DBSystemInfo represents system information in the database (renamed to avoid conflict)
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

// ConvertSystemMetricsToDBMetric converts SystemMetrics to database Metric (simplified)
func ConvertSystemMetricsToDBMetric(sm *SystemMetrics) *Metric {
	return &Metric{
		Hostname:  sm.Hostname,
		Timestamp: sm.Timestamp,

		// Basic metrics only
		CPUUsage:      sm.CPU.Usage,
		MemoryPercent: sm.Memory.Percent,
		DiskPercent:   sm.Disk.Percent,
	}
}

// ConvertDBMetricToSystemMetrics converts database Metric to SystemMetrics (simplified)
func ConvertDBMetricToSystemMetrics(m *Metric) *SystemMetrics {
	return &SystemMetrics{
		Hostname:  m.Hostname,
		Timestamp: m.Timestamp,

		CPU: CPUMetrics{
			Usage: m.CPUUsage,
		},

		Memory: MemoryMetrics{
			Percent: m.MemoryPercent,
		},

		Disk: DiskMetrics{
			Percent: m.DiskPercent,
		},

		Network: NetworkMetrics{},
	}
}

/*
// ConvertSystemInfoToDB converts SystemInfo to database DBSystemInfo
func ConvertSystemInfoToDB(si *SystemInfo) *DBSystemInfo {
	return &DBSystemInfo{
		Hostname:        si.Hostname,
		Platform:        si.Platform,
		PlatformFamily:  si.PlatformFamily,
		PlatformVersion: si.PlatformVersion,
		KernelVersion:   si.KernelVersion,
		KernelArch:      si.KernelArch,
		HostID:          si.HostID,
		BootTime:        si.BootTime,
		ProcessCount:    si.Processes,
		LastSeen:        time.Now(),
	}
}
*/

// Additional models for repository responses

// AverageMetrics represents average resource usage over time
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
}

// MetricsSummary provides summary statistics for a time period
type MetricsSummary struct {
	From           time.Time     `json:"from"`
	To             time.Time     `json:"to"`
	Duration       time.Duration `json:"duration"`
	TotalRecords   int64         `json:"total_records"`
	UniqueHosts    int64         `json:"unique_hosts"`
	AvgCPUUsage    float64       `json:"avg_cpu_usage"`
	AvgMemoryUsage float64       `json:"avg_memory_usage"`
	AvgDiskUsage   float64       `json:"avg_disk_usage"`
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
func GetAllModels() []interface{} {
	return []interface{}{
		&Metric{},
		&DBSystemInfo{},
		&Alert{},
		&AlertHistory{},
	}
}
