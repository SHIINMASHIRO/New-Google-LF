// Package executor provides task executors for the Agent.
package executor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aven/ngoogle/internal/master/scheduler"
	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/pkg/ratelimit"
)

// StaticResult holds the result of a static download.
type StaticResult struct {
	BytesDownloaded int64
	Duration        time.Duration
	Err             error
}

// StaticExecutor downloads a static HTTP resource with rate limiting.
type StaticExecutor struct{}

// Run downloads the target URL respecting the rate limit and context.
func (e *StaticExecutor) Run(ctx context.Context, task *model.Task, meter *ratelimit.Meter, progress func(int64)) error {
	task.Normalize()
	urls := task.URLs()
	if len(urls) == 0 {
		return fmt.Errorf("target_url is required for static task")
	}

	workers := task.ConcurrentFragments
	if workers <= 1 {
		workers = 8
	}

	tb := ratelimit.New(task.TargetRateMbps, 2.0)

	startedAt := time.Now()
	endAt := computeEndTime(task, startedAt)

	reqCtx, cancel := context.WithDeadline(ctx, endAt)
	defer cancel()

	// Apply jitter to first request
	if task.JitterPct > 0 {
		jitterWait := scheduler.ApplyJitter(100*time.Millisecond, task.JitterPct)
		select {
		case <-reqCtx.Done():
			return nil
		case <-time.After(jitterWait):
		}
	}

	// Rate adjustment goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-reqCtx.Done():
				return
			case <-ticker.C:
				var elapsed time.Duration
				if task.StartedAt != nil {
					elapsed = time.Since(*task.StartedAt)
				} else {
					elapsed = time.Since(startedAt)
				}
				mult := scheduler.RateForTask(task, elapsed, nil)
				tb.SetRate(task.TargetRateMbps * mult)
			}
		}
	}()

	var totalBytes atomic.Int64
	var reqCount atomic.Int64

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-reqCtx.Done():
					return
				default:
				}

				// Check volume targets
				if task.TotalBytesTarget > 0 && totalBytes.Load() >= task.TotalBytesTarget {
					return
				}
				if task.TotalRequestsTarget > 0 && reqCount.Load() >= task.TotalRequestsTarget {
					return
				}

				idx := reqCount.Add(1) - 1
				targetURL := urls[int(idx)%len(urls)]
				n, err := downloadOnce(reqCtx, targetURL, tb)
				if err != nil {
					if reqCtx.Err() != nil {
						return
					}
					slog.Warn("static download err, retrying", "worker", workerID, "err", err)
					select {
					case <-reqCtx.Done():
						return
					case <-time.After(2 * time.Second):
					}
					continue
				}

				newTotal := totalBytes.Add(n)
				meter.Record(n)
				if progress != nil {
					progress(newTotal)
				}

				// Apply inter-request jitter
				if task.DispatchRateTpm > 0 {
					interval := scheduler.DispatchInterval(task.DispatchRateTpm, task.DispatchBatchSize)
					interval = scheduler.ApplyJitter(interval, task.JitterPct)
					select {
					case <-reqCtx.Done():
						return
					case <-time.After(interval):
					}
				}
			}
		}(w)
	}

	wg.Wait()
	return nil
}

func downloadOnce(ctx context.Context, url string, tb *ratelimit.TokenBucket) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ngoogle-agent/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Read with rate limiting
	buf := make([]byte, 64*1024) // 64 KB chunks
	var total int64
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			// Wait for token bucket before consuming
			if waitErr := tb.Wait(ctx, int64(n)); waitErr != nil {
				return total, nil // context cancelled
			}
			total += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func computeEndTime(task *model.Task, startedAt time.Time) time.Time {
	if task.EndAt != nil {
		return *task.EndAt
	}
	if task.DurationSec > 0 {
		return startedAt.Add(time.Duration(task.DurationSec) * time.Second)
	}
	// Default: run for 1 hour max
	return startedAt.Add(1 * time.Hour)
}
