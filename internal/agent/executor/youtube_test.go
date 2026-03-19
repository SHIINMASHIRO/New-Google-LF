package executor

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aven/ngoogle/internal/model"
)

func TestYoutubeWorkerCountUsesTargetRateAndCapsByMax(t *testing.T) {
	task := &model.Task{TargetRateMbps: 1000}
	// maxYoutubeWorkers is 2, so high rate still caps at 2
	if got := youtubeWorkerCount(task, 20); got != 2 {
		t.Fatalf("expected 2 workers (capped by maxYoutubeWorkers), got %d", got)
	}
	if got := youtubeWorkerCount(task, 1); got != 1 {
		t.Fatalf("expected worker count capped by url count, got %d", got)
	}
}

func TestYoutubeWorkerCountCappedByMax(t *testing.T) {
	task := &model.Task{TargetRateMbps: 50, ConcurrentFragments: 6}
	// Even with high hint, capped at maxYoutubeWorkers (2)
	if got := youtubeWorkerCount(task, 10); got != 2 {
		t.Fatalf("expected 2 workers (capped by maxYoutubeWorkers), got %d", got)
	}
}

func TestYoutubeWorkerCountReturnsOneForSingleURL(t *testing.T) {
	task := &model.Task{TargetRateMbps: 5000, ConcurrentFragments: 8}
	if got := youtubeWorkerCount(task, 1); got != 1 {
		t.Fatalf("expected single worker for single url, got %d", got)
	}
}

func TestBuildYtdlpArgsIncludesJSRuntimeWhenAvailable(t *testing.T) {
	task := &model.Task{TargetRateMbps: 100}
	args := buildYtdlpArgsWithJSRuntime(task, "https://youtu.be/test", "node:/usr/bin/node")

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--js-runtimes node:/usr/bin/node") {
		t.Fatalf("expected js runtime hint in args, got %v", args)
	}
}

func TestMonitorYoutubeProgressFailsAfterTimeout(t *testing.T) {
	var totalBytes int64
	errTracker := &taskErrorTracker{}
	errTracker.Set(errors.New("sign in to confirm you're not a bot"))

	err := monitorYoutubeProgress(context.Background(), &totalBytes, 30*time.Millisecond, 5*time.Millisecond, errTracker)
	if err == nil {
		t.Fatal("expected stall error")
	}
	if !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("expected stall error, got %v", err)
	}
	if !strings.Contains(err.Error(), "not a bot") {
		t.Fatalf("expected last yt-dlp error to be included, got %v", err)
	}
}

func TestMonitorYoutubeProgressReturnsNilAfterCancel(t *testing.T) {
	var totalBytes int64
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(15 * time.Millisecond)
		atomic.StoreInt64(&totalBytes, 1024)
		cancel()
	}()

	err := monitorYoutubeProgress(ctx, &totalBytes, 100*time.Millisecond, 5*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("expected nil after cancel, got %v", err)
	}
}

func TestLineTailKeepsMostRecentLines(t *testing.T) {
	tail := newLineTail(3)
	tail.Add("first")
	tail.Add("second")
	tail.Add("third")
	tail.Add("fourth")

	if got := tail.String(); got != "second | third | fourth" {
		t.Fatalf("unexpected tail contents: %s", got)
	}
}
