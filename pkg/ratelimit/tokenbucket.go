// Package ratelimit provides a token bucket rate limiter for bandwidth control.
package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

// TokenBucket is a thread-safe token bucket rate limiter.
// It limits byte throughput to a configured rate in Mbps.
type TokenBucket struct {
	mu       sync.Mutex
	tokens   float64   // available tokens (bytes)
	capacity float64   // max burst capacity (bytes)
	rate     float64   // fill rate (bytes/sec)
	lastFill time.Time
}

// New creates a TokenBucket for the given rate in Mbps.
// burst is the burst capacity multiplier (e.g. 2 = allow 2x the 1s rate as burst).
func New(rateMbps float64, burstMultiplier float64) *TokenBucket {
	if rateMbps <= 0 {
		rateMbps = math.MaxFloat64 / 1e6 // effectively unlimited
	}
	bps := rateMbps * 1e6 / 8 // bytes per second
	capacity := bps * burstMultiplier
	if capacity < 65536 {
		capacity = 65536 // min 64 KB burst
	}
	return &TokenBucket{
		tokens:   capacity,
		capacity: capacity,
		rate:     bps,
		lastFill: time.Now(),
	}
}

// SetRate updates the rate at runtime (Mbps).
func (tb *TokenBucket) SetRate(rateMbps float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	bps := rateMbps * 1e6 / 8
	tb.rate = bps
	tb.capacity = bps * 2
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
}

// Wait blocks until n bytes can be consumed from the bucket,
// respecting the provided context.
func (tb *TokenBucket) Wait(ctx context.Context, n int64) error {
	for {
		tb.mu.Lock()
		tb.fill()
		if tb.tokens >= float64(n) {
			tb.tokens -= float64(n)
			tb.mu.Unlock()
			return nil
		}
		// calculate wait duration
		deficit := float64(n) - tb.tokens
		waitDur := time.Duration(deficit / tb.rate * float64(time.Second))
		tb.mu.Unlock()

		if waitDur < time.Millisecond {
			waitDur = time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDur):
		}
	}
}

// TryConsume immediately consumes n bytes if available. Returns false if not enough tokens.
func (tb *TokenBucket) TryConsume(n int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.fill()
	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// fill refills the bucket based on elapsed time. Must be called with lock held.
func (tb *TokenBucket) fill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastFill).Seconds()
	tb.lastFill = now
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
}

// ─── Sliding Window Rate Meter ────────────────────────────────────────────────

// Meter tracks byte throughput over sliding windows.
type Meter struct {
	mu      sync.Mutex
	samples []sample
}

type sample struct {
	ts    time.Time
	bytes int64
}

// Record adds a byte count at the current time.
func (m *Meter) Record(n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.samples = append(m.samples, sample{ts: now, bytes: n})
	// keep only last 30s
	cutoff := now.Add(-30 * time.Second)
	for len(m.samples) > 0 && m.samples[0].ts.Before(cutoff) {
		m.samples = m.samples[1:]
	}
}

// Rate5s returns the average rate in Mbps over the last 5 seconds.
func (m *Meter) Rate5s() float64 { return m.rateOver(5 * time.Second) }

// Rate30s returns the average rate in Mbps over the last 30 seconds.
func (m *Meter) Rate30s() float64 { return m.rateOver(30 * time.Second) }

func (m *Meter) rateOver(window time.Duration) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-window)
	var total int64
	for _, s := range m.samples {
		if s.ts.After(cutoff) {
			total += s.bytes
		}
	}
	secs := window.Seconds()
	return float64(total) / secs / 1e6 * 8 // Mbps
}
