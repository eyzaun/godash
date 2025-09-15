package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eyzaun/godash/internal/collector"
	"github.com/eyzaun/godash/internal/models"
	"github.com/fatih/color"
)

func main() {
	RunCLI()
}

func RunCLI() {
	// Parse command line flags
	var config CLIConfig
	flag.DurationVar(&config.Interval, "interval", 5*time.Second, "Update interval")
	flag.BoolVar(&config.OutputJSON, "json", false, "Output in JSON format")
	flag.BoolVar(&config.ShowProcesses, "processes", false, "Show top processes")
	flag.BoolVar(&config.Continuous, "continuous", false, "Continuous monitoring mode")
	flag.IntVar(&config.Count, "count", 0, "Number of updates (0 for infinite)")
	flag.BoolVar(&config.NoColor, "no-color", false, "Disable colored output")

	help := flag.Bool("help", false, "Show help message")
	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	// Handle help and version
	if *help {
		fmt.Println("GoDash System Monitor - CLI Tool")
		fmt.Println()
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Println("GoDash System Monitor v1.0.0")
		return
	}

	// Disable colors if requested or if not a terminal
	if config.NoColor {
		color.NoColor = true
	}

	// Create system collector
	systemCollector := collector.NewSystemCollector(nil)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		if !config.NoColor {
			fmt.Println(color.YellowString("\nShutting down gracefully..."))
		} else {
			fmt.Println("\nShutting down gracefully...")
		}
		cancel()
	}()

	// Get system info once
	systemInfo, err := systemCollector.GetSystemInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting system info: %v\n", err)
		os.Exit(1)
	}

	// Single run mode
	if !config.Continuous {
		snapshot, err := systemCollector.GetMetricsSnapshot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error collecting metrics: %v\n", err)
			os.Exit(1)
		}

		if config.OutputJSON {
			if err := printJSON(snapshot); err != nil {
				fmt.Fprintf(os.Stderr, "Error printing JSON: %v\n", err)
				os.Exit(1)
			}
		} else {
			printHeader(config.NoColor)
			printSystemInfo(systemInfo, config.NoColor)
			printMetrics(&snapshot.SystemMetrics, config.NoColor)

			if config.ShowProcesses {
				printTopProcesses(snapshot.TopProcesses, config.NoColor)
			}
		}
		return
	}

	// Continuous monitoring mode
	if !config.OutputJSON {
		printHeader(config.NoColor)
		printSystemInfo(systemInfo, config.NoColor)
	}

	metricsChan := systemCollector.StartCollection(ctx, config.Interval)
	updateCount := 0

	for {
		select {
		case metrics, ok := <-metricsChan:
			if !ok {
				return
			}

			if config.OutputJSON {
				if err := printJSON(metrics); err != nil {
					fmt.Fprintf(os.Stderr, "Error printing JSON: %v\n", err)
				}
			} else {
				// Clear screen for continuous updates (except first update)
				if updateCount > 0 {
					clearScreen()
					printHeader(config.NoColor)
					printSystemInfo(systemInfo, config.NoColor)
				}

				printMetrics(metrics, config.NoColor)

				if config.ShowProcesses {
					topProcesses, err := systemCollector.GetTopProcesses(10, "cpu")
					if err == nil {
						printTopProcesses(topProcesses, config.NoColor)
					}
				}
			}

			updateCount++
			if config.Count > 0 && updateCount >= config.Count {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// CLIConfig holds CLI configuration
type CLIConfig struct {
	Interval      time.Duration
	OutputJSON    bool
	ShowProcesses bool
	Continuous    bool
	Count         int
	NoColor       bool
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes uint64) string {
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

// formatPercent formats percentage with color coding
func formatPercent(percent float64, noColor bool) string {
	if noColor {
		return fmt.Sprintf("%.1f%%", percent)
	}

	var colorFunc func(string, ...interface{}) string
	switch {
	case percent >= 90:
		colorFunc = color.New(color.FgRed, color.Bold).SprintfFunc()
	case percent >= 75:
		colorFunc = color.New(color.FgYellow).SprintfFunc()
	case percent >= 50:
		colorFunc = color.New(color.FgBlue).SprintfFunc()
	default:
		colorFunc = color.New(color.FgGreen).SprintfFunc()
	}

	return colorFunc("%.1f%%", percent)
}

// printHeader prints the application header
func printHeader(noColor bool) {
	if noColor {
		fmt.Println("=" + strings.Repeat("=", 50) + "=")
		fmt.Println("                 GoDash System Monitor")
		fmt.Println("=" + strings.Repeat("=", 50) + "=")
	} else {
		cyan := color.New(color.FgCyan, color.Bold)
		cyan.Println("=" + strings.Repeat("=", 50) + "=")
		cyan.Println("                 GoDash System Monitor")
		cyan.Println("=" + strings.Repeat("=", 50) + "=")
	}
}

// printSystemInfo prints basic system information
func printSystemInfo(systemInfo *models.SystemInfo, noColor bool) {
	if noColor {
		fmt.Printf("System: %s %s (%s)\n", systemInfo.Platform, systemInfo.PlatformVersion, systemInfo.KernelArch)
		fmt.Printf("Kernel: %s\n", systemInfo.KernelVersion)
		fmt.Printf("Uptime: %s\n", time.Since(systemInfo.BootTime).Round(time.Minute))
		fmt.Printf("Processes: %d\n", systemInfo.Processes)
	} else {
		bold := color.New(color.Bold)
		fmt.Print("System: ")
		bold.Printf("%s %s (%s)\n", systemInfo.Platform, systemInfo.PlatformVersion, systemInfo.KernelArch)
		fmt.Print("Kernel: ")
		bold.Printf("%s\n", systemInfo.KernelVersion)
		fmt.Print("Uptime: ")
		bold.Printf("%s\n", time.Since(systemInfo.BootTime).Round(time.Minute))
		fmt.Print("Processes: ")
		bold.Printf("%d\n", systemInfo.Processes)
	}
	fmt.Println()
}

// printMetrics prints system metrics in a formatted way
func printMetrics(metrics *models.SystemMetrics, noColor bool) {
	timestamp := metrics.Timestamp.Format("15:04:05")

	if noColor {
		fmt.Printf("[%s] System Metrics for %s\n", timestamp, metrics.Hostname)
		fmt.Println(strings.Repeat("-", 60))
	} else {
		cyan := color.New(color.FgCyan)
		cyan.Printf("[%s] System Metrics for %s\n", timestamp, metrics.Hostname)
		fmt.Println(strings.Repeat("-", 60))
	}

	// CPU Information
	printCPUMetrics(&metrics.CPU, noColor)

	// Memory Information
	printMemoryMetrics(&metrics.Memory, noColor)

	// Disk Information
	printDiskMetrics(&metrics.Disk, noColor)

	// Network Information
	printNetworkMetrics(&metrics.Network, noColor)

	fmt.Println()
}

// printCPUMetrics prints CPU metrics
func printCPUMetrics(cpu *models.CPUMetrics, noColor bool) {
	var header func(string, ...interface{}) string
	if noColor {
		header = fmt.Sprintf
	} else {
		header = color.New(color.FgMagenta, color.Bold).SprintfFunc()
	}

	fmt.Println(header("CPU:"))
	fmt.Printf("  Usage: %s", formatPercent(cpu.Usage, noColor))
	fmt.Printf(" | Cores: %d", cpu.Cores)
	fmt.Printf(" | Frequency: %.0f MHz\n", cpu.Frequency)

	if len(cpu.LoadAvg) >= 3 {
		fmt.Printf("  Load Average: %.2f, %.2f, %.2f (1m, 5m, 15m)\n",
			cpu.LoadAvg[0], cpu.LoadAvg[1], cpu.LoadAvg[2])
	}

	// Show per-core usage if available
	if len(cpu.CoreUsage) > 0 && len(cpu.CoreUsage) <= 8 { // Only show for reasonable number of cores
		fmt.Print("  Per-Core: ")
		for i, usage := range cpu.CoreUsage {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("CPU%d: %s", i, formatPercent(usage, noColor))
		}
		fmt.Println()
	}
	fmt.Println()
}

// printMemoryMetrics prints memory metrics
func printMemoryMetrics(memory *models.MemoryMetrics, noColor bool) {
	var header func(string, ...interface{}) string
	if noColor {
		header = fmt.Sprintf
	} else {
		header = color.New(color.FgYellow, color.Bold).SprintfFunc()
	}

	fmt.Println(header("Memory:"))
	fmt.Printf("  Usage: %s", formatPercent(memory.Percent, noColor))
	fmt.Printf(" | Used: %s", formatBytes(memory.Used))
	fmt.Printf(" | Total: %s\n", formatBytes(memory.Total))
	fmt.Printf("  Available: %s", formatBytes(memory.Available))
	fmt.Printf(" | Free: %s", formatBytes(memory.Free))
	fmt.Printf(" | Cached: %s\n", formatBytes(memory.Cached))

	if memory.SwapTotal > 0 {
		fmt.Printf("  Swap: %s", formatPercent(memory.SwapPercent, noColor))
		fmt.Printf(" | Used: %s", formatBytes(memory.SwapUsed))
		fmt.Printf(" | Total: %s\n", formatBytes(memory.SwapTotal))
	}
	fmt.Println()
}

// printDiskMetrics prints disk metrics
func printDiskMetrics(disk *models.DiskMetrics, noColor bool) {
	var header func(string, ...interface{}) string
	if noColor {
		header = fmt.Sprintf
	} else {
		header = color.New(color.FgBlue, color.Bold).SprintfFunc()
	}

	fmt.Println(header("Disk:"))
	fmt.Printf("  Usage: %s", formatPercent(disk.Percent, noColor))
	fmt.Printf(" | Used: %s", formatBytes(disk.Used))
	fmt.Printf(" | Total: %s\n", formatBytes(disk.Total))
	fmt.Printf("  Free: %s", formatBytes(disk.Free))

	// I/O Statistics
	if disk.IOStats.ReadBytes > 0 || disk.IOStats.WriteBytes > 0 {
		fmt.Printf(" | Read: %s", formatBytes(disk.IOStats.ReadBytes))
		fmt.Printf(" | Write: %s\n", formatBytes(disk.IOStats.WriteBytes))
		fmt.Printf("  Read Ops: %d | Write Ops: %d\n",
			disk.IOStats.ReadOps, disk.IOStats.WriteOps)
	} else {
		fmt.Println()
	}

	// Show individual partitions if there are multiple
	if len(disk.Partitions) > 1 {
		fmt.Println("  Partitions:")
		for _, partition := range disk.Partitions {
			if partition.Total > 0 {
				fmt.Printf("    %s: %s (%s) - %s used\n",
					partition.Mountpoint,
					formatBytes(partition.Total),
					partition.Fstype,
					formatPercent(partition.Percent, noColor))
			}
		}
	}
	fmt.Println()
}

// printNetworkMetrics prints network metrics
func printNetworkMetrics(network *models.NetworkMetrics, noColor bool) {
	var header func(string, ...interface{}) string
	if noColor {
		header = fmt.Sprintf
	} else {
		header = color.New(color.FgGreen, color.Bold).SprintfFunc()
	}

	fmt.Println(header("Network:"))
	fmt.Printf("  Total Sent: %s", formatBytes(network.TotalSent))
	fmt.Printf(" | Total Received: %s\n", formatBytes(network.TotalReceived))

	// Show active interfaces
	if len(network.Interfaces) > 0 {
		fmt.Println("  Interfaces:")
		for _, iface := range network.Interfaces {
			if iface.BytesSent > 0 || iface.BytesRecv > 0 {
				fmt.Printf("    %s: ↑%s ↓%s",
					iface.Name,
					formatBytes(iface.BytesSent),
					formatBytes(iface.BytesRecv))
				if iface.Errors > 0 || iface.Drops > 0 {
					fmt.Printf(" (Errors: %d, Drops: %d)", iface.Errors, iface.Drops)
				}
				fmt.Println()
			}
		}
	}
	fmt.Println()
}

// printTopProcesses prints top processes
func printTopProcesses(processes []models.ProcessInfo, noColor bool) {
	if len(processes) == 0 {
		return
	}

	var header func(string, ...interface{}) string
	if noColor {
		header = fmt.Sprintf
	} else {
		header = color.New(color.FgRed, color.Bold).SprintfFunc()
	}

	fmt.Println(header("Top Processes:"))
	fmt.Printf("%-8s %-20s %-8s %-10s %-10s\n", "PID", "Name", "Status", "CPU%", "Memory")
	fmt.Println(strings.Repeat("-", 60))

	for _, proc := range processes {
		fmt.Printf("%-8d %-20s %-8s %7.1f%% %10s\n",
			proc.PID,
			truncateString(proc.Name, 20),
			proc.Status,
			proc.CPUPercent,
			formatBytes(proc.MemoryBytes))
	}
	fmt.Println()
}

// truncateString truncates a string to the specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// printJSON prints metrics in JSON format
func printJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}
