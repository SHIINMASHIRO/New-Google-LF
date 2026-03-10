package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

type URLPoolHandler struct {
	store store.Store
}

func NewURLPoolHandler(st store.Store) *URLPoolHandler {
	return &URLPoolHandler{store: st}
}

func (h *URLPoolHandler) Router(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/url-pools", h.Create)
	mux.HandleFunc("GET /api/v1/url-pools", h.List)
	mux.HandleFunc("GET /api/v1/url-pools/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/url-pools/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/url-pools/{id}", h.Delete)
}

func (h *URLPoolHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string            `json:"name"`
		Type        model.URLPoolType `json:"type"`
		Description string            `json:"description"`
		URLs        []string          `json:"urls"`
	}
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		respondErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type != model.URLPoolTypeYoutube && req.Type != model.URLPoolTypeStatic {
		respondErr(w, http.StatusBadRequest, "invalid pool type")
		return
	}
	p := &model.URLPool{
		ID:          newURLPoolID(),
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	p.SetURLs(req.URLs)
	if len(p.URLs) == 0 {
		respondErr(w, http.StatusBadRequest, "urls is required")
		return
	}
	if err := validatePoolURLs(p.Type, p.URLs); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.URLPools().Create(r.Context(), p); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, p)
}

func (h *URLPoolHandler) List(w http.ResponseWriter, r *http.Request) {
	pools, err := h.store.URLPools().List(r.Context())
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, pools)
}

func (h *URLPoolHandler) Get(w http.ResponseWriter, r *http.Request) {
	pool, err := h.store.URLPools().Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}
	respond(w, http.StatusOK, pool)
}

func (h *URLPoolHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := h.store.URLPools().Get(r.Context(), id)
	if err != nil {
		respondErr(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		Name        string            `json:"name"`
		Type        model.URLPoolType `json:"type"`
		Description string            `json:"description"`
		URLs        []string          `json:"urls"`
	}
	if err := decode(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name == "" {
		respondErr(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type != model.URLPoolTypeYoutube && req.Type != model.URLPoolTypeStatic {
		respondErr(w, http.StatusBadRequest, "invalid pool type")
		return
	}

	referenced, err := h.isPoolReferenced(r.Context(), id)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if referenced && req.Type != existing.Type {
		respondErr(w, http.StatusBadRequest, "cannot change pool type while it is referenced by tasks")
		return
	}

	existing.Name = req.Name
	existing.Type = req.Type
	existing.Description = req.Description
	existing.SetURLs(req.URLs)
	existing.UpdatedAt = time.Now()
	if len(existing.URLs) == 0 {
		respondErr(w, http.StatusBadRequest, "urls is required")
		return
	}
	if err := validatePoolURLs(existing.Type, existing.URLs); err != nil {
		respondErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.URLPools().Update(r.Context(), existing); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, existing)
}

func (h *URLPoolHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	referenced, err := h.isPoolReferenced(r.Context(), id)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if referenced {
		respondErr(w, http.StatusBadRequest, "url pool is still referenced by tasks")
		return
	}
	if err := h.store.URLPools().Delete(r.Context(), id); err != nil {
		respondErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func newURLPoolID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func validatePoolURLs(poolType model.URLPoolType, urls []string) error {
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid url: %s", raw)
		}
		if poolType == model.URLPoolTypeYoutube && !isYoutubeURL(raw) {
			return fmt.Errorf("youtube pool contains non-youtube url: %s", raw)
		}
		if poolType == model.URLPoolTypeStatic && isYoutubeURL(raw) {
			return fmt.Errorf("static pool contains youtube url: %s", raw)
		}
	}
	return nil
}

func isYoutubeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be")
}

func (h *URLPoolHandler) isPoolReferenced(ctx context.Context, id string) (bool, error) {
	tasks, err := h.store.Tasks().List(ctx)
	if err != nil {
		return false, err
	}
	for _, task := range tasks {
		if task.URLPoolID == id {
			return true, nil
		}
	}
	return false, nil
}
