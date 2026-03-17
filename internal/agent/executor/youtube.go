package executor

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/pkg/ratelimit"
)

// YoutubeExecutor runs yt-dlp as a managed subprocess.
type YoutubeExecutor struct{}

const (
	youtubeWorkerTargetMbps     = 250.0
	maxYoutubeWorkers           = 8
	youtubeRetryDelay           = 2 * time.Second
	youtubeNoProgressTimeout    = 2 * time.Minute
	youtubeProgressPollInterval = 10 * time.Second
	youtubeStderrTailLines      = 8
)

var (
	youtubeJSRuntimeOnce sync.Once
	youtubeJSRuntime     string
)

// Run executes the youtube task via yt-dlp in a loop until the task duration expires.
// Video data is piped to stdout and counted without writing to disk.
func (e *YoutubeExecutor) Run(ctx context.Context, task *model.Task, meter *ratelimit.Meter, progress func(int64)) error {
	task.Normalize()
	urls := task.URLs()
	if len(urls) == 0 {
		return fmt.Errorf("target_url is required for youtube task")
	}

	endAt := computeEndTime(task, time.Now())
	loopCtx, cancel := context.WithDeadline(ctx, endAt)
	defer cancel()

	workerCount := youtubeWorkerCount(task, len(urls))
	perWorkerRate := task.TargetRateMbps
	if task.TargetRateMbps > 0 && workerCount > 1 {
		perWorkerRate = task.TargetRateMbps / float64(workerCount)
	}

	slog.Info("youtube executor starting workers",
		"task", task.ID,
		"workers", workerCount,
		"target_rate_mbps", task.TargetRateMbps,
		"per_worker_rate_mbps", perWorkerRate,
	)

	errCh := make(chan error, workerCount)
	var totalBytes int64
	errTracker := &taskErrorTracker{}
	for workerID := 0; workerID < workerCount; workerID++ {
		workerTask := task.Clone()
		workerTask.TargetRateMbps = perWorkerRate
		go func(workerID int, workerTask *model.Task) {
			errCh <- e.runWorker(loopCtx, workerTask, urls, workerID, workerCount, meter, progress, &totalBytes, errTracker)
		}(workerID, workerTask)
	}

	stallCh := make(chan error, 1)
	go func() {
		stallCh <- monitorYoutubeProgress(loopCtx, &totalBytes, youtubeNoProgressTimeout, youtubeProgressPollInterval, errTracker)
	}()

	var firstErr error
	completedWorkers := 0
	for completedWorkers < workerCount {
		select {
		case err := <-stallCh:
			stallCh = nil
			if err != nil && firstErr == nil {
				firstErr = err
				cancel()
			}
		case err := <-errCh:
			completedWorkers++
			if err != nil && firstErr == nil {
				firstErr = err
				cancel()
			}
		}
	}
	return firstErr
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

	tail := newLineTail(youtubeStderrTailLines)
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			tail.Add(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			tail.Add("stderr read error: " + err.Error())
		}
	}()

	err = cmd.Wait()
	<-stderrDone
	if err != nil {
		if ctx.Err() != nil {
			return nil // context cancelled = normal stop
		}
		if stderrSummary := tail.String(); stderrSummary != "" {
			return fmt.Errorf("yt-dlp exited: %w (%s)", err, stderrSummary)
		}
		return fmt.Errorf("yt-dlp exited: %w", err)
	}
	return nil
}

// countingWriter counts bytes written to it, records to meter, and discards the data.
type countingWriter struct {
	total    *int64
	meter    *ratelimit.Meter
	progress func(int64)
}

func newCountingWriter(total *int64, meter *ratelimit.Meter, progress func(int64)) *countingWriter {
	if total == nil {
		total = new(int64)
	}
	return &countingWriter{
		total:    total,
		meter:    meter,
		progress: progress,
	}
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n := len(p)
	total := atomic.AddInt64(w.total, int64(n))
	w.meter.Record(int64(n))
	if w.progress != nil {
		w.progress(total)
	}
	return n, nil
}

func (w *countingWriter) Total() int64 {
	if w.total == nil {
		return 0
	}
	return atomic.LoadInt64(w.total)
}

func (e *YoutubeExecutor) runWorker(
	ctx context.Context,
	task *model.Task,
	urls []string,
	workerID int,
	workerCount int,
	meter *ratelimit.Meter,
	progress func(int64),
	totalBytes *int64,
	errTracker *taskErrorTracker,
) error {
	cw := newCountingWriter(totalBytes, meter, progress)
	runIndex := workerID

	for {
		if ctx.Err() != nil {
			return nil
		}

		targetURL := urls[runIndex%len(urls)]
		args := buildYtdlpArgs(task, targetURL)
		slog.Info("youtube worker", "task", task.ID, "worker", workerID, "url", targetURL, "args", args)

		err := runYtdlp(ctx, args, cw)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errTracker != nil {
				errTracker.Set(err)
			}
			slog.Warn("yt-dlp error, retrying", "task", task.ID, "worker", workerID, "retry_delay", youtubeRetryDelay.String(), "err", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(youtubeRetryDelay):
			}
			continue
		}

		runIndex += workerCount
		slog.Info("yt-dlp finished, restarting download", "task", task.ID, "worker", workerID, "total_bytes", cw.Total())
	}
}

func youtubeWorkerCount(task *model.Task, urlCount int) int {
	if urlCount <= 1 {
		return 1
	}

	desired := 1
	if task.TargetRateMbps > 0 {
		desired = int(math.Ceil(task.TargetRateMbps / youtubeWorkerTargetMbps))
	}
	if task.ConcurrentFragments > desired {
		desired = task.ConcurrentFragments
	}
	if desired < 1 {
		desired = 1
	}
	if desired > maxYoutubeWorkers {
		desired = maxYoutubeWorkers
	}
	if desired > urlCount {
		desired = urlCount
	}
	return desired
}

func buildYtdlpArgs(task *model.Task, targetURL string) []string {
	return buildYtdlpArgsWithJSRuntime(task, targetURL, detectYoutubeJSRuntime())
}

func buildYtdlpArgsWithJSRuntime(task *model.Task, targetURL, jsRuntime string) []string {
	args := []string{}

	if jsRuntime != "" {
		args = append(args, "--js-runtimes", jsRuntime)
	}

	// Use iOS player client to reduce bot detection on headless servers
	args = append(args, "--extractor-args", "youtube:player_client=ios,web")

	// Cookies file for authenticated access (required on datacenter IPs)
	if cf := youtubeCookiesFile(); cf != "" {
		args = append(args, "--cookies", cf)
	}

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
		targetURL,
	)

	return args
}

// youtubeCookiesFile returns the path to the YouTube cookies file if it exists.
// Checks YOUTUBE_COOKIES_FILE env var first, then the default path.
func youtubeCookiesFile() string {
	if p := os.Getenv("YOUTUBE_COOKIES_FILE"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	const defaultPath = "/etc/ngoogle/youtube-cookies.txt"
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	return ""
}

func detectYoutubeJSRuntime() string {
	youtubeJSRuntimeOnce.Do(func() {
		for _, candidate := range []string{"node", "nodejs", "deno", "bun"} {
			path, err := exec.LookPath(candidate)
			if err == nil {
				youtubeJSRuntime = candidate + ":" + path
				return
			}
		}
	})
	return youtubeJSRuntime
}

func monitorYoutubeProgress(ctx context.Context, totalBytes *int64, timeout, pollInterval time.Duration, errTracker *taskErrorTracker) error {
	lastBytes := atomic.LoadInt64(totalBytes)
	lastProgressAt := time.Now()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			currentBytes := atomic.LoadInt64(totalBytes)
			if currentBytes > lastBytes {
				lastBytes = currentBytes
				lastProgressAt = time.Now()
				continue
			}
			if time.Since(lastProgressAt) < timeout {
				continue
			}

			msg := fmt.Sprintf("youtube task stalled for %s without byte progress", timeout)
			if errTracker != nil {
				if lastErr := errTracker.Last(); lastErr != "" {
					msg += ": " + lastErr
				}
			}
			return fmt.Errorf("%s", msg)
		}
	}
}

type taskErrorTracker struct {
	mu   sync.Mutex
	last string
}

func (t *taskErrorTracker) Set(err error) {
	if err == nil {
		return
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return
	}
	t.mu.Lock()
	t.last = msg
	t.mu.Unlock()
}

func (t *taskErrorTracker) Last() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last
}

type lineTail struct {
	limit int
	lines []string
}

func newLineTail(limit int) *lineTail {
	return &lineTail{limit: limit}
}

func (t *lineTail) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" || t.limit <= 0 {
		return
	}
	if len(t.lines) == t.limit {
		copy(t.lines, t.lines[1:])
		t.lines[len(t.lines)-1] = line
		return
	}
	t.lines = append(t.lines, line)
}

func (t *lineTail) String() string {
	return strings.Join(t.lines, " | ")
}
