package collector

import (
	"fmt"

	"github.com/eyzaun/godash/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessCollector handles process-related metrics collection
type ProcessCollector struct{}

// NewProcessCollector creates a new process collector
func NewProcessCollector() *ProcessCollector {
	return &ProcessCollector{}
}

// GetProcessActivity collects real process activity information
func (p *ProcessCollector) GetProcessActivity() (*models.ProcessActivity, error) {
	// Get all running processes
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("failed to get process IDs: %w", err)
	}

	// Count total processes
	totalProcesses := len(pids)
	runningProcesses := 0
	stoppedProcesses := 0
	zombieProcesses := 0

	// Count processes by status - Windows-optimized approach
	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			// Process might have disappeared, skip it
			continue
		}

		// Windows'ta basit yaklaşım: process.Pids() çağrısından gelen her PID running kabul edilir
		// Çünkü Windows'ta process status bilgisi güvenilir değil
		runningProcesses++

		// Ek kontroller (opsiyonel)
		isRunning, err := proc.IsRunning()
		if err != nil || !isRunning {
			// Eğer process artık running değilse düzelt
			runningProcesses--
			stoppedProcesses++
		}
	}

	// Get top processes by CPU usage
	topProcesses, err := p.getTopProcesses(10)
	if err != nil {
		// Continue with empty top processes if this fails
		topProcesses = []models.ProcessInfo{}
	}

	return &models.ProcessActivity{
		TotalProcesses:   totalProcesses,
		RunningProcesses: runningProcesses,
		StoppedProcesses: stoppedProcesses,
		ZombieProcesses:  zombieProcesses,
		TopProcesses:     topProcesses,
	}, nil
}

// getTopProcesses gets top processes by CPU usage (internal method)
func (p *ProcessCollector) getTopProcesses(limit int) ([]models.ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, fmt.Errorf("failed to get process IDs: %w", err)
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
			continue
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
		var memoryBytes uint64
		if err == nil && memInfo != nil {
			memoryBytes = memInfo.RSS
		}

		statuses, err := proc.Status()
		var status string
		if err != nil {
			status = "unknown"
		} else if len(statuses) > 0 {
			status = statuses[0]
		} else {
			status = "unknown"
		}

		// Group by process name
		if existing, exists := processGroups[name]; exists {
			// Add to existing group
			existing.CPUPercent += cpuPercent
			existing.MemoryBytes += memoryBytes
			// Keep the lowest PID as representative
			if int32(pid) < existing.PID {
				existing.PID = int32(pid)
			}
		} else {
			// Create new group
			processGroups[name] = &models.ProcessInfo{
				PID:         int32(pid),
				Name:        name,
				CPUPercent:  cpuPercent,
				MemoryBytes: memoryBytes,
				Status:      status,
			}
		}
	}

	// Convert map to slice for sorting
	var processes []models.ProcessInfo
	for _, proc := range processGroups {
		processes = append(processes, *proc)
	}

	// Sort by CPU usage and return top processes
	// Simple bubble sort for top processes
	for i := 0; i < len(processes)-1; i++ {
		for j := 0; j < len(processes)-i-1; j++ {
			if processes[j].CPUPercent < processes[j+1].CPUPercent {
				processes[j], processes[j+1] = processes[j+1], processes[j]
			}
		}
	}

	if len(processes) > limit {
		processes = processes[:limit]
	}

	return processes, nil
}
