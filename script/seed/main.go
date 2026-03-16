package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

// --------------- CLI flags ---------------

var (
	dsn       = flag.String("dsn", "root:wen123...@tcp(rermrf.icu:3306)/?charset=utf8mb4&parseTime=true&loc=Local", "MySQL DSN (不含数据库名)")
	resetFlag = flag.Bool("reset", false, "先清空所有表数据再初始化")
	tenant    = flag.String("tenant", "演示商城", "租户名称")
	subdomain = flag.String("subdomain", "shop1", "店铺子域名")
	domain    = flag.String("domain", "localhost", "店铺自定义域名")
)

// --------------- globals ---------------

var (
	gCtx  context.Context
	gConn *sql.Conn
)

func main() {
	flag.Parse()
	gCtx = context.Background()

	log.Println("🔗 连接 MySQL...")
	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping 失败: %v", err)
	}

	// Use a single connection so session variables persist across USE statements
	gConn, err = db.Conn(gCtx)
	if err != nil {
		log.Fatalf("获取连接失败: %v", err)
	}
	defer gConn.Close()

	now := time.Now().UnixMilli()
	mustExec(fmt.Sprintf("SET @NOW = %d, @DAY = 86400000", now))

	// ---- Reset (truncate all) ----
	if *resetFlag {
		log.Println("🗑️  清空所有表数据...")
		resetAll()
	}

	// ---- Generate bcrypt hashes ----
	log.Println("🔐 生成密码 hash...")
	adminHash := hashPw("admin123")
	merchantHash := hashPw("merchant123")
	staffHash := hashPw("staff123")
	buyerHash := hashPw("user123")

	// ---- Minimal seed (always runs) ----
	seedTenantDB()
	seedUserDB(adminHash, merchantHash, staffHash, buyerHash)

	// ---- Full test data (only with --reset) ----
	if *resetFlag {
		seedProductDB()
		seedInventoryDB()
		seedOrderDB()
		seedPaymentDB()
		seedCartDB()
		seedMarketingDB()
		seedLogisticsDB()
		seedNotificationDB()
	}

	printSummary()
}

// --------------- helpers ---------------

func mustExec(query string, args ...any) {
	_, err := gConn.ExecContext(gCtx, query, args...)
	if err != nil {
		q := query
		if len(q) > 300 {
			q = q[:300] + "..."
		}
		log.Fatalf("❌ SQL 失败: %v\n   Query: %s", err, q)
	}
}

func hashPw(password string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt 失败: %v", err)
	}
	return string(h)
}

// --------------- reset ---------------

func resetAll() {
	databases := []string{
		"mall_notification", "mall_logistics", "mall_marketing", "mall_cart",
		"mall_payment", "mall_order", "mall_inventory", "mall_product",
		"mall_user", "mall_tenant",
	}
	mustExec("SET FOREIGN_KEY_CHECKS = 0")
	for _, dbName := range databases {
		mustExec("USE " + dbName)
		rows, err := gConn.QueryContext(gCtx, "SHOW TABLES")
		if err != nil {
			log.Printf("⚠️  跳过 %s: %v", dbName, err)
			continue
		}
		var tables []string
		for rows.Next() {
			var t string
			_ = rows.Scan(&t)
			tables = append(tables, t)
		}
		rows.Close()
		for _, t := range tables {
			mustExec("TRUNCATE TABLE `" + t + "`")
		}
		log.Printf("   ✅ %s: 清空 %d 张表", dbName, len(tables))
	}
	mustExec("SET FOREIGN_KEY_CHECKS = 1")
}

// --------------- seed: tenant ---------------

func seedTenantDB() {
	mustExec("USE mall_tenant")
	log.Println("🏢 创建租户...")

	mustExec(`INSERT INTO tenant_plans (name, price, duration_days, max_products, max_staff, features, status, ctime, utime)
		VALUES ('免费试用', 0, 365, 100, 5, '基础店铺功能,商品管理,订单管理', 1, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE utime=VALUES(utime)`)
	mustExec("SET @PLAN = (SELECT id FROM tenant_plans WHERE name='免费试用' LIMIT 1)")

	mustExec(`INSERT INTO tenants (name, contact_name, contact_phone, status, plan_id, plan_expire_time, ctime, utime)
		VALUES (?, '管理员', '13800000000', 2, @PLAN, @NOW + @DAY * 365, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE utime=VALUES(utime)`, *tenant)
	mustExec("SET @T = (SELECT id FROM tenants WHERE name = ? LIMIT 1)", *tenant)

	mustExec(`INSERT INTO shops (tenant_id, name, logo, description, status, rating, subdomain, custom_domain, ctime, utime)
		VALUES (@T, ?, '', '开发环境演示店铺', 1, '5.0', ?, ?, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE subdomain=VALUES(subdomain), custom_domain=VALUES(custom_domain), utime=VALUES(utime)`,
		*tenant, *subdomain, *domain)

	for _, q := range []struct{ typ string; max int }{{"product_count", 100}, {"staff_count", 5}} {
		mustExec(`INSERT INTO tenant_quota_usage (tenant_id, quota_type, used, max_limit, utime)
			VALUES (@T, ?, 0, ?, @NOW)
			ON DUPLICATE KEY UPDATE max_limit=VALUES(max_limit), utime=VALUES(utime)`, q.typ, q.max)
	}
	log.Println("   ✅ 租户/店铺/配额创建完成")
}

// --------------- seed: users ---------------

func seedUserDB(adminHash, merchantHash, staffHash, buyerHash string) {
	mustExec("USE mall_user")
	log.Println("👤 创建用户...")

	// Platform admin (tenant_id=0)
	mustExec(`INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
		VALUES (0, '13800000000', '', ?, '平台管理员', '', 1, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE password=VALUES(password), utime=VALUES(utime)`, adminHash)

	// Merchant admin (tenant_id=@T)
	mustExec(`INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
		VALUES (@T, '13900000001', 'merchant@demo.com', ?, '店铺管理员', '', 1, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE password=VALUES(password), utime=VALUES(utime)`, merchantHash)

	// Staff
	mustExec(`INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
		VALUES (@T, '13900000002', 'staff@demo.com', ?, '客服小王', '', 1, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE password=VALUES(password), utime=VALUES(utime)`, staffHash)

	// Buyer
	mustExec(`INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
		VALUES (@T, '13800001111', 'buyer@demo.com', ?, '测试买家', '', 1, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE password=VALUES(password), utime=VALUES(utime)`, buyerHash)

	// Get actual user IDs
	mustExec("SET @MERCHANT = (SELECT id FROM users WHERE tenant_id = @T AND phone = '13900000001')")
	mustExec("SET @STAFF    = (SELECT id FROM users WHERE tenant_id = @T AND phone = '13900000002')")
	mustExec("SET @BUYER    = (SELECT id FROM users WHERE tenant_id = @T AND phone = '13800001111')")

	// Roles
	mustExec(`INSERT INTO roles (tenant_id, name, code, description, ctime, utime) VALUES
		(@T, '店铺管理员', 'merchant_admin', '拥有全部店铺管理权限', @NOW, @NOW),
		(@T, '客服',       'merchant_cs',    '订单查看、退款处理',   @NOW, @NOW),
		(@T, '运营',       'merchant_ops',   '商品管理、营销管理',   @NOW, @NOW)
		ON DUPLICATE KEY UPDATE utime = @NOW`)
	mustExec("SET @ROLE_ADMIN = (SELECT id FROM roles WHERE tenant_id = @T AND code = 'merchant_admin')")
	mustExec("SET @ROLE_CS    = (SELECT id FROM roles WHERE tenant_id = @T AND code = 'merchant_cs')")

	// Role assignments
	mustExec(`INSERT INTO user_roles (user_id, tenant_id, role_id, ctime, utime) VALUES
		(@MERCHANT, @T, @ROLE_ADMIN, @NOW, @NOW),
		(@STAFF,    @T, @ROLE_CS,    @NOW, @NOW)
		ON DUPLICATE KEY UPDATE utime = @NOW`)

	// Buyer addresses
	mustExec(`INSERT INTO user_addresses (user_id, name, phone, province, city, district, detail, is_default, ctime, utime) VALUES
		(@BUYER, '张三', '13800001111', '广东省', '深圳市', '南山区', '科技园南路88号创业大厦A座1201', true,  @NOW, @NOW),
		(@BUYER, '李四', '13800002222', '北京市', '朝阳区', '三里屯', '工体北路甲2号盈科中心3层',     false, @NOW, @NOW)
		ON DUPLICATE KEY UPDATE utime = @NOW`)

	log.Println("   ✅ 用户/角色/地址创建完成")
}

// --------------- seed: products ---------------

func seedProductDB() {
	mustExec("USE mall_product")
	log.Println("📦 创建商品体系...")

	// Categories (2-level)
	mustExec(`INSERT INTO categories (id, tenant_id, parent_id, name, level, sort, icon, status, ctime, utime) VALUES
		(1,  @T, 0, '手机数码', 1, 1, '', 1, @NOW, @NOW),
		(2,  @T, 0, '服装鞋帽', 1, 2, '', 1, @NOW, @NOW),
		(3,  @T, 0, '食品饮料', 1, 3, '', 1, @NOW, @NOW),
		(4,  @T, 0, '家居百货', 1, 4, '', 1, @NOW, @NOW),
		(5,  @T, 1, '智能手机', 2, 1, '', 1, @NOW, @NOW),
		(6,  @T, 1, '平板电脑', 2, 2, '', 1, @NOW, @NOW),
		(7,  @T, 1, '智能手表', 2, 3, '', 1, @NOW, @NOW),
		(8,  @T, 2, '男装',     2, 1, '', 1, @NOW, @NOW),
		(9,  @T, 2, '女装',     2, 2, '', 1, @NOW, @NOW),
		(10, @T, 2, '运动鞋',   2, 3, '', 1, @NOW, @NOW),
		(11, @T, 3, '零食',     2, 1, '', 1, @NOW, @NOW),
		(12, @T, 3, '饮品',     2, 2, '', 1, @NOW, @NOW)`)

	// Brands
	mustExec(`INSERT INTO brands (id, tenant_id, name, logo, status, ctime, utime) VALUES
		(1, @T, 'TechPro',     '', 1, @NOW, @NOW),
		(2, @T, 'StyleWear',   '', 1, @NOW, @NOW),
		(3, @T, 'FreshBite',   '', 1, @NOW, @NOW),
		(4, @T, 'HomeComfort', '', 1, @NOW, @NOW)`)

	// Products (8)
	mustExec(`INSERT INTO products (id, tenant_id, category_id, brand_id, name, subtitle, main_image, images, description, status, sales, ctime, utime) VALUES
		(1, @T, 5, 1, 'TechPro X20 智能手机',
		 '旗舰芯片 | 1亿像素 | 5000mAh 超长续航',
		 'https://picsum.photos/seed/phone1/400/400',
		 '["https://picsum.photos/seed/phone1a/800/800","https://picsum.photos/seed/phone1b/800/800"]',
		 '搭载最新旗舰芯片，1亿像素主摄，支持100W快充，5000mAh超大电池，120Hz AMOLED屏幕。',
		 1, 328, @NOW, @NOW),
		(2, @T, 5, 1, 'TechPro Lite 轻薄手机',
		 '轻薄设计 | 6400万像素 | 快充',
		 'https://picsum.photos/seed/phone2/400/400',
		 '["https://picsum.photos/seed/phone2a/800/800"]',
		 '仅7.5mm超薄机身，158g轻盈手感，6400万像素AI三摄。',
		 1, 156, @NOW, @NOW),
		(3, @T, 6, 1, 'TechPro Pad 11 平板电脑',
		 '11英寸2K屏 | 骁龙处理器 | 手写笔',
		 'https://picsum.photos/seed/pad1/400/400',
		 '["https://picsum.photos/seed/pad1a/800/800"]',
		 '11英寸2K IPS屏幕，骁龙8系芯片，8GB+256GB，支持手写笔和键盘套件。',
		 1, 89, @NOW, @NOW),
		(4, @T, 7, 1, 'TechPro Watch S3 智能手表',
		 '全天候健康监测 | GPS | 14天续航',
		 'https://picsum.photos/seed/watch1/400/400',
		 '[]',
		 '1.43英寸AMOLED圆表盘，血氧/心率/睡眠监测，100+运动模式。',
		 1, 212, @NOW, @NOW),
		(5, @T, 8, 2, 'StyleWear 商务休闲衬衫',
		 '免烫面料 | 修身版型 | 春秋款',
		 'https://picsum.photos/seed/shirt1/400/400',
		 '["https://picsum.photos/seed/shirt1a/800/800"]',
		 '60支免烫面料，修身剪裁，适合商务和日常穿搭。',
		 1, 567, @NOW, @NOW),
		(6, @T, 10, 2, 'StyleWear Air 跑步鞋',
		 '超轻缓震 | 透气网面 | 碳板加持',
		 'https://picsum.photos/seed/shoe1/400/400',
		 '["https://picsum.photos/seed/shoe1a/800/800"]',
		 '全新碳板中底，回弹率85%，透气飞织鞋面，仅220g。',
		 1, 891, @NOW, @NOW),
		(7, @T, 11, 3, 'FreshBite 每日坚果混合装',
		 '7种坚果 | 独立包装 | 30日量',
		 'https://picsum.photos/seed/nut1/400/400',
		 '[]',
		 '精选7种优质坚果: 巴旦木、腰果、核桃仁、榛子、蔓越莓、蓝莓干、南瓜子。每日一袋，营养均衡。',
		 1, 2345, @NOW, @NOW),
		(8, @T, 4, 4, 'HomeComfort 四件套床上用品',
		 '60支长绒棉 | 纯色简约 | 多色可选',
		 'https://picsum.photos/seed/bed1/400/400',
		 '["https://picsum.photos/seed/bed1a/800/800"]',
		 '60支新疆长绒棉，活性印染不掉色，包含被套*1、床单*1、枕套*2。',
		 1, 433, @NOW, @NOW)`)

	// Product specs (14)
	mustExec("INSERT INTO product_specs (id, product_id, tenant_id, name, `values`) VALUES " +
		"(1,  1, @T, '颜色', '星空黑,冰川蓝,晨曦金')," +
		"(2,  1, @T, '存储', '128GB,256GB,512GB')," +
		"(3,  2, @T, '颜色', '珍珠白,薄荷绿')," +
		"(4,  2, @T, '存储', '128GB,256GB')," +
		"(5,  3, @T, '颜色', '深空灰,银色')," +
		"(6,  3, @T, '配置', '8+128GB,8+256GB')," +
		"(7,  4, @T, '颜色', '曜石黑,雾霾蓝,樱花粉')," +
		"(8,  5, @T, '颜色', '白色,浅蓝,灰色')," +
		"(9,  5, @T, '尺码', 'M,L,XL,2XL')," +
		"(10, 6, @T, '颜色', '黑白,全黑,荧光绿')," +
		"(11, 6, @T, '尺码', '39,40,41,42,43,44')," +
		"(12, 7, @T, '规格', '30日装,15日装')," +
		"(13, 8, @T, '颜色', '奶白,浅灰,雾蓝,豆沙粉')," +
		"(14, 8, @T, '尺寸', '1.5m床,1.8m床,2.0m床')")

	// SKUs (37)
	mustExec(`INSERT INTO product_skus (id, tenant_id, product_id, spec_values, price, original_price, cost_price, sku_code, bar_code, status, ctime, utime) VALUES
		-- TechPro X20
		(1,  @T, 1, '星空黑,128GB', 399900, 449900, 280000, 'TP-X20-BK-128', '6901234000001', 1, @NOW, @NOW),
		(2,  @T, 1, '星空黑,256GB', 449900, 499900, 310000, 'TP-X20-BK-256', '6901234000002', 1, @NOW, @NOW),
		(3,  @T, 1, '冰川蓝,128GB', 399900, 449900, 280000, 'TP-X20-BL-128', '6901234000003', 1, @NOW, @NOW),
		(4,  @T, 1, '冰川蓝,256GB', 449900, 499900, 310000, 'TP-X20-BL-256', '6901234000004', 1, @NOW, @NOW),
		(5,  @T, 1, '晨曦金,256GB', 459900, 509900, 320000, 'TP-X20-GD-256', '6901234000005', 1, @NOW, @NOW),
		(6,  @T, 1, '晨曦金,512GB', 519900, 569900, 360000, 'TP-X20-GD-512', '6901234000006', 1, @NOW, @NOW),
		-- TechPro Lite
		(7,  @T, 2, '珍珠白,128GB', 199900, 229900, 140000, 'TP-LT-WH-128', '6901234000007', 1, @NOW, @NOW),
		(8,  @T, 2, '珍珠白,256GB', 229900, 259900, 160000, 'TP-LT-WH-256', '6901234000008', 1, @NOW, @NOW),
		(9,  @T, 2, '薄荷绿,128GB', 199900, 229900, 140000, 'TP-LT-GN-128', '6901234000009', 1, @NOW, @NOW),
		(10, @T, 2, '薄荷绿,256GB', 229900, 259900, 160000, 'TP-LT-GN-256', '6901234000010', 1, @NOW, @NOW),
		-- TechPro Pad 11
		(11, @T, 3, '深空灰,8+128GB', 269900, 299900, 190000, 'TP-PAD-GR-128', '6901234000011', 1, @NOW, @NOW),
		(12, @T, 3, '深空灰,8+256GB', 319900, 349900, 220000, 'TP-PAD-GR-256', '6901234000012', 1, @NOW, @NOW),
		(13, @T, 3, '银色,8+128GB',   269900, 299900, 190000, 'TP-PAD-SL-128', '6901234000013', 1, @NOW, @NOW),
		(14, @T, 3, '银色,8+256GB',   319900, 349900, 220000, 'TP-PAD-SL-256', '6901234000014', 1, @NOW, @NOW),
		-- TechPro Watch S3
		(15, @T, 4, '曜石黑', 149900, 179900, 90000, 'TP-WS3-BK', '6901234000015', 1, @NOW, @NOW),
		(16, @T, 4, '雾霾蓝', 149900, 179900, 90000, 'TP-WS3-BL', '6901234000016', 1, @NOW, @NOW),
		(17, @T, 4, '樱花粉', 149900, 179900, 90000, 'TP-WS3-PK', '6901234000017', 1, @NOW, @NOW),
		-- 商务休闲衬衫
		(18, @T, 5, '白色,L',   19900, 29900, 8000, 'SW-SH-WH-L',  '6901234000018', 1, @NOW, @NOW),
		(19, @T, 5, '白色,XL',  19900, 29900, 8000, 'SW-SH-WH-XL', '6901234000019', 1, @NOW, @NOW),
		(20, @T, 5, '浅蓝,L',   19900, 29900, 8000, 'SW-SH-BL-L',  '6901234000020', 1, @NOW, @NOW),
		(21, @T, 5, '浅蓝,XL',  19900, 29900, 8000, 'SW-SH-BL-XL', '6901234000021', 1, @NOW, @NOW),
		(22, @T, 5, '灰色,M',   19900, 29900, 8000, 'SW-SH-GR-M',  '6901234000022', 1, @NOW, @NOW),
		(23, @T, 5, '灰色,2XL', 19900, 29900, 8000, 'SW-SH-GR-2X', '6901234000023', 1, @NOW, @NOW),
		-- 跑步鞋
		(24, @T, 6, '黑白,42',   59900, 79900, 25000, 'SW-RN-BW-42', '6901234000024', 1, @NOW, @NOW),
		(25, @T, 6, '黑白,43',   59900, 79900, 25000, 'SW-RN-BW-43', '6901234000025', 1, @NOW, @NOW),
		(26, @T, 6, '全黑,42',   59900, 79900, 25000, 'SW-RN-BK-42', '6901234000026', 1, @NOW, @NOW),
		(27, @T, 6, '荧光绿,41', 62900, 79900, 28000, 'SW-RN-GN-41', '6901234000027', 1, @NOW, @NOW),
		(28, @T, 6, '荧光绿,42', 62900, 79900, 28000, 'SW-RN-GN-42', '6901234000028', 1, @NOW, @NOW),
		(29, @T, 6, '荧光绿,43', 62900, 79900, 28000, 'SW-RN-GN-43', '6901234000029', 1, @NOW, @NOW),
		-- 每日坚果
		(30, @T, 7, '30日装', 12900, 16900, 6000, 'FB-NUT-30', '6901234000030', 1, @NOW, @NOW),
		(31, @T, 7, '15日装',  6900,  8900, 3200, 'FB-NUT-15', '6901234000031', 1, @NOW, @NOW),
		-- 四件套
		(32, @T, 8, '奶白,1.8m床',   39900, 59900, 18000, 'HC-BD-WH-18', '6901234000032', 1, @NOW, @NOW),
		(33, @T, 8, '浅灰,1.8m床',   39900, 59900, 18000, 'HC-BD-GR-18', '6901234000033', 1, @NOW, @NOW),
		(34, @T, 8, '雾蓝,1.5m床',   36900, 56900, 16000, 'HC-BD-BL-15', '6901234000034', 1, @NOW, @NOW),
		(35, @T, 8, '雾蓝,1.8m床',   39900, 59900, 18000, 'HC-BD-BL-18', '6901234000035', 1, @NOW, @NOW),
		(36, @T, 8, '豆沙粉,1.8m床', 39900, 59900, 18000, 'HC-BD-PK-18', '6901234000036', 1, @NOW, @NOW),
		(37, @T, 8, '豆沙粉,2.0m床', 42900, 62900, 20000, 'HC-BD-PK-20', '6901234000037', 1, @NOW, @NOW)`)

	log.Println("   ✅ 商品体系: 12分类, 4品牌, 8商品, 14规格组, 37 SKU")
}

// --------------- seed: inventory ---------------

func seedInventoryDB() {
	mustExec("USE mall_inventory")
	log.Println("📊 创建库存...")

	mustExec(`INSERT INTO inventories (tenant_id, sku_id, total, available, locked, sold, alert_threshold, ctime, utime) VALUES
		(@T, 1,  500, 420, 5, 75, 50, @NOW, @NOW),
		(@T, 2,  300, 230, 3, 67, 30, @NOW, @NOW),
		(@T, 3,  500, 430, 2, 68, 50, @NOW, @NOW),
		(@T, 4,  300, 248, 4, 48, 30, @NOW, @NOW),
		(@T, 5,  200, 158, 2, 40, 20, @NOW, @NOW),
		(@T, 6,  100,  70, 0, 30, 10, @NOW, @NOW),
		(@T, 7,  600, 530, 6, 64, 50, @NOW, @NOW),
		(@T, 8,  400, 355, 3, 42, 40, @NOW, @NOW),
		(@T, 9,  600, 548, 2, 50, 50, @NOW, @NOW),
		(@T, 10, 400, 370, 0, 30, 40, @NOW, @NOW),
		(@T, 11, 200, 165, 2, 33, 20, @NOW, @NOW),
		(@T, 12, 150, 118, 1, 31, 15, @NOW, @NOW),
		(@T, 13, 200, 180, 0, 20, 20, @NOW, @NOW),
		(@T, 14, 150, 140, 1,  9, 15, @NOW, @NOW),
		(@T, 15, 800, 650, 10, 140, 80, @NOW, @NOW),
		(@T, 16, 600, 530,  5,  65, 60, @NOW, @NOW),
		(@T, 17, 400, 390,  3,   7, 40, @NOW, @NOW),
		(@T, 18, 1000, 800, 10, 190, 100, @NOW, @NOW),
		(@T, 19, 1000, 850,  5, 145, 100, @NOW, @NOW),
		(@T, 20,  800, 680,  8, 112,  80, @NOW, @NOW),
		(@T, 21,  800, 720,  0,  80,  80, @NOW, @NOW),
		(@T, 22,  600, 570,  2,  28,  60, @NOW, @NOW),
		(@T, 23,  500, 488,  0,  12,  50, @NOW, @NOW),
		(@T, 24, 500, 350, 8, 142, 50, @NOW, @NOW),
		(@T, 25, 500, 380, 5, 115, 50, @NOW, @NOW),
		(@T, 26, 400, 290, 3, 107, 40, @NOW, @NOW),
		(@T, 27, 300, 210, 6,  84, 30, @NOW, @NOW),
		(@T, 28, 300, 180, 4, 116, 30, @NOW, @NOW),
		(@T, 29, 300, 230, 2,  68, 30, @NOW, @NOW),
		(@T, 30, 5000, 3200, 50, 1750, 500, @NOW, @NOW),
		(@T, 31, 3000, 2350, 20,  630, 300, @NOW, @NOW),
		(@T, 32, 600, 450, 5, 145, 60, @NOW, @NOW),
		(@T, 33, 600, 480, 3, 117, 60, @NOW, @NOW),
		(@T, 34, 400, 330, 2,  68, 40, @NOW, @NOW),
		(@T, 35, 600, 460, 4, 136, 60, @NOW, @NOW),
		(@T, 36, 500, 430, 3,  67, 50, @NOW, @NOW),
		(@T, 37, 300, 270, 0,  30, 30, @NOW, @NOW)`)

	mustExec(`INSERT INTO inventory_logs (sku_id, order_id, type, quantity, before_available, after_available, tenant_id, ctime) VALUES
		(1,  0,    1, 500,  0,    500,  @T, @NOW - @DAY * 30),
		(30, 0,    1, 5000, 0,    5000, @T, @NOW - @DAY * 30),
		(1,  1001, 2, 1,    421,  420,  @T, @NOW - 3600000),
		(30, 1003, 2, 2,    3202, 3200, @T, @NOW - 1800000),
		(24, 0,    3, 100,  250,  350,  @T, @NOW - 7200000)`)

	log.Println("   ✅ 库存: 37 SKU + 5条变更日志")
}

// --------------- seed: orders ---------------

func seedOrderDB() {
	mustExec("USE mall_order")
	log.Println("📋 创建订单...")

	mustExec(`INSERT INTO orders (id, tenant_id, order_no, buyer_id, buyer_hash, status,
		total_amount, discount_amount, freight_amount, pay_amount, refunded_amount,
		coupon_id, payment_no, receiver_name, receiver_phone, receiver_address,
		remark, paid_at, shipped_at, received_at, closed_at, ctime, utime) VALUES
		-- 已完成 x2
		(1001, @T, 'ORD20260301001', @BUYER, 'hash_001', 4,
		 399900, 0, 0, 399900, 0, 0, 'PAY20260301001',
		 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '请尽快发货',
		 @NOW-@DAY*10, @NOW-@DAY*9, @NOW-@DAY*5, 0, @NOW-@DAY*10, @NOW-@DAY*5),
		(1002, @T, 'ORD20260302001', @BUYER, 'hash_002', 4,
		 19900, 0, 0, 19900, 0, 0, 'PAY20260302001',
		 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '',
		 @NOW-@DAY*8, @NOW-@DAY*7, @NOW-@DAY*3, 0, @NOW-@DAY*8, @NOW-@DAY*3),
		-- 已发货
		(1003, @T, 'ORD20260310001', @BUYER, 'hash_003', 3,
		 25800, 0, 0, 25800, 0, 0, 'PAY20260310001',
		 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '送货前打电话',
		 @NOW-@DAY*3, @NOW-@DAY*2, 0, 0, @NOW-@DAY*3, @NOW-@DAY*2),
		-- 待发货 x2
		(1004, @T, 'ORD20260313001', @BUYER, 'hash_004', 2,
		 59900, 5000, 800, 55700, 0, 1, 'PAY20260313001',
		 '李四', '13800002222', '北京市朝阳区三里屯工体北路甲2号', '',
		 @NOW-@DAY, 0, 0, 0, @NOW-@DAY, @NOW-@DAY),
		(1005, @T, 'ORD20260313002', @BUYER, 'hash_005', 2,
		 149900, 10000, 0, 139900, 0, 0, 'PAY20260313002',
		 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '生日礼物',
		 @NOW-43200000, 0, 0, 0, @NOW-43200000, @NOW-43200000),
		-- 待付款
		(1006, @T, 'ORD20260314001', @BUYER, 'hash_006', 1,
		 449900, 0, 0, 449900, 0, 0, '',
		 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '',
		 0, 0, 0, 0, @NOW-1800000, @NOW-1800000),
		-- 已取消
		(1007, @T, 'ORD20260312001', @BUYER, 'hash_007', 0,
		 39900, 0, 0, 39900, 0, 0, '',
		 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '',
		 0, 0, 0, @NOW-@DAY*2, @NOW-@DAY*3, @NOW-@DAY*2),
		-- 退款中
		(1008, @T, 'ORD20260311001', @BUYER, 'hash_008', 5,
		 19900, 0, 0, 19900, 0, 0, 'PAY20260311001',
		 '张三', '13800001111', '广东省深圳市南山区科技园南路88号', '尺码不合适',
		 @NOW-@DAY*3, @NOW-@DAY*2, @NOW-@DAY, 0, @NOW-@DAY*3, @NOW-@DAY)`)

	// Order items
	mustExec(`INSERT INTO order_items (order_id, tenant_id, product_id, sku_id, product_name, sku_spec, image, price, quantity, subtotal, refunded_quantity, ctime) VALUES
		(1001, @T, 1, 1,  'TechPro X20 智能手机',       '星空黑,128GB', 'https://picsum.photos/seed/phone1/400/400', 399900, 1, 399900, 0, @NOW-@DAY*10),
		(1002, @T, 5, 18, 'StyleWear 商务休闲衬衫',      '白色,L',      'https://picsum.photos/seed/shirt1/400/400',  19900, 1,  19900, 0, @NOW-@DAY*8),
		(1003, @T, 7, 30, 'FreshBite 每日坚果混合装',     '30日装',      'https://picsum.photos/seed/nut1/400/400',    12900, 2,  25800, 0, @NOW-@DAY*3),
		(1004, @T, 6, 24, 'StyleWear Air 跑步鞋',        '黑白,42',     'https://picsum.photos/seed/shoe1/400/400',   59900, 1,  59900, 0, @NOW-@DAY),
		(1005, @T, 4, 15, 'TechPro Watch S3 智能手表',    '曜石黑',      'https://picsum.photos/seed/watch1/400/400', 149900, 1, 149900, 0, @NOW-43200000),
		(1006, @T, 1, 2,  'TechPro X20 智能手机',        '星空黑,256GB', 'https://picsum.photos/seed/phone1/400/400', 449900, 1, 449900, 0, @NOW-1800000),
		(1007, @T, 8, 32, 'HomeComfort 四件套床上用品',    '奶白,1.8m床', 'https://picsum.photos/seed/bed1/400/400',    39900, 1,  39900, 0, @NOW-@DAY*3),
		(1008, @T, 5, 20, 'StyleWear 商务休闲衬衫',      '浅蓝,L',      'https://picsum.photos/seed/shirt1/400/400',  19900, 1,  19900, 0, @NOW-@DAY*3)`)

	// Refund order
	mustExec(`INSERT INTO refund_orders (tenant_id, order_id, refund_no, buyer_id, type, status, refund_amount, reason, reject_reason, items, ctime, utime) VALUES
		(@T, 1008, 'REF20260312001', @BUYER, 2, 1, 19900, '尺码不合适，想换大一号', '', '[]', @NOW-@DAY, @NOW-@DAY)`)

	// Order status logs
	mustExec(`INSERT INTO order_status_logs (order_id, from_status, to_status, operator_id, operator_type, remark, ctime) VALUES
		(1001, 0, 1, @BUYER,    2, '创建订单',   @NOW-@DAY*10),
		(1001, 1, 2, 0,         1, '支付成功',   @NOW-@DAY*10),
		(1001, 2, 3, @MERCHANT, 3, '商家发货',   @NOW-@DAY*9),
		(1001, 3, 4, @BUYER,    2, '确认收货',   @NOW-@DAY*5),
		(1004, 0, 1, @BUYER,    2, '创建订单',   @NOW-@DAY),
		(1004, 1, 2, 0,         1, '支付成功',   @NOW-@DAY),
		(1006, 0, 1, @BUYER,    2, '创建订单',   @NOW-1800000),
		(1007, 0, 1, @BUYER,    2, '创建订单',   @NOW-@DAY*3),
		(1007, 1, 0, 0,         1, '超时未支付', @NOW-@DAY*2),
		(1008, 0, 1, @BUYER,    2, '创建订单',   @NOW-@DAY*3),
		(1008, 1, 2, 0,         1, '支付成功',   @NOW-@DAY*3),
		(1008, 2, 3, @MERCHANT, 3, '商家发货',   @NOW-@DAY*2),
		(1008, 3, 5, @BUYER,    2, '申请退款',   @NOW-@DAY)`)

	log.Println("   ✅ 订单: 8笔(完成2/发货1/待发货2/待付款1/取消1/退款1)")
}

// --------------- seed: payment ---------------

func seedPaymentDB() {
	mustExec("USE mall_payment")
	log.Println("💳 创建支付记录...")

	mustExec(`INSERT INTO payment_orders (tenant_id, payment_no, order_id, order_no, channel, amount, status, channel_trade_no, pay_time, expire_time, notify_url, ctime, utime) VALUES
		(@T, 'PAY20260301001', 1001, 'ORD20260301001', 'wechat',  399900, 2, 'WX4200001234202603010001', @NOW-@DAY*10, 0, '', @NOW-@DAY*10, @NOW-@DAY*10),
		(@T, 'PAY20260302001', 1002, 'ORD20260302001', 'alipay',   19900, 2, 'ALI2026030222001234',       @NOW-@DAY*8,  0, '', @NOW-@DAY*8,  @NOW-@DAY*8),
		(@T, 'PAY20260310001', 1003, 'ORD20260310001', 'wechat',   25800, 2, 'WX4200001234202603100001', @NOW-@DAY*3,  0, '', @NOW-@DAY*3,  @NOW-@DAY*3),
		(@T, 'PAY20260313001', 1004, 'ORD20260313001', 'alipay',   55700, 2, 'ALI2026031322001235',       @NOW-@DAY,    0, '', @NOW-@DAY,    @NOW-@DAY),
		(@T, 'PAY20260313002', 1005, 'ORD20260313002', 'wechat',  139900, 2, 'WX4200001234202603130001', @NOW-43200000, 0, '', @NOW-43200000, @NOW-43200000),
		(@T, 'PAY20260311001', 1008, 'ORD20260311001', 'wechat',   19900, 2, 'WX4200001234202603110001', @NOW-@DAY*3,  0, '', @NOW-@DAY*3,  @NOW-@DAY*3)`)

	mustExec(`INSERT INTO refund_records (tenant_id, payment_no, refund_no, channel, amount, status, channel_refund_no, ctime, utime) VALUES
		(@T, 'PAY20260311001', 'REF20260312001', 'wechat', 19900, 1, '', @NOW-@DAY, @NOW-@DAY)`)

	log.Println("   ✅ 支付: 6笔 + 1笔退款")
}

// --------------- seed: cart ---------------

func seedCartDB() {
	mustExec("USE mall_cart")
	log.Println("🛒 创建购物车...")

	mustExec(`INSERT INTO cart_items (user_id, sku_id, product_id, tenant_id, quantity, selected, ctime, utime) VALUES
		(@BUYER, 5,  1, @T, 1, true,  @NOW-7200000,  @NOW-7200000),
		(@BUYER, 30, 7, @T, 3, true,  @NOW-3600000,  @NOW-3600000),
		(@BUYER, 34, 8, @T, 1, false, @NOW-1800000,  @NOW-1800000)`)

	log.Println("   ✅ 购物车: 3件商品")
}

// --------------- seed: marketing ---------------

func seedMarketingDB() {
	mustExec("USE mall_marketing")
	log.Println("🎯 创建营销数据...")

	// Coupons
	mustExec(`INSERT INTO coupons (id, tenant_id, name, type, threshold, discount_value, total_count, received_count, used_count, per_limit, start_time, end_time, scope_type, scope_ids, status, ctime, utime) VALUES
		(1, @T, '新人专享满100减10', 1, 10000, 1000, 1000, 320, 180, 1, @NOW-@DAY*30, @NOW+@DAY*60, 1, '', 1, @NOW-@DAY*30, @NOW),
		(2, @T, '手机数码9折券',     2, 0,     90,   500,  210,  88, 1, @NOW-@DAY*15, @NOW+@DAY*45, 2, '1', 1, @NOW-@DAY*15, @NOW),
		(3, @T, '坚果立减5元',       3, 0,     500,  2000, 890, 654, 3, @NOW-@DAY*20, @NOW+@DAY*40, 3, '7', 1, @NOW-@DAY*20, @NOW),
		(4, @T, '满500减50大额券',   1, 50000, 5000, 200,  200, 180, 1, @NOW-@DAY*60, @NOW-@DAY*5,  1, '', 0, @NOW-@DAY*60, @NOW-@DAY*5),
		(5, @T, '服装鞋帽85折',      2, 0,     85,   800,   45,  12, 1, @NOW-@DAY*3,  @NOW+@DAY*27, 2, '2', 1, @NOW-@DAY*3, @NOW)`)

	// User coupons
	mustExec(`INSERT INTO user_coupons (user_id, coupon_id, tenant_id, status, order_id, receive_time, use_time, ctime, utime) VALUES
		(@BUYER, 1, @T, 2, 1004, @NOW-@DAY*5,  @NOW-@DAY,   @NOW-@DAY*5,  @NOW-@DAY),
		(@BUYER, 2, @T, 1, 0,    @NOW-@DAY*3,  0,           @NOW-@DAY*3,  @NOW-@DAY*3),
		(@BUYER, 3, @T, 2, 1003, @NOW-@DAY*10, @NOW-@DAY*3, @NOW-@DAY*10, @NOW-@DAY*3),
		(@BUYER, 5, @T, 1, 0,    @NOW-@DAY,    0,           @NOW-@DAY,    @NOW-@DAY)`)

	// Seckill activities
	mustExec(`INSERT INTO seckill_activities (id, tenant_id, name, start_time, end_time, status, ctime, utime) VALUES
		(1, @T, '每日11点手机秒杀', @NOW, @NOW+@DAY*7, 2, @NOW-@DAY*2, @NOW),
		(2, @T, '周末家居特惠',     @NOW+@DAY*2, @NOW+@DAY*4, 1, @NOW-@DAY, @NOW)`)

	// Seckill items
	mustExec(`INSERT INTO seckill_items (id, activity_id, tenant_id, sku_id, seckill_price, seckill_stock, per_limit) VALUES
		(1, 1, @T, 7,  169900, 50,  1),
		(2, 1, @T, 9,  169900, 50,  1),
		(3, 1, @T, 15, 119900, 30,  1),
		(4, 2, @T, 32,  29900, 100, 2),
		(5, 2, @T, 35,  29900, 100, 2)`)

	// Promotion rules
	mustExec(`INSERT INTO promotion_rules (id, tenant_id, name, type, threshold, discount_value, start_time, end_time, status, ctime, utime) VALUES
		(1, @T, '全场满300减30',       1, 30000,  3000, @NOW-@DAY*10, @NOW+@DAY*20, 2, @NOW-@DAY*10, @NOW),
		(2, @T, '数码产品满2000打95折', 2, 200000, 95,   @NOW-@DAY*5,  @NOW+@DAY*25, 2, @NOW-@DAY*5,  @NOW),
		(3, @T, '双倍积分活动',         1, 0,      0,    @NOW+@DAY*10, @NOW+@DAY*17, 1, @NOW,         @NOW)`)

	log.Println("   ✅ 营销: 5优惠券 + 2秒杀 + 3促销")
}

// --------------- seed: logistics ---------------

func seedLogisticsDB() {
	mustExec("USE mall_logistics")
	log.Println("🚚 创建物流数据...")

	// Freight templates
	mustExec(`INSERT INTO freight_templates (id, tenant_id, name, charge_type, free_threshold, ctime, utime) VALUES
		(1, @T, '全国包邮',   2, 0,     @NOW, @NOW),
		(2, @T, '满99包邮',   2, 9900,  @NOW, @NOW),
		(3, @T, '按重量计费', 1, 29900, @NOW, @NOW)`)

	// Freight rules
	mustExec(`INSERT INTO freight_rules (id, template_id, regions, first_unit, first_price, additional_unit, additional_price) VALUES
		(1, 1, '["全国"]',        1, 0,    1, 0),
		(2, 2, '["江浙沪"]',      1, 0,    1, 0),
		(3, 2, '["全国"]',        1, 800,  1, 300),
		(4, 3, '["江浙沪"]',      1, 500,  1, 100),
		(5, 3, '["西藏","新疆"]', 1, 1500, 1, 500),
		(6, 3, '["全国"]',        1, 800,  1, 200)`)

	// Shipments
	mustExec(`INSERT INTO shipments (tenant_id, order_id, carrier_code, carrier_name, tracking_no, status, ctime, utime) VALUES
		(@T, 1001, 'SF',  '顺丰速运', 'SF1234567890001', 4, @NOW-@DAY*9, @NOW-@DAY*5),
		(@T, 1002, 'YTO', '圆通速递', 'YT9876543210001', 4, @NOW-@DAY*7, @NOW-@DAY*3),
		(@T, 1003, 'ZTO', '中通快递', 'ZT1122334455001', 3, @NOW-@DAY*2, @NOW-@DAY),
		(@T, 1008, 'SF',  '顺丰速运', 'SF1234567890002', 4, @NOW-@DAY*2, @NOW-@DAY)`)

	// Shipment tracks (use subquery for shipment IDs)
	mustExec("SET @SHIP1 = (SELECT id FROM shipments WHERE order_id = 1001 LIMIT 1)")
	mustExec("SET @SHIP3 = (SELECT id FROM shipments WHERE order_id = 1003 LIMIT 1)")

	mustExec(`INSERT INTO shipment_tracks (shipment_id, description, location, track_time) VALUES
		(@SHIP1, '快件已签收',   '广东省深圳市南山区', @NOW-@DAY*5),
		(@SHIP1, '正在派送中',   '广东省深圳市南山区', @NOW-@DAY*5-3600000),
		(@SHIP1, '已到达目的地', '深圳转运中心',       @NOW-@DAY*6),
		(@SHIP1, '运输中',       '广州转运中心',       @NOW-@DAY*7),
		(@SHIP1, '已揽件',       '浙江省杭州市',       @NOW-@DAY*9),
		(@SHIP3, '运输中',       '深圳转运中心',       @NOW-@DAY),
		(@SHIP3, '已到达',       '广州转运中心',       @NOW-@DAY-7200000),
		(@SHIP3, '运输中',       '上海转运中心',       @NOW-@DAY*2+3600000),
		(@SHIP3, '已揽件',       '上海市浦东新区',     @NOW-@DAY*2)`)

	log.Println("   ✅ 物流: 3运费模板 + 4发货 + 9条轨迹")
}

// --------------- seed: notifications ---------------

func seedNotificationDB() {
	mustExec("USE mall_notification")
	log.Println("🔔 创建通知...")

	// Templates
	mustExec(`INSERT INTO notification_templates (tenant_id, code, channel, title, content, status, ctime, utime) VALUES
		(@T, 'order_paid',      3, '订单支付成功', '您的订单 {{.OrderNo}} 已支付成功，金额 ¥{{.Amount}}，商家将尽快为您发货。', 1, @NOW, @NOW),
		(@T, 'order_shipped',   3, '订单已发货',   '您的订单 {{.OrderNo}} 已发货，{{.CarrierName}} 运单号: {{.TrackingNo}}，请注意查收。', 1, @NOW, @NOW),
		(@T, 'refund_applied',  3, '退款申请',     '订单 {{.OrderNo}} 有新的退款申请，退款金额 ¥{{.Amount}}，请及时处理。', 1, @NOW, @NOW),
		(@T, 'stock_alert',     3, '库存预警',     'SKU {{.SkuCode}} 当前可用库存 {{.Available}}，已低于预警阈值 {{.Threshold}}。', 1, @NOW, @NOW),
		(@T, 'coupon_expiring', 3, '优惠券即将过期', '您的优惠券「{{.CouponName}}」将于 {{.ExpireDate}} 过期，赶快使用吧！', 1, @NOW, @NOW)`)

	// Merchant notifications
	mustExec(`INSERT INTO notifications (user_id, tenant_id, channel, title, content, is_read, status, ctime, utime) VALUES
		(@MERCHANT, @T, 3, '新订单提醒',   '您有新的待发货订单 ORD20260313001，买家: 李四，金额: ¥557.00。',                  false, 1, @NOW-@DAY,      @NOW-@DAY),
		(@MERCHANT, @T, 3, '新订单提醒',   '您有新的待发货订单 ORD20260313002，买家: 张三，金额: ¥1399.00。',                false, 1, @NOW-43200000,  @NOW-43200000),
		(@MERCHANT, @T, 3, '退款申请',     '订单 ORD20260311001 收到退款申请，退款金额: ¥199.00，原因: 尺码不合适。',         false, 1, @NOW-@DAY,      @NOW-@DAY),
		(@MERCHANT, @T, 3, '库存预警',     'SKU TP-X20-GD-512 (晨曦金512GB) 当前库存 70 件，低于预警线 10 件。',            true,  1, @NOW-@DAY*2,    @NOW-@DAY*2),
		(@MERCHANT, @T, 3, '订单完成',     '订单 ORD20260301001 已确认收货，交易完成。',                                     true,  1, @NOW-@DAY*5,    @NOW-@DAY*5),
		(@MERCHANT, @T, 3, '每日经营报告', '昨日新增订单 3 笔，成交金额 ¥6,151.00，退款 0 笔。',                             true,  1, @NOW-@DAY,      @NOW-@DAY)`)

	// Buyer notifications
	mustExec(`INSERT INTO notifications (user_id, tenant_id, channel, title, content, is_read, status, ctime, utime) VALUES
		(@BUYER, @T, 3, '支付成功',       '您的订单 ORD20260313001 已支付成功，金额 ¥557.00，商家将尽快为您发货。',                  true,  1, @NOW-@DAY,   @NOW-@DAY),
		(@BUYER, @T, 3, '订单已发货',     '您的订单 ORD20260310001 已发货，中通快递 运单号: ZT1122334455001，请注意查收。',          false, 1, @NOW-@DAY*2, @NOW-@DAY*2),
		(@BUYER, @T, 3, '优惠券即将过期', '您的优惠券「手机数码9折券」还有15天过期，赶快使用吧！',                                   false, 1, @NOW-@DAY,   @NOW-@DAY)`)

	log.Println("   ✅ 通知: 5模板 + 9条消息")
}

// --------------- summary ---------------

func printSummary() {
	mode := "最小数据 (仅登录账号)"
	if *resetFlag {
		mode = "完整测试数据 (已清空重建)"
	}

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("  🎉 种子数据初始化完成!")
	fmt.Printf("  📌 模式: %s\n", mode)
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  🏪 商家管理员 (merchant-frontend 登录):")
	fmt.Println("     手机号:    13900000001")
	fmt.Println("     密码:      merchant123")
	fmt.Println("     商户ID:    1 (登录页默认)")
	fmt.Println("     前端地址:  http://localhost:3001")
	fmt.Println()
	fmt.Println("  👤 商家店员:")
	fmt.Println("     手机号:    13900000002")
	fmt.Println("     密码:      staff123")
	fmt.Println()
	fmt.Println("  🛍️  消费者 (consumer frontend 登录):")
	fmt.Println("     手机号:    13800001111")
	fmt.Println("     密码:      user123")
	fmt.Println("     前端地址:  http://localhost:3000")
	fmt.Println()
	fmt.Println("  🔧 平台管理员 (预留，暂无前端):")
	fmt.Println("     手机号:    13800000000")
	fmt.Println("     密码:      admin123")
	fmt.Println("     tenant_id: 0")
	fmt.Println()

	if *resetFlag {
		fmt.Println("  📊 数据概览:")
		fmt.Println("     分类: 12 (4顶级 + 8子分类)")
		fmt.Println("     品牌: 4")
		fmt.Println("     商品: 8 (14规格组, 37 SKU)")
		fmt.Println("     库存: 37 SKU 已初始化")
		fmt.Println("     订单: 8笔 (完成2/发货1/待发货2/待付款1/取消1/退款1)")
		fmt.Println("     支付: 6笔 + 1笔退款")
		fmt.Println("     购物车: 3件")
		fmt.Println("     优惠券: 5张 + 秒杀2场 + 促销3条")
		fmt.Println("     物流: 3模板 + 4发货 + 轨迹")
		fmt.Println("     通知: 5模板 + 9条消息")
		fmt.Println()
	}

	fmt.Println("  💡 用法:")
	fmt.Println("     初始化登录账号: go run ./script/seed/")
	fmt.Println("     清空+完整数据:  go run ./script/seed/ -reset")
	fmt.Println("     make seed       (等同于 go run ./script/seed/)")
	fmt.Println("     make seed-reset (等同于 go run ./script/seed/ -reset)")
	fmt.Println("════════════════════════════════════════════════════════")
}
