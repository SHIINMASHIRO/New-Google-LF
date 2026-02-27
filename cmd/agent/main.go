// Command agent is the task execution worker for the Google traffic task system.
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aven/ngoogle/internal/agent/client"
	"github.com/aven/ngoogle/internal/agent/executor"
	"github.com/aven/ngoogle/internal/agent/reporter"
	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/pkg/ratelimit"
)

const agentVersion = "1.0.0"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	masterURL := envOr("MASTER_URL", "http://localhost:8080")
	hostIP := envOr("AGENT_HOST_IP", detectIP())
	agentPort := 0 // agents don't expose a public port

	slog.Info("agent starting", "master", masterURL, "ip", hostIP)

	mc := client.New(masterURL)

	// ─── Register with retry ─────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hostname, _ := os.Hostname()
	var regResp *client.RegisterResponse
	for {
		var err error
		regResp, err = mc.Register(ctx, hostname, hostIP, agentPort, agentVersion)
		if err == nil {
			slog.Info("registered", "agent_id", regResp.ID)
			break
		}
		slog.Error("register failed, retrying in 5s", "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		slog.Info("agent shutting down...")
		cancel()
	}()

	// ─── Task runner ──────────────────────────────────────────────────────────
	runner := &taskRunner{
		client:  mc,
		agentID: regResp.ID,
		running: make(map[string]context.CancelFunc),
	}

	// ─── Main loop: heartbeat + task pull ────────────────────────────────────
	heartbeatTicker := time.NewTicker(10 * time.Second)
	pullTicker := time.NewTicker(5 * time.Second)
	defer heartbeatTicker.Stop()
	defer pullTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			runner.stopAll()
			return

		case <-heartbeatTicker.C:
			if err := mc.Heartbeat(ctx, runner.totalRate()); err != nil {
				slog.Warn("heartbeat failed", "err", err)
			}

		case <-pullTicker.C:
			runner.pull(ctx)
		}
	}
}

// ─── Task Runner ──────────────────────────────────────────────────────────────

type taskRunner struct {
	client  *client.Client
	agentID string

	mu      sync.Mutex
	running map[string]context.CancelFunc
	meters  map[string]*ratelimit.Meter
}

func (r *taskRunner) pull(ctx context.Context) {
	tasks, err := r.client.PullTasks(ctx)
	if err != nil {
		slog.Warn("pull tasks failed", "err", err)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, task := range tasks {
		if _, ok := r.running[task.ID]; ok {
			continue // already running
		}
		taskCtx, cancel := context.WithCancel(ctx)
		r.running[task.ID] = cancel
		if r.meters == nil {
			r.meters = make(map[string]*ratelimit.Meter)
		}
		meter := &ratelimit.Meter{}
		r.meters[task.ID] = meter
		go r.execute(taskCtx, task, meter, cancel)
	}

	// Check for tasks that should be stopped
	for taskID, cancel := range r.running {
		found := false
		for _, t := range tasks {
			if t.ID == taskID {
				found = true
				break
			}
		}
		if !found {
			slog.Info("task no longer assigned, stopping", "task", taskID)
			cancel()
			delete(r.running, taskID)
			delete(r.meters, taskID)
		}
	}
}

func (r *taskRunner) execute(ctx context.Context, task *model.Task, meter *ratelimit.Meter, cancel context.CancelFunc) {
	defer func() {
		cancel()
		r.mu.Lock()
		delete(r.running, task.ID)
		if r.meters != nil {
			delete(r.meters, task.ID)
		}
		r.mu.Unlock()
	}()

	slog.Info("executing task", "task", task.ID, "type", task.Type, "url", task.TargetURL)

	rep := reporter.NewTaskReporter(task.ID, r.agentID, r.client, meter)
	go rep.Run(ctx)

	progressFn := func(bytesTotal int64) {
		// metrics are handled by reporter
	}

	var err error
	switch task.Type {
	case model.TaskTypeYoutube:
		exe := &executor.YoutubeExecutor{}
		err = exe.Run(ctx, task, rep.Meter(), progressFn)
	case model.TaskTypeStatic:
		exe := &executor.StaticExecutor{}
		err = exe.Run(ctx, task, rep.Meter(), progressFn)
	default:
		slog.Error("unknown task type", "type", task.Type)
		return
	}

	if err != nil {
		slog.Error("task failed", "task", task.ID, "err", err)
		// Report failure
		_ = r.client.ReportMetrics(context.Background(), &model.TaskMetrics{
			TaskID:  task.ID,
			AgentID: r.agentID,
		})
	} else {
		slog.Info("task completed", "task", task.ID)
	}
}

func (r *taskRunner) totalRate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	var total float64
	for _, m := range r.meters {
		total += m.Rate5s()
	}
	return total
}

func (r *taskRunner) stopAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, cancel := range r.running {
		cancel()
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func detectIP() string {
	// Try to find the non-loopback IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
