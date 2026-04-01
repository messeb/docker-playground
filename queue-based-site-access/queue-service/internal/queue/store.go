package queue

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ── Redis keys ────────────────────────────────────────────────────────────────

const (
	keyQueueList  = "queue:list"            // Redis list   — FIFO waiting session IDs
	keyActive     = "queue:active"          // Redis set    — currently active session IDs
	keyCapacity   = "queue:capacity"        // Redis string — max concurrent users
	keyHistory    = "queue:timing_history"  // Redis list   — unix timestamps of admissions

	// heartbeatTTL is how long a session may be silent before being reaped.
	// Pages ping every 10 s, so 30 s gives a comfortable 3× margin.
	heartbeatTTL = 30 * time.Second
)

func sessionKey(id string) string   { return "session:" + id }
func heartbeatKey(id string) string { return "heartbeat:" + id }

// ── Store ─────────────────────────────────────────────────────────────────────

// Store wraps Redis and exposes all queue operations.
// All mutations use atomic Lua scripts to prevent race conditions.
type Store struct {
	rdb *redis.Client
}

// New creates a Store backed by the given Redis client.
func New(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

// InitCapacity seeds the queue capacity only when the key does not already
// exist, so a value set via the admin API survives service restarts.
func (s *Store) InitCapacity(ctx context.Context, initial int) {
	s.rdb.SetNX(ctx, keyCapacity, initial, 0)
}

// Ping checks the Redis connection.
func (s *Store) Ping(ctx context.Context) error {
	return s.rdb.Ping(ctx).Err()
}

// ── Session state ─────────────────────────────────────────────────────────────

// GetStatus returns the session's current status: "active", "queued", or "".
func (s *Store) GetStatus(ctx context.Context, sid string) string {
	return s.rdb.HGet(ctx, sessionKey(sid), "status").Val()
}

// IsInActiveSet reports whether the session holds an active slot.
func (s *Store) IsInActiveSet(ctx context.Context, sid string) bool {
	ok, _ := s.rdb.SIsMember(ctx, keyActive, sid).Result()
	return ok
}

// DeleteSession removes a session hash from Redis.
func (s *Store) DeleteSession(ctx context.Context, sid string) {
	s.rdb.Del(ctx, sessionKey(sid))
}

// SetHeartbeat refreshes the heartbeat key, keeping the session alive.
func (s *Store) SetHeartbeat(ctx context.Context, sid string) {
	s.rdb.Set(ctx, heartbeatKey(sid), "1", heartbeatTTL)
}

// ── Queue operations ──────────────────────────────────────────────────────────

// TryAdmit atomically attempts to claim an active slot for the session.
// Returns true if the session was admitted.
func (s *Store) TryAdmit(ctx context.Context, sid, now string) bool {
	n, _ := luaTryAdmit.Run(ctx, s.rdb,
		[]string{keyCapacity, keyActive, sessionKey(sid), keyHistory},
		sid, now,
	).Int()
	return n == 1
}

// JoinQueue appends the session to the FIFO waiting list (idempotent).
func (s *Store) JoinQueue(ctx context.Context, sid, now string) {
	luaJoinQueue.Run(ctx, s.rdb,
		[]string{sessionKey(sid), keyQueueList},
		sid, now,
	)
}

// Leave removes the session from the active set or queue list immediately.
// Returns true if the session held an active slot (a new slot is now free).
func (s *Store) Leave(ctx context.Context, sid string) bool {
	wasActive, _ := luaLeave.Run(ctx, s.rdb, []string{keyActive, keyQueueList}, sid).Int()
	if wasActive > 0 {
		s.admitPending(ctx)
	}
	return wasActive > 0
}

// ── Position & ETA ────────────────────────────────────────────────────────────

// Position holds a session's place in the waiting list.
type Position struct {
	Pos   int // 1-indexed position
	Total int // total number of waiting sessions
}

// GetPosition returns the session's current position in the queue.
// The second return value is false when the session is not in the queue.
func (s *Store) GetPosition(ctx context.Context, sid string) (Position, bool) {
	list, _ := s.rdb.LRange(ctx, keyQueueList, 0, -1).Result()
	for i, id := range list {
		if id == sid {
			return Position{Pos: i + 1, Total: len(list)}, true
		}
	}
	return Position{}, false
}

// CalculateETA returns the estimated wait in seconds for the given position.
// Returns -1 when fewer than 5 admissions have been recorded.
func (s *Store) CalculateETA(ctx context.Context, position int) int {
	history, err := s.rdb.LRange(ctx, keyHistory, 0, -1).Result()
	if err != nil || len(history) < 5 {
		return -1
	}
	timestamps := make([]int64, 0, len(history))
	for _, h := range history {
		if ts, err := strconv.ParseInt(h, 10, 64); err == nil {
			timestamps = append(timestamps, ts)
		}
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
	if len(timestamps) < 2 {
		return -1
	}
	var total int64
	for i := 1; i < len(timestamps); i++ {
		total += timestamps[i] - timestamps[i-1]
	}
	avg := total / int64(len(timestamps)-1)
	if avg == 0 {
		return 0
	}
	return int(avg) * (position - 1)
}

// ── Admin operations ──────────────────────────────────────────────────────────

// Snapshot is a point-in-time view of the queue returned by admin endpoints.
type Snapshot struct {
	Capacity int   `json:"capacity"`
	Active   int64 `json:"active"`
	Queued   int64 `json:"queued"`
	History  int64 `json:"history_entries"`
}

// SetCapacity updates the maximum number of concurrent users and immediately
// admits queued sessions into any newly opened slots.
func (s *Store) SetCapacity(ctx context.Context, n int) Snapshot {
	s.rdb.Set(ctx, keyCapacity, n, 0)
	s.admitPending(ctx)
	return s.Snapshot(ctx)
}

// ReleaseSlots evicts n random active sessions and admits the next users
// from the queue. Returns the number released and a current snapshot.
func (s *Store) ReleaseSlots(ctx context.Context, n int) (released int, snap Snapshot) {
	released, _ = luaRelease.Run(ctx, s.rdb, []string{keyActive}, strconv.Itoa(n)).Int()
	s.admitPending(ctx)
	snap = s.Snapshot(ctx)
	return
}

// Snapshot returns a live summary of queue state.
func (s *Store) Snapshot(ctx context.Context) Snapshot {
	capacity, _ := s.rdb.Get(ctx, keyCapacity).Int()
	active, _ := s.rdb.SCard(ctx, keyActive).Result()
	queued, _ := s.rdb.LLen(ctx, keyQueueList).Result()
	history, _ := s.rdb.LLen(ctx, keyHistory).Result()
	return Snapshot{
		Capacity: capacity,
		Active:   active,
		Queued:   queued,
		History:  history,
	}
}
