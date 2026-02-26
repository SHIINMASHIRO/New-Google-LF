// Command master is the central control plane for the Google traffic task system.
package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aven/ngoogle/internal/master/handler"
	"github.com/aven/ngoogle/internal/master/provision"
	"github.com/aven/ngoogle/internal/master/scheduler"
	"github.com/aven/ngoogle/internal/master/service"
	"github.com/aven/ngoogle/internal/store/sqlite"
	ngweb "github.com/aven/ngoogle/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// ─── Config from env ──────────────────────────────────────────────────────
	addr := envOr("MASTER_ADDR", ":8080")
	dsn := envOr("SQLITE_DSN", "file:master.db?cache=shared&_fk=on")
	masterURL := envOr("MASTER_URL", "http://localhost:8080")
	agentBin := envOr("AGENT_BIN_PATH", "")

	// ─── Store ────────────────────────────────────────────────────────────────
	st, err := sqlite.New(dsn)
	if err != nil {
		slog.Error("open store", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// ─── Services ─────────────────────────────────────────────────────────────
	agentSvc := service.NewAgentService(st)
	taskSvc := service.NewTaskService(st)
	dashSvc := service.NewDashboardService(st)
	provSvc := provision.NewService(st, masterURL, agentBin)
	sched := scheduler.New(st)

	// ─── Handlers ─────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	handler.NewAgentHandler(agentSvc).Router(mux)
	handler.NewTaskHandler(taskSvc).Router(mux)
	handler.NewDashboardHandler(dashSvc).Router(mux)
	handler.NewProvisionHandler(provSvc).Router(mux)
	handler.NewProfileHandler(st).Router(mux)

	// ─── Health + Metrics ─────────────────────────────────────────────────────
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		agents, _ := st.Agents().List(r.Context())
		tasks, _ := st.Tasks().List(r.Context())
		online := 0
		for _, a := range agents {
			if a.Status == "online" {
				online++
			}
		}
		running := 0
		for _, t := range tasks {
			if t.Status == "running" {
				running++
			}
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("# HELP ngoogle_agents_online Number of online agents\n"))
		_, _ = w.Write([]byte("# TYPE ngoogle_agents_online gauge\n"))
		_, _ = w.Write([]byte("ngoogle_agents_online " + itoa(online) + "\n"))
		_, _ = w.Write([]byte("# HELP ngoogle_tasks_running Number of running tasks\n"))
		_, _ = w.Write([]byte("# TYPE ngoogle_tasks_running gauge\n"))
		_, _ = w.Write([]byte("ngoogle_tasks_running " + itoa(running) + "\n"))
	})

	// ─── Web UI (embedded) ────────────────────────────────────────────────────
	webFS, err := fs.Sub(ngweb.Assets, "dist")
	if err != nil {
		slog.Warn("web assets not available", "err", err)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Web UI not built. Run: cd web && npm install && npm run build", http.StatusServiceUnavailable)
		})
	} else {
		fileServer := http.FileServer(http.FS(webFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// SPA fallback: serve index.html for non-asset routes
			if r.URL.Path != "/" {
				if _, ferr := webFS.Open(r.URL.Path[1:]); ferr != nil {
					r.URL.Path = "/"
				}
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	// ─── HTTP server ──────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ─── Background goroutines ─────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sched.Run(ctx)
	go agentSvc.RunOfflineDetection(ctx)
	go dashSvc.RunPurge(ctx)

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		slog.Info("shutting down...")
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			slog.Error("shutdown", "err", err)
		}
	}()

	slog.Info("master listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
