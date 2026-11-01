-- Called while holding the persistence coordinator. Upgrade legacy hashes once.
if redis.call('EXISTS', KEYS[1]) == 0 then return {} end
redis.call('HSETNX', KEYS[1], '_g', ARGV[1])
return redis.call('HGETALL', KEYS[1])
