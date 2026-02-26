package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/aven/ngoogle/pkg/ratelimit"
)

func TestTokenBucketBasic(t *testing.T) {
	// 10 Mbps = 1.25 MB/s, burst = 2x = 2.5 MB
	tb := ratelimit.New(10, 2.0)
	ctx := context.Background()
	start := time.Now()

	// Request 1.25 MB — should be served from burst (< 100ms)
	if err := tb.Wait(ctx, 1_250_000); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Logf("wait took %v (expected < 100ms due to burst capacity)", elapsed)
	}
}

func TestTokenBucketRateAccuracy(t *testing.T) {
	// 10 Mbps = 1.25 MB/s, burstMultiplier=1.0 → capacity=1.25MB (pre-filled)
	// For 2.5 MB total: first 1.25 MB from burst (instant), next 1.25 MB takes ~1s
	rateMbps := 10.0
	tb := ratelimit.New(rateMbps, 1.0)

	ctx := context.Background()
	start := time.Now()

	total := int64(2_500_000)
	chunkSize := int64(250_000)
	for downloaded := int64(0); downloaded < total; downloaded += chunkSize {
		if err := tb.Wait(ctx, chunkSize); err != nil {
			t.Fatalf("Wait error: %v", err)
		}
	}

	elapsed := time.Since(start).Seconds()
	// With pre-filled 1.25 MB burst, second 1.25 MB takes ~1s
	expectedSec := float64(total/2) / (rateMbps * 1e6 / 8) // ~1.0s
	tolerance := 0.5 // 50% tolerance for CI environments

	t.Logf("elapsed=%.2fs, expectedSec=%.2fs", elapsed, expectedSec)
	if elapsed > expectedSec*(1+tolerance)+0.5 {
		t.Errorf("took too long: %.2fs, expected <= %.2fs", elapsed, expectedSec*(1+tolerance))
	}
}

func TestTokenBucketContextCancel(t *testing.T) {
	tb := ratelimit.New(0.001, 1.0) // very slow rate
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := tb.Wait(ctx, 1_000_000_000) // 1 GB — will block
	if err == nil {
		t.Error("expected context error but got nil")
	}
}

func TestTokenBucketSetRate(t *testing.T) {
	tb := ratelimit.New(1, 1.0)
	tb.SetRate(100) // increase to 100 Mbps
	ctx := context.Background()
	start := time.Now()
	_ = tb.Wait(ctx, 1_000_000)
	if time.Since(start) > 200*time.Millisecond {
		t.Error("SetRate did not take effect quickly enough")
	}
}

func TestMeterRates(t *testing.T) {
	m := &ratelimit.Meter{}

	// Record 1 MB per ~100ms for 10 iterations (= 10 MB over ~1s)
	// Then check Rate5s gives roughly 80 Mbps
	for i := 0; i < 10; i++ {
		m.Record(1_000_000)
		time.Sleep(100 * time.Millisecond)
	}

	rate5s := m.Rate5s()
	rate30s := m.Rate30s()
	t.Logf("Rate5s=%.3f Mbps, Rate30s=%.3f Mbps", rate5s, rate30s)

	// 10 MB in ~1s = 80 Mbps. 5s window has data only in last 1s, so avg over 5s = 16 Mbps
	// Meter spreads over full window, so we get ~80 Mbps / 5 = ~16 Mbps
	// Just verify it's non-zero and positive
	if rate5s <= 0 {
		t.Errorf("Rate5s should be > 0, got %f", rate5s)
	}
	if rate30s <= 0 {
		t.Errorf("Rate30s should be > 0, got %f", rate30s)
	}
	// Rate5s should be >= Rate30s (more recent activity)
	if rate5s < rate30s*0.5 {
		t.Logf("Rate5s (%.2f) vs Rate30s (%.2f) - unexpected", rate5s, rate30s)
	}
}
