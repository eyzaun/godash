package utils

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"time"
)

// FormatBytes converts bytes to human readable format
func FormatBytes(bytes uint64) string {
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
	if exp >= len(units) {
		exp = len(units) - 1
	}
	
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatBytesDecimal converts bytes to human readable format using decimal (1000) instead of binary (1024)
func FormatBytesDecimal(bytes uint64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"kB", "MB", "GB", "TB", "PB", "EB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// ParseBytes parses human readable bytes string to uint64
func ParseBytes(s string) (uint64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	
	// Handle empty string
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	
	// Define units in order of priority (longest first to avoid conflicts)
	unitPairs := []struct {
		suffix     string
		multiplier uint64
	}{
		{"PB", 1024 * 1024 * 1024 * 1024 * 1024},
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}
	
	// Try to match units
	for _, unit := range unitPairs {
		if strings.HasSuffix(s, unit.suffix) {
			numStr := strings.TrimSuffix(s, unit.suffix)
			numStr = strings.TrimSpace(numStr)
			
			if numStr == "" {
				return 0, fmt.Errorf("no number specified before unit %s", unit.suffix)
			}
			
			var num float64
			if _, err := fmt.Sscanf(numStr, "%f", &num); err != nil {
				return 0, fmt.Errorf("invalid number format: %s", numStr)
			}
			
			if num < 0 {
				return 0, fmt.Errorf("negative numbers not allowed: %f", num)
			}
			
			result := uint64(num * float64(unit.multiplier))
			return result, nil
		}
	}
	
	// If no unit specified, assume bytes
	var num uint64
	if _, err := fmt.Sscanf(s, "%d", &num); err != nil {
		return 0, fmt.Errorf("invalid bytes format: %s", s)
	}
	
	return num, nil
}

// FormatDuration formats duration in human readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}

// FormatPercent formats percentage with specified decimal places
func FormatPercent(percent float64, decimals int) string {
	format := fmt.Sprintf("%%.%df%%%%", decimals)
	return fmt.Sprintf(format, percent)
}

// RoundFloat rounds float64 to specified decimal places
func RoundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// ClampFloat clamps float64 value between min and max
func ClampFloat(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// ClampPercent clamps percentage value between 0 and 100
func ClampPercent(percent float64) float64 {
	return ClampFloat(percent, 0, 100)
}

// GetHostname returns the system hostname
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// GetPlatform returns the current platform information
func GetPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsMacOS returns true if running on macOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// SafeDivide performs safe division avoiding division by zero
func SafeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

// SafeDivideUint64 performs safe division for uint64 values
func SafeDivideUint64(numerator, denominator uint64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

// CalculatePercentage calculates percentage of used/total
func CalculatePercentage(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// CreateDirIfNotExists creates directory if it doesn't exist
func CreateDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// TruncateString truncates string to specified length with ellipsis
func TruncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

// PadString pads string to specified length
func PadString(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// Contains checks if slice contains string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveFromSlice removes string from slice
func RemoveFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// MaxInt returns the maximum of two integers
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the minimum of two integers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxFloat64 returns the maximum of two float64 values
func MaxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// MinFloat64 returns the minimum of two float64 values
func MinFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GetMemoryPressureLevel returns memory pressure level based on usage percentage
func GetMemoryPressureLevel(percent float64) string {
	switch {
	case percent < 50:
		return "low"
	case percent < 75:
		return "moderate"
	case percent < 90:
		return "high"
	default:
		return "critical"
	}
}

// GetDiskHealthLevel returns disk health level based on usage percentage
func GetDiskHealthLevel(percent float64) string {
	switch {
	case percent < 80:
		return "healthy"
	case percent < 90:
		return "warning"
	default:
		return "critical"
	}
}

// GetCPULoadLevel returns CPU load level based on usage percentage
func GetCPULoadLevel(percent float64) string {
	switch {
	case percent < 50:
		return "low"
	case percent < 80:
		return "moderate"
	case percent < 95:
		return "high"
	default:
		return "critical"
	}
}

// FormatUptime formats uptime duration in human readable format
func FormatUptime(uptime time.Duration) string {
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

// GetCurrentTimestamp returns current timestamp in RFC3339 format
func GetCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// ParseTimestamp parses RFC3339 timestamp string
func ParseTimestamp(timestamp string) (time.Time, error) {
	return time.Parse(time.RFC3339, timestamp)
}

// ValidatePercentage validates if value is a valid percentage (0-100)
func ValidatePercentage(percent float64) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("percentage must be between 0 and 100, got %.2f", percent)
	}
	return nil
}

// SanitizeHostname sanitizes hostname for safe usage
func SanitizeHostname(hostname string) string {
	// Replace invalid characters with underscore
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			return r
		}
		return '_'
	}, hostname)
	
	// Remove leading/trailing dots and dashes
	sanitized = strings.Trim(sanitized, ".-")
	
	if sanitized == "" {
		return "unknown"
	}
	
	return sanitized
}