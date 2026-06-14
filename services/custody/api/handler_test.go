package api

import (
	"testing"
)

func TestParseDecimalAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"12.345", "12345000", false},
		{"0.000001", "1", false},
		{"10", "10000000", false},
		{"0", "", true}, // Must be greater than zero
		{"-5.5", "", true},
		{"12.3456789", "", true},
		{"12a.34", "", true},
		{"", "", true},
		{".5", "500000", false},
		{"5.", "5000000", false},
		{" 1.5 ", "1500000", false},
		{"1.5000000000", "1500000", false}, // Trailing zeros should be normalized out
		{"1.0000000001", "", true},       // Exceeds 6 decimal places
		{"1e-6", "1", false},              // Scientific notation (1 * 10^-6)
	}

	for _, tt := range tests {
		got, err := parseDecimalAmount(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDecimalAmount(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("parseDecimalAmount(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
