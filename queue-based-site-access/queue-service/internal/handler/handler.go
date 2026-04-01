package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"

	"github.com/google/uuid"
	"github.com/messeb/docker-playground/queue-based-site-access/internal/config"
	"github.com/messeb/docker-playground/queue-based-site-access/internal/queue"
)

const cookieName = "qsid"

// Handler holds the shared dependencies for all HTTP handlers.
type Handler struct {
	store  *queue.Store
	cfg    config.Config
	proxy  *httputil.ReverseProxy
}

// New creates a Handler wired to the given store, config, and proxy.
func New(store *queue.Store, cfg config.Config, proxy *httputil.ReverseProxy) *Handler {
	return &Handler{store: store, cfg: cfg, proxy: proxy}
}

// Routes registers all HTTP handlers and returns the configured mux.
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", h.healthHandler)

	// Queue UI + position polling
	mux.HandleFunc("GET /queue", h.queuePageHandler)
	mux.HandleFunc("GET /api/queue/position", h.positionHandler)

	// Post-session confirmation page (shown after user clicks "Done")
	mux.HandleFunc("GET /done", h.doneHandler)

	// Session lifecycle (called by JS on the queue and target pages)
	mux.HandleFunc("POST /api/session/heartbeat", h.sessionHeartbeatHandler)
	mux.HandleFunc("POST /api/session/leave", h.sessionLeaveHandler)

	// Admin
	mux.HandleFunc("POST /api/admin/capacity", h.adminCapacityHandler)
	mux.HandleFunc("POST /api/admin/release", h.adminReleaseHandler)
	mux.HandleFunc("GET /api/admin/status", h.adminStatusHandler)

	// Static assets for the queue page
	mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(h.cfg.WebDir+"/static"))))

	// Catch-all: gate all requests through the queue
	mux.HandleFunc("/", h.rootHandler)

	return mux
}

// ── Shared helpers ────────────────────────────────────────────────────────────

// getOrCreateSession returns the session ID from the qsid cookie, creating and
// setting a new one if the request doesn't carry one yet.
func (h *Handler) getOrCreateSession(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	id := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return id
}

// sessionID reads the session ID from the cookie without creating one.
// Returns "" if the cookie is absent.
func sessionID(r *http.Request) string {
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
