package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

type TaskGroupService struct {
	store   store.Store
	taskSvc *TaskService
}

func NewTaskGroupService(st store.Store, taskSvc *TaskService) *TaskGroupService {
	return &TaskGroupService{store: st, taskSvc: taskSvc}
}

type CreateTaskGroupRequest struct {
	Name                string                   `json:"name"`
	Description         string                   `json:"description"`
	PoolIDs             []string                 `json:"pool_ids"`
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

func (s *TaskGroupService) Create(ctx context.Context, req *CreateTaskGroupRequest) (*model.TaskGroup, error) {
	poolIDs := uniqueStrings(req.PoolIDs)
	if len(poolIDs) == 0 {
		return nil, fmt.Errorf("pool_ids is required")
	}

	pools := make([]*model.URLPool, 0, len(poolIDs))
	for _, id := range poolIDs {
		pool, err := s.store.URLPools().Get(ctx, id)
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}

	scope := req.ExecutionScope
	if scope == "" {
		scope = model.TaskExecutionScopeGlobal
	}
	if scope != model.TaskExecutionScopeSingleAgent && scope != model.TaskExecutionScopeGlobal {
		return nil, fmt.Errorf("invalid execution_scope: %s", scope)
	}
	if scope == model.TaskExecutionScopeGlobal {
		req.AgentID = ""
	}
	if scope == model.TaskExecutionScopeSingleAgent && req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required for single_agent task groups")
	}

	dist := req.Distribution
	if dist == "" {
		dist = model.DistributionFlat
	}
	now := time.Now()
	group := &model.TaskGroup{
		ID:                  generateID(),
		Name:                req.Name,
		Description:         req.Description,
		AgentID:             req.AgentID,
		ExecutionScope:      scope,
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
		Pools:               pools,
	}
	group.SetPoolIDs(poolIDs)
	if group.DispatchBatchSize <= 0 {
		group.DispatchBatchSize = 1
	}
	if group.Retries <= 0 {
		group.Retries = 3
	}
	if group.ConcurrentFragments <= 0 {
		group.ConcurrentFragments = 1
	}
	if err := s.store.TaskGroups().Create(ctx, group); err != nil {
		return nil, err
	}

	perTaskRate := splitFloat(group.TargetRateMbps, len(pools))
	perTaskBytes := splitInt64(group.TotalBytesTarget, len(pools))
	perTaskRequests := splitInt64(group.TotalRequestsTarget, len(pools))

	children := make([]*model.Task, 0, len(pools))
	for i, pool := range pools {
		child := &model.Task{
			ID:                  generateID(),
			GroupID:             group.ID,
			Name:                childTaskName(group.Name, pool.Name),
			Type:                pool.TaskType(),
			URLPoolID:           pool.ID,
			URLPool:             pool.Clone(),
			AgentID:             group.AgentID,
			ExecutionScope:      group.ExecutionScope,
			Status:              model.TaskStatusPending,
			TargetRateMbps:      perTaskRate[i],
			StartAt:             group.StartAt,
			EndAt:               group.EndAt,
			DurationSec:         group.DurationSec,
			TotalBytesTarget:    perTaskBytes[i],
			TotalRequestsTarget: perTaskRequests[i],
			DispatchRateTpm:     group.DispatchRateTpm,
			DispatchBatchSize:   group.DispatchBatchSize,
			Distribution:        group.Distribution,
			JitterPct:           group.JitterPct,
			RampUpSec:           group.RampUpSec,
			RampDownSec:         group.RampDownSec,
			TrafficProfileID:    group.TrafficProfileID,
			ConcurrentFragments: group.ConcurrentFragments,
			Retries:             group.Retries,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		child.SetTargetURLs(pool.URLs)
		if err := s.store.Tasks().Create(ctx, child); err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	group.Children = children
	return s.enrichGroup(ctx, group)
}

func (s *TaskGroupService) List(ctx context.Context) ([]*model.TaskGroup, error) {
	groups, err := s.store.TaskGroups().List(ctx)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return groups, nil
	}

	pools, err := s.store.URLPools().List(ctx)
	if err != nil {
		return nil, err
	}
	poolByID := make(map[string]*model.URLPool, len(pools))
	for _, pool := range pools {
		poolByID[pool.ID] = pool
	}

	tasks, err := s.store.Tasks().List(ctx)
	if err != nil {
		return nil, err
	}
	childrenByGroup := make(map[string][]*model.Task, len(groups))
	for _, task := range tasks {
		if task.GroupID == "" {
			continue
		}
		childrenByGroup[task.GroupID] = append(childrenByGroup[task.GroupID], task.Clone())
	}
	for _, children := range childrenByGroup {
		sort.Slice(children, func(i, j int) bool {
			return children[i].CreatedAt.Before(children[j].CreatedAt)
		})
	}

	summaries := make([]*model.TaskGroup, 0, len(groups))
	for _, group := range groups {
		summary := group.Clone()
		summary.Normalize()

		groupPools := make([]*model.URLPool, 0, len(summary.PoolIDs))
		for _, id := range summary.PoolIDs {
			if pool, ok := poolByID[id]; ok {
				groupPools = append(groupPools, pool)
			}
		}
		children := childrenByGroup[summary.ID]

		summary.PoolCount = len(groupPools)
		summary.ChildCount = len(children)
		summary.TotalBytesDone = aggregateGroupBytes(children)
		summary.Status = aggregateGroupStatus(children)
		summary.Type = aggregateGroupType(groupPools, children)
		summary.Pools = nil
		summary.Children = nil

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *TaskGroupService) Get(ctx context.Context, id string) (*model.TaskGroup, error) {
	group, err := s.store.TaskGroups().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.enrichGroup(ctx, group)
}

func (s *TaskGroupService) Dispatch(ctx context.Context, id string) error {
	children, err := s.store.Tasks().ListByGroup(ctx, id)
	if err != nil {
		return err
	}
	if len(children) == 0 {
		return fmt.Errorf("task group %s has no child tasks", id)
	}
	for _, child := range children {
		if child.Status == model.TaskStatusPending {
			if err := s.taskSvc.Dispatch(ctx, child.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *TaskGroupService) Stop(ctx context.Context, id string) error {
	children, err := s.store.Tasks().ListByGroup(ctx, id)
	if err != nil {
		return err
	}
	if len(children) == 0 {
		return fmt.Errorf("task group %s has no child tasks", id)
	}
	for _, child := range children {
		if child.Status != model.TaskStatusDone && child.Status != model.TaskStatusFailed && child.Status != model.TaskStatusStopped {
			if err := s.taskSvc.Stop(ctx, child.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *TaskGroupService) GetMetrics(ctx context.Context, id string, from, to time.Time) ([]*model.TaskMetrics, error) {
	children, err := s.store.Tasks().ListByGroup(ctx, id)
	if err != nil {
		return nil, err
	}
	var all []*model.TaskMetrics
	for _, child := range children {
		metrics, err := s.store.TaskMetrics().ListByTask(ctx, child.ID, from, to)
		if err != nil {
			return nil, err
		}
		all = append(all, metrics...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].RecordedAt.Before(all[j].RecordedAt)
	})
	return all, nil
}

func (s *TaskGroupService) enrichGroup(ctx context.Context, group *model.TaskGroup) (*model.TaskGroup, error) {
	group = group.Clone()
	group.Normalize()

	pools := make([]*model.URLPool, 0, len(group.PoolIDs))
	for _, id := range group.PoolIDs {
		pool, err := s.store.URLPools().Get(ctx, id)
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}
	group.Pools = pools

	children, err := s.store.Tasks().ListByGroup(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	poolByID := make(map[string]*model.URLPool, len(pools))
	for _, pool := range pools {
		poolByID[pool.ID] = pool
	}
	for i := range children {
		children[i] = children[i].Clone()
		if pool, ok := poolByID[children[i].URLPoolID]; ok {
			children[i].URLPool = pool.Clone()
			if len(children[i].TargetURLs) == 0 {
				children[i].SetTargetURLs(pool.URLs)
			}
			if children[i].Type == "" {
				children[i].Type = pool.TaskType()
			}
		}
		children[i].Normalize()
	}
	group.Children = children
	group.PoolCount = len(group.Pools)
	group.ChildCount = len(group.Children)
	group.TotalBytesDone = aggregateGroupBytes(children)
	group.Status = aggregateGroupStatus(children)
	group.Type = aggregateGroupType(pools, children)
	return group, nil
}

func childTaskName(groupName, poolName string) string {
	if groupName == "" {
		return poolName
	}
	if poolName == "" {
		return groupName
	}
	return fmt.Sprintf("%s [%s]", groupName, poolName)
}

func splitFloat(total float64, n int) []float64 {
	out := make([]float64, n)
	if n == 0 || total == 0 {
		return out
	}
	base := total / float64(n)
	for i := range out {
		out[i] = base
	}
	return out
}

func splitInt64(total int64, n int) []int64 {
	out := make([]int64, n)
	if n == 0 || total == 0 {
		return out
	}
	base := total / int64(n)
	rem := total % int64(n)
	for i := range out {
		out[i] = base
		if int64(i) < rem {
			out[i]++
		}
	}
	return out
}

func aggregateGroupBytes(children []*model.Task) int64 {
	var total int64
	for _, child := range children {
		total += child.TotalBytesDone
	}
	return total
}

func aggregateGroupStatus(children []*model.Task) model.TaskStatus {
	if len(children) == 0 {
		return model.TaskStatusPending
	}
	counts := map[model.TaskStatus]int{}
	for _, child := range children {
		counts[child.Status]++
	}
	switch {
	case counts[model.TaskStatusRunning] > 0:
		return model.TaskStatusRunning
	case counts[model.TaskStatusDispatched] > 0:
		return model.TaskStatusDispatched
	case counts[model.TaskStatusPending] == len(children):
		return model.TaskStatusPending
	case counts[model.TaskStatusFailed] > 0:
		return model.TaskStatusFailed
	case counts[model.TaskStatusStopped] > 0:
		return model.TaskStatusStopped
	case counts[model.TaskStatusDone] == len(children):
		return model.TaskStatusDone
	default:
		return model.TaskStatusPending
	}
}

func aggregateGroupType(pools []*model.URLPool, children []*model.Task) model.TaskType {
	types := map[model.TaskType]struct{}{}
	for _, pool := range pools {
		types[pool.TaskType()] = struct{}{}
	}
	for _, child := range children {
		if child.Type != "" {
			types[child.Type] = struct{}{}
		}
	}
	switch len(types) {
	case 0:
		return model.TaskTypeStatic
	case 1:
		for taskType := range types {
			return taskType
		}
	}
	return model.TaskTypeMixed
}

func uniqueStrings(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
