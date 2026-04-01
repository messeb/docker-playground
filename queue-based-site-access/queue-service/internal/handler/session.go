package handler

import (
	"log"
	"net/http"
)

// sessionHeartbeatHandler refreshes the heartbeat TTL for the calling session.
// Called every 10 s by JavaScript on both the queue page and the target page.
// Without regular heartbeats the server-side reaper will evict the session.
func (h *Handler) sessionHeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	sid := sessionID(r)
	if sid == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	ctx := r.Context()
	status := h.store.GetStatus(ctx, sid)
	if status == "active" || status == "queued" {
		h.store.SetHeartbeat(ctx, sid)
	}
	w.WriteHeader(http.StatusOK)
}

// sessionLeaveHandler immediately removes the calling session from the active
// set or queue list. Called via navigator.sendBeacon on the pagehide event so
// that closing a tab frees the slot for the next person in line.
func (h *Handler) sessionLeaveHandler(w http.ResponseWriter, r *http.Request) {
	sid := sessionID(r)
	if sid == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	ctx := r.Context()
	if wasActive := h.store.Leave(ctx, sid); wasActive {
		log.Printf("session %s left target — slot freed", sid[:8])
	} else {
		log.Printf("session %s left queue", sid[:8])
	}
	w.WriteHeader(http.StatusOK)
}
