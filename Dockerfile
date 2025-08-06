# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for downloading dependencies)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o godash .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh godash

# Set working directory
WORKDIR /home/godash

# Copy the binary from builder stage
COPY --from=builder /app/godash .

# Copy configuration files (if any)
COPY --from=builder /app/configs ./configs

# Copy web assets (for future web interface)
COPY --from=builder /app/web ./web

# Create directories for data and logs
RUN mkdir -p data logs && chown -R godash:godash /home/godash

# Switch to non-root user
USER godash

# Expose port (for future web interface)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["./godash"]

# Metadata
LABEL maintainer="GoDash Team"
LABEL version="1.0.0"
LABEL description="GoDash System Monitoring Tool"
LABEL org.opencontainers.image.source="https://github.com/eyzaun/godash"