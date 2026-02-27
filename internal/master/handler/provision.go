package handler

import (
	"net/http"

	"github.com/aven/ngoogle/internal/master/provision"
)

// ProvisionHandler handles agent provisioning endpoints.
type ProvisionHandler struct {
	svc *provision.Service
}

// NewProvisionHandler creates a new ProvisionHandler.
func NewProvisionHandler(svc *provision.Service) *ProvisionHandler {
	return &ProvisionHandler{svc: svc}
}

// Router registers all provision routes.
func (h *ProvisionHandler) Router(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/agents/provision", h.StartProvision)
	mux.HandleFunc("GET /api/v1/agents/provision-jobs", h.ListJobs)
	mux.HandleFunc("GET /api/v1/agents/provision-jobs/{job_id}", h.GetJob)
	mux.HandleFunc("POST /api/v1/agents/provision-jobs/{job_id}/retry", h.RetryJob)
	mux.HandleFunc("DELETE /api/v1/agents/provision-jobs/{job_id}", h.DeleteJob)
	mux.HandleFunc("POST /api/v1/credentials", h.CreateCredential)
	mux.HandleFunc("GET /api/v1/credentials", h.ListCredentials)
	mux.HandleFunc("DELETE /api/v1/credentials/{id}", h.DeleteCredential)
}

// StartProvision handles POST /api/v1/agents/provision
func (h *ProvisionHandler) StartProvision(w http.ResponseWriter, r *http.Request) {
	var req provision.JobRequest
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	job, err := h.svc.Start(r.Context(), &req)
	if err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, job)
}

// ListJobs handles GET /api/v1/agents/provision-jobs
func (h *ProvisionHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.svc.ListJobs(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, jobs)
}

// GetJob handles GET /api/v1/agents/provision-jobs/{job_id}
func (h *ProvisionHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("job_id")
	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, job)
}

// RetryJob handles POST /api/v1/agents/provision-jobs/{job_id}/retry
func (h *ProvisionHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("job_id")
	job, err := h.svc.Retry(r.Context(), id)
	if err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, job)
}

// DeleteJob handles DELETE /api/v1/agents/provision-jobs/{job_id}
func (h *ProvisionHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("job_id")
	if err := h.svc.DeleteJob(r.Context(), id); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// DeleteCredential handles DELETE /api/v1/credentials/{id}
func (h *ProvisionHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteCredential(r.Context(), id); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// CreateCredential handles POST /api/v1/credentials
func (h *ProvisionHandler) CreateCredential(w http.ResponseWriter, r *http.Request) {
	var req provision.CredentialRequest
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	cred, err := h.svc.CreateCredential(r.Context(), &req)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Do not return payload in response
	cred.Payload = ""
	respond(w, http.StatusCreated, cred)
}

// ListCredentials handles GET /api/v1/credentials
func (h *ProvisionHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	creds, err := h.svc.ListCredentials(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Scrub payloads
	for _, c := range creds {
		c.Payload = ""
	}
	respond(w, http.StatusOK, creds)
}
