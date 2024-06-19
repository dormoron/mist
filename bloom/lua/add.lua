-- `key` holds the Redis key for the Bloom filter.
-- KEYS and ARGV are the standard arguments passed to Redis Lua scripts.

local key = KEYS[1] -- The first (and only) key passed to the script, representing the Bloom filter in Redis.
local elements = ARGV -- All arguments passed to the script, which are the elements to be added to the Bloom filter.
local addedCount = 0 -- Initialize a count to keep track of the number of successfully added elements.

-- Loop through each element in the `elements` array.
for i = 1, #elements do
    -- Call the Redis Bloom filter 'ADD' command with the key and the current element.
    local result = redis.call("BF.ADD", key, elements[i])

    -- If the element is successfully added to the Bloom filter, the result will be 1.
    if result == 1 then
        addedCount = addedCount + 1 -- Increment the count of successfully added elements.
    end
end

-- Return the total number of added elements.
return addedCount