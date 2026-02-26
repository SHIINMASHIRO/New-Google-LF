package service

import (
	"context"
	"time"

	"github.com/aven/ngoogle/internal/store"
)

// DashboardService aggregates metrics for the dashboard.
type DashboardService struct {
	store store.Store
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(st store.Store) *DashboardService {
	return &DashboardService{store: st}
}

// Overview returns current totals and per-agent stats.
func (s *DashboardService) Overview(ctx context.Context) (*OverviewResponse, error) {
	agents, err := s.store.Agents().List(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := s.store.Tasks().List(ctx)
	if err != nil {
		return nil, err
	}

	var totalMbps float64
	onlineCount := 0
	type agentStat struct {
		ID       string  `json:"id"`
		Hostname string  `json:"hostname"`
		IP       string  `json:"ip"`
		RateMbps float64 `json:"rate_mbps"`
		Status   string  `json:"status"`
	}
	agentStats := make([]agentStat, 0, len(agents))
	for _, a := range agents {
		if string(a.Status) == "online" {
			onlineCount++
			totalMbps += a.CurrentRateMbps
		}
		agentStats = append(agentStats, agentStat{
			ID:       a.ID,
			Hostname: a.Hostname,
			IP:       a.IP,
			RateMbps: a.CurrentRateMbps,
			Status:   string(a.Status),
		})
	}

	runningTasks := 0
	for _, t := range tasks {
		if t.Status == "running" {
			runningTasks++
		}
	}

	return &OverviewResponse{
		TotalAgents:   len(agents),
		OnlineAgents:  onlineCount,
		TotalTasks:    len(tasks),
		RunningTasks:  runningTasks,
		TotalRateMbps: totalMbps,
		Agents:        agentStats,
	}, nil
}

// OverviewResponse is the dashboard overview payload.
type OverviewResponse struct {
	TotalAgents   int         `json:"total_agents"`
	OnlineAgents  int         `json:"online_agents"`
	TotalTasks    int         `json:"total_tasks"`
	RunningTasks  int         `json:"running_tasks"`
	TotalRateMbps float64     `json:"total_rate_mbps"`
	Agents        interface{} `json:"agents"`
}

// BandwidthHistory returns aggregated bandwidth samples.
func (s *DashboardService) BandwidthHistory(ctx context.Context, from, to time.Time, stepSec int) ([]store.BandwidthPoint, error) {
	if stepSec <= 0 {
		stepSec = 60
	}
	return s.store.Bandwidth().AggregateHistory(ctx, from, to, stepSec)
}

// RunPurge runs a daily purge of bandwidth samples older than 7 days.
func (s *DashboardService) RunPurge(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-7 * 24 * time.Hour)
			if err := s.store.Bandwidth().PurgeOlderThan(ctx, cutoff); err != nil {
				// log silently
				_ = err
			}
		}
	}
}
