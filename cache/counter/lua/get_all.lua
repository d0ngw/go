local key = KEYS[1]
if redis.call('EXISTS', key) == 0 then return {} end
redis.call('ZADD', ARGV[1], ARGV[2], key)
return redis.call('HGETALL', key)
