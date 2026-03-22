package service

import (
	"context"
	"sync"
	"time"

	"github.com/aven/ngoogle/internal/store"
)

// DashboardService aggregates metrics for the dashboard.
type DashboardService struct {
	store store.Store

	// In-memory cache to avoid hitting SQLite on every request.
	// Overview is refreshed by a background goroutine every second.
	overviewMu    sync.RWMutex
	overviewCache *OverviewResponse
	overviewAt    time.Time

	historyMu    sync.RWMutex
	historyCache []store.BandwidthPoint
	historyKey   string
	historyAt    time.Time
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(st store.Store) *DashboardService {
	return &DashboardService{store: st}
}

// RunOverviewRefresh periodically refreshes the overview cache in background.
// This decouples dashboard reads from the database entirely.
func (s *DashboardService) RunOverviewRefresh(ctx context.Context) {
	// Initial load
	s.refreshOverview(ctx)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshOverview(ctx)
		}
	}
}

func (s *DashboardService) refreshOverview(ctx context.Context) {
	agents, err := s.store.Agents().List(ctx)
	if err != nil {
		return
	}
	tasks, err := s.store.Tasks().List(ctx)
	if err != nil {
		return
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

	resp := &OverviewResponse{
		TotalAgents:   len(agents),
		OnlineAgents:  onlineCount,
		TotalTasks:    len(tasks),
		RunningTasks:  runningTasks,
		TotalRateMbps: totalMbps,
		Agents:        agentStats,
	}

	s.overviewMu.Lock()
	s.overviewCache = resp
	s.overviewAt = time.Now()
	s.overviewMu.Unlock()
}

// Overview returns the cached overview (never blocks on DB).
func (s *DashboardService) Overview(_ context.Context) (*OverviewResponse, error) {
	s.overviewMu.RLock()
	resp := s.overviewCache
	s.overviewMu.RUnlock()
	if resp != nil {
		return resp, nil
	}
	// Fallback: cache not yet populated (first few ms after startup)
	return &OverviewResponse{}, nil
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

// BandwidthHistory returns aggregated bandwidth samples (cached for 30s).
func (s *DashboardService) BandwidthHistory(ctx context.Context, from, to time.Time, stepSec int) ([]store.BandwidthPoint, error) {
	if stepSec <= 0 {
		stepSec = 60
	}

	key := from.Truncate(time.Minute).Format(time.RFC3339) + "|" + to.Truncate(time.Minute).Format(time.RFC3339)

	s.historyMu.RLock()
	if s.historyKey == key && time.Since(s.historyAt) < 30*time.Second {
		cached := s.historyCache
		s.historyMu.RUnlock()
		return cached, nil
	}
	s.historyMu.RUnlock()

	points, err := s.store.Bandwidth().AggregateHistory(ctx, from, to, stepSec)
	if err != nil {
		return nil, err
	}

	s.historyMu.Lock()
	s.historyCache = points
	s.historyKey = key
	s.historyAt = time.Now()
	s.historyMu.Unlock()

	return points, nil
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
