package list

import "github.com/garyburd/redigo/redis"

var addLua = `
local list_key = KEYS[1]
local must_exist_key = tonumber(ARGV[1])
local expire_seconds = tonumber(ARGV[2])

if #ARGV < 4 or (#ARGV - 4) % 2 ~= 0 then
    return redis.error_reply("Wrong score and member args numbers")
end

redis.call("PERSIST", list_key)
local exist = redis.call("EXISTS", list_key)
local updated = 0
local need_update = false

if exist == 1 then
    need_update = true
else
    if must_exist_key == 1 then
        need_update = false
    else
        need_update = true
    end
end

if need_update then
    local score_members = {}
    for i = 3, #ARGV, 2 do
        score_members[#score_members + 1] = ARGV[i]
        score_members[#score_members + 1] = ARGV[i + 1]
    end
    redis.call("ZADD", list_key, unpack(score_members))
    updated = 1
    exist = 1
end

if expire_seconds > 0 then
    redis.call("EXPIRE", list_key, expire_seconds)
end

return { exist, updated }
`
var addScript = redis.NewScript(1, addLua)

var delLua = `
local list_key = KEYS[1]
local expire_seconds = tonumber(ARGV[1])

if #ARGV < 2 then
    return redis.error_reply("Wrong args numbers")
end

redis.call("PERSIST", list_key)
local exist = redis.call("EXISTS", list_key)
local deleted = 0
local last_member = ""
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

`

var delScript = redis.NewScript(1, delLua)
