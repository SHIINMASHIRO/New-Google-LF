package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/pkg/ratelimit"
)

// YoutubeExecutor runs yt-dlp as a managed subprocess.
type YoutubeExecutor struct{}

// progressRe matches yt-dlp progress lines:
// [download]  45.3% of 12.34MiB at  5.23MiB/s ETA 00:12
var progressRe = regexp.MustCompile(`(\d+\.?\d*)%.*?at\s+([\d.]+)([\w/]+)`)

// Run executes the youtube task via yt-dlp.
func (e *YoutubeExecutor) Run(ctx context.Context, task *model.Task, meter *ratelimit.Meter, progress func(int64)) error {
	if task.TargetURL == "" {
		return fmt.Errorf("target_url is required for youtube task")
	}

	// Build yt-dlp args
	args := buildYtdlpArgs(task)
	slog.Info("youtube executor", "task", task.ID, "args", args)

	endAt := computeEndTime(task, time.Now())
	cmdCtx, cancel := context.WithDeadline(ctx, endAt)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "yt-dlp", args...)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start yt-dlp: %w", err)
	}

	// Parse progress from stdout/stderr
	var totalBytes int64
	go func() {
		combined := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(combined)
		for scanner.Scan() {
			line := scanner.Text()
			slog.Debug("yt-dlp", "line", line)

			// Parse rate from progress line
			if bytes, rateMbps := parseProgress(line); bytes > 0 || rateMbps > 0 {
				if bytes > 0 {
					totalBytes = bytes
					meter.Record(bytes)
					if progress != nil {
						progress(totalBytes)
					}
				}
				if rateMbps > 0 {
					meter.Record(int64(rateMbps * 1e6 / 8))
				}
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		if cmdCtx.Err() != nil {
			return nil // context cancelled = normal stop
		}
		return fmt.Errorf("yt-dlp exited: %w", err)
	}
	return nil
}

func buildYtdlpArgs(task *model.Task) []string {
	args := []string{}

	// Rate limit
	if task.TargetRateMbps > 0 {
		// Convert Mbps to bytes/s for yt-dlp
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

	// Output to temp directory (we don't actually keep the files)
	tmpDir := os.TempDir()
	args = append(args,
		"--no-playlist",
		"--output", fmt.Sprintf("%s/ngoogle-yt-%%(id)s.%%(ext)s", tmpDir),
		"--progress",
		"--newline",
		task.TargetURL,
	)

	return args
}

// parseProgress extracts bytes downloaded and rate from a yt-dlp line.
// Returns (totalBytes, rateMbps).
func parseProgress(line string) (int64, float64) {
	if !strings.Contains(line, "[download]") {
		return 0, 0
	}

	var totalBytes int64
	var rateMbps float64

	// Extract percentage and rate
	matches := progressRe.FindStringSubmatch(line)
	if len(matches) >= 4 {
		rateVal, _ := strconv.ParseFloat(matches[2], 64)
		rateUnit := matches[3]

		switch {
		case strings.Contains(rateUnit, "GiB/s"):
			rateMbps = rateVal * 1024 * 8
		case strings.Contains(rateUnit, "MiB/s"):
			rateMbps = rateVal * 8
		case strings.Contains(rateUnit, "KiB/s"):
			rateMbps = rateVal * 8 / 1024
		case strings.Contains(rateUnit, "B/s"):
			rateMbps = rateVal * 8 / 1e6
		}
	}

	// Try to extract file size from "of X.XXMiB"
	if idx := strings.Index(line, "of "); idx >= 0 {
		rest := line[idx+3:]
		spaceIdx := strings.Index(rest, " ")
		if spaceIdx > 0 {
			sizeStr := rest[:spaceIdx]
			size, unit := parseSizeStr(sizeStr)
			pctStr := ""
			if m := progressRe.FindStringSubmatch(line); len(m) >= 2 {
				pctStr = m[1]
			}
			pct, _ := strconv.ParseFloat(pctStr, 64)
			if size > 0 && pct > 0 {
				totalBytes = int64(size * unitMultiplier(unit) * pct / 100)
			}
		}
	}

	return totalBytes, rateMbps
}

func parseSizeStr(s string) (float64, string) {
	for _, unit := range []string{"GiB", "MiB", "KiB", "B"} {
		if strings.HasSuffix(s, unit) {
			val, err := strconv.ParseFloat(s[:len(s)-len(unit)], 64)
			if err == nil {
				return val, unit
			}
		}
	}
	return 0, ""
}

func unitMultiplier(unit string) float64 {
	switch unit {
	case "GiB":
		return 1024 * 1024 * 1024
	case "MiB":
		return 1024 * 1024
	case "KiB":
		return 1024
	default:
		return 1
	}
}
