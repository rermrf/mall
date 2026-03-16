-- ============================================================
-- [DEPRECATED] 请使用 Go seed 工具代替:
--   go run ./script/seed/         (登录账号)
--   go run ./script/seed/ -reset  (清空+完整测试数据)
-- ============================================================
-- Mall 开发环境种子数据 (旧版，仅供参考)
-- ============================================================

-- ============================================================
-- 1. 创建平台管理员 (tenant_id=0)
--    手机号: 13800000000  密码: admin123
--    bcrypt hash 由 Go seed 工具生成，这里用占位符
--    如需手动执行，请先用 Go 工具生成 hash 或替换下面的值
-- ============================================================
-- 注意: 请使用 go run script/seed/main.go 来执行，它会自动处理 bcrypt

-- ============================================================
-- 2. 创建套餐
-- ============================================================
USE mall_tenant;

INSERT INTO tenant_plans (name, price, duration_days, max_products, max_staff, features, status, ctime, utime)
VALUES ('免费试用', 0, 365, 100, 5, '基础店铺功能,商品管理,订单管理', 1, UNIX_TIMESTAMP()*1000, UNIX_TIMESTAMP()*1000)
ON DUPLICATE KEY UPDATE name=name;

-- ============================================================
-- 3. 创建租户 (status=2 已审核)
-- ============================================================
INSERT INTO tenants (name, contact_name, contact_phone, status, plan_id, plan_expire_time, ctime, utime)
VALUES ('演示商城', '张三', '13800000001', 2, 1, (UNIX_TIMESTAMP() + 365*86400)*1000, UNIX_TIMESTAMP()*1000, UNIX_TIMESTAMP()*1000)
ON DUPLICATE KEY UPDATE name=name;

-- ============================================================
-- 4. 创建店铺 (subdomain=demo, custom_domain=localhost)
--    consumer-bff TenantResolve 中间件:
--    - 生产: 通过 subdomain/custom_domain 解析
--    - localhost: 通过 X-Tenant-Domain 请求头解析
-- ============================================================
INSERT INTO shops (tenant_id, name, logo, description, status, rating, subdomain, custom_domain, ctime, utime)
VALUES (1, '演示商城', '', '演示用商城店铺', 1, '5.0', 'shop1', 'localhost', UNIX_TIMESTAMP()*1000, UNIX_TIMESTAMP()*1000)
ON DUPLICATE KEY UPDATE name=name;

-- ============================================================
-- 5. 初始化配额
-- ============================================================
INSERT INTO tenant_quota_usage (tenant_id, quota_type, used, max_limit, utime)
VALUES (1, 'product_count', 0, 100, UNIX_TIMESTAMP()*1000)
ON DUPLICATE KEY UPDATE max_limit=100;

INSERT INTO tenant_quota_usage (tenant_id, quota_type, used, max_limit, utime)
VALUES (1, 'staff_count', 0, 5, UNIX_TIMESTAMP()*1000)
ON DUPLICATE KEY UPDATE max_limit=5;

SELECT '✅ tenant + shop + quotas 创建完成' AS result;
