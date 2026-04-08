package services_test

import (
	"testing"

	"github.com/romanzipp/domain-manager/internal/services"
)

func TestExtractTLD(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "com"},
		{"example.co.uk", "co.uk"},
		{"sub.example.org", "example.org"},
		{"example", "example"},
		{"a.b.c.d", "b.c.d"},
	}

	for _, tt := range tests {
		got := services.ExtractTLD(tt.input)
		if got != tt.expected {
			t.Errorf("ExtractTLD(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
