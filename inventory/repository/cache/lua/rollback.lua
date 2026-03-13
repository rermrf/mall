-- rollback.lua: 多 SKU 原子回滚
-- KEYS: [stock_key1, stock_key2, ...]
-- ARGV: [qty1, qty2, ...]

local n = #KEYS
for i = 1, n do
    local qty = tonumber(ARGV[i])
    redis.call('HINCRBY', KEYS[i], 'available', qty)
    redis.call('HINCRBY', KEYS[i], 'locked', -qty)
end
return 0
