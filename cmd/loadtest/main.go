package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	mathrand "math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	clickhousedriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

var (
	services = []string{
		"checkout-gateway", "auth-service", "order-processor",
		"payment-vault", "inventory-service", "fraud-detector",
		"notification-worker", "recommendation-engine", "shipping-calculator",
		"customer-portal",
	}

	endpoints = map[string][]string{
		"checkout-gateway":      {"POST /api/v1/checkout", "GET /api/v1/cart", "PUT /api/v1/cart/items"},
		"auth-service":          {"POST /oauth/token", "GET /api/v1/user/profile", "POST /api/v1/verify-mfa"},
		"order-processor":       {"ProcessOrderCommand", "ValidateOrderLimits", "PublishOrderCreatedEvent"},
		"payment-vault":         {"POST /v2/charges", "POST /v2/refunds", "GET /v2/payment-methods"},
		"inventory-service":     {"ReserveInventoryItem", "ReleaseInventoryLock", "GET /v1/stock-levels"},
		"fraud-detector":        {"EvaluateTransactionRisk", "CheckDeviceFingerprint", "ScoreVelocityRules"},
		"notification-worker":   {"SendOrderConfirmationEmail", "DispatchSMSReceipt", "PushMobileAlert"},
		"recommendation-engine": {"GetSimilarProducts", "ScoreCollaborativeFiltering", "RankPersonalizedFeed"},
		"shipping-calculator":   {"ComputeTransitRate", "ValidatePostalCode", "GenerateCarrierLabel"},
		"customer-portal":       {"GET /account/orders", "GET /account/invoices", "PUT /account/preferences"},
	}

	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
		"curl/8.4.0 (x86_64-pc-linux-gnu)",
		"Go-http-client/1.1",
		"OpenTelemetry-Go-OTLP-Exporter/1.24.0",
	}

	sqlQueries = []string{
		"SELECT id, balance, status FROM accounts WHERE tenant_id = $1 AND user_id = $2 FOR UPDATE",
		"INSERT INTO order_items (order_id, sku, quantity, unit_price_cents) VALUES ($1, $2, $3, $4)",
		"UPDATE inventory_allocations SET reserved_units = reserved_units + $1 WHERE warehouse_id = $2 AND item_id = $3",
		"SELECT r.rule_id, r.weight FROM fraud_rules r WHERE r.active = true AND r.category = $1",
		"DELETE FROM session_tokens WHERE expires_at < NOW() AND org_id = $1",
	}

	errorTypes = []string{
		"pq: could not serialize access due to concurrent update",
		"context deadline exceeded while contacting upstream payment-vault",
		"redis: connection pool exhausted during peak checkout burst",
		"oauth2: token signature verification failed (key_id not found)",
		"inventory_insufficient: requested 4 units, 1 available in warehouse-east-2",
	}
)

func randomHex(bytesLen int) string {
	b := make([]byte, bytesLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func generateRealisticBatch(spanCount int, orgID string) []domain.Span {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	spans := make([]domain.Span, spanCount)
	now := time.Now().UTC()

	for i := 0; i < spanCount; i++ {
		svc := services[r.Intn(len(services))]
		ops := endpoints[svc]
		op := ops[r.Intn(len(ops))]

		normVal := r.NormFloat64()
		durationMs := uint32(math.Max(2.0, math.Exp(3.8+0.7*normVal)))

		statusCode := uint8(1)
		var errMsg string
		if r.Float64() < 0.08 {
			statusCode = 2
			errMsg = errorTypes[r.Intn(len(errorTypes))]
		}

		tags := map[string]string{
			"http.method":        []string{"GET", "POST", "PUT", "DELETE"}[r.Intn(4)],
			"http.status_code":   fmt.Sprintf("%d", []int{200, 201, 204, 400, 401, 403, 404, 429, 500, 502, 503}[r.Intn(11)]),
			"http.user_agent":    userAgents[r.Intn(len(userAgents))],
			"http.url":           fmt.Sprintf("https://api.example.com/%s?ref=%s&ts=%d", svc, randomHex(4), time.Now().Unix()),
			"user.id":            fmt.Sprintf("usr_%s", randomHex(8)),
			"session.id":         fmt.Sprintf("sess_%s", randomHex(12)),
			"client.ip":          fmt.Sprintf("198.51.%d.%d", r.Intn(250)+1, r.Intn(250)+1),
			"k8s.pod.name":       fmt.Sprintf("%s-deployment-%s-%s", svc, randomHex(3), randomHex(2)),
			"k8s.node.name":      fmt.Sprintf("ip-10-0-%d-%d.ec2.internal", r.Intn(64), r.Intn(254)),
			"region":             []string{"us-east-1", "us-west-2", "eu-central-1", "ap-southeast-1"}[r.Intn(4)],
			"availability_zone":  []string{"us-east-1a", "us-east-1b", "us-east-1c"}[r.Intn(3)],
		}

		if r.Float64() < 0.4 {
			tags["db.system"] = "postgresql"
			tags["db.statement"] = sqlQueries[r.Intn(len(sqlQueries))]
			tags["db.name"] = "production_db"
		}

		if errMsg != "" {
			tags["exception.type"] = errMsg
			tags["exception.message"] = fmt.Sprintf("Failed handling operation %s on %s: %s", op, svc, errMsg)
			tags["exception.stacktrace"] = fmt.Sprintf("github.com/example/%s.(*Handler).ServeHTTP(handler.go:%d)\n\tgolang.org/x/net/http2.(*serverConn).run(server.go:%d)",
				svc, r.Intn(250)+10, r.Intn(400)+50)
		}

		spans[i] = domain.Span{
			TraceID:       randomHex(16),
			SpanID:        randomHex(8),
			ParentSpanID:  randomHex(8),
			ServiceName:   svc,
			OperationName: op,
			DurationMs:    durationMs,
			StatusCode:    statusCode,
			ErrorMessage:  errMsg,
			Tags:          tags,
			StartTime:     now.Add(-time.Duration(durationMs) * time.Millisecond),
			OrgID:         orgID,
		}
	}

	return spans
}

func buildOTLPProtobuf(spans []domain.Span) ([]byte, error) {
	resourceSpans := make([]*tracepb.ResourceSpans, 0, len(spans))

	for _, s := range spans {
		attrs := make([]*commonpb.KeyValue, 0, len(s.Tags))
		for k, v := range s.Tags {
			attrs = append(attrs, &commonpb.KeyValue{
				Key:   k,
				Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: v}},
			})
		}

		traceIDBytes, _ := hex.DecodeString(s.TraceID)
		spanIDBytes, _ := hex.DecodeString(s.SpanID)
		parentIDBytes, _ := hex.DecodeString(s.ParentSpanID)

		protoSpan := &tracepb.Span{
			TraceId:           traceIDBytes,
			SpanId:            spanIDBytes,
			ParentSpanId:      parentIDBytes,
			Name:              s.OperationName,
			Kind:              tracepb.Span_SPAN_KIND_SERVER,
			StartTimeUnixNano: uint64(s.StartTime.UnixNano()),
			EndTimeUnixNano:   uint64(s.StartTime.Add(time.Duration(s.DurationMs) * time.Millisecond).UnixNano()),
			Attributes:        attrs,
			Status: &tracepb.Status{
				Code:    tracepb.Status_StatusCode(s.StatusCode),
				Message: s.ErrorMessage,
			},
		}

		rs := &tracepb.ResourceSpans{
			Resource: &resourcepb.Resource{
				Attributes: []*commonpb.KeyValue{
					{
						Key:   "service.name",
						Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: s.ServiceName}},
					},
				},
			},
			ScopeSpans: []*tracepb.ScopeSpans{
				{
					Spans: []*tracepb.Span{protoSpan},
				},
			},
		}
		resourceSpans = append(resourceSpans, rs)
	}

	req := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: resourceSpans,
	}

	return proto.Marshal(req)
}

func detectHardware() string {
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		return fmt.Sprintf("Shared 2-4 vCPU box (GitHub Actions Runner / %s %s) - INDICATIVE ONLY, NOT A SIZING GUIDE", runtime.GOOS, runtime.GOARCH)
	}
	if os.Getenv("AWS_EXECUTION_ENV") != "" {
		return fmt.Sprintf("Cloud VM (AWS EC2 / %s %s) - INDICATIVE ONLY", runtime.GOOS, runtime.GOARCH)
	}

	return fmt.Sprintf("Local Development Machine (Laptop / %s %s, %d CPUs) - INDICATIVE ONLY",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
}

type PartitionStat struct {
	Partition   string `json:"partition"`
	ActiveParts uint64 `json:"active_parts"`
	Rows        uint64 `json:"rows"`
	BytesOnDisk uint64 `json:"bytes_on_disk"`
}

type TablePartStat struct {
	Table               string  `json:"table"`
	Classification      string  `json:"classification"`
	ActiveParts         uint64  `json:"active_parts"`
	TotalRows           uint64  `json:"total_rows"`
	DataCompressedBytes uint64  `json:"data_compressed_bytes"`
	DataUncompBytes     uint64  `json:"data_uncompressed_bytes"`
	MarksBytes          uint64  `json:"marks_bytes"`
	BytesOnDisk         uint64  `json:"bytes_on_disk"`
	CompBytesPerSpan    float64 `json:"compressed_bytes_per_span"`
	TotalDiskPerSpan    float64 `json:"total_bytes_on_disk_per_span"`
	CompressionRatio    float64 `json:"compression_ratio"`
	Note                string  `json:"note"`
}

type SkippingIndexStat struct {
	Table       string `json:"table"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Expr        string `json:"expr"`
	Granularity uint64 `json:"granularity"`
}

type ColumnStorageStat struct {
	Name         string  `json:"name"`
	Compressed   uint64  `json:"compressed_bytes"`
	Uncompressed uint64  `json:"uncompressed_bytes"`
	Ratio        float64 `json:"ratio"`
}

type BenchmarkReportPayload struct {
	Timestamp               string              `json:"timestamp"`
	HardwareEnvironment     string              `json:"hardware_environment"`
	SpansSent               int                 `json:"spans_sent"`
	SpansPersisted          int                 `json:"spans_persisted"`
	IngestionDurationSec    float64             `json:"ingestion_duration_seconds"`
	SustainedSpansPerSec    float64             `json:"sustained_spans_per_second"`
	Partitions              []PartitionStat     `json:"partitions"`
	Tables                  []TablePartStat     `json:"tables"`
	SkippingIndices         []SkippingIndexStat `json:"skipping_indices"`
	TopColumns              []ColumnStorageStat `json:"top_columns"`
	AllInclusiveDiskPerSpan float64             `json:"all_inclusive_disk_bytes_per_span"`
	DiskFootprintGBPerM     float64             `json:"disk_footprint_gb_per_million"`
	EBSMonthlyCostPerM      float64             `json:"aws_ebs_monthly_cost_per_million"`
	S3MonthlyCostPerM       float64             `json:"aws_s3_monthly_cost_per_million"`
	WireCrossAZCostPerM     float64             `json:"network_cross_az_cost_per_million"`
	RealTotalIngestCostPerM float64             `json:"real_total_ingest_cost_per_million"`
	StatusNote              string              `json:"status_note"`
}

func runWireBenchmark(spanCount int, hardwareLabel string) {
	fmt.Println("================================================================================")
	fmt.Println("DistributedTrace Empirical Ingestion & High-Cardinality Compression Benchmark")
	fmt.Printf("Hardware Environment Label : %s\n", hardwareLabel)
	fmt.Printf("Dataset Configuration      : %d spans with varied high-cardinality attributes\n", spanCount)
	fmt.Println("Mode                       : Wire Serialization & Compression Benchmark (In-Memory)")
	fmt.Println("================================================================================")

	orgID := "00000000-0000-0000-0000-000000000001"

	t0 := time.Now()
	spans := generateRealisticBatch(spanCount, orgID)
	genDuration := time.Since(t0)
	fmt.Printf("✔ Generated %d realistic spans across %d distinct services in %v\n",
		len(spans), len(services), genDuration)

	jsonBytes, err := json.Marshal(spans)
	if err != nil {
		fmt.Printf("❌ Failed to marshal spans JSON: %v\n", err)
		return
	}

	var jsonGzipBuf bytes.Buffer
	gzWriter := gzip.NewWriter(&jsonGzipBuf)
	_, _ = gzWriter.Write(jsonBytes)
	_ = gzWriter.Close()

	tProto := time.Now()
	protoBytes, err := buildOTLPProtobuf(spans)
	protoDuration := time.Since(tProto)
	if err != nil {
		fmt.Printf("❌ Failed to marshal spans Protobuf: %v\n", err)
		return
	}

	var protoGzipBuf bytes.Buffer
	protoGzWriter := gzip.NewWriter(&protoGzipBuf)
	_, _ = protoGzWriter.Write(protoBytes)
	_ = protoGzWriter.Close()

	rawJSONLen := len(jsonBytes)
	gzJSONLen := jsonGzipBuf.Len()
	rawProtoLen := len(protoBytes)
	gzProtoLen := protoGzipBuf.Len()

	rawJSONPerSpan := float64(rawJSONLen) / float64(spanCount)
	gzJSONPerSpan := float64(gzJSONLen) / float64(spanCount)
	rawProtoPerSpan := float64(rawProtoLen) / float64(spanCount)
	gzProtoPerSpan := float64(gzProtoLen) / float64(spanCount)

	jsonRatio := float64(rawJSONLen) / float64(gzJSONLen)
	protoRatio := float64(rawProtoLen) / float64(gzProtoLen)

	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Println("Wire Format Empirical Sizing & Compression Results (VARIED ATTRIBUTES):")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("JSON Raw Total               : %d bytes (%.1f bytes/span)\n", rawJSONLen, rawJSONPerSpan)
	fmt.Printf("JSON Gzip Compressed Total   : %d bytes (%.1f bytes/span, %.2fx ratio)\n", gzJSONLen, gzJSONPerSpan, jsonRatio)
	fmt.Printf("OTLP Proto Raw Total         : %d bytes (%.1f bytes/span, encoded in %v)\n", rawProtoLen, rawProtoPerSpan, protoDuration)
	fmt.Printf("OTLP Proto Gzip Total        : %d bytes (%.1f bytes/span, %.2fx ratio)\n", gzProtoLen, gzProtoPerSpan, protoRatio)
	fmt.Println("--------------------------------------------------------------------------------")

	clickhouseEstimatedBytesPerSpan := 160.0
	gbPerMillionSpans := (clickhouseEstimatedBytesPerSpan * 1_000_000.0) / (1024 * 1024 * 1024)
	ebsMonthlyStoragePerMillion := gbPerMillionSpans * 0.08
	s3MonthlyStoragePerMillion := gbPerMillionSpans * 0.023

	networkGBPerMillion := (gzProtoPerSpan * 1_000_000.0) / (1024 * 1024 * 1024)
	networkCrossAZCostPerMillion := networkGBPerMillion * 0.01

	totalInfraCostPerMillion := ebsMonthlyStoragePerMillion + networkCrossAZCostPerMillion

	fmt.Println("\n================================================================================")
	fmt.Println("Preliminary Theoretical SaaS Cost-Per-Million-Spans Projection:")
	fmt.Println("  [STATUS: UNMEASURED PRELIMINARY ESTIMATE - HELD PENDING SYSTEM.PARTS DATA]")
	fmt.Println("================================================================================")
	fmt.Printf("Assumed ClickHouse Column Storage : %.1f bytes/span (%.2f GB / million spans)\n", clickhouseEstimatedBytesPerSpan, gbPerMillionSpans)
	fmt.Printf("Theoretical Monthly Storage (EBS)  : $%.4f per million spans ($0.08/GB-mo)\n", ebsMonthlyStoragePerMillion)
	fmt.Printf("Theoretical Monthly Storage (S3)   : $%.4f per million spans ($0.023/GB-mo)\n", s3MonthlyStoragePerMillion)
	fmt.Printf("Network Wire Ingress (OTLP/gzip)   : %.2f GB / million spans (Cross-AZ: $%.4f)\n", networkGBPerMillion, networkCrossAZCostPerMillion)
	fmt.Printf("Theoretical Unit Ingest Cost       : ~$%.4f / million spans (UNMEASURED ESTIMATE)\n", totalInfraCostPerMillion)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("CRITICAL NOTICE: Final SaaS pricing tiers are strictly held pending live")
	fmt.Println("empirical ClickHouse system.parts measurements under sustained load.")
	fmt.Println("================================================================================")
}

func runLiveBenchmark(serverURL, chURL, chDB, hardwareLabel, outputJSON, outputMD string, totalSpans, batchSize int) {
	fmt.Println("================================================================================")
	fmt.Println("DistributedTrace Empirical Ingestion & Live ClickHouse system.parts Benchmark")
	fmt.Printf("Hardware Environment Label : %s\n", hardwareLabel)
	fmt.Printf("Target Server              : %s\n", serverURL)
	fmt.Printf("Target ClickHouse          : %s (DB: %s)\n", chURL, chDB)
	fmt.Printf("Dataset Configuration      : %d spans (batch size: %d, Protobuf + Gzip)\n", totalSpans, batchSize)
	fmt.Println("================================================================================")

	chConn, err := clickhousedriver.Open(&clickhousedriver.Options{
		Addr: []string{chURL},
		Auth: clickhousedriver.Auth{
			Database: chDB,
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to ClickHouse at %s: %v", chURL, err)
	}
	defer chConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := chConn.Ping(ctx); err != nil {
		log.Fatalf("❌ ClickHouse ping failed at %s: %v", chURL, err)
	}
	fmt.Println("✔ Connected to live ClickHouse cluster.")

	timestamp := time.Now().UnixNano()
	regPayload, _ := json.Marshal(map[string]string{
		"email":             fmt.Sprintf("bench_%d@example.com", timestamp),
		"password":          "BenchmarkPass123!@#",
		"organization_name": fmt.Sprintf("Benchmark Org %d", timestamp),
	})
	resp, err := http.Post(serverURL+"/api/v1/auth/register", "application/json", bytes.NewReader(regPayload))
	if err != nil {
		log.Fatalf("❌ Failed to register benchmark tenant at %s: %v", serverURL, err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	var regResp struct {
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
		APIKey string `json:"api_key"`
		Token  string `json:"token"`
	}
	_ = json.Unmarshal(bodyBytes, &regResp)
	orgID := regResp.Organization.ID
	apiKey := regResp.APIKey

	if apiKey == "" {
		req, _ := http.NewRequest(http.MethodPost, serverURL+"/api/v1/auth/api-keys", bytes.NewReader([]byte(`{"name":"bench-key","role":"admin"}`)))
		req.Header.Set("Authorization", "Bearer "+regResp.Token)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		keyResp, err := client.Do(req)
		if err == nil {
			defer keyResp.Body.Close()
			var kResp struct {
				Key string `json:"key"`
			}
			_ = json.NewDecoder(keyResp.Body).Decode(&kResp)
			apiKey = kResp.Key
		}
	}

	keyPrefix := apiKey
	if len(keyPrefix) > 12 {
		keyPrefix = keyPrefix[:12] + "..."
	}
	fmt.Printf("✔ Registered benchmark tenant (OrgID: %s, APIKey: %s)\n", orgID, keyPrefix)

	// Clean table before benchmark to ensure exact 1-to-1 system.parts row accounting
	_ = chConn.Exec(ctx, "TRUNCATE TABLE otel_spans")

	var initialCount uint64
	_ = chConn.QueryRow(ctx, "SELECT count() FROM otel_spans WHERE org_id = ?", orgID).Scan(&initialCount)

	httpClient := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	fmt.Printf("\n[Ingestion] Dispatching %d spans in batches of %d via OTLP Protobuf + Gzip...\n", totalSpans, batchSize)
	tIngestStart := time.Now()
	var totalWireBytes int64
	var totalIngestedSpans int

	remaining := totalSpans
	batchNum := 0
	for remaining > 0 {
		curBatch := batchSize
		if curBatch > remaining {
			curBatch = remaining
		}
		batchNum++

		spans := generateRealisticBatch(curBatch, orgID)
		protoBytes, err := buildOTLPProtobuf(spans)
		if err != nil {
			log.Fatalf("❌ Failed to marshal protobuf: %v", err)
		}

		var gzBuf bytes.Buffer
		gzWriter := gzip.NewWriter(&gzBuf)
		_, _ = gzWriter.Write(protoBytes)
		_ = gzWriter.Close()

		req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/traces", &gzBuf)
		if err != nil {
			log.Fatalf("❌ Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/x-protobuf")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("X-API-Key", apiKey)

		bResp, err := httpClient.Do(req)
		if err != nil {
			log.Fatalf("❌ Ingest request failed on batch %d: %v", batchNum, err)
		}
		bResp.Body.Close()
		if bResp.StatusCode != http.StatusOK && bResp.StatusCode != http.StatusAccepted {
			log.Fatalf("❌ Server rejected batch %d with status %d", batchNum, bResp.StatusCode)
		}

		totalWireBytes += int64(gzBuf.Len())
		totalIngestedSpans += curBatch
		remaining -= curBatch
		if batchNum%10 == 0 || remaining == 0 {
			elapsed := time.Since(tIngestStart)
			rate := float64(totalIngestedSpans) / elapsed.Seconds()
			fmt.Printf("  • Batch %3d: Ingested %d/%d spans (%.0f spans/sec, %.1f KB wire)\n",
				batchNum, totalIngestedSpans, totalSpans, rate, float64(totalWireBytes)/1024.0)
		}
	}
	ingestDuration := time.Since(tIngestStart)
	sustainedIngestRate := float64(totalIngestedSpans) / ingestDuration.Seconds()
	fmt.Printf("✔ Completed sending %d spans in %v (sustained rate: %.0f spans/sec)\n",
		totalIngestedSpans, ingestDuration, sustainedIngestRate)

	fmt.Println("\n[ClickHouse Pipeline Drain] Waiting for background Kafka batch consumer to flush rows...")
	expectedTotal := initialCount + uint64(totalIngestedSpans)
	tDrainStart := time.Now()
	var currentCount uint64
	flushed := false
	for i := 0; i < 120; i++ {
		_ = chConn.QueryRow(ctx, "SELECT count() FROM otel_spans WHERE org_id = ?", orgID).Scan(&currentCount)
		if currentCount >= expectedTotal {
			flushed = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !flushed {
		fmt.Printf("⚠️ Warning: Polling timed out. Persisted %d of %d expected spans.\n", currentCount-initialCount, totalIngestedSpans)
	} else {
		fmt.Printf("✔ All %d spans successfully flushed to ClickHouse in %v!\n", totalIngestedSpans, time.Since(tDrainStart))
	}

	fmt.Println("\n[ClickHouse Compaction] Executing OPTIMIZE TABLE otel_spans FINAL to merge parts...")
	tOptimize := time.Now()
	if err := chConn.Exec(ctx, "OPTIMIZE TABLE otel_spans FINAL"); err != nil {
		fmt.Printf("⚠️ OPTIMIZE TABLE notice: %v\n", err)
	} else {
		fmt.Printf("✔ Part merges finalized in %v.\n", time.Since(tOptimize))
	}

	// 1. Partition Consolidation Accounting (system.parts per partition for otel_spans)
	var partitionStats []PartitionStat
	partRows, err := chConn.Query(ctx, fmt.Sprintf(`
		SELECT
			partition,
			count() AS active_parts,
			sum(rows) AS rows,
			sum(bytes_on_disk) AS bytes_on_disk
		FROM system.parts
		WHERE database = '%s' AND table = 'otel_spans' AND active = 1
		GROUP BY partition
		ORDER BY partition
	`, chDB))
	if err == nil {
		defer partRows.Close()
		for partRows.Next() {
			var ps PartitionStat
			if err := partRows.Scan(&ps.Partition, &ps.ActiveParts, &ps.Rows, &ps.BytesOnDisk); err == nil {
				partitionStats = append(partitionStats, ps)
			}
		}
	}

	allPartitionsSinglePart := true
	var partsTotalRows uint64
	for _, ps := range partitionStats {
		partsTotalRows += ps.Rows
		if ps.ActiveParts != 1 {
			allPartitionsSinglePart = false
			fmt.Printf("⚠️ Notice: Partition %s has %d active parts (OPTIMIZE FINAL merges per-partition)\n", ps.Partition, ps.ActiveParts)
		}
	}
	if allPartitionsSinglePart && len(partitionStats) > 0 {
		fmt.Printf("✔ Partition verification: Exactly 1 active part per partition across %d partition(s).\n", len(partitionStats))
	}
	if partsTotalRows == uint64(totalIngestedSpans) {
		fmt.Printf("✔ Row count exact match: system.parts has %d rows (matches %d sent spans).\n", partsTotalRows, totalIngestedSpans)
	} else {
		fmt.Printf("⚠️ Row count check: system.parts has %d rows (total spans sent: %d).\n", partsTotalRows, totalIngestedSpans)
	}

	// 2. Comprehensive Multi-Table Accounting with Classification & Notes
	type tableMeta struct {
		name           string
		classification string
		note           string
	}
	tablesToMeasure := []tableMeta{
		{
			name:           "otel_spans",
			classification: "Primary Ingest (Raw Spans + Indexes)",
			note:           "Receives 100% of OTLP spans; accounts for primary physical storage footprint.",
		},
		{
			name:           "service_metrics_hourly",
			classification: "Auxiliary Aggregate Rollup (SummingMergeTree)",
			note:           "Analytical aggregate table; not written per span during ingestion.",
		},
		{
			name:           "anomalies",
			classification: "Auxiliary Incident Alerts (MergeTree)",
			note:           "Written out-of-band by anomaly detector engine on threshold violation.",
		},
	}

	var tableStats []TablePartStat
	var grandTotalBytesOnDisk uint64

	for _, tm := range tablesToMeasure {
		var activeParts, totalRows, dataComp, dataUncomp, marksBytes, bytesOnDisk uint64
		q := fmt.Sprintf(`
			SELECT
				count(),
				sum(rows),
				sum(data_compressed_bytes),
				sum(data_uncompressed_bytes),
				sum(marks_bytes),
				sum(bytes_on_disk)
			FROM system.parts
			WHERE database = '%s' AND table = '%s' AND active = 1
		`, chDB, tm.name)

		row := chConn.QueryRow(ctx, q)
		if err := row.Scan(&activeParts, &totalRows, &dataComp, &dataUncomp, &marksBytes, &bytesOnDisk); err == nil {
			stat := TablePartStat{
				Table:               tm.name,
				Classification:      tm.classification,
				ActiveParts:         activeParts,
				TotalRows:           totalRows,
				DataCompressedBytes: dataComp,
				DataUncompBytes:     dataUncomp,
				MarksBytes:          marksBytes,
				BytesOnDisk:         bytesOnDisk,
				Note:                tm.note,
			}
			if totalRows > 0 {
				stat.CompBytesPerSpan = float64(dataComp) / float64(totalRows)
				stat.TotalDiskPerSpan = float64(bytesOnDisk) / float64(totalRows)
				if dataComp > 0 {
					stat.CompressionRatio = float64(dataUncomp) / float64(dataComp)
				}
			}
			tableStats = append(tableStats, stat)
			grandTotalBytesOnDisk += bytesOnDisk
		}
	}

	// 3. Skipping Index Accounting (system.data_skipping_indices)
	var skippingIndices []SkippingIndexStat
	idxRows, err := chConn.Query(ctx, fmt.Sprintf(`
		SELECT name, type, expr, granularity
		FROM system.data_skipping_indices
		WHERE database = '%s' AND table = 'otel_spans'
	`, chDB))
	if err == nil {
		defer idxRows.Close()
		for idxRows.Next() {
			var sis SkippingIndexStat
			sis.Table = "otel_spans"
			if err := idxRows.Scan(&sis.Name, &sis.Type, &sis.Expr, &sis.Granularity); err == nil {
				skippingIndices = append(skippingIndices, sis)
			}
		}
	}

	// 4. Top Columns Storage Footprint
	colRows, err := chConn.Query(ctx, fmt.Sprintf(`
		SELECT
			column,
			sum(data_compressed_bytes) AS col_comp,
			sum(data_uncompressed_bytes) AS col_uncomp
		FROM system.parts_columns
		WHERE database = '%s' AND table = 'otel_spans' AND active = 1
		GROUP BY column
		ORDER BY col_comp DESC
	`, chDB))
	var colStats []ColumnStorageStat
	if err == nil {
		defer colRows.Close()
		for colRows.Next() {
			var cs ColumnStorageStat
			if err := colRows.Scan(&cs.Name, &cs.Compressed, &cs.Uncompressed); err == nil {
				if cs.Compressed > 0 {
					cs.Ratio = float64(cs.Uncompressed) / float64(cs.Compressed)
				}
				colStats = append(colStats, cs)
			}
		}
	}

	allInclusiveDiskPerSpan := 0.0
	if totalIngestedSpans > 0 {
		allInclusiveDiskPerSpan = float64(grandTotalBytesOnDisk) / float64(totalIngestedSpans)
	}

	realGBPerMillion := (allInclusiveDiskPerSpan * 1_000_000.0) / (1024 * 1024 * 1024)
	realEBSMonthlyCostPerMillion := realGBPerMillion * 0.08
	realS3MonthlyCostPerMillion := realGBPerMillion * 0.023
	wireGBPerMillion := (float64(totalWireBytes) / float64(totalIngestedSpans) * 1_000_000.0) / (1024 * 1024 * 1024)
	wireCrossAZCostPerMillion := wireGBPerMillion * 0.01
	realDirectUnitCostPerMillion := realEBSMonthlyCostPerMillion + wireCrossAZCostPerMillion

	fmt.Println("\n================================================================================")
	fmt.Println("EMPIRICAL CLICKHOUSE SYSTEM.PARTS MULTI-TABLE ACCOUNTING & DISK MEASUREMENTS:")
	fmt.Println("================================================================================")
	fmt.Printf("Hardware Environment Label : %s\n", hardwareLabel)
	fmt.Printf("Sustained Ingestion Rate   : %.0f spans / second (duration: %v)\n", sustainedIngestRate, ingestDuration)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Partition Consolidation Summary (otel_spans):")
	for _, ps := range partitionStats {
		fmt.Printf("  • Partition %-12s: %d active parts, %d rows, %d bytes on disk\n",
			ps.Partition, ps.ActiveParts, ps.Rows, ps.BytesOnDisk)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Active Table Footprint Summary:")
	for _, ts := range tableStats {
		fmt.Printf("  • Table %-24s: %d parts, %d rows, %d compressed bytes, %d bytes on disk [%s]\n",
			ts.Table, ts.ActiveParts, ts.TotalRows, ts.DataCompressedBytes, ts.BytesOnDisk, ts.Classification)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Data Skipping Indices (Bloom Filter):")
	for _, si := range skippingIndices {
		fmt.Printf("  • Index %-18s: type=%s, expr=%s, granularity=%d\n", si.Name, si.Type, si.Expr, si.Granularity)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Top Column Compressed Footprint (otel_spans):")
	for i, col := range colStats {
		if i >= 8 {
			break
		}
		fmt.Printf("  • %-18s : %8d bytes compressed (uncomp: %8d, %.1fx ratio)\n",
			col.Name, col.Compressed, col.Uncompressed, col.Ratio)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Empirically Measured SaaS Economics (All-Inclusive Disk Footprint):")
	fmt.Printf("  • All-Inclusive Disk / Span        : %.2f bytes / span (columns + skipping indices + marks)\n", allInclusiveDiskPerSpan)
	fmt.Printf("  • Real Disk Footprint / 1M Spans   : %.3f GB / million spans\n", realGBPerMillion)
	fmt.Printf("  • Monthly Storage (AWS EBS gp3)    : $%.4f / million spans ($0.08/GB-mo)\n", realEBSMonthlyCostPerMillion)
	fmt.Printf("  • Monthly Storage (AWS S3 Tier)    : $%.4f / million spans ($0.023/GB-mo)\n", realS3MonthlyCostPerMillion)
	fmt.Printf("  • Wire Ingress Transfer (Cross-AZ) : $%.4f / million spans (%.3f GB)\n", wireCrossAZCostPerMillion, wireGBPerMillion)
	fmt.Printf("  • Measured Direct Unit Ingest Cost : ~$%.4f / million spans (EBS + Wire)\n", realDirectUnitCostPerMillion)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("CRITICAL NOTICE: Results are indicative of GitHub runner hardware (shared 2-4 vCPU).")
	fmt.Println("Final SaaS pricing tiers are strictly held pending multi-million cloud VM production sizing.")
	fmt.Println("================================================================================")

	payload := BenchmarkReportPayload{
		Timestamp:               time.Now().UTC().Format(time.RFC3339),
		HardwareEnvironment:     hardwareLabel,
		SpansSent:               totalSpans,
		SpansPersisted:          int(currentCount - initialCount),
		IngestionDurationSec:    ingestDuration.Seconds(),
		SustainedSpansPerSec:    sustainedIngestRate,
		Partitions:              partitionStats,
		Tables:                  tableStats,
		SkippingIndices:         skippingIndices,
		TopColumns:              colStats,
		AllInclusiveDiskPerSpan: allInclusiveDiskPerSpan,
		DiskFootprintGBPerM:     realGBPerMillion,
		EBSMonthlyCostPerM:      realEBSMonthlyCostPerMillion,
		S3MonthlyCostPerM:       realS3MonthlyCostPerMillion,
		WireCrossAZCostPerM:     wireCrossAZCostPerMillion,
		RealTotalIngestCostPerM: realDirectUnitCostPerMillion,
		StatusNote:              "Indicative of runner hardware; SaaS pricing held pending cloud VM sizing.",
	}

	if outputJSON != "" {
		jBytes, err := json.MarshalIndent(payload, "", "  ")
		if err == nil {
			_ = os.WriteFile(outputJSON, jBytes, 0644)
			fmt.Printf("✔ Saved benchmark JSON artifact: %s\n", outputJSON)
		}
	}

	if outputMD != "" {
		var md strings.Builder
		md.WriteString("# DistributedTrace Empirical Ingestion & Storage Benchmark Report\n\n")
		md.WriteString(fmt.Sprintf("**Date:** %s  \n", payload.Timestamp))
		md.WriteString(fmt.Sprintf("**Hardware Environment:** `%s`  \n", payload.HardwareEnvironment))
		md.WriteString(fmt.Sprintf("**Dataset:** %d spans sent (%d persisted) in %d-span batches  \n", payload.SpansSent, payload.SpansPersisted, batchSize))
		md.WriteString(fmt.Sprintf("**Sustained Throughput:** `%.0f spans/second` (duration: `%.2fs`)  \n\n", payload.SustainedSpansPerSec, payload.IngestionDurationSec))

		md.WriteString("## 1. ClickHouse Partition Consolidation (system.parts)\n\n")
		md.WriteString("| Partition | Active Parts (Post-OPTIMIZE) | Rows | Bytes on Disk | Single Part Verified |\n")
		md.WriteString("|---|---|---|---|---|\n")
		for _, ps := range payload.Partitions {
			verified := "✅ Yes (1 part)"
			if ps.ActiveParts != 1 {
				verified = fmt.Sprintf("⚠️ %d parts", ps.ActiveParts)
			}
			md.WriteString(fmt.Sprintf("| `%s` | %d | %d | **%d B** | %s |\n",
				ps.Partition, ps.ActiveParts, ps.Rows, ps.BytesOnDisk, verified))
		}

		md.WriteString("\n## 2. Multi-Table Storage Footprint & Classification\n\n")
		md.WriteString("| Table | Role / Classification | Active Parts | Rows | Compressed Data | Bytes on Disk |\n")
		md.WriteString("|---|---|---|---|---|---|\n")
		for _, ts := range payload.Tables {
			md.WriteString(fmt.Sprintf("| `%s` | %s | %d | %d | %d B | **%d B** |\n",
				ts.Table, ts.Classification, ts.ActiveParts, ts.TotalRows, ts.DataCompressedBytes, ts.BytesOnDisk))
		}
		md.WriteString("\n> [!NOTE]\n")
		md.WriteString("> Telemetry ingest storage is accounted for by `otel_spans` (columns + skipping indices + marks). `service_metrics_hourly` and `anomalies` are auxiliary analytical tables populated out-of-band or via separate rollup workers, not by raw span ingestion.\n\n")

		md.WriteString("## 3. Data Skipping Indices (Bloom Filter)\n\n")
		md.WriteString("| Table | Index Name | Type | Expression | Granularity |\n")
		md.WriteString("|---|---|---|---|---|\n")
		for _, si := range payload.SkippingIndices {
			md.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | %d |\n",
				si.Table, si.Name, si.Type, si.Expr, si.Granularity))
		}

		md.WriteString("\n## 4. Top Column Compressed Footprint (otel_spans)\n\n")
		md.WriteString("| Column | Compressed Bytes | Uncompressed Bytes | Compression Ratio |\n")
		md.WriteString("|---|---|---|---|\n")
		for i, c := range payload.TopColumns {
			if i >= 10 {
				break
			}
			md.WriteString(fmt.Sprintf("| `%s` | %d B | %d B | %.1fx |\n", c.Name, c.Compressed, c.Uncompressed, c.Ratio))
		}

		md.WriteString("\n## 5. Empirical SaaS Cost-Per-Million-Spans Economics\n\n")
		md.WriteString(fmt.Sprintf("- **All-Inclusive Disk Footprint per Span:** `%.2f bytes` (columns + skipping indices + marks)\n", payload.AllInclusiveDiskPerSpan))
		md.WriteString(fmt.Sprintf("- **Physical Disk Footprint per 1M Spans:** `%.3f GB`\n", payload.DiskFootprintGBPerM))
		md.WriteString(fmt.Sprintf("- **Monthly Storage Cost (AWS EBS gp3):** `$%.4f / million spans` ($0.08/GB-mo)\n", payload.EBSMonthlyCostPerM))
		md.WriteString(fmt.Sprintf("- **Monthly Storage Cost (AWS S3 Tiered):** `$%.4f / million spans` ($0.023/GB-mo)\n", payload.S3MonthlyCostPerM))
		md.WriteString(fmt.Sprintf("- **Cross-AZ Network Wire Ingress:** `$%.4f / million spans`\n", payload.WireCrossAZCostPerM))
		md.WriteString(fmt.Sprintf("- **Direct Measured Unit Cost:** `~$%.4f / million spans` (Direct Storage + Wire)\n\n", payload.RealTotalIngestCostPerM))

		md.WriteString("> [!IMPORTANT]\n")
		md.WriteString("> **Hardware Disclaimer & Pricing Guardrail:** Results are indicative of GitHub runner hardware (shared 2-4 vCPU box). Final SaaS pricing tiers are strictly held pending dedicated cloud VM production sizing.\n")

		_ = os.WriteFile(outputMD, []byte(md.String()), 0644)
		fmt.Printf("✔ Saved benchmark Markdown artifact: %s\n", outputMD)
	}
}

func main() {
	mode := flag.String("mode", "wire", "Benchmark mode: 'wire' (in-memory serialization) or 'live' (full-stack system.parts measurement)")
	spanCount := flag.Int("spans", 5000, "Number of spans to generate for load test")
	batchSize := flag.Int("batch-size", 8192, "Spans per HTTP request batch (for live mode)")
	serverURL := flag.String("server", "http://127.0.0.1:8080", "Server base URL (for live mode)")
	clickhouseURL := flag.String("clickhouse", "127.0.0.1:9000", "ClickHouse native connection address (for live mode)")
	clickhouseDB := flag.String("clickhouse-db", "default", "ClickHouse database name")
	outputJSON := flag.String("output-json", "", "Path to write JSON benchmark artifact")
	outputMD := flag.String("output-md", "", "Path to write Markdown benchmark report")
	customHardware := flag.String("hardware", "", "Override hardware label (e.g. 'Laptop' or 'Cloud VM')")
	flag.Parse()

	hardwareLabel := *customHardware
	if hardwareLabel == "" {
		hardwareLabel = detectHardware()
	}

	if *mode == "live" {
		runLiveBenchmark(*serverURL, *clickhouseURL, *clickhouseDB, hardwareLabel, *outputJSON, *outputMD, *spanCount, *batchSize)
	} else {
		runWireBenchmark(*spanCount, hardwareLabel)
	}
}
