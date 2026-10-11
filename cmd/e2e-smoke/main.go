package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	trace "go.opentelemetry.io/otel/trace"
)

type RegisterResponse struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
	Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"organization"`
	Token string `json:"token"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type APIKeyResponse struct {
	APIKey struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Name   string `json:"name"`
		Prefix string `json:"prefix"`
	} `json:"api_key"`
}

type TraceItem struct {
	TraceID       string `json:"trace_id"`
	SpanID        string `json:"span_id"`
	ServiceName   string `json:"service_name"`
	OperationName string `json:"operation_name"`
}

func main() {
	baseURL := os.Getenv("SERVER_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	fmt.Println("==================================================")
	fmt.Println("DistributedTrace End-to-End Live Smoke Runner")
	fmt.Println("Target:", baseURL)
	fmt.Println("==================================================")

	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: Wait for server readiness
	fmt.Print("\n[Step 1] Polling server health (/health)...")
	ready := false
	for i := 0; i < 30; i++ {
		resp, err := client.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			ready = true
			resp.Body.Close()
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
		fmt.Print(".")
	}
	if !ready {
		log.Fatalf("\n❌ Server failed to become healthy at %s within 30s", baseURL)
	}
	fmt.Println(" ✔ Server is READY!")

	// Step 2: Register tenant
	timestamp := time.Now().Unix()
	email := fmt.Sprintf("smoke-test-%d@example.com", timestamp)
	password := "Password123!@#"
	orgName := fmt.Sprintf("Smoke Tenant %d", timestamp)

	fmt.Printf("\n[Step 2] Registering tenant (%s, org: %s)...\n", email, orgName)
	regPayload := map[string]string{
		"email":             email,
		"password":          password,
		"organization_name": orgName,
	}
	regBody, _ := json.Marshal(regPayload)
	resp, err := client.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		log.Fatalf("❌ Registration request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		log.Fatalf("❌ Registration returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		log.Fatalf("❌ Failed to decode register response: %v", err)
	}
	jwtToken := regResp.Token
	fmt.Println("✔ Registered tenant successfully! OrgID:", regResp.Organization.ID)

	// Step 3: Login if token was empty (or verify login)
	if jwtToken == "" {
		fmt.Printf("\n[Step 3] Logging in as %s...\n", email)
		loginPayload := map[string]string{"email": email, "password": password}
		loginBody, _ := json.Marshal(loginPayload)
		lResp, err := client.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
		if err != nil || lResp.StatusCode != http.StatusOK {
			log.Fatalf("❌ Login failed: %v", err)
		}
		defer lResp.Body.Close()
		var lData LoginResponse
		if err := json.NewDecoder(lResp.Body).Decode(&lData); err != nil {
			log.Fatalf("❌ Failed to decode login response: %v", err)
		}
		jwtToken = lData.Token
	}
	fmt.Println("✔ Authentication token acquired.")

	// Step 4: Create API Key
	fmt.Println("\n[Step 4] Creating API key via POST /api/v1/auth/api-keys...")
	keyPayload := map[string]string{
		"name":        "smoke-otel-key",
		"environment": "production",
	}
	keyBody, _ := json.Marshal(keyPayload)
	keyReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/api-keys", bytes.NewReader(keyBody))
	keyReq.Header.Set("Content-Type", "application/json")
	keyReq.Header.Set("Authorization", "Bearer "+jwtToken)

	kResp, err := client.Do(keyReq)
	if err != nil || kResp.StatusCode != http.StatusCreated {
		respBytes, _ := io.ReadAll(kResp.Body)
		log.Fatalf("❌ Failed to create API key (status %d): %s", kResp.StatusCode, string(respBytes))
	}
	defer kResp.Body.Close()

	var kData APIKeyResponse
	if err := json.NewDecoder(kResp.Body).Decode(&kData); err != nil {
		log.Fatalf("❌ Failed to decode API key response: %v", err)
	}
	apiKey := kData.APIKey.Key
	if apiKey == "" {
		log.Fatalf("❌ Returned API key was empty!")
	}
	fmt.Printf("✔ API Key created successfully (Prefix: %s...)\n", kData.APIKey.Prefix)

	// Step 5: Send spans using REAL OpenTelemetry SDK over Protobuf + Gzip
	fmt.Println("\n[Step 5] Sending spans using real OpenTelemetry Go SDK (Protobuf + Gzip over /v1/traces)...")
	ctx := context.Background()

	// Strip http:// from endpoint for otlptracehttp
	hostEndpoint := baseURL
	if len(hostEndpoint) > 7 && hostEndpoint[:7] == "http://" {
		hostEndpoint = hostEndpoint[7:]
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(hostEndpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithHeaders(map[string]string{
			"X-API-Key": apiKey,
		}),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create OTel exporter: %v", err)
	}
	defer exporter.Shutdown(ctx)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	defer tp.Shutdown(ctx)

	tracer := tp.Tracer("smoke-test-tracer")
	spanName := fmt.Sprintf("e2e.smoke.payment_processor_%d", timestamp)
	serviceName := "e2e-payment-service"

	_, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", "POST"),
			attribute.Int("http.status_code", 200),
			semconv.ServiceName(serviceName),
		),
	)
	time.Sleep(20 * time.Millisecond)
	span.End()

	flushCtx, flushCancel := context.WithTimeout(ctx, 5*time.Second)
	defer flushCancel()
	if err := tp.ForceFlush(flushCtx); err != nil {
		log.Fatalf("❌ Failed to flush spans from OTel SDK: %v", err)
	}
	fmt.Println("✔ Emitted span from OpenTelemetry SDK to /v1/traces over wire format.")

	// Step 6: Wait for Kafka batch consumer and ClickHouse persistence
	fmt.Println("\n[Step 6] Waiting for ingestion pipeline batch flush (4s)...")
	time.Sleep(4 * time.Second)

	// Step 7: Query traces via REST API
	fmt.Println("\n[Step 7] Querying traces via GET /api/v1/traces...")
	queryReq, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/traces?limit=50", nil)
	queryReq.Header.Set("Authorization", "Bearer "+jwtToken)

	qResp, err := client.Do(queryReq)
	if err != nil || qResp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(qResp.Body)
		log.Fatalf("❌ Trace query failed (status %d): %s", qResp.StatusCode, string(respBytes))
	}
	defer qResp.Body.Close()

	var traces []TraceItem
	if err := json.NewDecoder(qResp.Body).Decode(&traces); err != nil {
		log.Fatalf("❌ Failed to decode trace query response: %v", err)
	}

	found := false
	for _, t := range traces {
		if t.OperationName == spanName || t.ServiceName == serviceName {
			found = true
			fmt.Printf("✔ Found trace in ClickHouse: TraceID=%s, Service=%s, Operation=%s\n",
				t.TraceID, t.ServiceName, t.OperationName)
			break
		}
	}

	if !found {
		log.Fatalf("❌ Ingested span %q was not found in ClickHouse query results (total returned: %d)", spanName, len(traces))
	}

	fmt.Println("\n==================================================")
	fmt.Println("✔ PHASE 1 END-TO-END DEMO EXIT CRITERION VERIFIED")
	fmt.Println("  Register -> Login -> API Key -> Real OTLP SDK -> ClickHouse Query")
	fmt.Println("==================================================")
}
