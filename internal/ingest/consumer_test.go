package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/segmentio/kafka-go"
)

type mockMessageReader struct {
	mu           sync.Mutex
	messages     []kafka.Message
	committed    []kafka.Message
	closed       bool
	fetchDelay   time.Duration
	fetchBlockCh chan struct{}
}

func newMockMessageReader(msgs []kafka.Message) *mockMessageReader {
	return &mockMessageReader{
		messages:     msgs,
		fetchBlockCh: make(chan struct{}),
	}
}

func (m *mockMessageReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	m.mu.Lock()
	if len(m.messages) > 0 {
		msg := m.messages[0]
		m.messages = m.messages[1:]
		m.mu.Unlock()
		return msg, nil
	}
	m.mu.Unlock()

	// Block until context canceled or closed
	select {
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	case <-m.fetchBlockCh:
		return kafka.Message{}, errors.New("reader closed")
	}
}

func (m *mockMessageReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.committed = append(m.committed, msgs...)
	return nil
}

func (m *mockMessageReader) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	select {
	case <-m.fetchBlockCh:
	default:
		close(m.fetchBlockCh)
	}
	return nil
}

type mockSpanBatchWriter struct {
	mu            sync.Mutex
	batches       [][]domain.Span
	failWithErr   error
	failCallCount int
	calls         int
}

func (m *mockSpanBatchWriter) InsertSpansBatch(ctx context.Context, spans []domain.Span) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.failWithErr != nil && (m.failCallCount == 0 || m.calls <= m.failCallCount) {
		return m.failWithErr
	}
	// Make a copy of spans
	batchCopy := make([]domain.Span, len(spans))
	copy(batchCopy, spans)
	m.batches = append(m.batches, batchCopy)
	return nil
}

func TestSpanConsumer_BatchSizeFlush(t *testing.T) {
	// 5 valid spans
	var testMsgs []kafka.Message
	for i := 0; i < 5; i++ {
		span := domain.Span{
			OrgID:       "org-alpha",
			TraceID:     "trace-batch",
			SpanID:      "span-test",
			ServiceName: "auth-svc",
			DurationMs:  50,
			StartTime:   time.Now(),
		}
		data, _ := json.Marshal(span)
		testMsgs = append(testMsgs, kafka.Message{
			Topic:     "otel-spans",
			Partition: 0,
			Offset:    int64(i),
			Value:     data,
		})
	}

	mockReader := newMockMessageReader(testMsgs)
	mockWriter := &mockSpanBatchWriter{}

	cfg := ConsumerConfig{
		Topic:         "otel-spans",
		BatchSize:     5,
		FlushInterval: 10 * time.Second, // Long ticker so only batch size triggers flush
	}

	consumer := NewSpanConsumerWithReader(mockReader, mockWriter, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumer.Run(ctx)
	}()

	// Wait for consumer to process batch
	deadline := time.Now().Add(2 * time.Second)
	for {
		mockWriter.mu.Lock()
		bCount := len(mockWriter.batches)
		mockWriter.mu.Unlock()
		if bCount >= 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	<-consumerDone

	mockWriter.mu.Lock()
	defer mockWriter.mu.Unlock()
	if len(mockWriter.batches) != 1 {
		t.Fatalf("expected exactly 1 batch written, got %d", len(mockWriter.batches))
	}
	if len(mockWriter.batches[0]) != 5 {
		t.Fatalf("expected batch size 5, got %d", len(mockWriter.batches[0]))
	}

	mockReader.mu.Lock()
	defer mockReader.mu.Unlock()
	if len(mockReader.committed) != 5 {
		t.Fatalf("expected 5 committed messages, got %d", len(mockReader.committed))
	}
}

func TestSpanConsumer_IntervalTimerFlush(t *testing.T) {
	// Only 2 spans, with batch size 100 and short flush interval of 50ms
	var testMsgs []kafka.Message
	for i := 0; i < 2; i++ {
		span := domain.Span{
			OrgID:       "org-alpha",
			TraceID:     "trace-timer",
			SpanID:      "span-timer",
			ServiceName: "cart-svc",
			DurationMs:  25,
			StartTime:   time.Now(),
		}
		data, _ := json.Marshal(span)
		testMsgs = append(testMsgs, kafka.Message{
			Topic:     "otel-spans",
			Partition: 0,
			Offset:    int64(i),
			Value:     data,
		})
	}

	mockReader := newMockMessageReader(testMsgs)
	mockWriter := &mockSpanBatchWriter{}

	cfg := ConsumerConfig{
		Topic:         "otel-spans",
		BatchSize:     100,
		FlushInterval: 50 * time.Millisecond,
	}

	consumer := NewSpanConsumerWithReader(mockReader, mockWriter, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumer.Run(ctx)
	}()

	// Wait for ticker flush
	deadline := time.Now().Add(2 * time.Second)
	for {
		mockWriter.mu.Lock()
		bCount := len(mockWriter.batches)
		mockWriter.mu.Unlock()
		if bCount >= 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	<-consumerDone

	mockWriter.mu.Lock()
	defer mockWriter.mu.Unlock()
	if len(mockWriter.batches) != 1 {
		t.Fatalf("expected 1 timer-flushed batch, got %d", len(mockWriter.batches))
	}
	if len(mockWriter.batches[0]) != 2 {
		t.Fatalf("expected batch size 2, got %d", len(mockWriter.batches[0]))
	}
}

func TestSpanConsumer_WriterErrorPreventsOffsetCommit(t *testing.T) {
	span := domain.Span{
		OrgID:       "org-alpha",
		TraceID:     "trace-err",
		SpanID:      "span-err",
		ServiceName: "pay-svc",
		DurationMs:  10,
		StartTime:   time.Now(),
	}
	data, _ := json.Marshal(span)
	testMsgs := []kafka.Message{
		{
			Topic:  "otel-spans",
			Offset: 0,
			Value:  data,
		},
	}

	mockReader := newMockMessageReader(testMsgs)
	mockWriter := &mockSpanBatchWriter{
		failWithErr: errors.New("clickhouse connection refused"),
	}

	cfg := ConsumerConfig{
		Topic:         "otel-spans",
		BatchSize:     1,
		FlushInterval: 50 * time.Millisecond,
	}

	consumer := NewSpanConsumerWithReader(mockReader, mockWriter, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = consumer.Run(ctx)

	mockReader.mu.Lock()
	defer mockReader.mu.Unlock()
	// When ClickHouse fails, offsets must NOT be committed to prevent silent data loss!
	if len(mockReader.committed) != 0 {
		t.Fatalf("expected 0 committed messages on ClickHouse write failure, got %d", len(mockReader.committed))
	}
}

func TestSpanConsumer_MalformedJSONIsCommittedWithoutCrash(t *testing.T) {
	badMsg := kafka.Message{
		Topic:  "otel-spans",
		Offset: 0,
		Value:  []byte("{not-a-valid-json"),
	}

	mockReader := newMockMessageReader([]kafka.Message{badMsg})
	mockWriter := &mockSpanBatchWriter{}

	cfg := ConsumerConfig{
		Topic:         "otel-spans",
		BatchSize:     10,
		FlushInterval: 50 * time.Millisecond,
	}

	consumer := NewSpanConsumerWithReader(mockReader, mockWriter, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = consumer.Run(ctx)

	// Malformed message committed so consumer is not blocked forever
	mockReader.mu.Lock()
	defer mockReader.mu.Unlock()
	if len(mockReader.committed) != 1 {
		t.Fatalf("expected 1 committed message for malformed payload, got %d", len(mockReader.committed))
	}

	// But no spans written to ClickHouse
	mockWriter.mu.Lock()
	defer mockWriter.mu.Unlock()
	if len(mockWriter.batches) != 0 {
		t.Fatalf("expected 0 batches written for malformed payload, got %d", len(mockWriter.batches))
	}
}

func TestSpanConsumer_BoundedPostBatchHook_ExecutesAndDrains(t *testing.T) {
	testSpans := []domain.Span{
		{
			OrgID:       "org-hook",
			TraceID:     "trace-h1",
			SpanID:      "span-h1",
			ServiceName: "auth-service",
			DurationMs:  25,
			StartTime:   time.Now(),
		},
		{
			OrgID:       "org-hook",
			TraceID:     "trace-h2",
			SpanID:      "span-h2",
			ServiceName: "billing-service",
			DurationMs:  85,
			StartTime:   time.Now(),
		},
	}

	var testMsgs []kafka.Message
	for i, s := range testSpans {
		data, _ := json.Marshal(s)
		testMsgs = append(testMsgs, kafka.Message{
			Topic:  "otel-spans",
			Offset: int64(i),
			Value:  data,
		})
	}

	mockReader := newMockMessageReader(testMsgs)
	mockWriter := &mockSpanBatchWriter{}

	cfg := ConsumerConfig{
		Topic:         "otel-spans",
		BatchSize:     2,
		FlushInterval: 50 * time.Millisecond,
	}

	consumer := NewSpanConsumerWithReader(mockReader, mockWriter, cfg)

	var hookReceived []domain.Span
	var hookMu sync.Mutex
	consumer.SetPostBatchHook(func(ctx context.Context, spans []domain.Span) {
		hookMu.Lock()
		defer hookMu.Unlock()
		hookReceived = append(hookReceived, spans...)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	consumerDone := make(chan struct{})
	go func() {
		_ = consumer.Run(ctx)
		close(consumerDone)
	}()

	// Wait for consumer to process batch and exit cleanly
	select {
	case <-consumerDone:
	case <-time.After(1 * time.Second):
		t.Fatal("consumer did not finish within timeout")
	}

	hookMu.Lock()
	defer hookMu.Unlock()
	if len(hookReceived) != 2 {
		t.Fatalf("expected 2 spans passed to post-batch hook, got %d", len(hookReceived))
	}
	if hookReceived[0].TraceID != "trace-h1" || hookReceived[1].TraceID != "trace-h2" {
		t.Errorf("unexpected spans received by hook: %+v", hookReceived)
	}
}

