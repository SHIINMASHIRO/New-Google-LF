// Package handler provides HTTP handlers for the Master API.
package handler

import (
	"encoding/json"
	"net/http"
)

// respond writes a JSON response.
func respond(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// respondErr writes a JSON error response.
func respondErr(w http.ResponseWriter, code int, msg string) {
	respond(w, code, map[string]string{"error": msg})
}

// decode decodes JSON request body.
func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// pathParam extracts a path segment after the given prefix.
// e.g. pathParam("/api/v1/tasks/", "/api/v1/tasks/abc123/metrics") == "abc123"
func pathParam(prefix, path string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	rest := path[len(prefix):]
	// return first segment
	for i, c := range rest {
		if c == '/' {
			return rest[:i]
		}
	}
	return rest
}
