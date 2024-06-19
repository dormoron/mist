-- Retrieve the first key from the KEYS array, which will be the Bloom filter key in Redis.
local key = KEYS[1]

-- Retrieve all the elements to be checked from the ARGV array.
local elements = ARGV

-- Initialize an empty table to store the results of the existence checks.
local results = {}

-- Iterate over each element in the elements array.
for i = 1, #elements do
    -- Execute the Redis command `BF.EXISTS` to check if the current element exists in the Bloom filter.
    -- Parameters:
    -- - key: The Redis key for the Bloom filter.
    -- - elements[i]: The current element being checked for existence.
    -- Returns:
    -- - result: 1 if the element may exist, 0 if it definitely does not exist.
    local result = redis.call("BF.EXISTS", key, elements[i])

    -- Insert the result (1 or 0) into the results table.
    table.insert(results, result)
end

-- Return the results table containing the existence check results for all elements.
return results