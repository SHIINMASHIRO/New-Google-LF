package executor

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/pkg/ratelimit"
)

// YoutubeExecutor runs yt-dlp as a managed subprocess.
type YoutubeExecutor struct{}

// Run executes the youtube task via yt-dlp in a loop until the task duration expires.
// Video data is piped to stdout and counted without writing to disk.
func (e *YoutubeExecutor) Run(ctx context.Context, task *model.Task, meter *ratelimit.Meter, progress func(int64)) error {
	if task.TargetURL == "" {
		return fmt.Errorf("target_url is required for youtube task")
	}

	args := buildYtdlpArgs(task)
	slog.Info("youtube executor", "task", task.ID, "args", args)

	endAt := computeEndTime(task, time.Now())
	loopCtx, cancel := context.WithDeadline(ctx, endAt)
	defer cancel()

	cw := &countingWriter{meter: meter, progress: progress}

	// Loop: re-download the video until the task duration expires
	for {
		if loopCtx.Err() != nil {
			return nil
		}

		err := runYtdlp(loopCtx, args, cw)
		if err != nil {
			if loopCtx.Err() != nil {
				return nil // deadline reached = normal stop
			}
			slog.Warn("yt-dlp error, retrying in 2s", "task", task.ID, "err", err)
			select {
			case <-loopCtx.Done():
				return nil
			case <-time.After(2 * time.Second):
			}
			continue
		}

		slog.Info("yt-dlp finished, restarting download", "task", task.ID, "total_bytes", atomic.LoadInt64(&cw.total))
	}
}

// runYtdlp runs a single yt-dlp process and blocks until it exits.
func runYtdlp(ctx context.Context, args []string, cw *countingWriter) error {
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = cw

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start yt-dlp: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			slog.Debug("yt-dlp", "line", scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil // context cancelled = normal stop
		}
		return fmt.Errorf("yt-dlp exited: %w", err)
	}
	return nil
}

// countingWriter counts bytes written to it, records to meter, and discards the data.
type countingWriter struct {
	total    int64
	meter    *ratelimit.Meter
	progress func(int64)
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n := len(p)
	total := atomic.AddInt64(&w.total, int64(n))
	w.meter.Record(int64(n))
	if w.progress != nil {
		w.progress(total)
	}
	return n, nil
}

func buildYtdlpArgs(task *model.Task) []string {
	args := []string{}

	// Rate limit
	if task.TargetRateMbps > 0 {
		rateBytesPerSec := int64(task.TargetRateMbps * 1e6 / 8)
		args = append(args, "--limit-rate", fmt.Sprintf("%d", rateBytesPerSec))
	}

	// Concurrent fragments
	if task.ConcurrentFragments > 1 {
		args = append(args, "--concurrent-fragments", strconv.Itoa(task.ConcurrentFragments))
	}

	// Retries
	if task.Retries > 0 {
		args = append(args, "--retries", strconv.Itoa(task.Retries))
	}

	// Output to stdout (-o -) so nothing is written to disk
	args = append(args,
		"--no-playlist",
		"--output", "-",
		task.TargetURL,
	)

	return args
}
