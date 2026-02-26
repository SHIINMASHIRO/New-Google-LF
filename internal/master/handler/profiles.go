package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

// ProfileHandler handles traffic profile endpoints.
type ProfileHandler struct {
	store store.Store
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(st store.Store) *ProfileHandler {
	return &ProfileHandler{store: st}
}

// Router registers all profile routes.
func (h *ProfileHandler) Router(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/traffic-profiles", h.Create)
	mux.HandleFunc("GET /api/v1/traffic-profiles", h.List)
}

// Create handles POST /api/v1/traffic-profiles
func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string             `json:"name"`
		Description  string             `json:"description"`
		Distribution model.Distribution `json:"distribution"`
		Points       string             `json:"points"`
	}
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	p := &model.TrafficProfile{
		ID:           newID(),
		Name:         req.Name,
		Description:  req.Description,
		Distribution: req.Distribution,
		Points:       req.Points,
		CreatedAt:    time.Now(),
	}
	if p.Points == "" {
		p.Points = "[]"
	}
	if err := h.store.TrafficProfiles().Create(r.Context(), p); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, p)
}

// List handles GET /api/v1/traffic-profiles
func (h *ProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.store.TrafficProfiles().List(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, profiles)
}

// newID generates a random hex ID.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
