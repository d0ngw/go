local key, sync_set = KEYS[1], ARGV[1]
-- A stale scan must never remove a newer incarnation or its sync-set membership.
if redis.call('EXISTS', key) == 0 or redis.call('HGET', key, '_g') ~= ARGV[3] then return {0} end
local written, synced = redis.call('HGET', key, '_w'), redis.call('HGET', key, '_s')
if written ~= ARGV[2] or synced ~= written then return {0} end
redis.call('DEL', key)
redis.call('ZREM', sync_set, key)
return {1}
