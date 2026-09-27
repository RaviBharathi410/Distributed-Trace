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

func NewSpanProducer(brokers string, topic string) *SpanProducer {
	brokerList := strings.Split(brokers, ",")
	
	// Create Kafka writer
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // partition by key (trace_id)
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		WriteTimeout: 5 * time.Second,
	}

	return &SpanProducer{writer: w}
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
