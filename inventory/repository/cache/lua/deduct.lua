-- deduct.lua: 多 SKU 原子预扣
-- KEYS: [stock_key1, stock_key2, ...]
-- ARGV: [qty1, qty2, ...]
-- 返回: 0=成功, >0=失败的 key 索引(1-based)

local n = #KEYS
-- 第一轮：检查所有 SKU 库存是否充足
for i = 1, n do
    local available = tonumber(redis.call('HGET', KEYS[i], 'available') or 0)
    local qty = tonumber(ARGV[i])
    if available < qty then
        return i
    end
end
-- 第二轮：执行扣减
for i = 1, n do
    local qty = tonumber(ARGV[i])
    redis.call('HINCRBY', KEYS[i], 'available', -qty)
    redis.call('HINCRBY', KEYS[i], 'locked', qty)
end
return 0
