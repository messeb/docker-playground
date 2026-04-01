package handler

import (
	"encoding/json"
	"net/http"
)

// adminCapacityHandler sets the maximum number of concurrently active users.
// Queued sessions are admitted immediately if the new limit opens up slots.
//
// POST /api/admin/capacity
// Body: {"capacity": 5}
func (h *Handler) adminCapacityHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Capacity int `json:"capacity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Capacity <= 0 {
		http.Error(w, `{"error":"capacity must be a positive integer"}`, http.StatusBadRequest)
		return
	}
	snap := h.store.SetCapacity(r.Context(), req.Capacity)
	writeJSON(w, http.StatusOK, snap)
}

// adminReleaseHandler evicts N active sessions and admits the next users from
// the queue. Simulates users leaving the target page.
//
// POST /api/admin/release
// Body: {"count": 1}
func (h *Handler) adminReleaseHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Count int `json:"count"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 100 {
		req.Count = 100
	}
	released, snap := h.store.ReleaseSlots(r.Context(), req.Count)
	writeJSON(w, http.StatusOK, map[string]any{
		"released": released,
		"capacity": snap.Capacity,
		"active":   snap.Active,
		"queued":   snap.Queued,
	})
}

// adminStatusHandler returns a live snapshot of the queue state.
//
// GET /api/admin/status
func (h *Handler) adminStatusHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Snapshot(r.Context()))
}
