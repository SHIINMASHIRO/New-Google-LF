package handler

import (
	"net/http"
	"time"

	"github.com/aven/ngoogle/internal/master/service"
	"github.com/aven/ngoogle/internal/model"
)

type TaskGroupHandler struct {
	svc *service.TaskGroupService
}

func NewTaskGroupHandler(svc *service.TaskGroupService) *TaskGroupHandler {
	return &TaskGroupHandler{svc: svc}
}

func (h *TaskGroupHandler) Router(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/task-groups", h.Create)
	mux.HandleFunc("GET /api/v1/task-groups", h.List)
	mux.HandleFunc("GET /api/v1/task-groups/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/task-groups/{id}/dispatch", h.Dispatch)
	mux.HandleFunc("POST /api/v1/task-groups/{id}/stop", h.Stop)
	mux.HandleFunc("GET /api/v1/task-groups/{id}/metrics", h.GetMetrics)
}

func (h *TaskGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTaskGroupRequest
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	group, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, group)
}

func (h *TaskGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.svc.List(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if groups == nil {
		groups = []*model.TaskGroup{}
	}
	respond(w, http.StatusOK, groups)
}

func (h *TaskGroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	group, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, group)
}

func (h *TaskGroupHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Dispatch(r.Context(), r.PathValue("id")); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "dispatched"})
}

func (h *TaskGroupHandler) Stop(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Stop(r.Context(), r.PathValue("id")); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *TaskGroupHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := parseTime(q.Get("from"), time.Now().Add(-1*time.Hour))
	to := parseTime(q.Get("to"), time.Now())
	metrics, err := h.svc.GetMetrics(r.Context(), r.PathValue("id"), from, to)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if metrics == nil {
		metrics = []*model.TaskMetrics{}
	}
	respond(w, http.StatusOK, metrics)
}
