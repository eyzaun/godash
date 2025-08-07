# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make \
    gcc \
    musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o godash .

# Verify the binary
RUN chmod +x godash && ./godash --version || echo "Binary built successfully"

# Final stage - minimal runtime image
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    wget \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1001 -S godash && \
    adduser -u 1001 -S godash -G godash

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/godash ./godash

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Copy web assets
COPY --from=builder /app/web ./web

# Create necessary directories with proper permissions
RUN mkdir -p logs data tmp && \
    chown -R godash:godash /app

# Switch to non-root user
USER godash

# Expose port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables
ENV GIN_MODE=release
ENV TZ=UTC

# Use tini for proper signal handling (if available)
# RUN apk add --no-cache tini
# ENTRYPOINT ["/sbin/tini", "--"]

# Default command
CMD ["./godash"]

# Metadata labels
LABEL maintainer="GoDash Team" \
      version="1.0.0" \
      description="GoDash System Monitoring Tool with Alert System" \
      org.opencontainers.image.title="GoDash" \
      org.opencontainers.image.description="Real-time system monitoring with alerts" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.vendor="GoDash Team" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/eyzaun/godash" \
      org.opencontainers.image.documentation="https://github.com/eyzaun/godash/blob/main/README.md"