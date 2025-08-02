/**
 * WebSocket Client for GoDash Dashboard
 * Handles real-time communication with the server
 */

class WebSocketClient {
    constructor(options = {}) {
        this.options = {
            url: this.getWebSocketURL(),
            reconnectInterval: 5000,
            maxReconnectAttempts: 10,
            heartbeatInterval: 5000,  // Even faster heartbeat for ultra responsive feel
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
            lastMessageTime: null
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

        console.log('🔌 WebSocket attempting to connect to:', this.options.url);
        this.log('Connecting to WebSocket...', this.options.url);

        try {
            console.log('🚀 Creating WebSocket connection to:', this.options.url);
            this.ws = new WebSocket(this.options.url);
            console.log('✅ WebSocket object created successfully');
            this.setupEventListeners();
        } catch (error) {
            console.error('❌ Failed to create WebSocket:', error);
            this.log('Connection error:', error);
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
        console.log('📡 Setting up WebSocket event listeners...');
        
        this.ws.onopen = (event) => {
            console.log('🎉 WebSocket OPENED!', event);
            this.log('Connected to WebSocket server');
            this.isConnected = true;
            
            // Visual feedback for debugging
            if (window.location.hostname === 'localhost') {
                const successDiv = document.createElement('div');
                successDiv.style.cssText = 'position:fixed;top:10px;right:10px;background:green;color:white;padding:10px;z-index:9999;border-radius:5px;';
                successDiv.textContent = '✅ WebSocket Connected!';
                document.body.appendChild(successDiv);
                setTimeout(() => successDiv.remove(), 3000);
            }
            this.reconnectAttempts = 0;
            this.stats.connectionTime = new Date();
            this.stats.reconnectCount += (this.stats.connectionTime ? 1 : 0);

            // Start heartbeat
            this.startHeartbeat();

            // Process queued messages
            this.processMessageQueue();

            // Trigger connect event
            this.trigger('connect', event);

            if (this.reconnectAttempts > 0) {
                this.trigger('reconnect', event);
            }
        };

        this.ws.onclose = (event) => {
            console.log('❌ WebSocket CLOSED!', event.code, event.reason);
            this.log('WebSocket connection closed', event.code, event.reason);
            this.isConnected = false;

            // Stop heartbeat
            if (this.heartbeatTimer) {
                clearInterval(this.heartbeatTimer);
                this.heartbeatTimer = null;
            }

            // Trigger disconnect event
            this.trigger('disconnect', event);

            // Schedule reconnect if it wasn't a manual close
            if (event.code !== 1000 && this.reconnectAttempts < this.options.maxReconnectAttempts) {
                this.scheduleReconnect();
            }
        };

        this.ws.onerror = (event) => {
            console.error('💥 WebSocket ERROR!', event);
            this.log('WebSocket error:', event);
            
            // Visual feedback for debugging
            if (window.location.hostname === 'localhost') {
                const errorDiv = document.createElement('div');
                errorDiv.style.cssText = 'position:fixed;top:10px;right:10px;background:red;color:white;padding:10px;z-index:9999;border-radius:5px;';
                errorDiv.textContent = '❌ WebSocket Connection Error!';
                document.body.appendChild(errorDiv);
                setTimeout(() => errorDiv.remove(), 5000);
            }
            
            this.handleError(event);
        };

        this.ws.onmessage = (event) => {
            this.handleMessage(event);
        };
    }

    /**
     * Handle incoming messages
     */
    handleMessage(event) {
        this.stats.messagesReceived++;
        this.stats.lastMessageTime = new Date();

        try {
            const message = JSON.parse(event.data);
            this.log('📨 Received message:', message.type, message);

            // Trigger general message event
            this.trigger('message', message);

            // Trigger specific event based on message type
            if (message.type && this.eventHandlers[message.type]) {
                this.log(`🎯 Triggering event: ${message.type}`, message.data);
                this.trigger(message.type, message.data, message);
            } else {
                this.log(`⚠️ No handler for message type: ${message.type}`);
            }

        } catch (error) {
            this.log('❌ Error parsing message:', error, event.data);
            this.handleError(error);
        }
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
                this.log('Sent message:', type, message);
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
     * Subscribe to specific metrics
     */
    subscribe(metricTypes = ['all']) {
        this.send('subscribe', { types: metricTypes });
    }

    /**
     * Unsubscribe from metrics
     */
    unsubscribe(metricTypes = ['all']) {
        this.send('unsubscribe', { types: metricTypes });
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
                    this.log(`Error in event handler for ${event}:`, error);
                }
            });
        }
    }

    /**
     * Handle errors
     */
    handleError(error) {
        this.log('Error:', error);
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
            queuedMessages: this.messageQueue.length
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
     * Get connection statistics
     */
    getStats() {
        return {
            ...this.stats,
            connected: this.isConnected,
            reconnectAttempts: this.reconnectAttempts,
            queuedMessages: this.messageQueue.length,
            uptime: this.stats.connectionTime ? Date.now() - this.stats.connectionTime.getTime() : 0
        };
    }

    /**
     * Enable/disable debug logging
     */
    setDebug(enabled) {
        this.options.debug = enabled;
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
        this.log('Destroying WebSocket client');
        this.disconnect();
        
        // Clear all event handlers
        this.eventHandlers = {};
        
        // Clear message queue
        this.messageQueue = [];
        
        // Reset stats
        this.stats = {
            messagesReceived: 0,
            messagesSent: 0,
            reconnectCount: 0,
            connectionTime: null,
            lastMessageTime: null
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