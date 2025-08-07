/**
 * Enhanced GoDash Dashboard - With Alert System Integration
 */

class Dashboard {
    constructor(options = {}) {
        this.options = {
            wsUrl: options.wsUrl || this.getWebSocketURL(),
            apiUrl: options.apiUrl || '/api/v1',
            updateInterval: 1000,
            reconnectAttempts: 10,
            chartUpdateAnimation: true,
            alertTimeout: 5000,
            debug: window.location.hostname === 'localhost',
            alertsEnabled: options.alertsEnabled || false,
            ...options
        };

        // Application state
        this.isInitialized = false;
        this.isPaused = false;
        this.lastUpdateTime = null;
        this.connectionAttempts = 0;
        
        // Moving averages
        this.averages = {
            cpu: { current: 0, count: 0 },
            memory: { current: 0, count: 0 },
            diskIO: { current: 0, count: 0 },
            network: { current: 0, count: 0 },
            alpha: 0.1
        };

        // Component instances
        this.websocket = null;
        this.chartManager = null;
        this.alertManager = null;
        this.elements = {};
        
        // Data storage
        this.currentMetrics = null;
        this.stats = {
            totalUpdates: 0,
            errors: 0,
            connectionTime: null
        };

        this.log('Dashboard initialized:', this.options);
    }

    /**
     * Get WebSocket URL based on current location
     */
    getWebSocketURL() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        return `${protocol}//${host}/ws`;
    }

    /**
     * Initialize the dashboard application
     */
    async init() {
        if (this.isInitialized) return;

        try {
            this.log('🚀 Initializing Dashboard...');

            this.cacheElements();
            this.initializeMetrics();
            await this.initializeChartManager();
            this.initializeWebSocket();
            
            // Initialize Alert Manager if enabled
            if (this.options.alertsEnabled) {
                await this.initializeAlertManager();
            }
            
            this.setupEventListeners();
            await this.loadInitialData();
            this.showDashboard();

            this.isInitialized = true;
            this.stats.connectionTime = new Date();
            
            this.log('✅ Dashboard initialized successfully');
            this.updateConnectionStatus('connected', 'Connected');

        } catch (error) {
            this.log('❌ Error initializing dashboard:', error);
            this.handleError(error);
            this.showError('Failed to initialize dashboard');
            this.showDashboard();
        }
    }

    /**
     * Initialize Alert Manager
     */
    async initializeAlertManager() {
        try {
            if (typeof AlertManager === 'undefined') {
                console.warn('AlertManager not found, alerts disabled');
                return;
            }

            this.alertManager = new AlertManager(this.options.apiUrl);
            
            // Set up alert notification handling
            if (this.alertManager.handleAlertNotification) {
                this.handleAlertNotification = this.alertManager.handleAlertNotification.bind(this.alertManager);
            }

            this.log('✅ Alert Manager initialized');
        } catch (error) {
            this.log('❌ Error initializing Alert Manager:', error);
        }
    }

    /**
     * Cache DOM elements
     */
    cacheElements() {
        this.elements = {
            // Loading and main containers
            loadingOverlay: document.getElementById('loadingOverlay'),
            container: document.querySelector('.container'),

            // Connection status
            connectionStatus: document.getElementById('connectionStatus'),
            wsStatus: document.getElementById('wsStatus'),
            statusDot: document.getElementById('statusDot'),
            statusText: document.getElementById('statusText'),

            // Metric values
            cpuValue: document.getElementById('cpuValue'),
            memoryValue: document.getElementById('memoryValue'),
            diskValue: document.getElementById('diskValue'),
            networkSpeed: document.getElementById('networkSpeed'),
            temperatureValue: document.getElementById('temperatureValue'),
            processValue: document.getElementById('processValue'),

            // Metric details
            cpuCores: document.getElementById('cpuCores'),
            cpuFreq: document.getElementById('cpuFreq'),
            loadAvg: document.getElementById('loadAvg'),
            memoryUsed: document.getElementById('memoryUsed'),
            memoryTotal: document.getElementById('memoryTotal'),
            memoryAvailable: document.getElementById('memoryAvailable'),
            diskUsed: document.getElementById('diskUsed'),
            diskTotal: document.getElementById('diskTotal'),
            diskFree: document.getElementById('diskFree'),
            networkUpload: document.getElementById('networkUpload'),
            networkDownload: document.getElementById('networkDownload'),
            networkSent: document.getElementById('networkSent'),
            
            // Temperature & Process
            currentTemp: document.getElementById('currentTemp'),
            tempStatus: document.getElementById('tempStatus'),
            runningProcesses: document.getElementById('runningProcesses'),
            sleepingProcesses: document.getElementById('sleepingProcesses'),
            zombieProcesses: document.getElementById('zombieProcesses'),

            // System info
            hostname: document.getElementById('hostname'),
            platform: document.getElementById('platform'),
            uptime: document.getElementById('uptime'),
            processCount: document.getElementById('processCount'),

            // Time range selector
            timeRange: document.getElementById('timeRange')
        };

        this.log('DOM elements cached');
    }

    /**
     * Initialize Metrics with default values
     */
    initializeMetrics() {
        // Set initial values for all metrics
        this.updateElementText(this.elements.cpuValue, '0%');
        this.updateElementText(this.elements.memoryValue, '0%');
        this.updateElementText(this.elements.diskValue, '0%');
        this.updateElementText(this.elements.networkSpeed, '0 Mbps');
        this.updateElementText(this.elements.temperatureValue, '0°C');
        this.updateElementText(this.elements.processValue, '0');
        
        this.log('✅ Initial metrics values set');
    }

    /**
     * Initialize chart manager
     */
    async initializeChartManager() {
        try {
            this.chartManager = new ChartManager({
                maxDataPoints: 50,
                animationDuration: this.options.chartUpdateAnimation ? 300 : 0,
                theme: 'dark'
            });

            await new Promise((resolve) => {
                const checkInitialized = () => {
                    if (this.chartManager.isInitialized) {
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
        try {
            this.websocket = new WebSocketClient({
                url: this.options.wsUrl,
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
            this.connectionAttempts = 0;
            this.updateConnectionStatus('connected', 'Connected');
            this.hideNotification();
            this.websocket.subscribe(['metrics', 'system_status', 'alert_triggered']);
        });

        this.websocket.on('disconnect', (event) => {
            this.updateConnectionStatus('disconnected', 'Disconnected');
            if (event.code !== 1000) {
                this.showNotification('Connection lost. Attempting to reconnect...', 'warning', 0);
            }
        });

        this.websocket.on('reconnect', (event) => {
            this.updateConnectionStatus('connected', 'Reconnected');
            this.showNotification('Connection restored!', 'success', 3000);
        });

        this.websocket.on('error', (error) => {
            this.connectionAttempts++;
            
            if (this.connectionAttempts > 3) {
                this.updateConnectionStatus('error', 'Connection Error');
                this.showNotification('Unable to connect to server', 'error', 0);
            } else {
                this.updateConnectionStatus('connecting', 'Connecting...');
            }
            
            this.handleError(error);
        });

        this.websocket.on('metrics', (data) => {
            this.handleMetricsUpdate(data);
        });

        this.websocket.on('system_status', (data) => {
            this.handleSystemStatusUpdate(data);
        });

        // Alert notification handler
        this.websocket.on('alert_triggered', (data) => {
            this.handleAlertNotification(data);
        });

        this.websocket.connect();
    }

    /**
     * Setup event listeners
     */
    setupEventListeners() {
        // Time range selector
        if (this.elements.timeRange) {
            this.elements.timeRange.addEventListener('change', (e) => {
                this.changeTimeRange(e.target.value);
            });
        }

        // Window resize handler
        window.addEventListener('resize', () => {
            if (this.chartManager) {
                this.chartManager.resizeCharts();
            }
        });

        // Page visibility change
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.isPaused = true;
            } else {
                this.isPaused = false;
                this.refreshData();
            }
        });

        this.log('👂 Event listeners setup complete');
    }

    /**
     * Load initial data
     */
    async loadInitialData() {
        try {
            await this.loadCurrentMetrics();
            await this.loadSystemStatus();
            await this.loadTopProcesses();
            await this.loadHistoricalData();
            this.setupRefreshIntervals();
        } catch (error) {
            this.handleError(error);
        }
    }

    /**
     * Load current metrics
     */
    async loadCurrentMetrics() {
        try {
            const response = await fetch(`${this.options.apiUrl}/metrics/current`);
            const result = await response.json();

            if (result.success && result.data) {
                this.updateMetricsDisplay(result.data);
                
                if (this.chartManager) {
                    this.chartManager.updateMetrics(result.data);
                }
                
                this.currentMetrics = result.data;
                this.updateConnectionStatus();
            }
        } catch (error) {
            this.log('Error loading current metrics:', error);
        }
    }

    /**
     * Load system status
     */
    async loadSystemStatus() {
        try {
            const response = await fetch('/health');
            if (response.ok) {
                const result = await response.json();
                if (result.success || result.status === 'healthy') {
                    this.updateSystemStatusDisplay(result.data || result);
                }
            }
        } catch (error) {
            this.log('Error loading system status:', error);
        }
    }

    /**
     * Load top processes
     */
    async loadTopProcesses() {
        try {
            // Mock data for now - replace with real API when available
            const mockProcesses = [
                { name: 'chrome.exe', cpu: Math.random() * 30 + 10 },
                { name: 'node.exe', cpu: Math.random() * 20 + 5 },
                { name: 'vscode.exe', cpu: Math.random() * 15 + 3 },
                { name: 'firefox.exe', cpu: Math.random() * 25 + 8 },
                { name: 'docker.exe', cpu: Math.random() * 10 + 2 }
            ].sort((a, b) => b.cpu - a.cpu);

            this.updateTopProcessesDisplay(mockProcesses);
        } catch (error) {
            this.log('Error loading top processes:', error);
        }
    }

    /**
     * Load historical data
     */
    async loadHistoricalData(timeRange = '24h') {
        try {
            const response = await fetch(`${this.options.apiUrl}/metrics/trends?range=${timeRange}`);
            const result = await response.json();

            if (result.success && result.data && this.chartManager) {
                this.chartManager.updateTrendsChartWithHistoricalData(result.data);
            }
        } catch (error) {
            this.log('❌ Error loading historical data:', error);
        }
    }

    /**
     * Handle metrics update from WebSocket
     */
    handleMetricsUpdate(data) {
        if (!data) return;

        this.stats.totalUpdates++;
        this.lastUpdateTime = new Date();

        this.updateMetricsDisplay(data);

        if (this.chartManager) {
            this.chartManager.updateMetrics(data);
        }

        this.updateLastUpdateTime();
        this.currentMetrics = data;
        this.updateMovingAverages(data);
        this.updateConnectionStatus();
    }

    /**
     * Handle system status update
     */
    handleSystemStatusUpdate(data) {
        if (!data) return;
        this.updateSystemStatusDisplay(data);
    }

    /**
     * Handle alert notifications
     */
    handleAlertNotification(alertData) {
        if (!alertData) return;

        if (alertData.type === 'alert_triggered' || alertData.alert_id) {
            const alert = alertData.alert || alertData;
            const severity = alert.severity || 'warning';
            
            // Show notification
            const message = `🚨 ${severity.toUpperCase()}: ${alert.alert_name || alert.name} on ${alert.hostname || 'system'}`;
            this.showNotification(message, severity === 'critical' ? 'error' : 'warning', 10000);
            
            // Update alert badge if alert manager exists
            if (this.alertManager && this.alertManager.updateAlertBadge) {
                this.alertManager.loadAlertStats();
            }
            
            this.log('🚨 Alert notification handled:', alert);
        }
    }

    /**
     * Update metrics display
     */
    updateMetricsDisplay(metrics) {
        if (!metrics) return;

        try {
            // Basic metrics with proper rounding
            if (metrics.cpu_usage !== undefined) {
                this.updateElementText(this.elements.cpuValue, Math.round(metrics.cpu_usage));
            }

            if (metrics.memory_percent !== undefined) {
                this.updateElementText(this.elements.memoryValue, Math.round(metrics.memory_percent));
            }

            if (metrics.disk_percent !== undefined) {
                this.updateElementText(this.elements.diskValue, Math.round(metrics.disk_percent));
            }

            // Network speed (total upload + download)
            if (metrics.network_upload_speed_mbps !== undefined && metrics.network_download_speed_mbps !== undefined) {
                const totalSpeed = (metrics.network_upload_speed_mbps + metrics.network_download_speed_mbps).toFixed(1);
                this.updateElementText(this.elements.networkSpeed, totalSpeed);
                this.updateElementText(this.elements.networkUpload, metrics.network_upload_speed_mbps.toFixed(1));
                this.updateElementText(this.elements.networkDownload, metrics.network_download_speed_mbps.toFixed(1));
            }

            // Temperature
            const temperature = metrics.cpu_temperature_c || metrics.simulated_temperature || 45;
            this.updateElementText(this.elements.temperatureValue, temperature.toFixed(1));
            this.updateElementText(this.elements.currentTemp, `${temperature.toFixed(1)}°C`);
            
            // Temperature status
            let tempStatus = 'Normal';
            if (temperature > 75) tempStatus = 'Hot';
            else if (temperature > 65) tempStatus = 'Warm';
            else if (temperature > 55) tempStatus = 'Moderate';
            this.updateElementText(this.elements.tempStatus, tempStatus);

            // Process count
            if (metrics.processes) {
                const totalProcesses = (metrics.processes.running_processes || 0) + 
                                     (metrics.processes.stopped_processes || 0) + 
                                     (metrics.processes.zombie_processes || 0);
                this.updateElementText(this.elements.processValue, totalProcesses);
                this.updateElementText(this.elements.runningProcesses, metrics.processes.running_processes || 0);
                this.updateElementText(this.elements.sleepingProcesses, metrics.processes.stopped_processes || 0);
                this.updateElementText(this.elements.zombieProcesses, metrics.processes.zombie_processes || 0);
            }

            // CPU details
            if (metrics.cpu_cores !== undefined) {
                this.updateElementText(this.elements.cpuCores, metrics.cpu_cores);
            }
            if (metrics.cpu_frequency !== undefined) {
                const freqGHz = (metrics.cpu_frequency / 1000).toFixed(2);
                this.updateElementText(this.elements.cpuFreq, `${freqGHz} GHz`);
            }
            if (metrics.cpu_load_avg && metrics.cpu_load_avg.length > 0) {
                this.updateElementText(this.elements.loadAvg, metrics.cpu_load_avg[0].toFixed(2));
            }

            // Memory details
            if (metrics.memory_used !== undefined && metrics.memory_total !== undefined) {
                const usedGB = (metrics.memory_used / (1024*1024*1024)).toFixed(1);
                const totalGB = (metrics.memory_total / (1024*1024*1024)).toFixed(1);
                const availableGB = ((metrics.memory_total - metrics.memory_used) / (1024*1024*1024)).toFixed(1);
                
                this.updateElementText(this.elements.memoryUsed, `${usedGB} GB`);
                this.updateElementText(this.elements.memoryTotal, `${totalGB} GB`);
                this.updateElementText(this.elements.memoryAvailable, `${availableGB} GB`);
            }

            // Disk details
            if (metrics.disk_used !== undefined && metrics.disk_total !== undefined && metrics.disk_free !== undefined) {
                const usedGB = (metrics.disk_used / (1024*1024*1024)).toFixed(1);
                const totalGB = (metrics.disk_total / (1024*1024*1024)).toFixed(1);
                const freeGB = (metrics.disk_free / (1024*1024*1024)).toFixed(1);
                
                this.updateElementText(this.elements.diskUsed, `${usedGB} GB`);
                this.updateElementText(this.elements.diskTotal, `${totalGB} GB`);
                this.updateElementText(this.elements.diskFree, `${freeGB} GB`);
            }

            // Network sent/received
            if (metrics.network_sent !== undefined && metrics.network_received !== undefined) {
                const sentMB = (metrics.network_sent / (1024*1024)).toFixed(1);
                const receivedMB = (metrics.network_received / (1024*1024)).toFixed(1);
                this.updateElementText(this.elements.networkSent, `${sentMB} MB`);
            }

            // System info
            if (metrics.hostname) {
                this.updateElementText(this.elements.hostname, metrics.hostname);
            }

            // Uptime handling
            if (metrics.uptime_seconds !== undefined) {
                this.updateElementText(this.elements.uptime, this.formatUptime(metrics.uptime_seconds));
            } else if (metrics.uptime !== undefined) {
                if (typeof metrics.uptime === 'number') {
                    let uptimeInSeconds = metrics.uptime;
                    
                    // Handle different time formats
                    if (uptimeInSeconds > 1000000000000) {
                        uptimeInSeconds = Math.floor(uptimeInSeconds / 1000000000);
                    } else if (uptimeInSeconds > 1000000000) {
                        uptimeInSeconds = Math.floor(uptimeInSeconds / 1000);
                    }
                    
                    this.updateElementText(this.elements.uptime, this.formatUptime(uptimeInSeconds));
                } else {
                    this.updateElementText(this.elements.uptime, metrics.uptime);
                }
            }

            // Process count for system info
            if (metrics.processes) {
                const totalProcesses = (metrics.processes.running_processes || 0) + 
                                     (metrics.processes.stopped_processes || 0) + 
                                     (metrics.processes.zombie_processes || 0);
                this.updateElementText(this.elements.processCount, totalProcesses);
            }

        } catch (error) {
            this.log('❌ Error updating metrics display:', error);
        }
    }

    /**
     * Update moving averages
     */
    updateMovingAverages(metrics) {
        if (!metrics) return;

        try {
            const alpha = this.averages.alpha;
            
            // CPU smoothing
            if (metrics.cpu_usage !== undefined && !isNaN(metrics.cpu_usage)) {
                const cpuValue = Math.max(0, Math.min(100, parseFloat(metrics.cpu_usage)));
                
                if (this.averages.cpu.count === 0) {
                    this.averages.cpu.current = cpuValue;
                } else {
                    this.averages.cpu.current = alpha * cpuValue + (1 - alpha) * this.averages.cpu.current;
                }
                this.averages.cpu.count++;
            }

            // Memory smoothing
            if (metrics.memory_percent !== undefined && !isNaN(metrics.memory_percent)) {
                const memoryValue = Math.max(0, Math.min(100, parseFloat(metrics.memory_percent)));
                
                if (this.averages.memory.count === 0) {
                    this.averages.memory.current = memoryValue;
                } else {
                    this.averages.memory.current = alpha * memoryValue + (1 - alpha) * this.averages.memory.current;
                }
                this.averages.memory.count++;
            }

            // Disk I/O smoothing
            if (metrics.disk_read_speed_mbps !== undefined && metrics.disk_write_speed_mbps !== undefined) {
                const readSpeed = Math.max(0, parseFloat(metrics.disk_read_speed_mbps) || 0);
                const writeSpeed = Math.max(0, parseFloat(metrics.disk_write_speed_mbps) || 0);
                const totalDiskIO = readSpeed + writeSpeed;
                
                if (this.averages.diskIO.count === 0) {
                    this.averages.diskIO.current = totalDiskIO;
                } else {
                    this.averages.diskIO.current = alpha * totalDiskIO + (1 - alpha) * this.averages.diskIO.current;
                }
                this.averages.diskIO.count++;
            }

            // Network smoothing
            if (metrics.network_upload_speed_mbps !== undefined && metrics.network_download_speed_mbps !== undefined) {
                const uploadSpeed = Math.max(0, parseFloat(metrics.network_upload_speed_mbps) || 0);
                const downloadSpeed = Math.max(0, parseFloat(metrics.network_download_speed_mbps) || 0);
                const totalNetwork = uploadSpeed + downloadSpeed;
                
                if (this.averages.network.count === 0) {
                    this.averages.network.current = totalNetwork;
                } else {
                    this.averages.network.current = alpha * totalNetwork + (1 - alpha) * this.averages.network.current;
                }
                this.averages.network.count++;
            }

        } catch (error) {
            this.log('❌ Error updating moving averages:', error);
        }
    }

    /**
     * Update connection status
     */
    updateConnectionStatus() {
        if (!this.currentMetrics) return;

        try {
            // Update total metrics counter
            this.stats.totalUpdates++;
            
            // Simple status update - we don't need complex quick stats in the new template
            this.log(`📊 Updated metrics count: ${this.stats.totalUpdates}`);
        } catch (error) {
            this.log('Error updating connection status:', error);
        }
    }

    /**
     * Update system status display
     */
    updateSystemStatusDisplay(statusData) {
        if (!statusData) return;

        try {
            // Simple status update - most elements don't exist in new template
            this.log(`🖥️  System status updated: ${statusData.hostname || 'localhost'}`);
        } catch (error) {
            this.log('Error updating system status display:', error);
        }
    }

    /**
     * Update top processes display
     */
    updateTopProcessesDisplay(processes) {
        if (!processes) return;

        try {
            // Top processes element doesn't exist in new template - just log
            this.log(`🔄 Processes updated: ${processes.length} processes`);
        } catch (error) {
            this.log('Error updating top processes display:', error);
        }
    }

    /**
     * Update connection status indicator
     */
    updateConnectionStatus(status, message) {
        if (!this.elements.connectionStatus) return;

        try {
            const statusElement = this.elements.connectionStatus;
            
            // Clear all status classes
            statusElement.classList.remove('connected', 'disconnected', 'connecting', 'error');
            statusElement.classList.add(status);

            // Update status text if elements exist
            if (this.elements.statusText) {
                this.elements.statusText.textContent = message;
            }

            // Update status dot color
            if (this.elements.statusDot) {
                this.elements.statusDot.className = 'status-dot';
                this.elements.statusDot.classList.add(status);
            }

            // Update WS status if element exists
            if (this.elements.wsStatus) {
                this.updateElementText(this.elements.wsStatus, status === 'connected' ? 'Connected' : 'Disconnected');
            }
        } catch (error) {
            this.log('Error updating connection status:', error);
        }
    }

    /**
     * Change time range for historical data
     */
    async changeTimeRange(range) {
        try {
            await this.loadHistoricalData(range);
        } catch (error) {
            this.log('❌ Error changing time range:', error);
        }
    }

    /**
     * Setup refresh intervals
     */
    setupRefreshIntervals() {
        // Clear existing intervals
        if (this.metricsInterval) clearInterval(this.metricsInterval);
        if (this.timestampInterval) clearInterval(this.timestampInterval);
        if (this.processInterval) clearInterval(this.processInterval);
        
        // Metrics refresh (fallback if WebSocket fails)
        this.metricsInterval = setInterval(() => {
            if (!this.isPaused && (!this.websocket || !this.websocket.isConnected)) {
                this.loadCurrentMetrics();
            }
        }, 5000);

        // Process refresh
        this.processInterval = setInterval(() => {
            if (!this.isPaused) {
                this.loadTopProcesses();
            }
        }, 10000);

        // Timestamp update
        this.timestampInterval = setInterval(() => {
            this.updateLastUpdateTime();
        }, 1000);
    }

    /**
     * Show notification
     */
    showNotification(message, type = 'info', duration = 5000) {
        // Get or create notification container
        let container = document.getElementById('notifications');
        if (!container) {
            container = document.createElement('div');
            container.id = 'notifications';
            container.className = 'notification-container';
            document.body.appendChild(container);
        }

        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <div class="notification-content">
                <span class="notification-message">${this.escapeHtml(message)}</span>
                <button class="notification-close" onclick="this.parentElement.parentElement.remove()">&times;</button>
            </div>
        `;

        container.appendChild(notification);

        // Auto-remove after duration (0 = permanent)
        if (duration > 0) {
            setTimeout(() => {
                if (notification.parentElement) {
                    notification.remove();
                }
            }, duration);
        }
    }

    /**
     * Hide notification
     */
    hideNotification() {
        const container = document.getElementById('notifications');
        if (container) {
            container.innerHTML = '';
        }
    }

    /**
     * Show dashboard and hide loading screen
     */
    showDashboard() {
        if (this.elements.loadingOverlay) {
            this.elements.loadingOverlay.style.display = 'none';
        }
        
        if (this.elements.container) {
            this.elements.container.style.display = 'block';
        }
    }

    /**
     * Show error state
     */
    showError(message) {
        this.showNotification(message, 'error', 0);
        this.stats.errors++;
    }

    /**
     * Refresh all data
     */
    async refreshData() {
        try {
            this.showNotification('Refreshing data...', 'info', 2000);
            await this.loadInitialData();
            this.showNotification('Data refreshed successfully!', 'success', 3000);
        } catch (error) {
            this.showNotification('Failed to refresh data', 'error');
            this.handleError(error);
        }
    }

    /**
     * Update element text safely
     */
    updateElementText(element, text) {
        if (element && text !== undefined) {
            element.textContent = text;
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
     * Format uptime from seconds
     */
    formatUptime(seconds) {
        if (!seconds || seconds < 0) return 'Unknown';
        
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = Math.floor(seconds % 60);
        
        let parts = [];
        
        if (days > 0) parts.push(`${days}d`);
        if (hours > 0) parts.push(`${hours}h`);
        if (minutes > 0) parts.push(`${minutes}m`);
        if (secs > 0 || parts.length === 0) parts.push(`${secs}s`);
        
        return parts.join(' ');
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
     * Format number with commas
     */
    formatNumber(num) {
        if (typeof num !== 'number') return '0';
        return num.toLocaleString();
    }

    /**
     * Escape HTML to prevent XSS
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Handle errors
     */
    handleError(error) {
        this.stats.errors++;
        this.log('❌ Error handled:', error);
        
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
            chartStats: this.chartManager ? this.chartManager.getStats() : null,
            averages: this.averages
        };
    }

    /**
     * Cleanup resources
     */
    cleanup() {
        this.log('Cleaning up dashboard...');

        if (this.websocket) {
            this.websocket.destroy();
            this.websocket = null;
        }

        if (this.chartManager) {
            this.chartManager.destroyCharts();
            this.chartManager = null;
        }

        if (this.alertManager) {
            this.alertManager.stopAutoRefresh();
            this.alertManager = null;
        }

        [this.metricsInterval, this.timestampInterval, this.processInterval].forEach(interval => {
            if (interval) clearInterval(interval);
        });

        this.isInitialized = false;
    }

    /**
     * Log messages
     */
    log(...args) {
        if (this.options.debug) {
            console.log('[Dashboard]', ...args);
        }
    }
}

// Make globally available
if (typeof window !== 'undefined') {
    window.Dashboard = Dashboard;
    window.DashboardApp = Dashboard; // Backward compatibility
}

// Export for Node.js
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Dashboard;
}