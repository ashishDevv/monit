package scheduler

// scheduleKey = "monitor:schedule"
// inflightKey = "monitor:inflight"

const fetchDueMonitorsScript = `
local key = KEYS[1]
local now = ARGV[1]
local limit = tonumber(ARGV[2])

local items = redis.call("ZRANGEBYSCORE", key, "-inf", now, "LIMIT", 0, limit)

for i, member in ipairs(items) do
	redis.call("ZREM", key, member)
end

return items
`

const fetchAndMoveToInflightScript = `
local scheduleKey = KEYS[1]
local inflightKey = KEYS[2]

local now = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local visibilityTimeout = tonumber(ARGV[3])

-- Step 1: Fetch due jobs
local items = redis.call("ZRANGEBYSCORE", scheduleKey, "-inf", now, "LIMIT", 0, limit)

-- Step 2: Move each to inflight
for i, member in ipairs(items) do
    redis.call("ZREM", scheduleKey, member)
    redis.call("ZADD", inflightKey, now + visibilityTimeout, member)
end

return items
`

const reclaimMonitorsScript = `
local inflightKey = KEYS[1]
local scheduleKey = KEYS[2]

local now = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])

local items = redis.call("ZRANGEBYSCORE", inflightKey, "-inf", now, "LIMIT", 0, limit)

for i, member in ipairs(items) do
    redis.call("ZREM", inflightKey, member)
    redis.call("ZADD", scheduleKey, now, member)
end

return #items
`
