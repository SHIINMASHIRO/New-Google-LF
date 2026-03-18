package service

import (
	"context"
	"testing"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store/sqlite"
)

func TestTaskGroupListReturnsSummaries(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	poolA := &model.URLPool{
		ID:        "pool-a",
		Name:      "Videos",
		Type:      model.URLPoolTypeYoutube,
		CreatedAt: now,
		UpdatedAt: now,
	}
	poolA.SetURLs([]string{"https://youtu.be/example"})
	if err := st.URLPools().Create(ctx, poolA); err != nil {
		t.Fatal(err)
	}

	poolB := &model.URLPool{
		ID:        "pool-b",
		Name:      "Files",
		Type:      model.URLPoolTypeStatic,
		CreatedAt: now,
		UpdatedAt: now,
	}
	poolB.SetURLs([]string{"https://example.com/file.bin"})
	if err := st.URLPools().Create(ctx, poolB); err != nil {
		t.Fatal(err)
	}

	group := &model.TaskGroup{
		ID:             "group-1",
		Name:           "Mixed Group",
		ExecutionScope: model.TaskExecutionScopeGlobal,
		TargetRateMbps: 100,
		Distribution:   model.DistributionFlat,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	group.SetPoolIDs([]string{poolA.ID, poolB.ID})
	if err := st.TaskGroups().Create(ctx, group); err != nil {
		t.Fatal(err)
	}

	taskA := &model.Task{
		ID:             "task-a",
		GroupID:        group.ID,
		Name:           "task-a",
		Type:           model.TaskTypeYoutube,
		URLPoolID:      poolA.ID,
		Status:         model.TaskStatusRunning,
		TargetRateMbps: 50,
		Distribution:   model.DistributionFlat,
		TotalBytesDone: 12,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	taskA.SetTargetURLs(poolA.URLs)
	if err := st.Tasks().Create(ctx, taskA); err != nil {
		t.Fatal(err)
	}

	taskB := &model.Task{
		ID:             "task-b",
		GroupID:        group.ID,
		Name:           "task-b",
		Type:           model.TaskTypeStatic,
		URLPoolID:      poolB.ID,
		Status:         model.TaskStatusDone,
		TargetRateMbps: 50,
		Distribution:   model.DistributionFlat,
		TotalBytesDone: 23,
		CreatedAt:      now.Add(time.Second),
		UpdatedAt:      now.Add(time.Second),
	}
	taskB.SetTargetURLs(poolB.URLs)
	if err := st.Tasks().Create(ctx, taskB); err != nil {
		t.Fatal(err)
	}

	svc := NewTaskGroupService(st, NewTaskService(st))
	groups, err := svc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	got := groups[0]
	if got.PoolCount != 2 {
		t.Fatalf("expected pool_count=2, got %d", got.PoolCount)
	}
	if got.ChildCount != 2 {
		t.Fatalf("expected child_count=2, got %d", got.ChildCount)
	}
	if got.TotalBytesDone != 35 {
		t.Fatalf("expected total_bytes_done=35, got %d", got.TotalBytesDone)
	}
	if got.Status != model.TaskStatusRunning {
		t.Fatalf("expected running status, got %s", got.Status)
	}
	if got.Type != model.TaskTypeMixed {
		t.Fatalf("expected mixed type, got %s", got.Type)
	}
	if len(got.Pools) != 0 {
		t.Fatalf("expected list payload to omit pools, got %d", len(got.Pools))
	}
	if len(got.Children) != 0 {
		t.Fatalf("expected list payload to omit children, got %d", len(got.Children))
	}
}
