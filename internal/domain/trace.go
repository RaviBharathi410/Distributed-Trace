package domain

import "time"

type Span struct {
	TraceID        string            `json:"trace_id"`
	SpanID         string            `json:"span_id"`
	ParentSpanID   string            `json:"parent_span_id,omitempty"`
	ServiceName    string            `json:"service_name"`
	OperationName  string            `json:"operation_name"`
	DurationMs     uint32            `json:"duration_ms"`
	StatusCode     uint8             `json:"status_code"` // 0 = Unset, 1 = Ok, 2 = Error
	ErrorMessage   string            `json:"error_message,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	StartTime      time.Time         `json:"start_time"`
	ReceivedAt     time.Time         `json:"received_at,omitempty"`
	OrgID          string            `json:"org_id,omitempty"`
	StartOffsetMs  uint32            `json:"start_offset_ms,omitempty"`
}

type TraceDetail struct {
	TraceID       string `json:"trace_id"`
	RootService   string `json:"root_service"`
	RootOperation string `json:"root_operation"`
	DurationMs    uint32 `json:"duration_ms"`
	SpanCount     int    `json:"span_count"`
	ErrorCount    int    `json:"error_count"`
	StartTime     int64  `json:"start_time"` // unix timestamp ms
	Spans         []Span `json:"spans"`
}

type TraceRow struct {
	TraceID       string `json:"trace_id"`
	RootService   string `json:"root_service"`
	RootOperation string `json:"root_operation"`
	DurationMs    uint32 `json:"duration_ms"`
	SpanCount     int    `json:"span_count"`
	ErrorCount    int    `json:"error_count"`
	StartTime     int64  `json:"start_time"` // unix timestamp ms
}

type TraceFilter struct {
	Service       string    `json:"service"`
	Status        string    `json:"status"` // "all", "success", "error"
	From          time.Time `json:"from"`
	To            time.Time `json:"to"`
	MinDurationMs int       `json:"min_duration_ms"`
	MaxDurationMs int       `json:"max_duration_ms"`
	Limit         int       `json:"limit"`
	Offset        int       `json:"offset"`
}
