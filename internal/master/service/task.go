package service

import (
	"context"
	"fmt"
	"hash/crc32"
	"net/url"
	"strings"
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
	pool, urls, taskType, err := s.resolveTaskSource(ctx, req)
	if err != nil {
		return nil, err
	}
	dist := req.Distribution
	if dist == "" {
		dist = model.DistributionFlat
	}
	scope := req.ExecutionScope
	if scope == "" {
		if pool != nil || len(urls) > 1 {
			scope = model.TaskExecutionScopeGlobal
		} else {
			scope = model.TaskExecutionScopeSingleAgent
		}
	}
	if scope != model.TaskExecutionScopeSingleAgent && scope != model.TaskExecutionScopeGlobal {
		return nil, fmt.Errorf("invalid execution_scope: %s", scope)
	}
	if scope == model.TaskExecutionScopeGlobal {
		req.AgentID = ""
	}
	if scope == model.TaskExecutionScopeSingleAgent && req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required for single_agent tasks")
	}
	now := time.Now()
	t := &model.Task{
		ID:                  generateID(),
		Name:                req.Name,
		Type:                taskType,
		URLPoolID:           req.URLPoolID,
		AgentID:             req.AgentID,
		ExecutionScope:      scope,
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
	t.URLPool = pool
	t.SetTargetURLs(urls)
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
	Name                string                   `json:"name"`
	Type                model.TaskType           `json:"type"`
	URLPoolID           string                   `json:"url_pool_id"`
	TargetURL           string                   `json:"target_url"`
	TargetURLs          []string                 `json:"target_urls"`
	AgentID             string                   `json:"agent_id"`
	ExecutionScope      model.TaskExecutionScope `json:"execution_scope"`
	TargetRateMbps      float64                  `json:"target_rate_mbps"`
	StartAt             *time.Time               `json:"start_at,omitempty"`
	EndAt               *time.Time               `json:"end_at,omitempty"`
	DurationSec         int                      `json:"duration_sec"`
	TotalBytesTarget    int64                    `json:"total_bytes_target"`
	TotalRequestsTarget int64                    `json:"total_requests_target"`
	DispatchRateTpm     int                      `json:"dispatch_rate_tpm"`
	DispatchBatchSize   int                      `json:"dispatch_batch_size"`
	Distribution        model.Distribution       `json:"distribution"`
	JitterPct           float64                  `json:"jitter_pct"`
	RampUpSec           int                      `json:"ramp_up_sec"`
	RampDownSec         int                      `json:"ramp_down_sec"`
	TrafficProfileID    string                   `json:"traffic_profile_id"`
	ConcurrentFragments int                      `json:"concurrent_fragments"`
	Retries             int                      `json:"retries"`
}

// Get returns a single task.
func (s *TaskService) Get(ctx context.Context, id string) (*model.Task, error) {
	t, err := s.store.Tasks().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.enrichTask(ctx, t)
}

// List returns all tasks.
func (s *TaskService) List(ctx context.Context) ([]*model.Task, error) {
	tasks, err := s.store.Tasks().List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i], err = s.enrichTask(ctx, tasks[i])
		if err != nil {
			return nil, err
		}
	}
	return tasks, nil
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
	snapshots, err := s.store.TaskMetrics().LatestByTaskAgents(ctx, m.TaskID)
	if err != nil {
		return err
	}
	var totalBytes int64
	for _, snap := range snapshots {
		totalBytes += snap.BytesTotal
	}
	return s.store.Tasks().UpdateBytes(ctx, m.TaskID, totalBytes)
}

// PullTasks returns tasks assigned to an agent that are ready to execute.
func (s *TaskService) PullTasks(ctx context.Context, agentID string) ([]*model.Task, error) {
	tasks, err := s.store.Tasks().List(ctx)
	if err != nil {
		return nil, err
	}
	agents, err := s.store.Agents().List(ctx)
	if err != nil {
		return nil, err
	}
	onlineAgents := 0
	for _, a := range agents {
		if a.Status == model.AgentStatusOnline {
			onlineAgents++
		}
	}
	var runnable []*model.Task
	for _, task := range tasks {
		if task.Status != model.TaskStatusDispatched && task.Status != model.TaskStatusRunning {
			continue
		}
		task, err = s.attachURLPool(ctx, task)
		if err != nil {
			return nil, err
		}
		task.Normalize()
		switch task.ExecutionScope {
		case model.TaskExecutionScopeGlobal:
			runnable = append(runnable, prepareTaskForAgent(task, agentID, onlineAgents))
		case model.TaskExecutionScopeSingleAgent, "":
			if task.AgentID == agentID {
				runnable = append(runnable, prepareTaskForAgent(task, agentID, onlineAgents))
			}
		}
	}
	return runnable, nil
}

// MarkRunning marks a task as running.
func (s *TaskService) MarkRunning(ctx context.Context, taskID string) error {
	t, err := s.store.Tasks().Get(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case model.TaskStatusRunning, model.TaskStatusDone, model.TaskStatusFailed, model.TaskStatusStopped:
		return nil
	}
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusRunning, time.Now(), "started_at")
}

// MarkDone marks a task as done.
func (s *TaskService) MarkDone(ctx context.Context, taskID string) error {
	t, err := s.store.Tasks().Get(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case model.TaskStatusDone, model.TaskStatusFailed, model.TaskStatusStopped:
		return nil
	}
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusDone, time.Now(), "finished_at")
}

// MarkFailed marks a task as failed with an error.
func (s *TaskService) MarkFailed(ctx context.Context, taskID string, reason string) error {
	t, err := s.store.Tasks().Get(ctx, taskID)
	if err != nil {
		return err
	}
	switch t.Status {
	case model.TaskStatusDone, model.TaskStatusFailed, model.TaskStatusStopped:
		return nil
	}
	_ = s.store.Tasks().SetError(ctx, taskID, reason)
	return s.store.Tasks().UpdateStatusWithTime(ctx, taskID, model.TaskStatusFailed, time.Now(), "finished_at")
}

// GetMetrics returns metrics for a task.
func (s *TaskService) GetMetrics(ctx context.Context, taskID string, from, to time.Time) ([]*model.TaskMetrics, error) {
	return s.store.TaskMetrics().ListByTask(ctx, taskID, from, to)
}

func (s *TaskService) enrichTask(ctx context.Context, task *model.Task) (*model.Task, error) {
	task = task.Clone()
	var err error
	task, err = s.attachURLPool(ctx, task)
	if err != nil {
		return nil, err
	}
	task.Normalize()

	snapshots, err := s.store.TaskMetrics().LatestByTaskAgents(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	var totalBytes int64
	for _, snap := range snapshots {
		totalBytes += snap.BytesTotal
	}
	task.TotalBytesDone = totalBytes
	return task, nil
}

func (s *TaskService) resolveTaskSource(ctx context.Context, req *CreateTaskRequest) (*model.URLPool, []string, model.TaskType, error) {
	if req.URLPoolID != "" {
		pool, err := s.store.URLPools().Get(ctx, req.URLPoolID)
		if err != nil {
			return nil, nil, "", err
		}
		pool.Normalize()
		if len(pool.URLs) == 0 {
			return nil, nil, "", fmt.Errorf("url pool %s has no urls", req.URLPoolID)
		}
		taskType := pool.TaskType()
		if req.Type != "" && req.Type != taskType {
			return nil, nil, "", fmt.Errorf("task type %s does not match url pool type %s", req.Type, pool.Type)
		}
		return pool, pool.URLs, taskType, nil
	}

	urls := normalizeTaskURLs(req.TargetURLs, req.TargetURL)
	if len(urls) == 0 {
		return nil, nil, "", fmt.Errorf("url_pool_id is required")
	}
	if req.Type == "" {
		req.Type = inferTaskType(urls)
	}
	if req.Type != model.TaskTypeYoutube && req.Type != model.TaskTypeStatic && req.Type != model.TaskTypeMixed {
		return nil, nil, "", fmt.Errorf("invalid task type: %s", req.Type)
	}
	if err := validateTaskURLs(req.Type, urls); err != nil {
		return nil, nil, "", err
	}
	return nil, urls, req.Type, nil
}

func (s *TaskService) attachURLPool(ctx context.Context, task *model.Task) (*model.Task, error) {
	if task.URLPoolID == "" {
		return task, nil
	}
	pool, err := s.store.URLPools().Get(ctx, task.URLPoolID)
	if err != nil {
		return nil, err
	}
	task = task.Clone()
	task.URLPool = pool
	task.Type = pool.TaskType()
	task.SetTargetURLs(pool.URLs)
	return task, nil
}

func normalizeTaskURLs(urls []string, single string) []string {
	combined := make([]string, 0, len(urls)+1)
	combined = append(combined, urls...)
	if single != "" {
		combined = append(combined, single)
	}

	out := make([]string, 0, len(combined))
	seen := make(map[string]struct{}, len(combined))
	for _, raw := range combined {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

func validateTaskURLs(taskType model.TaskType, urls []string) error {
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid url: %s", raw)
		}
		if taskType == model.TaskTypeYoutube && !isYoutubeURL(raw) {
			return fmt.Errorf("youtube task contains non-youtube url: %s", raw)
		}
		if taskType == model.TaskTypeStatic && isYoutubeURL(raw) {
			return fmt.Errorf("static task contains youtube url: %s", raw)
		}
	}
	return nil
}

func inferTaskType(urls []string) model.TaskType {
	if len(urls) == 0 {
		return model.TaskTypeStatic
	}
	firstIsYoutube := isYoutubeURL(urls[0])
	for _, u := range urls[1:] {
		if isYoutubeURL(u) != firstIsYoutube {
			return model.TaskTypeMixed
		}
	}
	if firstIsYoutube {
		return model.TaskTypeYoutube
	}
	return model.TaskTypeStatic
}

func isYoutubeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be")
}

func prepareTaskForAgent(task *model.Task, agentID string, onlineAgents int) *model.Task {
	cp := task.Clone()
	cp.Normalize()
	if cp.ExecutionScope == model.TaskExecutionScopeGlobal && onlineAgents > 0 && cp.TargetRateMbps > 0 {
		cp.TargetRateMbps = cp.TargetRateMbps / float64(onlineAgents)
	}
	urls := cp.URLs()
	if len(urls) > 1 {
		offset := int(crc32.ChecksumIEEE([]byte(cp.ID+":"+agentID))) % len(urls)
		cp.TargetURLs = rotateURLs(urls, offset)
		cp.TargetURL = cp.TargetURLs[0]
		cp.TargetURLsJSON = ""
		cp.Normalize()
	}
	return cp
}

func rotateURLs(urls []string, offset int) []string {
	if len(urls) == 0 {
		return nil
	}
	offset = offset % len(urls)
	if offset == 0 {
		out := make([]string, len(urls))
		copy(out, urls)
		return out
	}
	out := make([]string, 0, len(urls))
	out = append(out, urls[offset:]...)
	out = append(out, urls[:offset]...)
	return out
}
