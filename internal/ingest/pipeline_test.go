package ingest

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/segmentio/kafka-go"
)

// ChannelMessageBus provides an in-memory Kafka transport for testing producer -> consumer pipeline
type ChannelMessageBus struct {
	mu        sync.Mutex
	ch        chan kafka.Message
	committed []kafka.Message
	closed    bool
}

func NewChannelMessageBus(bufferSize int) *ChannelMessageBus {
	return &ChannelMessageBus{
		ch: make(chan kafka.Message, bufferSize),
	}
}

func (b *ChannelMessageBus) PublishSpans(ctx context.Context, spans []domain.Span) error {
	for i, s := range spans {
		data, err := json.Marshal(s)
		if err != nil {
			return err
		}
		msg := kafka.Message{
			Topic:  "otel-spans",
			Key:    []byte(s.TraceID),
			Value:  data,
			Offset: int64(i),
		}
		select {
		case b.ch <- msg:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (b *ChannelMessageBus) FetchMessage(ctx context.Context) (kafka.Message, error) {
	select {
	case msg, ok := <-b.ch:
		if !ok {
			return kafka.Message{}, context.Canceled
		}
		return msg, nil
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	}
}

func (b *ChannelMessageBus) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.committed = append(b.committed, msgs...)
	return nil
}

func (b *ChannelMessageBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		b.closed = true
		close(b.ch)
	}
	return nil
}

func TestEndToEndIngestionPipeline_TenantIsolationAndBatchCommit(t *testing.T) {
	bus := NewChannelMessageBus(100)
	defer bus.Close()

	writer := &mockSpanBatchWriter{}

	cfg := ConsumerConfig{
		Topic:         "otel-spans",
		BatchSize:     3,
		FlushInterval: 50 * time.Millisecond,
	}

	consumer := NewSpanConsumerWithReader(bus, writer, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumer.Run(ctx)
	}()

	// 1. Simulate multi-tenant span batches published by IngestSpans
	tenantASpans := []domain.Span{
		{
			TraceID:       "trace-alpha-1",
			SpanID:        "span-a1",
			ServiceName:   "order-api",
			OperationName: "Checkout",
			DurationMs:    100,
			StartTime:     time.Now(),
		},
		{
			TraceID:       "trace-alpha-1",
			SpanID:        "span-a2",
			ServiceName:   "payment-api",
			OperationName: "Charge",
			DurationMs:    45,
			StartTime:     time.Now(),
		},
	}

	tenantBSpans := []domain.Span{
		{
			TraceID:       "trace-beta-1",
			SpanID:        "span-b1",
			ServiceName:   "inventory-api",
			OperationName: "ReserveStock",
			DurationMs:    30,
			StartTime:     time.Now(),
		},
	}

	// Enrich with authentic org IDs (simulating auth context injection)
	enrichedA := EnrichSpans(tenantASpans, "org-alpha-111")
	enrichedB := EnrichSpans(tenantBSpans, "org-beta-222")

	// Publish via bus
	if err := bus.PublishSpans(ctx, enrichedA); err != nil {
		t.Fatalf("failed to publish tenant A spans: %v", err)
	}
	if err := bus.PublishSpans(ctx, enrichedB); err != nil {
		t.Fatalf("failed to publish tenant B spans: %v", err)
	}

	// 2. Wait for batch flush (total 3 spans matches BatchSize=3)
	deadline := time.Now().Add(2 * time.Second)
	for {
		writer.mu.Lock()
		bCount := len(writer.batches)
		writer.mu.Unlock()
		if bCount >= 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	<-consumerDone

	// 3. Verify assertions
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if len(writer.batches) != 1 {
		t.Fatalf("expected 1 batch written to ClickHouse, got %d", len(writer.batches))
	}

	batch := writer.batches[0]
	if len(batch) != 3 {
		t.Fatalf("expected 3 spans in batch, got %d", len(batch))
	}

	// Verify tenant isolation in persisted batch rows
	if batch[0].OrgID != "org-alpha-111" || batch[1].OrgID != "org-alpha-111" {
		t.Errorf("tenant A spans corrupted in pipeline: %+v, %+v", batch[0], batch[1])
	}
	if batch[2].OrgID != "org-beta-222" {
		t.Errorf("tenant B span corrupted in pipeline: %+v", batch[2])
	}

	// Verify Kafka offsets were committed
	bus.mu.Lock()
	defer bus.mu.Unlock()
	if len(bus.committed) != 3 {
		t.Fatalf("expected 3 committed offsets in Kafka, got %d", len(bus.committed))
	}
}
