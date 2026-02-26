package handler

import (
	"net/http"
	"time"

	"github.com/aven/ngoogle/internal/master/service"
	"github.com/aven/ngoogle/internal/model"
)

// TaskHandler handles task-related endpoints.
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Router registers all task routes.
func (h *TaskHandler) Router(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tasks", h.Create)
	mux.HandleFunc("GET /api/v1/tasks", h.List)
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/tasks/{id}/dispatch", h.Dispatch)
	mux.HandleFunc("POST /api/v1/tasks/{id}/stop", h.Stop)
	mux.HandleFunc("POST /api/v1/tasks/{id}/metrics", h.ReportMetrics)
	mux.HandleFunc("GET /api/v1/tasks/{id}/metrics", h.GetMetrics)
	mux.HandleFunc("GET /api/v1/agents/{agent_id}/tasks/pull", h.PullTasks)
}

// Create handles POST /api/v1/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTaskRequest
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	task, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, task)
}

// List handles GET /api/v1/tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, tasks)
}

// Get handles GET /api/v1/tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := h.svc.Get(r.Context(), id)
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, task)
}

// Dispatch handles POST /api/v1/tasks/{id}/dispatch
func (h *TaskHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Dispatch(r.Context(), id); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "dispatched"})
}

// Stop handles POST /api/v1/tasks/{id}/stop
func (h *TaskHandler) Stop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Stop(r.Context(), id); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// ReportMetrics handles POST /api/v1/tasks/{id}/metrics
func (h *TaskHandler) ReportMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var m model.TaskMetrics
	if err := decode(r, &m); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	m.TaskID = id
	if err := h.svc.RecordMetrics(r.Context(), &m); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetMetrics handles GET /api/v1/tasks/{id}/metrics
func (h *TaskHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	q := r.URL.Query()
	from := parseTime(q.Get("from"), time.Now().Add(-1*time.Hour))
	to := parseTime(q.Get("to"), time.Now())
	metrics, err := h.svc.GetMetrics(r.Context(), id, from, to)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, metrics)
}

// PullTasks handles GET /api/v1/agents/{agent_id}/tasks/pull
func (h *TaskHandler) PullTasks(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agent_id")
	tasks, err := h.svc.PullTasks(r.Context(), agentID)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	respond(w, http.StatusOK, tasks)
}

func parseTime(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return def
	}
	return t
}
