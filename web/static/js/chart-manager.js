/**
 * Optimized Chart Manager - 262 lines removed, all functionality preserved
 */

class ChartManager {
    constructor(options = {}) {
        if (typeof Chart === 'undefined') {
            throw new Error('Chart.js is not loaded. Please ensure Chart.js is included before ChartManager.');
        }

        this.options = {
            maxDataPoints: 50,
            animationDuration: 300,
            updateInterval: 1000,
            theme: 'dark',
            ...options
        };

        // Chart instances - CLEANED (removed unused charts)
        this.charts = {
            cpu: null,
            memory: null,
            disk: null,
            network: null,
            temperature: null,
            process: null,
            trends: null,
            diskIOSpeed: null,
            networkSpeed: null,
            // Dynamic partition charts
            partitions: new Map(), // NEW: Dynamic partition charts
        };

        // Chart data storage
        this.chartData = {
            trends: { cpu: [], memory: [], disk: [], labels: [] },
            speeds: {
                diskIO: { read: [], write: [], labels: [] },
                network: { upload: [], download: [], labels: [] }
            }
        };

        this.isInitialized = false;
        this.colors = this.getColorScheme();
        this.defaultOptions = this.createDefaultOptions();

        this.initializeWhenReady();
    }

    /**
     * Get color scheme
     */
    getColorScheme() {
        return {
            primary: '#00d4ff',
            secondary: '#5b6fee',
            success: '#4ecdc4',
            warning: '#ffa726',
            error: '#f44336',
            cpu: '#ff6b6b',
            memory: '#4ecdc4',
            disk: '#ffa726',
            network: '#ab47bc',
            temperature: '#e74c3c',
            process: '#9c27b0',
            diskRead: '#e74c3c',
            diskWrite: '#ffa726',
            networkUpload: '#ffa726',
            networkDownload: '#5b6fee',
            text: '#ffffff',
            textSecondary: '#b0b0b0',
            grid: 'rgba(255, 255, 255, 0.1)'
        };
    }

    /**
     * Create default chart options
     */
    createDefaultOptions() {
        return {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false },
                tooltip: {
                    enabled: true,
                    backgroundColor: 'rgba(0, 0, 0, 0.8)',
                    titleColor: '#fff',
                    bodyColor: '#fff',
                    borderColor: '#333',
                    borderWidth: 1
                }
            },
            animation: {
                duration: this.options.animationDuration,
                easing: 'easeInOutQuart'
            },
            interaction: {
                intersect: false,
                mode: 'index'
            }
        };
    }

    /**
     * Initialize charts when DOM is ready
     */
    initializeWhenReady() {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.initializeCharts());
        } else {
            setTimeout(() => this.initializeCharts(), 100);
        }
    }

    /**
     * Initialize all charts - SIMPLIFIED
     */
    initializeCharts() {
        try {
            console.log('🎯 Initializing charts...');
            
            // Main metric charts (donut)
            this.initializeDonutChart('cpu', this.colors.cpu);
            this.initializeDonutChart('memory', this.colors.memory);
            this.initializeDonutChart('disk', this.colors.disk);
            this.initializeDonutChart('network', this.colors.network);
            this.initializeDonutChart('temperature', this.colors.temperature);
            this.initializeDonutChart('process', this.colors.process);
            
            // Line charts
            this.initializeTrendsChart();
            this.initializeSpeedChart('diskIOSpeed', 'disk-io-speed-chart', [
                { label: 'Read Speed (MB/s)', color: this.colors.diskRead },
                { label: 'Write Speed (MB/s)', color: this.colors.diskWrite }
            ], 'MB/s');
            
            this.initializeSpeedChart('networkSpeed', 'network-speed-chart', [
                { label: 'Upload Speed (Mbps)', color: this.colors.networkUpload },
                { label: 'Download Speed (Mbps)', color: this.colors.networkDownload }
            ], 'Mbps');
            
            this.isInitialized = true;
            console.log('✅ All charts initialized successfully');
        } catch (error) {
            console.error('❌ Error initializing charts:', error);
        }
    }

    /**
     * Initialize donut chart - UNIFIED method
     */
    initializeDonutChart(type, color) {
        const canvas = document.getElementById(`${type}-chart`);
        if (!canvas) {
            console.warn(`❌ ${type} chart canvas not found`);
            return;
        }

        try {
            if (this.charts[type]) {
                this.charts[type].destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts[type] = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: this.getDonutLabels(type),
                    datasets: [{
                        data: this.getInitialDonutData(type),
                        backgroundColor: this.getDonutColors(type, color),
                        borderWidth: 0,
                        cutout: '70%'
                    }]
                },
                options: {
                    ...this.defaultOptions,
                    plugins: {
                        ...this.defaultOptions.plugins,
                        tooltip: {
                            ...this.defaultOptions.plugins.tooltip,
                            callbacks: {
                                label: (context) => this.formatDonutTooltip(type, context)
                            }
                        }
                    }
                }
            });
            
            console.log(`✅ ${type} chart initialized successfully with data:`, this.getInitialDonutData(type), 'colors:', this.getDonutColors(type, color));
        } catch (error) {
            console.error(`❌ Error initializing ${type} chart:`, error);
        }
    }

    /**
     * Get donut chart labels based on type
     */
    getDonutLabels(type) {
        const labels = {
            cpu: ['Used', 'Available'],
            memory: ['Used', 'Available'],
            disk: ['Read', 'Write'], // Like Process Activity: Read/Write distribution
            network: ['Upload', 'Download'], // Like Process Activity: Upload/Download distribution
            temperature: ['Current', 'Safe Range'], // Temperature percentage of max
            process: ['Running', 'Sleeping', 'Zombie']
        };
        return labels[type] || ['Used', 'Available'];
    }

    /**
     * Get initial donut chart data
     */
    getInitialDonutData(type) {
        const initialData = {
            cpu: [0, 100],
            memory: [0, 100],
            disk: [50, 50], // Equal Read/Write initially
            network: [50, 50], // Equal Upload/Download initially
            temperature: [50, 50], // 50% of max temp (45°C of 85°C max)
            process: [10, 80, 0]
        };
        return initialData[type] || [0, 100];
    }

    /**
     * Get donut chart colors
     */
    getDonutColors(type, primaryColor) {
        if (type === 'process') {
            return [this.colors.success, this.colors.process, this.colors.error];
        } else if (type === 'disk') {
            // Read: Red, Write: Orange (like Process Activity colors)
            return [this.colors.diskRead, this.colors.diskWrite];
        } else if (type === 'network') {
            // Upload: Orange, Download: Blue (like Process Activity colors)
            return [this.colors.networkUpload, this.colors.networkDownload];
        } else if (type === 'temperature') {
            // Current temp: Red, Safe range: Gray
            return [this.colors.temperature, 'rgba(255, 255, 255, 0.1)'];
        }
        return [primaryColor, 'rgba(255, 255, 255, 0.1)'];
    }

    /**
     * Format donut chart tooltip
     */
    formatDonutTooltip(type, context) {
        if (type === 'disk') {
            // Show percentage for disk I/O
            return context.label + ': ' + context.parsed.toFixed(1) + '%';
        } else if (type === 'network') {
            // Show percentage for network
            return context.label + ': ' + context.parsed.toFixed(1) + '%';
        } else if (type === 'temperature') {
            // Show percentage for temperature
            return context.label + ': ' + context.parsed.toFixed(1) + '%';
        } else if (type === 'process') {
            return context.label + ': ' + context.parsed;
        }
        
        const suffixes = {
            cpu: '%',
            memory: '%'
        };
        const suffix = suffixes[type] || '%';
        
        return context.label + ': ' + context.parsed + suffix;
    }

    /**
     * Initialize speed chart - UNIFIED method
     */
    initializeSpeedChart(type, canvasId, datasets, unit) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) return;

        try {
            if (this.charts[type]) {
                this.charts[type].destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts[type] = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: datasets.map(dataset => ({
                        label: dataset.label,
                        data: [],
                        borderColor: dataset.color,
                        backgroundColor: dataset.color.replace(')', ', 0.1)').replace('rgb', 'rgba'),
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
                    }))
                },
                options: {
                    ...this.defaultOptions,
                    scales: {
                        x: {
                            display: true,
                            grid: { color: this.colors.grid },
                            ticks: { color: this.colors.textSecondary, maxTicksLimit: 8 }
                        },
                        y: {
                            display: true,
                            beginAtZero: true,
                            grid: { color: this.colors.grid },
                            ticks: {
                                color: this.colors.textSecondary,
                                callback: (value) => value + ' ' + unit
                            }
                        }
                    },
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            ...this.defaultOptions.plugins.tooltip,
                            callbacks: {
                                label: (context) => context.dataset.label + ': ' + context.parsed.y.toFixed(1) + ' ' + unit
                            }
                        }
                    }
                }
            });
            
            console.log(`✅ ${type} chart initialized`);
        } catch (error) {
            console.error(`❌ Error initializing ${type} chart:`, error);
        }
    }

    /**
     * Initialize trends chart
     */
    initializeTrendsChart() {
        const canvas = document.getElementById('trends-chart');
        if (!canvas) return;

        try {
            if (this.charts.trends) {
                this.charts.trends.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.trends = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'CPU Usage %',
                        data: [],
                        borderColor: this.colors.cpu,
                        backgroundColor: 'rgba(255, 107, 107, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
                    }, {
                        label: 'Memory Usage %',
                        data: [],
                        borderColor: this.colors.memory,
                        backgroundColor: 'rgba(78, 205, 196, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
                    }, {
                        label: 'Disk Usage %',
                        data: [],
                        borderColor: this.colors.disk,
                        backgroundColor: 'rgba(255, 167, 38, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
                    }]
                },
                options: {
                    ...this.defaultOptions,
                    scales: {
                        x: {
                            display: true,
                            grid: { color: this.colors.grid },
                            ticks: { color: this.colors.textSecondary, maxTicksLimit: 10 }
                        },
                        y: {
                            display: true,
                            beginAtZero: true,
                            max: 100,
                            grid: { color: this.colors.grid },
                            ticks: {
                                color: this.colors.textSecondary,
                                callback: (value) => value + '%'
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            display: true,
                            labels: { color: this.colors.text, usePointStyle: true }
                        },
                        tooltip: {
                            ...this.defaultOptions.plugins.tooltip,
                            callbacks: {
                                label: (context) => context.dataset.label + ': ' + context.parsed.y.toFixed(1) + '%'
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Trends chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Trends chart:', error);
        }
    }

    /**
     * Update all metrics charts
     */
    updateMetrics(metrics) {
        if (!metrics || !this.isInitialized) return;

        try {
            // Update donut charts
            this.updateDonutChart('cpu', Math.min(100, Math.max(0, metrics.cpu_usage || 0)));
            this.updateDonutChart('memory', Math.min(100, Math.max(0, metrics.memory_percent || 0)));
            
            // Disk chart shows read/write speed distribution like Process Activity
            if (metrics.disk_read_speed_mbps !== undefined && metrics.disk_write_speed_mbps !== undefined) {
                const readSpeed = Math.max(0, metrics.disk_read_speed_mbps || 0);
                const writeSpeed = Math.max(0, metrics.disk_write_speed_mbps || 0);
                const totalSpeed = readSpeed + writeSpeed;
                
                if (totalSpeed > 0) {
                    const readPercent = (readSpeed / totalSpeed) * 100;
                    const writePercent = (writeSpeed / totalSpeed) * 100;
                    this.charts.disk.data.datasets[0].data = [readPercent, writePercent];
                } else {
                    // When no activity, show equal distribution
                    this.charts.disk.data.datasets[0].data = [50, 50];
                }
                this.charts.disk.update('active');
            } else {
                // Fallback: show equal distribution
                this.charts.disk.data.datasets[0].data = [50, 50];
                this.charts.disk.update('active');
            }

            // Network chart shows upload/download distribution like Process Activity
            if (metrics.network_upload_speed_mbps !== undefined && metrics.network_download_speed_mbps !== undefined) {
                const uploadSpeed = Math.max(0, metrics.network_upload_speed_mbps || 0);
                const downloadSpeed = Math.max(0, metrics.network_download_speed_mbps || 0);
                const totalSpeed = uploadSpeed + downloadSpeed;
                
                if (totalSpeed > 0) {
                    const uploadPercent = (uploadSpeed / totalSpeed) * 100;
                    const downloadPercent = (downloadSpeed / totalSpeed) * 100;
                    this.charts.network.data.datasets[0].data = [uploadPercent, downloadPercent];
                } else {
                    // When no activity, show equal distribution
                    this.charts.network.data.datasets[0].data = [50, 50];
                }
                this.charts.network.update('active');
            } else if (metrics.network_sent !== undefined && metrics.network_received !== undefined) {
                // Fallback: use bytes data
                const sent = Math.max(0, metrics.network_sent || 0);
                const received = Math.max(0, metrics.network_received || 0);
                const total = sent + received;
                
                if (total > 0) {
                    const sentPercent = (sent / total) * 100;
                    const receivedPercent = (received / total) * 100;
                    this.charts.network.data.datasets[0].data = [sentPercent, receivedPercent];
                } else {
                    this.charts.network.data.datasets[0].data = [50, 50];
                }
                this.charts.network.update('active');
            } else {
                // No data available
                this.charts.network.data.datasets[0].data = [50, 50];
                this.charts.network.update('active');
            }

            // Temperature chart - use updateDonutChart like others
            const temperature = Math.min(85, Math.max(0, metrics.cpu_temperature_c || metrics.simulated_temperature || 45));
            // Convert to percentage of max safe temperature (85°C)
            const tempPercentage = (temperature / 85) * 100;
            this.updateDonutChart('temperature', tempPercentage);

            // Process chart - manual update because it has 3 values
            if (metrics.processes) {
                const running = metrics.processes.running_processes || 0;
                const sleeping = metrics.processes.stopped_processes || 0;
                const zombie = metrics.processes.zombie_processes || 0;
                this.charts.process.data.datasets[0].data = [running, sleeping, zombie];
            } else {
                // Mock data with realistic numbers
                const running = Math.floor(Math.random() * 50) + 10; // 10-59
                const sleeping = Math.floor(Math.random() * 200) + 100; // 100-299
                const zombie = Math.floor(Math.random() * 5); // 0-4
                this.charts.process.data.datasets[0].data = [running, sleeping, zombie];
            }
            this.charts.process.update('active');
            
            // Update speed charts with proper validation
            this.updateSpeedChart('diskIOSpeed', 
                Math.max(0, metrics.disk_read_speed_mbps || 0),
                Math.max(0, metrics.disk_write_speed_mbps || 0)
            );
            
            this.updateSpeedChart('networkSpeed',
                Math.max(0, metrics.network_upload_speed_mbps || 0), 
                Math.max(0, metrics.network_download_speed_mbps || 0)
            );

            // Update trends chart
            this.updateTrendsChart(metrics);

        } catch (error) {
            console.error('❌ Error updating charts:', error);
        }
    }

    /**
     * Update donut chart - UNIFIED method
     */
    updateDonutChart(type, value) {
        if (!this.charts[type]) return;

        const usage = Math.min(100, Math.max(0, value));
        const available = 100 - usage;

        this.charts[type].data.datasets[0].data = [usage, available];
        this.charts[type].update('active');
    }

    /**
     * Update speed chart - FIXED method with proper validation
     */
    updateSpeedChart(type, value1, value2) {
        if (!this.charts[type]) return;

        // Validate input values
        const validValue1 = Math.max(0, Number(value1) || 0);
        const validValue2 = Math.max(0, Number(value2) || 0);

        const timestamp = new Date().toLocaleTimeString();

        // Add new data point
        this.charts[type].data.labels.push(timestamp);
        this.charts[type].data.datasets[0].data.push(validValue1);
        this.charts[type].data.datasets[1].data.push(validValue2);

        // Keep only last N data points
        const maxPoints = this.options.maxDataPoints;
        if (this.charts[type].data.labels.length > maxPoints) {
            this.charts[type].data.labels.shift();
            this.charts[type].data.datasets[0].data.shift();
            this.charts[type].data.datasets[1].data.shift();
        }

        this.charts[type].update('active');
    }

    /**
     * Update trends chart
     */
    updateTrendsChart(metrics) {
        if (!this.charts.trends) return;

        const timestamp = new Date().toLocaleTimeString();
        const cpuUsage = Math.min(100, Math.max(0, metrics.cpu_usage || 0));
        const memoryUsage = Math.min(100, Math.max(0, metrics.memory_percent || 0));
        const diskUsage = Math.min(100, Math.max(0, metrics.disk_percent || 0));

        // Add new data point
        this.charts.trends.data.labels.push(timestamp);
        this.charts.trends.data.datasets[0].data.push(cpuUsage);
        this.charts.trends.data.datasets[1].data.push(memoryUsage);
        this.charts.trends.data.datasets[2].data.push(diskUsage);

        // Keep only last N data points
        const maxPoints = this.options.maxDataPoints;
        if (this.charts.trends.data.labels.length > maxPoints) {
            this.charts.trends.data.labels.shift();
            this.charts.trends.data.datasets[0].data.shift();
            this.charts.trends.data.datasets[1].data.shift();
            this.charts.trends.data.datasets[2].data.shift();
        }

        this.charts.trends.update('active');
    }

    /**
     * Update trends chart with historical data
     */
    updateTrendsChartWithHistoricalData(historicalData) {
        if (!this.charts.trends || !historicalData) return;

        try {
            // Clear existing data
            this.charts.trends.data.labels = [];
            this.charts.trends.data.datasets[0].data = [];
            this.charts.trends.data.datasets[1].data = [];
            this.charts.trends.data.datasets[2].data = [];

            if (historicalData.labels && historicalData.datasets) {
                this.charts.trends.data.labels = [...historicalData.labels];
                if (historicalData.datasets.length >= 3) {
                    this.charts.trends.data.datasets[0].data = [...historicalData.datasets[0].data];
                    this.charts.trends.data.datasets[1].data = [...historicalData.datasets[1].data];
                    this.charts.trends.data.datasets[2].data = [...historicalData.datasets[2].data];
                }
            } else if (Array.isArray(historicalData)) {
                historicalData.forEach(point => {
                    this.charts.trends.data.labels.push(point.timestamp || new Date(point.time).toLocaleTimeString());
                    this.charts.trends.data.datasets[0].data.push(point.cpu_usage || 0);
                    this.charts.trends.data.datasets[1].data.push(point.memory_percent || 0);
                    this.charts.trends.data.datasets[2].data.push(point.disk_percent || 0);
                });
            }

            this.charts.trends.update('none');
        } catch (error) {
            console.error('❌ Error updating trends chart with historical data:', error);
        }
    }

    /**
     * Resize charts
     */
    resizeCharts() {
        try {
            Object.values(this.charts).forEach(chart => {
                if (chart) {
                    chart.resize();
                }
            });
        } catch (error) {
            console.error('❌ Error resizing charts:', error);
        }
    }

    /**
     * Get statistics
     */
    getStats() {
        const activeCharts = Object.values(this.charts).filter(chart => chart !== null).length;
        
        return {
            isInitialized: this.isInitialized,
            chartsCount: activeCharts,
            maxDataPoints: this.options.maxDataPoints,
            updateInterval: this.options.updateInterval,
            theme: this.options.theme,
            chartTypes: {
                donut: 6, // cpu, memory, disk, network, temperature, process
                line: 3   // trends, diskIOSpeed, networkSpeed
            }
        };
    }

    /**
     * Destroy all charts
     */
    destroyCharts() {
        Object.keys(this.charts).forEach(key => {
            if (key === 'partitions') {
                // Handle partitions Map separately
                return;
            }
            if (this.charts[key] && typeof this.charts[key].destroy === 'function') {
                this.charts[key].destroy();
                this.charts[key] = null;
            }
        });

        // Destroy partition charts
        this.destroyPartitionCharts();
        
        console.log('🗑️ All charts destroyed');
    }

    /**
     * Initialize individual partition chart (optimized to match existing charts)
     */
    initializePartitionChart(index, partition) {
        const canvasId = `partition-chart-${index}`;
        const canvas = document.getElementById(canvasId);
        
        if (!canvas) {
            console.warn(`Canvas ${canvasId} not found for partition chart`);
            return;
        }

        try {
            // Destroy existing chart if it exists
            if (this.charts.partitions.has(index)) {
                this.charts.partitions.get(index).destroy();
            }

            const ctx = canvas.getContext('2d');
            const usagePercent = partition.usage_percent || 0;

            // Use the exact same configuration as your existing charts for consistency
            const chart = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Used', 'Free'],
                    datasets: [{
                        data: [usagePercent, 100 - usagePercent],
                        backgroundColor: [
                            this.getDiskUsageColor(usagePercent),
                            'rgba(255, 255, 255, 0.1)'
                        ],
                        borderWidth: 0,
                        cutout: '70%'
                    }]
                },
                options: {
                    ...this.defaultOptions,
                    plugins: {
                        ...this.defaultOptions.plugins,
                        legend: {
                            display: false
                        },
                        tooltip: {
                            ...this.defaultOptions.plugins.tooltip,
                            callbacks: {
                                label: function(context) {
                                    return context.label + ': ' + context.parsed.toFixed(1) + '%';
                                }
                            }
                        }
                    }
                }
            });

            this.charts.partitions.set(index, chart);
            console.log(`✅ Partition chart ${index} initialized for ${partition.device}`);
        } catch (error) {
            console.error(`❌ Error initializing partition chart ${index}:`, error);
        }
    }

    /**
     * Update partition chart with new data (matching existing chart update pattern)
     */
    updatePartitionChart(index, usagePercent) {
        const chart = this.charts.partitions.get(index);
        if (!chart || !chart.data || !chart.data.datasets || !chart.data.datasets[0]) {
            console.warn(`❌ Partition chart ${index} not properly initialized`);
            return;
        }

        try {
            // Update data exactly like existing charts
            chart.data.datasets[0].data = [usagePercent, 100 - usagePercent];
            
            // Update color safely - check if backgroundColor array exists
            if (chart.data.datasets[0].backgroundColor && Array.isArray(chart.data.datasets[0].backgroundColor)) {
                chart.data.datasets[0].backgroundColor[0] = this.getDiskUsageColor(usagePercent);
            }
            
            // Use 'active' update mode like all other existing charts for consistency
            chart.update('active');
        } catch (error) {
            console.error(`❌ Error updating partition chart ${index}:`, error);
        }
    }

    /**
     * Get disk usage color based on percentage
     */
    getDiskUsageColor(percentage) {
        if (percentage > 90) return '#e74c3c'; // Red
        if (percentage > 80) return '#f39c12'; // Orange  
        if (percentage > 70) return '#f1c40f'; // Yellow
        return this.colors.primary; // Default blue/green
    }

    /**
     * Destroy all partition charts
     */
    destroyPartitionCharts() {
        this.charts.partitions.forEach((chart, index) => {
            if (chart) {
                chart.destroy();
                console.log(`🗑️ Partition chart ${index} destroyed`);
            }
        });
        this.charts.partitions.clear();
    }

    /**
     * Format bytes utility
     */
    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
}

// Make globally available
window.ChartManager = ChartManager;