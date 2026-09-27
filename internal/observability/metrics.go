package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type QueryDurationTracker struct {
	vec *prometheus.HistogramVec
}

func (t *QueryDurationTracker) WithLabelValues(lvs ...string) *prometheus.Timer {
	return prometheus.NewTimer(t.vec.WithLabelValues(lvs...))
}

var (
	// HTTP Layer
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests handled",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Latency of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	HttpRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of concurrent HTTP requests currently in flight",
	})

	// Database Layer
	dbQueryDurationVec = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Database query latencies in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"database", "query_name"})

	DbQueryDuration = &QueryDurationTracker{vec: dbQueryDurationVec}

	DbConnectionsOpen = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_connections_open",
		Help: "Active connections to database instances",
	}, []string{"database"})

	DbErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_errors_total",
		Help: "Total errors thrown by database engine operations",
	}, []string{"database", "error_type"})

	// Kafka Producer
	KafkaMessagesProduced = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kafka_messages_produced_total",
		Help: "Total spans written to Kafka bus",
	}, []string{"topic", "status"})

	KafkaProducerLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "kafka_producer_latency_seconds",
		Help:    "Producer write operations duration",
		Buckets: prometheus.DefBuckets,
	})

	// WebSockets
	WsConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_connections_active",
		Help: "Total connected streaming websocket sessions",
	})

	WsMessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_messages_sent_total",
		Help: "Total payload messages streamed to websockets",
	}, []string{"type"})

	WsConnectionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ws_connection_errors_total",
		Help: "Total errors seen in active streaming pipelines",
	})

	// Business metrics
	SpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "spans_ingested_total",
		Help: "Spans batch counts handled by public ingest routing",
	}, []string{"org_id", "status"})

	AnomaliesDetected = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "anomalies_detected_total",
		Help: "Identified service anomalies partitioned by severity scale",
	}, []string{"severity"})

	TailSamplerBufferDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tail_sampler_buffer_depth",
		Help: "Current buffer sizing measured from Redis active trace pools",
	})
)
