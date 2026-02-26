package handler

import (
	"net/http"

	"github.com/aven/ngoogle/internal/master/service"
)

// AgentHandler handles agent-related endpoints.
type AgentHandler struct {
	svc *service.AgentService
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(svc *service.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

// Register handles POST /api/v1/agents/register
func (h *AgentHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hostname string `json:"hostname"`
		IP       string `json:"ip"`
		Port     int    `json:"port"`
		Version  string `json:"version"`
	}
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	agent, err := h.svc.Register(r.Context(), req.Hostname, req.IP, req.Port, req.Version)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, agent)
}

// Heartbeat handles POST /api/v1/agents/heartbeat
func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID  string  `json:"agent_id"`
		Token    string  `json:"token"`
		RateMbps float64 `json:"rate_mbps"`
	}
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ValidateToken(r.Context(), req.AgentID, req.Token); err != nil {
		respondErr(w, http.StatusUnauthorized, "invalid token")
		return
	}
	if err := h.svc.Heartbeat(r.Context(), req.AgentID, req.RateMbps); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

// List handles GET /api/v1/agents
func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	agents, err := h.svc.List(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, agents)
}

// Router registers all agent routes.
func (h *AgentHandler) Router(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/agents/register", h.Register)
	mux.HandleFunc("POST /api/v1/agents/heartbeat", h.Heartbeat)
	mux.HandleFunc("GET /api/v1/agents", h.List)
	mux.HandleFunc("GET /api/v1/agents/{id}", h.agentByID)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", h.deleteAgent)
}

func (h *AgentHandler) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondErr(w, http.StatusBadRequest, "missing id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AgentHandler) agentByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondErr(w, http.StatusBadRequest, "missing id")
		return
	}
	agent, err := h.svc.Get(r.Context(), id)
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, agent)
}
