package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var (
	userDSN   = flag.String("user-dsn", "root:wen123...@tcp(rermrf.icu:3306)/mall_user?charset=utf8mb4&parseTime=true&loc=Local", "mall_user 数据库 DSN")
	tenantDSN = flag.String("tenant-dsn", "root:wen123...@tcp(rermrf.icu:3306)/mall_tenant?charset=utf8mb4&parseTime=true&loc=Local", "mall_tenant 数据库 DSN")

	adminPhone    = flag.String("admin-phone", "13800000000", "平台管理员手机号")
	adminPassword = flag.String("admin-password", "admin123", "平台管理员密码")
	tenantName    = flag.String("tenant-name", "演示商城", "租户名称")
	shopSubdomain = flag.String("subdomain", "shop1", "店铺子域名 (需与 vite.config.ts proxy X-Tenant-Domain 一致)")
	shopDomain    = flag.String("domain", "localhost", "店铺自定义域名 (本地开发用 localhost)")
)

func main() {
	flag.Parse()
	now := time.Now().UnixMilli()

	// ========== 1. Admin User ==========
	log.Println("🔗 连接 mall_user 数据库...")
	userDB, err := sql.Open("mysql", *userDSN)
	if err != nil {
		log.Fatalf("连接 mall_user 失败: %v", err)
	}
	defer userDB.Close()
	if err := userDB.Ping(); err != nil {
		log.Fatalf("ping mall_user 失败: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt 失败: %v", err)
	}

	log.Printf("👤 创建平台管理员 (tenant_id=0, phone=%s)...", *adminPhone)
	_, err = userDB.Exec(`
		INSERT INTO users (tenant_id, phone, email, password, nickname, avatar, status, ctime, utime)
		VALUES (0, ?, '', ?, '平台管理员', '', 1, ?, ?)
		ON DUPLICATE KEY UPDATE password=VALUES(password), utime=VALUES(utime)
	`, *adminPhone, string(hash), now, now)
	if err != nil {
		log.Fatalf("创建 admin 用户失败: %v", err)
	}
	log.Println("✅ Admin 用户创建完成")

	// ========== 2. Tenant + Plan + Shop ==========
	log.Println("🔗 连接 mall_tenant 数据库...")
	tenantDB, err := sql.Open("mysql", *tenantDSN)
	if err != nil {
		log.Fatalf("连接 mall_tenant 失败: %v", err)
	}
	defer tenantDB.Close()
	if err := tenantDB.Ping(); err != nil {
		log.Fatalf("ping mall_tenant 失败: %v", err)
	}

	// 2a. Plan
	log.Println("📋 创建套餐...")
	result, err := tenantDB.Exec(`
		INSERT INTO tenant_plans (name, price, duration_days, max_products, max_staff, features, status, ctime, utime)
		VALUES ('免费试用', 0, 365, 100, 5, '基础店铺功能,商品管理,订单管理', 1, ?, ?)
		ON DUPLICATE KEY UPDATE utime=VALUES(utime)
	`, now, now)
	if err != nil {
		log.Fatalf("创建套餐失败: %v", err)
	}
	planId, _ := result.LastInsertId()
	if planId == 0 {
		// ON DUPLICATE KEY, query it
		row := tenantDB.QueryRow("SELECT id FROM tenant_plans WHERE name='免费试用' LIMIT 1")
		row.Scan(&planId)
	}
	log.Printf("✅ 套餐 ID: %d", planId)

	// 2b. Tenant
	expireTime := time.Now().Add(365 * 24 * time.Hour).UnixMilli()
	log.Printf("🏢 创建租户 '%s'...", *tenantName)
	result, err = tenantDB.Exec(`
		INSERT INTO tenants (name, contact_name, contact_phone, status, plan_id, plan_expire_time, ctime, utime)
		VALUES (?, '管理员', ?, 2, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE utime=VALUES(utime)
	`, *tenantName, *adminPhone, planId, expireTime, now, now)
	if err != nil {
		log.Fatalf("创建租户失败: %v", err)
	}
	tenantId, _ := result.LastInsertId()
	if tenantId == 0 {
		row := tenantDB.QueryRow("SELECT id FROM tenants WHERE name=? LIMIT 1", *tenantName)
		row.Scan(&tenantId)
	}
	log.Printf("✅ 租户 ID: %d (status=2 已审核)", tenantId)

	// 2c. Shop
	log.Printf("🏪 创建店铺 (subdomain=%s, domain=%s)...", *shopSubdomain, *shopDomain)
	_, err = tenantDB.Exec(`
		INSERT INTO shops (tenant_id, name, logo, description, status, rating, subdomain, custom_domain, ctime, utime)
		VALUES (?, ?, '', '开发环境演示店铺', 1, '5.0', ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE subdomain=VALUES(subdomain), custom_domain=VALUES(custom_domain), utime=VALUES(utime)
	`, tenantId, *tenantName, *shopSubdomain, *shopDomain, now, now)
	if err != nil {
		log.Fatalf("创建店铺失败: %v", err)
	}
	log.Println("✅ 店铺创建完成")

	// 2d. Quotas
	log.Println("📊 初始化配额...")
	for _, qt := range []struct{ typ string; max int32 }{{"product_count", 100}, {"staff_count", 5}} {
		_, err = tenantDB.Exec(`
			INSERT INTO tenant_quota_usage (tenant_id, quota_type, used, max_limit, utime)
			VALUES (?, ?, 0, ?, ?)
			ON DUPLICATE KEY UPDATE max_limit=VALUES(max_limit), utime=VALUES(utime)
		`, tenantId, qt.typ, qt.max, now)
		if err != nil {
			log.Fatalf("创建配额 %s 失败: %v", qt.typ, err)
		}
	}
	log.Println("✅ 配额初始化完成")

	// ========== Summary ==========
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════")
	fmt.Println("  🎉 种子数据初始化完成!")
	fmt.Println("════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("  📌 Admin 登录:\n")
	fmt.Printf("     手机号: %s\n", *adminPhone)
	fmt.Printf("     密码:   %s\n", *adminPassword)
	fmt.Printf("     Admin BFF: POST /api/v1/login\n")
	fmt.Println()
	fmt.Printf("  📌 租户信息:\n")
	fmt.Printf("     租户 ID:  %d\n", tenantId)
	fmt.Printf("     租户名称: %s\n", *tenantName)
	fmt.Printf("     套餐 ID:  %d\n", planId)
	fmt.Printf("     状态:     已审核(2)\n")
	fmt.Println()
	fmt.Printf("  📌 店铺信息:\n")
	fmt.Printf("     子域名:     %s\n", *shopSubdomain)
	fmt.Printf("     自定义域名: %s\n", *shopDomain)
	fmt.Println()
	fmt.Println("  📌 Consumer 前端配置:")
	fmt.Println("     方式 1 (推荐): 前端请求添加 Header")
	fmt.Println("       X-Tenant-Domain: localhost")
	fmt.Println()
	fmt.Println("     方式 2: 配置 hosts + 浏览器直接访问")
	fmt.Printf("       echo '127.0.0.1 %s.mall.local' >> /etc/hosts\n", *shopSubdomain)
	fmt.Printf("       然后访问 http://%s.mall.local:5173\n", *shopSubdomain)
	fmt.Println()
	fmt.Println("  📌 Consumer 用户注册:")
	fmt.Println("     启动 consumer-bff + 前端后")
	fmt.Println("     访问 /signup 注册新用户即可")
	fmt.Println("════════════════════════════════════════════════════")
}
