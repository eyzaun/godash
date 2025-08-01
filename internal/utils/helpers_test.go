package utils

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.input)
		if result != test.expected {
			t.Errorf("FormatBytes(%d) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestFormatBytesDecimal(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1000, "1.0 kB"},
		{1500, "1.5 kB"},
		{1000000, "1.0 MB"},
		{1000000000, "1.0 GB"},
	}

	for _, test := range tests {
		result := FormatBytesDecimal(test.input)
		if result != test.expected {
			t.Errorf("FormatBytesDecimal(%d) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		hasError bool
	}{
		{"1024", 1024, false},
		{"1 KB", 1024, false},
		{"1.5 KB", 1536, false},
		{"1 MB", 1048576, false},
		{"1 GB", 1073741824, false},
		{"invalid", 0, true},
			   {"1.5 XB", 1, false},
		{"", 0, true},
		{" KB", 0, true}, // No number before unit
		{"-1 KB", 0, true}, // Negative number
	}

	for _, test := range tests {
		result, err := ParseBytes(test.input)
		hasError := err != nil

		if hasError != test.hasError {
			t.Errorf("ParseBytes(%s) error status = %v; expected %v (error: %v)", test.input, hasError, test.hasError, err)
			continue
		}

		if !hasError && result != test.expected {
			t.Errorf("ParseBytes(%s) = %d; expected %d", test.input, result, test.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3661 * time.Second, "1h1m"},
		{90061 * time.Second, "1d1h"},
	}

	for _, test := range tests {
		result := FormatDuration(test.input)
		if result != test.expected {
			t.Errorf("FormatDuration(%v) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		percent  float64
		decimals int
		expected string
	}{
		{50.0, 1, "50.0%"},
		{33.333, 2, "33.33%"},
		{66.666, 0, "67%"},
	}

	for _, test := range tests {
		result := FormatPercent(test.percent, test.decimals)
		if result != test.expected {
			t.Errorf("FormatPercent(%.3f, %d) = %s; expected %s", test.percent, test.decimals, result, test.expected)
		}
	}
}

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		value     float64
		precision int
		expected  float64
	}{
		{3.14159, 2, 3.14},
		{3.14159, 4, 3.1416},
		{3.0, 2, 3.0},
	}

	for _, test := range tests {
		result := RoundFloat(test.value, test.precision)
		if result != test.expected {
			t.Errorf("RoundFloat(%.5f, %d) = %.5f; expected %.5f", test.value, test.precision, result, test.expected)
		}
	}
}

func TestClampFloat(t *testing.T) {
	tests := []struct {
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{5.0, 0.0, 10.0, 5.0},
		{-5.0, 0.0, 10.0, 0.0},
		{15.0, 0.0, 10.0, 10.0},
	}

	for _, test := range tests {
		result := ClampFloat(test.value, test.min, test.max)
		if result != test.expected {
			t.Errorf("ClampFloat(%.1f, %.1f, %.1f) = %.1f; expected %.1f", 
				test.value, test.min, test.max, result, test.expected)
		}
	}
}

func TestClampPercent(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{50.0, 50.0},
		{-10.0, 0.0},
		{150.0, 100.0},
	}

	for _, test := range tests {
		result := ClampPercent(test.input)
		if result != test.expected {
			t.Errorf("ClampPercent(%.1f) = %.1f; expected %.1f", test.input, result, test.expected)
		}
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

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hel"},
	}

	for _, test := range tests {
		result := TruncateString(test.input, test.length)
		if result != test.expected {
			t.Errorf("TruncateString(%s, %d) = %s; expected %s", 
				test.input, test.length, result, test.expected)
		}
	}
}

func TestPadString(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"hello", 10, "hello     "},
		{"hello world", 8, "hello world"},
		{"", 5, "     "},
	}

	for _, test := range tests {
		result := PadString(test.input, test.length)
		if result != test.expected {
			t.Errorf("PadString(%s, %d) = %s; expected %s", 
				test.input, test.length, result, test.expected)
		}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}
	
	tests := []struct {
		item     string
		expected bool
	}{
		{"apple", true},
		{"banana", true},
		{"orange", false},
		{"", false},
	}

	for _, test := range tests {
		result := Contains(slice, test.item)
		if result != test.expected {
			t.Errorf("Contains(slice, %s) = %v; expected %v", test.item, result, test.expected)
		}
	}
}

func TestRemoveFromSlice(t *testing.T) {
	slice := []string{"apple", "banana", "cherry", "banana"}
	result := RemoveFromSlice(slice, "banana")
	expected := []string{"apple", "cherry"}
	
	if len(result) != len(expected) {
		t.Errorf("RemoveFromSlice length = %d; expected %d", len(result), len(expected))
		return
	}
	
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("RemoveFromSlice result[%d] = %s; expected %s", i, v, expected[i])
		}
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{5, 10, 10},
		{10, 5, 10},
		{5, 5, 5},
		{-5, -10, -5},
	}

	for _, test := range tests {
		result := MaxInt(test.a, test.b)
		if result != test.expected {
			t.Errorf("MaxInt(%d, %d) = %d; expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{5, 5, 5},
		{-5, -10, -10},
	}

	for _, test := range tests {
		result := MinInt(test.a, test.b)
		if result != test.expected {
			t.Errorf("MinInt(%d, %d) = %d; expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestGetMemoryPressureLevel(t *testing.T) {
	tests := []struct {
		percent  float64
		expected string
	}{
		{25.0, "low"},
		{60.0, "moderate"},
		{85.0, "high"},
		{95.0, "critical"},
	}

	for _, test := range tests {
		result := GetMemoryPressureLevel(test.percent)
		if result != test.expected {
			t.Errorf("GetMemoryPressureLevel(%.1f) = %s; expected %s", 
				test.percent, result, test.expected)
		}
	}
}

func TestGetDiskHealthLevel(t *testing.T) {
	tests := []struct {
		percent  float64
		expected string
	}{
		{50.0, "healthy"},
		{85.0, "warning"},
		{95.0, "critical"},
	}

	for _, test := range tests {
		result := GetDiskHealthLevel(test.percent)
		if result != test.expected {
			t.Errorf("GetDiskHealthLevel(%.1f) = %s; expected %s", 
				test.percent, result, test.expected)
		}
	}
}

func TestGetCPULoadLevel(t *testing.T) {
	tests := []struct {
		percent  float64
		expected string
	}{
		{30.0, "low"},
		{70.0, "moderate"},
		{90.0, "high"},
		{98.0, "critical"},
	}

	for _, test := range tests {
		result := GetCPULoadLevel(test.percent)
		if result != test.expected {
			t.Errorf("GetCPULoadLevel(%.1f) = %s; expected %s", 
				test.percent, result, test.expected)
		}
	}
}

func TestValidatePercentage(t *testing.T) {
	tests := []struct {
		percent  float64
		hasError bool
	}{
		{50.0, false},
		{0.0, false},
		{100.0, false},
		{-10.0, true},
		{150.0, true},
	}

	for _, test := range tests {
		err := ValidatePercentage(test.percent)
		hasError := err != nil
		
		if hasError != test.hasError {
			t.Errorf("ValidatePercentage(%.1f) error status = %v; expected %v", 
				test.percent, hasError, test.hasError)
		}
	}
}

func TestSanitizeHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"valid-hostname", "valid-hostname"},
		{"hostname.domain.com", "hostname.domain.com"},
		{"invalid@hostname!", "invalid_hostname_"},
		{"", "unknown"},
		{"..-invalid-..", "invalid"},
		{"123-abc-XYZ", "123-abc-XYZ"},
	}

	for _, test := range tests {
		result := SanitizeHostname(test.input)
		if result != test.expected {
			t.Errorf("SanitizeHostname(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

// Benchmark tests
func BenchmarkFormatBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatBytes(1234567890)
	}
}

func BenchmarkCalculatePercentage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculatePercentage(1234567890, 2345678901)
	}
}

func BenchmarkTruncateString(b *testing.B) {
	longString := "This is a very long string that needs to be truncated"
	for i := 0; i < b.N; i++ {
		TruncateString(longString, 20)
	}
}