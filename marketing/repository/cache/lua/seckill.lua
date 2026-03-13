-- seckill.lua: 秒杀原子扣减 + 防重复
-- KEYS[1]: seckill:stock:{itemId}
-- KEYS[2]: seckill:user:{itemId}
-- ARGV[1]: userId
-- ARGV[2]: perLimit
-- 返回: 0=成功, 1=库存不足, 2=已抢购过, 3=超出限购

-- 检查是否已抢购
local userSet = KEYS[2]
local userId = ARGV[1]
local isMember = redis.call('SISMEMBER', userSet, userId)
if isMember == 1 then
    return 2
end

-- 检查库存
local stockKey = KEYS[1]
local stock = tonumber(redis.call('GET', stockKey) or '0')
if stock <= 0 then
    return 1
end

-- 扣减库存
redis.call('DECR', stockKey)
-- 记录用户
redis.call('SADD', userSet, userId)
return 0
