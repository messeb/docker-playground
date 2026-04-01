package queue

import "github.com/redis/go-redis/v9"

// All queue mutations run as atomic Lua scripts on the Redis server.
// This prevents race conditions such as two users claiming the same last slot.

// luaTryAdmit atomically checks remaining capacity and claims a slot.
// KEYS: [capacity, active-set, session-hash, timing-history]
// ARGV: [sessionID, unix-timestamp]
// Returns 1 if admitted, 0 if at capacity.
var luaTryAdmit = redis.NewScript(`
local capacity = tonumber(redis.call('GET', KEYS[1]) or '3')
local activeCount = redis.call('SCARD', KEYS[2])
if activeCount >= capacity then return 0 end
redis.call('SADD', KEYS[2], ARGV[1])
redis.call('HSET', KEYS[3], 'status', 'active', 'admitted_at', ARGV[2])
redis.call('EXPIRE', KEYS[3], 86400)
redis.call('LPUSH', KEYS[4], ARGV[2])
redis.call('LTRIM', KEYS[4], 0, 99)
return 1
`)

// luaJoinQueue appends a session to the waiting list (idempotent).
// KEYS: [session-hash, queue-list]
// ARGV: [sessionID, unix-timestamp]
// Returns 1 if newly queued, 0 if already queued or active.
var luaJoinQueue = redis.NewScript(`
local existing = redis.call('HGET', KEYS[1], 'status')
if existing == 'queued' or existing == 'active' then return 0 end
redis.call('RPUSH', KEYS[2], ARGV[1])
redis.call('HSET', KEYS[1], 'status', 'queued', 'joined_at', ARGV[2])
redis.call('EXPIRE', KEYS[1], 86400)
return 1
`)

// luaAdmitNext pops the next valid session from the queue and admits it.
// Expired/stale entries are silently discarded until a live one is found.
// KEYS: [capacity, active-set, queue-list, timing-history]
// ARGV: [unix-timestamp]
// Returns the admitted sessionID, or nil if the queue is empty or at capacity.
var luaAdmitNext = redis.NewScript(`
local capacity = tonumber(redis.call('GET', KEYS[1]) or '3')
for _ = 1, 50 do
  local activeCount = redis.call('SCARD', KEYS[2])
  if activeCount >= capacity then return nil end
  local sid = redis.call('LPOP', KEYS[3])
  if not sid then return nil end
  if redis.call('HEXISTS', 'session:' .. sid, 'status') == 1 then
    redis.call('SADD', KEYS[2], sid)
    redis.call('HSET', 'session:' .. sid, 'status', 'active', 'admitted_at', ARGV[1])
    redis.call('LPUSH', KEYS[4], ARGV[1])
    redis.call('LTRIM', KEYS[4], 0, 99)
    return sid
  end
end
return nil
`)

// luaRelease removes N random sessions from the active set (admin use).
// KEYS: [active-set]
// ARGV: [count]
// Returns the number of sessions actually removed.
var luaRelease = redis.NewScript(`
local count = tonumber(ARGV[1])
local released = 0
for _ = 1, count do
  local sid = redis.call('SPOP', KEYS[1])
  if not sid then break end
  redis.call('DEL', 'session:' .. sid)
  redis.call('DEL', 'heartbeat:' .. sid)
  released = released + 1
end
return released
`)

// luaLeave removes a specific session from the active set or queue list.
// KEYS: [active-set, queue-list]
// ARGV: [sessionID]
// Returns 1 if the session held an active slot, 0 if it was only queued.
var luaLeave = redis.NewScript(`
local sid = ARGV[1]
local wasActive = redis.call('SREM', KEYS[1], sid)
if wasActive == 0 then
  redis.call('LREM', KEYS[2], 1, sid)
end
redis.call('DEL', 'session:' .. sid)
redis.call('DEL', 'heartbeat:' .. sid)
return wasActive
`)

// luaReapStale scans both the active set and queue list, evicting sessions
// whose heartbeat key has expired (the user's tab is gone).
// KEYS: [active-set, queue-list]
// Returns the total number of sessions removed.
var luaReapStale = redis.NewScript(`
local evicted = 0
for _, sid in ipairs(redis.call('SMEMBERS', KEYS[1])) do
  if redis.call('EXISTS', 'heartbeat:' .. sid) == 0 then
    redis.call('SREM', KEYS[1], sid)
    redis.call('DEL', 'session:' .. sid)
    evicted = evicted + 1
  end
end
for _, sid in ipairs(redis.call('LRANGE', KEYS[2], 0, -1)) do
  if redis.call('EXISTS', 'heartbeat:' .. sid) == 0 then
    redis.call('LREM', KEYS[2], 1, sid)
    redis.call('DEL', 'session:' .. sid)
    evicted = evicted + 1
  end
end
return evicted
`)
