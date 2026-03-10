package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store/sqlite"
)

func newTestStore(t *testing.T) interface{ Close() error } {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	return st
}

func TestAgentUpsertAndGet(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	a := &model.Agent{
		ID:            "agent1",
		Hostname:      "test-host",
		IP:            "10.0.0.1",
		Port:          0,
		Token:         "tok123",
		Status:        model.AgentStatusOnline,
		Version:       "1.0.0",
		LastHeartbeat: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := st.Agents().Upsert(ctx, a); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := st.Agents().Get(ctx, "agent1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Hostname != "test-host" {
		t.Errorf("expected test-host, got %s", got.Hostname)
	}
	if got.Status != model.AgentStatusOnline {
		t.Errorf("expected online, got %s", got.Status)
	}
}

func TestTaskCreateAndGet(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	task := &model.Task{
		ID:             "task1",
		Name:           "test",
		Type:           model.TaskTypeMixed,
		AgentID:        "",
		ExecutionScope: model.TaskExecutionScopeGlobal,
		Status:         model.TaskStatusPending,
		TargetRateMbps: 10,
		Distribution:   model.DistributionFlat,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	task.SetTargetURLs([]string{"https://example.com/a", "https://www.youtube.com/watch?v=test"})

	if err := st.Tasks().Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	got, err := st.Tasks().Get(ctx, "task1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.TargetURL != "https://example.com/a" {
		t.Errorf("expected first URL, got %s", got.TargetURL)
	}
	if got.Status != model.TaskStatusPending {
		t.Errorf("expected pending, got %s", got.Status)
	}
	if got.ExecutionScope != model.TaskExecutionScopeGlobal {
		t.Errorf("expected global scope, got %s", got.ExecutionScope)
	}
	if len(got.TargetURLs) != 2 {
		t.Fatalf("expected 2 target URLs, got %d", len(got.TargetURLs))
	}
}

func TestURLPoolCreateAndGet(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	pool := &model.URLPool{
		ID:          "pool1",
		Name:        "yt-pool",
		Type:        model.URLPoolTypeYoutube,
		Description: "videos",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	pool.SetURLs([]string{"https://youtu.be/a", "https://youtu.be/b"})

	if err := st.URLPools().Create(ctx, pool); err != nil {
		t.Fatalf("create pool: %v", err)
	}

	got, err := st.URLPools().Get(ctx, "pool1")
	if err != nil {
		t.Fatalf("get pool: %v", err)
	}
	if got.Type != model.URLPoolTypeYoutube {
		t.Fatalf("expected youtube pool, got %s", got.Type)
	}
	if len(got.URLs) != 2 {
		t.Fatalf("expected 2 urls, got %d", len(got.URLs))
	}
}

func TestURLPoolUpdate(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	pool := &model.URLPool{
		ID:          "pool-update",
		Name:        "before",
		Type:        model.URLPoolTypeStatic,
		Description: "before",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	pool.SetURLs([]string{"https://example.com/a"})
	if err := st.URLPools().Create(ctx, pool); err != nil {
		t.Fatalf("create pool: %v", err)
	}

	pool.Name = "after"
	pool.Description = "after"
	pool.UpdatedAt = now.Add(time.Minute)
	pool.SetURLs([]string{"https://example.com/b", "https://example.com/c"})
	if err := st.URLPools().Update(ctx, pool); err != nil {
		t.Fatalf("update pool: %v", err)
	}

	got, err := st.URLPools().Get(ctx, "pool-update")
	if err != nil {
		t.Fatalf("get updated pool: %v", err)
	}
	if got.Name != "after" || got.Description != "after" {
		t.Fatalf("unexpected updated metadata: %+v", got)
	}
	if len(got.URLs) != 2 || got.URLs[0] != "https://example.com/b" {
		t.Fatalf("unexpected updated urls: %+v", got.URLs)
	}
}

func TestTaskGroupCreateAndTaskListByGroup(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	group := &model.TaskGroup{
		ID:             "group1",
		Name:           "group",
		ExecutionScope: model.TaskExecutionScopeGlobal,
		TargetRateMbps: 100,
		Distribution:   model.DistributionFlat,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	group.SetPoolIDs([]string{"pool-a", "pool-b"})
	if err := st.TaskGroups().Create(ctx, group); err != nil {
		t.Fatalf("create group: %v", err)
	}

	task := &model.Task{
		ID:             "task-group-child",
		GroupID:        "group1",
		Name:           "child",
		Type:           model.TaskTypeStatic,
		Status:         model.TaskStatusPending,
		TargetRateMbps: 10,
		Distribution:   model.DistributionFlat,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	task.SetTargetURLs([]string{"https://example.com/a"})
	if err := st.Tasks().Create(ctx, task); err != nil {
		t.Fatalf("create child task: %v", err)
	}

	gotGroup, err := st.TaskGroups().Get(ctx, "group1")
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if len(gotGroup.PoolIDs) != 2 {
		t.Fatalf("expected 2 pool ids, got %d", len(gotGroup.PoolIDs))
	}

	children, err := st.Tasks().ListByGroup(ctx, "group1")
	if err != nil {
		t.Fatalf("list by group: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child task, got %d", len(children))
	}
	if children[0].GroupID != "group1" {
		t.Fatalf("expected child task group1, got %s", children[0].GroupID)
	}
}

func TestTaskStatusTransition(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	task := &model.Task{
		ID: "t1", Type: model.TaskTypeStatic, TargetURL: "https://x.com",
		Status: model.TaskStatusPending, Distribution: model.DistributionFlat,
		CreatedAt: now, UpdatedAt: now,
	}
	_ = st.Tasks().Create(ctx, task)
	_ = st.Tasks().UpdateStatus(ctx, "t1", model.TaskStatusRunning)
	got, _ := st.Tasks().Get(ctx, "t1")
	if got.Status != model.TaskStatusRunning {
		t.Errorf("expected running, got %s", got.Status)
	}
}

func TestBandwidthPurge(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	old := time.Now().Add(-8 * 24 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	_ = st.Bandwidth().Insert(ctx, &model.BandwidthSample{AgentID: "a1", RateMbps: 5, RecordedAt: old})
	_ = st.Bandwidth().Insert(ctx, &model.BandwidthSample{AgentID: "a1", RateMbps: 10, RecordedAt: recent})

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	if err := st.Bandwidth().PurgeOlderThan(ctx, cutoff); err != nil {
		t.Fatalf("purge: %v", err)
	}

	pts, _ := st.Bandwidth().History(ctx, "a1",
		time.Now().Add(-10*24*time.Hour),
		time.Now())
	if len(pts) != 1 {
		t.Errorf("expected 1 sample after purge, got %d", len(pts))
	}
	if pts[0].RateMbps != 10 {
		t.Errorf("expected recent sample (10 Mbps), got %f", pts[0].RateMbps)
	}
}

func TestMetricsInsertAndList(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	m := &model.TaskMetrics{
		TaskID: "t1", AgentID: "a1",
		BytesTotal: 1_000_000, BytesDelta: 50_000,
		RateMbps5s: 8.0, RateMbps30s: 7.5,
		RequestCount: 10, ErrorCount: 0,
		RecordedAt: now,
	}
	if err := st.TaskMetrics().Insert(ctx, m); err != nil {
		t.Fatalf("insert metrics: %v", err)
	}

	list, err := st.TaskMetrics().ListByTask(ctx, "t1",
		now.Add(-1*time.Minute), now.Add(1*time.Minute))
	if err != nil {
		t.Fatalf("list metrics: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 metric, got %d", len(list))
	}
	if list[0].RateMbps5s != 8.0 {
		t.Errorf("expected 8.0 Mbps, got %f", list[0].RateMbps5s)
	}
}

func TestMetricsLatestByTaskAgents(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()
	metrics := []*model.TaskMetrics{
		{TaskID: "t1", AgentID: "a1", BytesTotal: 100, RecordedAt: now.Add(-10 * time.Second)},
		{TaskID: "t1", AgentID: "a1", BytesTotal: 200, RecordedAt: now},
		{TaskID: "t1", AgentID: "a2", BytesTotal: 300, RecordedAt: now.Add(-5 * time.Second)},
	}
	for _, m := range metrics {
		if err := st.TaskMetrics().Insert(ctx, m); err != nil {
			t.Fatalf("insert metrics: %v", err)
		}
	}

	latest, err := st.TaskMetrics().LatestByTaskAgents(ctx, "t1")
	if err != nil {
		t.Fatalf("latest by agents: %v", err)
	}
	if len(latest) != 2 {
		t.Fatalf("expected 2 latest metrics, got %d", len(latest))
	}
	if latest[0].AgentID != "a1" || latest[0].BytesTotal != 200 {
		t.Fatalf("unexpected latest for a1: %+v", latest[0])
	}
	if latest[1].AgentID != "a2" || latest[1].BytesTotal != 300 {
		t.Fatalf("unexpected latest for a2: %+v", latest[1])
	}
}
