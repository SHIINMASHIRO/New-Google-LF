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
		Type:           model.TaskTypeStatic,
		TargetURL:      "https://example.com",
		AgentID:        "agent1",
		Status:         model.TaskStatusPending,
		TargetRateMbps: 10,
		Distribution:   model.DistributionFlat,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := st.Tasks().Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	got, err := st.Tasks().Get(ctx, "task1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.TargetURL != "https://example.com" {
		t.Errorf("expected example.com, got %s", got.TargetURL)
	}
	if got.Status != model.TaskStatusPending {
		t.Errorf("expected pending, got %s", got.Status)
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
