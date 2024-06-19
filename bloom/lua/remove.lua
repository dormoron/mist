-- The script expects one key and a variable number of elements to remove from the Cuckoo filter in Redis.
-- Redis Lua scripting conventionally passes KEYS and ARGV arrays to the script.

-- Assign the first key from the KEYS array to the variable `key`.
-- This key represents the Cuckoo filter in Redis where elements will be removed.
local key = KEYS[1]

-- Assign all items in the ARGV array to the variable `elements`.
-- These are the elements that need to be removed from the Cuckoo filter.
local elements = ARGV

-- Initialize a counter to zero to keep track of the number of successfully removed elements.
local removedCount = 0

-- Loop over each element in the `elements` array.
for i = 1, #elements do
    -- Call the Redis `CF.DEL` command with the key and the current element.
    -- The `CF.DEL` command attempts to remove the specified element from the Cuckoo filter.
    local result = redis.call("CF.DEL", key, elements[i])

    -- If the element is successfully removed, `result` will equal 1.
    if result == 1 then
        -- Increment the `removedCount` counter by 1 to record the successful removal.
        removedCount = removedCount + 1
    end
end

-- Return the total number of successfully removed elements.
return removedCount