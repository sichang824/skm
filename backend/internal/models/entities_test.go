package models

import (
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	// Test encoding
	zid, err := Encode("TEST", 123)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if zid == "" {
		t.Fatal("Encoded zid is empty")
	}

	// Test decoding
	prefix, id, err := Decode(zid)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if prefix != "TEST" {
		t.Errorf("Expected prefix 'TEST', got '%s'", prefix)
	}

	if id != 123 {
		t.Errorf("Expected id 123, got %d", id)
	}
}

func TestGetPrefixForTable(t *testing.T) {
	tests := []struct {
		table  string
		prefix string
	}{
		{"providers", "PROV"},
		{"skills", "SKIL"},
		{"scan_jobs", "SCAN"},
		{"scan_issues", "SISS"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			prefix := GetPrefixForTable(tt.table)
			if prefix != tt.prefix {
				t.Errorf("Expected prefix '%s' for table '%s', got '%s'", tt.prefix, tt.table, prefix)
			}
		})
	}
}
