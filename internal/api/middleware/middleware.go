package middleware

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Logger returns a gin middleware for logging requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()

		// Get request method
		method := c.Request.Method

		// Get status code
		statusCode := c.Writer.Status()

		// Get response size
		bodySize := c.Writer.Size()

		// Build query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Get request ID from context
		requestID := c.GetString("X-Request-ID")

		// Log the request
		log.Printf("[%s] %s %s %d %v %d %s",
			requestID,
			method,
			path,
			statusCode,
			latency,
			bodySize,
			clientIP,
		)
	}
}

// RequestID generates and adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get request ID from header first
		requestID := c.GetHeader("X-Request-ID")

		// If not present, generate a new one
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Set request ID in context and response header
		c.Set("X-Request-ID", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return uuid.New().String()[:8] // Use first 8 characters for brevity
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Updated CSP to allow external CDNs and inline styles/scripts for dashboard
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; " +
			"style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com https://fonts.googleapis.com; " +
			"font-src 'self' https://cdnjs.cloudflare.com https://fonts.gstatic.com; " +
			"img-src 'self' data: blob:; " +
			"connect-src 'self' ws: wss:; " +
			"object-src 'none'; " +
			"base-uri 'self'"

		c.Header("Content-Security-Policy", csp)

		// Only add HSTS for HTTPS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// APIVersion adds API version information to responses
func APIVersion() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-API-Version", "v1")
		c.Next()
	}
}

// BasicAuth provides basic authentication middleware
func BasicAuth() gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{
		"admin": "godash2024", // In production, use environment variables
	})
}

// rateLimiter implements a simple in-memory rate limiter
type rateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
}

// allow checks if the client is allowed to make a request
func (rl *rateLimiter) allow(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// Initialize or clean up client's request history
	if requests, exists := rl.requests[clientIP]; exists {
		// Remove old requests
		validRequests := make([]time.Time, 0, len(requests))
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[clientIP] = validRequests
	} else {
		rl.requests[clientIP] = make([]time.Time, 0)
	}

	// Check if client has exceeded rate limit (100 requests per minute)
	if len(rl.requests[clientIP]) >= 100 {
		return false
	}

	// Add current request
	rl.requests[clientIP] = append(rl.requests[clientIP], now)
	return true
}

// cleanup removes old entries from the rate limiter
func (rl *rateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)

	for clientIP, requests := range rl.requests {
		if len(requests) == 0 {
			delete(rl.requests, clientIP)
			continue
		}

		// If the latest request is older than cutoff, remove this client
		if requests[len(requests)-1].Before(cutoff) {
			delete(rl.requests, clientIP)
		}
	}
}

// RateLimit implements a simple rate limiting middleware
func RateLimit() gin.HandlerFunc {
	limiter := &rateLimiter{
		requests: make(map[string][]time.Time),
	}

	// Cleanup old requests every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !limiter.allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from this IP address",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ErrorHandler middleware for handling errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Log the error
			requestID := c.GetString("X-Request-ID")
			log.Printf("[%s] Error: %v", requestID, err.Err)

			// Return appropriate error response
			switch err.Type {
			case gin.ErrorTypeBind:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Bad Request",
					"message": "Invalid request format",
				})
			case gin.ErrorTypePublic:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": err.Error(),
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "An unexpected error occurred",
				})
			}
		}
	}
}

// Timeout middleware for handling request timeouts
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap the request with a timeout context
		ctx := c.Request.Context()
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(timeoutCtx)

		// Channel to signal completion
		finished := make(chan struct{})

		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		select {
		case <-finished:
			// Request completed within timeout
			return
		case <-timeoutCtx.Done():
			// Request timed out
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":   "Request Timeout",
				"message": fmt.Sprintf("Request timed out after %v", timeout),
			})
			c.Abort()
			return
		}
	}
}

// JSONContentType middleware ensures content type is application/json for POST/PUT requests
func JSONContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "" && contentType != "application/json" {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error":   "Unsupported Media Type",
					"message": "Content-Type must be application/json",
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// ValidateAPIKey middleware for API key validation (if needed)
func ValidateAPIKey(validAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "API key is required",
			})
			c.Abort()
			return
		}

		// Use constant time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validAPIKey)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid API key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
