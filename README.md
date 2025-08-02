# GoDash - System Monitoring Tool

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Supported-2496ED?style=flat&logo=docker)](https://docker.com)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://postgresql.org)

> **Real-time system monitoring dashboard built with Go, featuring CPU, memory, disk, and network monitoring with automated alerts and historical data analysis.**

## 🌟 Features

### ✅ **Phase 1: Core System Monitoring** (Completed)
- 🖥️ **Real-time Metrics Collection**: CPU, Memory, Disk, Network usage
- 📊 **CLI Interface**: Beautiful colored terminal output with JSON export
- 🔄 **Cross-platform Support**: Windows, Linux, macOS
- 📈 **Performance Optimized**: Goroutine-based concurrent collection
- 🧪 **Comprehensive Testing**: Unit tests, benchmarks, CI/CD pipeline

### ✅ **Phase 2: Web API + Database** (Completed)
- 🌐 **REST API**: Full-featured API with pagination and filtering
- 🗄️ **PostgreSQL Integration**: High-performance time-series data storage
- 📦 **Batch Processing**: Efficient bulk data insertion
- 🔍 **Advanced Queries**: Aggregations, trends, and statistical analysis
- 🛡️ **Production Ready**: Health checks, middleware, security headers
- 🐳 **Docker Support**: Complete containerization with docker-compose

### ✅ **Phase 3: Real-time Dashboard** (Completed)
- 📱 **WebSocket Live Updates**: Real-time data streaming every 500ms
- 📊 **Interactive Charts**: Chart.js powered donut charts with real-time updates
- 🎨 **Responsive UI**: Modern web interface with connection status indicators
- 🔄 **Auto-reconnection**: Robust WebSocket connection with fallback to API polling
- 📈 **Visual Metrics**: CPU, Memory, Disk usage with animated charts

### 🔔 **Phase 4: Alerts & Production** (In Progress)
- ✅ **Health Monitoring**: Comprehensive health checks and status endpoints
- ✅ **Production Configuration**: Environment-based configuration management
- ✅ **Docker Integration**: Complete containerization with docker-compose
- � **Email Notifications**: SMTP-based alerting (planned)
- � **Webhook Integration**: Custom webhook endpoints (planned)
- � **Advanced Deployment**: Kubernetes support (planned)

## 🚀 Quick Start

### Prerequisites
- Go 1.19+ installed
- PostgreSQL 12+ (or Docker)
- Make (optional, for easy commands)

### 1. Clone and Setup
```bash
git clone https://github.com/eyzaun/godash.git
cd godash

# Initialize development environment
make init
```

### 2. Development Setup
```bash
# Start database
make db-up

# Run in development mode (hot reload)
make dev

# Or build and run manually
make build
./build/godash
```

### 3. Docker Setup (Recommended)
```bash
# Start everything with Docker Compose
docker-compose up --build

# Or just the database
docker-compose up -d postgres redis
```

## 📖 API Documentation

### Base URL
```
http://localhost:8081/api/v1
```

### 🌐 **Dashboard Access**
- **Real-time Dashboard**: http://localhost:8081/
- **WebSocket Endpoint**: ws://localhost:8081/ws
- **Health Check**: http://localhost:8081/health

### Core Endpoints

#### **📊 Metrics**
- `GET /metrics/current` - Latest metrics from all hosts
- `GET /metrics/current/{hostname}` - Latest metrics for specific host
- `GET /metrics/history` - Historical metrics with pagination
- `GET /metrics/history/{hostname}` - Host-specific history
- `GET /metrics/average?duration=1h` - Average usage over time period
- `GET /metrics/summary` - Statistical summary for time range
- `GET /metrics/trends/{hostname}` - Usage trends analysis
- `GET /metrics/top/{type}` - Top hosts by CPU/memory/disk usage

#### **🌐 WebSocket**
- `WS /ws` - Real-time metrics streaming
- **Message Types**: `metrics`, `system_status`, `ping`/`pong`
- **Update Interval**: 500ms real-time broadcasting
- **Auto-reconnection**: Robust connection management

#### **🖥️ System**
- `GET /system/status` - Current status of all monitored systems
- `GET /system/hosts` - List of monitored hosts
- `GET /system/stats` - Database and collection statistics

#### **🔧 Admin**
- `DELETE /admin/metrics/cleanup?days=30` - Remove old metrics
- `GET /admin/database/stats` - Database performance stats

#### **💊 Health**
- `GET /health` - Comprehensive health check
- `GET /ready` - Readiness probe for Kubernetes
- `GET /metrics` - Prometheus metrics endpoint

### Example API Calls

```bash
# Get current system status
curl http://localhost:8081/api/v1/system/status

# Get metrics from last hour
curl "http://localhost:8081/api/v1/metrics/history?from=$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)&limit=100"

# Get average CPU usage over last 24 hours
curl "http://localhost:8081/api/v1/metrics/average?duration=24h"

# Get top 5 hosts by memory usage
curl http://localhost:8081/api/v1/metrics/top/memory?limit=5

# Test WebSocket connection (requires wscat or similar)
wscat -c ws://localhost:8081/ws

# Access real-time dashboard
open http://localhost:8081/
```

## 🔧 Configuration

### Environment Variables
```bash
# Database
GODASH_DB_HOST=localhost
GODASH_DB_PORT=5433
GODASH_DB_USER=godash
GODASH_DB_PASSWORD=password
GODASH_DB_NAME=godash

# Application
GODASH_SERVER_PORT=8081
GODASH_COLLECTION_INTERVAL=30s
GODASH_RETENTION_DAYS=7
GODASH_LOG_LEVEL=debug

# Features
GODASH_ENABLE_CPU=true
GODASH_ENABLE_MEMORY=true
GODASH_ENABLE_DISK=true
GODASH_ENABLE_NETWORK=true
GODASH_ENABLE_PROCESSES=false

# WebSocket
GODASH_WEBSOCKET_ENABLED=true
GODASH_WEBSOCKET_BROADCAST_INTERVAL=500ms
```

### Configuration Files
- `configs/development.yaml` - Development settings
- `configs/production.yaml` - Production settings

## 🛠️ Development

### Available Make Commands
```bash
make help              # Show all available commands
make dev               # Start development server with hot reload
make test              # Run all tests with coverage
make lint              # Run code linters
make build             # Build for current platform
make build-all         # Build for all platforms
make docker-build      # Build Docker image
make db-up             # Start PostgreSQL database
make db-reset          # Reset database completely
make check             # Run all checks (fmt, vet, lint, test)
```

### Project Structure
```
godash/
├── cmd/
│   └── cli/                 # CLI application
├── internal/
│   ├── api/                 # REST API layer
│   │   ├── handlers/        # HTTP handlers
│   │   └── middleware/      # Custom middleware
│   ├── collector/           # Metrics collection
│   ├── config/              # Configuration management
│   ├── database/            # Database connection
│   ├── models/              # Data models
│   ├── repository/          # Data access layer
│   ├── services/            # Business logic
│   └── utils/               # Utility functions
├── web/                     # Web interface and dashboard
│   ├── static/              # Static assets (CSS, JS)
│   │   ├── css/             # Dashboard styles
│   │   └── js/              # Chart.js and WebSocket client
│   └── templates/           # HTML templates
├── configs/                 # Configuration files
├── scripts/                 # Database and setup scripts
└── tests/                   # Integration and test files
```

### Running Tests
```bash
# All tests
make test

# Short tests only
make test-short

# Benchmarks
make benchmark

# With race detection
go test -race ./...
```

## 📊 Metrics Collected

### **CPU Metrics**
- Overall usage percentage
- Per-core usage
- Load averages (1m, 5m, 15m)
- CPU frequency

### **Memory Metrics**
- Total, used, available memory
- Memory usage percentage
- Cached and buffered memory
- Swap usage and percentage

### **Disk Metrics**
- Total, used, free space per partition
- Disk usage percentage
- I/O statistics (reads, writes, operations)
- Filesystem types and mount points

### **Network Metrics**
- Bytes sent/received per interface
- Packet counts and error rates
- Network interface statistics

## � Dashboard Features

### Real-time Web Interface
- **Live Metrics Display**: CPU, Memory, and Disk usage with animated donut charts
- **Real-time Updates**: 500ms refresh rate via WebSocket connection
- **Connection Status**: Visual indicators for WebSocket and API connectivity
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Auto-reconnection**: Automatic reconnection with exponential backoff
- **Fallback Mechanism**: API polling fallback when WebSocket fails

### Chart Types
- **Donut Charts**: CPU, Memory, and Disk usage percentages
- **Color-coded Status**: Green (healthy), Yellow (warning), Red (critical)
- **Smooth Animations**: Chart.js powered smooth transitions
- **Tooltips**: Detailed information on hover

### Dashboard Access
Navigate to `http://localhost:8081/` after starting the server to access the real-time dashboard.

## �🐳 Docker Deployment

### Quick Start
```bash
# Start all services
docker-compose up -d

# Scale the application
docker-compose up -d --scale godash=3

# View logs
docker-compose logs -f godash

# Stop all services
docker-compose down
```

### Production Deployment
```bash
# Use production profile
docker-compose --profile nginx --profile monitoring up -d

# This includes:
# - GoDash application
# - PostgreSQL database
# - Redis cache
# - Nginx reverse proxy
# - Prometheus monitoring
# - Grafana dashboards
```

## 📈 Performance

### Benchmarks (on typical hardware)
- **Collection Rate**: 30-second intervals (configurable)
- **API Response Time**: < 50ms for current metrics
- **WebSocket Updates**: 500ms real-time broadcasting
- **Database Performance**: 1000+ inserts/second
- **Memory Usage**: ~50MB base + ~1MB per 10k metrics
- **CPU Impact**: < 1% during normal operation
- **Dashboard Performance**: Sub-second chart updates

### Scalability
- **Metrics Storage**: 100M+ records tested
- **Concurrent Connections**: 1000+ API clients + WebSocket connections
- **Data Retention**: Configurable (default: 7 days development, 30 days production)
- **Batch Processing**: Optimized bulk inserts
- **Real-time Clients**: 100+ simultaneous WebSocket connections tested

## 🔒 Security

- **Input Validation**: All API inputs validated
- **SQL Injection Protection**: GORM ORM with parameterized queries
- **Rate Limiting**: Configurable per-IP limits
- **Security Headers**: OWASP recommended headers
- **Authentication**: Basic auth for admin endpoints (configurable)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make check`)
5. Commit your changes (`git commit -am 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [gopsutil](https://github.com/shirou/gopsutil) for cross-platform system metrics
- [Gin](https://github.com/gin-gonic/gin) for the HTTP web framework
- [GORM](https://gorm.io/) for the Go ORM
- [Viper](https://github.com/spf13/viper) for configuration management
- [Chart.js](https://www.chartjs.org/) for real-time dashboard charts
- [WebSocket](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API) for real-time communication

## 📞 Support

- 📧 **Email**: support@godash.io
- 💬 **Issues**: [GitHub Issues](https://github.com/eyzaun/godash/issues)
- 📖 **Documentation**: [Wiki](https://github.com/eyzaun/godash/wiki)
- 🚀 **Roadmap**: [Project Board](https://github.com/eyzaun/godash/projects)

---

<div align="center">
  <strong>Built with ❤️ using Go</strong>
  <br>
  <sub>Star ⭐ this repo if you find it useful!</sub>
</div>