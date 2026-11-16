package main

import (
	"fmt"
	"math"
	mathrand "math/rand"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
)

// OpenTelemetry Demo ("Astronomy Shop") official services and RPC/HTTP endpoints.
var (
	otelDemoServices = []string{
		"frontend", "cartservice", "productcatalogservice", "currencyservice",
		"paymentservice", "shippingservice", "emailservice", "checkoutservice",
		"recommendationservice", "adservice", "accountingservice", "frauddetectionservice",
	}

	otelDemoProductIDs = []string{
		"OLJCESPC7Z", "66VCHSJNUP", "1YMWWN1N4O", "L9ECAV7KIM",
		"2ZYFJ3GM2N", "0PUK6V6EV0", "LS4PSXUNUM", "9SIQT8TOJO",
	}

	otelDemoCurrencies = []string{"USD", "EUR", "JPY", "GBP", "CAD"}

	otelDemoLanguages = []string{"go", "python", "java", "csharp", "nodejs"}
)

// GenerateOTelDemoSpans creates realistic spans modeled directly on the official OpenTelemetry Demo Astronomy Shop trace call graphs.
// Each trace consists of an 8-span hierarchical call tree:
//   1. frontend: HTTP POST /cart/checkout (root span)
//   2. frontend -> cartservice: rpc.GetCart
//   3. frontend -> productcatalogservice: rpc.GetProduct
//   4. frontend -> currencyservice: rpc.Convert
//   5. frontend -> checkoutservice: rpc.PlaceOrder
//   6. checkoutservice -> paymentservice: rpc.Charge
//   7. checkoutservice -> shippingservice: rpc.ShipOrder
//   8. checkoutservice -> emailservice: rpc.SendOrderConfirmation
func GenerateOTelDemoSpans(totalSpans int, orgID string, baseTime time.Time) []domain.Span {
	r := mathrand.New(mathrand.NewSource(42 + time.Now().UnixNano()))
	spans := make([]domain.Span, 0, totalSpans)

	traceCount := totalSpans / 8
	if traceCount == 0 {
		traceCount = 1
	}

	spanIndex := 0
	for t := 0; t < traceCount && spanIndex < totalSpans; t++ {
		traceID := randomHex(16) // 32 hex chars (16 bytes)
		orderID := fmt.Sprintf("order-%s", randomHex(6))
		productID := otelDemoProductIDs[r.Intn(len(otelDemoProductIDs))]
		currency := otelDemoCurrencies[r.Intn(len(otelDemoCurrencies))]
		traceStart := baseTime.Add(time.Duration(t*50) * time.Millisecond)

		// Determine if this trace has an injected error (e.g. payment failure)
		isErrorTrace := r.Float64() < 0.05

		// Span 1: Root frontend HTTP POST /cart/checkout
		rootSpanID := randomHex(8) // 16 hex chars (8 bytes)
		rootDuration := uint32(math.Max(180, math.Exp(5.4+0.3*r.NormFloat64())))
		rootStatus := uint8(1)
		rootErrMsg := ""
		if isErrorTrace {
			rootStatus = 2
			rootErrMsg = "rpc error: code = Internal desc = Payment authorization failed: insufficient credit"
		}

		spans = append(spans, domain.Span{
			OrgID:         orgID,
			TraceID:       traceID,
			SpanID:        rootSpanID,
			ParentSpanID:  "", // Root span: zero-byte literal
			ServiceName:   "frontend",
			OperationName: "HTTP POST /cart/checkout",
			DurationMs:    rootDuration,
			StatusCode:    rootStatus,
			ErrorMessage:  rootErrMsg,
			Tags: map[string]string{
				"telemetry.sdk.name":     "opentelemetry",
				"telemetry.sdk.language": "go",
				"telemetry.sdk.version":  "1.24.0",
				"service.name":           "frontend",
				"service.version":        "1.10.0",
				"http.method":            "POST",
				"http.route":             "/cart/checkout",
				"http.status_code":       fmt.Sprintf("%d", []int{200, 200, 200, 500}[r.Intn(4)]),
				"http.user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/124.0",
				"net.host.name":          "frontend.astronomy-shop.svc.cluster.local",
				"net.peer.name":          "ingress-nginx",
				"net.peer.port":          "8080",
				"app.order.id":           orderID,
				"app.cart.items_count":   fmt.Sprintf("%d", r.Intn(4)+1),
				"k8s.pod.name":           fmt.Sprintf("frontend-%s", randomHex(3)),
				"k8s.namespace.name":     "otel-demo",
			},
			StartTime: traceStart,
		})
		spanIndex++
		if spanIndex >= totalSpans {
			break
		}

		// Helper to append child span
		addChild := func(svc, op, rpcService, rpcMethod, lang string, relStartMs, durationMs uint32, extraTags map[string]string, isErr bool) string {
			if spanIndex >= totalSpans {
				return ""
			}
			childID := randomHex(8)
			st := uint8(1)
			errMsg := ""
			if isErr {
				st = 2
				errMsg = "payment declined by processor: card limit exceeded"
			}
			tags := map[string]string{
				"telemetry.sdk.name":     "opentelemetry",
				"telemetry.sdk.language": lang,
				"service.name":           svc,
				"rpc.system":             "grpc",
				"rpc.service":            rpcService,
				"rpc.method":             rpcMethod,
				"rpc.grpc.status_code":   fmt.Sprintf("%d", st-1),
				"net.peer.name":          fmt.Sprintf("%s.astronomy-shop.svc.cluster.local", svc),
				"net.peer.port":          "50051",
				"k8s.pod.name":           fmt.Sprintf("%s-%s", svc, randomHex(3)),
				"k8s.namespace.name":     "otel-demo",
			}
			for k, v := range extraTags {
				tags[k] = v
			}
			spans = append(spans, domain.Span{
				OrgID:         orgID,
				TraceID:       traceID,
				SpanID:        childID,
				ParentSpanID:  rootSpanID,
				ServiceName:   svc,
				OperationName: op,
				DurationMs:    durationMs,
				StatusCode:    st,
				ErrorMessage:  errMsg,
				Tags:          tags,
				StartTime:     traceStart.Add(time.Duration(relStartMs) * time.Millisecond),
			})
			spanIndex++
			return childID
		}

		// Span 2: frontend -> cartservice
		addChild("cartservice", "hipstershop.CartService/GetCart", "hipstershop.CartService", "GetCart", "csharp",
			5, 25, map[string]string{"app.user.id": fmt.Sprintf("usr-%s", randomHex(4))}, false)

		// Span 3: frontend -> productcatalogservice
		addChild("productcatalogservice", "hipstershop.ProductCatalogService/GetProduct", "hipstershop.ProductCatalogService", "GetProduct", "go",
			12, 18, map[string]string{"app.product.id": productID}, false)

		// Span 4: frontend -> currencyservice
		addChild("currencyservice", "hipstershop.CurrencyService/Convert", "hipstershop.CurrencyService", "Convert", "nodejs",
			20, 15, map[string]string{"app.currency.from": "USD", "app.currency.to": currency}, false)

		// Span 5: frontend -> checkoutservice
		checkoutSpanID := addChild("checkoutservice", "hipstershop.CheckoutService/PlaceOrder", "hipstershop.CheckoutService", "PlaceOrder", "go",
			35, 120, map[string]string{"app.order.id": orderID, "app.order.total": fmt.Sprintf("$%d.99", r.Intn(200)+20)}, isErrorTrace)

		if checkoutSpanID != "" && spanIndex < totalSpans {
			// Helper for grandchildren (parent is checkoutservice)
			addGrandchild := func(svc, op, rpcService, rpcMethod, lang string, relStartMs, durationMs uint32, extraTags map[string]string, isErr bool) {
				if spanIndex >= totalSpans {
					return
				}
				gID := randomHex(8)
				st := uint8(1)
				errMsg := ""
				if isErr {
					st = 2
					errMsg = "payment authorization declined"
				}
				tags := map[string]string{
					"telemetry.sdk.name":     "opentelemetry",
					"telemetry.sdk.language": lang,
					"service.name":           svc,
					"rpc.system":             "grpc",
					"rpc.service":            rpcService,
					"rpc.method":             rpcMethod,
					"rpc.grpc.status_code":   fmt.Sprintf("%d", st-1),
					"net.peer.name":          fmt.Sprintf("%s.astronomy-shop.svc.cluster.local", svc),
					"net.peer.port":          "50051",
					"k8s.pod.name":           fmt.Sprintf("%s-%s", svc, randomHex(3)),
					"k8s.namespace.name":     "otel-demo",
				}
				for k, v := range extraTags {
					tags[k] = v
				}
				spans = append(spans, domain.Span{
					OrgID:         orgID,
					TraceID:       traceID,
					SpanID:        gID,
					ParentSpanID:  checkoutSpanID,
					ServiceName:   svc,
					OperationName: op,
					DurationMs:    durationMs,
					StatusCode:    st,
					ErrorMessage:  errMsg,
					Tags:          tags,
					StartTime:     traceStart.Add(time.Duration(relStartMs) * time.Millisecond),
				})
				spanIndex++
			}

			// Span 6: checkoutservice -> paymentservice
			addGrandchild("paymentservice", "hipstershop.PaymentService/Charge", "hipstershop.PaymentService", "Charge", "javascript",
				45, 40, map[string]string{"app.payment.card_type": "visa", "app.payment.amount": "$145.00"}, isErrorTrace)

			// Span 7: checkoutservice -> shippingservice
			addGrandchild("shippingservice", "hipstershop.ShippingService/ShipOrder", "hipstershop.ShippingService", "ShipOrder", "rust",
				75, 30, map[string]string{"app.shipping.carrier": "ups", "app.shipping.tracking_id": fmt.Sprintf("1Z%s", randomHex(8))}, false)

			// Span 8: checkoutservice -> emailservice
			addGrandchild("emailservice", "hipstershop.EmailService/SendOrderConfirmation", "hipstershop.EmailService", "SendOrderConfirmation", "ruby",
				95, 20, map[string]string{"app.email.recipient": fmt.Sprintf("user_%s@example.com", randomHex(4))}, false)
		}
	}

	return spans
}

