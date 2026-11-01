redis.call('ZREM', ARGV[1], KEYS[1])
return redis.call('DEL', KEYS[1])
