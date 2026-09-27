package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type SpanBatchWriter interface {
	InsertSpansBatch(ctx context.Context, spans []domain.Span) error
}

type MessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type ConsumerConfig struct {
	Brokers       string
	Topic         string
	GroupID       string
	BatchSize     int
	FlushInterval time.Duration
	MaxWait       time.Duration
}

type SpanConsumer struct {
	reader MessageReader
	writer SpanBatchWriter
	cfg    ConsumerConfig
	stopCh chan struct{}
	doneCh chan struct{}
	mu     sync.Mutex
}

func NewSpanConsumer(cfg ConsumerConfig, writer SpanBatchWriter) *SpanConsumer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 1 * time.Second
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 500 * time.Millisecond
	}
	if cfg.GroupID == "" {
		cfg.GroupID = "distributedtrace-ingest-consumer"
	}
	brokerList := strings.Split(cfg.Brokers, ",")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokerList,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		MaxWait:        cfg.MaxWait,
		CommitInterval: 0,
	})

	return NewSpanConsumerWithReader(reader, writer, cfg)
}

func NewSpanConsumerWithReader(reader MessageReader, writer SpanBatchWriter, cfg ConsumerConfig) *SpanConsumer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 1 * time.Second
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = 500 * time.Millisecond
	}
	if cfg.GroupID == "" {
		cfg.GroupID = "distributedtrace-ingest-consumer"
	}

	return &SpanConsumer{
		reader: reader,
		writer: writer,
		cfg:    cfg,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (c *SpanConsumer) Done() <-chan struct{} {
	return c.doneCh
}

func (c *SpanConsumer) Stop() {
	c.mu.Lock()
	select {
	case <-c.stopCh:
		c.mu.Unlock()
	default:
		close(c.stopCh)
		c.mu.Unlock()
	}
	<-c.doneCh
}

func (c *SpanConsumer) Run(ctx context.Context) error {
	defer close(c.doneCh)

	msgCh := make(chan kafka.Message, c.cfg.BatchSize)
	fetchCtx, cancelFetch := context.WithCancel(ctx)
	defer cancelFetch()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg, err := c.reader.FetchMessage(fetchCtx)
			if err != nil {
				if fetchCtx.Err() != nil {
					return
				}
				// Transient network error or rebalancing, backoff briefly
				select {
				case <-fetchCtx.Done():
					return
				case <-time.After(100 * time.Millisecond):
				}
				continue
			}
			select {
			case msgCh <- msg:
			case <-fetchCtx.Done():
				return
			}
		}
	}()

	ticker := time.NewTicker(c.cfg.FlushInterval)
	defer ticker.Stop()

	var spansBuffer []domain.Span
	var msgsBuffer []kafka.Message

	var consecutiveErrors int
	const (
		initialBackoff = 200 * time.Millisecond
		maxBackoff     = 10 * time.Second
	)

	applyBackoff := func() {
		consecutiveErrors++
		shift := consecutiveErrors - 1
		if shift > 6 {
			shift = 6
		}
		backoff := initialBackoff * time.Duration(1<<uint(shift))
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		observability.Log.Warn("Applying exponential backoff after downstream persistence failure",
			zap.Int("consecutive_errors", consecutiveErrors),
			zap.Duration("backoff", backoff),
		)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
		case <-c.stopCh:
		}
	}

	flush := func() error {
		if len(spansBuffer) == 0 {
			return nil
		}
		timer := prometheus.NewTimer(observability.KafkaConsumerBatchDuration)
		defer timer.ObserveDuration()

		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := c.writer.InsertSpansBatch(flushCtx, spansBuffer); err != nil {
			observability.KafkaMessagesConsumed.WithLabelValues(c.cfg.Topic, "write_error").Add(float64(len(spansBuffer)))
			return fmt.Errorf("failed to persist span batch to ClickHouse: %w", err)
		}

		if err := c.reader.CommitMessages(flushCtx, msgsBuffer...); err != nil {
			observability.KafkaMessagesConsumed.WithLabelValues(c.cfg.Topic, "commit_error").Add(float64(len(spansBuffer)))
			return fmt.Errorf("failed to commit messages to Kafka: %w", err)
		}

		consecutiveErrors = 0
		observability.KafkaMessagesConsumed.WithLabelValues(c.cfg.Topic, "ok").Add(float64(len(spansBuffer)))
		spansBuffer = spansBuffer[:0]
		msgsBuffer = msgsBuffer[:0]
		return nil
	}

	defer func() {
		cancelFetch()
		wg.Wait()
		// Drain and flush remaining buffered messages on shutdown
		for {
			select {
			case msg := <-msgCh:
				var span domain.Span
				if err := json.Unmarshal(msg.Value, &span); err == nil {
					spansBuffer = append(spansBuffer, span)
					msgsBuffer = append(msgsBuffer, msg)
				}
			default:
				goto drained
			}
		}
	drained:
		if len(spansBuffer) > 0 {
			if err := flush(); err != nil {
				observability.Log.Error("Failed to flush remaining spans on shutdown", zap.Error(err))
			}
		}
		_ = c.reader.Close()
	}()

	for {
		// Strict Upstream Backpressure:
		// If spansBuffer has reached or exceeded batch capacity, do NOT pull any more messages from msgCh.
		// Attempt flush; if flush fails due to downstream outage, applyBackoff() and loop without ingesting more.
		// Because msgCh (capacity batchSize) remains full, the fetch goroutine blocks on msgCh <- msg,
		// pausing reader.FetchMessage and pushing backpressure directly to the Kafka partition.
		if len(spansBuffer) >= c.cfg.BatchSize {
			if err := flush(); err != nil {
				observability.Log.Error("Consumer batch flush failed under backpressure", zap.Error(err))
				applyBackoff()
				select {
				case <-ctx.Done():
					return nil
				case <-c.stopCh:
					return nil
				default:
				}
			}
			continue
		}

		select {
		case <-ctx.Done():
			return nil

		case <-c.stopCh:
			return nil

		case msg := <-msgCh:
			var span domain.Span
			if err := json.Unmarshal(msg.Value, &span); err != nil {
				observability.Log.Warn("Failed to unmarshal span JSON from Kafka", zap.Error(err))
				observability.KafkaMessagesConsumed.WithLabelValues(c.cfg.Topic, "malformed").Inc()
				// Commit malformed message so queue does not stall
				commitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = c.reader.CommitMessages(commitCtx, msg)
				cancel()
				continue
			}

			spansBuffer = append(spansBuffer, span)
			msgsBuffer = append(msgsBuffer, msg)

			if len(spansBuffer) >= c.cfg.BatchSize {
				if err := flush(); err != nil {
					observability.Log.Error("Consumer batch flush failed", zap.Error(err))
					applyBackoff()
				}
			}

		case <-ticker.C:
			if len(spansBuffer) > 0 {
				if err := flush(); err != nil {
					observability.Log.Error("Consumer interval flush failed", zap.Error(err))
					applyBackoff()
				}
			}
		}
	}
}
