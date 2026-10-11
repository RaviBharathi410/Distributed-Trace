package ingest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/protobuf/proto"
)

// OTLP JSON Schema definitions according to OpenTelemetry specification (v1.0+)
type otlpAnyValue struct {
	StringValue *string  `json:"stringValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
	IntValue    any      `json:"intValue,omitempty"`
	DoubleValue *float64 `json:"doubleValue,omitempty"`
}

func (v *otlpAnyValue) AsString() string {
	if v == nil {
		return ""
	}
	if v.StringValue != nil {
		return *v.StringValue
	}
	if v.BoolValue != nil {
		if *v.BoolValue {
			return "true"
		}
		return "false"
	}
	if v.DoubleValue != nil {
		return fmt.Sprintf("%g", *v.DoubleValue)
	}
	if v.IntValue != nil {
		switch val := v.IntValue.(type) {
		case string:
			return val
		case float64:
			return strconv.FormatInt(int64(val), 10)
		case int64:
			return strconv.FormatInt(val, 10)
		case int:
			return strconv.Itoa(val)
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

type otlpKeyValue struct {
	Key   string       `json:"key"`
	Value otlpAnyValue `json:"value"`
}

type otlpStatus struct {
	Code    int    `json:"code"` // 0 = STATUS_CODE_UNSET, 1 = STATUS_CODE_OK, 2 = STATUS_CODE_ERROR
	Message string `json:"message"`
}

type otlpSpan struct {
	TraceID           string         `json:"traceId"`
	SpanID            string         `json:"spanId"`
	ParentSpanID      string         `json:"parentSpanId,omitempty"`
	Name              string         `json:"name"`
	Kind              int            `json:"kind"`
	StartTimeUnixNano any            `json:"startTimeUnixNano"`
	EndTimeUnixNano   any            `json:"endTimeUnixNano"`
	Attributes        []otlpKeyValue `json:"attributes,omitempty"`
	Status            *otlpStatus    `json:"status,omitempty"`
}

type otlpScopeSpan struct {
	Scope struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"scope"`
	Spans []otlpSpan `json:"spans"`
}

type otlpResourceSpan struct {
	Resource struct {
		Attributes []otlpKeyValue `json:"attributes,omitempty"`
	} `json:"resource"`
	ScopeSpans []otlpScopeSpan `json:"scopeSpans"`
}

type otlpExportRequest struct {
	ResourceSpans []otlpResourceSpan `json:"resourceSpans"`
}

func parseNanoTime(raw any) (int64, error) {
	switch v := raw.(type) {
	case string:
		return strconv.ParseInt(v, 10, 64)
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	default:
		return 0, fmt.Errorf("unexpected time representation: %T", raw)
	}
}

// AnyValueToString converts an OTLP commonpb.AnyValue to its string representation
func AnyValueToString(val *commonpb.AnyValue) string {
	if val == nil {
		return ""
	}
	switch v := val.GetValue().(type) {
	case *commonpb.AnyValue_StringValue:
		return v.StringValue
	case *commonpb.AnyValue_BoolValue:
		if v.BoolValue {
			return "true"
		}
		return "false"
	case *commonpb.AnyValue_IntValue:
		return strconv.FormatInt(v.IntValue, 10)
	case *commonpb.AnyValue_DoubleValue:
		return fmt.Sprintf("%g", v.DoubleValue)
	case *commonpb.AnyValue_BytesValue:
		return hex.EncodeToString(v.BytesValue)
	default:
		return ""
	}
}

// ParseOTLPTracesProto decodes an official binary protobuf ExportTraceServiceRequest payload into []domain.Span
func ParseOTLPTracesProto(body []byte) ([]domain.Span, error) {
	var req coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OTLP protobuf: %w", err)
	}

	var result []domain.Span

	for _, rSpan := range req.ResourceSpans {
		serviceName := "unknown-service"
		resourceTags := make(map[string]string)

		if rSpan.Resource != nil {
			for _, attr := range rSpan.Resource.Attributes {
				valStr := AnyValueToString(attr.Value)
				if attr.Key == "service.name" && valStr != "" {
					serviceName = valStr
				} else {
					resourceTags[attr.Key] = valStr
				}
			}
		}

		for _, sSpan := range rSpan.ScopeSpans {
			for _, span := range sSpan.Spans {
				if len(span.TraceId) == 0 {
					return nil, fmt.Errorf("missing traceId for span")
				}
				if len(span.SpanId) == 0 {
					return nil, fmt.Errorf("missing spanId")
				}

				traceID := hex.EncodeToString(span.TraceId)
				spanID := hex.EncodeToString(span.SpanId)
				parentSpanID := ""
				if len(span.ParentSpanId) > 0 {
					parentSpanID = hex.EncodeToString(span.ParentSpanId)
				}

				startNano := int64(span.StartTimeUnixNano)
				if startNano <= 0 {
					return nil, fmt.Errorf("invalid startTimeUnixNano for span %s", spanID)
				}
				endNano := int64(span.EndTimeUnixNano)
				if endNano < startNano {
					endNano = startNano
				}

				startTime := time.Unix(0, startNano).UTC()
				durationMs := uint32((endNano - startNano) / 1_000_000)
				if durationMs == 0 {
					durationMs = 1 // Floor at 1ms for sub-millisecond spans
				}

				tags := make(map[string]string, len(resourceTags)+len(span.Attributes))
				for k, v := range resourceTags {
					tags[k] = v
				}
				for _, attr := range span.Attributes {
					tags[attr.Key] = AnyValueToString(attr.Value)
				}

				statusCode := uint8(0)
				errorMessage := ""
				if span.Status != nil {
					statusCode = uint8(span.Status.Code)
					errorMessage = span.Status.Message
				}

				result = append(result, domain.Span{
					TraceID:       traceID,
					SpanID:        spanID,
					ParentSpanID:  parentSpanID,
					ServiceName:   serviceName,
					OperationName: span.Name,
					DurationMs:    durationMs,
					StatusCode:    statusCode,
					ErrorMessage:  errorMessage,
					Tags:          tags,
					StartTime:     startTime,
				})
			}
		}
	}

	return result, nil
}

// ParseOTLPTracesJSON parses an OTLP/HTTP JSON ExportTraceServiceRequest payload into domain.Span slice
func ParseOTLPTracesJSON(body []byte) ([]domain.Span, error) {
	var req otlpExportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid OTLP JSON payload: %w", err)
	}

	var result []domain.Span

	for _, rSpan := range req.ResourceSpans {
		serviceName := "unknown-service"
		resourceTags := make(map[string]string)

		for _, attr := range rSpan.Resource.Attributes {
			valStr := attr.Value.AsString()
			if attr.Key == "service.name" && valStr != "" {
				serviceName = valStr
			} else {
				resourceTags[attr.Key] = valStr
			}
		}

		for _, sSpan := range rSpan.ScopeSpans {
			for _, span := range sSpan.Spans {
				startNano, err := parseNanoTime(span.StartTimeUnixNano)
				if err != nil || startNano <= 0 {
					return nil, fmt.Errorf("invalid startTimeUnixNano for span %s: %w", span.SpanID, err)
				}

				endNano, err := parseNanoTime(span.EndTimeUnixNano)
				if err != nil || endNano < startNano {
					endNano = startNano
				}

				startTime := time.Unix(0, startNano).UTC()
				durationMs := uint32((endNano - startNano) / 1_000_000)
				if durationMs == 0 {
					durationMs = 1 // Floor at 1ms for sub-millisecond spans
				}

				tags := make(map[string]string, len(resourceTags)+len(span.Attributes))
				for k, v := range resourceTags {
					tags[k] = v
				}
				for _, attr := range span.Attributes {
					tags[attr.Key] = attr.Value.AsString()
				}

				statusCode := uint8(0)
				errorMessage := ""
				if span.Status != nil {
					statusCode = uint8(span.Status.Code)
					errorMessage = span.Status.Message
				}

				result = append(result, domain.Span{
					TraceID:       span.TraceID,
					SpanID:        span.SpanID,
					ParentSpanID:  span.ParentSpanID,
					ServiceName:   serviceName,
					OperationName: span.Name,
					DurationMs:    durationMs,
					StatusCode:    statusCode,
					ErrorMessage:  errorMessage,
					Tags:          tags,
					StartTime:     startTime,
				})
			}
		}
	}

	return result, nil
}

// ParseOTLPTraces auto-detects Content-Type (protobuf vs JSON) and parses into []domain.Span
func ParseOTLPTraces(body []byte, contentType string) ([]domain.Span, error) {
	if strings.Contains(contentType, "application/json") {
		return ParseOTLPTracesJSON(body)
	}

	// Default to protobuf (standard OTel behavior)
	spans, err := ParseOTLPTracesProto(body)
	if err == nil {
		return spans, nil
	}

	// If protobuf unmarshal fails and request was not explicitly tagged protobuf, fallback to JSON
	if !strings.Contains(contentType, "application/x-protobuf") {
		if jsonSpans, jsonErr := ParseOTLPTracesJSON(body); jsonErr == nil {
			return jsonSpans, nil
		}
	}

	return nil, err
}

// BuildOTLPProtoSuccessResponse serializes a standard empty ExportTraceServiceResponse proto message
func BuildOTLPProtoSuccessResponse() ([]byte, error) {
	resp := &coltracepb.ExportTraceServiceResponse{}
	return proto.Marshal(resp)
}
