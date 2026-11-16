package domain

import (
	"errors"
	"testing"
)

func TestParseTraceIDHex(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		wantErr error
	}{
		{
			name:    "valid trace id",
			hexStr:  "4bf92f3577b34da6a3ce929d0e0e4736",
			wantErr: nil,
		},
		{
			name:    "invalid length short",
			hexStr:  "4bf92f35",
			wantErr: ErrInvalidTraceIDLength,
		},
		{
			name:    "invalid length long",
			hexStr:  "4bf92f3577b34da6a3ce929d0e0e473600",
			wantErr: ErrInvalidTraceIDLength,
		},
		{
			name:    "all zero bytes",
			hexStr:  "00000000000000000000000000000000",
			wantErr: ErrZeroTraceID,
		},
		{
			name:    "non hex characters",
			hexStr:  "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: errors.New("invalid hexadecimal trace_id"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := ParseTraceIDHex(tt.hexStr)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error containing %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				gotHex := FormatTraceID(b)
				if gotHex != tt.hexStr {
					t.Fatalf("round-trip mismatch: expected %s, got %s", tt.hexStr, gotHex)
				}
			}
		})
	}
}

func TestParseSpanIDHex(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		wantErr error
	}{
		{
			name:    "valid span id",
			hexStr:  "00f067aa0ba902b7",
			wantErr: nil,
		},
		{
			name:    "invalid length short",
			hexStr:  "00f067aa",
			wantErr: ErrInvalidSpanIDLength,
		},
		{
			name:    "all zero bytes",
			hexStr:  "0000000000000000",
			wantErr: ErrZeroSpanID,
		},
		{
			name:    "non hex characters",
			hexStr:  "zzzzzzzzzzzzzzzz",
			wantErr: errors.New("invalid hexadecimal span_id"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := ParseSpanIDHex(tt.hexStr)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error containing %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				gotHex := FormatSpanID(b)
				if gotHex != tt.hexStr {
					t.Fatalf("round-trip mismatch: expected %s, got %s", tt.hexStr, gotHex)
				}
			}
		})
	}
}

func TestParseParentSpanIDHex_RootSpanDecodesToEmpty(t *testing.T) {
	// Root span with empty string
	b1, err := ParseParentSpanIDHex("")
	if err != nil {
		t.Fatalf("unexpected error for empty parent span: %v", err)
	}
	if b1 != ZeroSpanIDBytes {
		t.Fatalf("expected ZeroSpanIDBytes, got %v", b1)
	}
	if dec := FormatParentSpanID(b1); dec != "" {
		t.Fatalf("expected empty string decoded from zero bytes, got %q", dec)
	}

	// Root span with 16 zeroes
	b2, err := ParseParentSpanIDHex("0000000000000000")
	if err != nil {
		t.Fatalf("unexpected error for zero parent span: %v", err)
	}
	if b2 != ZeroSpanIDBytes {
		t.Fatalf("expected ZeroSpanIDBytes, got %v", b2)
	}
	if dec := FormatParentSpanID(b2); dec != "" {
		t.Fatalf("expected empty string decoded from zero bytes, got %q", dec)
	}

	// Valid child span parent
	childHex := "00f067aa0ba902b7"
	b3, err := ParseParentSpanIDHex(childHex)
	if err != nil {
		t.Fatalf("unexpected error for child parent span: %v", err)
	}
	if dec := FormatParentSpanID(b3); dec != childHex {
		t.Fatalf("round-trip mismatch: expected %s, got %s", childHex, dec)
	}
}

