package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

// TaskService handles task CRUD and state transitions.
type TaskService struct {
	store store.Store
}

// NewTaskService creates a new TaskService.
func NewTaskService(st store.Store) *TaskService {
	return &TaskService{store: st}
}

// Create creates a new task.
func (s *TaskService) Create(ctx context.Context, req *CreateTaskRequest) (*model.Task, error) {
	if req.TargetURL == "" {
		return nil, fmt.Errorf("target_url is required")
	}
	if req.Type != model.TaskTypeYoutube && req.Type != model.TaskTypeStatic {
		return nil, fmt.Errorf("invalid task type: %s", req.Type)
	}
	dist := req.Distribution
	if dist == "" {
		dist = model.DistributionFlat
	}
	now := time.Now()
	t := &model.Task{
		ID:                  generateID(),
		Name:                req.Name,
		Type:                req.Type,
		TargetURL:           req.TargetURL,
		AgentID:             req.AgentID,
		Status:              model.TaskStatusPending,
		TargetRateMbps:      req.TargetRateMbps,
		StartAt:             req.StartAt,
		EndAt:               req.EndAt,
		DurationSec:         req.DurationSec,
		TotalBytesTarget:    req.TotalBytesTarget,
		TotalRequestsTarget: req.TotalRequestsTarget,
		DispatchRateTpm:     req.DispatchRateTpm,
		DispatchBatchSize:   req.DispatchBatchSize,
		Distribution:        dist,
		JitterPct:           req.JitterPct,
		RampUpSec:           req.RampUpSec,
		RampDownSec:         req.RampDownSec,
		TrafficProfileID:    req.TrafficProfileID,
		ConcurrentFragments: req.ConcurrentFragments,
		Retries:             req.Retries,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if t.DispatchBatchSize <= 0 {
		t.DispatchBatchSize = 1
	}
	if t.Retries <= 0 {
		t.Retries = 3
	}
	if t.ConcurrentFragments <= 0 {
		t.ConcurrentFragments = 1
	}
	if err := s.store.Tasks().Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// CreateTaskRequest is the input for task creation.
type CreateTaskRequest struct {
	Name                string            `json:"name"`
	Type                model.TaskType    `json:"type"`
	TargetURL           string            `json:"target_url"`
	AgentID             string            `json:"agent_id"`
	TargetRateMbps      float64           `json:"target_rate_mbps"`
	StartAt             *time.Time        `json:"start_at,omitempty"`
	EndAt               *time.Time        `json:"end_at,omitempty"`
	DurationSec         int               `json:"duration_sec"`
	TotalBytesTarget    int64             `json:"total_bytes_target"`
	TotalRequestsTarget int64             `json:"total_requests_target"`
	DispatchRateTpm     int               `json:"dispatch_rate_tpm"`
	DispatchBatchSize   int               `json:"dispatch_batch_size"`
	Distribution        model.Distribution `json:"distribution"`
	JitterPct           float64           `json:"jitter_pct"`
	RampUpSec           int               `json:"ramp_up_sec"`
	RampDownSec         int               `json:"ramp_down_sec"`
	TrafficProfileID    string            `json:"traffic_profile_id"`
	ConcurrentFragments int               `json:"concurrent_fragments"`
	Retries             int               `json:"retries"`
}

// Get returns a single task.
func (s *TaskService) Get(ctx context.Context, id string) (*model.Task, error) {
	return s.store.Tasks().Get(ctx, id)
}

// List returns all tasks.
func (s *TaskService) List(ctx context.Context) ([]*model.Task, error) {
	return s.store.Tasks().List(ctx)
}

// Dispatch dispatches a task to its assigned agent.
func (s *TaskService) Dispatch(ctx context.Context, taskID string) error {
	t, err := s.store.Tasks().Get(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != model.TaskStatusPending {
		return fmt.Errorf("task %s is not pending (status=%s)", taskID, t.Status)
	}
	now := time.Now()
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusDispatched, now, "dispatched_at")
}

// Stop stops a running or dispatched task.
func (s *TaskService) Stop(ctx context.Context, taskID string) error {
	t, err := s.store.Tasks().Get(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status == model.TaskStatusDone || t.Status == model.TaskStatusFailed || t.Status == model.TaskStatusStopped {
		return fmt.Errorf("task %s is already terminal (status=%s)", taskID, t.Status)
	}
	now := time.Now()
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusStopped, now, "finished_at")
}

// RecordMetrics saves task metrics from an agent report.
func (s *TaskService) RecordMetrics(ctx context.Context, m *model.TaskMetrics) error {
	m.RecordedAt = time.Now()
	if err := s.store.TaskMetrics().Insert(ctx, m); err != nil {
		return err
	}
	// Update total bytes on the task
	return s.store.Tasks().UpdateBytes(ctx, m.TaskID, m.BytesTotal)
}

// PullTasks returns tasks assigned to an agent that are ready to execute.
func (s *TaskService) PullTasks(ctx context.Context, agentID string) ([]*model.Task, error) {
	statuses := []model.TaskStatus{model.TaskStatusDispatched, model.TaskStatusRunning}
	return s.store.Tasks().ListByAgent(ctx, agentID, statuses)
}

// MarkRunning marks a task as running.
func (s *TaskService) MarkRunning(ctx context.Context, taskID string) error {
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusRunning, time.Now(), "started_at")
}

// MarkDone marks a task as done.
func (s *TaskService) MarkDone(ctx context.Context, taskID string) error {
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusDone, time.Now(), "finished_at")
}

// MarkFailed marks a task as failed with an error.
func (s *TaskService) MarkFailed(ctx context.Context, taskID string, reason string) error {
	_ = s.store.Tasks().SetError(ctx, taskID, reason)
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusFailed, time.Now(), "finished_at")
}

// GetMetrics returns metrics for a task.
func (s *TaskService) GetMetrics(ctx context.Context, taskID string, from, to time.Time) ([]*model.TaskMetrics, error) {
	return s.store.TaskMetrics().ListByTask(ctx, taskID, from, to)
}
