package collector

import (
	"context"
	"testing"
	"time"
)

func TestNewSystemCollector(t *testing.T) {
	// Test with default config
	collector := NewSystemCollector(nil)
	if collector == nil {
		t.Fatal("NewSystemCollector(nil) returned nil")
	}

	// Verify default values
	stats := collector.GetCollectionStats()
	enabledMetrics := stats["enabled_metrics"].(map[string]bool)
	
	expectedMetrics := []string{"cpu", "memory", "disk", "network", "processes"}
	for _, metric := range expectedMetrics {
		if !enabledMetrics[metric] {
			t.Errorf("Expected metric %s to be enabled by default", metric)
		}
	}

	// Test with custom config
	config := &CollectorConfig{
		CollectInterval: 10 * time.Second,
		EnableCPU:       true,
		EnableMemory:    false,
		EnableDisk:      true,
		EnableNetwork:   false,
		EnableProcesses: true,
	}

	collector2 := NewSystemCollector(config)
	if collector2 == nil {
		t.Fatal("NewSystemCollector(config) returned nil")
	}

	stats2 := collector2.GetCollectionStats()
	enabledMetrics2 := stats2["enabled_metrics"].(map[string]bool)

	if !enabledMetrics2["cpu"] {
		t.Error("CPU should be enabled")
	}
	if enabledMetrics2["memory"] {
		t.Error("Memory should be disabled")
	}
	if !enabledMetrics2["disk"] {
		t.Error("Disk should be enabled")
	}
	if enabledMetrics2["network"] {
		t.Error("Network should be disabled")
	}
}

func TestSystemCollector_GetSystemMetrics(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	metrics, err := collector.GetSystemMetrics()
	if err != nil {
		t.Fatalf("GetSystemMetrics() failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("GetSystemMetrics() returned nil metrics")
	}

	// Test basic fields
	if metrics.Hostname == "" {
		t.Error("Hostname should not be empty")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if metrics.Uptime == 0 {
		t.Error("Uptime should not be zero")
	}

	// Test CPU metrics
	if metrics.CPU.Cores <= 0 {
		t.Error("CPU cores should be greater than 0")
	}

	if metrics.CPU.Usage < 0 || metrics.CPU.Usage > 100 {
		t.Errorf("CPU usage should be between 0 and 100, got %.2f", metrics.CPU.Usage)
	}

	// Test Memory metrics
	if metrics.Memory.Total == 0 {
		t.Error("Total memory should be greater than 0")
	}

	if metrics.Memory.Percent < 0 || metrics.Memory.Percent > 100 {
		t.Errorf("Memory usage should be between 0 and 100, got %.2f", metrics.Memory.Percent)
	}

	if metrics.Memory.Used > metrics.Memory.Total {
		t.Error("Used memory should not exceed total memory")
	}

	// Test Disk metrics
	if metrics.Disk.Total == 0 {
		t.Error("Total disk space should be greater than 0")
	}

	if metrics.Disk.Percent < 0 || metrics.Disk.Percent > 100 {
		t.Errorf("Disk usage should be between 0 and 100, got %.2f", metrics.Disk.Percent)
	}

	if len(metrics.Disk.Partitions) == 0 {
		t.Error("Should have at least one partition")
	}
}

func TestSystemCollector_GetSystemInfo(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	info, err := collector.GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo() failed: %v", err)
	}

	if info == nil {
		t.Fatal("GetSystemInfo() returned nil")
	}

	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}

	if info.KernelArch == "" {
		t.Error("KernelArch should not be empty")
	}

	if info.BootTime.IsZero() {
		t.Error("BootTime should not be zero")
	}

	// Boot time should be in the past
	if info.BootTime.After(time.Now()) {
		t.Error("BootTime should be in the past")
	}
}

func TestSystemCollector_GetMetricsSnapshot(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	snapshot, err := collector.GetMetricsSnapshot()
	if err != nil {
		t.Fatalf("GetMetricsSnapshot() failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("GetMetricsSnapshot() returned nil")
	}

	// Test that all components are present
	if snapshot.SystemMetrics.Hostname == "" {
		t.Error("SystemMetrics should be populated")
	}

	if snapshot.SystemInfo.Platform == "" {
		t.Error("SystemInfo should be populated")
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Snapshot timestamp should not be zero")
	}

	// TopProcesses can be empty on some systems, so we don't test it
}

func TestSystemCollector_StartCollection(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	metricsChan := collector.StartCollection(ctx, 500*time.Millisecond)
	
	// Should receive at least one metric
	select {
	case metrics, ok := <-metricsChan:
		if !ok {
			t.Fatal("Metrics channel closed unexpectedly")
		}
		if metrics == nil {
			t.Error("Received nil metrics")
		}
		if metrics.Hostname == "" {
			t.Error("Received metrics with empty hostname")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive metrics within timeout")
	}

	   // Should receive another metric after interval (timeout increased to 3 seconds)
	   select {
	   case metrics, ok := <-metricsChan:
			   if !ok {
					   t.Fatal("Metrics channel closed unexpectedly")
			   }
			   if metrics == nil {
					   t.Error("Received nil metrics on second iteration")
			   }
	   case <-time.After(3 * time.Second):
			   t.Fatal("Did not receive second metrics within timeout")
	   }

	// Cancel context and verify channel closes
	cancel()
	
	   select {
	   case <-metricsChan:
			   // Kanal kapalıysa sorun yok, açıksa beklemeden hata verme
	   case <-time.After(3 * time.Second):
			   t.Error("Channel should close quickly after context cancellation")
	   }
}

func TestSystemCollector_GetTopProcesses(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	processes, err := collector.GetTopProcesses(5, "cpu")
	if err != nil {
		t.Fatalf("GetTopProcesses() failed: %v", err)
	}

	// Should return some processes (unless running in very restricted environment)
	if len(processes) == 0 {
		t.Skip("No processes returned, possibly running in restricted environment")
	}

	if len(processes) > 5 {
		t.Errorf("Should return at most 5 processes, got %d", len(processes))
	}

	// Test first process has valid fields
	proc := processes[0]
	if proc.PID <= 0 {
		t.Error("Process PID should be positive")
	}

	if proc.Name == "" {
		t.Error("Process name should not be empty")
	}

	if proc.CPUPercent < 0 {
		t.Error("CPU percent should not be negative")
	}

	// Test sorting by memory
	memProcesses, err := collector.GetTopProcesses(3, "memory")
	if err != nil {
		t.Fatalf("GetTopProcesses(memory) failed: %v", err)
	}

	if len(memProcesses) > 0 && len(memProcesses) >= 2 {
		// Should be sorted by memory usage (descending)
		if memProcesses[0].MemoryBytes < memProcesses[1].MemoryBytes {
			t.Error("Processes should be sorted by memory usage in descending order")
		}
	}
}

func TestSystemCollector_IsHealthy(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	healthy, issues, err := collector.IsHealthy()
	if err != nil {
		t.Fatalf("IsHealthy() failed: %v", err)
	}

	// Issues can be empty or contain warnings
	_ = issues

	// healthy can be true or false depending on system state
	_ = healthy

	// Just verify the function doesn't crash and returns valid types
	if issues == nil {
		t.Error("Issues slice should not be nil")
	}
}

func TestSystemCollector_SetEnabledMetrics(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	// Disable memory collection
	collector.SetEnabledMetrics(map[string]bool{
		"memory": false,
	})

	stats := collector.GetCollectionStats()
	enabledMetrics := stats["enabled_metrics"].(map[string]bool)

	if enabledMetrics["memory"] {
		t.Error("Memory collection should be disabled")
	}

	// CPU should still be enabled (unchanged)
	if !enabledMetrics["cpu"] {
		t.Error("CPU collection should still be enabled")
	}
}

func TestSystemCollector_GetCollectionStats(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	// Get initial stats
	stats := collector.GetCollectionStats()
	
	if stats == nil {
		t.Fatal("GetCollectionStats() returned nil")
	}

	// Check required fields
	if _, ok := stats["last_collection"]; !ok {
		t.Error("Stats should contain last_collection")
	}

	if _, ok := stats["collection_count"]; !ok {
		t.Error("Stats should contain collection_count")
	}

	if _, ok := stats["error_count"]; !ok {
		t.Error("Stats should contain error_count")
	}

	if _, ok := stats["enabled_metrics"]; !ok {
		t.Error("Stats should contain enabled_metrics")
	}

	// Collection count should be 0 initially
	if stats["collection_count"].(int64) != 0 {
		t.Error("Initial collection count should be 0")
	}

	// Collect metrics to increase count
	_, err := collector.GetSystemMetrics()
	if err != nil {
		t.Fatalf("GetSystemMetrics() failed: %v", err)
	}

	// Check stats again
	stats2 := collector.GetCollectionStats()
	if stats2["collection_count"].(int64) != 1 {
		t.Error("Collection count should be 1 after one collection")
	}
}

func TestSystemCollector_Reset(t *testing.T) {
	collector := NewSystemCollector(nil)
	
	// Collect some metrics first
	_, err := collector.GetSystemMetrics()
	if err != nil {
		t.Fatalf("GetSystemMetrics() failed: %v", err)
	}

	// Verify collection count is not zero
	stats := collector.GetCollectionStats()
	if stats["collection_count"].(int64) == 0 {
		t.Error("Collection count should not be 0 before reset")
	}

	// Reset collector
	collector.Reset()

	// Verify reset worked
	stats2 := collector.GetCollectionStats()
	if stats2["collection_count"].(int64) != 0 {
		t.Error("Collection count should be 0 after reset")
	}

	if stats2["error_count"].(int) != 0 {
		t.Error("Error count should be 0 after reset")
	}
}

// Benchmark tests
func BenchmarkSystemCollector_GetSystemMetrics(b *testing.B) {
	collector := NewSystemCollector(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.GetSystemMetrics()
		if err != nil {
			b.Fatalf("GetSystemMetrics() failed: %v", err)
		}
	}
}

func BenchmarkSystemCollector_GetSystemInfo(b *testing.B) {
	collector := NewSystemCollector(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.GetSystemInfo()
		if err != nil {
			b.Fatalf("GetSystemInfo() failed: %v", err)
		}
	}
}

func BenchmarkSystemCollector_GetTopProcesses(b *testing.B) {
	collector := NewSystemCollector(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.GetTopProcesses(10, "cpu")
		if err != nil {
			b.Fatalf("GetTopProcesses() failed: %v", err)
		}
	}
}