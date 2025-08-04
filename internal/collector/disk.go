package collector

import (
	"fmt"
	"strings"
	"time"

	"github.com/eyzaun/godash/internal/models"
	"github.com/shirou/gopsutil/v3/disk"
)

// DiskCollector handles disk metrics collection
type DiskCollector struct {
	excludeFilesystems []string
	lastIOStats        map[string]disk.IOCountersStat
	lastIOTime         time.Time
}

// NewDiskCollector creates a new disk collector
func NewDiskCollector() *DiskCollector {
	// Default filesystems to exclude - more comprehensive list
	excludeFS := []string{
		"tmpfs", "devtmpfs", "sysfs", "proc", "devfs", "fdescfs",
		"overlay", "squashfs", "iso9660", "udf", "fuse", "cgroup",
		"cgroup2", "configfs", "debugfs", "mqueue", "pstore", "tracefs",
		"binfmt_misc", "autofs", "rpc_pipefs", "nfsd", "bpf", "ramfs",
		"hugetlbfs", "securityfs", "efivarfs", "fusectl", "selinuxfs",
	}

	return &DiskCollector{
		excludeFilesystems: excludeFS,
		lastIOStats:        make(map[string]disk.IOCountersStat),
		lastIOTime:         time.Now(),
	}
}

// GetDiskMetrics collects disk usage metrics with separate partition data
func (d *DiskCollector) GetDiskMetrics() (*models.DiskMetrics, error) {
	// Get all partitions
	partitions, err := disk.Partitions(false) // Only get mounted partitions
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	var totalBytes, usedBytes, freeBytes uint64
	var partitionInfos []models.PartitionInfo

	// Process each partition
	for _, partition := range partitions {
		// Skip excluded filesystems
		if d.shouldExcludeFilesystem(partition.Fstype) {
			continue
		}

		// Skip network drives and other virtual mounts on Windows
		if d.shouldExcludeMountpoint(partition.Mountpoint) {
			continue
		}

		// Get usage for this partition
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			// Log error but continue with other partitions
			fmt.Printf("Warning: Cannot access partition %s: %v\n", partition.Mountpoint, err)
			continue
		}

		// Skip if total is 0 (empty or virtual filesystems)
		if usage.Total == 0 {
			continue
		}

		// Add to totals
		totalBytes += usage.Total
		usedBytes += usage.Used
		freeBytes += usage.Free

		// Calculate partition usage percentage
		partitionPercent := float64(usage.Used) / float64(usage.Total) * 100

		// Add partition info with more detailed information
		partitionInfos = append(partitionInfos, models.PartitionInfo{
			Device:     partition.Device,
			Mountpoint: partition.Mountpoint,
			Fstype:     partition.Fstype,
			Total:      usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			Percent:    partitionPercent,
		})

		fmt.Printf("Found partition: %s (%s) - %s - Total: %.2f GB, Used: %.2f GB (%.1f%%)\n",
			partition.Device, partition.Mountpoint, partition.Fstype,
			float64(usage.Total)/(1024*1024*1024),
			float64(usage.Used)/(1024*1024*1024),
			partitionPercent)
	}

	// Calculate overall usage percentage
	var usagePercent float64
	if totalBytes > 0 {
		usagePercent = float64(usedBytes) / float64(totalBytes) * 100
	}

	// Get disk I/O statistics
	ioStats, err := d.getDiskIOStats()
	if err != nil {
		// If we can't get I/O stats, use empty stats
		ioStats = models.DiskIOStats{}
	}

	return &models.DiskMetrics{
		Total:      totalBytes,
		Used:       usedBytes,
		Free:       freeBytes,
		Percent:    usagePercent,
		Partitions: partitionInfos,
		IOStats:    ioStats,
	}, nil
}

// getDiskIOStats collects disk I/O statistics
func (d *DiskCollector) getDiskIOStats() (models.DiskIOStats, error) {
	// Get current I/O counters
	ioCounters, err := disk.IOCounters()
	if err != nil {
		return models.DiskIOStats{}, fmt.Errorf("failed to get disk I/O counters: %w", err)
	}

	var totalStats models.DiskIOStats
	currentTime := time.Now()

	// Aggregate I/O stats from all devices
	for deviceName, counter := range ioCounters {
		// Skip loop devices and other virtual devices on Linux
		if d.shouldExcludeDevice(deviceName) {
			continue
		}

		totalStats.ReadBytes += counter.ReadBytes
		totalStats.WriteBytes += counter.WriteBytes
		totalStats.ReadOps += counter.ReadCount
		totalStats.WriteOps += counter.WriteCount
		totalStats.ReadTime += counter.ReadTime
		totalStats.WriteTime += counter.WriteTime
	}

	// Update last I/O stats for future calculations
	d.lastIOStats = ioCounters
	d.lastIOTime = currentTime

	return totalStats, nil
}

// shouldExcludeFilesystem checks if a filesystem should be excluded
func (d *DiskCollector) shouldExcludeFilesystem(fstype string) bool {
	fstype = strings.ToLower(fstype)
	for _, excludeFS := range d.excludeFilesystems {
		if fstype == excludeFS {
			return true
		}
	}
	return false
}

// shouldExcludeMountpoint checks if a mountpoint should be excluded
func (d *DiskCollector) shouldExcludeMountpoint(mountpoint string) bool {
	// Windows specific exclusions
	excludeMounts := []string{
		"\\\\",         // Network drives
		"A:\\", "B:\\", // Floppy drives
	}

	for _, excludeMount := range excludeMounts {
		if strings.HasPrefix(mountpoint, excludeMount) {
			return true
		}
	}

	// Linux/Unix specific exclusions
	linuxExcludes := []string{
		"/dev", "/proc", "/sys", "/run", "/tmp",
		"/snap", "/boot/efi",
	}

	for _, excludeMount := range linuxExcludes {
		if mountpoint == excludeMount || strings.HasPrefix(mountpoint, excludeMount+"/") {
			return true
		}
	}

	return false
}

// shouldExcludeDevice checks if a device should be excluded from I/O stats
func (d *DiskCollector) shouldExcludeDevice(deviceName string) bool {
	deviceName = strings.ToLower(deviceName)

	// Exclude loop devices (Linux)
	if strings.HasPrefix(deviceName, "loop") {
		return true
	}

	// Exclude RAM disks
	if strings.HasPrefix(deviceName, "ram") {
		return true
	}

	// Exclude other virtual devices
	virtualDevices := []string{"dm-", "md", "sr", "fd"}
	for _, vdev := range virtualDevices {
		if strings.HasPrefix(deviceName, vdev) {
			return true
		}
	}

	return false
}

// GetDiskUsageByPath gets disk usage for a specific path
func (d *DiskCollector) GetDiskUsageByPath(path string) (*disk.UsageStat, error) {
	return disk.Usage(path)
}

// GetAllPartitions gets all disk partitions
func (d *DiskCollector) GetAllPartitions() ([]disk.PartitionStat, error) {
	return disk.Partitions(true)
}

// GetDiskIOCounters gets I/O counters for all disks
func (d *DiskCollector) GetDiskIOCounters() (map[string]disk.IOCountersStat, error) {
	return disk.IOCounters()
}

// FormatBytes formats bytes into human readable format
func (d *DiskCollector) FormatBytes(bytes uint64) string {
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

// GetDiskHealth checks disk health status
func (d *DiskCollector) GetDiskHealth() (*DiskHealthInfo, error) {
	metrics, err := d.GetDiskMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk metrics: %w", err)
	}

	health := &DiskHealthInfo{
		OverallHealth: "healthy",
		Warnings:      []string{},
		Critical:      []string{},
	}

	// Check overall disk usage
	if metrics.Percent > 95 {
		health.Critical = append(health.Critical, "Overall disk usage above 95%")
		health.OverallHealth = "critical"
	} else if metrics.Percent > 85 {
		health.Warnings = append(health.Warnings, "Overall disk usage above 85%")
		if health.OverallHealth == "healthy" {
			health.OverallHealth = "warning"
		}
	}

	// Check individual partitions
	for _, partition := range metrics.Partitions {
		if partition.Percent > 95 {
			health.Critical = append(health.Critical,
				fmt.Sprintf("Partition %s usage above 95%% (%.1f%%)",
					partition.Mountpoint, partition.Percent))
			health.OverallHealth = "critical"
		} else if partition.Percent > 85 {
			health.Warnings = append(health.Warnings,
				fmt.Sprintf("Partition %s usage above 85%% (%.1f%%)",
					partition.Mountpoint, partition.Percent))
			if health.OverallHealth == "healthy" {
				health.OverallHealth = "warning"
			}
		}

		// Check for very low free space
		if partition.Free < 1024*1024*1024 { // Less than 1GB free
			health.Warnings = append(health.Warnings,
				fmt.Sprintf("Partition %s has less than 1GB free space",
					partition.Mountpoint))
			if health.OverallHealth == "healthy" {
				health.OverallHealth = "warning"
			}
		}
	}

	return health, nil
}

// DiskHealthInfo contains disk health information
type DiskHealthInfo struct {
	OverallHealth string   `json:"overall_health"`
	Warnings      []string `json:"warnings"`
	Critical      []string `json:"critical"`
}

// GetLargestPartition finds the partition with the most used space
func (d *DiskCollector) GetLargestPartition() (*models.PartitionInfo, error) {
	metrics, err := d.GetDiskMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk metrics: %w", err)
	}

	if len(metrics.Partitions) == 0 {
		return nil, fmt.Errorf("no partitions found")
	}

	var largest *models.PartitionInfo
	for i, partition := range metrics.Partitions {
		if largest == nil || partition.Used > largest.Used {
			largest = &metrics.Partitions[i]
		}
	}

	return largest, nil
}

// GetDiskTrend analyzes disk usage trend
func (d *DiskCollector) GetDiskTrend(samples []models.DiskMetrics) DiskTrend {
	if len(samples) < 2 {
		return DiskTrendStable
	}

	// Calculate average change over the samples
	var totalChange float64
	for i := 1; i < len(samples); i++ {
		change := samples[i].Percent - samples[i-1].Percent
		totalChange += change
	}

	avgChange := totalChange / float64(len(samples)-1)

	switch {
	case avgChange > 1:
		return DiskTrendIncreasing
	case avgChange < -1:
		return DiskTrendDecreasing
	default:
		return DiskTrendStable
	}
}

// DiskTrend represents disk usage trend
type DiskTrend int

const (
	DiskTrendStable DiskTrend = iota
	DiskTrendIncreasing
	DiskTrendDecreasing
)

// String returns string representation of disk trend
func (dt DiskTrend) String() string {
	switch dt {
	case DiskTrendIncreasing:
		return "increasing"
	case DiskTrendDecreasing:
		return "decreasing"
	default:
		return "stable"
	}
}

// SetExcludedFilesystems sets the list of filesystems to exclude
func (d *DiskCollector) SetExcludedFilesystems(filesystems []string) {
	d.excludeFilesystems = filesystems
}

// GetExcludedFilesystems returns the list of excluded filesystems
func (d *DiskCollector) GetExcludedFilesystems() []string {
	return d.excludeFilesystems
}

// IsDiskSpaceLow checks if any partition has low disk space
func (d *DiskCollector) IsDiskSpaceLow(threshold float64) (bool, []string, error) {
	metrics, err := d.GetDiskMetrics()
	if err != nil {
		return false, nil, fmt.Errorf("failed to get disk metrics: %w", err)
	}

	var lowSpacePartitions []string

	for _, partition := range metrics.Partitions {
		if partition.Percent > threshold {
			lowSpacePartitions = append(lowSpacePartitions,
				fmt.Sprintf("%s (%.1f%%)", partition.Mountpoint, partition.Percent))
		}
	}

	return len(lowSpacePartitions) > 0, lowSpacePartitions, nil
}
