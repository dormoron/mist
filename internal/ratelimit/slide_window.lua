-- This Lua script is used by a RedisSlidingWindowLimiter to control access frequency to resources.
-- It ensures that a particular action can only occur at a specified rate within a sliding window of time.

-- KEYS[1] holds the key for the ZSET used for maintaining timestamps of actions.
local key = KEYS[1]

-- ARGV parameters are used to pass the sliding window's length, the maximum allowed rate, and the current timestamp.
local window = tonumber(ARGV[1])       -- The sliding window's length in milliseconds.
local threshold = tonumber(ARGV[2])    -- The maximum number of actions allowed in the window.
local now = tonumber(ARGV[3])          -- The current timestamp in milliseconds.

-- Compute the minimum score for the ZSET to determine which entries are within the sliding window.
local min = now - window

-- Remove all entries in the ZSET that are outside of the sliding window.
redis.call('ZREMRANGEBYSCORE', key, '-inf', min)

-- Count the number of remaining entries in the ZSET, which equals the number of actions in the sliding window.
local cnt = redis.call('ZCOUNT', key, '-inf', '+inf')

-- If the count of actions exceeds the threshold, return "true" to indicate rate limit has been exceeded.
if cnt >= threshold then
    return "true"

    -- Otherwise, add the current timestamp to the ZSET and set an expiration equal to the window size.
    -- This action signifies a successful attempt within rate limits and returns "false" to signify availability for more actions.
else
    redis.call('ZADD', key, now, now)         -- Add the current timestamp as score and value.
    redis.call('PEXPIRE', key, window)        -- Set the expiration for the ZSET to the window size for automatic cleanup.
    return "false"
end