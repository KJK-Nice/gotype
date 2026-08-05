package lnauth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Handler serves LUD-04 callback and status endpoints.
type Handler struct {
	Svc *Service
}

// NewHandler returns HTTP handlers for LNURL-auth.
func NewHandler(svc *Service) *Handler {
	return &Handler{Svc: svc}
}

// Mount registers routes on mux.
func (h *Handler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/auth/lnurl", h.handleLNURL)
	mux.HandleFunc("/auth/lnurl/status", h.handleStatus)
}

func (h *Handler) handleLNURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	k1 := strings.TrimSpace(q.Get("k1"))
	sig := strings.TrimSpace(q.Get("sig"))
	key := strings.TrimSpace(q.Get("key"))

	if sig == "" && key == "" {
		// Wallet may probe; respond with tag metadata when only k1 is present.
		if k1 == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"status": "ERROR", "reason": "missing k1"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "OK",
			"tag":    "login",
			"k1":     k1,
		})
		return
	}

	ip := clientIP(r)
	if err := h.Svc.HandleCallback(k1, sig, key, ip, time.Now()); err != nil {
		reason := err.Error()
		if err == ErrBadChallenge || err == ErrNotFound || err == ErrUsed {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ERROR", "reason": reason})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ERROR", "reason": reason})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	k1 := strings.TrimSpace(r.URL.Query().Get("k1"))
	if k1 == "" {
		http.Error(w, "missing k1", http.StatusBadRequest)
		return
	}
	st, err := h.Svc.Status(k1)
	if err == ErrNotFound {
		writeJSON(w, http.StatusNotFound, map[string]string{"state": "missing"})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		return host[:i]
	}
	return host
}
