package domain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidTraceIDLength = errors.New("trace_id must be 32 hexadecimal characters (16 bytes)")
	ErrZeroTraceID          = errors.New("trace_id cannot be all zero bytes")
	ErrInvalidSpanIDLength  = errors.New("span_id must be 16 hexadecimal characters (8 bytes)")
	ErrZeroSpanID           = errors.New("span_id cannot be all zero bytes")
	ErrInvalidParentSpanID  = errors.New("parent_span_id must be empty or 16 hexadecimal characters (8 bytes)")
)

// ZeroSpanIDBytes represents the 8-byte zero literal used for root span parent IDs in ClickHouse FixedString(8).
var ZeroSpanIDBytes = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}

// ParseTraceIDHex parses and validates a 32-character hex trace ID into a 16-byte binary array.
// Returns an error if the hex string is malformed or represents all zero bytes.
func ParseTraceIDHex(hexStr string) ([16]byte, error) {
	var id [16]byte
	hexStr = strings.TrimSpace(hexStr)
	if len(hexStr) != 32 {
		return id, fmt.Errorf("%w: got length %d", ErrInvalidTraceIDLength, len(hexStr))
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return id, fmt.Errorf("invalid hexadecimal trace_id: %w", err)
	}
	copy(id[:], b)
	if id == [16]byte{} {
		return id, ErrZeroTraceID
	}
	return id, nil
}

// FormatTraceID formats a 16-byte binary trace ID into a 32-character lowercase hex string.
func FormatTraceID(id [16]byte) string {
	return hex.EncodeToString(id[:])
}

// ParseSpanIDHex parses and validates a 16-character hex span ID into an 8-byte binary array.
// Returns an error if the hex string is malformed or represents all zero bytes.
func ParseSpanIDHex(hexStr string) ([8]byte, error) {
	var id [8]byte
	hexStr = strings.TrimSpace(hexStr)
	if len(hexStr) != 16 {
		return id, fmt.Errorf("%w: got length %d", ErrInvalidSpanIDLength, len(hexStr))
	}
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return id, fmt.Errorf("invalid hexadecimal span_id: %w", err)
	}
	copy(id[:], b)
	if id == [8]byte{} {
		return id, ErrZeroSpanID
	}
	return id, nil
}

// FormatSpanID formats an 8-byte binary span ID into a 16-character lowercase hex string.
func FormatSpanID(id [8]byte) string {
	return hex.EncodeToString(id[:])
}

// ParseParentSpanIDHex parses a parent span ID.
// Root spans (empty string or 16 zeroes) return ZeroSpanIDBytes with nil error.
func ParseParentSpanIDHex(hexStr string) ([8]byte, error) {
	hexStr = strings.TrimSpace(hexStr)
	if hexStr == "" || hexStr == "0000000000000000" {
		return ZeroSpanIDBytes, nil
	}
	if len(hexStr) != 16 {
		return ZeroSpanIDBytes, fmt.Errorf("%w: got length %d", ErrInvalidParentSpanID, len(hexStr))
	}
	var id [8]byte
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return ZeroSpanIDBytes, fmt.Errorf("invalid hexadecimal parent_span_id: %w", err)
	}
	copy(id[:], b)
	return id, nil
}

// FormatParentSpanID formats an 8-byte binary parent span ID.
// Root spans (all zero bytes) decode back to an empty string "".
func FormatParentSpanID(id [8]byte) string {
	if id == ZeroSpanIDBytes {
		return ""
	}
	return hex.EncodeToString(id[:])
}
