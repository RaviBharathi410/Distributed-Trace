package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

type SpanProducer struct {
	writer *kafka.Writer
}

type ProducerConfig struct {
	Brokers                string
	Topic                  string
	AllowAutoTopicCreation bool
}

func NewSpanProducerWithConfig(cfg ProducerConfig) *SpanProducer {
	brokerList := strings.Split(cfg.Brokers, ",")

	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokerList...),
		Topic:                  cfg.Topic,
		Balancer:               &kafka.Hash{}, // partition by key (trace_id)
		RequiredAcks:           kafka.RequireAll,
		Async:                  false,
		WriteTimeout:           5 * time.Second,
		AllowAutoTopicCreation: cfg.AllowAutoTopicCreation,
	}

	return &SpanProducer{writer: w}
}

// NewSpanProducer maintains backwards compatibility with default AllowAutoTopicCreation: false
// to prevent accidental or malicious dynamic topic generation in production.
func NewSpanProducer(brokers string, topic string) *SpanProducer {
	return NewSpanProducerWithConfig(ProducerConfig{
		Brokers:                brokers,
		Topic:                  topic,
		AllowAutoTopicCreation: false,
	})
}

func (p *SpanProducer) PublishSpans(ctx context.Context, spans []domain.Span) error {
	timer := prometheus.NewTimer(observability.KafkaProducerLatency)
	defer timer.ObserveDuration()

	messages := make([]kafka.Message, len(spans))
	for i, s := range spans {
		data, err := json.Marshal(s)
		if err != nil {
			observability.KafkaMessagesProduced.WithLabelValues(p.writer.Topic, "error").Inc()
			return fmt.Errorf("failed to marshal span: %w", err)
		}

		messages[i] = kafka.Message{
			Key:   []byte(s.TraceID),
			Value: data,
		}
	}

	err := p.writer.WriteMessages(ctx, messages...)
	if err != nil {
		observability.KafkaMessagesProduced.WithLabelValues(p.writer.Topic, "error").Inc()
		return fmt.Errorf("failed to write messages to kafka: %w", err)
	}

	observability.KafkaMessagesProduced.WithLabelValues(p.writer.Topic, "ok").Add(float64(len(spans)))
	return nil
}

func (p *SpanProducer) Close() error {
	return p.writer.Close()
}
