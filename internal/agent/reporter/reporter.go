// Package reporter handles periodic metrics reporting to the Master.
package reporter

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aven/ngoogle/internal/agent/client"
	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/pkg/ratelimit"
)

// TaskReporter tracks and reports metrics for a single task.
type TaskReporter struct {
	taskID  string
	agentID string
	client  *client.Client
	meter   *ratelimit.Meter

	mu         sync.Mutex
	bytesTotal int64
	reqCount   int64
	errCount   int64
}

// NewTaskReporter creates a reporter for a task.
func NewTaskReporter(taskID, agentID string, c *client.Client) *TaskReporter {
	return &TaskReporter{
		taskID:  taskID,
		agentID: agentID,
		client:  c,
		meter:   &ratelimit.Meter{},
	}
}

// Meter returns the rate meter (for use by executors).
func (r *TaskReporter) Meter() *ratelimit.Meter { return r.meter }

// RecordBytes records downloaded bytes.
func (r *TaskReporter) RecordBytes(n int64) {
	r.mu.Lock()
	r.bytesTotal += n
	r.reqCount++
	r.mu.Unlock()
	r.meter.Record(n)
}

// RecordError records an error.
func (r *TaskReporter) RecordError() {
	r.mu.Lock()
	r.errCount++
	r.mu.Unlock()
}

// Run starts periodic reporting until ctx is cancelled.
func (r *TaskReporter) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Final report
			r.report(context.Background())
			return
		case <-ticker.C:
			r.report(ctx)
		}
	}
}

func (r *TaskReporter) report(ctx context.Context) {
	r.mu.Lock()
	m := &model.TaskMetrics{
		TaskID:       r.taskID,
		AgentID:      r.agentID,
		BytesTotal:   r.bytesTotal,
		RequestCount: r.reqCount,
		ErrorCount:   r.errCount,
		RateMbps5s:   r.meter.Rate5s(),
		RateMbps30s:  r.meter.Rate30s(),
	}
	r.mu.Unlock()

	if err := r.client.ReportMetrics(ctx, m); err != nil {
		slog.Warn("report metrics failed", "task", r.taskID, "err", err)
	}
}

// CurrentRate returns the current 5s average rate in Mbps.
func (r *TaskReporter) CurrentRate() float64 { return r.meter.Rate5s() }
