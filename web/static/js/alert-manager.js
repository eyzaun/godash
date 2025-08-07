/**
 * Enhanced Alert Manager - Dashboard Integration Ready
 */
class AlertManager {
    constructor(apiUrl = '/api/v1') {
        this.apiUrl = apiUrl;
        this.alerts = [];
        this.alertHistory = [];
        this.stats = {};
        this.isVisible = false;
        this.refreshInterval = 30000; // 30 seconds
        this.refreshTimer = null;
        
        // Integration with dashboard
        this.dashboard = null;
        this.initialized = false;
        
        this.log('Alert Manager initialized with API:', this.apiUrl);
    }

    /**
     * Initialize with dashboard integration
     */
    async init(dashboard = null) {
        if (this.initialized) return;

        try {
            this.dashboard = dashboard;
            this.log('Initializing Alert Manager...');
            
            this.setupEventListeners();
            await this.loadAlerts();
            await this.loadAlertStats();
            this.startAutoRefresh();
            
            this.initialized = true;
            this.log('Alert Manager initialized successfully');
        } catch (error) {
            this.log('Error initializing Alert Manager:', error);
        }
    }

    /**
     * Setup event listeners
     */
    setupEventListeners() {
        // Form submission
        const createForm = document.getElementById('createAlertForm');
        if (createForm) {
            createForm.addEventListener('submit', this.handleCreateAlert.bind(this));
        }

        // Modal controls
        const modal = document.getElementById('createAlertModal');
        if (modal) {
            // Close modal when clicking outside
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.closeCreateAlertModal();
                }
            });
        }

        // Checkbox dependencies
        const emailCheckbox = document.getElementById('emailEnabled');
        const webhookCheckbox = document.getElementById('webhookEnabled');
        
        if (emailCheckbox) {
            emailCheckbox.addEventListener('change', this.toggleEmailFields.bind(this));
        }
        
        if (webhookCheckbox) {
            webhookCheckbox.addEventListener('change', this.toggleWebhookFields.bind(this));
        }

        this.log('👂 Alert Manager event listeners setup complete');
    }

    /**
     * Load alerts from API
     */
    async loadAlerts() {
        try {
            const response = await fetch(`${this.apiUrl}/alerts`);
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const result = await response.json();
            if (result.success) {
                this.alerts = result.data || [];
                this.renderAlerts();
                this.updateAlertBadge();
                this.log('✅ Loaded', this.alerts.length, 'alerts');
            } else {
                throw new Error(result.message || 'Failed to load alerts');
            }
        } catch (error) {
            this.log('❌ Failed to load alerts:', error);
            this.showNotification('Failed to load alerts: ' + error.message, 'error');
        }
    }

    /**
     * Load alert statistics
     */
    async loadAlertStats() {
        try {
            const response = await fetch(`${this.apiUrl}/alerts/stats`);
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const result = await response.json();
            if (result.success) {
                this.stats = result.data || {};
                this.updateStatsDisplay();
                this.updateAlertBadge();
                this.log('✅ Alert stats loaded:', this.stats);
            } else {
                throw new Error(result.message || 'Failed to load alert stats');
            }
        } catch (error) {
            this.log('❌ Failed to load alert stats:', error);
        }
    }

    /**
     * Load alert history
     */
    async loadAlertHistory() {
        try {
            const response = await fetch(`${this.apiUrl}/alerts/history?limit=10`);
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const result = await response.json();
            if (result.success) {
                this.alertHistory = result.data || [];
                this.renderAlertHistory();
                this.log('✅ Alert history loaded:', this.alertHistory.length, 'entries');
            } else {
                throw new Error(result.message || 'Failed to load alert history');
            }
        } catch (error) {
            this.log('❌ Failed to load alert history:', error);
        }
    }

    /**
     * Render alerts list
     */
    renderAlerts() {
        const alertsList = document.getElementById('alertsList');
        if (!alertsList) return;

        if (this.alerts.length === 0) {
            alertsList.innerHTML = `
                <div class="no-alerts">
                    <div class="no-alerts-icon">!</div>
                    <div class="no-alerts-text">No alerts configured</div>
                    <button class="btn btn-primary btn-sm" onclick="window.alertManager.showCreateAlertModal()">
                        Create your first alert
                    </button>
                </div>
            `;
            return;
        }

        const alertsHTML = this.alerts.map(alert => `
            <div class="alert-item ${alert.is_active ? 'active' : 'inactive'}" data-alert-id="${alert.id}">
                <div class="alert-header">
                    <div class="alert-title">
                        <span class="alert-name">${this.escapeHtml(alert.name)}</span>
                        <span class="alert-severity severity-${alert.severity}">${alert.severity.toUpperCase()}</span>
                    </div>
                    <div class="alert-actions">
                        <button class="btn-icon" onclick="window.alertManager.testAlert(${alert.id})" title="Test Alert">
                            T
                        </button>
                        <button class="btn-icon" onclick="window.alertManager.toggleAlert(${alert.id})" title="${alert.is_active ? 'Disable' : 'Enable'} Alert">
                            ${alert.is_active ? 'OFF' : 'ON'}
                        </button>
                        <button class="btn-icon btn-danger" onclick="window.alertManager.deleteAlert(${alert.id})" title="Delete Alert">
                            X
                        </button>
                    </div>
                </div>
                <div class="alert-details">
                    <div class="alert-condition">
                        ${alert.metric_type.toUpperCase()} ${alert.condition} ${alert.threshold}${this.getMetricUnit(alert.metric_type)}
                    </div>
                    <div class="alert-meta">
                        <span>Triggered: ${alert.triggered_count || 0} times</span>
                        ${alert.last_triggered ? `<span>Last: ${this.formatDate(alert.last_triggered)}</span>` : ''}
                    </div>
                    <div class="alert-notifications">
                        ${alert.email_enabled ? '<span class="notification-badge">Email</span>' : ''}
                        ${alert.webhook_enabled ? '<span class="notification-badge">Webhook</span>' : ''}
                    </div>
                </div>
            </div>
        `).join('');

        alertsList.innerHTML = alertsHTML;
        this.log('Rendered', this.alerts.length, 'alerts');
    }

    /**
     * Update stats display
     */
    updateStatsDisplay() {
        const elements = {
            totalAlerts: document.getElementById('totalAlerts'),
            activeAlerts: document.getElementById('activeAlerts'),
            triggeredToday: document.getElementById('triggeredToday'),
            unresolvedAlerts: document.getElementById('unresolvedAlerts')
        };

        if (elements.totalAlerts) elements.totalAlerts.textContent = this.stats.total_alerts || 0;
        if (elements.activeAlerts) elements.activeAlerts.textContent = this.stats.active_alerts || 0;
        if (elements.triggeredToday) elements.triggeredToday.textContent = this.stats.triggered_today || 0;
        if (elements.unresolvedAlerts) elements.unresolvedAlerts.textContent = this.stats.unresolved_alerts || 0;

        this.log('📊 Stats display updated');
    }

    /**
     * Update alert badge
     */
    updateAlertBadge() {
        const badge = document.getElementById('alertBadge');
        if (badge) {
            const unresolvedCount = this.stats.unresolved_alerts || 0;
            badge.textContent = unresolvedCount;
            
            if (unresolvedCount > 0) {
                badge.classList.add('visible');
                badge.style.display = 'inline-block';
            } else {
                badge.classList.remove('visible');
                badge.style.display = 'none';
            }
        }
    }

    /**
     * Handle create alert form submission
     */
    async handleCreateAlert(event) {
        event.preventDefault();
        
        const formData = new FormData(event.target);
        const alertData = {
            name: formData.get('name'),
            metric_type: formData.get('metric_type'),
            condition: formData.get('condition'),
            threshold: parseFloat(formData.get('threshold')),
            duration: parseInt(formData.get('duration')) || 0,
            severity: formData.get('severity'),
            description: formData.get('description') || '',
            email_enabled: formData.has('email_enabled'),
            email_recipients: formData.get('email_recipients') || '',
            webhook_enabled: formData.has('webhook_enabled'),
            webhook_url: formData.get('webhook_url') || ''
        };

        try {
            const response = await fetch(`${this.apiUrl}/alerts`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(alertData)
            });

            const result = await response.json();
            
            if (result.success) {
                this.showNotification('Alert created successfully', 'success');
                this.closeCreateAlertModal();
                await this.loadAlerts();
                await this.loadAlertStats();
                this.log('✅ Alert created:', alertData.name);
            } else {
                throw new Error(result.message || 'Failed to create alert');
            }
        } catch (error) {
            this.log('❌ Failed to create alert:', error);
            this.showNotification(`Failed to create alert: ${error.message}`, 'error');
        }
    }

    /**
     * Test alert
     */
    async testAlert(alertId) {
        try {
            const response = await fetch(`${this.apiUrl}/alerts/${alertId}/test`, {
                method: 'POST'
            });

            const result = await response.json();
            
            if (result.success) {
                const testResults = result.data.test_results || [];
                const message = testResults.join('; ');
                this.showNotification(`Alert test completed: ${message}`, 'info');
                this.log('✅ Alert test completed:', alertId);
            } else {
                throw new Error(result.message || 'Failed to test alert');
            }
        } catch (error) {
            this.log('❌ Failed to test alert:', error);
            this.showNotification(`Failed to test alert: ${error.message}`, 'error');
        }
    }

    /**
     * Toggle alert active/inactive
     */
    async toggleAlert(alertId) {
        const alert = this.alerts.find(a => a.id === alertId);
        if (!alert) return;

        try {
            const response = await fetch(`${this.apiUrl}/alerts/${alertId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    is_active: !alert.is_active
                })
            });

            const result = await response.json();
            
            if (result.success) {
                const action = alert.is_active ? 'disabled' : 'enabled';
                this.showNotification(`Alert ${action} successfully`, 'success');
                await this.loadAlerts();
                await this.loadAlertStats();
                this.log('✅ Alert toggled:', alertId, action);
            } else {
                throw new Error(result.message || 'Failed to toggle alert');
            }
        } catch (error) {
            this.log('❌ Failed to toggle alert:', error);
            this.showNotification(`Failed to toggle alert: ${error.message}`, 'error');
        }
    }

    /**
     * Delete alert
     */
    async deleteAlert(alertId) {
        if (!confirm('Are you sure you want to delete this alert? This action cannot be undone.')) {
            return;
        }

        try {
            const response = await fetch(`${this.apiUrl}/alerts/${alertId}`, {
                method: 'DELETE'
            });

            const result = await response.json();
            
            if (result.success) {
                this.showNotification('Alert deleted successfully', 'success');
                await this.loadAlerts();
                await this.loadAlertStats();
                this.log('✅ Alert deleted:', alertId);
            } else {
                throw new Error(result.message || 'Failed to delete alert');
            }
        } catch (error) {
            this.log('❌ Failed to delete alert:', error);
            this.showNotification(`Failed to delete alert: ${error.message}`, 'error');
        }
    }

    /**
     * Show create alert modal
     */
    showCreateAlertModal() {
        const modal = document.getElementById('createAlertModal');
        if (modal) {
            modal.classList.add('visible');
            modal.style.display = 'flex';
            document.body.style.overflow = 'hidden';
            
            // Reset form
            this.resetCreateAlertForm();
            
            this.log('📝 Create alert modal opened');
        }
    }

    /**
     * Close create alert modal
     */
    closeCreateAlertModal() {
        const modal = document.getElementById('createAlertModal');
        if (modal) {
            modal.classList.remove('visible');
            modal.style.display = 'none';
            document.body.style.overflow = 'auto';
            
            // Reset form
            this.resetCreateAlertForm();
            
            this.log('❌ Create alert modal closed');
        }
    }

    /**
     * Reset create alert form
     */
    resetCreateAlertForm() {
        const form = document.getElementById('createAlertForm');
        if (form) {
            form.reset();
            this.toggleEmailFields();
            this.toggleWebhookFields();
        }
    }

    /**
     * Toggle email fields based on checkbox
     */
    toggleEmailFields() {
        const emailEnabled = document.getElementById('emailEnabled');
        const emailRecipients = document.getElementById('emailRecipients');
        
        if (emailEnabled && emailRecipients) {
            emailRecipients.disabled = !emailEnabled.checked;
            emailRecipients.required = emailEnabled.checked;
        }
    }

    /**
     * Toggle webhook fields based on checkbox
     */
    toggleWebhookFields() {
        const webhookEnabled = document.getElementById('webhookEnabled');
        const webhookUrl = document.getElementById('webhookUrl');
        
        if (webhookEnabled && webhookUrl) {
            webhookUrl.disabled = !webhookEnabled.checked;
            webhookUrl.required = webhookEnabled.checked;
        }
    }

    /**
     * Toggle alerts panel visibility
     */
    toggleAlertsPanel() {
        const panel = document.getElementById('alertsPanel');
        if (panel) {
            this.isVisible = !this.isVisible;
            
            if (this.isVisible) {
                panel.classList.add('visible');
                panel.style.display = 'block';
                this.loadAlerts();
                this.loadAlertStats();
                this.log('👁️ Alerts panel opened');
            } else {
                panel.classList.remove('visible');
                panel.style.display = 'none';
                this.log('🙈 Alerts panel closed');
            }
        }
    }

    /**
     * Start auto refresh
     */
    startAutoRefresh() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
        }
        
        this.refreshTimer = setInterval(() => {
            if (this.isVisible) {
                this.loadAlerts();
                this.loadAlertStats();
            }
        }, this.refreshInterval);
        
        this.log('🔄 Auto refresh started');
    }

    /**
     * Stop auto refresh
     */
    stopAutoRefresh() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
            this.log('⏹️ Auto refresh stopped');
        }
    }

    /**
     * Manual refresh
     */
    async refresh() {
        try {
            await this.loadAlerts();
            await this.loadAlertStats();
            this.showNotification('Alerts refreshed', 'success', 2000);
            this.log('🔄 Manual refresh completed');
        } catch (error) {
            this.showNotification('Failed to refresh alerts', 'error');
            this.log('❌ Manual refresh failed:', error);
        }
    }

    /**
     * Handle real-time alert notifications via WebSocket
     */
    handleAlertNotification(alertData) {
        if (!alertData) return;

        this.log('Handling alert notification:', alertData);

        if (alertData.type === 'alert_triggered' || alertData.alert_id) {
            const alert = alertData.alert || alertData;
            const severity = alert.severity || 'warning';
            const hostname = alert.hostname || 'system';
            const alertName = alert.alert_name || alert.name || 'System Alert';
            
            // Show notification
            const message = `ALERT ${severity.toUpperCase()}: ${alertName} on ${hostname}`;
            this.showNotification(message, severity === 'critical' ? 'error' : 'warning', 10000);
            
            // Refresh stats
            this.loadAlertStats();
            
            this.log('Alert notification displayed');
        }
    }

    /**
     * Get metric unit for display
     */
    getMetricUnit(metricType) {
        const units = {
            cpu: '%',
            memory: '%',
            disk: '%',
            load_avg_1: '',
            load_avg_5: '',
            load_avg_15: ''
        };
        return units[metricType] || '%';
    }

    /**
     * Format date for display
     */
    formatDate(dateString) {
        return new Date(dateString).toLocaleString();
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
     * Show notification - integrates with dashboard
     */
    showNotification(message, type = 'info', duration = 5000) {
        // Use dashboard notification system if available
        if (this.dashboard && this.dashboard.showNotification) {
            this.dashboard.showNotification(message, type, duration);
            return;
        }

        // Fallback to own notification system
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

        // Auto-remove after duration
        if (duration > 0) {
            setTimeout(() => {
                if (notification.parentElement) {
                    notification.remove();
                }
            }, duration);
        }
    }

    /**
     * Log messages
     */
    log(...args) {
        if (this.dashboard && this.dashboard.options && this.dashboard.options.debug) {
            console.log('[AlertManager]', ...args);
        } else if (window.location.hostname === 'localhost') {
            console.log('[AlertManager]', ...args);
        }
    }

    /**
     * Get alert manager statistics
     */
    getStats() {
        return {
            alertsCount: this.alerts.length,
            stats: this.stats,
            isVisible: this.isVisible,
            initialized: this.initialized,
            refreshInterval: this.refreshInterval
        };
    }

    /**
     * Cleanup resources
     */
    cleanup() {
        this.stopAutoRefresh();
        this.alerts = [];
        this.alertHistory = [];
        this.stats = {};
        this.initialized = false;
        this.log('🧹 Alert Manager cleaned up');
    }
}

// Global functions for template onclick handlers
if (typeof window !== 'undefined') {
    window.showCreateAlertModal = function() {
        if (window.alertManager) {
            window.alertManager.showCreateAlertModal();
        }
    };

    window.closeCreateAlertModal = function() {
        if (window.alertManager) {
            window.alertManager.closeCreateAlertModal();
        }
    };

    window.toggleAlertsPanel = function() {
        if (window.alertManager) {
            window.alertManager.toggleAlertsPanel();
        }
    };

    window.refreshAlerts = function() {
        if (window.alertManager) {
            window.alertManager.refresh();
        }
    };
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = AlertManager;
} else if (typeof window !== 'undefined') {
    window.AlertManager = AlertManager;
}