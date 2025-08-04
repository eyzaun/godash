/**
 * Enhanced GoDash Dashboard Main Application
 * Orchestrates WebSocket connections, chart management, and UI updates with enhanced features
 */

class DashboardApp {
    constructor(options = {}) {
        this.options = {
            updateInterval: 1000, // 1 second for smooth real-time updates
            reconnectAttempts: 10,
            chartUpdateAnimation: true,
            alertTimeout: 5000,
            debug: window.location.hostname === 'localhost',
            ...options
        };

        // Application state
        this.isInitialized = false;
        this.isPaused = false;
        this.lastUpdateTime = null;
        this.connectionAttempts = 0;
        this.metricsBuffer = [];
        this.hasApiAverages = false;
        
        // SIMPLE moving averages for Quick Stats - BASIC APPROACH
        this.movingAverages = {
            cpu: [],
            memory: [],
            diskIO: [],
            network: [],
            maxSamples: 30 // 15 seconds at 500ms intervals - keep it simple
        };

        // Simple exponential smoothing averages
        this.simpleAverages = {
            cpu: { current: 0, count: 0 },
            memory: { current: 0, count: 0 },
            diskIO: { current: 0, count: 0 },
            network: { current: 0, count: 0 }
        };

        // Component instances
        this.websocket = null;
        this.chartManager = null;

        // Enhanced DOM element caches
        this.elements = {};
        
        // Data storage
        this.currentMetrics = null;
        this.systemInfo = null;
        this.alerts = [];
        this.topProcesses = [];
        
        // Simulated temperature state (more realistic)
        this.simulatedTemperature = {
            current: 45, // Starting temperature
            target: 45,  // Target temperature (changes gradually)
            lastUpdate: Date.now()
        };

        // Statistics
        this.stats = {
            totalUpdates: 0,
            errors: 0,
            connectionTime: null,
            lastError: null,
            enhancedFeatures: true
        };

        // Event handlers registry
        this.eventHandlers = new Map();

        this.log('🎯 Enhanced Dashboard app initialized:', this.options);
    }

    /**
     * Initialize Quick Stats with default values
     */
    initializeQuickStats() {
        try {
            // Check if elements are available
            if (!this.elements || !this.elements.avgCpu) {
                this.log('⚠️ Elements not cached yet, skipping Quick Stats initialization');
                return;
            }

            this.updateElementText(this.elements.avgCpu, '0.0%');
            this.updateElementText(this.elements.avgMemory, '0.0%');
            this.updateElementText(this.elements.avgDiskIO, '0.0 MB/s');
            this.updateElementText(this.elements.avgNetwork, '0.0 Mbps');
            this.updateElementText(this.elements.totalHosts, '1');
            this.updateElementText(this.elements.totalMetrics, '0');
            this.log('🎯 Quick Stats initialized with default values');
        } catch (error) {
            this.log('❌ Error initializing Quick Stats:', error);
        }
    }

    /**
     * Initialize the enhanced dashboard application
     */
    async initialize() {
        if (this.isInitialized) {
            this.log('⚠️ Dashboard already initialized');
            return;
        }

        try {
            this.log('🚀 Initializing Enhanced GoDash Dashboard...');

            // Cache DOM elements
            this.cacheElements();

            // Initialize Quick Stats with default values (after elements are cached)
            this.initializeQuickStats();

            // Initialize chart manager
            await this.initializeChartManager();

            // Initialize WebSocket connection
            this.initializeWebSocket();

            // Setup enhanced event listeners
            this.setupEventListeners();

            // Setup enhanced UI components
            this.setupUIComponents();

            // Load initial data
            await this.loadInitialData();

            // Hide loading screen and show dashboard
            this.showDashboard();

            this.isInitialized = true;
            this.stats.connectionTime = new Date();
            
            this.log('✅ Enhanced Dashboard initialized successfully');
            this.updateConnectionStatus('connected', 'Connected');

        } catch (error) {
            this.log('❌ Error initializing enhanced dashboard:', error);
            this.handleError(error);
            this.showError('Failed to initialize enhanced dashboard');
            this.showDashboard();
        }
    }

    /**
     * Cache enhanced DOM elements
     */
    cacheElements() {
        this.elements = {
            // Main containers
            loadingScreen: document.getElementById('loading-screen'),
            dashboard: document.getElementById('dashboard'),
            alertBanner: document.getElementById('alert-banner'),
            alertMessage: document.getElementById('alert-message'),
            closeAlert: document.getElementById('close-alert'),

            // Connection status
            wsStatus: document.getElementById('ws-status'),
            clientCount: document.getElementById('client-count'),

            // Enhanced metric values
            cpuValue: document.getElementById('cpu-value'),
            memoryValue: document.getElementById('memory-value'),
            diskValue: document.getElementById('disk-value'),
            networkValue: document.getElementById('network-value'),
            temperatureValue: document.getElementById('temperature-value'),
            processValue: document.getElementById('process-value'),

            // Enhanced metric details
            cpuCores: document.getElementById('cpu-cores'),
            cpuFrequency: document.getElementById('cpu-frequency'),
            cpuLoadAvg: document.getElementById('cpu-load-avg'),
            memoryUsed: document.getElementById('memory-used'),
            memoryTotal: document.getElementById('memory-total'),
            memoryCached: document.getElementById('memory-cached'),
            diskUsagePercent: document.getElementById('disk-usage-percent'),
            diskUsed: document.getElementById('disk-used'),
            diskTotal: document.getElementById('disk-total'),
            diskFree: document.getElementById('disk-free'),
            diskStorageValue: document.getElementById('disk-storage-value'),
            diskStorageUsed: document.getElementById('disk-storage-used'),
            diskStorageTotal: document.getElementById('disk-storage-total'),
            diskStorageFree: document.getElementById('disk-storage-free'),
            networkSent: document.getElementById('network-sent'),
            networkReceived: document.getElementById('network-received'),
            networkErrors: document.getElementById('network-errors'),
            
            // NEW: Temperature elements
            cpuTemperature: document.getElementById('cpu-temperature'),
            temperatureStatus: document.getElementById('temperature-status'),
            temperatureMax: document.getElementById('temperature-max'),

            // NEW: Process elements
            processRunning: document.getElementById('process-running'),
            processSleeping: document.getElementById('process-sleeping'),
            processZombie: document.getElementById('process-zombie'),

            // NEW: Speed monitoring elements
            diskReadSpeed: document.getElementById('disk-read-speed'),
            diskWriteSpeed: document.getElementById('disk-write-speed'),
            networkUploadSpeed: document.getElementById('network-upload-speed'),
            networkDownloadSpeed: document.getElementById('network-download-speed'),

            // Enhanced system information
            systemHostname: document.getElementById('system-hostname'),
            systemPlatform: document.getElementById('system-platform'),
            systemArch: document.getElementById('system-arch'),
            systemUptime: document.getElementById('system-uptime'),
            loggedUsers: document.getElementById('logged-users'),
            lastUpdate: document.getElementById('last-update'),

            // Enhanced quick stats
            totalHosts: document.getElementById('total-hosts'),
            totalMetrics: document.getElementById('total-metrics'),
            avgCpu: document.getElementById('avg-cpu'),
            avgMemory: document.getElementById('avg-memory'),
            avgDiskIO: document.getElementById('avg-disk-io'),
            avgNetwork: document.getElementById('avg-network'),

            // System status and processes
            systemStatus: document.getElementById('system-status'),
            topProcesses: document.getElementById('top-processes'),

            // Time range buttons
            timeButtons: document.querySelectorAll('.time-btn'),

            // Mobile menu
            mobileMenuToggle: document.getElementById('mobile-menu-toggle')
        };

        this.log('📋 Enhanced DOM elements cached');
        this.logElementCheck();
    }

    /**
     * Log element check for debugging
     */
    logElementCheck() {
        const criticalElements = [
            'cpuValue', 'memoryValue', 'diskValue', 'networkValue',
            'diskReadSpeed', 'diskWriteSpeed',
            'networkUploadSpeed', 'networkDownloadSpeed', 'avgCpu', 'avgMemory'
        ];

        console.log('🔍 Enhanced Element check:');
        criticalElements.forEach(elementKey => {
            const element = this.elements[elementKey];
            console.log(`${elementKey}:`, element ? '✅' : '❌');
        });
    }

    /**
     * Initialize enhanced chart manager
     */
    async initializeChartManager() {
        try {
            console.log('🎯 Initializing Enhanced Chart Manager...');
            
            this.chartManager = new ChartManager({
                maxDataPoints: 50,
                animationDuration: this.options.chartUpdateAnimation ? 300 : 0,
                theme: 'dark'
            });

            // Wait for Chart Manager to be fully initialized
            await new Promise((resolve) => {
                const checkInitialized = () => {
                    if (this.chartManager.isInitialized) {
                        console.log('📊 Enhanced chart manager initialized successfully');
                        resolve(true);
                    } else {
                        setTimeout(checkInitialized, 100);
                    }
                };
                checkInitialized();
            });

            return true;
        } catch (error) {
            console.error('❌ Enhanced chart manager initialization failed:', error);
            this.chartManager = null;
            return false;
        }
    }

    /**
     * Initialize WebSocket connection (UNCHANGED)
     */
    initializeWebSocket() {
        console.log('🔌 Enhanced Dashboard: Initializing WebSocket connection...');
        
        try {
            this.websocket = new WebSocketClient({
                debug: this.options.debug,
                reconnectInterval: 5000,
                maxReconnectAttempts: this.options.reconnectAttempts
            });
            
        } catch (error) {
            console.error('❌ Failed to create WebSocket client:', error);
            return;
        }

        // Enhanced WebSocket event handlers
        this.websocket.on('connect', (event) => {
            this.log('🔌 WebSocket connected');
            this.connectionAttempts = 0;
            this.updateConnectionStatus('connected', 'Connected');
            this.hideAlert();
            this.websocket.subscribe(['metrics', 'system_status']);
        });

        this.websocket.on('disconnect', (event) => {
            this.log('🔌 WebSocket disconnected');
            this.updateConnectionStatus('disconnected', 'Disconnected');
            if (event.code !== 1000) {
                this.showAlert('Connection lost. Attempting to reconnect...', 'warning');
            }
        });

        this.websocket.on('reconnect', (event) => {
            this.log('🔄 WebSocket reconnected');
            this.updateConnectionStatus('connected', 'Reconnected');
            this.showAlert('Connection restored!', 'success');
            setTimeout(() => this.hideAlert(), 3000);
        });

        this.websocket.on('error', (error) => {
            this.log('❌ WebSocket error:', error);
            this.connectionAttempts++;
            
            if (this.connectionAttempts > 3) {
                this.updateConnectionStatus('error', 'Connection Error');
                this.showAlert('Unable to connect to server', 'error');
            } else {
                this.updateConnectionStatus('connecting', 'Connecting...');
            }
            
            this.handleError(error);
        });

        this.websocket.on('metrics', (data) => {
            console.log('🔥 Enhanced DETAILED METRICS DATA RECEIVED:', data);
            this.handleMetricsUpdate(data);
        });

        this.websocket.on('system_status', (data) => {
            this.handleSystemStatusUpdate(data);
        });

        this.websocket.on('pong', (data) => {
            this.log('🏓 Pong received:', data);
        });

        this.websocket.connect();
        this.log('🔌 Enhanced WebSocket connection initiated');
    }

    /**
     * Setup enhanced event listeners
     */
    setupEventListeners() {
        // Alert close button
        if (this.elements.closeAlert) {
            this.elements.closeAlert.addEventListener('click', () => {
                this.hideAlert();
            });
        }

        // Time range selector buttons
        this.elements.timeButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                const range = e.target.dataset.range;
                this.changeTimeRange(range);
            });
        });

        // Enhanced window resize handler
        window.addEventListener('resize', () => {
            if (this.chartManager) {
                this.chartManager.resizeCharts();
            }
        });

        // Enhanced keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            this.handleKeyboardShortcuts(e);
        });

        // Enhanced page visibility change
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.pauseUpdates();
            } else {
                this.resumeUpdates();
            }
        });

        this.log('👂 Enhanced event listeners setup complete');
    }

    /**
     * Setup enhanced UI components
     */
    setupUIComponents() {
        this.initializeTooltips();
        this.setupMobileMenu();
        this.setupRefreshIntervals();
        this.log('🎨 Enhanced UI components setup complete');
    }

    /**
     * Load enhanced initial data
     */
    async loadInitialData() {
        try {
            this.log('📥 Loading enhanced initial data...');

            // Load current metrics
            await this.loadCurrentMetrics();

            // Load enhanced system statistics
            await this.loadSystemStats();

            // Load enhanced system details
            await this.loadSystemDetails();

            // Load system status
            await this.loadSystemStatus();

            // Load top processes
            await this.loadTopProcesses();

            // Load historical data for trends
            await this.loadHistoricalData();

            this.log('✅ Enhanced initial data loaded successfully');
        } catch (error) {
            this.log('❌ Error loading enhanced initial data:', error);
            this.handleError(error);
        }
    }

    /**
     * Load current metrics (ENHANCED)
     */
    async loadCurrentMetrics() {
        try {
            console.log('🔄 Loading enhanced current metrics...');
            const response = await fetch('/api/v1/metrics/current');
            const result = await response.json();

            if (result.success && result.data) {
                console.log('📊 Enhanced current metrics received:', result.data);
                
                this.updateEnhancedMetricsDisplay(result.data);
                
                if (this.chartManager) {
                    console.log('📈 Calling enhanced chartManager.updateMetrics...');
                    // Use real temperature from CPU metrics instead of simulated
                    const enhancedMetrics = {
                        ...result.data,
                        temperature: result.data.cpu_temperature_c || 0
                    };
                    this.chartManager.updateMetrics(enhancedMetrics);
                    console.log('✅ Enhanced chart manager update completed');
                }
                
                this.currentMetrics = result.data;
                this.log('📊 Enhanced current metrics loaded and charts updated');
            } else {
                console.warn('❌ Invalid enhanced metrics response:', result);
            }
        } catch (error) {
            this.log('❌ Error loading enhanced current metrics:', error);
            console.error('❌ Error loading enhanced current metrics:', error);
        }
    }

    /**
     * Load enhanced system statistics
     */
    async loadSystemStats() {
        try {
            const response = await fetch('/api/v1/system/stats');
            const result = await response.json();

            if (result.success && result.data) {
                this.updateEnhancedStatsDisplay(result.data);
                this.log('📈 Enhanced system stats loaded');
            }
        } catch (error) {
            this.log('❌ Error loading enhanced system stats:', error);
        }
    }

    /**
     * Load enhanced system details
     */
    async loadSystemDetails() {
        try {
            // Get current client count from WebSocket connection
            const currentClientCount = this.getCurrentClientCount();
            
            // Enhanced system information using real client count
            const systemInfo = {
                platform: navigator.platform || 'Unknown',
                architecture: navigator.userAgent.includes('x64') ? 'x64' : 'x86',
                uptime: 'Calculating...',
                loggedUsers: currentClientCount
            };

            this.updateEnhancedSystemDetailsDisplay(systemInfo);
            this.log('🖥️ Enhanced system details loaded');
        } catch (error) {
            this.log('❌ Error loading enhanced system details:', error);
        }
    }

    /**
     * Get current client count from UI or WebSocket
     */
    getCurrentClientCount() {
        try {
            // Try to get from the client count element first
            if (this.elements.clientCount && this.elements.clientCount.textContent) {
                const count = parseInt(this.elements.clientCount.textContent);
                if (!isNaN(count)) {
                    return count;
                }
            }
            
            // Fallback to WebSocket client count if available
            if (this.websocket && this.websocket.clientCount !== undefined) {
                return this.websocket.clientCount;
            }
            
            // Default fallback
            return 1;
        } catch (error) {
            this.log('❌ Error getting client count:', error);
            return 1;
        }
    }

    /**
     * Load system status (UNCHANGED)
     */
    async loadSystemStatus() {
        try {
            const response = await fetch('/api/v1/system/status');
            const result = await response.json();

            if (result.success && result.data) {
                this.updateSystemStatusDisplay(result.data);
                this.log('🟢 System status loaded');
            }
        } catch (error) {
            this.log('❌ Error loading system status:', error);
        }
    }

    /**
     * Load top processes (NEW)
     */
    async loadTopProcesses() {
        try {
            // Mock top processes data (you can add real API endpoint)
            const mockProcesses = [
                { name: 'chrome', cpu: Math.random() * 30 + 10 },
                { name: 'node', cpu: Math.random() * 20 + 5 },
                { name: 'vscode', cpu: Math.random() * 15 + 3 },
                { name: 'firefox', cpu: Math.random() * 25 + 8 },
                { name: 'docker', cpu: Math.random() * 10 + 2 }
            ].sort((a, b) => b.cpu - a.cpu);

            this.updateTopProcessesDisplay(mockProcesses);
            this.log('📋 Top processes loaded');
        } catch (error) {
            this.log('❌ Error loading top processes:', error);
        }
    }

    /**
     * Load historical data (UNCHANGED)
     */
    async loadHistoricalData(timeRange = '1h') {
        try {
            console.log(`🔄 Loading historical trends for ${timeRange}...`);
            const response = await fetch(`/api/v1/metrics/trends?range=${timeRange}`);
            const result = await response.json();

            if (result.success && result.data && this.chartManager) {
                console.log('📈 Historical trends data received:', result.data);
                this.chartManager.updateTrendsChartWithHistoricalData(result.data);
                this.log(`📈 Historical data loaded for ${timeRange}`);
            } else {
                console.warn('❌ Failed to load historical trends:', result);
            }
        } catch (error) {
            this.log('❌ Error loading historical data:', error);
            console.error('❌ Error loading historical data:', error);
        }
    }

    /**
     * Handle enhanced metrics update from WebSocket
     */
    handleMetricsUpdate(data) {
        if (!data) {
            this.log('❌ No data received in handleMetricsUpdate');
            return;
        }

        // DEBUG: Log the actual data structure - DETAILED
        console.log('🔍 RAW DATA FROM WEBSOCKET:', data);
        console.log('� CPU VALUE ANALYSIS:', {
            raw_cpu: data.cpu_usage,
            type: typeof data.cpu_usage,
            is_number: !isNaN(data.cpu_usage),
            parsed: parseFloat(data.cpu_usage)
        });
        console.log('🔍 MEMORY VALUE ANALYSIS:', {
            raw_memory: data.memory_percent,
            type: typeof data.memory_percent,
            is_number: !isNaN(data.memory_percent),
            parsed: parseFloat(data.memory_percent)
        });

        this.stats.totalUpdates++;
        this.lastUpdateTime = new Date();

        // Update enhanced metrics display
        this.updateEnhancedMetricsDisplay(data);

        // Update charts
        if (this.chartManager) {
            this.log('📈 Updating enhanced charts...');
            // Use real temperature from CPU metrics instead of simulated
            const enhancedData = {
                ...data,
                temperature: data.cpu_temperature_c || 0
            };
            this.chartManager.updateMetrics(enhancedData);
        } else {
            this.log('⚠️ Chart manager not available, skipping chart updates');
        }

        // Update last update time
        this.updateLastUpdateTime();

        // Store current metrics
        this.currentMetrics = data;

        // Update STABLE moving averages (this now handles ALL calculations)
        this.updateEnhancedMovingAverages(data);

        // Update logged users if client count info is available
        if (data.client_count !== undefined) {
            this.updateElementText(this.elements.clientCount, data.client_count.toString());
            this.updateLoggedUsersFromClientCount();
        }

        this.log('✅ Enhanced detailed metrics update completed');
    }

    /**
     * Handle system status update (UNCHANGED)
     */
    handleSystemStatusUpdate(data) {
        if (!data) return;
        this.updateSystemStatusDisplay(data);
        this.log('🖥️ System status updated');
    }

    /**
     * Update enhanced metrics display in UI
     */
    updateEnhancedMetricsDisplay(metrics) {
        if (!metrics) {
            this.log('❌ No metrics data provided to updateEnhancedMetricsDisplay');
            return;
        }

        try {
            console.log('🔄 Updating enhanced metrics display:', metrics);
            
            // Basic metrics
            if (metrics.cpu_usage !== undefined) {
                const cpuPercentage = Math.round(metrics.cpu_usage);
                this.updateElementText(this.elements.cpuValue, cpuPercentage);
            }

            if (metrics.memory_percent !== undefined) {
                const memoryPercentage = Math.round(metrics.memory_percent);
                this.updateElementText(this.elements.memoryValue, memoryPercentage);
            }

            if (metrics.disk_percent !== undefined) {
                const diskPercentage = Math.round(metrics.disk_percent);
                this.updateElementText(this.elements.diskValue, diskPercentage);
            }

            // Enhanced CPU details
            if (metrics.cpu_cores !== undefined) {
                this.updateElementText(this.elements.cpuCores, metrics.cpu_cores);
            }
            if (metrics.cpu_frequency !== undefined) {
                const freqGHz = (metrics.cpu_frequency / 1000).toFixed(2);
                this.updateElementText(this.elements.cpuFrequency, `${freqGHz} GHz`);
            }
            if (metrics.cpu_load_avg && metrics.cpu_load_avg.length > 0) {
                this.updateElementText(this.elements.cpuLoadAvg, metrics.cpu_load_avg[0].toFixed(2));
            }

            // Real CPU Temperature (instead of simulated)
            if (metrics.cpu && metrics.cpu.temperature_c !== undefined) {
                const temperature = metrics.cpu.temperature_c;
                this.updateElementText(this.elements.temperatureValue, temperature.toFixed(1));
                this.updateElementText(this.elements.cpuTemperature, `${temperature.toFixed(1)}°C`);
                
                // Temperature status
                let status = 'Normal';
                if (temperature > 75) status = 'Hot';
                else if (temperature > 65) status = 'Warm';
                else if (temperature > 55) status = 'Moderate';
                
                this.updateElementText(this.elements.temperatureStatus, status);
                this.updateElementText(this.elements.temperatureMax, '85°C'); // Safe max
            } else if (metrics.cpu_temperature_c !== undefined) {
                // Fallback: check for flat field
                const temperature = metrics.cpu_temperature_c;
                this.updateElementText(this.elements.temperatureValue, temperature.toFixed(1));
                this.updateElementText(this.elements.cpuTemperature, `${temperature.toFixed(1)}°C`);
                
                // Temperature status
                let status = 'Normal';
                if (temperature > 75) status = 'Hot';
                else if (temperature > 65) status = 'Warm';
                else if (temperature > 55) status = 'Moderate';
                
                this.updateElementText(this.elements.temperatureStatus, status);
                this.updateElementText(this.elements.temperatureMax, '85°C'); // Safe max
            }

            // Real Process Data (instead of mock)
            if (metrics.processes) {
                this.updateElementText(this.elements.processValue || document.getElementById('process-value'), metrics.processes.total_processes);
                this.updateElementText(this.elements.processRunning || document.getElementById('process-running'), metrics.processes.running_processes);
                this.updateElementText(this.elements.processSleeping || document.getElementById('process-sleeping'), metrics.processes.stopped_processes);
                this.updateElementText(this.elements.processZombie || document.getElementById('process-zombie'), metrics.processes.zombie_processes);
            }

            // Handle disk partition data (create once, update afterwards)
            if (metrics.disk_partitions && metrics.disk_partitions.length > 0) {
                console.log('🔥 DISK PARTITIONS DETECTED:', metrics.disk_partitions);
                
                // Only create if section doesn't exist yet
                if (!document.getElementById('disk-partitions-section')) {
                    this.createDiskPartitionSection(metrics.disk_partitions);
                } else {
                    // Just update existing charts
                    this.updateDiskPartitionCharts(metrics.disk_partitions);
                }
            } else {
                console.log('❌ No disk partitions data found in metrics:', {
                    hasPartitions: !!metrics.disk_partitions,
                    partitionsLength: metrics.disk_partitions ? metrics.disk_partitions.length : 'undefined',
                    allMetricsKeys: Object.keys(metrics)
                });
            }

            // Enhanced Memory details
            if (metrics.memory_used !== undefined && metrics.memory_total !== undefined) {
                const usedGB = (metrics.memory_used / (1024*1024*1024)).toFixed(1);
                const totalGB = (metrics.memory_total / (1024*1024*1024)).toFixed(1);
                this.updateElementText(this.elements.memoryUsed, `${usedGB} GB`);
                this.updateElementText(this.elements.memoryTotal, `${totalGB} GB`);
            }
            if (metrics.memory_cached !== undefined) {
                const cachedGB = (metrics.memory_cached / (1024*1024*1024)).toFixed(1);
                this.updateElementText(this.elements.memoryCached, `${cachedGB} GB`);
            }

            // Enhanced Disk details - SPEED FOCUSED (instead of usage percentage)
            if (metrics.disk_read_speed_mbps !== undefined && metrics.disk_write_speed_mbps !== undefined) {
                // Show SPEED in main disk card instead of usage percentage
                const readSpeed = metrics.disk_read_speed_mbps.toFixed(1);
                const writeSpeed = metrics.disk_write_speed_mbps.toFixed(1);
                const totalSpeed = (metrics.disk_read_speed_mbps + metrics.disk_write_speed_mbps).toFixed(1);
                
                // Main disk value shows total I/O speed
                this.updateElementText(this.elements.diskValue, totalSpeed);
                this.updateElementText(this.elements.diskReadSpeed, `📖 ${readSpeed} MB/s`);
                this.updateElementText(this.elements.diskWriteSpeed, `✏️ ${writeSpeed} MB/s`);
            } else {
                // Fallback: show old disk usage percentage
                if (metrics.disk_percent !== undefined) {
                    const diskPercentage = Math.round(metrics.disk_percent);
                    this.updateElementText(this.elements.diskValue, diskPercentage);
                    // Update unit to show % for fallback
                    const diskUnit = document.querySelector('.disk-card .unit');
                    if (diskUnit) diskUnit.textContent = '%';
                }
            }
            
            // Always show disk usage percentage in details
            if (metrics.disk_percent !== undefined) {
                const diskPercentage = Math.round(metrics.disk_percent);
                this.updateElementText(this.elements.diskUsagePercent, `${diskPercentage}%`);
            }
            
            // Always show disk space information
            if (metrics.disk_used !== undefined && metrics.disk_total !== undefined && metrics.disk_free !== undefined) {
                const usedGB = (metrics.disk_used / (1024*1024*1024)).toFixed(1);
                const totalGB = (metrics.disk_total / (1024*1024*1024)).toFixed(1);
                const freeGB = (metrics.disk_free / (1024*1024*1024)).toFixed(1);
                this.updateElementText(this.elements.diskUsed, `${usedGB} GB`);
                this.updateElementText(this.elements.diskTotal, `${totalGB} GB`);
                this.updateElementText(this.elements.diskFree, `${freeGB} GB`);
                
                // Update disk storage card in real-time metrics
                this.updateElementText(this.elements.diskStorageUsed, `${usedGB} GB`);
                this.updateElementText(this.elements.diskStorageTotal, `${totalGB} GB`);
                this.updateElementText(this.elements.diskStorageFree, `${freeGB} GB`);
            }
            
            // Update disk storage percentage in real-time metrics
            if (metrics.disk_percent !== undefined) {
                const diskPercentage = Math.round(metrics.disk_percent);
                this.updateElementText(this.elements.diskStorageValue, diskPercentage);
            }

            // Enhanced Network details - SPEED FOCUSED (instead of total bytes)
            if (metrics.network_upload_speed_mbps !== undefined && metrics.network_download_speed_mbps !== undefined) {
                // Show SPEED in main network card instead of total bytes
                const uploadSpeed = metrics.network_upload_speed_mbps;
                const downloadSpeed = metrics.network_download_speed_mbps;
                const totalSpeed = uploadSpeed + downloadSpeed;
                
                // Update main network value to show speed instead of total MB
                this.updateElementText(this.elements.networkValue, totalSpeed.toFixed(1));
                
                // Keep legacy fields for backward compatibility but update with speed values
                this.updateElementText(this.elements.networkSent, `↑ ${uploadSpeed.toFixed(1)} Mbps`);
                this.updateElementText(this.elements.networkReceived, `↓ ${downloadSpeed.toFixed(1)} Mbps`);
            }
            
            // Fallback: If no speed data, show total bytes
            else if (metrics.network_sent !== undefined && metrics.network_received !== undefined) {
                const sentMB = (metrics.network_sent / (1024*1024)).toFixed(1);
                const receivedMB = (metrics.network_received / (1024*1024)).toFixed(1);
                const totalMB = parseFloat(sentMB) + parseFloat(receivedMB);
                this.updateElementText(this.elements.networkValue, totalMB.toFixed(1));
                this.updateElementText(this.elements.networkSent, `${sentMB} MB`);
                this.updateElementText(this.elements.networkReceived, `${receivedMB} MB`);
            }

            // NEW: Enhanced Speed metrics
            if (metrics.disk_read_speed_mbps !== undefined && metrics.disk_write_speed_mbps !== undefined) {
                const readSpeed = metrics.disk_read_speed_mbps.toFixed(1);
                const writeSpeed = metrics.disk_write_speed_mbps.toFixed(1);
                
                this.updateElementText(this.elements.diskReadSpeed, `${readSpeed} MB/s`);
                this.updateElementText(this.elements.diskWriteSpeed, `${writeSpeed} MB/s`);
            }

            if (metrics.network_upload_speed_mbps !== undefined && metrics.network_download_speed_mbps !== undefined) {
                const uploadSpeed = metrics.network_upload_speed_mbps.toFixed(1);
                const downloadSpeed = metrics.network_download_speed_mbps.toFixed(1);
                
                this.updateElementText(this.elements.networkUploadSpeed, `${uploadSpeed} Mbps`);
                this.updateElementText(this.elements.networkDownloadSpeed, `${downloadSpeed} Mbps`);
            }

            // Real data is now used instead of mock enhanced metrics
            // this.updateMockEnhancedMetrics(); // Disabled - using real temperature and process data

            // System information
            if (metrics.hostname) {
                this.updateElementText(this.elements.systemHostname, metrics.hostname);
            }

            // Update uptime if available from metrics
            console.log('🔍 UPTIME DEBUG:', {
                uptime_seconds: metrics.uptime_seconds,
                uptime: metrics.uptime,
                uptime_type: typeof metrics.uptime,
                uptime_raw_value: metrics.uptime,
                all_keys: Object.keys(metrics).filter(key => key.toLowerCase().includes('uptime'))
            });
            
            if (metrics.uptime_seconds !== undefined) {
                const uptime = this.formatUptime(metrics.uptime_seconds);
                console.log('✅ UPTIME FORMATTED (uptime_seconds):', uptime);
                this.updateElementText(this.elements.systemUptime, uptime);
            } else if (metrics.uptime !== undefined) {
                // Check if uptime is a number (seconds) or already formatted string
                if (typeof metrics.uptime === 'number') {
                    // Try different conversions based on the value size
                    let uptimeInSeconds = metrics.uptime;
                    
                    // If the number is very large, it might be in nanoseconds
                    if (uptimeInSeconds > 1000000000000) {
                        // Nanoseconds to seconds (divide by 1,000,000,000)
                        uptimeInSeconds = Math.floor(uptimeInSeconds / 1000000000);
                        console.log('🔧 Converting from nanoseconds:', metrics.uptime, '→', uptimeInSeconds, 'seconds');
                    }
                    // If still large, might be in milliseconds
                    else if (uptimeInSeconds > 1000000000) {
                        // Milliseconds to seconds (divide by 1,000)
                        uptimeInSeconds = Math.floor(uptimeInSeconds / 1000);
                        console.log('🔧 Converting from milliseconds:', metrics.uptime, '→', uptimeInSeconds, 'seconds');
                    }
                    
                    const uptime = this.formatUptime(uptimeInSeconds);
                    console.log('✅ UPTIME FORMATTED (from uptime field):', uptime);
                    this.updateElementText(this.elements.systemUptime, uptime);
                } else {
                    console.log('✅ UPTIME AS STRING:', metrics.uptime);
                    this.updateElementText(this.elements.systemUptime, metrics.uptime);
                }
            }

            this.log('✅ Enhanced metrics display updated');

        } catch (error) {
            this.log('❌ Error updating enhanced metrics display:', error);
        }
    }

    /**
     * Update realistic simulated temperature (instead of random values)
     */
    updateRealisticTemperature() {
        try {
            const now = Date.now();
            const timeDiff = (now - this.simulatedTemperature.lastUpdate) / 1000; // seconds
            
            // Temperature changes very slowly (max 0.1°C per second)
            const maxTempChange = 0.1 * timeDiff;
            
            // Occasionally change target temperature (every ~30 seconds)
            if (Math.random() < 0.02) { // 2% chance per update
                // CPU usage affects target temperature
                const cpuUsage = this.currentMetrics?.cpu_usage || 0;
                
                // Base temperature: 35-50°C
                // High CPU usage adds 5-15°C
                this.simulatedTemperature.target = 35 + (cpuUsage * 0.15) + Math.random() * 5;
                this.simulatedTemperature.target = Math.min(80, Math.max(30, this.simulatedTemperature.target));
            }
            
            // Gradually move current temperature towards target
            const diff = this.simulatedTemperature.target - this.simulatedTemperature.current;
            if (Math.abs(diff) > maxTempChange) {
                this.simulatedTemperature.current += Math.sign(diff) * maxTempChange;
            } else {
                this.simulatedTemperature.current = this.simulatedTemperature.target;
            }
            
            this.simulatedTemperature.lastUpdate = now;
            
            // Round to 1 decimal place
            const currentTemp = Math.round(this.simulatedTemperature.current * 10) / 10;
            
            // Update temperature displays
            this.updateElementText(this.elements.temperatureValue, currentTemp);
            this.updateElementText(this.elements.cpuTemperature, `${currentTemp}°C`);
            
            // Temperature status
            let status = 'Normal';
            if (currentTemp > 75) status = 'Hot';
            else if (currentTemp > 65) status = 'Warm';
            else if (currentTemp > 55) status = 'Moderate';
            
            this.updateElementText(this.elements.temperatureStatus, status);
            this.updateElementText(this.elements.temperatureMax, '85°C'); // Safe max
            
        } catch (error) {
            this.log('❌ Error updating realistic temperature:', error);
        }
    }

    /**
     * Update mock enhanced metrics (Process only - Temperature moved to realistic simulation)
     */
    updateMockEnhancedMetrics() {
        try {
            // Update realistic temperature instead of random
            this.updateRealisticTemperature();

            // Mock process data (still simulated but more stable)
            const baseProcessCount = 150;
            const variation = Math.sin(Date.now() / 30000) * 20; // Slow sine wave variation
            const mockProcessTotal = Math.floor(baseProcessCount + variation);
            const mockRunning = Math.floor(mockProcessTotal * (0.08 + Math.random() * 0.04));
            const mockSleeping = Math.floor(mockProcessTotal * (0.85 + Math.random() * 0.05));
            const mockOther = mockProcessTotal - mockRunning - mockSleeping;

            this.updateElementText(this.elements.processValue, mockProcessTotal);
            this.updateElementText(this.elements.processRunning, mockRunning);
            this.updateElementText(this.elements.processSleeping, mockSleeping);
            this.updateElementText(this.elements.processZombie, mockOther);

        } catch (error) {
            this.log('❌ Error updating mock enhanced metrics:', error);
        }
    }

    /**
     * Update STABLE moving averages for Quick Stats - COMPLETELY REWRITTEN
     */
    updateEnhancedMovingAverages(metrics) {
        if (!metrics) return;

        try {
            // STABLE APPROACH: Use simple rolling averages with decay
            const alpha = 0.1; // Smoothing factor (0.1 = %10 new data, %90 old average)
            
            // CPU with exponential smoothing
            if (metrics.cpu_usage !== undefined && metrics.cpu_usage !== null && !isNaN(metrics.cpu_usage)) {
                const cpuValue = Math.max(0, Math.min(100, parseFloat(metrics.cpu_usage)));
                
                if (this.simpleAverages.cpu.count === 0) {
                    // First value
                    this.simpleAverages.cpu.current = cpuValue;
                } else {
                    // Exponential smoothing: new_avg = alpha * new_value + (1-alpha) * old_avg
                    this.simpleAverages.cpu.current = alpha * cpuValue + (1 - alpha) * this.simpleAverages.cpu.current;
                }
                this.simpleAverages.cpu.count++;
                
                // Update display immediately with smoothed value
                const displayCpu = Math.round(this.simpleAverages.cpu.current * 10) / 10;
                this.updateElementText(this.elements.avgCpu, `${displayCpu}%`);
                
                this.log('� CPU Smooth Average:', {
                    raw: cpuValue,
                    smoothed: displayCpu,
                    count: this.simpleAverages.cpu.count
                });
            }

            // Memory with exponential smoothing
            if (metrics.memory_percent !== undefined && metrics.memory_percent !== null && !isNaN(metrics.memory_percent)) {
                const memoryValue = Math.max(0, Math.min(100, parseFloat(metrics.memory_percent)));
                
                if (this.simpleAverages.memory.count === 0) {
                    // First value
                    this.simpleAverages.memory.current = memoryValue;
                } else {
                    // Exponential smoothing
                    this.simpleAverages.memory.current = alpha * memoryValue + (1 - alpha) * this.simpleAverages.memory.current;
                }
                this.simpleAverages.memory.count++;
                
                // Update display immediately with smoothed value
                const displayMemory = Math.round(this.simpleAverages.memory.current * 10) / 10;
                this.updateElementText(this.elements.avgMemory, `${displayMemory}%`);
                
                this.log('🧠 Memory Smooth Average:', {
                    raw: memoryValue,
                    smoothed: displayMemory,
                    count: this.simpleAverages.memory.count
                });
            }

            // Disk I/O with exponential smoothing
            if (metrics.disk_read_speed_mbps !== undefined && metrics.disk_write_speed_mbps !== undefined) {
                const diskRead = parseFloat(metrics.disk_read_speed_mbps) || 0;
                const diskWrite = parseFloat(metrics.disk_write_speed_mbps) || 0;
                const totalDiskIO = Math.max(0, diskRead + diskWrite);
                
                if (this.simpleAverages.diskIO.count === 0) {
                    this.simpleAverages.diskIO.current = totalDiskIO;
                } else {
                    this.simpleAverages.diskIO.current = alpha * totalDiskIO + (1 - alpha) * this.simpleAverages.diskIO.current;
                }
                this.simpleAverages.diskIO.count++;
                
                const displayDiskIO = Math.round(this.simpleAverages.diskIO.current * 10) / 10;
                this.updateElementText(this.elements.avgDiskIO, `${displayDiskIO} MB/s`);
            }

            // Network with exponential smoothing
            if (metrics.network_upload_speed_mbps !== undefined && metrics.network_download_speed_mbps !== undefined) {
                const netUpload = parseFloat(metrics.network_upload_speed_mbps) || 0;
                const netDownload = parseFloat(metrics.network_download_speed_mbps) || 0;
                const totalNetwork = Math.max(0, netUpload + netDownload);
                
                if (this.simpleAverages.network.count === 0) {
                    this.simpleAverages.network.current = totalNetwork;
                } else {
                    this.simpleAverages.network.current = alpha * totalNetwork + (1 - alpha) * this.simpleAverages.network.current;
                }
                this.simpleAverages.network.count++;
                
                const displayNetwork = Math.round(this.simpleAverages.network.current * 10) / 10;
                this.updateElementText(this.elements.avgNetwork, `${displayNetwork} Mbps`);
            }

        } catch (error) {
            this.log('❌ Error updating stable moving averages:', error);
        }
    }

    /**
     * Update enhanced Quick Stats averages - REMOVED (now handled in updateEnhancedMovingAverages)
     * This method is no longer needed as we use exponential smoothing directly
     */
    updateEnhancedQuickStatsAverages() {
        // This method is intentionally empty - calculations are now done in updateEnhancedMovingAverages
        // using exponential smoothing for much more stable results
    }

    /**
     * Update Enhanced Quick Stats from current metrics - SIMPLIFIED (rarely used)
     */
    updateEnhancedQuickStatsFromCurrentMetrics(metrics) {
        if (!metrics) return;

        try {
            // This is now only used for initial values or as emergency fallback
            // The main calculations are done in updateEnhancedMovingAverages with exponential smoothing
            
            // Total hosts is always 1 for single-host setup
            this.updateElementText(this.elements.totalHosts, '1');

        } catch (error) {
            this.log('❌ Error updating enhanced quick stats from current metrics:', error);
        }
    }

    /**
     * Update enhanced system details display
     */
    updateEnhancedSystemDetailsDisplay(systemInfo) {
        if (!systemInfo) return;

        try {
            this.updateElementText(this.elements.systemPlatform, systemInfo.platform || 'Unknown');
            this.updateElementText(this.elements.systemArch, systemInfo.architecture || 'Unknown');
            this.updateElementText(this.elements.systemUptime, systemInfo.uptime || 'Unknown');
            this.updateElementText(this.elements.loggedUsers, systemInfo.loggedUsers || '0');
            
            this.log('🖥️ Enhanced system details display updated');
        } catch (error) {
            this.log('❌ Error updating enhanced system details display:', error);
        }
    }

    /**
     * Update enhanced stats display
     */
    updateEnhancedStatsDisplay(stats) {
        if (!stats) return;

        try {
            // Total metrics
            this.updateElementText(this.elements.totalMetrics, this.formatNumber(stats.total_metrics || 0));
            
            // Total hosts
            if (stats.total_hosts !== undefined) {
                this.updateElementText(this.elements.totalHosts, stats.total_hosts);
            } else {
                this.updateElementText(this.elements.totalHosts, '1');
            }
            
            // Use API averages if available, otherwise rely on moving averages
            if (stats.avg_cpu_usage !== undefined && stats.avg_cpu_usage !== null && stats.avg_cpu_usage > 0) {
                this.updateElementText(this.elements.avgCpu, `${stats.avg_cpu_usage.toFixed(1)}%`);
                this.hasApiAverages = true;
            }
            
            if (stats.avg_memory_usage !== undefined && stats.avg_memory_usage !== null && stats.avg_memory_usage > 0) {
                this.updateElementText(this.elements.avgMemory, `${stats.avg_memory_usage.toFixed(1)}%`);
                this.hasApiAverages = true;
            }

            this.log('📊 Enhanced stats display updated');
        } catch (error) {
            this.log('❌ Error updating enhanced stats display:', error);
        }
    }

    /**
     * Update system status display - IMPROVED LAYOUT
     */
    updateSystemStatusDisplay(statusData) {
        if (!statusData || !this.elements.systemStatus) return;

        try {
            const statusArray = Array.isArray(statusData) ? statusData : [statusData];
            
            const statusHTML = statusArray.map(status => `
                <div class="status-item">
                    <div class="status-item-left">
                        <div class="status-badge ${status.status || 'online'}"></div>
                        <div class="status-content">
                            <div class="status-hostname">${status.hostname || 'localhost'}</div>
                            <div class="status-metrics">
                                CPU: ${(status.cpu_usage || 0).toFixed(1)}% | Memory: ${(status.memory_usage || status.memory_percent || 0).toFixed(1)}% | Disk: ${(status.disk_usage || status.disk_percent || 0).toFixed(1)}%
                            </div>
                            <div class="status-time">${this.formatTime(status.timestamp || new Date().toISOString())}</div>
                        </div>
                    </div>
                    <div class="status-badge-text ${status.status || 'online'}">${status.status || 'online'}</div>
                </div>
            `).join('');

            this.elements.systemStatus.innerHTML = statusHTML;
        } catch (error) {
            this.log('❌ Error updating system status display:', error);
        }
    }

    /**
     * Update top processes display (NEW)
     */
    updateTopProcessesDisplay(processes) {
        if (!processes || !this.elements.topProcesses) return;

        try {
            const processesHTML = processes.map(process => `
                <div class="process-item">
                    <div class="process-info">
                        <span class="process-name">${process.name}</span>
                        <span class="process-cpu">${process.cpu.toFixed(1)}%</span>
                    </div>
                </div>
            `).join('');

            this.elements.topProcesses.innerHTML = processesHTML;
            this.log('📋 Top processes display updated');
        } catch (error) {
            this.log('❌ Error updating top processes display:', error);
        }
    }

    /**
     * Update connection status indicator (UNCHANGED)
     */
    updateConnectionStatus(status, message) {
        if (!this.elements.wsStatus) return;

        const statusElement = this.elements.wsStatus;
        const statusText = statusElement.querySelector('span');

        statusElement.classList.remove('connected', 'disconnected', 'connecting', 'error');
        statusElement.classList.add(status);

        if (statusText) {
            statusText.textContent = message;
        }

        if (this.websocket && this.elements.clientCount) {
            this.elements.clientCount.textContent = '1';
            
            // Update logged users to match client count
            this.updateLoggedUsersFromClientCount();
        }

        this.log(`🔗 Connection status updated: ${status} - ${message}`);
    }

    /**
     * Update logged users based on current client count
     */
    updateLoggedUsersFromClientCount() {
        try {
            const clientCount = this.getCurrentClientCount();
            if (this.elements.loggedUsers) {
                this.updateElementText(this.elements.loggedUsers, clientCount.toString());
                this.log(`👥 Logged users updated to: ${clientCount}`);
            }
        } catch (error) {
            this.log('❌ Error updating logged users from client count:', error);
        }
    }

    /**
     * Change time range for historical data (UNCHANGED)
     */
    async changeTimeRange(range) {
        try {
            this.elements.timeButtons.forEach(btn => btn.classList.remove('active'));
            const activeButton = document.querySelector(`[data-range="${range}"]`);
            if (activeButton) {
                activeButton.classList.add('active');
            }

            await this.loadHistoricalData(range);
            this.log(`📅 Time range changed to: ${range}`);
        } catch (error) {
            this.log('❌ Error changing time range:', error);
        }
    }

    /**
     * Setup enhanced refresh intervals
     */
    setupRefreshIntervals() {
        console.log('🚀 Setting up enhanced 1-second intervals for smooth real-time updates');
        
        // Clear existing intervals
        if (this.statsInterval) clearInterval(this.statsInterval);
        if (this.statusInterval) clearInterval(this.statusInterval);
        if (this.metricsInterval) clearInterval(this.metricsInterval);
        if (this.processInterval) clearInterval(this.processInterval);
        
        // Enhanced polling intervals (1 second for smooth updates)
        this.statsInterval = setInterval(() => {
            if (!this.isPaused) {
                this.loadSystemStats();
            }
        }, 1000);

        this.statusInterval = setInterval(() => {
            if (!this.isPaused) {
                this.loadSystemStatus();
            }
        }, 2000); // System status every 2 seconds

        this.metricsInterval = setInterval(() => {
            if (!this.isPaused) {
                this.loadCurrentMetrics();
            }
        }, 1000);

        // NEW: Top processes refresh (every 5 seconds)
        this.processInterval = setInterval(() => {
            if (!this.isPaused) {
                this.loadTopProcesses();
            }
        }, 5000);

        // Update timestamps every second
        this.timestampInterval = setInterval(() => {
            this.updateLastUpdateTime();
        }, 1000);
        
        console.log('✅ Enhanced 1-second intervals set up successfully');
    }

    // Include all remaining methods from the original dashboard.js
    // (showAlert, hideAlert, showDashboard, showError, handleKeyboardShortcuts, 
    //  refreshData, toggleDebugMode, pauseUpdates, resumeUpdates, etc.)
    // These remain unchanged from the original implementation

    /**
     * Show alert banner (UNCHANGED)
     */
    showAlert(message, type = 'info') {
        if (!this.elements.alertBanner || !this.elements.alertMessage) return;

        this.elements.alertMessage.textContent = message;
        this.elements.alertBanner.className = `alert-banner ${type}`;
        this.elements.alertBanner.style.display = 'block';

        if (this.alertTimeout) {
            clearTimeout(this.alertTimeout);
        }

        if (type === 'success') {
            this.alertTimeout = setTimeout(() => {
                this.hideAlert();
            }, this.options.alertTimeout);
        }

        this.log(`🚨 Alert shown: ${type} - ${message}`);
    }

    /**
     * Hide alert banner (UNCHANGED)
     */
    hideAlert() {
        if (!this.elements.alertBanner) return;

        this.elements.alertBanner.style.display = 'none';
        
        if (this.alertTimeout) {
            clearTimeout(this.alertTimeout);
            this.alertTimeout = null;
        }
    }

    /**
     * Show dashboard and hide loading screen (UNCHANGED)
     */
    showDashboard() {
        if (this.elements.loadingScreen) {
            this.elements.loadingScreen.style.display = 'none';
        }
        
        if (this.elements.dashboard) {
            this.elements.dashboard.style.display = 'block';
        }

        this.log('🎨 Enhanced dashboard displayed');
    }

    /**
     * Show error state (UNCHANGED)
     */
    showError(message) {
        this.showAlert(message, 'error');
        this.stats.errors++;
        this.stats.lastError = new Date();
    }

    /**
     * Handle keyboard shortcuts (UNCHANGED)
     */
    handleKeyboardShortcuts(event) {
        if (event.key === 'Escape') {
            this.hideAlert();
        }

        if ((event.ctrlKey && event.key === 'r') || event.key === 'F5') {
            event.preventDefault();
            this.refreshData();
        }

        if (event.ctrlKey && event.key === 'd') {
            event.preventDefault();
            this.toggleDebugMode();
        }
    }

    /**
     * Refresh all data (ENHANCED)
     */
    async refreshData() {
        try {
            this.showAlert('Refreshing enhanced data...', 'info');
            await this.loadInitialData();
            this.showAlert('Enhanced data refreshed successfully!', 'success');
        } catch (error) {
            this.showAlert('Failed to refresh enhanced data', 'error');
            this.handleError(error);
        }
    }

    /**
     * Toggle debug mode (UNCHANGED)
     */
    toggleDebugMode() {
        this.options.debug = !this.options.debug;
        
        if (this.websocket) {
            this.websocket.setDebug(this.options.debug);
        }

        this.showAlert(`Enhanced debug mode ${this.options.debug ? 'enabled' : 'disabled'}`, 'info');
    }

    /**
     * Pause updates (UNCHANGED)
     */
    pauseUpdates() {
        if (this.isPaused) return;
        this.isPaused = true;
        this.log('⏸️ Enhanced updates paused');
    }

    /**
     * Resume updates (UNCHANGED)
     */
    resumeUpdates() {
        if (!this.isPaused) return;
        this.isPaused = false;
        this.refreshData();
        this.log('▶️ Enhanced updates resumed');
    }

    /**
     * Initialize tooltips (UNCHANGED)
     */
    initializeTooltips() {
        this.log('💡 Enhanced tooltips initialized');
    }

    /**
     * Setup mobile menu (UNCHANGED)
     */
    setupMobileMenu() {
        if (this.elements.mobileMenuToggle) {
            this.elements.mobileMenuToggle.addEventListener('click', () => {
                this.log('📱 Enhanced mobile menu toggled');
            });
        }
    }

    /**
     * Get status icon based on status (UNCHANGED)
     */
    getStatusIcon(status) {
        const icons = {
            'online': 'check-circle',
            'warning': 'exclamation-triangle',
            'error': 'times-circle',
            'offline': 'minus-circle'
        };
        return icons[status] || 'question-circle';
    }

    /**
     * Update element text safely (UNCHANGED)
     */
    updateElementText(element, text) {
        if (element && text !== undefined) {
            element.textContent = text;
        }
    }

    /**
     * Update last update time display (UNCHANGED)
     */
    updateLastUpdateTime() {
        if (this.elements.lastUpdate && this.lastUpdateTime) {
            const timeAgo = this.formatTimeAgo(this.lastUpdateTime);
            this.updateElementText(this.elements.lastUpdate, timeAgo);
        }
    }

    /**
     * Format uptime from seconds to human readable format with days, hours, minutes, seconds
     */
    formatUptime(seconds) {
        if (!seconds || seconds < 0) return 'Unknown';
        
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = Math.floor(seconds % 60);
        
        let parts = [];
        
        if (days > 0) {
            parts.push(`${days} gün`);
        }
        if (hours > 0) {
            parts.push(`${hours} saat`);
        }
        if (minutes > 0) {
            parts.push(`${minutes} dakika`);
        }
        if (secs > 0 || parts.length === 0) {
            parts.push(`${secs} saniye`);
        }
        
        return parts.join(', ');
    }

    /**
     * Format time ago string (UNCHANGED)
     */
    formatTimeAgo(date) {
        const now = new Date();
        const diff = now - date;
        const seconds = Math.floor(diff / 1000);

        if (seconds < 60) return `${seconds}s ago`;
        
        const minutes = Math.floor(seconds / 60);
        if (minutes < 60) return `${minutes}m ago`;
        
        const hours = Math.floor(minutes / 60);
        if (hours < 24) return `${hours}h ago`;
        
        const days = Math.floor(hours / 24);
        return `${days}d ago`;
    }

    /**
     * Format time string (UNCHANGED)
     */
    formatTime(timestamp) {
        if (!timestamp) return 'Never';
        const date = new Date(timestamp);
        return date.toLocaleTimeString();
    }

    /**
     * Format number with commas (UNCHANGED)
     */
    formatNumber(num) {
        return num.toLocaleString();
    }

    /**
     * Handle errors (UNCHANGED)
     */
    handleError(error) {
        this.stats.errors++;
        this.stats.lastError = new Date();
        this.log('❌ Error handled:', error);
        
        if (this.options.debug) {
            console.error('Enhanced Dashboard Error:', error);
        }
    }

    /**
     * Create disk partition section using existing structure (create once only)
     */
    createDiskPartitionSection(diskPartitions) {
        if (!diskPartitions || diskPartitions.length === 0) {
            console.log('No disk partitions data available');
            return;
        }

        console.log('Creating disk partition section for:', diskPartitions);

        // Find the metrics section
        const metricsSection = document.querySelector('.metrics-section');
        if (!metricsSection) {
            console.error('Metrics section not found');
            return;
        }

        // Check if partition section already exists - if yes, don't recreate
        let partitionSection = document.getElementById('disk-partitions-section');
        
        if (partitionSection) {
            console.log('Partition section already exists, skipping creation');
            return; // Don't recreate if already exists
        }

        // Create new section only if it doesn't exist
        partitionSection = document.createElement('section');
        partitionSection.id = 'disk-partitions-section';
        partitionSection.className = 'metrics-section';
        partitionSection.innerHTML = `
            <h2>
                <i class="fas fa-hdd"></i>
                Disk Partitions
            </h2>
            <div class="metrics-grid" id="disk-partitions-grid">
            </div>
        `;
        
        // Insert after main metrics section
        metricsSection.parentNode.insertBefore(partitionSection, metricsSection.nextSibling);

        const partitionsGrid = document.getElementById('disk-partitions-grid');
        if (!partitionsGrid) {
            console.error('Partitions grid not found');
            return;
        }

        // Create card for each partition
        diskPartitions.forEach((partition, index) => {
            if (!partition.device || partition.total_bytes === 0) {
                console.log('Skipping partition with no device or 0 total bytes:', partition);
                return;
            }

            const usagePercent = partition.usage_percent || 0;
            const usedGB = (partition.used_bytes / (1024*1024*1024)).toFixed(1);
            const freeGB = (partition.free_bytes / (1024*1024*1024)).toFixed(1);
            const totalGB = (partition.total_bytes / (1024*1024*1024)).toFixed(1);

            const partitionCard = document.createElement('div');
            partitionCard.className = 'metric-card disk-card';
            partitionCard.innerHTML = `
                <div class="metric-header">
                    <h3>
                        <i class="fas fa-hdd"></i>
                        ${partition.device} (${partition.mountpoint})
                    </h3>
                </div>
                
                <div class="metric-value">
                    <span id="partition-usage-${index}">${usagePercent.toFixed(1)}</span><span class="unit">%</span>
                </div>
                
                <div class="chart-container">
                    <canvas id="partition-chart-${index}"></canvas>
                </div>
                
                <div class="metric-details">
                    <div class="detail-item">
                        <span>Used</span>
                        <span id="partition-used-${index}">${usedGB} GB</span>
                    </div>
                    <div class="detail-item">
                        <span>Free</span>
                        <span id="partition-free-${index}">${freeGB} GB</span>
                    </div>
                    <div class="detail-item">
                        <span>Total</span>
                        <span id="partition-total-${index}">${totalGB} GB</span>
                    </div>
                    <div class="detail-item">
                        <span>Type</span>
                        <span>${partition.fstype}</span>
                    </div>
                </div>
            `;

            partitionsGrid.appendChild(partitionCard);

            // Initialize chart for this partition
            setTimeout(() => {
                if (this.chartManager && this.chartManager.initializePartitionChart) {
                    console.log(`Partition chart initialized for ${partition.device}`);
                    this.chartManager.initializePartitionChart(index, partition);
                }
            }, 100); // Small delay to ensure DOM is ready
        });

        console.log(`Created ${diskPartitions.length} partition cards`);
    }

    /**
     * Update existing disk partition charts with new data
     */
    updateDiskPartitionCharts(diskPartitions) {
        if (!diskPartitions || diskPartitions.length === 0 || !this.chartManager) return;

        diskPartitions.forEach((partition, index) => {
            const usagePercent = partition.usage_percent || 0;
            
            // Update chart
            if (this.chartManager && this.chartManager.updatePartitionChart) {
                this.chartManager.updatePartitionChart(index, usagePercent);
            }
            
            // Update text values
            const usageElement = document.getElementById(`partition-usage-${index}`);
            if (usageElement) {
                usageElement.textContent = usagePercent.toFixed(1);
            }
            
            const usedGB = (partition.used_bytes / (1024*1024*1024)).toFixed(1);
            const freeGB = (partition.free_bytes / (1024*1024*1024)).toFixed(1);
            const totalGB = (partition.total_bytes / (1024*1024*1024)).toFixed(1);
            
            this.updateElementText(document.getElementById(`partition-used-${index}`), `${usedGB} GB`);
            this.updateElementText(document.getElementById(`partition-free-${index}`), `${freeGB} GB`);
            this.updateElementText(document.getElementById(`partition-total-${index}`), `${totalGB} GB`);
        });
    }

    /**
     * Get enhanced dashboard statistics
     */
    getStats() {
        return {
            ...this.stats,
            isInitialized: this.isInitialized,
            isPaused: this.isPaused,
            lastUpdateTime: this.lastUpdateTime,
            websocketStats: this.websocket ? this.websocket.getStats() : null,
            chartStats: this.chartManager ? this.chartManager.getStats() : null,
            movingAverages: {
                cpu: this.movingAverages.cpu.length,
                memory: this.movingAverages.memory.length,
                diskIO: this.movingAverages.diskIO.length,
                network: this.movingAverages.network.length
            }
        };
    }

    /**
     * Enhanced cleanup resources
     */
    cleanup() {
        this.log('🧹 Cleaning up enhanced dashboard...');

        if (this.websocket) {
            this.websocket.destroy();
            this.websocket = null;
        }

        if (this.chartManager) {
            this.chartManager.destroyCharts();
            this.chartManager = null;
        }

        // Clear enhanced intervals
        [this.statsInterval, this.statusInterval, this.metricsInterval, 
         this.processInterval, this.timestampInterval].forEach(interval => {
            if (interval) {
                clearInterval(interval);
            }
        });

        if (this.alertTimeout) {
            clearTimeout(this.alertTimeout);
            this.alertTimeout = null;
        }

        this.eventHandlers.clear();

        // Clear enhanced moving averages
        Object.keys(this.movingAverages).forEach(key => {
            if (Array.isArray(this.movingAverages[key])) {
                this.movingAverages[key] = [];
            }
        });

        this.isInitialized = false;
        this.log('✅ Enhanced dashboard cleanup complete');
    }

    /**
     * Log messages (UNCHANGED)
     */
    log(...args) {
        if (this.options.debug) {
            console.log('[Enhanced Dashboard]', ...args);
        }
    }
}

// Export enhanced version
if (typeof module !== 'undefined' && module.exports) {
    module.exports = DashboardApp;
} else if (typeof window !== 'undefined') {
    window.DashboardApp = DashboardApp;
}