-- A generation identifies a hash incarnation; versions alone repeat after deletion.
local key = KEYS[1]
if redis.call('EXISTS', key) == 0 or redis.call('HGET', key, '_g') ~= ARGV[3] then return 0 end
local function valid(v)
    return v and (v == '0' or string.match(v, '^[1-9]%d*$')) and (#v < 19 or (#v == 19 and v <= '9223372036854775807'))
end
local function less(a, b)
    return #a < #b or (#a == #b and a < b)
end
local version = ARGV[1]
local written, synced = redis.call('HGET', key, '_w'), redis.call('HGET', key, '_s')
if not valid(version) or not valid(written) or not valid(synced) then return redis.error_reply('invalid counter version') end
if less(written, version) then return redis.error_reply('sync version exceeds write version') end
if not less(version, synced) then redis.call('HMSET', key, '_s', version, '_st', ARGV[2]) end
return 1
