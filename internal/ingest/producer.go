package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
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
	timer := prometheusTimer()
	if timer != nil {
		defer timer.ObserveDuration()
	}

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

func prometheusTimer() *observability.TimerObserver {
	// Utility wrapper to resolve prometheus observers safely
	t := observability.KafkaProducerLatency
	if t == nil {
		return nil
	}
	return &observability.TimerObserver{Observer: t}
}

// Helper utility to make prometheus timing simple
type TimerObserver struct {
	Observer prometheus.Observer
	start    time.Time
}

func (to *TimerObserver) ObserveDuration() {
	if to != nil && to.Observer != nil {
		to.Observer.Observe(time.Since(to.start).Seconds())
	}
}
