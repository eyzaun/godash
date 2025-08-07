/**
 * Enhanced WebSocket Client - With Alert Notification Support
 */

class WebSocketClient {
    constructor(options = {}) {
        this.options = {
            url: options.url || this.getWebSocketURL(),
            reconnectInterval: 5000,
            maxReconnectAttempts: 10,
            heartbeatInterval: 5000,
            debug: options.debug || false,
            ...options
        };

        // Connection state
        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.heartbeatTimer = null;
        this.reconnectTimer = null;

        // Event handlers
        this.eventHandlers = {
            connect: [],
            disconnect: [],
            reconnect: [],
            error: [],
            message: [],
            metrics: [],
            system_status: [],
            alert_triggered: [], // NEW: Alert handlers
            alert_resolved: [],  // NEW: Alert resolved handlers
            pong: []
        };

        // Message queue for when disconnected
        this.messageQueue = [];

        // Statistics
        this.stats = {
            messagesReceived: 0,
            messagesSent: 0,
            reconnectCount: 0,
            connectionTime: null,
            lastMessageTime: null,
            alertsReceived: 0 // NEW: Alert statistics
        };

        this.log('WebSocket client initialized', this.options);
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
     * Connect to WebSocket server
     */
    connect() {
        if (this.ws && this.ws.readyState === WebSocket.CONNECTING) {
            this.log('Connection already in progress');
            return;
        }

        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.log('Already connected');
            return;
        }

        this.log('Connecting to WebSocket...', this.options.url);

        try {
            this.ws = new WebSocket(this.options.url);
            this.setupEventListeners();
        } catch (error) {
            console.error('❌ Failed to create WebSocket:', error);
            this.handleError(error);
            this.scheduleReconnect();
        }
    }

    /**
     * Disconnect from WebSocket server
     */
    disconnect() {
        this.log('Disconnecting...');
        
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }

        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        if (this.ws) {
            this.ws.close(1000, 'Manual disconnect');
            this.ws = null;
        }

        this.isConnected = false;
    }

    /**
     * Setup WebSocket event listeners
     */
    setupEventListeners() {
        this.ws.onopen = (event) => {
            this.log('Connected to WebSocket server');
            this.isConnected = true;
            
            // Debug visual feedback
            if (this.options.debug && window.location.hostname === 'localhost') {
                this.showDebugMessage('WebSocket Connected!', '#4ecdc4');
            }
            
            this.reconnectAttempts = 0;
            this.stats.connectionTime = new Date();
            if (this.stats.connectionTime) {
                this.stats.reconnectCount++;
            }

            this.startHeartbeat();
            this.processMessageQueue();
            this.trigger('connect', event);

            if (this.reconnectAttempts > 0) {
                this.trigger('reconnect', event);
            }
        };

        this.ws.onclose = (event) => {
            this.log('WebSocket connection closed', event.code, event.reason);
            this.isConnected = false;

            if (this.heartbeatTimer) {
                clearInterval(this.heartbeatTimer);
                this.heartbeatTimer = null;
            }

            this.trigger('disconnect', event);

            if (event.code !== 1000 && this.reconnectAttempts < this.options.maxReconnectAttempts) {
                this.scheduleReconnect();
            }
        };

        this.ws.onerror = (event) => {
            console.error('WebSocket ERROR!', event);
            
            // Debug visual feedback
            if (this.options.debug && window.location.hostname === 'localhost') {
                this.showDebugMessage('WebSocket Connection Error!', '#f44336');
            }
            
            this.handleError(event);
        };

        this.ws.onmessage = (event) => {
            this.handleMessage(event);
        };
    }

    /**
     * Show debug message (for localhost debugging)
     */
    showDebugMessage(text, color) {
        const div = document.createElement('div');
        div.style.cssText = `
            position: fixed;
            top: 10px;
            right: 10px;
            background: ${color};
            color: white;
            padding: 10px;
            z-index: 9999;
            border-radius: 5px;
            font-size: 14px;
            box-shadow: 0 4px 8px rgba(0,0,0,0.2);
        `;
        div.textContent = text;
        document.body.appendChild(div);
        setTimeout(() => div.remove(), 3000);
    }

    /**
     * Handle incoming messages - ENHANCED with alert support
     */
    handleMessage(event) {
        this.stats.messagesReceived++;
        this.stats.lastMessageTime = new Date();

        try {
            const message = JSON.parse(event.data);
            this.log('📨 Received message:', message.type);

            // Generic message handler
            this.trigger('message', message);

            // Handle different message types
            switch (message.type) {
                case 'metrics':
                    this.trigger('metrics', message.data, message);
                    break;
                    
                case 'system_status':
                    this.trigger('system_status', message.data, message);
                    break;
                    
                case 'alert_triggered':
                    this.handleAlertMessage(message);
                    break;
                    
                case 'alert_resolved':
                    this.handleAlertResolvedMessage(message);
                    break;
                    
                case 'pong':
                    this.trigger('pong', message.data, message);
                    break;
                    
                case 'connected':
                    this.log('Server confirmed connection:', message.data);
                    break;
                    
                default:
                    // Handle any other message types
                    if (this.eventHandlers[message.type]) {
                        this.trigger(message.type, message.data, message);
                    } else {
                        this.log('Unknown message type:', message.type);
                    }
                    break;
            }

        } catch (error) {
            this.log('❌ Error parsing message:', error);
            this.handleError(error);
        }
    }

    /**
     * Handle alert triggered messages
     */
    handleAlertMessage(message) {
        this.stats.alertsReceived++;
        
        const alertData = message.data || message;
        this.log('Alert triggered:', alertData);
        
        // Show visual notification for debugging
        if (this.options.debug) {
            const alertName = alertData.alert_name || alertData.name || 'System Alert';
            const severity = alertData.severity || 'warning';
            const color = severity === 'critical' ? '#f44336' : '#ffa726';
            this.showDebugMessage(`${alertName}`, color);
        }
        
        // Trigger alert handlers
        this.trigger('alert_triggered', alertData, message);
    }

    /**
     * Handle alert resolved messages
     */
    handleAlertResolvedMessage(message) {
        const alertData = message.data || message;
        this.log('Alert resolved:', alertData);
        
        // Show visual notification for debugging
        if (this.options.debug) {
            const alertName = alertData.alert_name || alertData.name || 'System Alert';
            this.showDebugMessage(`${alertName} resolved`, '#4ecdc4');
        }
        
        // Trigger alert resolved handlers
        this.trigger('alert_resolved', alertData, message);
    }

    /**
     * Send message to server
     */
    send(type, data = {}) {
        const message = {
            type: type,
            data: data,
            timestamp: new Date().toISOString(),
            client_id: this.getClientId()
        };

        if (this.isConnected && this.ws.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(JSON.stringify(message));
                this.stats.messagesSent++;
                this.log('Sent message:', type);
                return true;
            } catch (error) {
                this.log('Error sending message:', error);
                this.messageQueue.push(message);
                return false;
            }
        } else {
            this.log('Not connected, queuing message:', type);
            this.messageQueue.push(message);
            return false;
        }
    }

    /**
     * Process queued messages
     */
    processMessageQueue() {
        if (this.messageQueue.length === 0) return;

        this.log(`Processing ${this.messageQueue.length} queued messages`);
        
        const queue = [...this.messageQueue];
        this.messageQueue = [];

        queue.forEach(message => {
            this.send(message.type, message.data);
        });
    }

    /**
     * Schedule reconnection attempt
     */
    scheduleReconnect() {
        if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
            this.log('Max reconnection attempts reached');
            this.handleError(new Error('Max reconnection attempts exceeded'));
            return;
        }

        this.reconnectAttempts++;
        const delay = Math.min(
            this.options.reconnectInterval * Math.pow(1.5, this.reconnectAttempts - 1),
            30000 // Max 30 seconds
        );

        this.log(`Scheduling reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);

        this.reconnectTimer = setTimeout(() => {
            this.log(`Reconnection attempt ${this.reconnectAttempts}`);
            this.connect();
        }, delay);
    }

    /**
     * Start heartbeat to keep connection alive
     */
    startHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
        }

        this.heartbeatTimer = setInterval(() => {
            if (this.isConnected) {
                this.ping();
            }
        }, this.options.heartbeatInterval);
    }

    /**
     * Send ping to server
     */
    ping() {
        this.send('ping', { timestamp: Date.now() });
    }

    /**
     * Subscribe to specific metrics/alerts
     */
    subscribe(types = ['metrics', 'system_status', 'alert_triggered']) {
        this.send('subscribe', { types: types });
        this.log('📡 Subscribed to:', types);
    }

    /**
     * Unsubscribe from metrics/alerts
     */
    unsubscribe(types = ['all']) {
        this.send('unsubscribe', { types: types });
        this.log('📡 Unsubscribed from:', types);
    }

    /**
     * Subscribe specifically to alert notifications
     */
    subscribeToAlerts() {
        this.subscribe(['alert_triggered', 'alert_resolved']);
    }

    /**
     * Request alert history
     */
    requestAlertHistory(limit = 10) {
        this.send('get_alert_history', { limit: limit });
    }

    /**
     * Request alert statistics
     */
    requestAlertStats() {
        this.send('get_alert_stats', {});
    }

    /**
     * Add event listener
     */
    on(event, handler) {
        if (!this.eventHandlers[event]) {
            this.eventHandlers[event] = [];
        }
        this.eventHandlers[event].push(handler);
        this.log(`Added handler for event: ${event}`);
    }

    /**
     * Remove event listener
     */
    off(event, handler) {
        if (this.eventHandlers[event]) {
            const index = this.eventHandlers[event].indexOf(handler);
            if (index !== -1) {
                this.eventHandlers[event].splice(index, 1);
                this.log(`Removed handler for event: ${event}`);
            }
        }
    }

    /**
     * Trigger event handlers
     */
    trigger(event, ...args) {
        if (this.eventHandlers[event]) {
            this.eventHandlers[event].forEach(handler => {
                try {
                    handler(...args);
                } catch (error) {
                    this.log(`❌ Error in event handler for ${event}:`, error);
                }
            });
        }
    }

    /**
     * Handle errors
     */
    handleError(error) {
        this.log('❌ Error:', error);
        this.trigger('error', error);
    }

    /**
     * Get connection status
     */
    getStatus() {
        return {
            connected: this.isConnected,
            readyState: this.ws ? this.ws.readyState : WebSocket.CLOSED,
            reconnectAttempts: this.reconnectAttempts,
            stats: this.stats,
            queuedMessages: this.messageQueue.length,
            url: this.options.url
        };
    }

    /**
     * Get client ID
     */
    getClientId() {
        if (!this.clientId) {
            this.clientId = 'client_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
        }
        return this.clientId;
    }

    /**
     * Get connection statistics - ENHANCED with alert stats
     */
    getStats() {
        return {
            ...this.stats,
            connected: this.isConnected,
            reconnectAttempts: this.reconnectAttempts,
            queuedMessages: this.messageQueue.length,
            uptime: this.stats.connectionTime ? Date.now() - this.stats.connectionTime.getTime() : 0,
            url: this.options.url
        };
    }

    /**
     * Enable/disable debug logging
     */
    setDebug(enabled) {
        this.options.debug = enabled;
        this.log('Debug mode:', enabled ? 'enabled' : 'disabled');
    }

    /**
     * Check if connection is healthy
     */
    isHealthy() {
        return this.isConnected && 
               this.ws && 
               this.ws.readyState === WebSocket.OPEN &&
               this.reconnectAttempts < this.options.maxReconnectAttempts;
    }

    /**
     * Get readable connection state
     */
    getReadableState() {
        if (!this.ws) return 'Not initialized';
        
        switch (this.ws.readyState) {
            case WebSocket.CONNECTING: return 'Connecting';
            case WebSocket.OPEN: return 'Connected';
            case WebSocket.CLOSING: return 'Closing';
            case WebSocket.CLOSED: return 'Closed';
            default: return 'Unknown';
        }
    }

    /**
     * Log messages (if debug enabled)
     */
    log(...args) {
        if (this.options.debug) {
            console.log('[WebSocket]', ...args);
        }
    }

    /**
     * Cleanup resources
     */
    destroy() {
        this.log('🧹 Destroying WebSocket client');
        this.disconnect();
        
        // Clear all event handlers
        this.eventHandlers = {
            connect: [],
            disconnect: [],
            reconnect: [],
            error: [],
            message: [],
            metrics: [],
            system_status: [],
            alert_triggered: [],
            alert_resolved: [],
            pong: []
        };
        
        // Clear message queue
        this.messageQueue = [];
        
        // Reset stats
        this.stats = {
            messagesReceived: 0,
            messagesSent: 0,
            reconnectCount: 0,
            connectionTime: null,
            lastMessageTime: null,
            alertsReceived: 0
        };
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = WebSocketClient;
} else if (typeof window !== 'undefined') {
    window.WebSocketClient = WebSocketClient;
}

/**
 * WebSocket Status Constants
 */
WebSocketClient.CONNECTING = 0;
WebSocketClient.OPEN = 1;
WebSocketClient.CLOSING = 2;
WebSocketClient.CLOSED = 3;

/**
 * Utility function to create WebSocket client with default options
 */
function createWebSocketClient(options = {}) {
    return new WebSocketClient({
        debug: window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1',
        ...options
    });
}

// Make utility function globally available
if (typeof window !== 'undefined') {
    window.createWebSocketClient = createWebSocketClient;
}