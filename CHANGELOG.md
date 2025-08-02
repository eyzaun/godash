# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-08-03

### Added
- 📊 **Real-time Web Dashboard**: Complete web interface with Chart.js donut charts
- 🔌 **WebSocket Support**: Real-time data streaming with 500ms updates
- 🎨 **Modern UI**: Responsive dashboard with connection status indicators
- 🔄 **Auto-reconnection**: Robust WebSocket connection with exponential backoff
- 📡 **API Polling Fallback**: Automatic fallback when WebSocket fails
- 🌐 **CORS Configuration**: Proper CORS setup for web interface
- 📝 **Enhanced Configuration**: YAML-based configuration with environment support
- 🐳 **PostgreSQL Integration**: Complete database setup with Docker support
- 🔍 **Health Monitoring**: Comprehensive health checks and status endpoints
- 📈 **Advanced Metrics Collection**: Enhanced CPU calculation with fallback mechanisms
- 🧪 **Test Infrastructure**: Comprehensive test suite with mock implementations

### Changed
- 🔧 **Server Port**: Changed from 8080 to 8081 for development
- 🗄️ **Database Port**: Changed from 5432 to 5433 to avoid conflicts
- ⚡ **Update Frequency**: Increased to 500ms for ultra-responsive dashboard
- 📊 **Metrics Format**: Enhanced API response format for frontend compatibility
- 🎯 **Project Structure**: Added web interface files and static assets

### Technical Details
- **Backend**: Go 1.24 with Gin framework and GORM ORM
- **Frontend**: Vanilla JavaScript with Chart.js v3.9.1
- **Database**: PostgreSQL 15+ with optimized time-series storage
- **Real-time**: WebSocket with message queuing and auto-reconnection
- **Configuration**: YAML-based with environment variable override support
- **Docker**: Complete containerization with docker-compose setup

### API Endpoints
- `GET /` - Real-time dashboard interface
- `WS /ws` - WebSocket endpoint for real-time metrics
- `GET /health` - Application health check
- `GET /api/v1/metrics/current` - Current system metrics
- `GET /api/v1/system/status` - System status overview
- `GET /api/v1/system/stats` - Database and collection statistics

### Dashboard Features
- **Real-time Charts**: CPU, Memory, and Disk usage donut charts
- **Connection Status**: Visual WebSocket and API connectivity indicators
- **Auto-refresh**: 500ms update interval with smooth animations
- **Responsive Design**: Works on desktop, tablet, and mobile
- **Error Handling**: Graceful degradation with user feedback

### Performance Optimizations
- **WebSocket Broadcasting**: Efficient real-time data streaming
- **Database Batching**: Optimized bulk metric storage
- **Chart Animations**: Smooth transitions without performance impact
- **Memory Management**: Efficient resource cleanup and garbage collection
- **Connection Pooling**: PostgreSQL connection optimization

### Configuration Files
- `configs/config.yaml` - Main configuration
- `configs/development.yaml` - Development-specific settings
- `configs/production.yaml` - Production-ready configuration
- Environment variable support for all configuration options

### Documentation Updates
- **README.md**: Updated with current features and dashboard information
- **SETUP.md**: Enhanced with web dashboard setup instructions
- **API Documentation**: Complete endpoint documentation with examples
- **Docker Instructions**: Updated compose configuration and deployment guides

### Development Tools
- **Hot Reload**: Development server with automatic restart
- **Test Coverage**: Comprehensive unit and integration tests
- **Linting**: Go code quality checks and formatting
- **Build Scripts**: Cross-platform build automation

## [0.2.0] - Previous Release

### Added
- REST API with full CRUD operations
- PostgreSQL database integration
- Metrics collection and storage
- Health checks and monitoring
- Docker containerization

## [0.1.0] - Initial Release

### Added
- Basic CLI interface
- System metrics collection (CPU, Memory, Disk, Network)
- Cross-platform support
- JSON output format
- Basic project structure
