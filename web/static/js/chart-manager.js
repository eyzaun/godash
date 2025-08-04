/**
 * Enhanced Chart Manager for GoDash Dashboard
 * Manages all Chart.js instances with separate Speed Trends section
 */

class ChartManager {
    constructor(options = {}) {
        console.log('🎯 Enhanced ChartManager constructor started');
        
        // Check if Chart.js is available
        if (typeof Chart === 'undefined') {
            throw new Error('Chart.js is not loaded. Please ensure Chart.js is included before ChartManager.');
        }

        this.options = {
            maxDataPoints: 50,
            animationDuration: 300,   // Smoother animation
            updateInterval: 1000,     // 1 second updates
            theme: 'dark',
            ...options
        };

        // Chart instances (REORGANIZED)
        this.charts = {
            // Main metric donut charts
            cpu: null,
            memory: null,
            disk: null,
            network: null,        // Now donut chart for total bytes
            temperature: null,    // NEW
            process: null,        // NEW
            diskStorage: null,    // NEW: Traditional disk storage chart
            
            // Trends charts
            trends: null,         // Main historical trends
            diskIOSpeed: null,    // NEW: Separate disk I/O speed chart
            networkSpeed: null,   // NEW: Separate network speed chart
            
            // Dynamic partition charts
            partitions: new Map(), // NEW: Dynamic partition charts
        };

        // Chart data storage (REORGANIZED)
        this.chartData = {
            cpu: [],
            memory: [],
            disk: [],
            network: [],
            temperature: [],
            process: [],
            trends: {
                cpu: [],
                memory: [],
                disk: [],
                labels: []
            },
            speeds: {
                diskIO: {
                    read: [],
                    write: [],
                    labels: []
                },
                network: {
                    upload: [],
                    download: [],
                    labels: []
                }
            }
        };

        // Moving averages for Quick Stats
        this.movingAverages = {
            diskIO: [],
            network: [],
            maxSamples: 60 // 1 minute of data
        };

        // State flags
        this.isLoadingHistorical = false;
        this.isInitialized = false;

        // Enhanced color schemes
        this.colors = this.getColorScheme();

        // Initialize Chart.js defaults
        this.setupChartDefaults();

        // Initialize charts when DOM is ready
        this.initializeWhenReady();
    }

    /**
     * Get enhanced color scheme
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
            network: '#ab47bc',
            temperature: '#e74c3c',
            process: '#9c27b0',
            // Speed chart colors
            diskRead: '#e74c3c',
            diskWrite: '#ffa726',
            networkUpload: '#ffa726',
            networkDownload: '#5b6fee',
            background: 'rgba(0, 212, 255, 0.1)',
            border: 'rgba(0, 212, 255, 0.8)',
            text: '#ffffff',
            textSecondary: '#b0b0b0',
            grid: 'rgba(255, 255, 255, 0.1)'
        };

        return colors;
    }

    /**
     * Setup Chart.js global defaults (ENHANCED)
     */
    setupChartDefaults() {
        try {
            console.log('Setting up enhanced Chart.js defaults...');
            
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
            
            console.log('✅ Enhanced Chart.js defaults configured');
        } catch (error) {
            console.error('❌ Error setting up Chart.js defaults:', error);
        }
    }

    /**
     * Initialize charts when DOM elements are available
     */
    initializeWhenReady() {
        if (document.readyState === 'loading') {
            console.log('📝 DOM not ready, waiting for DOMContentLoaded...');
            document.addEventListener('DOMContentLoaded', () => {
                console.log('📝 DOMContentLoaded fired, initializing enhanced charts...');
                this.initializeCharts();
            });
        } else {
            console.log('📝 DOM ready, initializing enhanced charts immediately...');
            setTimeout(() => this.initializeCharts(), 100);
        }
        
        console.log('📊 Enhanced Chart Manager initialized');
    }

    /**
     * Initialize all charts (ENHANCED)
     */
    initializeCharts() {
        try {
            console.log('🎯 Initializing all enhanced charts...');
            
            // Check canvas elements
            const canvasIds = [
                'cpu-chart', 'memory-chart', 'disk-chart', 'network-chart', 
                'temperature-chart', 'process-chart', 'trends-chart',
                'disk-io-speed-chart', 'network-speed-chart'
            ];
            
            console.log('🔍 Checking for canvas elements...');
            const missingElements = [];
            const foundElements = [];
            canvasIds.forEach(id => {
                const element = document.getElementById(id);
                if (!element) {
                    missingElements.push(id);
                } else {
                    foundElements.push(id);
                    console.log(`✅ Found canvas element: ${id}`);
                }
            });
            
            console.log('📊 Found canvas elements:', foundElements);
            if (missingElements.length > 0) {
                console.warn('⚠️ Missing canvas elements:', missingElements);
                console.log('🔍 Available elements in DOM:', 
                    Array.from(document.querySelectorAll('canvas')).map(c => c.id || c.className));
            }
            
            // Initialize main metric charts (donut charts)
            console.log('🎯 Initializing main metric charts...');
            this.initializeCPUChart();
            this.initializeMemoryChart();
            this.initializeDiskChart();
            this.initializeNetworkChart();      // Now donut chart
            this.initializeTemperatureChart();  // NEW
            this.initializeProcessChart();      // NEW
            
            // Initialize trends charts
            console.log('📈 Initializing trends charts...');
            this.initializeTrendsChart();
            this.initializeDiskIOSpeedChart();  // NEW: Separate speed chart
            this.initializeNetworkSpeedChart(); // NEW: Separate speed chart
            
            this.isInitialized = true;
            console.log('✅ All enhanced charts initialized successfully');
        } catch (error) {
            console.error('❌ Error initializing enhanced charts:', error);
        }
    }

    /**
     * Initialize CPU Chart (UNCHANGED)
     */
    initializeCPUChart() {
        const canvas = document.getElementById('cpu-chart');
        if (!canvas) {
            console.warn('❌ CPU chart canvas not found');
            return;
        }

        try {
            if (this.charts.cpu) {
                this.charts.cpu.destroy();
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
     * Initialize Memory Chart (UNCHANGED)
     */
    initializeMemoryChart() {
        const canvas = document.getElementById('memory-chart');
        if (!canvas) return;

        try {
            if (this.charts.memory) {
                this.charts.memory.destroy();
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
     * Initialize Disk Chart (UNCHANGED)
     */
    initializeDiskChart() {
        const canvas = document.getElementById('disk-chart');
        if (!canvas) return;

        try {
            if (this.charts.disk) {
                this.charts.disk.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.disk = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Read Speed', 'Write Speed'],
                    datasets: [{
                        data: [50, 50],
                        backgroundColor: [this.colors.primary, this.colors.disk],
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
                                    return context.label + ': ' + context.parsed.toFixed(1) + '%';
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Disk chart initialized with speed display');
        } catch (error) {
            console.error('❌ Error initializing Disk chart:', error);
        }
    }

    /**
     * Initialize Network Chart (CHANGED TO DONUT)
     */
    initializeNetworkChart() {
        const canvas = document.getElementById('network-chart');
        if (!canvas) return;

        try {
            if (this.charts.network) {
                this.charts.network.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.network = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Sent', 'Received'],
                    datasets: [{
                        data: [50, 50],
                        backgroundColor: [this.colors.warning, this.colors.network],
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
                                    return context.label + ': ' + DashboardUtils.formatBytes(context.parsed);
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Network chart initialized (donut)');
        } catch (error) {
            console.error('❌ Error initializing Network chart:', error);
        }
    }

    /**
     * Initialize Temperature Chart (NEW)
     */
    initializeTemperatureChart() {
        const canvas = document.getElementById('temperature-chart');
        if (!canvas) return;

        try {
            if (this.charts.temperature) {
                this.charts.temperature.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.temperature = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Current', 'Safe Range'],
                    datasets: [{
                        data: [25, 60], // 25°C current, 85°C max safe
                        backgroundColor: [this.colors.temperature, 'rgba(255, 255, 255, 0.1)'],
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
                                    return context.label + ': ' + context.parsed + '°C';
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Temperature chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Temperature chart:', error);
        }
    }

    /**
     * Initialize Process Chart (NEW)
     */
    initializeProcessChart() {
        const canvas = document.getElementById('process-chart');
        if (!canvas) return;

        try {
            if (this.charts.process) {
                this.charts.process.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.process = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Running', 'Sleeping', 'Zombie'],
                    datasets: [{
                        data: [10, 80, 0],
                        backgroundColor: [this.colors.success, this.colors.process, this.colors.danger],
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
                                    return context.label + ': ' + context.parsed;
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Process chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Process chart:', error);
        }
    }

    /**
     * Initialize Disk Storage Chart (Traditional Usage)
     */
    initializeDiskStorageChart() {
        const canvas = document.getElementById('disk-storage-chart');
        if (!canvas) return;

        try {
            if (this.charts.diskStorage) {
                this.charts.diskStorage.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.diskStorage = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Used', 'Free'],
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
            
            console.log('✅ Disk Storage chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Disk Storage chart:', error);
        }
    }

    /**
     * Initialize Disk I/O Speed Chart (NEW SEPARATE CHART)
     */
    initializeDiskIOSpeedChart() {
        const canvas = document.getElementById('disk-io-speed-chart');
        if (!canvas) return;

        try {
            if (this.charts.diskIOSpeed) {
                console.log('🗑️ Destroying existing diskIOSpeed chart');
                this.charts.diskIOSpeed.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.diskIOSpeed = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Read Speed (MB/s)',
                        data: [],
                        borderColor: this.colors.diskRead,
                        backgroundColor: 'rgba(231, 76, 60, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
                    }, {
                        label: 'Write Speed (MB/s)',
                        data: [],
                        borderColor: this.colors.diskWrite,
                        backgroundColor: 'rgba(255, 167, 38, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
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
                            display: false // Using custom legend
                        },
                        tooltip: {
                            ...this.defaultChartOptions.plugins.tooltip,
                            callbacks: {
                                label: function(context) {
                                    return context.dataset.label + ': ' + context.parsed.y.toFixed(1) + ' MB/s';
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Disk I/O Speed chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Disk I/O Speed chart:', error);
        }
    }

    /**
     * Initialize Network Speed Chart (NEW SEPARATE CHART)
     */
    initializeNetworkSpeedChart() {
        const canvas = document.getElementById('network-speed-chart');
        if (!canvas) return;

        try {
            if (this.charts.networkSpeed) {
                this.charts.networkSpeed.destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts.networkSpeed = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Upload Speed (Mbps)',
                        data: [],
                        borderColor: this.colors.networkUpload,
                        backgroundColor: 'rgba(255, 167, 38, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
                    }, {
                        label: 'Download Speed (Mbps)',
                        data: [],
                        borderColor: this.colors.networkDownload,
                        backgroundColor: 'rgba(91, 111, 238, 0.1)',
                        fill: false,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 4,
                        borderWidth: 2
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
                            display: false // Using custom legend
                        },
                        tooltip: {
                            ...this.defaultChartOptions.plugins.tooltip,
                            callbacks: {
                                label: function(context) {
                                    return context.dataset.label + ': ' + context.parsed.y.toFixed(1) + ' Mbps';
                                }
                            }
                        }
                    }
                }
            });
            
            console.log('✅ Network Speed chart initialized');
        } catch (error) {
            console.error('❌ Error initializing Network Speed chart:', error);
        }
    }

    /**
     * Initialize Trends Chart (UNCHANGED)
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
                                color: this.colors.text,
                                usePointStyle: true
                            }
                        },
                        tooltip: {
                            ...this.defaultChartOptions.plugins.tooltip,
                            callbacks: {
                                label: function(context) {
                                    return context.dataset.label + ': ' + context.parsed.y.toFixed(1) + '%';
                                }
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
     * Update all metrics charts (ENHANCED)
     */
    updateMetrics(metrics) {
        if (!metrics || !this.isInitialized) {
            console.warn('❌ No metrics data or charts not initialized');
            return;
        }

        try {
            console.log('🔄 Enhanced Chart Manager updating all metrics');
            
            // Update donut charts
            this.updateCPUChart({ 
                usage_percent: metrics.cpu_usage || 0,
                cores: metrics.cpu_cores || 0,
                frequency: metrics.cpu_frequency || 0,
                load_avg: metrics.cpu_load_avg || [0, 0, 0]
            });
            
            this.updateMemoryChart({ 
                usage_percent: metrics.memory_percent || 0,
                total: metrics.memory_total || 0,
                used: metrics.memory_used || 0,
                cached: metrics.memory_cached || 0
            });
            
            this.updateDiskChart({ 
                usage_percent: metrics.disk_percent || 0,
                total: metrics.disk_total || 0,
                used: metrics.disk_used || 0,
                free: metrics.disk_free || 0,
                read_speed: metrics.disk_read_speed_mbps || 0,
                write_speed: metrics.disk_write_speed_mbps || 0
            });
            
            this.updateNetworkChart({
                sent: metrics.network_sent || 0,
                received: metrics.network_received || 0
            });

            // Update new charts
            this.updateTemperatureChart({
                current: metrics.simulated_temperature || 45, // Use realistic simulated temperature
                max_safe: 85
            });

            // Update Process chart with real data
            if (metrics.processes) {
                this.updateProcessChart({
                    running: metrics.processes.running_processes || 0,
                    sleeping: metrics.processes.stopped_processes || 0,
                    zombie: metrics.processes.zombie_processes || 0
                });
            } else {
                // Fallback to mock data if no process data available
                this.updateProcessChart({
                    running: Math.floor(Math.random() * 20) + 10,
                    sleeping: Math.floor(Math.random() * 100) + 50,
                    other: Math.floor(Math.random() * 10) + 5
                });
            }
            
            // Update disk storage chart (traditional usage)
            this.updateDiskStorageChart({
                usage_percent: metrics.disk_percent || 0,
                total: metrics.disk_total || 0,
                used: metrics.disk_used || 0,
                free: metrics.disk_free || 0
            });
            
            // Update speed charts
            this.updateDiskIOSpeedChart({
                read_speed: metrics.disk_read_speed_mbps || 0,
                write_speed: metrics.disk_write_speed_mbps || 0
            });
            
            this.updateNetworkSpeedChart({ 
                upload_speed: metrics.network_upload_speed_mbps || 0,
                download_speed: metrics.network_download_speed_mbps || 0
            });

            // Update trends chart with current metrics
            this.updateTrendsChart(metrics);

            console.log('✅ Enhanced Chart Manager: All charts updated');
        } catch (error) {
            console.error('❌ Enhanced Chart Manager error updating charts:', error);
        }
    }

    /**
     * Update CPU chart (UNCHANGED)
     */
    updateCPUChart(cpuData) {
        if (!this.charts.cpu || !cpuData) return;

        const usage = Math.min(100, Math.max(0, cpuData.usage_percent || 0));
        const available = 100 - usage;

        this.charts.cpu.data.datasets[0].data = [usage, available];
        this.charts.cpu.update('active');
    }

    /**
     * Update Memory chart (UNCHANGED)
     */
    updateMemoryChart(memoryData) {
        if (!this.charts.memory || !memoryData) return;

        const usage = Math.min(100, Math.max(0, memoryData.usage_percent || 0));
        const available = 100 - usage;

        this.charts.memory.data.datasets[0].data = [usage, available];
        this.charts.memory.update('active');
    }

    /**
     * Update Disk chart (CHANGED TO SHOW SPEED)
     */
    updateDiskChart(diskData) {
        if (!this.charts.disk || !diskData) return;

        // Prioritize speed display if available
        if (diskData.read_speed !== undefined && diskData.write_speed !== undefined) {
            const readSpeed = diskData.read_speed || 0;
            const writeSpeed = diskData.write_speed || 0;
            const totalSpeed = readSpeed + writeSpeed;
            
            // Show speed distribution (read vs write)
            if (totalSpeed > 0) {
                const readPercent = (readSpeed / totalSpeed) * 100;
                const writePercent = (writeSpeed / totalSpeed) * 100;
                this.charts.disk.data.datasets[0].data = [readPercent, writePercent];
            } else {
                // No activity - show 50/50 split
                this.charts.disk.data.datasets[0].data = [50, 50];
            }
        } else {
            // Fallback to usage percentage
            const usage = Math.min(100, Math.max(0, diskData.usage_percent || 0));
            const available = 100 - usage;
            this.charts.disk.data.datasets[0].data = [usage, available];
        }
        
        this.charts.disk.update('active');
    }

    /**
     * Update Disk Storage chart (Traditional Usage)
     */
    updateDiskStorageChart(diskData) {
        if (!this.charts.diskStorage || !diskData) return;

        const usage = Math.min(100, Math.max(0, diskData.usage_percent || 0));
        const available = 100 - usage;

        this.charts.diskStorage.data.datasets[0].data = [usage, available];
        this.charts.diskStorage.update('active');
    }

    /**
     * Update Network chart (CHANGED TO DONUT)
     */
    updateNetworkChart(networkData) {
        if (!this.charts.network || !networkData) return;

        const sent = networkData.sent || 0;
        const received = networkData.received || 0;
        const total = sent + received;

        if (total > 0) {
            const sentPercent = (sent / total) * 100;
            const receivedPercent = (received / total) * 100;
            this.charts.network.data.datasets[0].data = [sentPercent, receivedPercent];
        } else {
            this.charts.network.data.datasets[0].data = [50, 50];
        }

        this.charts.network.update('active');
    }

    /**
     * Update Temperature chart (NEW)
     */
    updateTemperatureChart(tempData) {
        if (!this.charts.temperature || !tempData) return;

        const current = Math.min(85, Math.max(0, tempData.current || 0));
        const remaining = 85 - current;

        this.charts.temperature.data.datasets[0].data = [current, remaining];
        this.charts.temperature.update('active');
    }

    /**
     * Update Process chart with real data
     */
    updateProcessChart(processData) {
        if (!this.charts.process || !processData) return;

        const running = processData.running || 0;
        const sleeping = processData.sleeping || 0;
        const zombie = processData.zombie || 0;

        // Chart has 3 segments: Running, Sleeping, Zombie
        this.charts.process.data.datasets[0].data = [running, sleeping, zombie];
        this.charts.process.update('active');
    }

    /**
     * Update Disk I/O Speed chart (NEW)
     */
    updateDiskIOSpeedChart(speedData) {
        if (!this.charts.diskIOSpeed || !speedData) return;

        const timestamp = new Date().toLocaleTimeString();
        const readSpeed = speedData.read_speed || 0;
        const writeSpeed = speedData.write_speed || 0;

        // Add new data point
        this.charts.diskIOSpeed.data.labels.push(timestamp);
        this.charts.diskIOSpeed.data.datasets[0].data.push(readSpeed);
        this.charts.diskIOSpeed.data.datasets[1].data.push(writeSpeed);

        // Keep only last N data points
        const maxPoints = this.options.maxDataPoints;
        if (this.charts.diskIOSpeed.data.labels.length > maxPoints) {
            this.charts.diskIOSpeed.data.labels.shift();
            this.charts.diskIOSpeed.data.datasets[0].data.shift();
            this.charts.diskIOSpeed.data.datasets[1].data.shift();
        }

        // Update moving averages
        this.movingAverages.diskIO.push(readSpeed + writeSpeed);
        if (this.movingAverages.diskIO.length > this.movingAverages.maxSamples) {
            this.movingAverages.diskIO.shift();
        }

        this.charts.diskIOSpeed.update('active');
    }

    /**
     * Update Network Speed chart (NEW)
     */
    updateNetworkSpeedChart(speedData) {
        if (!this.charts.networkSpeed || !speedData) return;

        const timestamp = new Date().toLocaleTimeString();
        const uploadSpeed = speedData.upload_speed || 0;
        const downloadSpeed = speedData.download_speed || 0;

        // Add new data point
        this.charts.networkSpeed.data.labels.push(timestamp);
        this.charts.networkSpeed.data.datasets[0].data.push(uploadSpeed);
        this.charts.networkSpeed.data.datasets[1].data.push(downloadSpeed);

        // Keep only last N data points
        const maxPoints = this.options.maxDataPoints;
        if (this.charts.networkSpeed.data.labels.length > maxPoints) {
            this.charts.networkSpeed.data.labels.shift();
            this.charts.networkSpeed.data.datasets[0].data.shift();
            this.charts.networkSpeed.data.datasets[1].data.shift();
        }

        // Update moving averages
        this.movingAverages.network.push(uploadSpeed + downloadSpeed);
        if (this.movingAverages.network.length > this.movingAverages.maxSamples) {
            this.movingAverages.network.shift();
        }

        this.charts.networkSpeed.update('active');
    }

    /**
     * Update Trends chart (UNCHANGED)
     */
    updateTrendsChart(metrics) {
        if (!this.charts.trends || !metrics) return;

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
    }

    /**
     * Update Trends chart with historical data (UNCHANGED)
     */
    updateTrendsChartWithHistoricalData(historicalData) {
        if (!this.charts.trends || !historicalData) return;

        try {
            console.log('📈 Updating trends chart with historical data');

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
            console.log('✅ Historical trends chart updated successfully');

        } catch (error) {
            console.error('❌ Error updating trends chart with historical data:', error);
        }
    }

    /**
     * Get moving averages for Quick Stats (NEW)
     */
    getMovingAverages() {
        const calculateAverage = (arr) => {
            if (arr.length === 0) return 0;
            return arr.reduce((a, b) => a + b, 0) / arr.length;
        };

        return {
            diskIO: calculateAverage(this.movingAverages.diskIO),
            network: calculateAverage(this.movingAverages.network)
        };
    }

    /**
     * Resize charts (ENHANCED)
     */
    resizeCharts() {
        try {
            Object.values(this.charts).forEach(chart => {
                if (chart) {
                    chart.resize();
                }
            });
            console.log('📐 Enhanced charts resized');
        } catch (error) {
            console.error('❌ Error resizing enhanced charts:', error);
        }
    }

    /**
     * Get statistics (ENHANCED)
     */
    getStats() {
        const activeCharts = Object.values(this.charts).filter(chart => chart !== null).length;
        
        return {
            isInitialized: this.isInitialized,
            chartsCount: activeCharts,
            maxDataPoints: this.options.maxDataPoints,
            updateInterval: this.options.updateInterval,
            theme: this.options.theme,
            movingAverages: this.getMovingAverages(),
            chartTypes: {
                donut: ['cpu', 'memory', 'disk', 'network', 'temperature', 'process'].length,
                line: ['trends', 'diskIOSpeed', 'networkSpeed'].length
            }
        };
    }

    /**
     * Destroy all charts (ENHANCED)
     */
    destroyCharts() {
        // Destroy standard charts
        Object.keys(this.charts).forEach(key => {
            if (this.charts[key] && typeof this.charts[key].destroy === 'function') {
                this.charts[key].destroy();
                this.charts[key] = null;
            }
        });
        
        // Destroy partition charts
        this.destroyPartitionCharts();
        
        console.log('🗑️ All enhanced charts destroyed');
    }

    /**
     * Initialize dynamic disk charts for multiple disks
     */
    initializeDynamicDiskCharts(diskPartitions) {
        if (!diskPartitions || diskPartitions.length === 0) return;

        diskPartitions.forEach((partition, index) => {
            if (!partition.device) return;

            // Initialize storage chart for this disk
            this.initializeDiskStorageChart(index);
            
            // Initialize I/O speed chart for this disk
            this.initializePartitionDiskIOChart(index);
        });
    }

    /**
     * Initialize individual disk storage chart
     */
    initializeDiskStorageChart(index) {
        const canvas = document.getElementById(`disk-storage-chart-${index}`);
        if (!canvas) return;

        try {
            if (this.charts[`diskStorage${index}`]) {
                this.charts[`diskStorage${index}`].destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts[`diskStorage${index}`] = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Used', 'Free'],
                    datasets: [{
                        data: [50, 50],
                        backgroundColor: [this.colors.primary, this.colors.secondary],
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
                                    const label = context.label;
                                    const value = (context.parsed / 1024 / 1024 / 1024).toFixed(1);
                                    return `${label}: ${value} GB`;
                                }
                            }
                        }
                    }
                }
            });
            
            console.log(`✅ Disk storage chart ${index} initialized`);
        } catch (error) {
            console.error(`❌ Error initializing disk storage chart ${index}:`, error);
        }
    }

    /**
     * Initialize individual disk I/O speed chart for specific partition
     */
    initializePartitionDiskIOChart(index) {
        const canvas = document.getElementById(`disk-io-speed-chart-${index}`);
        if (!canvas) return;

        try {
            if (this.charts[`diskIO${index}`]) {
                this.charts[`diskIO${index}`].destroy();
            }
            
            const ctx = canvas.getContext('2d');
            
            this.charts[`diskIO${index}`] = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [
                        {
                            label: 'Read Speed',
                            data: [],
                            borderColor: this.colors.primary,
                            backgroundColor: this.getTransparentColor(this.colors.primary, 0.1),
                            tension: 0.4,
                            fill: true
                        },
                        {
                            label: 'Write Speed',
                            data: [],
                            borderColor: this.colors.secondary,
                            backgroundColor: this.getTransparentColor(this.colors.secondary, 0.1),
                            tension: 0.4,
                            fill: true
                        }
                    ]
                },
                options: {
                    ...this.defaultChartOptions,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Speed (MB/s)'
                            }
                        }
                    }
                }
            });
            
            console.log(`✅ Disk I/O speed chart ${index} initialized`);
        } catch (error) {
            console.error(`❌ Error initializing disk I/O speed chart ${index}:`, error);
        }
    }

    /**
     * Initialize dynamic disk partition charts
     */
    initializeDynamicDiskCharts(diskPartitions) {
        if (!diskPartitions || diskPartitions.length === 0) return;

        console.log('🔄 Initializing dynamic disk partition charts:', diskPartitions);

        diskPartitions.forEach((partition, index) => {
            if (!partition.device) return;
            this.initializePartitionChart(index, partition);
        });
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
                    ...this.defaultChartOptions,
                    plugins: {
                        ...this.defaultChartOptions.plugins,
                        legend: {
                            display: false
                        },
                        tooltip: {
                            ...this.defaultChartOptions.plugins.tooltip,
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
}

// Make enhanced ChartManager globally available
window.ChartManager = ChartManager;