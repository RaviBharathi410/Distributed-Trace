package ingest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

func isRedpandaReachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

type recordingBatchWriter struct {
	mu            sync.Mutex
	batches       [][]domain.Span
	failWithErr   error
	failCallCount int
	calls         int
}

func (r *recordingBatchWriter) InsertSpansBatch(ctx context.Context, spans []domain.Span) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.failWithErr != nil && (r.failCallCount == 0 || r.calls <= r.failCallCount) {
		return r.failWithErr
	}
	cpy := make([]domain.Span, len(spans))
	copy(cpy, spans)
	r.batches = append(r.batches, cpy)
	return nil
}

func (r *recordingBatchWriter) TotalSpans() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0
	for _, b := range r.batches {
		total += len(b)
	}
	return total
}

func TestSpanConsumer_RealRedpanda_OffsetSafetyAndRestart(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "127.0.0.1:9092"
	}

	if !isRedpandaReachable(brokers) {
		t.Skipf("skipping live broker integration test: Redpanda not reachable at %s. Run 'docker compose up -d' or execute in CI.", brokers)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Generate isolated topic and consumer group for this test run
	topic := fmt.Sprintf("test-otel-spans-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	// 1. Produce 6 real spans to Redpanda via SpanProducer
	producer := NewSpanProducer(brokers, topic)
	defer producer.Close()

	var testSpans []domain.Span
	for i := 0; i < 6; i++ {
		testSpans = append(testSpans, domain.Span{
			OrgID:         "org-redpanda-test",
			TraceID:       fmt.Sprintf("trace-rp-%d", i),
			SpanID:        fmt.Sprintf("span-rp-%d", i),
			ServiceName:   "order-service",
			OperationName: "ProcessOrder",
			DurationMs:    uint32(100 + i*10),
			StatusCode:    1,
			StartTime:     time.Now().UTC(),
		})
	}

	if err := producer.PublishSpans(ctx, testSpans); err != nil {
		t.Fatalf("failed to publish spans to real Redpanda: %v", err)
	}

	// 2. Scenario 1: Consumer 1 encounters downstream database failure on ClickHouse insert.
	// It must NOT commit offsets.
	failingWriter := &recordingBatchWriter{
		failWithErr: errors.New("clickhouse connection refused - simulated downtime"),
	}

	cfg1 := ConsumerConfig{
		Brokers:       brokers,
		Topic:         topic,
		GroupID:       groupID,
		BatchSize:     6,
		FlushInterval: 100 * time.Millisecond,
		MaxWait:       100 * time.Millisecond,
	}

	consumer1 := NewSpanConsumer(cfg1, failingWriter)

	runCtx1, runCancel1 := context.WithTimeout(ctx, 1500*time.Millisecond)
	_ = consumer1.Run(runCtx1)
	runCancel1()

	// Failing writer had flush attempts, but no successful batches persisted
	if failingWriter.TotalSpans() != 0 {
		t.Fatalf("expected 0 spans persisted by failing consumer, got %d", failingWriter.TotalSpans())
	}

	// 3. Scenario 2: Consumer restarts after downtime (Consumer 2 with same GroupID)
	// Because Consumer 1 did not commit offsets, Redpanda re-delivers the in-flight spans.
	workingWriter := &recordingBatchWriter{}

	cfg2 := ConsumerConfig{
		Brokers:       brokers,
		Topic:         topic,
		GroupID:       groupID,
		BatchSize:     6,
		FlushInterval: 500 * time.Millisecond,
		MaxWait:       100 * time.Millisecond,
	}

	consumer2 := NewSpanConsumer(cfg2, workingWriter)

	runCtx2, runCancel2 := context.WithTimeout(ctx, 4*time.Second)
	doneCh2 := make(chan error, 1)
	go func() {
		doneCh2 <- consumer2.Run(runCtx2)
	}()

	// Wait for consumer2 to process all 6 spans and commit offsets
	deadline := time.Now().Add(3 * time.Second)
	for {
		if workingWriter.TotalSpans() >= 6 || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	runCancel2()
	<-doneCh2

	// Verify all 6 spans were cleanly received and persisted without loss
	if workingWriter.TotalSpans() != 6 {
		t.Fatalf("expected exactly 6 spans recovered and persisted after consumer restart, got %d", workingWriter.TotalSpans())
	}

	// 4. Scenario 3: Verify offset commit persistence in Redpanda broker (No duplicate reprocessing)
	// Start Consumer 3 with the same GroupID. Since Consumer 2 committed offsets, 0 spans should be re-processed.
	duplicateCheckWriter := &recordingBatchWriter{}

	cfg3 := ConsumerConfig{
		Brokers:       brokers,
		Topic:         topic,
		GroupID:       groupID,
		BatchSize:     1,
		FlushInterval: 100 * time.Millisecond,
		MaxWait:       100 * time.Millisecond,
	}

	consumer3 := NewSpanConsumer(cfg3, duplicateCheckWriter)

	runCtx3, runCancel3 := context.WithTimeout(ctx, 1500*time.Millisecond)
	_ = consumer3.Run(runCtx3)
	runCancel3()

	// Assert zero duplicates reprocessed
	if duplicateCheckWriter.TotalSpans() != 0 {
		t.Fatalf("detected %d duplicate spans reprocessed! Offsets were not committed properly to Redpanda", duplicateCheckWriter.TotalSpans())
	}
}
