package analysis

import (
	"math"
	"testing"
)

func TestWelfordAccumulator(t *testing.T) {
	acc := &WelfordAccumulator{}
	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}

	for _, v := range values {
		acc.Update(v)
	}

	if acc.Count != 5 {
		t.Fatalf("expected count 5, got %d", acc.Count)
	}

	expectedMean := 30.0
	if math.Abs(acc.Mean-expectedMean) > 1e-6 {
		t.Errorf("expected mean %.2f, got %.2f", expectedMean, acc.Mean)
	}

	// Sample variance for [10, 20, 30, 40, 50] is 250.0
	expectedVar := 250.0
	if math.Abs(acc.Variance()-expectedVar) > 1e-6 {
		t.Errorf("expected variance %.2f, got %.2f", expectedVar, acc.Variance())
	}

	expectedStdDev := math.Sqrt(250.0) // ~15.811
	if math.Abs(acc.StdDev()-expectedStdDev) > 1e-6 {
		t.Errorf("expected stdDev %.4f, got %.4f", expectedStdDev, acc.StdDev())
	}
}

func TestBaselineCalculator_ColdStartAndThresholds(t *testing.T) {
	calc := NewBaselineCalculator(BaselineOptions{
		MinSamples:  5,
		MinDuration: 50.0,
		MinDeltaMs:  30.0,
	})

	orgA := "org-test-alpha"
	service := "order-service"
	op := "CreateOrder"

	// 1. Cold start guard: with 4 samples, no anomaly should trigger even with large value
	for i := 0; i < 4; i++ {
		calc.Update(orgA, service, op, 100.0)
	}

	z, _, isAnom := calc.CalculateZScore(orgA, service, op, 500.0)
	if isAnom || z != 0 {
		t.Fatalf("expected cold start guard to suppress anomaly with < 5 samples, got isAnom=%v, z=%.2f", isAnom, z)
	}

	// 2. Add 5th sample: baseline now established
	calc.Update(orgA, service, op, 100.0)

	mean, stdDev, count, exists := calc.GetBaseline(orgA, service, op)
	if !exists || count != 5 {
		t.Fatalf("expected baseline to exist with count 5, got exists=%v count=%d", exists, count)
	}
	if mean != 100.0 {
		t.Errorf("expected mean 100.0, got %.2f", mean)
	}
	// All values identical (100.0) -> stdDev should be bounded by minStdDev (1.0)
	if stdDev < 1.0 {
		t.Errorf("expected stdDev floor >= 1.0, got %.2f", stdDev)
	}

	// 3. Normal request (faster or equal to baseline) should never be anomalous
	z, _, isAnom = calc.CalculateZScore(orgA, service, op, 90.0)
	if isAnom {
		t.Errorf("faster request (90ms) should not be flagged as anomaly, got isAnom=true, z=%.2f", z)
	}

	// 4. Large spike: 100ms baseline -> 250ms observed (delta 150ms >= 30ms, observed >= 50ms, Z >= 3.0)
	z, bMean, isAnom := calc.CalculateZScore(orgA, service, op, 250.0)
	if !isAnom {
		t.Fatalf("expected 250ms spike to be flagged as anomalous, got isAnom=false, z=%.2f", z)
	}
	if bMean != 100.0 {
		t.Errorf("expected baseline mean 100.0, got %.2f", bMean)
	}
	if z < 3.0 {
		t.Errorf("expected Z >= 3.0, got %.2f", z)
	}

	// 5. Small duration filter: 2ms baseline -> 10ms spike (delta 8ms < 30ms, observed 10ms < 50ms)
	calc.Update(orgA, "cache-svc", "Get", 2.0)
	calc.Update(orgA, "cache-svc", "Get", 2.0)
	calc.Update(orgA, "cache-svc", "Get", 2.0)
	calc.Update(orgA, "cache-svc", "Get", 2.0)
	calc.Update(orgA, "cache-svc", "Get", 2.0)

	_, _, isSmallAnom := calc.CalculateZScore(orgA, "cache-svc", "Get", 10.0)
	if isSmallAnom {
		t.Errorf("microsecond noise (10ms) should be filtered out by MinDuration / MinDeltaMs")
	}
}

func TestBaselineCalculator_TenantIsolation(t *testing.T) {
	calc := NewBaselineCalculator()

	orgA := "org-alpha"
	orgB := "org-beta"
	service := "checkout-api"
	op := "ProcessCheckout"

	// Train Org A with 100ms baseline
	for i := 0; i < 10; i++ {
		calc.Update(orgA, service, op, 100.0)
	}

	// Train Org B with 1000ms baseline
	for i := 0; i < 10; i++ {
		calc.Update(orgB, service, op, 1000.0)
	}

	meanA, _, _, existsA := calc.GetBaseline(orgA, service, op)
	meanB, _, _, existsB := calc.GetBaseline(orgB, service, op)

	if !existsA || !existsB {
		t.Fatalf("expected both baselines to exist")
	}

	if math.Abs(meanA-100.0) > 1e-3 {
		t.Errorf("Org A mean expected 100.0, got %.2f", meanA)
	}
	if math.Abs(meanB-1000.0) > 1e-3 {
		t.Errorf("Org B mean expected 1000.0, got %.2f", meanB)
	}

	// 300ms is anomalous for Org A (baseline 100ms), but NOT for Org B (baseline 1000ms)
	_, _, isAnomA := calc.CalculateZScore(orgA, service, op, 300.0)
	_, _, isAnomB := calc.CalculateZScore(orgB, service, op, 300.0)

	if !isAnomA {
		t.Errorf("expected 300ms to be anomalous for Org A (baseline 100ms)")
	}
	if isAnomB {
		t.Errorf("expected 300ms to NOT be anomalous for Org B (baseline 1000ms)")
	}
}
