package scheduler_test

import (
	"testing"
	"time"

	"github.com/aven/ngoogle/internal/master/scheduler"
	"github.com/aven/ngoogle/internal/model"
)

func TestRateForTask_Flat(t *testing.T) {
	task := &model.Task{
		Distribution: model.DistributionFlat,
		DurationSec:  60,
		RampUpSec:    0,
		RampDownSec:  0,
	}
	// At any point during steady state, multiplier should be 1.0
	for _, sec := range []int{0, 15, 30, 45, 59} {
		elapsed := time.Duration(sec) * time.Second
		mult := scheduler.RateForTask(task, elapsed, nil)
		if mult != 1.0 {
			t.Errorf("sec=%d: expected 1.0, got %f", sec, mult)
		}
	}
}

func TestRateForTask_Ramp(t *testing.T) {
	task := &model.Task{
		Distribution: model.DistributionRamp,
		DurationSec:  60,
		RampUpSec:    10,
		RampDownSec:  10,
	}
	// DurationSec=60, RampDownSec=10 → ramp-down starts at 50s
	tests := []struct {
		sec     int
		minMult float64
		maxMult float64
	}{
		{0, 0.0, 0.01},   // start of ramp-up
		{5, 0.45, 0.55},  // mid ramp-up
		{10, 0.99, 1.01}, // end of ramp-up
		{30, 0.99, 1.01}, // steady
		{55, 0.45, 0.55}, // mid ramp-down (50s start, 60s end → 55s is mid)
		{60, 0.0, 0.01},  // end of ramp-down
	}
	for _, tc := range tests {
		elapsed := time.Duration(tc.sec) * time.Second
		mult := scheduler.RateForTask(task, elapsed, nil)
		if mult < tc.minMult || mult > tc.maxMult {
			t.Errorf("sec=%d: mult=%f, expected [%f, %f]", tc.sec, mult, tc.minMult, tc.maxMult)
		}
	}
}

func TestRateForTask_Diurnal(t *testing.T) {
	points := []scheduler.ProfilePoint{
		{OffsetSec: 0, RatePct: 20},
		{OffsetSec: 30, RatePct: 100},
		{OffsetSec: 60, RatePct: 50},
	}
	task := &model.Task{Distribution: model.DistributionDiurnal}

	// At 15s, should be interpolated between 20% and 100% = 60%
	mult := scheduler.RateForTask(task, 15*time.Second, points)
	expected := 0.60
	if mult < expected-0.05 || mult > expected+0.05 {
		t.Errorf("15s: mult=%f, expected ~%f", mult, expected)
	}

	// At 30s, should be exactly 100%
	mult = scheduler.RateForTask(task, 30*time.Second, points)
	if mult < 0.99 || mult > 1.01 {
		t.Errorf("30s: mult=%f, expected 1.0", mult)
	}
}

func TestApplyJitter(t *testing.T) {
	base := 100 * time.Millisecond
	// With 10% jitter, result should be in [90ms, 110ms]
	for i := 0; i < 100; i++ {
		result := scheduler.ApplyJitter(base, 10.0)
		if result < 90*time.Millisecond || result > 110*time.Millisecond {
			t.Errorf("jitter out of bounds: %v", result)
		}
	}
}

func TestDispatchInterval(t *testing.T) {
	// 60 tpm, batch 1 = 1 req/s = 1s interval
	interval := scheduler.DispatchInterval(60, 1)
	if interval != time.Second {
		t.Errorf("expected 1s, got %v", interval)
	}
	// 120 tpm = 0.5s
	interval = scheduler.DispatchInterval(120, 1)
	if interval != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", interval)
	}
}
