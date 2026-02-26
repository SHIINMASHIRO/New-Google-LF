package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aven/ngoogle/internal/master/service"
)

// DashboardHandler handles dashboard endpoints.
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Router registers all dashboard routes.
func (h *DashboardHandler) Router(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dashboard/overview", h.Overview)
	mux.HandleFunc("GET /api/v1/dashboard/bandwidth/history", h.BandwidthHistory)
}

// Overview handles GET /api/v1/dashboard/overview
func (h *DashboardHandler) Overview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Overview(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, resp)
}

// BandwidthHistory handles GET /api/v1/dashboard/bandwidth/history
func (h *DashboardHandler) BandwidthHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := parseTime(q.Get("from"), time.Now().Add(-7*24*time.Hour))
	to := parseTime(q.Get("to"), time.Now())
	stepSec := 60
	if s := q.Get("step"); s != "" {
		// parse "1m", "5m", or plain seconds
		if s == "5m" {
			stepSec = 300
		} else if s == "1m" {
			stepSec = 60
		} else if n, err := strconv.Atoi(s); err == nil {
			stepSec = n
		}
	}
	points, err := h.svc.BandwidthHistory(r.Context(), from, to, stepSec)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, points)
}
