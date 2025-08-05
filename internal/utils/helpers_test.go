package utils

import (
	"testing"
)

func TestGetHostname(t *testing.T) {
	hostname := GetHostname()
	if hostname == "" {
		t.Error("GetHostname() returned empty string")
	}
	// Should not be empty and should not be "unknown" in normal circumstances
	if len(hostname) < 1 {
		t.Error("GetHostname() returned invalid hostname")
	}
}

func TestGetPlatform(t *testing.T) {
	platform := GetPlatform()
	if platform == "" {
		t.Error("GetPlatform() returned empty string")
	}
	// Should contain OS and architecture separated by /
	if len(platform) < 3 {
		t.Error("GetPlatform() returned invalid format")
	}
}

func TestSafeDivide(t *testing.T) {
	tests := []struct {
		numerator   float64
		denominator float64
		expected    float64
	}{
		{10.0, 2.0, 5.0},
		{10.0, 0.0, 0.0},
		{0.0, 5.0, 0.0},
	}

	for _, test := range tests {
		result := SafeDivide(test.numerator, test.denominator)
		if result != test.expected {
			t.Errorf("SafeDivide(%.1f, %.1f) = %.1f; expected %.1f",
				test.numerator, test.denominator, result, test.expected)
		}
	}
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		used     uint64
		total    uint64
		expected float64
	}{
		{50, 100, 50.0},
		{0, 100, 0.0},
		{100, 0, 0.0},
		{75, 100, 75.0},
	}

	for _, test := range tests {
		result := CalculatePercentage(test.used, test.total)
		if result != test.expected {
			t.Errorf("CalculatePercentage(%d, %d) = %.1f; expected %.1f",
				test.used, test.total, result, test.expected)
		}
	}
}

// Benchmark tests for performance-critical functions
func BenchmarkCalculatePercentage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculatePercentage(1234567890, 2345678901)
	}
}

func BenchmarkSafeDivide(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SafeDivide(123.456, 78.9)
	}
}
