package queue

import (
	"context"
	"log"
	"strconv"
	"time"
)

// RunWorker starts two background loops:
//   - every 1 s: fill open slots from the queue
//   - every 5 s: evict sessions whose heartbeat has expired
//
// It blocks until ctx is cancelled and is intended to run as a goroutine.
func (s *Store) RunWorker(ctx context.Context) {
	admitTick := time.NewTicker(time.Second)
	reaperTick := time.NewTicker(5 * time.Second)
	defer admitTick.Stop()
	defer reaperTick.Stop()

	for {
		select {
		case <-admitTick.C:
			s.admitPending(ctx)
		case <-reaperTick.C:
			s.reapStale(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// admitPending moves sessions from the waiting list into open slots.
// It loops until capacity is reached or the queue is empty, seeding a fresh
// heartbeat key for every newly admitted session.
func (s *Store) admitPending(ctx context.Context) {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	for {
		result, err := luaAdmitNext.Run(ctx, s.rdb,
			[]string{keyCapacity, keyActive, keyQueueList, keyHistory},
			now,
		).Result()
		if err != nil || result == nil {
			return
		}
		if sid, ok := result.(string); ok {
			s.SetHeartbeat(ctx, sid)
		}
	}
}

// reapStale evicts sessions that have stopped sending heartbeats (e.g. the
// browser was closed without the pagehide beacon firing). Frees up any slots
// held by ghost active sessions and cleans stale queue entries.
func (s *Store) reapStale(ctx context.Context) {
	evicted, _ := luaReapStale.Run(ctx, s.rdb, []string{keyActive, keyQueueList}).Int()
	if evicted > 0 {
		log.Printf("reaped %d stale session(s)", evicted)
		s.admitPending(ctx)
	}
}
