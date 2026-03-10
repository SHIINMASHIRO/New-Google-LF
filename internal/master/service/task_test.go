package service

import (
	"context"
	"testing"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store/sqlite"
)

func TestMarkDoneDoesNotOverrideStoppedTask(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	svc := NewTaskService(st)
	now := time.Now()
	task := &model.Task{
		ID:           "t1",
		Type:         model.TaskTypeYoutube,
		TargetURL:    "https://youtu.be/example",
		Status:       model.TaskStatusStopped,
		FinishedAt:   &now,
		Distribution: model.DistributionFlat,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := st.Tasks().Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}

	if err := svc.MarkDone(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}

	got, err := st.Tasks().Get(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.TaskStatusStopped {
		t.Fatalf("expected stopped status to be preserved, got %s", got.Status)
	}
}

func TestMarkRunningAndDoneTransitionTask(t *testing.T) {
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	svc := NewTaskService(st)
	now := time.Now()
	task := &model.Task{
		ID:           "t2",
		Type:         model.TaskTypeStatic,
		TargetURL:    "https://example.com/file.bin",
		Status:       model.TaskStatusDispatched,
		Distribution: model.DistributionFlat,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := st.Tasks().Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}

	if err := svc.MarkRunning(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	running, err := st.Tasks().Get(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if running.Status != model.TaskStatusRunning {
		t.Fatalf("expected running, got %s", running.Status)
	}
	if running.StartedAt == nil {
		t.Fatal("expected started_at to be set")
	}

	if err := svc.MarkDone(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	done, err := st.Tasks().Get(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != model.TaskStatusDone {
		t.Fatalf("expected done, got %s", done.Status)
	}
	if done.FinishedAt == nil {
		t.Fatal("expected finished_at to be set")
	}
}
