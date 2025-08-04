/**
 * Chart Manager for GoDash Dashboard (WITH DISK I/O & NETWORK SPEED SUPPORT)
 * Manages all Chart.js instances and real-time updates with speed monitoring
 */

class ChartManager {
    constructor(options = {}) {
        console.log('🎯 ChartManager constructor started with speed support');
        
        // Check if Chart.js is available
        if (typeof Chart === 'undefined') {
            throw new Error('Chart.js is not loaded. Please ensure Chart.js is included before ChartManager.');
        }

        this.options = {
            maxDataPoints: 50,
            animationDuration: 50,   // Ultra fast animation for real-time feel
            updateInterval: 500,     // Update charts every 500ms for real-time
            theme: 'dark',
            ...options
        };

        // Chart instances (DISK I/O ADDED)
        this.charts = {
            cpu: null,
            memory: null,
            disk: null,
            diskIO: null,     // NEW: Disk I/O speed chart
            network: null,
            trends: null
        };

        // Chart data storage (DISK I/O ADDED)
        this.chartData = {
            cpu: [],
            memory: [],
            disk: [],
            diskIO: [],       // NEW: Disk I/O speed data
            network: [],
            trends: {
                cpu: [],
                memory: [],
                disk: [],
                labels: []
            }
        };

        // State flags
        this.isLoadingHistorical = false;
        this.isInitialized = false;

        // Color schemes (DISK I/O COLOR ADDED)
        this.colors = this.getColorScheme();

        // Initialize Chart.js defaults
        this.setupChartDefaults();

        // Initialize charts when DOM is ready
        this.initializeWhenReady();
    }

    /**
     * Initialize charts when DOM elements are available
     */
    initializeWhenReady() {
        if (document.readyState === 'loading') {
            console.log('📝 DOM not ready, waiting for DOMContentLoaded...');
            document.addEventListener('DOMContentLoaded', () => {
                console.log('📝 DOMContentLoaded fired, initializing charts with speed support...');
                this.initializeCharts();
            });
        } else {
            console.log('📝 DOM ready, initializing charts with speed support immediately...');
            // DOM is already ready, initialize immediately
            setTimeout(() => this.initializeCharts(), 100); // Small delay to ensure elements are fully rendered
        }
        
        console.log('📊 Chart Manager initialized with speed support');
    }

    /**
     * Get color scheme based on theme (DISK I/O COLOR ADDED)
     */
    getColorScheme() {
        const colors = {
            primary: '#00d4ff',
            secondary: '#5b6fee',
            success: '#4ecdc4',
            warning: '#ffa726',
            error: '#f44336',
            cpu: '#ff6b6b',
            memory: '#4ecdc4',
            disk: '#ffa726',
            diskIO: '#e74c3c',      // NEW: Red for disk I/O
            network: '#ab47bc',
            background: 'rgba(0, 212, 255, 0.1)',
            border: 'rgba(0, 212, 255, 0.8)',
            text: '#ffffff',
            textSecondary: '#b0b0b0',
            grid: 'rgba(255, 255, 255, 0.1)'
        };

        return colors;
    }

    /**
     * Setup Chart.js global defaults
     */
    setupChartDefaults() {
        try {
            console.log('Setting up Chart.js defaults...');
            
            // Chart.js v3 configuration through default options in each chart
            this.defaultChartOptions = {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        enabled: true,
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        titleColor: '#fff',
                        bodyColor: '#fff'
                    }
                },
                animation: {
                    duration: this.options.animationDuration
                }
            };
            
            console.log('✅ Chart.js defaults configured');
        } catch (error) {
            console.error('❌ Error setting up Chart.js defaults:', error);
        }
    }

    /**
     * Initialize all charts (DISK I/O INCLUDED)
     */
    initializeCharts() {
        try {
            console.log('🎯 Initializing all charts with speed support...');
            
            // Check if DOM elements exist before creating charts (DISK I/O INCLUDED)
            const canvasIds = ['cpu-chart', 'memory-chart', 'disk-chart', 'disk-io-chart', 'network-chart', 'trends-chart'];
            const missingElements = [];
            
            canvasIds.forEach(id => {
                const element = document.getElementById(id);
                if (!element) {
                    missingElements.push(id);
                } else {
                    console.log(`✅ Found canvas element: ${id}`);
                }
            });
            
            if (missingElements.length > 0) {
                console.error('❌ Missing canvas elements:', missingElements);
                console.log('📝 Available elements in DOM:', 
                    Array.from(document.querySelectorAll('canvas')).map(c => c.id || 'no-id'));
                return;
            }
            
            this.initializeCPUChart();
            this.initializeMemoryChart();
            this.initializeDiskChart();
            this.initializeDiskIOChart(); // NEW: Initialize disk I/O chart
            this.initializeNetworkChart();
            this.initializeTrendsChart();
            
            this.isInitialized = true;
            console.log('✅ All charts initialized successfully with speed support');
        } catch (error) {
            console.error('❌ Error initializing charts:', error);
        }
    }

    /**
     * Initialize CPU Chart
     */
    initializeCPUChart() {
        const canvas = document.getElementById('cpu-chart');
        if (!canvas) {
            console.warn('❌ CPU chart canvas not found - element ID: cpu-chart');
            return;
        }
        console.log('✅ Found CPU chart canvas element');

        try {
            // Destroy existing chart if it exists to prevent Canvas reuse error
            if (this.charts.cpu) {
                this.charts.cpu.destroy();
                this.charts.cpu = null;
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.cpu = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Used', 'Available'],
                    datasets: [{
                        data: [0, 100],
                        backgroundColor: [this.colors.cpu, 'rgba(255, 255, 255, 0.1)'],
                        borderWidth: 0,
                        cutout: '70%'
                    }]
                },
                options: {
                    ...this.defaultChartOptions,
                    plugins: {
                        ...this.defaultChartOptions.plugins,
                        tooltip: {
                            ...this.defaultChartOptions.plugins.tooltip,
                            callbacks: {
                                label: function(context) {
                                    return context.label + ': ' + context.parsed + '%';
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ CPU chart initialized');
        } catch (error) {
            console.error('❌ Error initializing CPU chart:', error);
        }
    }

    /**
     * Initialize Memory Chart
     */
    initializeMemoryChart() {
        const canvas = document.getElementById('memory-chart');
        if (!canvas) {
            console.warn('❌ Memory chart canvas not found - element ID: memory-chart');
            return;
        }
        console.log('✅ Found Memory chart canvas element');

        try {
            // Destroy existing chart if it exists to prevent Canvas reuse error
            if (this.charts.memory) {
                this.charts.memory.destroy();
                this.charts.memory = null;
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.memory = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Used', 'Available'],
                    datasets: [{
                        data: [0, 100],
                        backgroundColor: [this.colors.memory, 'rgba(255, 255, 255, 0.1)'],
                        borderWidth: 0,
                        cutout: '70%'
                    }]
                },
                options: {
                    ...this.defaultChartOptions,
                    plugins: {
                        ...this.defaultChartOptions.plugins,
                        tooltip: {
                            ...this.defaultChartOptions.plugins.tooltip,
                            callbacks: {
                                label: function(context) {
                                    return context.label + ': ' + context.parsed + '%';
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Memory chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Memory chart:', error);
        }
    }

    /**
     * Initialize Disk Chart
     */
    initializeDiskChart() {
        const canvas = document.getElementById('disk-chart');
        if (!canvas) {
            console.warn('❌ Disk chart canvas not found - element ID: disk-chart');
            return;
        }
        console.log('✅ Found Disk chart canvas element');

        try {
            // Destroy existing chart if it exists to prevent Canvas reuse error
            if (this.charts.disk) {
                this.charts.disk.destroy();
                this.charts.disk = null;
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.disk = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Used', 'Available'],
                    datasets: [{
                        data: [0, 100],
                        backgroundColor: [this.colors.disk, 'rgba(255, 255, 255, 0.1)'],
                        borderWidth: 0,
                        cutout: '70%'
                    }]
                },
                options: {
                    ...this.defaultChartOptions,
                    plugins: {
                        ...this.defaultChartOptions.plugins,
                        tooltip: {
                            ...this.defaultChartOptions.plugins.tooltip,
                            callbacks: {
                                label: function(context) {
                                    return context.label + ': ' + context.parsed + '%';
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Disk chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Disk chart:', error);
        }
    }

    /**
     * Initialize Disk I/O Speed Chart (NEW)
     */
    initializeDiskIOChart() {
        const canvas = document.getElementById('disk-io-chart');
        if (!canvas) {
            console.warn('❌ Disk I/O chart canvas not found - element ID: disk-io-chart');
            return;
        }
        console.log('✅ Found Disk I/O chart canvas element');

        try {
            // Destroy existing chart if it exists to prevent Canvas reuse error
            if (this.charts.diskIO) {
                this.charts.diskIO.destroy();
                this.charts.diskIO = null;
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.diskIO = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Read Speed (MB/s)',
                        data: [],
                        borderColor: this.colors.diskIO,
                        backgroundColor: 'rgba(231, 76, 60, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4
                    }, {
                        label: 'Write Speed (MB/s)',
                        data: [],
                        borderColor: this.colors.warning,
                        backgroundColor: 'rgba(255, 167, 38, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4
                    }]
                },
                options: {
                    ...this.defaultChartOptions,
                    scales: {
                        x: {
                            display: true,
                            grid: {
                                color: this.colors.grid
                            },
                            ticks: {
                                color: this.colors.textSecondary,
                                maxTicksLimit: 8
                            }
                        },
                        y: {
                            display: true,
                            beginAtZero: true,
                            grid: {
                                color: this.colors.grid
                            },
                            ticks: {
                                color: this.colors.textSecondary,
                                callback: function(value) {
                                    return value + ' MB/s';
                                }
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            display: true,
                            labels: {
                                color: this.colors.text
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Disk I/O chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Disk I/O chart:', error);
        }
    }

    /**
     * Initialize Network Chart (UPDATED FOR SPEED)
     */
    initializeNetworkChart() {
        const canvas = document.getElementById('network-chart');
        if (!canvas) {
            console.warn('❌ Network chart canvas not found - element ID: network-chart');
            return;
        }
        console.log('✅ Found Network chart canvas element');

        try {
            const ctx = canvas.getContext('2d');
            
            this.charts.network = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Upload Speed (Mbps)',
                        data: [],
                        borderColor: this.colors.warning,
                        backgroundColor: 'rgba(255, 167, 38, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4
                    }, {
                        label: 'Download Speed (Mbps)',
                        data: [],
                        borderColor: this.colors.secondary,
                        backgroundColor: 'rgba(91, 111, 238, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4
                    }]
                },
                options: {
                    ...this.defaultChartOptions,
                    scales: {
                        x: {
                            display: true,
                            grid: {
                                color: this.colors.grid
                            },
                            ticks: {
                                color: this.colors.textSecondary,
                                maxTicksLimit: 8
                            }
                        },
                        y: {
                            display: true,
                            beginAtZero: true,
                            grid: {
                                color: this.colors.grid
                            },
                            ticks: {
                                color: this.colors.textSecondary,
                                callback: function(value) {
                                    return value + ' Mbps';
                                }
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            display: true,
                            labels: {
                                color: this.colors.text
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Network chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Network chart:', error);
        }
    }

    /**
     * Initialize Trends Chart (DÜZELTİLMİŞ VERSİYON)
     */
    initializeTrendsChart() {
        const canvas = document.getElementById('trends-chart');
        if (!canvas) {
            console.warn('❌ Trends chart canvas not found - element ID: trends-chart');
            return;
        }
        console.log('✅ Found Trends chart canvas element');

        try {
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
                        pointHoverRadius: 4
                    }, {
                        label: 'Memory Usage %',
                        data: [],
                        borderColor: this.colors.memory,
                        backgroundColor: 'rgba(78, 205, 196, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4
                    }, {
                        label: 'Disk Usage %',
                        data: [],
                        borderColor: this.colors.disk,
                        backgroundColor: 'rgba(255, 167, 38, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4
                    }]
                },
                options: {
                    ...this.defaultChartOptions,
                    scales: {
                        x: {
                            display: true,
                            grid: {
                                color: this.colors.grid
                            },
                            ticks: {
                                color: this.colors.textSecondary,
                                maxTicksLimit: 10
                            }
                        },
                        y: {
                            display: true,
                            beginAtZero: true,
                            max: 100,
                            grid: {
                                color: this.colors.grid
                            },
                            ticks: {
                                color: this.colors.textSecondary,
                                callback: function(value) {
                                    return value + '%';
                                }
                            }
                        }
                    },
                    plugins: {
                        legend: {
                            display: true,
                            labels: {
                                color: this.colors.text
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
     * Update all metrics charts (SPEED SUPPORT ADDED)
     */
    updateMetrics(metrics) {
        if (!metrics) {
            console.warn('❌ No metrics data provided to updateMetrics');
            return;
        }

        if (!this.isInitialized) {
            console.warn('❌ Charts not initialized yet, cannot update metrics');
            return;
        }

        try {
            console.log('🔄 Chart Manager updating all metrics with speed data:', metrics);
            
            // Update individual charts with detailed API data format
            this.updateCPUChart({ 
                usage_percent: metrics.cpu_usage || 0,
                cores: metrics.cpu_cores || 0,
                frequency: metrics.cpu_frequency || 0
            });
            
            this.updateMemoryChart({ 
                usage_percent: metrics.memory_percent || 0,
                total: metrics.memory_total || 0,
                used: metrics.memory_used || 0
            });
            
            this.updateDiskChart({ 
                usage_percent: metrics.disk_percent || 0,
                total: metrics.disk_total || 0,
                used: metrics.disk_used || 0
            });
            
            // NEW: Update Disk I/O Speed Chart
            this.updateDiskIOChart({
                read_speed: metrics.disk_read_speed_mbps || 0,
                write_speed: metrics.disk_write_speed_mbps || 0
            });
            
            // NEW: Update Network Speed Chart (enhanced)
            this.updateNetworkChart({ 
                upload_speed: metrics.network_upload_speed_mbps || 0,
                download_speed: metrics.network_download_speed_mbps || 0
            });

            // Update trends chart with current metrics (real-time data points)
            this.updateTrendsChart(metrics);

            console.log('✅ Chart Manager: All charts updated with speed data');
        } catch (error) {
            console.error('❌ Chart Manager error updating charts:', error);
        }
    }

    /**
     * Update CPU chart
     */
    updateCPUChart(cpuData) {
        if (!this.charts.cpu || !cpuData) {
            console.warn('❌ CPU chart or data not available:', {chart: !!this.charts.cpu, data: !!cpuData});
            return;
        }

        const usage = Math.min(100, Math.max(0, cpuData.usage_percent || cpuData.cpu_usage || 0));
        const available = 100 - usage;

        console.log(`🔄 Chart Manager updating CPU chart: ${usage}%`);
        this.charts.cpu.data.datasets[0].data = [usage, available];
        this.charts.cpu.update('active'); // Force immediate update
        console.log('✅ Chart Manager: CPU chart updated');
    }

    /**
     * Update Memory chart
     */
    updateMemoryChart(memoryData) {
        if (!this.charts.memory || !memoryData) {
            console.warn('❌ Memory chart or data not available:', {chart: !!this.charts.memory, data: !!memoryData});
            return;
        }

        const usage = Math.min(100, Math.max(0, memoryData.usage_percent || memoryData.memory_usage || 0));
        const available = 100 - usage;

        console.log(`🔄 Chart Manager updating Memory chart: ${usage}%`);
        this.charts.memory.data.datasets[0].data = [usage, available];
        this.charts.memory.update('active'); // Force immediate update
        console.log('✅ Chart Manager: Memory chart updated');
    }

    /**
     * Update Disk chart
     */
    updateDiskChart(diskData) {
        if (!this.charts.disk || !diskData) {
            console.warn('❌ Disk chart or data not available:', {chart: !!this.charts.disk, data: !!diskData});
            return;
        }

        const usage = Math.min(100, Math.max(0, diskData.usage_percent || diskData.disk_usage || 0));
        const available = 100 - usage;

        console.log(`🔄 Chart Manager updating Disk chart: ${usage}%`);
        this.charts.disk.data.datasets[0].data = [usage, available];
        this.charts.disk.update('active'); // Force immediate update
        console.log('✅ Chart Manager: Disk chart updated');
    }

    /**
     * Update Disk I/O Speed chart (NEW)
     */
    updateDiskIOChart(diskIOData) {
        if (!this.charts.diskIO || !diskIOData) {
            console.warn('❌ Disk I/O chart or data not available:', {chart: !!this.charts.diskIO, data: !!diskIOData});
            return;
        }

        const timestamp = new Date().toLocaleTimeString();
        const readSpeed = diskIOData.read_speed || 0;
        const writeSpeed = diskIOData.write_speed || 0;

        // Add new data point
        this.charts.diskIO.data.labels.push(timestamp);
        this.charts.diskIO.data.datasets[0].data.push(readSpeed);
        this.charts.diskIO.data.datasets[1].data.push(writeSpeed);

        // Keep only last N data points
        const maxPoints = this.options.maxDataPoints;
        if (this.charts.diskIO.data.labels.length > maxPoints) {
            this.charts.diskIO.data.labels.shift();
            this.charts.diskIO.data.datasets[0].data.shift();
            this.charts.diskIO.data.datasets[1].data.shift();
        }

        this.charts.diskIO.update('active');
        console.log(`🔄 Chart Manager updating Disk I/O: ${readSpeed.toFixed(1)} MB/s read, ${writeSpeed.toFixed(1)} MB/s write`);
    }

    /**
     * Update Network chart (SPEED VERSION)
     */
    updateNetworkChart(networkData) {
        if (!this.charts.network || !networkData) {
            console.warn('❌ Network chart or data not available');
            return;
        }

        const timestamp = new Date().toLocaleTimeString();
        const uploadSpeed = networkData.upload_speed || 0;
        const downloadSpeed = networkData.download_speed || 0;

        // Add new data point
        this.charts.network.data.labels.push(timestamp);
        this.charts.network.data.datasets[0].data.push(uploadSpeed);
        this.charts.network.data.datasets[1].data.push(downloadSpeed);

        // Keep only last N data points
        const maxPoints = this.options.maxDataPoints;
        if (this.charts.network.data.labels.length > maxPoints) {
            this.charts.network.data.labels.shift();
            this.charts.network.data.datasets[0].data.shift();
            this.charts.network.data.datasets[1].data.shift();
        }

        this.charts.network.update('active');
        console.log(`🔄 Chart Manager updating Network speed: ${uploadSpeed.toFixed(1)} Mbps upload, ${downloadSpeed.toFixed(1)} Mbps download`);
    }

    /**
     * Update Trends chart (DÜZELTİLMİŞ VERSİYON - REAL-TIME DATA POİNTS)
     */
    updateTrendsChart(metrics) {
        if (!this.charts.trends || !metrics) {
            console.warn('❌ Trends chart or metrics not available');
            return;
        }

        const timestamp = new Date().toLocaleTimeString();
        const cpuUsage = Math.min(100, Math.max(0, (metrics.cpu_usage || 0)));
        const memoryUsage = Math.min(100, Math.max(0, (metrics.memory_percent || 0)));
        const diskUsage = Math.min(100, Math.max(0, (metrics.disk_percent || 0)));

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
        console.log(`🔄 Chart Manager updating trends: CPU=${cpuUsage.toFixed(1)}%, Memory=${memoryUsage.toFixed(1)}%, Disk=${diskUsage.toFixed(1)}%`);
    }

    /**
     * Update Trends chart with historical data (YENİ METOD)
     */
    updateTrendsChartWithHistoricalData(historicalData) {
        if (!this.charts.trends || !historicalData) {
            console.warn('❌ Trends chart or historical data not available');
            return;
        }

        try {
            console.log('📈 Updating trends chart with historical data:', historicalData);

            // Clear existing data
            this.charts.trends.data.labels = [];
            this.charts.trends.data.datasets[0].data = [];
            this.charts.trends.data.datasets[1].data = [];
            this.charts.trends.data.datasets[2].data = [];

            // Check if data is in expected format
            if (historicalData.labels && historicalData.datasets) {
                // Use API provided data format
                this.charts.trends.data.labels = [...historicalData.labels];
                
                if (historicalData.datasets.length >= 3) {
                    this.charts.trends.data.datasets[0].data = [...historicalData.datasets[0].data];
                    this.charts.trends.data.datasets[1].data = [...historicalData.datasets[1].data];
                    this.charts.trends.data.datasets[2].data = [...historicalData.datasets[2].data];
                }
            } else if (Array.isArray(historicalData)) {
                // Handle array format - convert to chart format
                historicalData.forEach(point => {
                    this.charts.trends.data.labels.push(point.timestamp || new Date(point.time).toLocaleTimeString());
                    this.charts.trends.data.datasets[0].data.push(point.cpu_usage || 0);
                    this.charts.trends.data.datasets[1].data.push(point.memory_percent || 0);
                    this.charts.trends.data.datasets[2].data.push(point.disk_percent || 0);
                });
            }

            // Update chart
            this.charts.trends.update('none'); // No animation for historical data load
            console.log('✅ Historical trends chart updated successfully');

        } catch (error) {
            console.error('❌ Error updating trends chart with historical data:', error);
        }
    }

    /**
     * Clear trends data (YENİ METOD)
     */
    clearTrendsData() {
        if (!this.charts.trends) return;

        this.charts.trends.data.labels = [];
        this.charts.trends.data.datasets[0].data = [];
        this.charts.trends.data.datasets[1].data = [];
        this.charts.trends.data.datasets[2].data = [];
        this.charts.trends.update('none');
        
        console.log('🗑️ Trends chart data cleared');
    }

    /**
     * Resize charts (YENİ METOD)
     */
    resizeCharts() {
        try {
            Object.values(this.charts).forEach(chart => {
                if (chart) {
                    chart.resize();
                }
            });
            console.log('📐 Charts resized');
        } catch (error) {
            console.error('❌ Error resizing charts:', error);
        }
    }

    /**
     * Get statistics (YENİ METOD)
     */
    getStats() {
        return {
            isInitialized: this.isInitialized,
            chartsCount: Object.values(this.charts).filter(chart => chart !== null).length,
            maxDataPoints: this.options.maxDataPoints,
            updateInterval: this.options.updateInterval,
            theme: this.options.theme
        };
    }

    /**
     * Destroy all charts (cleanup) (DISK I/O INCLUDED)
     */
    destroyCharts() {
        Object.keys(this.charts).forEach(key => {
            if (this.charts[key]) {
                this.charts[key].destroy();
                this.charts[key] = null;
            }
        });
        console.log('🗑️ All charts destroyed (including Disk I/O)');
    }
}

// Make ChartManager globally available
window.ChartManager = ChartManager;