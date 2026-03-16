-- ============================================================
-- [DEPRECATED] 请使用 Go seed 工具代替:
--   go run ./script/seed/ -reset  (清空+完整测试数据，包含以下所有内容)
-- ============================================================
-- Mall 商家端测试数据初始化脚本 (旧版，仅供参考)
-- ============================================================

SET @NOW = UNIX_TIMESTAMP() * 1000;
SET @TENANT = 1;

-- ============================================================
-- 1. 商家用户 + 角色 + 权限
-- ============================================================
USE mall_user;

-- 1a. 商家管理员账号 (phone: 13900000001, password: merchant123)
--     bcrypt hash for "merchant123" (cost=10)
--     如果 hash 不匹配你的环境，请用 go run script/seed/main.go 重新生成
INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
VALUES (@TENANT, '13900000001', 'merchant@demo.com',
        '$2a$10$YE3xG3mIwByCFVhkfMQage.1NiMyJBfyelCKWMHOsv5Y7gSfyiHKm',
        '店铺管理员', '', 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

SET @MERCHANT_USER = (SELECT id FROM users WHERE tenant_id = @TENANT AND phone = '13900000001' LIMIT 1);

-- 1b. 店员账号 (phone: 13900000002, password: staff123)
INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
VALUES (@TENANT, '13900000002', 'staff@demo.com',
        '$2a$10$N8zVJqEAa/yFfNKvBQ4JFOxZJ0JT5lhVRkfL1PmkXpH6aJ0eXU4em',
        '客服小王', '', 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

SET @STAFF_USER = (SELECT id FROM users WHERE tenant_id = @TENANT AND phone = '13900000002' LIMIT 1);

-- 1c. 测试消费者账号 (phone: 13800001111, password: user123)
INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
VALUES (@TENANT, '13800001111', 'buyer@demo.com',
        '$2a$10$DCq7uP1BTZM0D14PFRbEb.QSw.nP3N6hI9LRMCCNkXsVmT2NOXfXG',
        '测试买家', '', 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

SET @BUYER_USER = (SELECT id FROM users WHERE tenant_id = @TENANT AND phone = '13800001111' LIMIT 1);

-- 1d. 角色定义
INSERT INTO roles (tenant_id, name, code, description, ctime, utime) VALUES
(@TENANT, '店铺管理员', 'merchant_admin', '拥有全部店铺管理权限', @NOW, @NOW),
(@TENANT, '客服',       'merchant_cs',    '订单查看、退款处理', @NOW, @NOW),
(@TENANT, '运营',       'merchant_ops',   '商品管理、营销管理', @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

SET @ROLE_ADMIN = (SELECT id FROM roles WHERE tenant_id = @TENANT AND code = 'merchant_admin' LIMIT 1);
SET @ROLE_CS    = (SELECT id FROM roles WHERE tenant_id = @TENANT AND code = 'merchant_cs' LIMIT 1);

-- 1e. 角色分配
INSERT INTO user_roles (user_id, tenant_id, role_id, ctime, utime) VALUES
(@MERCHANT_USER, @TENANT, @ROLE_ADMIN, @NOW, @NOW),
(@STAFF_USER,    @TENANT, @ROLE_CS,    @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 1f. 买家收货地址
INSERT INTO user_addresses (user_id, name, phone, province, city, district, detail, is_default, ctime, utime) VALUES
(@BUYER_USER, '张三', '13800001111', '广东省', '深圳市', '南山区', '科技园南路88号创业大厦A座1201', true,  @NOW, @NOW),
(@BUYER_USER, '李四', '13800002222', '北京市', '朝阳区', '三里屯', '工体北路甲2号盈科中心3层',     false, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

SELECT CONCAT('✅ 用户/角色创建完成: 商家管理员=', @MERCHANT_USER, ', 店员=', @STAFF_USER, ', 买家=', @BUYER_USER) AS result;

-- ============================================================
-- 2. 商品体系: 分类 + 品牌 + 商品 + SKU + 规格
-- ============================================================
USE mall_product;

-- 2a. 分类 (两级)
INSERT INTO categories (id, tenant_id, parent_id, name, level, sort, icon, status, ctime, utime) VALUES
(1, @TENANT, 0, '手机数码',   1, 1, '', 1, @NOW, @NOW),
(2, @TENANT, 0, '服装鞋帽',   1, 2, '', 1, @NOW, @NOW),
(3, @TENANT, 0, '食品饮料',   1, 3, '', 1, @NOW, @NOW),
(4, @TENANT, 0, '家居百货',   1, 4, '', 1, @NOW, @NOW),
(5, @TENANT, 1, '智能手机',   2, 1, '', 1, @NOW, @NOW),
(6, @TENANT, 1, '平板电脑',   2, 2, '', 1, @NOW, @NOW),
(7, @TENANT, 1, '智能手表',   2, 3, '', 1, @NOW, @NOW),
(8, @TENANT, 2, '男装',       2, 1, '', 1, @NOW, @NOW),
(9, @TENANT, 2, '女装',       2, 2, '', 1, @NOW, @NOW),
(10, @TENANT, 2, '运动鞋',   2, 3, '', 1, @NOW, @NOW),
(11, @TENANT, 3, '零食',     2, 1, '', 1, @NOW, @NOW),
(12, @TENANT, 3, '饮品',     2, 2, '', 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 2b. 品牌
INSERT INTO brands (id, tenant_id, name, logo, status, ctime, utime) VALUES
(1, @TENANT, 'TechPro',     '', 1, @NOW, @NOW),
(2, @TENANT, 'StyleWear',   '', 1, @NOW, @NOW),
(3, @TENANT, 'FreshBite',   '', 1, @NOW, @NOW),
(4, @TENANT, 'HomeComfort', '', 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 2c. 商品 (8 个，覆盖不同分类)
INSERT INTO products (id, tenant_id, category_id, brand_id, name, subtitle, main_image, images, description, status, sales, ctime, utime) VALUES
(1, @TENANT, 5, 1, 'TechPro X20 智能手机',
   '旗舰芯片 | 1亿像素 | 5000mAh 超长续航',
   'https://picsum.photos/seed/phone1/400/400',
   '["https://picsum.photos/seed/phone1a/800/800","https://picsum.photos/seed/phone1b/800/800"]',
   '搭载最新旗舰芯片，1亿像素主摄，支持100W快充，5000mAh超大电池，120Hz AMOLED屏幕。',
   1, 328, @NOW, @NOW),

(2, @TENANT, 5, 1, 'TechPro Lite 轻薄手机',
   '轻薄设计 | 6400万像素 | 快充',
   'https://picsum.photos/seed/phone2/400/400',
   '["https://picsum.photos/seed/phone2a/800/800"]',
   '仅7.5mm超薄机身，158g轻盈手感，6400万像素AI三摄。',
   1, 156, @NOW, @NOW),

(3, @TENANT, 6, 1, 'TechPro Pad 11 平板电脑',
   '11英寸2K屏 | 骁龙处理器 | 手写笔',
   'https://picsum.photos/seed/pad1/400/400',
   '["https://picsum.photos/seed/pad1a/800/800"]',
   '11英寸2K IPS屏幕，骁龙8系芯片，8GB+256GB，支持手写笔和键盘套件。',
   1, 89, @NOW, @NOW),

(4, @TENANT, 7, 1, 'TechPro Watch S3 智能手表',
   '全天候健康监测 | GPS | 14天续航',
   'https://picsum.photos/seed/watch1/400/400',
   '[]',
   '1.43英寸AMOLED圆表盘，血氧/心率/睡眠监测，100+运动模式。',
   1, 212, @NOW, @NOW),

(5, @TENANT, 8, 2, 'StyleWear 商务休闲衬衫',
   '免烫面料 | 修身版型 | 春秋款',
   'https://picsum.photos/seed/shirt1/400/400',
   '["https://picsum.photos/seed/shirt1a/800/800"]',
   '60支免烫面料，修身剪裁，适合商务和日常穿搭。',
   1, 567, @NOW, @NOW),

(6, @TENANT, 10, 2, 'StyleWear Air 跑步鞋',
   '超轻缓震 | 透气网面 | 碳板加持',
   'https://picsum.photos/seed/shoe1/400/400',
   '["https://picsum.photos/seed/shoe1a/800/800"]',
   '全新碳板中底，回弹率85%，透气飞织鞋面，仅220g。',
   1, 891, @NOW, @NOW),

(7, @TENANT, 11, 3, 'FreshBite 每日坚果混合装',
   '7种坚果 | 独立包装 | 30日量',
   'https://picsum.photos/seed/nut1/400/400',
   '[]',
   '精选7种优质坚果: 巴旦木、腰果、核桃仁、榛子、蔓越莓、蓝莓干、南瓜子。每日一袋，营养均衡。',
   1, 2345, @NOW, @NOW),

(8, @TENANT, 4, 4, 'HomeComfort 四件套床上用品',
   '60支长绒棉 | 纯色简约 | 多色可选',
   'https://picsum.photos/seed/bed1/400/400',
   '["https://picsum.photos/seed/bed1a/800/800"]',
   '60支新疆长绒棉，活性印染不掉色，包含被套*1、床单*1、枕套*2。',
   1, 433, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 2d. 商品规格
INSERT INTO product_specs (id, product_id, tenant_id, name, `values`) VALUES
(1,  1, @TENANT, '颜色', '星空黑,冰川蓝,晨曦金'),
(2,  1, @TENANT, '存储', '128GB,256GB,512GB'),
(3,  2, @TENANT, '颜色', '珍珠白,薄荷绿'),
(4,  2, @TENANT, '存储', '128GB,256GB'),
(5,  3, @TENANT, '颜色', '深空灰,银色'),
(6,  3, @TENANT, '配置', '8+128GB,8+256GB'),
(7,  4, @TENANT, '颜色', '曜石黑,雾霾蓝,樱花粉'),
(8,  5, @TENANT, '颜色', '白色,浅蓝,灰色'),
(9,  5, @TENANT, '尺码', 'M,L,XL,2XL'),
(10, 6, @TENANT, '颜色', '黑白,全黑,荧光绿'),
(11, 6, @TENANT, '尺码', '39,40,41,42,43,44'),
(12, 7, @TENANT, '规格', '30日装,15日装'),
(13, 8, @TENANT, '颜色', '奶白,浅灰,雾蓝,豆沙粉'),
(14, 8, @TENANT, '尺寸', '1.5m床,1.8m床,2.0m床')
ON DUPLICATE KEY UPDATE `values` = VALUES(`values`);

-- 2e. SKU (选取主要组合)
INSERT INTO product_skus (id, tenant_id, product_id, spec_values, price, original_price, cost_price, sku_code, barcode, status, ctime, utime) VALUES
-- TechPro X20 (3色 x 3存储 = 9 SKU, 这里放主要6个)
(1,  @TENANT, 1, '星空黑,128GB', 399900, 449900, 280000, 'TP-X20-BK-128', '6901234000001', 1, @NOW, @NOW),
(2,  @TENANT, 1, '星空黑,256GB', 449900, 499900, 310000, 'TP-X20-BK-256', '6901234000002', 1, @NOW, @NOW),
(3,  @TENANT, 1, '冰川蓝,128GB', 399900, 449900, 280000, 'TP-X20-BL-128', '6901234000003', 1, @NOW, @NOW),
(4,  @TENANT, 1, '冰川蓝,256GB', 449900, 499900, 310000, 'TP-X20-BL-256', '6901234000004', 1, @NOW, @NOW),
(5,  @TENANT, 1, '晨曦金,256GB', 459900, 509900, 320000, 'TP-X20-GD-256', '6901234000005', 1, @NOW, @NOW),
(6,  @TENANT, 1, '晨曦金,512GB', 519900, 569900, 360000, 'TP-X20-GD-512', '6901234000006', 1, @NOW, @NOW),
-- TechPro Lite (2色 x 2存储 = 4 SKU)
(7,  @TENANT, 2, '珍珠白,128GB', 199900, 229900, 140000, 'TP-LT-WH-128', '6901234000007', 1, @NOW, @NOW),
(8,  @TENANT, 2, '珍珠白,256GB', 229900, 259900, 160000, 'TP-LT-WH-256', '6901234000008', 1, @NOW, @NOW),
(9,  @TENANT, 2, '薄荷绿,128GB', 199900, 229900, 140000, 'TP-LT-GN-128', '6901234000009', 1, @NOW, @NOW),
(10, @TENANT, 2, '薄荷绿,256GB', 229900, 259900, 160000, 'TP-LT-GN-256', '6901234000010', 1, @NOW, @NOW),
-- TechPro Pad 11 (2色 x 2配置 = 4 SKU)
(11, @TENANT, 3, '深空灰,8+128GB', 269900, 299900, 190000, 'TP-PAD-GR-128', '6901234000011', 1, @NOW, @NOW),
(12, @TENANT, 3, '深空灰,8+256GB', 319900, 349900, 220000, 'TP-PAD-GR-256', '6901234000012', 1, @NOW, @NOW),
(13, @TENANT, 3, '银色,8+128GB',   269900, 299900, 190000, 'TP-PAD-SL-128', '6901234000013', 1, @NOW, @NOW),
(14, @TENANT, 3, '银色,8+256GB',   319900, 349900, 220000, 'TP-PAD-SL-256', '6901234000014', 1, @NOW, @NOW),
-- TechPro Watch S3 (3色 单配置)
(15, @TENANT, 4, '曜石黑', 149900, 179900, 90000, 'TP-WS3-BK', '6901234000015', 1, @NOW, @NOW),
(16, @TENANT, 4, '雾霾蓝', 149900, 179900, 90000, 'TP-WS3-BL', '6901234000016', 1, @NOW, @NOW),
(17, @TENANT, 4, '樱花粉', 149900, 179900, 90000, 'TP-WS3-PK', '6901234000017', 1, @NOW, @NOW),
-- 商务休闲衬衫 (3色 x 4码 = 12, 放6个)
(18, @TENANT, 5, '白色,L',   19900, 29900, 8000, 'SW-SH-WH-L',  '6901234000018', 1, @NOW, @NOW),
(19, @TENANT, 5, '白色,XL',  19900, 29900, 8000, 'SW-SH-WH-XL', '6901234000019', 1, @NOW, @NOW),
(20, @TENANT, 5, '浅蓝,L',   19900, 29900, 8000, 'SW-SH-BL-L',  '6901234000020', 1, @NOW, @NOW),
(21, @TENANT, 5, '浅蓝,XL',  19900, 29900, 8000, 'SW-SH-BL-XL', '6901234000021', 1, @NOW, @NOW),
(22, @TENANT, 5, '灰色,M',   19900, 29900, 8000, 'SW-SH-GR-M',  '6901234000022', 1, @NOW, @NOW),
(23, @TENANT, 5, '灰色,2XL', 19900, 29900, 8000, 'SW-SH-GR-2X', '6901234000023', 1, @NOW, @NOW),
-- 跑步鞋 (3色 x 6码 = 18, 放6个)
(24, @TENANT, 6, '黑白,42',   59900, 79900, 25000, 'SW-RN-BW-42', '6901234000024', 1, @NOW, @NOW),
(25, @TENANT, 6, '黑白,43',   59900, 79900, 25000, 'SW-RN-BW-43', '6901234000025', 1, @NOW, @NOW),
(26, @TENANT, 6, '全黑,42',   59900, 79900, 25000, 'SW-RN-BK-42', '6901234000026', 1, @NOW, @NOW),
(27, @TENANT, 6, '荧光绿,41', 62900, 79900, 28000, 'SW-RN-GN-41', '6901234000027', 1, @NOW, @NOW),
(28, @TENANT, 6, '荧光绿,42', 62900, 79900, 28000, 'SW-RN-GN-42', '6901234000028', 1, @NOW, @NOW),
(29, @TENANT, 6, '荧光绿,43', 62900, 79900, 28000, 'SW-RN-GN-43', '6901234000029', 1, @NOW, @NOW),
-- 每日坚果
(30, @TENANT, 7, '30日装', 12900, 16900, 6000, 'FB-NUT-30', '6901234000030', 1, @NOW, @NOW),
(31, @TENANT, 7, '15日装',  6900,  8900, 3200, 'FB-NUT-15', '6901234000031', 1, @NOW, @NOW),
-- 四件套 (4色 x 3尺寸 = 12, 放6个)
(32, @TENANT, 8, '奶白,1.8m床', 39900, 59900, 18000, 'HC-BD-WH-18', '6901234000032', 1, @NOW, @NOW),
(33, @TENANT, 8, '浅灰,1.8m床', 39900, 59900, 18000, 'HC-BD-GR-18', '6901234000033', 1, @NOW, @NOW),
(34, @TENANT, 8, '雾蓝,1.5m床', 36900, 56900, 16000, 'HC-BD-BL-15', '6901234000034', 1, @NOW, @NOW),
(35, @TENANT, 8, '雾蓝,1.8m床', 39900, 59900, 18000, 'HC-BD-BL-18', '6901234000035', 1, @NOW, @NOW),
(36, @TENANT, 8, '豆沙粉,1.8m床', 39900, 59900, 18000, 'HC-BD-PK-18', '6901234000036', 1, @NOW, @NOW),
(37, @TENANT, 8, '豆沙粉,2.0m床', 42900, 62900, 20000, 'HC-BD-PK-20', '6901234000037', 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

SELECT '✅ 商品体系创建完成: 12分类, 4品牌, 8商品, 37个SKU' AS result;

-- ============================================================
-- 3. 库存
-- ============================================================
USE mall_inventory;

INSERT INTO inventories (tenant_id, sku_id, total, available, locked, sold, alert_threshold, ctime, utime) VALUES
-- 手机类 (库存适中)
(@TENANT, 1,  500, 420, 5, 75, 50, @NOW, @NOW),
(@TENANT, 2,  300, 230, 3, 67, 30, @NOW, @NOW),
(@TENANT, 3,  500, 430, 2, 68, 50, @NOW, @NOW),
(@TENANT, 4,  300, 248, 4, 48, 30, @NOW, @NOW),
(@TENANT, 5,  200, 158, 2, 40, 20, @NOW, @NOW),
(@TENANT, 6,  100,  70, 0, 30, 10, @NOW, @NOW),
(@TENANT, 7,  600, 530, 6, 64, 50, @NOW, @NOW),
(@TENANT, 8,  400, 355, 3, 42, 40, @NOW, @NOW),
(@TENANT, 9,  600, 548, 2, 50, 50, @NOW, @NOW),
(@TENANT, 10, 400, 370, 0, 30, 40, @NOW, @NOW),
-- 平板
(@TENANT, 11, 200, 165, 2, 33, 20, @NOW, @NOW),
(@TENANT, 12, 150, 118, 1, 31, 15, @NOW, @NOW),
(@TENANT, 13, 200, 180, 0, 20, 20, @NOW, @NOW),
(@TENANT, 14, 150, 140, 1,  9, 15, @NOW, @NOW),
-- 手表
(@TENANT, 15, 800, 650, 10, 140, 80, @NOW, @NOW),
(@TENANT, 16, 600, 530,  5,  65, 60, @NOW, @NOW),
(@TENANT, 17, 400, 390,  3,   7, 40, @NOW, @NOW),
-- 衬衫 (高库存)
(@TENANT, 18, 1000, 800, 10, 190, 100, @NOW, @NOW),
(@TENANT, 19, 1000, 850,  5, 145, 100, @NOW, @NOW),
(@TENANT, 20,  800, 680,  8, 112,  80, @NOW, @NOW),
(@TENANT, 21,  800, 720,  0,  80,  80, @NOW, @NOW),
(@TENANT, 22,  600, 570,  2,  28,  60, @NOW, @NOW),
(@TENANT, 23,  500, 488,  0,  12,  50, @NOW, @NOW),
-- 跑步鞋
(@TENANT, 24, 500, 350, 8, 142, 50, @NOW, @NOW),
(@TENANT, 25, 500, 380, 5, 115, 50, @NOW, @NOW),
(@TENANT, 26, 400, 290, 3, 107, 40, @NOW, @NOW),
(@TENANT, 27, 300, 210, 6,  84, 30, @NOW, @NOW),
(@TENANT, 28, 300, 180, 4, 116, 30, @NOW, @NOW),
(@TENANT, 29, 300, 230, 2,  68, 30, @NOW, @NOW),
-- 坚果 (高销量)
(@TENANT, 30, 5000, 3200, 50, 1750, 500, @NOW, @NOW),
(@TENANT, 31, 3000, 2350, 20,  630, 300, @NOW, @NOW),
-- 四件套
(@TENANT, 32, 600, 450, 5, 145, 60, @NOW, @NOW),
(@TENANT, 33, 600, 480, 3, 117, 60, @NOW, @NOW),
(@TENANT, 34, 400, 330, 2,  68, 40, @NOW, @NOW),
(@TENANT, 35, 600, 460, 4, 136, 60, @NOW, @NOW),
(@TENANT, 36, 500, 430, 3,  67, 50, @NOW, @NOW),
(@TENANT, 37, 300, 270, 0,  30, 30, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 库存变更日志 (最近几条示例)
INSERT INTO inventory_logs (sku_id, order_id, type, quantity, before_available, after_available, tenant_id, ctime) VALUES
(1,  0, 1, 500, 0,   500, @TENANT, @NOW - 86400000 * 30),
(30, 0, 1, 5000, 0, 5000, @TENANT, @NOW - 86400000 * 30),
(1,  1001, 2, 1, 421, 420, @TENANT, @NOW - 3600000),
(30, 1003, 2, 2, 3202, 3200, @TENANT, @NOW - 1800000),
(24, 0, 3, 100, 250, 350, @TENANT, @NOW - 7200000);

SELECT '✅ 库存初始化完成: 37个SKU库存 + 示例日志' AS result;

-- ============================================================
-- 4. 订单 (多种状态)
-- ============================================================
USE mall_order;

-- 4a. 订单主表
INSERT INTO orders (id, tenant_id, order_no, buyer_id, buyer_hash, status, total_amount, discount_amount, freight_amount, pay_amount, refunded_amount, coupon_id, payment_no, receiver_name, receiver_phone, receiver_address, remark, paid_at, shipped_at, received_at, closed_at, ctime, utime) VALUES
-- 已完成订单
(1001, @TENANT, 'ORD20260301001', @BUYER_USER, 'hash_001', 4,
 399900, 0, 0, 399900, 0, 0, 'PAY20260301001',
 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '请尽快发货',
 @NOW - 86400000*10, @NOW - 86400000*9, @NOW - 86400000*5, 0,
 @NOW - 86400000*10, @NOW - 86400000*5),

(1002, @TENANT, 'ORD20260302001', @BUYER_USER, 'hash_002', 4,
 19900, 0, 0, 19900, 0, 0, 'PAY20260302001',
 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '',
 @NOW - 86400000*8, @NOW - 86400000*7, @NOW - 86400000*3, 0,
 @NOW - 86400000*8, @NOW - 86400000*3),

-- 已发货
(1003, @TENANT, 'ORD20260310001', @BUYER_USER, 'hash_003', 3,
 25800, 0, 0, 25800, 0, 0, 'PAY20260310001',
 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '送货前打电话',
 @NOW - 86400000*3, @NOW - 86400000*2, 0, 0,
 @NOW - 86400000*3, @NOW - 86400000*2),

-- 待发货
(1004, @TENANT, 'ORD20260313001', @BUYER_USER, 'hash_004', 2,
 59900, 5000, 800, 55700, 0, 1, 'PAY20260313001',
 '李四', '13800002222', '北京市朝阳区三里屯工体北路甲2号', '',
 @NOW - 86400000, 0, 0, 0,
 @NOW - 86400000, @NOW - 86400000),

(1005, @TENANT, 'ORD20260313002', @BUYER_USER, 'hash_005', 2,
 149900, 10000, 0, 139900, 0, 0, 'PAY20260313002',
 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '生日礼物',
 @NOW - 43200000, 0, 0, 0,
 @NOW - 43200000, @NOW - 43200000),

-- 待付款
(1006, @TENANT, 'ORD20260314001', @BUYER_USER, 'hash_006', 1,
 449900, 0, 0, 449900, 0, 0, '',
 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '',
 0, 0, 0, 0,
 @NOW - 1800000, @NOW - 1800000),

-- 已取消
(1007, @TENANT, 'ORD20260312001', @BUYER_USER, 'hash_007', 0,
 39900, 0, 0, 39900, 0, 0, '',
 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '',
 0, 0, 0, @NOW - 86400000*2,
 @NOW - 86400000*3, @NOW - 86400000*2),

-- 退款中
(1008, @TENANT, 'ORD20260311001', @BUYER_USER, 'hash_008', 5,
 19900, 0, 0, 19900, 0, 0, 'PAY20260311001',
 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '尺码不合适',
 @NOW - 86400000*3, @NOW - 86400000*2, @NOW - 86400000, 0,
 @NOW - 86400000*3, @NOW - 86400000)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 4b. 订单商品明细
INSERT INTO order_items (order_id, tenant_id, product_id, sku_id, product_name, sku_spec, image, price, quantity, subtotal, refunded_quantity, ctime) VALUES
(1001, @TENANT, 1, 1, 'TechPro X20 智能手机', '星空黑,128GB', 'https://picsum.photos/seed/phone1/400/400', 399900, 1, 399900, 0, @NOW - 86400000*10),
(1002, @TENANT, 5, 18, 'StyleWear 商务休闲衬衫', '白色,L', 'https://picsum.photos/seed/shirt1/400/400', 19900, 1, 19900, 0, @NOW - 86400000*8),
(1003, @TENANT, 7, 30, 'FreshBite 每日坚果混合装', '30日装', 'https://picsum.photos/seed/nut1/400/400', 12900, 2, 25800, 0, @NOW - 86400000*3),
(1004, @TENANT, 6, 24, 'StyleWear Air 跑步鞋', '黑白,42', 'https://picsum.photos/seed/shoe1/400/400', 59900, 1, 59900, 0, @NOW - 86400000),
(1005, @TENANT, 4, 15, 'TechPro Watch S3 智能手表', '曜石黑', 'https://picsum.photos/seed/watch1/400/400', 149900, 1, 149900, 0, @NOW - 43200000),
(1006, @TENANT, 1, 2, 'TechPro X20 智能手机', '星空黑,256GB', 'https://picsum.photos/seed/phone1/400/400', 449900, 1, 449900, 0, @NOW - 1800000),
(1007, @TENANT, 8, 32, 'HomeComfort 四件套床上用品', '奶白,1.8m床', 'https://picsum.photos/seed/bed1/400/400', 39900, 1, 39900, 0, @NOW - 86400000*3),
(1008, @TENANT, 5, 20, 'StyleWear 商务休闲衬衫', '浅蓝,L', 'https://picsum.photos/seed/shirt1/400/400', 19900, 1, 19900, 0, @NOW - 86400000*3)
ON DUPLICATE KEY UPDATE ctime = ctime;

-- 4c. 退款单
INSERT INTO refund_orders (tenant_id, order_id, refund_no, buyer_id, type, status, refund_amount, reason, reject_reason, items, ctime, utime) VALUES
(@TENANT, 1008, 'REF20260312001', @BUYER_USER, 2, 1, 19900,
 '尺码不合适，想换大一号', '', '[]',
 @NOW - 86400000, @NOW - 86400000)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 4d. 订单状态日志
INSERT INTO order_status_logs (order_id, from_status, to_status, operator_id, operator_type, remark, ctime) VALUES
(1001, 0, 1, @BUYER_USER,    2, '创建订单',   @NOW - 86400000*10),
(1001, 1, 2, 0,              1, '支付成功',   @NOW - 86400000*10),
(1001, 2, 3, @MERCHANT_USER, 3, '商家发货',   @NOW - 86400000*9),
(1001, 3, 4, @BUYER_USER,    2, '确认收货',   @NOW - 86400000*5),
(1004, 0, 1, @BUYER_USER,    2, '创建订单',   @NOW - 86400000),
(1004, 1, 2, 0,              1, '支付成功',   @NOW - 86400000),
(1006, 0, 1, @BUYER_USER,    2, '创建订单',   @NOW - 1800000),
(1007, 0, 1, @BUYER_USER,    2, '创建订单',   @NOW - 86400000*3),
(1007, 1, 0, 0,              1, '超时未支付', @NOW - 86400000*2),
(1008, 0, 1, @BUYER_USER,    2, '创建订单',   @NOW - 86400000*3),
(1008, 1, 2, 0,              1, '支付成功',   @NOW - 86400000*3),
(1008, 2, 3, @MERCHANT_USER, 3, '商家发货',   @NOW - 86400000*2),
(1008, 3, 5, @BUYER_USER,    2, '申请退款',   @NOW - 86400000);

SELECT '✅ 订单创建完成: 8笔订单(完成2/发货1/待发货2/待付款1/取消1/退款1)' AS result;

-- ============================================================
-- 5. 支付记录
-- ============================================================
USE mall_payment;

INSERT INTO payment_orders (tenant_id, payment_no, order_id, order_no, channel, amount, status, channel_trade_no, pay_time, expire_time, notify_url, ctime, utime) VALUES
(@TENANT, 'PAY20260301001', 1001, 'ORD20260301001', 'wechat',  399900, 2, 'WX4200001234202603010001', @NOW - 86400000*10, 0, '', @NOW - 86400000*10, @NOW - 86400000*10),
(@TENANT, 'PAY20260302001', 1002, 'ORD20260302001', 'alipay',   19900, 2, 'ALI2026030222001234',       @NOW - 86400000*8,  0, '', @NOW - 86400000*8,  @NOW - 86400000*8),
(@TENANT, 'PAY20260310001', 1003, 'ORD20260310001', 'wechat',   25800, 2, 'WX4200001234202603100001', @NOW - 86400000*3,  0, '', @NOW - 86400000*3,  @NOW - 86400000*3),
(@TENANT, 'PAY20260313001', 1004, 'ORD20260313001', 'alipay',   55700, 2, 'ALI2026031322001235',       @NOW - 86400000,    0, '', @NOW - 86400000,    @NOW - 86400000),
(@TENANT, 'PAY20260313002', 1005, 'ORD20260313002', 'wechat',  139900, 2, 'WX4200001234202603130001', @NOW - 43200000,    0, '', @NOW - 43200000,    @NOW - 43200000),
(@TENANT, 'PAY20260311001', 1008, 'ORD20260311001', 'wechat',   19900, 2, 'WX4200001234202603110001', @NOW - 86400000*3,  0, '', @NOW - 86400000*3,  @NOW - 86400000*3)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 退款记录 (退款中的那笔)
INSERT INTO refund_records (tenant_id, payment_no, refund_no, channel, amount, status, channel_refund_no, ctime, utime) VALUES
(@TENANT, 'PAY20260311001', 'REF20260312001', 'wechat', 19900, 1, '', @NOW - 86400000, @NOW - 86400000)
ON DUPLICATE KEY UPDATE utime = @NOW;

SELECT '✅ 支付记录创建完成: 6笔支付 + 1笔退款' AS result;

-- ============================================================
-- 6. 购物车
-- ============================================================
USE mall_cart;

INSERT INTO cart_items (user_id, sku_id, product_id, tenant_id, quantity, selected, ctime, utime) VALUES
(@BUYER_USER, 5,  1, @TENANT, 1, true,  @NOW - 7200000, @NOW - 7200000),
(@BUYER_USER, 30, 7, @TENANT, 3, true,  @NOW - 3600000, @NOW - 3600000),
(@BUYER_USER, 34, 8, @TENANT, 1, false, @NOW - 1800000, @NOW - 1800000)
ON DUPLICATE KEY UPDATE utime = @NOW;

SELECT '✅ 购物车创建完成: 3件商品' AS result;

-- ============================================================
-- 7. 营销: 优惠券 + 秒杀 + 促销
-- ============================================================
USE mall_marketing;

-- 7a. 优惠券
INSERT INTO coupons (id, tenant_id, name, type, threshold, discount_value, total_count, received_count, used_count, per_limit, start_time, end_time, scope_type, scope_ids, status, ctime, utime) VALUES
(1, @TENANT, '新人专享满100减10',
 1, 10000, 1000, 1000, 320, 180, 1,
 @NOW - 86400000*30, @NOW + 86400000*60,
 1, '', 1, @NOW - 86400000*30, @NOW),

(2, @TENANT, '手机数码9折券',
 2, 0, 90, 500, 210, 88, 1,
 @NOW - 86400000*15, @NOW + 86400000*45,
 2, '1', 1, @NOW - 86400000*15, @NOW),

(3, @TENANT, '坚果立减5元',
 3, 0, 500, 2000, 890, 654, 3,
 @NOW - 86400000*20, @NOW + 86400000*40,
 3, '7', 1, @NOW - 86400000*20, @NOW),

(4, @TENANT, '满500减50大额券',
 1, 50000, 5000, 200, 200, 180, 1,
 @NOW - 86400000*60, @NOW - 86400000*5,
 1, '', 0, @NOW - 86400000*60, @NOW - 86400000*5),

(5, @TENANT, '服装鞋帽85折',
 2, 0, 85, 800, 45, 12, 1,
 @NOW - 86400000*3, @NOW + 86400000*27,
 2, '2', 1, @NOW - 86400000*3, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 7b. 用户领取记录
INSERT INTO user_coupons (user_id, coupon_id, tenant_id, status, order_id, receive_time, use_time, ctime, utime) VALUES
(@BUYER_USER, 1, @TENANT, 2, 1004, @NOW - 86400000*5, @NOW - 86400000, @NOW - 86400000*5, @NOW - 86400000),
(@BUYER_USER, 2, @TENANT, 1, 0,    @NOW - 86400000*3, 0,               @NOW - 86400000*3, @NOW - 86400000*3),
(@BUYER_USER, 3, @TENANT, 2, 1003, @NOW - 86400000*10, @NOW - 86400000*3, @NOW - 86400000*10, @NOW - 86400000*3),
(@BUYER_USER, 5, @TENANT, 1, 0,    @NOW - 86400000,   0,               @NOW - 86400000,   @NOW - 86400000)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 7c. 秒杀活动
INSERT INTO seckill_activities (id, tenant_id, name, start_time, end_time, status, ctime, utime) VALUES
(1, @TENANT, '每日11点手机秒杀',
 @NOW, @NOW + 86400000 * 7, 2,
 @NOW - 86400000*2, @NOW),

(2, @TENANT, '周末家居特惠',
 @NOW + 86400000 * 2, @NOW + 86400000 * 4, 1,
 @NOW - 86400000, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 7d. 秒杀商品
INSERT INTO seckill_items (id, activity_id, tenant_id, sku_id, seckill_price, seckill_stock, per_limit) VALUES
(1, 1, @TENANT, 7,  169900, 50, 1),
(2, 1, @TENANT, 9,  169900, 50, 1),
(3, 1, @TENANT, 15, 119900, 30, 1),
(4, 2, @TENANT, 32,  29900, 100, 2),
(5, 2, @TENANT, 35,  29900, 100, 2)
ON DUPLICATE KEY UPDATE seckill_price = VALUES(seckill_price);

-- 7e. 促销规则
INSERT INTO promotion_rules (id, tenant_id, name, type, threshold, discount_value, start_time, end_time, status, ctime, utime) VALUES
(1, @TENANT, '全场满300减30',
 1, 30000, 3000,
 @NOW - 86400000*10, @NOW + 86400000*20,
 2, @NOW - 86400000*10, @NOW),

(2, @TENANT, '数码产品满2000打95折',
 2, 200000, 95,
 @NOW - 86400000*5, @NOW + 86400000*25,
 2, @NOW - 86400000*5, @NOW),

(3, @TENANT, '双倍积分活动',
 1, 0, 0,
 @NOW + 86400000*10, @NOW + 86400000*17,
 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

SELECT '✅ 营销数据创建完成: 5优惠券 + 2秒杀活动 + 3促销规则' AS result;

-- ============================================================
-- 8. 物流: 运费模板 + 发货记录
-- ============================================================
USE mall_logistics;

-- 8a. 运费模板
INSERT INTO freight_templates (id, tenant_id, name, charge_type, free_threshold, ctime, utime) VALUES
(1, @TENANT, '全国包邮',     2, 0,     @NOW, @NOW),
(2, @TENANT, '满99包邮',     2, 9900,  @NOW, @NOW),
(3, @TENANT, '按重量计费',   1, 29900, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 8b. 运费规则
INSERT INTO freight_rules (id, template_id, regions, first_unit, first_price, additional_unit, additional_price) VALUES
(1, 1, '["全国"]',      1, 0,    1, 0),
(2, 2, '["江浙沪"]',    1, 0,    1, 0),
(3, 2, '["全国"]',      1, 800,  1, 300),
(4, 3, '["江浙沪"]',    1, 500,  1, 100),
(5, 3, '["西藏","新疆"]', 1, 1500, 1, 500),
(6, 3, '["全国"]',      1, 800,  1, 200)
ON DUPLICATE KEY UPDATE first_price = VALUES(first_price);

-- 8c. 发货记录
INSERT INTO shipments (tenant_id, order_id, carrier_code, carrier_name, tracking_no, status, ctime, utime) VALUES
(@TENANT, 1001, 'SF',   '顺丰速运', 'SF1234567890001', 4, @NOW - 86400000*9, @NOW - 86400000*5),
(@TENANT, 1002, 'YTO',  '圆通速递', 'YT9876543210001', 4, @NOW - 86400000*7, @NOW - 86400000*3),
(@TENANT, 1003, 'ZTO',  '中通快递', 'ZT1122334455001', 3, @NOW - 86400000*2, @NOW - 86400000),
(@TENANT, 1008, 'SF',   '顺丰速运', 'SF1234567890002', 4, @NOW - 86400000*2, @NOW - 86400000)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 8d. 物流轨迹
SET @SHIP1 = (SELECT id FROM shipments WHERE order_id = 1001 LIMIT 1);
SET @SHIP3 = (SELECT id FROM shipments WHERE order_id = 1003 LIMIT 1);

INSERT INTO shipment_tracks (shipment_id, description, location, track_time) VALUES
(@SHIP1, '快件已签收', '广东省深圳市南山区',   @NOW - 86400000*5),
(@SHIP1, '正在派送中', '广东省深圳市南山区',   @NOW - 86400000*5 - 3600000),
(@SHIP1, '已到达目的地', '深圳转运中心',       @NOW - 86400000*6),
(@SHIP1, '运输中', '广州转运中心',             @NOW - 86400000*7),
(@SHIP1, '已揽件', '浙江省杭州市',             @NOW - 86400000*9),
(@SHIP3, '运输中', '深圳转运中心',             @NOW - 86400000),
(@SHIP3, '已到达', '广州转运中心',             @NOW - 86400000 - 7200000),
(@SHIP3, '运输中', '上海转运中心',             @NOW - 86400000*2 + 3600000),
(@SHIP3, '已揽件', '上海市浦东新区',           @NOW - 86400000*2);

SELECT '✅ 物流数据创建完成: 3运费模板 + 4发货记录 + 物流轨迹' AS result;

-- ============================================================
-- 9. 通知
-- ============================================================
USE mall_notification;

-- 9a. 通知模板
INSERT INTO notification_templates (tenant_id, code, channel, title, content, status, ctime, utime) VALUES
(@TENANT, 'order_paid',       3, '订单支付成功', '您的订单 {{.OrderNo}} 已支付成功，金额 ¥{{.Amount}}，商家将尽快为您发货。', 1, @NOW, @NOW),
(@TENANT, 'order_shipped',    3, '订单已发货',   '您的订单 {{.OrderNo}} 已发货，{{.CarrierName}} 运单号: {{.TrackingNo}}，请注意查收。', 1, @NOW, @NOW),
(@TENANT, 'refund_applied',   3, '退款申请',     '订单 {{.OrderNo}} 有新的退款申请，退款金额 ¥{{.Amount}}，请及时处理。', 1, @NOW, @NOW),
(@TENANT, 'stock_alert',      3, '库存预警',     'SKU {{.SkuCode}} 当前可用库存 {{.Available}}，已低于预警阈值 {{.Threshold}}。', 1, @NOW, @NOW),
(@TENANT, 'coupon_expiring',  3, '优惠券即将过期', '您的优惠券「{{.CouponName}}」将于 {{.ExpireDate}} 过期，赶快使用吧！', 1, @NOW, @NOW)
ON DUPLICATE KEY UPDATE utime = @NOW;

-- 9b. 商家通知 (发给商家管理员)
INSERT INTO notifications (user_id, tenant_id, channel, title, content, is_read, status, ctime, utime) VALUES
(@MERCHANT_USER, @TENANT, 3, '新订单提醒', '您有新的待发货订单 ORD20260313001，买家: 李四，金额: ¥557.00。', false, 1, @NOW - 86400000, @NOW - 86400000),
(@MERCHANT_USER, @TENANT, 3, '新订单提醒', '您有新的待发货订单 ORD20260313002，买家: 张三，金额: ¥1399.00。', false, 1, @NOW - 43200000, @NOW - 43200000),
(@MERCHANT_USER, @TENANT, 3, '退款申请',   '订单 ORD20260311001 收到退款申请，退款金额: ¥199.00，原因: 尺码不合适。', false, 1, @NOW - 86400000, @NOW - 86400000),
(@MERCHANT_USER, @TENANT, 3, '库存预警',   'SKU TP-X20-GD-512 (晨曦金512GB) 当前库存 70 件，低于预警线 10 件。', true, 1, @NOW - 86400000*2, @NOW - 86400000*2),
(@MERCHANT_USER, @TENANT, 3, '订单完成',   '订单 ORD20260301001 已确认收货，交易完成。', true, 1, @NOW - 86400000*5, @NOW - 86400000*5),
(@MERCHANT_USER, @TENANT, 3, '每日经营报告', '昨日新增订单 3 笔，成交金额 ¥6,151.00，退款 0 笔。', true, 1, @NOW - 86400000, @NOW - 86400000);

-- 9c. 买家通知
INSERT INTO notifications (user_id, tenant_id, channel, title, content, is_read, status, ctime, utime) VALUES
(@BUYER_USER, @TENANT, 3, '支付成功', '您的订单 ORD20260313001 已支付成功，金额 ¥557.00，商家将尽快为您发货。', true, 1, @NOW - 86400000, @NOW - 86400000),
(@BUYER_USER, @TENANT, 3, '订单已发货', '您的订单 ORD20260310001 已发货，中通快递 运单号: ZT1122334455001，请注意查收。', false, 1, @NOW - 86400000*2, @NOW - 86400000*2),
(@BUYER_USER, @TENANT, 3, '优惠券即将过期', '您的优惠券「手机数码9折券」还有15天过期，赶快使用吧！', false, 1, @NOW - 86400000, @NOW - 86400000);

SELECT '✅ 通知创建完成: 5模板 + 6商家通知 + 3买家通知' AS result;

-- ============================================================
-- 汇总
-- ============================================================
SELECT '' AS '';
SELECT '════════════════════════════════════════════════════════' AS '';
SELECT '  🎉 商家测试数据初始化完成!' AS '';
SELECT '════════════════════════════════════════════════════════' AS '';
SELECT '' AS '';
SELECT '  📌 商家管理员登录:' AS '';
SELECT '     手机号: 13900000001' AS '';
SELECT '     密码:   merchant123' AS '';
SELECT '     Merchant BFF: POST /api/v1/login' AS '';
SELECT '' AS '';
SELECT '  📌 店员登录:' AS '';
SELECT '     手机号: 13900000002' AS '';
SELECT '     密码:   staff123' AS '';
SELECT '' AS '';
SELECT '  📌 测试买家:' AS '';
SELECT '     手机号: 13800001111' AS '';
SELECT '     密码:   user123' AS '';
SELECT '' AS '';
SELECT '  📌 数据概览:' AS '';
SELECT '     分类: 12 (4顶级 + 8子分类)' AS '';
SELECT '     品牌: 4' AS '';
SELECT '     商品: 8 (含14组规格, 37个SKU)' AS '';
SELECT '     库存: 37个SKU均已初始化' AS '';
SELECT '     订单: 8笔 (多种状态)' AS '';
SELECT '     支付: 6笔 + 1笔退款' AS '';
SELECT '     优惠券: 5张 (含已过期)' AS '';
SELECT '     秒杀: 2场活动 (进行中+即将开始)' AS '';
SELECT '     促销: 3条规则' AS '';
SELECT '     运费模板: 3个 + 6条规则' AS '';
SELECT '     物流: 4笔发货 + 轨迹记录' AS '';
SELECT '     通知: 5模板 + 9条消息' AS '';
SELECT '════════════════════════════════════════════════════════' AS '';
