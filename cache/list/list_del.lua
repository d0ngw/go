--[[
    init or update list cache
    params:
    list_key,expire_seconds,member1,member2....
    retun:
    {deleted,last_member,length}
-- ]]
local list_key = KEYS[1]
local expire_seconds = tonumber(ARGV[1])

if #ARGV < 2 then
    return redis.error_reply("Wrong args numbers")
end

redis.call("PERSIST", list_key)
local exist = redis.call("EXISTS", list_key)
local deleted = 0
local last_member = {}
local length = 0

if exist == 1 then
    local members = {}
    for i = 2, #ARGV, 1 do
        members[#members + 1] = ARGV[i]
    end
    deleted=redis.call("ZREM", list_key, unpack(members))
    last_member = redis.call("ZRANGE",list_key,-1,-1)
    length = redis.call("ZCARD",list_key)
end

if expire_seconds > 0 then
    redis.call("EXPIRE", list_key, expire_seconds)
end

return { deleted, last_member,length}
