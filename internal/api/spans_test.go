package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/auth"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	chRepo "github.com/RaviBharathi410/distributedtrace/internal/repository/clickhouse"
)

type mockSpanPublisher struct {
	publishedSpans []domain.Span
	publishErr     error
}

func (m *mockSpanPublisher) PublishSpans(ctx context.Context, spans []domain.Span) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.publishedSpans = append(m.publishedSpans, spans...)
	return nil
}

func TestSpanHandler_DecoupledPipeline(t *testing.T) {
	now := time.Now()
	validSpan := domain.Span{
		TraceID:       "trace-producer-1",
		SpanID:        "span-1",
		ServiceName:   "order-service",
		OperationName: "CreateOrder",
		DurationMs:    150,
		StatusCode:    1,
		StartTime:     now,
	}

	t.Run("Publishes to Kafka when producer is configured (does not call ClickHouse)", func(t *testing.T) {
		mockConn := &apiMockConn{}
		traceRepo := chRepo.NewTraceRepository(mockConn)
		mockPub := &mockSpanPublisher{}
		handler := NewSpanHandler(traceRepo, mockPub)

		body, _ := json.Marshal([]domain.Span{validSpan})
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Assert publisher received the enriched span
		if len(mockPub.publishedSpans) != 1 {
			t.Fatalf("expected 1 span published to Kafka, got %d", len(mockPub.publishedSpans))
		}
		if mockPub.publishedSpans[0].OrgID != TestOrgA {
			t.Errorf("expected published span org_id to be %q, got %q", TestOrgA, mockPub.publishedSpans[0].OrgID)
		}

		// Assert ClickHouse was NOT called (no rows appended to batch)
		if len(mockConn.lastQuery) > 0 {
			t.Errorf("expected ClickHouse not to be queried, but got lastQuery: %s", mockConn.lastQuery)
		}
	})

	t.Run("Returns 500 when Kafka producer fails", func(t *testing.T) {
		mockConn := &apiMockConn{}
		traceRepo := chRepo.NewTraceRepository(mockConn)
		mockPub := &mockSpanPublisher{
			publishErr: errors.New("kafka broker connection refused"),
		}
		handler := NewSpanHandler(traceRepo, mockPub)

		body, _ := json.Marshal([]domain.Span{validSpan})
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 Internal Server Error, got %d", rec.Code)
		}
	})

	t.Run("Falls back to direct ClickHouse insert when producer is nil", func(t *testing.T) {
		mockConn := &apiMockConn{}
		traceRepo := chRepo.NewTraceRepository(mockConn)
		handler := NewSpanHandler(traceRepo, nil)

		body, _ := json.Marshal([]domain.Span{validSpan})
		ctx := context.WithValue(context.Background(), auth.OrgIDKey, TestOrgA)
		req := httptest.NewRequest(http.MethodPost, "/spans", bytes.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.IngestSpans(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d", rec.Code)
		}
	})
}
