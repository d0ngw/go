-- Stage and validate every signed int64 operation before writing any key.
-- Lua numbers cannot represent all int64 values, so arithmetic uses decimal strings.
local function parts(value)
    if type(value) ~= 'string' or not string.match(value, '^%-?%d+$') then return nil end
    local negative = string.sub(value, 1, 1) == '-'
    local digits = negative and string.sub(value, 2) or value
    digits = string.gsub(digits, '^0+', '')
    if digits == '' then digits, negative = '0', false end
    local limit = negative and '9223372036854775808' or '9223372036854775807'
    if #digits > #limit or (#digits == #limit and digits > limit) then return nil end
    return {negative = negative, digits = digits}
end
local function format(p)
    return (p.negative and p.digits ~= '0' and '-' or '') .. p.digits
end
local function compare(a, b)
    if #a ~= #b then return #a < #b and -1 or 1 end
    if a == b then return 0 end
    return a < b and -1 or 1
end
local function sum(a, b)
    local result, carry, i, j = '', 0, #a, #b
    while i > 0 or j > 0 or carry > 0 do
        local x = i > 0 and tonumber(string.sub(a, i, i)) or 0
        local y = j > 0 and tonumber(string.sub(b, j, j)) or 0
        local n = x + y + carry
        result, carry = tostring(n % 10) .. result, math.floor(n / 10)
        i, j = i - 1, j - 1
    end
    return result
end
local function subtract(a, b)
    local result, borrow, j = '', 0, #b
    for i = #a, 1, -1 do
        local y = j > 0 and tonumber(string.sub(b, j, j)) or 0
        local n = tonumber(string.sub(a, i, i)) - y - borrow
        borrow = n < 0 and 1 or 0
        if n < 0 then n = n + 10 end
        result, j = tostring(n) .. result, j - 1
    end
    result = string.gsub(result, '^0+', '')
    return result == '' and '0' or result
end
local function add(a, b)
    a, b = parts(a), parts(b)
    if not a or not b then return nil end
    local digits, negative
    if a.negative == b.negative then
        digits, negative = sum(a.digits, b.digits), a.negative
    elseif compare(a.digits, b.digits) >= 0 then
        digits, negative = subtract(a.digits, b.digits), a.negative
    else
        digits, negative = subtract(b.digits, a.digits), b.negative
    end
    local result = (negative and digits ~= '0' and '-' or '') .. digits
    return parts(result) and result or nil
end
local function key_type(key)
    return redis.call('TYPE', key).ok
end

local key, sync_set, now = KEYS[1], ARGV[1], ARGV[2]
if #ARGV < 5 or (#ARGV - 3) % 2 ~= 0 or key == sync_set then
    return redis.error_reply('invalid counter update arguments')
end
local kt, st = key_type(key), key_type(sync_set)
if (kt ~= 'none' and kt ~= 'hash') or (st ~= 'none' and st ~= 'zset') then
    return redis.error_reply('invalid counter or synchronization key type')
end
local timestamp = parts(now)
if not timestamp or timestamp.negative then return redis.error_reply('invalid counter timestamp') end
local exists = kt == 'hash'
local generation, initial, changes = nil, {}, {}
for i = 4, #ARGV, 2 do
    local field, value = ARGV[i], ARGV[i + 1]
    if field == '__g' then
        if value == '' then return redis.error_reply('empty counter generation') end
        generation = value
    elseif field == '' or field == '_' or string.sub(field, 1, 2) == '__' then
        return redis.error_reply('invalid counter field name')
    else
        local parsed = parts(value)
        if not parsed then return redis.error_reply('counter value is not an int64') end
        if string.sub(field, 1, 1) == '_' then
            initial[string.sub(field, 2)] = format(parsed)
        else
            changes[#changes + 1] = {field, format(parsed)}
        end
    end
end
if not generation then return redis.error_reply('missing counter generation') end
if not exists and ARGV[3] ~= '1' then return {0, 0} end
local staged = {}
if not exists then
    for field, value in pairs(initial) do staged[field] = value end
end
for _, change in ipairs(changes) do
    local field, delta = change[1], change[2]
    local current = staged[field] or (exists and redis.call('HGET', key, field)) or '0'
    local result = add(current, delta)
    if not result then return redis.error_reply('counter increment overflows int64 or contains invalid data') end
    staged[field] = result
end
local version = exists and redis.call('HGET', key, '_w') or '0'
local parsed_version = parts(version)
if not parsed_version or parsed_version.negative then return redis.error_reply('invalid counter version') end
if #changes > 0 then
    version = add(version, '1')
    if not version then return redis.error_reply('counter version overflows int64') end
end
-- All expected failures above leave both keys untouched, including initial loads.
local writes = {'_w', version, '_g', (exists and redis.call('HGET', key, '_g')) or generation}
if not exists then
    writes[#writes + 1], writes[#writes + 2] = '_s', '0'
    writes[#writes + 1], writes[#writes + 2] = '_st', now
end
for field, value in pairs(staged) do
    writes[#writes + 1], writes[#writes + 2] = field, value
end
redis.call('HMSET', key, unpack(writes))
redis.call('ZADD', sync_set, now, key)
return {1, 1}
