/**
 * GoDash Dashboard Main Application (WITH SPEED SUPPORT)
 * Orchestrates WebSocket connections, chart management, and UI updates with speed monitoring
 */

class DashboardApp {
    constructor(options = {}) {
        this.options = {
            updateInterval: 1000, // 1 second for real-time updates
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
        this.hasApiAverages = false; // Track if we have API calculated averages
        
        // Moving averages for Quick Stats
        this.movingAverages = {
            cpu: [],
            memory: [],
            maxSamples: 60 // Keep last 60 samples (1 minute at 1 sample/second)
        };

        // Component instances
        this.websocket = null;
        this.chartManager = null;

        // DOM element caches (SPEED ELEMENTS ADDED)
        this.elements = {};
        
        // Data storage
        this.currentMetrics = null;
        this.systemInfo = null;
        this.alerts = [];

        // Statistics
        this.stats = {
            totalUpdates: 0,
            errors: 0,
            connectionTime: null,
            lastError: null
        };

        // Event handlers registry
        this.eventHandlers = new Map();

        this.log('🎯 Dashboard app initialized with speed support:', this.options);
    }

    /**
     * Initialize the dashboard application
     */
    async initialize() {
        if (this.isInitialized) {
            this.log('⚠️ Dashboard already initialized');
            return;
        }

        try {
            this.log('🚀 Initializing GoDash Dashboard with speed support...');

            // Cache DOM elements
            this.cacheElements();

            // Initialize chart manager (wait for completion)
            await this.initializeChartManager();

            // Initialize WebSocket connection
            this.initializeWebSocket();

            // Setup event listeners
            this.setupEventListeners();

            // Setup UI components
            this.setupUIComponents();

            // Load initial data
            await this.loadInitialData();

            // Hide loading screen and show dashboard
            this.showDashboard();

            this.isInitialized = true;
            this.stats.connectionTime = new Date();
            
            this.log('✅ Dashboard initialized successfully with speed support');
            this.updateConnectionStatus('connected', 'Connected');

        } catch (error) {
            this.log('❌ Error initializing dashboard:', error);
            this.handleError(error);
            this.showError('Failed to initialize dashboard');
            
            // Show dashboard even with errors - basic functionality should still work
            this.showDashboard();
        }
    }

    /**
     * Cache frequently used DOM elements (SPEED ELEMENTS ADDED)
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

            // Metric values
            cpuValue: document.getElementById('cpu-value'),
            memoryValue: document.getElementById('memory-value'),
            diskValue: document.getElementById('disk-value'),
            networkValue: document.getElementById('network-value'),

            // Metric details (SPEED FIELDS ADDED)
            cpuCores: document.getElementById('cpu-cores'),
            cpuFrequency: document.getElementById('cpu-frequency'),
            memoryUsed: document.getElementById('memory-used'),
            memoryTotal: document.getElementById('memory-total'),
            diskUsed: document.getElementById('disk-used'),
            diskTotal: document.getElementById('disk-total'),
            networkSent: document.getElementById('network-sent'),
            networkReceived: document.getElementById('network-received'),
            
            // NEW: Speed display elements
            diskIOValue: document.getElementById('disk-io-value'),
            diskReadSpeed: document.getElementById('disk-read-speed'),
            diskWriteSpeed: document.getElementById('disk-write-speed'),
            networkUploadSpeed: document.getElementById('network-upload-speed'),
            networkDownloadSpeed: document.getElementById('network-download-speed'),

            // System information
            systemHostname: document.getElementById('system-hostname'),
            systemPlatform: document.getElementById('system-platform'),
            systemArch: document.getElementById('system-arch'),
            systemUptime: document.getElementById('system-uptime'),
            lastUpdate: document.getElementById('last-update'),

            // Quick stats
            totalHosts: document.getElementById('total-hosts'),
            totalMetrics: document.getElementById('total-metrics'),
            avgCpu: document.getElementById('avg-cpu'),
            avgMemory: document.getElementById('avg-memory'),

            // System status
            systemStatus: document.getElementById('system-status'),

            // Time range buttons
            timeButtons: document.querySelectorAll('.time-btn'),

            // Mobile menu
            mobileMenuToggle: document.getElementById('mobile-menu-toggle')
        };

        this.log('📋 DOM elements cached with speed support');
        
        // Debug element existence (SPEED ELEMENTS INCLUDED)
        console.log('🔍 Element check:');
        console.log('cpuValue:', this.elements.cpuValue);
        console.log('memoryValue:', this.elements.memoryValue);
        console.log('diskValue:', this.elements.diskValue);
        console.log('diskIOValue:', this.elements.diskIOValue);
        console.log('networkValue:', this.elements.networkValue);
        console.log('cpuCores:', this.elements.cpuCores);
        console.log('cpuFrequency:', this.elements.cpuFrequency);
        console.log('diskReadSpeed:', this.elements.diskReadSpeed);
        console.log('diskWriteSpeed:', this.elements.diskWriteSpeed);
        console.log('networkUploadSpeed:', this.elements.networkUploadSpeed);
        console.log('networkDownloadSpeed:', this.elements.networkDownloadSpeed);
    }

    /**
     * Initialize Chart Manager
     */
    async initializeChartManager() {
        try {
            console.log('🎯 Initializing Chart Manager with speed support...');
            
            this.chartManager = new ChartManager({
                maxDataPoints: 50,
                animationDuration: this.options.chartUpdateAnimation ? 100 : 0,
                theme: 'dark'
            });

            // Wait for Chart Manager to be fully initialized
            await new Promise((resolve) => {
                const checkInitialized = () => {
                    if (this.chartManager.isInitialized) {
                        console.log('📊 Chart manager initialized successfully with speed support');
                        resolve(true);
                    } else {
                        setTimeout(checkInitialized, 100);
                    }
                };
                checkInitialized();
            });

            return true;
        } catch (error) {
            console.error('❌ Chart manager initialization failed:', error);
            this.chartManager = null;
            return false;
        }
    }

    /**
     * Initialize WebSocket connection
     */
    initializeWebSocket() {
        console.log('🔌 Dashboard: Initializing WebSocket connection...');
        
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

        // WebSocket event handlers
        this.websocket.on('connect', (event) => {
            this.log('🔌 WebSocket connected');
            this.connectionAttempts = 0;
            this.updateConnectionStatus('connected', 'Connected');
            this.hideAlert();
            
            // Subscribe to metrics updates
            this.websocket.subscribe(['metrics', 'system_status']);
        });

        this.websocket.on('disconnect', (event) => {
            this.log('🔌 WebSocket disconnected');
            this.updateConnectionStatus('disconnected', 'Disconnected');
            
            if (event.code !== 1000) { // Not a clean disconnect
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
            console.log('🔥 DETAILED METRICS DATA WITH SPEED RECEIVED:', data);
            this.handleMetricsUpdate(data);
        });

        this.websocket.on('system_status', (data) => {
            this.handleSystemStatusUpdate(data);
        });

        this.websocket.on('pong', (data) => {
            this.log('🏓 Pong received:', data);
        });

        // Connect to WebSocket
        this.websocket.connect();
        this.log('🔌 WebSocket connection initiated');
    }

    /**
     * Setup event listeners
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

        // Window resize handler
        window.addEventListener('resize', () => {
            if (this.chartManager) {
                this.chartManager.resizeCharts();
            }
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            this.handleKeyboardShortcuts(e);
        });

        // Page visibility change
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.pauseUpdates();
            } else {
                this.resumeUpdates();
            }
        });

        this.log('👂 Event listeners setup complete');
    }

    /**
     * Setup UI components
     */
    setupUIComponents() {
        // Initialize tooltips (if needed)
        this.initializeTooltips();

        // Setup mobile menu (for future use)
        this.setupMobileMenu();

        // Setup refresh intervals
        this.setupRefreshIntervals();

        this.log('🎨 UI components setup complete');
    }

    /**
     * Load initial data from API
     */
    async loadInitialData() {
        try {
            this.log('📥 Loading initial data...');

            // Load current metrics
            await this.loadCurrentMetrics();

            // Load system statistics
            await this.loadSystemStats();

            // Load system details
            await this.loadSystemDetails();

            // Load system status
            await this.loadSystemStatus();

            // Load historical data for trends
            await this.loadHistoricalData();

            this.log('✅ Initial data loaded successfully');
        } catch (error) {
            this.log('❌ Error loading initial data:', error);
            this.handleError(error);
        }
    }

    /**
     * Load current metrics from API (SPEED SUPPORT ADDED)
     */
    async loadCurrentMetrics() {
        try {
            console.log('🔄 Loading current metrics with speed data...');
            const response = await fetch('/api/v1/metrics/current');
            const result = await response.json();

            if (result.success && result.data) {
                console.log('📊 Current detailed metrics with speed data:', result.data);
                
                this.updateMetricsDisplay(result.data);
                
                // Update charts as well
                if (this.chartManager) {
                    console.log('📈 Calling chartManager.updateMetrics with speed data...');
                    this.chartManager.updateMetrics(result.data);
                    console.log('✅ Chart manager update completed with speed data');
                } else {
                    console.warn('❌ Chart manager not available in dashboard');
                }
                
                this.currentMetrics = result.data;
                this.log('📊 Current metrics with speed loaded and charts updated');
            } else {
                console.warn('❌ Invalid metrics response:', result);
            }
        } catch (error) {
            this.log('❌ Error loading current metrics:', error);
            console.error('❌ Error loading current metrics:', error);
        }
    }

    /**
     * Load system statistics
     */
    async loadSystemStats() {
        try {
            const response = await fetch('/api/v1/system/stats');
            const result = await response.json();

            if (result.success && result.data) {
                this.updateStatsDisplay(result.data);
                this.log('📈 System stats loaded');
            }
        } catch (error) {
            this.log('❌ Error loading system stats:', error);
        }
    }

    /**
     * Load system details
     */
    async loadSystemDetails() {
        try {
            // For now, we'll use static system information
            // In a real implementation, you might have a dedicated endpoint
            const systemInfo = {
                platform: navigator.platform || 'Unknown',
                architecture: navigator.userAgent.includes('x64') ? 'x64' : 'x86',
                uptime: 'Calculating...'
            };

            this.updateSystemDetailsDisplay(systemInfo);
            this.log('🖥️ System details loaded');
        } catch (error) {
            this.log('❌ Error loading system details:', error);
        }
    }

    /**
     * Load system status
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
     * Load historical data for charts (DÜZELTİLMİŞ VERSİYON)
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
     * Handle metrics update from WebSocket (SPEED SUPPORT ADDED)
     */
    handleMetricsUpdate(data) {
        if (!data) {
            this.log('❌ No data received in handleMetricsUpdate');
            return;
        }

        this.log('📊 Detailed metrics with speed data received:', data);

        this.stats.totalUpdates++;
        this.lastUpdateTime = new Date();

        // Update metrics display with speed data
        this.updateMetricsDisplay(data);

        // Update charts
        if (this.chartManager) {
            this.log('📈 Updating charts with speed data...');
            this.chartManager.updateMetrics(data);
        } else {
            this.log('⚠️ Chart manager not available, skipping chart updates');
        }

        // Update last update time
        this.updateLastUpdateTime();

        // Store current metrics
        this.currentMetrics = data;

        // Update moving averages
        this.updateMovingAverages(data);

        // Also update Quick Stats with current values as fallback
        this.updateQuickStatsFromCurrentMetrics(data);

        this.log('✅ Detailed metrics with speed update completed');
    }

    /**
     * Handle system status update
     */
    handleSystemStatusUpdate(data) {
        if (!data) return;

        this.updateSystemStatusDisplay(data);
        this.log('🖥️ System status updated');
    }

    /**
     * Update metrics display in UI (SPEED SUPPORT ADDED)
     */
    updateMetricsDisplay(metrics) {
        if (!metrics) {
            this.log('❌ No metrics data provided to updateMetricsDisplay');
            return;
        }

        try {
            console.log('🔄 Updating detailed metrics display with speed data:', metrics);
            
            // CPU metrics
            if (metrics.cpu_usage !== undefined) {
                const cpuPercentage = Math.round(metrics.cpu_usage);
                this.updateElementText(this.elements.cpuValue, cpuPercentage);
                console.log('✅ CPU value updated to:', cpuPercentage);
            }

            // CPU detailed info
            if (metrics.cpu_cores !== undefined) {
                this.updateElementText(this.elements.cpuCores, metrics.cpu_cores);
                console.log('✅ CPU cores updated to:', metrics.cpu_cores);
            }
            if (metrics.cpu_frequency !== undefined) {
                const freqGHz = (metrics.cpu_frequency / 1000).toFixed(2);
                this.updateElementText(this.elements.cpuFrequency, `${freqGHz} GHz`);
                console.log('✅ CPU frequency updated to:', freqGHz + ' GHz');
            }

            // Memory metrics
            if (metrics.memory_percent !== undefined) {
                const memoryPercentage = Math.round(metrics.memory_percent);
                this.updateElementText(this.elements.memoryValue, memoryPercentage);
                console.log('✅ Memory value updated to:', memoryPercentage);
            }

            // Memory detailed info
            if (metrics.memory_used !== undefined && metrics.memory_total !== undefined) {
                const usedGB = (metrics.memory_used / (1024*1024*1024)).toFixed(1);
                const totalGB = (metrics.memory_total / (1024*1024*1024)).toFixed(1);
                this.updateElementText(this.elements.memoryUsed, `${usedGB} GB`);
                this.updateElementText(this.elements.memoryTotal, `${totalGB} GB`);
                console.log('✅ Memory details updated:', usedGB + ' / ' + totalGB + ' GB');
            }

            // Disk metrics
            if (metrics.disk_percent !== undefined) {
                const diskPercentage = Math.round(metrics.disk_percent);
                this.updateElementText(this.elements.diskValue, diskPercentage);
                console.log('✅ Disk value updated to:', diskPercentage);
            }

            // Disk detailed info
            if (metrics.disk_used !== undefined && metrics.disk_total !== undefined) {
                const usedGB = (metrics.disk_used / (1024*1024*1024)).toFixed(1);
                const totalGB = (metrics.disk_total / (1024*1024*1024)).toFixed(1);
                this.updateElementText(this.elements.diskUsed, `${usedGB} GB`);
                this.updateElementText(this.elements.diskTotal, `${totalGB} GB`);
                console.log('✅ Disk details updated:', usedGB + ' / ' + totalGB + ' GB');
            }

            // NEW: Disk I/O Speed metrics
            if (metrics.disk_read_speed_mbps !== undefined && metrics.disk_write_speed_mbps !== undefined) {
                const readSpeed = metrics.disk_read_speed_mbps.toFixed(1);
                const writeSpeed = metrics.disk_write_speed_mbps.toFixed(1);
                const totalIOSpeed = (metrics.disk_read_speed_mbps + metrics.disk_write_speed_mbps).toFixed(1);
                
                this.updateElementText(this.elements.diskIOValue, totalIOSpeed);
                this.updateElementText(this.elements.diskReadSpeed, `${readSpeed} MB/s`);
                this.updateElementText(this.elements.diskWriteSpeed, `${writeSpeed} MB/s`);
                console.log('✅ Disk I/O speed updated:', readSpeed + ' MB/s read, ' + writeSpeed + ' MB/s write');
            }

            // NEW: Network Speed metrics (ENHANCED)
            if (metrics.network_upload_speed_mbps !== undefined && metrics.network_download_speed_mbps !== undefined) {
                const uploadSpeed = metrics.network_upload_speed_mbps.toFixed(1);
                const downloadSpeed = metrics.network_download_speed_mbps.toFixed(1);
                const totalNetworkSpeed = (metrics.network_upload_speed_mbps + metrics.network_download_speed_mbps).toFixed(1);
                
                this.updateElementText(this.elements.networkValue, totalNetworkSpeed);
                this.updateElementText(this.elements.networkUploadSpeed, `${uploadSpeed} Mbps`);
                this.updateElementText(this.elements.networkDownloadSpeed, `${downloadSpeed} Mbps`);
                console.log('✅ Network speed updated:', uploadSpeed + ' Mbps upload, ' + downloadSpeed + ' Mbps download');
            }

            // Legacy network info (total bytes) - keeping for backward compatibility
            if (metrics.network_sent !== undefined && metrics.network_received !== undefined) {
                const sentMB = (metrics.network_sent / (1024*1024)).toFixed(1);
                const receivedMB = (metrics.network_received / (1024*1024)).toFixed(1);
                this.updateElementText(this.elements.networkSent, `${sentMB} MB`);
                this.updateElementText(this.elements.networkReceived, `${receivedMB} MB`);
            }

            // System information
            if (metrics.hostname) {
                this.updateElementText(this.elements.systemHostname, metrics.hostname);
            }

            // Update system details if available
            if (metrics.platform) {
                this.updateElementText(this.elements.systemPlatform, metrics.platform);
            }

            this.log('✅ Detailed metrics display updated with speed data');

        } catch (error) {
            this.log('❌ Error updating metrics display:', error);
        }
    }

    /**
     * Update moving averages for Quick Stats
     */
    updateMovingAverages(metrics) {
        if (!metrics) return;

        try {
            // Add new samples
            if (metrics.cpu_usage !== undefined) {
                this.movingAverages.cpu.push(metrics.cpu_usage);
                if (this.movingAverages.cpu.length > this.movingAverages.maxSamples) {
                    this.movingAverages.cpu.shift();
                }
            }

            if (metrics.memory_percent !== undefined) {
                this.movingAverages.memory.push(metrics.memory_percent);
                if (this.movingAverages.memory.length > this.movingAverages.maxSamples) {
                    this.movingAverages.memory.shift();
                }
            }

            // Calculate and update averages if we have enough samples
            if (this.movingAverages.cpu.length >= 5) { // Wait for at least 5 samples
                const avgCpu = this.movingAverages.cpu.reduce((a, b) => a + b, 0) / this.movingAverages.cpu.length;
                this.updateElementText(this.elements.avgCpu, `${avgCpu.toFixed(1)}%`);
            }

            if (this.movingAverages.memory.length >= 5) { // Wait for at least 5 samples
                const avgMemory = this.movingAverages.memory.reduce((a, b) => a + b, 0) / this.movingAverages.memory.length;
                this.updateElementText(this.elements.avgMemory, `${avgMemory.toFixed(1)}%`);
            }

        } catch (error) {
            this.log('❌ Error updating moving averages:', error);
        }
    }

    /**
     * Update Quick Stats from current metrics (only as emergency fallback)
     */
    updateQuickStatsFromCurrentMetrics(metrics) {
        if (!metrics) return;

        try {
            // Only update if we don't have enough samples for moving averages yet
            if (this.movingAverages.cpu.length < 5) {
                if (metrics.cpu_usage !== undefined) {
                    this.updateElementText(this.elements.avgCpu, `${metrics.cpu_usage.toFixed(1)}%`);
                }
            }

            if (this.movingAverages.memory.length < 5) {
                if (metrics.memory_percent !== undefined) {
                    this.updateElementText(this.elements.avgMemory, `${metrics.memory_percent.toFixed(1)}%`);
                }
            }

            // Total hosts is always 1 for single-host setup
            this.updateElementText(this.elements.totalHosts, '1');

        } catch (error) {
            this.log('❌ Error updating quick stats from current metrics:', error);
        }
    }

    /**
     * Update system details display
     */
    updateSystemDetailsDisplay(systemInfo) {
        if (!systemInfo) return;

        try {
            this.updateElementText(this.elements.systemPlatform, systemInfo.platform || 'Unknown');
            this.updateElementText(this.elements.systemArch, systemInfo.architecture || 'Unknown');
            this.updateElementText(this.elements.systemUptime, systemInfo.uptime || 'Unknown');
            
            this.log('🖥️ System details display updated');
        } catch (error) {
            this.log('❌ Error updating system details display:', error);
        }
    }

    /**
     * Update system status display
     */
    updateSystemStatusDisplay(statusData) {
        if (!statusData || !this.elements.systemStatus) return;

        try {
            // Handle both single object and array formats
            const statusArray = Array.isArray(statusData) ? statusData : [statusData];
            
            const statusHTML = statusArray.map(status => `
                <div class="status-item ${status.status}">
                    <div class="status-icon">
                        <i class="fas fa-${this.getStatusIcon(status.status)}"></i>
                    </div>
                    <div class="status-details">
                        <div class="status-hostname">${status.hostname}</div>
                        <div class="status-metrics">
                            CPU: ${(status.cpu_usage || 0).toFixed(1)}% | 
                            Memory: ${(status.memory_usage || status.memory_percent || 0).toFixed(1)}% | 
                            Disk: ${(status.disk_usage || status.disk_percent || 0).toFixed(1)}%
                        </div>
                        <div class="status-time">${this.formatTime(status.timestamp)}</div>
                    </div>
                    <div class="status-badge ${status.status}">${status.status}</div>
                </div>
            `).join('');

            this.elements.systemStatus.innerHTML = statusHTML;
        } catch (error) {
            this.log('❌ Error updating system status display:', error);
        }
    }

    /**
     * Update stats display
     */
    updateStatsDisplay(stats) {
        if (!stats) return;

        try {
            // Total metrics - always show this
            this.updateElementText(this.elements.totalMetrics, this.formatNumber(stats.total_metrics || 0));
            
            // Total hosts - prefer API data, fallback to 1
            if (stats.total_hosts !== undefined) {
                this.updateElementText(this.elements.totalHosts, stats.total_hosts);
            } else {
                this.updateElementText(this.elements.totalHosts, '1');
            }
            
            // Average CPU usage - prefer API calculated averages
            if (stats.avg_cpu_usage !== undefined && stats.avg_cpu_usage !== null && stats.avg_cpu_usage > 0) {
                this.updateElementText(this.elements.avgCpu, `${stats.avg_cpu_usage.toFixed(1)}%`);
                this.hasApiAverages = true; // Mark that we have API averages
            } else if (this.currentMetrics && this.currentMetrics.cpu_usage !== undefined) {
                // Only use current value as fallback if no API average available
                this.updateElementText(this.elements.avgCpu, `${this.currentMetrics.cpu_usage.toFixed(1)}%`);
            }
            
            // Average Memory usage - prefer API calculated averages
            if (stats.avg_memory_usage !== undefined && stats.avg_memory_usage !== null && stats.avg_memory_usage > 0) {
                this.updateElementText(this.elements.avgMemory, `${stats.avg_memory_usage.toFixed(1)}%`);
                this.hasApiAverages = true; // Mark that we have API averages
            } else if (this.currentMetrics && this.currentMetrics.memory_percent !== undefined) {
                // Only use current value as fallback if no API average available
                this.updateElementText(this.elements.avgMemory, `${this.currentMetrics.memory_percent.toFixed(1)}%`);
            }

            this.log('📊 Stats display updated (using averages):', {
                cpu_avg: stats.avg_cpu_usage,
                memory_avg: stats.avg_memory_usage,
                hosts: stats.total_hosts
            });
        } catch (error) {
            this.log('❌ Error updating stats display:', error);
        }
    }

    /**
     * Update connection status indicator
     */
    updateConnectionStatus(status, message) {
        if (!this.elements.wsStatus) return;

        const statusElement = this.elements.wsStatus;
        const statusText = statusElement.querySelector('span');

        // Remove existing status classes
        statusElement.classList.remove('connected', 'disconnected', 'connecting', 'error');
        
        // Add new status class
        statusElement.classList.add(status);

        // Update status text
        if (statusText) {
            statusText.textContent = message;
        }

        // Update client count if available
        if (this.websocket && this.elements.clientCount) {
            const wsStats = this.websocket.getStats();
            // Note: Client count would come from server, this is placeholder
            this.elements.clientCount.textContent = '1'; // Current client
        }

        this.log(`🔗 Connection status updated: ${status} - ${message}`);
    }

    /**
     * Change time range for historical data (DÜZELTİLMİŞ VERSİYON)
     */
    async changeTimeRange(range) {
        try {
            // Update active button
            this.elements.timeButtons.forEach(btn => btn.classList.remove('active'));
            const activeButton = document.querySelector(`[data-range="${range}"]`);
            if (activeButton) {
                activeButton.classList.add('active');
            }

            // Load new historical data
            await this.loadHistoricalData(range);
            
            this.log(`📅 Time range changed to: ${range}`);
        } catch (error) {
            this.log('❌ Error changing time range:', error);
        }
    }

    /**
     * Show alert banner
     */
    showAlert(message, type = 'info') {
        if (!this.elements.alertBanner || !this.elements.alertMessage) return;

        this.elements.alertMessage.textContent = message;
        this.elements.alertBanner.className = `alert-banner ${type}`;
        this.elements.alertBanner.style.display = 'block';

        // Auto-hide alert after timeout
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
     * Hide alert banner
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
     * Show dashboard and hide loading screen
     */
    showDashboard() {
        if (this.elements.loadingScreen) {
            this.elements.loadingScreen.style.display = 'none';
        }
        
        if (this.elements.dashboard) {
            this.elements.dashboard.style.display = 'block';
        }

        this.log('🎨 Dashboard displayed');
    }

    /**
     * Show error state
     */
    showError(message) {
        this.showAlert(message, 'error');
        this.stats.errors++;
        this.stats.lastError = new Date();
    }

    /**
     * Handle keyboard shortcuts
     */
    handleKeyboardShortcuts(event) {
        // Escape key closes alerts
        if (event.key === 'Escape') {
            this.hideAlert();
        }

        // Ctrl+R or F5 - refresh data
        if ((event.ctrlKey && event.key === 'r') || event.key === 'F5') {
            event.preventDefault();
            this.refreshData();
        }

        // Ctrl+D - toggle debug mode
        if (event.ctrlKey && event.key === 'd') {
            event.preventDefault();
            this.toggleDebugMode();
        }
    }

    /**
     * Refresh all data
     */
    async refreshData() {
        try {
            this.showAlert('Refreshing data...', 'info');
            await this.loadInitialData();
            this.showAlert('Data refreshed successfully!', 'success');
        } catch (error) {
            this.showAlert('Failed to refresh data', 'error');
            this.handleError(error);
        }
    }

    /**
     * Toggle debug mode
     */
    toggleDebugMode() {
        this.options.debug = !this.options.debug;
        
        if (this.websocket) {
            this.websocket.setDebug(this.options.debug);
        }

        this.showAlert(`Debug mode ${this.options.debug ? 'enabled' : 'disabled'}`, 'info');
    }

    /**
     * Pause updates (when page is hidden)
     */
    pauseUpdates() {
        if (this.isPaused) return;

        this.isPaused = true;
        this.log('⏸️ Updates paused');
    }

    /**
     * Resume updates (when page becomes visible)
     */
    resumeUpdates() {
        if (!this.isPaused) return;

        this.isPaused = false;
        this.refreshData();
        this.log('▶️ Updates resumed');
    }

    /**
     * Setup refresh intervals (ULTRA HIGH FREQUENCY FOR SPEED)
     */
    setupRefreshIntervals() {
        console.log('🚀 SETTING UP 500MS INTERVALS FOR ULTRA REAL-TIME WITH SPEED!');
        
        // Clear any existing intervals first
        if (this.statsInterval) clearInterval(this.statsInterval);
        if (this.statusInterval) clearInterval(this.statusInterval);
        if (this.metricsInterval) clearInterval(this.metricsInterval);
        
        // Ultra-aggressive polling for maximum real-time feel (500ms intervals)
        this.statsInterval = setInterval(() => {
            if (!this.isPaused) {
                console.log('📊 500ms Stats API call');
                this.log('📊 Loading system stats via API');
                this.loadSystemStats();
            }
        }, 500); // 500ms for ultra real-time updates

        // System status refresh (every 500ms for ultra real-time updates)
        this.statusInterval = setInterval(() => {
            if (!this.isPaused) {
                console.log('🖥️ 500ms Status API call');
                this.log('🖥️ Loading system status via API');
                this.loadSystemStatus();
            }
        }, 500); // 500ms for ultra real-time updates

        // Current metrics refresh (every 500ms)
        this.metricsInterval = setInterval(() => {
            if (!this.isPaused) {
                console.log('📈 500ms Metrics API call with speed data');
                this.loadCurrentMetrics();
            }
        }, 500); // 500ms for ultra real-time updates

        // Update last seen timestamps (every second)
        this.timestampInterval = setInterval(() => {
            this.updateLastUpdateTime();
        }, 1000);
        
        console.log('✅ 500MS INTERVALS SET UP SUCCESSFULLY WITH SPEED SUPPORT!');
    }

    /**
     * Initialize tooltips
     */
    initializeTooltips() {
        // Add tooltip functionality if needed
        this.log('💡 Tooltips initialized');
    }

    /**
     * Setup mobile menu
     */
    setupMobileMenu() {
        if (this.elements.mobileMenuToggle) {
            this.elements.mobileMenuToggle.addEventListener('click', () => {
                // Mobile menu functionality for future use
                this.log('📱 Mobile menu toggled');
            });
        }
    }

    /**
     * Get status icon based on status
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
     * Update element text safely
     */
    updateElementText(element, text) {
        if (element && text !== undefined) {
            console.log(`🔧 Updating ${element.id} with value: ${text}`);
            element.textContent = text;
        } else {
            console.error(`❌ Failed to update element: ${element ? element.id : 'null'} with text: ${text}`);
        }
    }

    /**
     * Update last update time display
     */
    updateLastUpdateTime() {
        if (this.elements.lastUpdate && this.lastUpdateTime) {
            const timeAgo = this.formatTimeAgo(this.lastUpdateTime);
            this.updateElementText(this.elements.lastUpdate, timeAgo);
        }
    }

    /**
     * Format time ago string
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
     * Format time string
     */
    formatTime(timestamp) {
        if (!timestamp) return 'Never';
        
        const date = new Date(timestamp);
        return date.toLocaleTimeString();
    }

    /**
     * Format number with commas
     */
    formatNumber(num) {
        return num.toLocaleString();
    }

    /**
     * Handle errors
     */
    handleError(error) {
        this.stats.errors++;
        this.stats.lastError = new Date();
        this.log('❌ Error handled:', error);
        
        // In production, you might want to send errors to a monitoring service
        if (this.options.debug) {
            console.error('Dashboard Error:', error);
        }
    }

    /**
     * Get dashboard statistics
     */
    getStats() {
        return {
            ...this.stats,
            isInitialized: this.isInitialized,
            isPaused: this.isPaused,
            lastUpdateTime: this.lastUpdateTime,
            websocketStats: this.websocket ? this.websocket.getStats() : null,
            chartStats: this.chartManager ? this.chartManager.getStats() : null
        };
    }

    /**
     * Cleanup resources
     */
    cleanup() {
        this.log('🧹 Cleaning up dashboard...');

        // Disconnect WebSocket
        if (this.websocket) {
            this.websocket.destroy();
            this.websocket = null;
        }

        // Destroy charts
        if (this.chartManager) {
            this.chartManager.destroyCharts();
            this.chartManager = null;
        }

        // Clear intervals
        if (this.statsInterval) {
            clearInterval(this.statsInterval);
            this.statsInterval = null;
        }

        if (this.statusInterval) {
            clearInterval(this.statusInterval);
            this.statusInterval = null;
        }

        if (this.metricsInterval) {
            clearInterval(this.metricsInterval);
            this.metricsInterval = null;
        }

        if (this.timestampInterval) {
            clearInterval(this.timestampInterval);
            this.timestampInterval = null;
        }

        // Clear timeouts
        if (this.alertTimeout) {
            clearTimeout(this.alertTimeout);
            this.alertTimeout = null;
        }

        // Clear event handlers
        this.eventHandlers.clear();

        // Clear moving averages
        this.movingAverages.cpu = [];
        this.movingAverages.memory = [];

        this.isInitialized = false;
        this.log('✅ Dashboard cleanup complete');
    }

    /**
     * Log messages (if debug enabled)
     */
    log(...args) {
        if (this.options.debug) {
            console.log('[Dashboard]', ...args);
        }
    }
}

// Export for use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = DashboardApp;
} else if (typeof window !== 'undefined') {
    window.DashboardApp = DashboardApp;
}

/**
 * Utility functions for dashboard
 */
const DashboardUtils = {
    /**
     * Format bytes to human readable format
     */
    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    },

    /**
     * Format duration to human readable format
     */
    formatDuration(seconds) {
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;

        if (days > 0) return `${days}d ${hours}h`;
        if (hours > 0) return `${hours}h ${minutes}m`;
        if (minutes > 0) return `${minutes}m ${secs}s`;
        return `${secs}s`;
    },

    /**
     * Debounce function calls
     */
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    /**
     * Throttle function calls
     */
    throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }
};

// Make utils available globally
if (typeof window !== 'undefined') {
    window.DashboardUtils = DashboardUtils;
}