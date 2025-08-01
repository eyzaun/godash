# GoDash Setup Instructions

This document provides detailed instructions for setting up GoDash System Monitor for development and production use.

## 📋 Prerequisites

### System Requirements
- **Operating System**: Linux, macOS, or Windows
- **Go**: Version 1.19 or higher
- **Git**: For version control
- **Make**: For build automation (optional but recommended)

### Hardware Requirements
- **Minimum**: 1 CPU core, 512MB RAM, 100MB disk space
- **Recommended**: 2+ CPU cores, 1GB+ RAM, 500MB+ disk space

## 🚀 Quick Start

### 1. Clone the Repository
```bash
git clone https://github.com/eyzaun/godash.git
cd godash
```

### 2. Install Dependencies
```bash
# Using Make (recommended)
make deps

# Or manually
go mod download
go mod tidy
```

### 3. Build the Application
```bash
# Build main application
make build

# Build CLI application
make build-cli

# Or build manually
go build -o build/godash .
go build -o build/godash-cli ./cmd/cli
```

### 4. Run the Application
```bash
# Run main application
make run

# Run CLI application
make run-cli

# Or run manually
./build/godash
./build/godash-cli
```

## 🔧 Development Setup

### IDE Configuration

#### VS Code (Recommended)
1. Install the Go extension
2. Create `.vscode/settings.json`:
```json
{
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.vetOnSave": "package",
    "go.testOnSave": true,
    "go.coverOnSave": true,
    "editor.formatOnSave": true
}
```

#### GoLand/IntelliJ IDEA
1. Install Go plugin
2. Configure Go SDK path
3. Enable format on save
4. Configure test runner

### Development Tools

#### Install Additional Tools
```bash
# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Hot reload for development
go install github.com/cosmtrek/air@latest

# Security scanner
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Test coverage tool
go install github.com/axw/gocov/gocov@latest
```

#### Environment Setup
```bash
# Set Go environment variables
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# For development
export GODASH_ENV=development
export GODASH_LOG_LEVEL=debug
```

### Git Hooks (Optional)
```bash
# Install pre-commit hooks
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
make fmt
make vet
make test-short
EOF

chmod +x .git/hooks/pre-commit
```

## 🏗️ Build Options

### Standard Build
```bash
make build
```

### Development Build (with hot reload)
```bash
make run-dev
```

### Production Build
```bash
make build-all  # All platforms
```

### Docker Build
```bash
make docker-build
make docker-run
```

### Cross-Platform Build
```bash
# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o godash-linux .
GOOS=windows GOARCH=amd64 go build -o godash-windows.exe .
GOOS=darwin GOARCH=amd64 go build -o godash-macos .
```

## 🧪 Testing

### Run All Tests
```bash
make test
```

### Run Tests with Coverage
```bash
make coverage
```

### Run Specific Tests
```bash
# Test specific package
go test ./internal/collector

# Test specific function
go test -run TestSystemCollector_GetSystemMetrics ./internal/collector

# Run benchmarks
make bench
```

### Test Options
```bash
# Short tests only
make test-short

# Race detection
make test-race

# Verbose output
go test -v ./...
```

## 📊 CLI Usage Examples

### Basic Usage
```bash
# Single system snapshot
./godash-cli

# Continuous monitoring (5-second intervals)
./godash-cli -continuous -interval=5s

# JSON output
./godash-cli -json

# Show top processes
./godash-cli -processes
```

### Advanced Usage
```bash
# Continuous monitoring with processes, no colors
./godash-cli -continuous -processes -no-color -interval=10s

# Limited updates
./godash-cli -continuous -count=10

# JSON output for scripting
./godash-cli -json > system_metrics.json
```

### Command Line Options
```
  -continuous        Continuous monitoring mode
  -count int         Number of updates (0 for infinite)
  -help              Show help message
  -interval duration Update interval (default 5s)
  -json              Output in JSON format
  -no-color          Disable colored output
  -processes         Show top processes
  -version           Show version information
```

## 🔍 Troubleshooting

### Common Issues

#### 1. Permission Errors
```bash
# Linux/macOS: Run with sudo if needed for system metrics
sudo ./godash-cli

# Windows: Run as Administrator
```

#### 2. Build Errors
```bash
# Clean and rebuild
make clean
make deps
make build
```

#### 3. Dependency Issues
```bash
# Update dependencies
make update-deps

# Verify dependencies
make mod-verify
```

#### 4. Test Failures
```bash
# Run tests with verbose output
go test -v ./...

# Run specific failing test
go test -v -run TestName ./path/to/package
```

### Platform-Specific Issues

#### Linux
- Install `build-essential` if compilation fails
- Some metrics require root privileges

#### Windows
- Use PowerShell or Command Prompt
- Some features may be limited

#### macOS
- Install Xcode Command Line Tools
- May require permissions for system monitoring

### Performance Issues

#### High CPU Usage
```bash
# Check collector interval
./godash-cli -interval=30s  # Increase interval

# Monitor specific metrics only
# (Future feature - metric selection)
```

#### Memory Usage
```bash
# Monitor memory usage
go tool pprof -http=:6060 ./godash
```

## 🚀 Production Deployment

### Systemd Service (Linux)
```ini
# /etc/systemd/system/godash.service
[Unit]
Description=GoDash System Monitor
After=network.target

[Service]
Type=simple
User=godash
ExecStart=/usr/local/bin/godash
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Docker Deployment
```bash
# Build image
docker build -t godash:latest .

# Run container
docker run -d --name godash \
  --restart unless-stopped \
  -p 8080:8080 \
  godash:latest
```

### Environment Variables
```bash
# Configuration
export GODASH_INTERVAL=30s
export GODASH_LOG_LEVEL=info
export GODASH_METRICS_ENABLED=cpu,memory,disk
```

## 📈 Monitoring and Logs

### Application Logs
```bash
# Development
tail -f godash.log

# Production (systemd)
journalctl -u godash -f
```

### Health Checks
```bash
# Future: HTTP health endpoint
curl http://localhost:8080/health
```

### Metrics Export
```bash
# JSON output for monitoring systems
./godash-cli -json | jq '.metrics.cpu.usage_percent'
```

## 🔧 Configuration

### Current Configuration (Week 1)
- Configuration via CLI flags
- No configuration file support (yet)

### Future Configuration (Week 2+)
- YAML configuration files
- Environment variable support
- Database configuration

## 📚 Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [gopsutil Documentation](https://github.com/shirou/gopsutil)
- [Contributing Guide](CONTRIBUTING.md)

## 🆘 Getting Help

1. **Check Documentation**: README.md, this file, code comments
2. **Search Issues**: Look for similar problems
3. **Create Issue**: Provide detailed information
4. **Community**: Join discussions

---

**Need more help?** Open an issue on GitHub or check our documentation!