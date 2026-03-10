package executor

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aven/ngoogle/internal/master/scheduler"
	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/pkg/ratelimit"
)

// MixedExecutor rotates across a mixed pool of YouTube and static URLs.
type MixedExecutor struct{}

func (e *MixedExecutor) Run(ctx context.Context, task *model.Task, meter *ratelimit.Meter, progress func(int64)) error {
	task.Normalize()
	urls := task.URLs()
	if len(urls) == 0 {
		return fmt.Errorf("target_urls is required for mixed task")
	}

	tb := ratelimit.New(task.TargetRateMbps, 2.0)
	startedAt := time.Now()
	endAt := computeEndTime(task, startedAt)
	reqCtx, cancel := context.WithDeadline(ctx, endAt)
	defer cancel()

	if task.JitterPct > 0 {
		jitterWait := scheduler.ApplyJitter(100*time.Millisecond, task.JitterPct)
		select {
		case <-reqCtx.Done():
			return nil
		case <-time.After(jitterWait):
		}
	}

	var totalBytes int64
	reqCount := int64(0)

	for {
		select {
		case <-reqCtx.Done():
			return nil
		default:
		}

		if task.TotalBytesTarget > 0 && totalBytes >= task.TotalBytesTarget {
			return nil
		}
		if task.TotalRequestsTarget > 0 && reqCount >= task.TotalRequestsTarget {
			return nil
		}

		var elapsed time.Duration
		if task.StartedAt != nil {
			elapsed = time.Since(*task.StartedAt)
		} else {
			elapsed = time.Since(startedAt)
		}
		mult := scheduler.RateForTask(task, elapsed, nil)
		tb.SetRate(task.TargetRateMbps * mult)

		targetURL := urls[int(reqCount)%len(urls)]
		if isYoutubeURL(targetURL) {
			sharedTotal := totalBytes
			cw := newCountingWriter(&sharedTotal, meter, progress)
			child := task.Clone()
			child.Type = model.TaskTypeYoutube
			child.TargetURL = targetURL
			if err := runYtdlp(reqCtx, buildYtdlpArgs(child, targetURL), cw); err != nil {
				if reqCtx.Err() != nil {
					return nil
				}
				select {
				case <-reqCtx.Done():
					return nil
				case <-time.After(2 * time.Second):
				}
				continue
			}
			totalBytes = cw.Total()
		} else {
			n, err := downloadOnce(reqCtx, targetURL, tb)
			if err != nil {
				if reqCtx.Err() != nil {
					return nil
				}
				select {
				case <-reqCtx.Done():
					return nil
				case <-time.After(2 * time.Second):
				}
				continue
			}
			totalBytes += n
			meter.Record(n)
			if progress != nil {
				progress(totalBytes)
			}
		}

		reqCount++
		if task.DispatchRateTpm > 0 {
			interval := scheduler.DispatchInterval(task.DispatchRateTpm, task.DispatchBatchSize)
			interval = scheduler.ApplyJitter(interval, task.JitterPct)
			select {
			case <-reqCtx.Done():
				return nil
			case <-time.After(interval):
			}
		}
	}
}

func isYoutubeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be")
}
