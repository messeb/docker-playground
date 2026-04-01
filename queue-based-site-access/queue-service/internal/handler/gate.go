package handler

import (
	"net/http"
	"strconv"
	"time"
)

// rootHandler is the entry point for every request.
// Admitted sessions are proxied to the target service; everyone else is queued.
//
// It accepts all paths (not just "/") so that the target page's static assets
// (e.g. /styles.css) are proxied correctly for admitted sessions.
func (h *Handler) rootHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := h.getOrCreateSession(w, r)

	// ── Active session: proxy immediately ────────────────────────────────────
	if h.store.GetStatus(ctx, sid) == "active" {
		if h.store.IsInActiveSet(ctx, sid) {
			h.proxy.ServeHTTP(w, r)
			return
		}
		// Cookie exists but the session was evicted server-side — start fresh.
		h.store.DeleteSession(ctx, sid)
	}

	// ── Sub-path with no active session ──────────────────────────────────────
	// Redirect asset requests (e.g. /styles.css) to root so the user goes
	// through the normal queue flow rather than receiving a 404.
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// ── Try to claim a slot directly ─────────────────────────────────────────
	now := strconv.FormatInt(time.Now().Unix(), 10)
	if h.store.TryAdmit(ctx, sid, now) {
		h.store.SetHeartbeat(ctx, sid)
		h.proxy.ServeHTTP(w, r)
		return
	}

	// ── No slot available — place in queue ───────────────────────────────────
	h.store.JoinQueue(ctx, sid, now)
	h.store.SetHeartbeat(ctx, sid)
	http.Redirect(w, r, "/queue", http.StatusFound)
}

// queuePageHandler serves the waiting room HTML.
// Already-admitted sessions are redirected to the target page immediately.
func (h *Handler) queuePageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := sessionID(r)
	if sid == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if h.store.GetStatus(ctx, sid) == "active" && h.store.IsInActiveSet(ctx, sid) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Refresh heartbeat so the TTL starts clean on every page load.
	h.store.SetHeartbeat(ctx, sid)
	http.ServeFile(w, r, h.cfg.WebDir+"/queue.html")
}

// doneHandler serves a confirmation page shown after a user voluntarily leaves
// the protected area via the "Done" button. It sits outside the queue gate so
// no session check or slot claim happens here. A "Re-enter" link on the page
// sends the user back to / for the normal queue-or-admit flow.
func (h *Handler) doneHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, h.cfg.WebDir+"/done.html")
}

// positionHandler returns the caller's current place in the queue as JSON.
// The queue page polls this endpoint every 2 seconds.
func (h *Handler) positionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sid := sessionID(r)
	if sid == "" {
		writeJSON(w, http.StatusOK, map[string]any{"status": "no_session"})
		return
	}

	if h.store.GetStatus(ctx, sid) == "active" {
		if h.store.IsInActiveSet(ctx, sid) {
			writeJSON(w, http.StatusOK, map[string]any{"status": "active", "redirect": "/"})
			return
		}
		h.store.DeleteSession(ctx, sid)
	}

	pos, found := h.store.GetPosition(ctx, sid)
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"status": "unknown"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "queued",
		"position":    pos.Pos,
		"total":       pos.Total,
		"eta_seconds": h.store.CalculateETA(ctx, pos.Pos),
	})
}

// healthHandler reports the service and Redis health.
func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"redis_unavailable"}`))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
