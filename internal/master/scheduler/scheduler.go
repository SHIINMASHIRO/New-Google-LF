// Package scheduler handles task lifecycle: scheduling, dispatching, stopping.
package scheduler

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

// Scheduler watches pending tasks and dispatches them according to their time windows.
type Scheduler struct {
	store  store.Store
	mu     sync.Mutex
	active map[string]context.CancelFunc // taskID → cancel
}

// New creates a new Scheduler.
func New(st store.Store) *Scheduler {
	return &Scheduler{
		store:  st,
		active: make(map[string]context.CancelFunc),
	}
}

// Run starts the scheduling loop, blocking until ctx is done.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	tasks, err := s.store.Tasks().List(ctx)
	if err != nil {
		slog.Error("scheduler list tasks", "err", err)
		return
	}
	now := time.Now()
	for _, t := range tasks {
		switch t.Status {
		case model.TaskStatusPending, model.TaskStatusDispatched:
			if shouldStart(t, now) {
				s.markRunning(ctx, t)
			}
		case model.TaskStatusRunning:
			if shouldStop(t, now) {
				s.markStopped(ctx, t)
			}
		}
	}
}

func shouldStart(t *model.Task, now time.Time) bool {
	if t.StartAt != nil && now.Before(*t.StartAt) {
		return false
	}
	return true
}

func shouldStop(t *model.Task, now time.Time) bool {
	if t.EndAt != nil && now.After(*t.EndAt) {
		return true
	}
	if t.DurationSec > 0 && t.StartedAt != nil && now.Sub(*t.StartedAt) > time.Duration(t.DurationSec)*time.Second {
		return true
	}
	if t.TotalBytesTarget > 0 && t.TotalBytesDone >= t.TotalBytesTarget {
		return true
	}
	return false
}

func (s *Scheduler) markRunning(ctx context.Context, t *model.Task) {
	now := time.Now()
	if err := s.store.Tasks().UpdateStatusWithTime(ctx, t.ID, model.TaskStatusRunning, now, "started_at"); err != nil {
		slog.Error("scheduler mark running", "task", t.ID, "err", err)
	}
}

func (s *Scheduler) markStopped(ctx context.Context, t *model.Task) {
	now := time.Now()
	if err := s.store.Tasks().UpdateStatusWithTime(ctx, t.ID, model.TaskStatusStopped, now, "finished_at"); err != nil {
		slog.Error("scheduler mark stopped", "task", t.ID, "err", err)
	}
}

// Stop requests cancellation for a given task.
func (s *Scheduler) Stop(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, ok := s.active[taskID]; ok {
		cancel()
		delete(s.active, taskID)
	}
}

// ─── Traffic Profile Curve ────────────────────────────────────────────────────

// ProfilePoint is {offset_sec, rate_pct} for diurnal curves.
type ProfilePoint struct {
	OffsetSec float64 `json:"offset_sec"`
	RatePct   float64 `json:"rate_pct"`
}

// RateForTask computes the target rate multiplier [0,1] at the given elapsed time
// based on the task's distribution and profile.
func RateForTask(t *model.Task, elapsed time.Duration, points []ProfilePoint) float64 {
	switch t.Distribution {
	case model.DistributionRamp:
		return rampMultiplier(t, elapsed)
	case model.DistributionDiurnal:
		return diurnalMultiplier(points, elapsed)
	default:
		return flatMultiplier(t, elapsed)
	}
}

func flatMultiplier(t *model.Task, elapsed time.Duration) float64 {
	rampUp := time.Duration(t.RampUpSec) * time.Second
	rampDown := time.Duration(t.RampDownSec) * time.Second
	totalDur := time.Duration(t.DurationSec) * time.Second
	if t.EndAt != nil && t.StartedAt != nil {
		totalDur = t.EndAt.Sub(*t.StartedAt)
	}
	if elapsed < rampUp {
		return elapsed.Seconds() / rampUp.Seconds()
	}
	if rampDown > 0 && totalDur > 0 && elapsed > totalDur-rampDown {
		remaining := totalDur - elapsed
		if remaining <= 0 {
			return 0
		}
		return remaining.Seconds() / rampDown.Seconds()
	}
	return 1.0
}

func rampMultiplier(t *model.Task, elapsed time.Duration) float64 {
	return flatMultiplier(t, elapsed)
}

func diurnalMultiplier(points []ProfilePoint, elapsed time.Duration) float64 {
	if len(points) == 0 {
		return 1.0
	}
	sec := elapsed.Seconds()
	// linear interpolation
	for i := 0; i < len(points)-1; i++ {
		p0, p1 := points[i], points[i+1]
		if sec >= p0.OffsetSec && sec <= p1.OffsetSec {
			frac := (sec - p0.OffsetSec) / (p1.OffsetSec - p0.OffsetSec)
			return p0.RatePct/100 + frac*(p1.RatePct-p0.RatePct)/100
		}
	}
	return points[len(points)-1].RatePct / 100
}

// ApplyJitter applies ±jitterPct random jitter to a duration.
func ApplyJitter(d time.Duration, jitterPct float64) time.Duration {
	if jitterPct <= 0 {
		return d
	}
	factor := 1.0 + (rand.Float64()*2-1)*jitterPct/100.0
	return time.Duration(math.Round(float64(d) * factor))
}

// DispatchInterval returns the interval between dispatch batches.
func DispatchInterval(tpm int, batchSize int) time.Duration {
	if tpm <= 0 || batchSize <= 0 {
		return 0
	}
	return time.Minute / time.Duration(tpm)
}
