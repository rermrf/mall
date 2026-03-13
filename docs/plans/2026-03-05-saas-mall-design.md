# SaaS 多租户商城系统设计文档

> 基于 Go 微服务架构的 SaaS 电商平台，支持平台管理端、商家端、C 端三端分离。

---

## 1. 设计决策摘要

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 多租户隔离 | 共享数据库 + tenant_id 字段隔离 | 实现简单，适合面试讲解，展示中间件拦截和数据隔离 |
| 架构模式 | 纯领域服务 + 编舞模式（Choreography） | 服务间松耦合，Kafka 事件驱动，与 template.md 模板完美契合 |
| BFF 拆分 | 3 个独立 BFF（admin / merchant / consumer） | 职责清晰，独立部署，独立鉴权 |
| C 端模式 | 独立商城 SaaS（类似 Shopify/有赞） | 每个商家独立 C 端入口，域名级租户隔离 |
| 用户隔离 | 租户级用户隔离 | users 表携带 tenant_id，同一手机号在不同店铺是不同用户 |
| 服务粒度 | 细粒度（11 个微服务） | 面试亮点最多，每个服务有明确技术深度 |
| 前端范围 | 仅规划后端 | 前端另开项目 |

---

## 2. 技术栈

| 类别 | 技术选型 |
|------|---------|
| 语言 | Go 1.23 |
| HTTP 框架 | Gin |
| ORM | GORM (MySQL) |
| 缓存 | Redis (go-redis/v9) |
| RPC | gRPC + Protocol Buffers |
| 消息队列 | Kafka (Sarama) |
| 服务注册/发现 | etcd |
| 依赖注入 | Google Wire |
| 配置管理 | Viper + YAML |
| 日志 | Zap (结构化日志) |
| 监控 | Prometheus + Grafana |
| 链路追踪 | OpenTelemetry + Zipkin |
| 搜索引擎 | Elasticsearch |
| 数据同步 | Canal (MySQL Binlog CDC) |
| 容器化 | Docker + Kubernetes |
| Proto 生成 | Buf |
| Mock 生成 | mockgen |

---

## 3. 整体架构

```
                        +------------------+
                        |    Nginx / LB    |
                        +--------+---------+
               +------------------+------------------+
               |                  |                  |
        +------v------+   +------v------+   +-------v-----+
        | admin-bff   |   |merchant-bff |   |consumer-bff |
        | (平台管理端) |   | (商家端)     |   | (C端)       |
        +------+------+   +------+------+   +-------+-----+
               |        gRPC     |                   |
        +------+------------------+------------------+------+
        |                    etcd (服务发现)                  |
        +------+------------------+------------------+------+
               |                  |                  |
  +------------v------------------v------------------v-----------+
  |                       gRPC 微服务层                           |
  |                                                              |
  |  user-svc      tenant-svc     product-svc    inventory-svc  |
  |  order-svc     payment-svc    cart-svc       search-svc     |
  |  marketing-svc logistics-svc  notification-svc              |
  +-------+-------------+-------------+-------------+-----------+
          |             |             |             |
    +-----v---+  +------v--+  +------v--+  +------v---+
    | MySQL   |  | Redis   |  | Kafka   |  |    ES    |
    +---------+  +---------+  +---------+  +----------+
```

### 通信规则

- **BFF → 微服务**：gRPC 同步调用
- **微服务 → 微服务**：gRPC 同步（查询）/ Kafka 异步（命令/事件）
- **服务注册发现**：所有服务通过 etcd 注册
- **三端认证**：各 BFF 独立 JWT 签发与验证，共享 user-svc

### 数据库规划

- 每个微服务独立数据库（MySQL 实例共享，库名隔离）
- 库名规范：`mall_user`、`mall_tenant`、`mall_product`、`mall_inventory`、`mall_order`、`mall_payment`、`mall_cart`、`mall_search`、`mall_marketing`、`mall_logistics`、`mall_notification`
- 所有业务表携带 `tenant_id` 字段，BFF 中间件自动注入

### 全局公共字段

所有表统一包含：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键自增 |
| tenant_id | bigint | 租户ID（商家ID） |
| ctime | bigint | 创建时间（毫秒时间戳） |
| utime | bigint | 更新时间（毫秒时间戳） |

---

## 4. 服务清单总览

| # | 服务名 | 库名 | 核心职责 | 面试亮点 |
|---|--------|------|----------|----------|
| 1 | user-svc | mall_user | 用户认证、RBAC 权限 | JWT 双 Token + Redis 黑名单、多租户 RBAC |
| 2 | tenant-svc | mall_tenant | 商家入驻、套餐配额 | SaaS 套餐配额控制 |
| 3 | product-svc | mall_product | 商品 SPU/SKU 管理 | SPU/SKU 两层模型、动态规格 |
| 4 | inventory-svc | mall_inventory | 库存扣减、预扣回滚 | Redis+Lua 原子扣减、预扣/确认/回滚三阶段 |
| 5 | order-svc | mall_order | 订单状态机、超时关单 | 状态机、Kafka 延迟队列关单、雪花ID |
| 6 | payment-svc | mall_payment | 支付对接、回调幂等 | 支付回调幂等性、Strategy 模式多渠道 |
| 7 | cart-svc | mall_cart | 购物车 | Redis Hash 主存储 |
| 8 | search-svc | mall_search | ES 搜索、数据同步 | Canal Binlog CDC → Kafka → ES |
| 9 | marketing-svc | mall_marketing | 优惠券、秒杀、满减 | 秒杀全链路：限流→Lua扣减→Kafka削峰 |
| 10 | logistics-svc | mall_logistics | 运费、物流追踪 | 运费模板区域差异化 |
| 11 | notification-svc | mall_notification | 短信/邮件/站内信 | Kafka 消费者统一发送 |
| - | admin-bff | - | 平台管理端 HTTP 网关 | 平台级 RBAC |
| - | merchant-bff | - | 商家端 HTTP 网关 | 商家级 RBAC |
| - | consumer-bff | - | C 端 HTTP 网关 | 限流、秒杀入口 |

---

## 5. 各服务详细设计

### 5.1 user-svc（用户服务）

**职责：** 用户注册/登录、多端身份认证、RBAC 权限管理

#### 核心功能

| 功能 | 说明 |
|------|------|
| 用户注册/登录 | 手机号+验证码、邮箱+密码、OAuth2（微信/Google） |
| JWT 管理 | 双 Token（access_token + refresh_token），Redis 存储黑名单 |
| 多端身份区分 | 同一用户可以是消费者，也可以是商家管理员或平台管理员 |
| RBAC 权限 | 角色-权限模型，平台管理端和商家端各自独立的角色体系 |
| 用户画像基础 | 基础信息、收货地址管理 |

#### 数据库表（库：`mall_user`）

**users 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 所属租户 |
| phone | varchar(20) | 手机号（唯一索引：tenant_id + phone） |
| email | varchar(100) | 邮箱（唯一索引：tenant_id + email） |
| password | varchar(255) | bcrypt 加密密码 |
| nickname | varchar(50) | 昵称 |
| avatar | varchar(500) | 头像 URL |
| status | tinyint | 1-正常 2-冻结 3-注销 |
| ctime / utime | bigint | 毫秒时间戳 |

**user_roles 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户ID |
| tenant_id | bigint | 租户ID（0=平台级角色） |
| role_id | bigint | 角色ID |
| ctime / utime | bigint | |

**roles 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 0=平台角色，其他=商家自定义角色 |
| name | varchar(50) | 角色名 |
| code | varchar(50) | 角色编码（唯一索引：tenant_id + code） |
| description | varchar(200) | 描述 |
| ctime / utime | bigint | |

**role_permissions 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| role_id | bigint | 角色ID |
| permission_id | bigint | 权限ID |
| ctime | bigint | |

**permissions 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| code | varchar(100) | 权限编码（如 product:create, order:view） |
| name | varchar(100) | 权限名 |
| type | tinyint | 1-菜单 2-按钮 3-API |
| resource | varchar(200) | 资源标识 |
| ctime / utime | bigint | |

**user_addresses 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户ID |
| name | varchar(50) | 收件人姓名 |
| phone | varchar(20) | 收件人手机号 |
| province / city / district | varchar(50) | 省市区 |
| detail | varchar(200) | 详细地址 |
| is_default | tinyint | 是否默认地址 |
| ctime / utime | bigint | |

**oauth_accounts 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 关联用户ID |
| tenant_id | bigint | 所属租户 |
| provider | varchar(20) | 提供方（wechat/google/github） |
| provider_uid | varchar(100) | 第三方用户ID |
| access_token | varchar(500) | 第三方 access_token |
| ctime / utime | bigint | |
| | | 唯一索引：tenant_id + provider + provider_uid |

#### Redis 缓存

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `user:info:{id}` | JSON(User) | 15min | 用户信息缓存 |
| `user:token:blacklist:{jti}` | "1" | token 剩余时间 | JWT 黑名单 |
| `user:permission:{uid}:{tid}` | JSON([]string) | 10min | 权限列表缓存 |
| `user:code:{tenant_id}:{phone}` | 验证码 | 5min | 短信验证码 |

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `user_registered` | 生产 | notification-svc, marketing-svc | 新用户注册 |
| `user_login` | 生产 | 审计 | 登录事件 |

---

### 5.2 tenant-svc（租户/商家服务）

**职责：** 商家入驻、店铺管理、租户配额与套餐管理

#### 核心功能

| 功能 | 说明 |
|------|------|
| 商家入驻 | 提交营业执照等资料，平台审核 |
| 店铺管理 | 店铺信息、LOGO、营业状态 |
| 套餐管理 | 不同 SaaS 套餐（基础版/专业版/旗舰版） |
| 配额控制 | 商品数量上限、员工数上限 |

#### 数据库表（库：`mall_tenant`）

**tenants 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键（即 tenant_id） |
| name | varchar(100) | 商家名称 |
| contact_name | varchar(50) | 联系人 |
| contact_phone | varchar(20) | 联系电话 |
| business_license | varchar(500) | 营业执照 URL |
| status | tinyint | 1-待审核 2-正常 3-冻结 4-注销 |
| plan_id | bigint | 当前套餐ID |
| plan_expire_time | bigint | 套餐到期时间 |
| ctime / utime | bigint | |

**tenant_plans 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| name | varchar(50) | 套餐名 |
| price | bigint | 价格（分） |
| duration_days | int | 套餐时长（天） |
| max_products | int | 最大商品数 |
| max_staff | int | 最大员工数 |
| features | text | JSON 功能列表 |
| status | tinyint | 1-启用 2-停用 |
| ctime / utime | bigint | |

**tenant_quota_usage 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| quota_type | varchar(30) | 配额类型 |
| used | int | 已使用量 |
| max_limit | int | 上限 |
| utime | bigint | |
| | | 唯一索引：tenant_id + quota_type |

**shops 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| name | varchar(100) | 店铺名 |
| logo | varchar(500) | 店铺 LOGO |
| description | text | 店铺描述 |
| status | tinyint | 1-营业中 2-休息中 3-关闭 |
| rating | decimal(2,1) | 店铺评分 |
| ctime / utime | bigint | |

#### Redis 缓存

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `tenant:info:{id}` | JSON(Tenant) | 30min | 租户信息 |
| `tenant:quota:{tid}:{type}` | int | 10min | 配额使用量 |
| `shop:info:{id}` | JSON(Shop) | 15min | 店铺信息 |

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `tenant_approved` | 生产 | user-svc, notification-svc | 商家审核通过 |
| `tenant_plan_changed` | 生产 | product-svc | 套餐变更 |

---

### 5.3 product-svc（商品服务）

**职责：** 商品管理、SPU/SKU、分类、品牌、规格

#### 核心功能

| 功能 | 说明 |
|------|------|
| 商品 CRUD | SPU + SKU 两层模型 |
| 分类管理 | 三级分类树 |
| 品牌管理 | 品牌 CRUD |
| 商品规格 | 动态属性 |
| 上下架 | 状态管理 |

#### 数据库表（库：`mall_product`）

**categories 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 0=平台分类 |
| parent_id | bigint | 父分类ID |
| name | varchar(50) | 分类名 |
| level | tinyint | 层级（1/2/3） |
| sort | int | 排序权重 |
| icon | varchar(500) | 分类图标 |
| status | tinyint | 1-启用 2-停用 |
| ctime / utime | bigint | |

**brands 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| name | varchar(100) | 品牌名 |
| logo | varchar(500) | 品牌 LOGO |
| status | tinyint | 1-启用 2-停用 |
| ctime / utime | bigint | |

**products 表（SPU）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| category_id | bigint | 分类ID |
| brand_id | bigint | 品牌ID |
| name | varchar(200) | 商品名称 |
| subtitle | varchar(500) | 副标题 |
| main_image | varchar(500) | 主图 URL |
| images | text | 图片列表 JSON |
| description | text | 商品详情（富文本） |
| status | tinyint | 1-草稿 2-上架 3-下架 |
| sales | bigint | 累计销量 |
| ctime / utime | bigint | |

**product_skus 表（SKU）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| product_id | bigint | SPU ID |
| spec_values | varchar(500) | 规格值组合 JSON |
| price | bigint | 价格（分） |
| original_price | bigint | 原价（分） |
| cost_price | bigint | 成本价（分） |
| sku_code | varchar(100) | SKU 编码 |
| bar_code | varchar(100) | 条形码 |
| status | tinyint | 1-启用 2-停用 |
| ctime / utime | bigint | |

**product_specs 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| product_id | bigint | SPU ID |
| name | varchar(50) | 规格名 |
| values | text | 规格值列表 JSON |
| ctime / utime | bigint | |

#### Redis 缓存

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `product:info:{id}` | JSON(Product+SKUs) | 15min | 商品详情缓存 |
| `product:category:tree:{tid}` | JSON(CategoryTree) | 30min | 分类树缓存 |

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `product_status_changed` | 生产 | search-svc | 上下架同步 ES |
| `product_updated` | 生产 | search-svc, cart-svc | 信息变更同步 |

---

### 5.4 inventory-svc（库存服务）

**职责：** 库存管理、高并发扣减、预扣/确认/回滚

#### 核心功能

| 功能 | 说明 |
|------|------|
| 库存设置 | 各 SKU 库存管理 |
| 库存预扣 | Redis+Lua 原子操作 |
| 库存确认 | 支付成功后 MySQL 落库 |
| 库存回滚 | 超时/取消订单回滚 |
| 库存预警 | 低于阈值通知商家 |

#### 数据库表（库：`mall_inventory`）

**inventories 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| sku_id | bigint | SKU ID（唯一索引：tenant_id + sku_id） |
| total | int | 总库存 |
| available | int | 可售库存 |
| locked | int | 锁定库存 |
| sold | int | 已售库存 |
| alert_threshold | int | 预警阈值 |
| ctime / utime | bigint | |

**inventory_logs 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| sku_id | bigint | SKU ID |
| order_id | bigint | 关联订单ID |
| type | tinyint | 1-预扣 2-确认 3-回滚 4-手动调整 |
| quantity | int | 变更数量 |
| before_available | int | 变更前可售 |
| after_available | int | 变更后可售 |
| ctime | bigint | |

#### Redis 设计

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `inventory:stock:{sku_id}` | int | 不过期 | **Redis+Lua 原子扣减** |
| `inventory:lock:{order_id}` | JSON([]SKU+Qty) | 30min | 预扣锁定记录 |

#### Redis+Lua 库存预扣脚本

```lua
-- KEYS[1]: inventory:stock:{sku_id}
-- ARGV[1]: 扣减数量
local stock = tonumber(redis.call('GET', KEYS[1]))
if stock == nil then return -1 end
if stock < tonumber(ARGV[1]) then return -2 end
redis.call('DECRBY', KEYS[1], ARGV[1])
return stock - tonumber(ARGV[1])
```

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `inventory_deducted` | 生产 | order-svc | 库存预扣成功 |
| `inventory_alert` | 生产 | notification-svc | 库存预警 |
| `order_paid` | 消费 | payment-svc | 确认扣减 |
| `order_cancelled` | 消费 | order-svc | 回滚库存 |

---

### 5.5 order-svc（订单服务）

**职责：** 订单创建、状态机流转、超时关单、售后

#### 订单状态机

```
  pending ──支付成功──> paid ──商家发货──> shipped ──确认收货──> received ──评价──> completed
     │                  │
     │超时/取消          │退款
     v                  v
  cancelled          refunded
```

#### 数据库表（库：`mall_order`）

**orders 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| order_no | varchar(32) | 订单号（唯一索引，雪花算法） |
| buyer_id | bigint | 买家用户ID |
| status | tinyint | 订单状态 |
| total_amount | bigint | 总金额（分） |
| discount_amount | bigint | 优惠金额（分） |
| freight_amount | bigint | 运费（分） |
| pay_amount | bigint | 实付金额（分） |
| coupon_id | bigint | 优惠券ID |
| payment_id | bigint | 支付单ID |
| receiver_name | varchar(50) | 收件人 |
| receiver_phone | varchar(20) | 收件人手机 |
| receiver_address | varchar(500) | 收件地址 |
| remark | varchar(500) | 买家备注 |
| pay_time | bigint | 支付时间 |
| ship_time | bigint | 发货时间 |
| receive_time | bigint | 收货时间 |
| close_time | bigint | 关闭时间 |
| ctime / utime | bigint | |

**order_items 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| order_id | bigint | 订单ID |
| product_id | bigint | SPU ID |
| sku_id | bigint | SKU ID |
| product_name | varchar(200) | 商品名（快照） |
| sku_spec | varchar(500) | 规格值（快照） |
| product_image | varchar(500) | 商品图片（快照） |
| price | bigint | 单价（分） |
| quantity | int | 购买数量 |
| total_amount | bigint | 小计（分） |
| ctime | bigint | |

**order_status_logs 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| order_id | bigint | 订单ID |
| from_status | tinyint | 原状态 |
| to_status | tinyint | 新状态 |
| operator_id | bigint | 操作人ID |
| operator_type | tinyint | 1-买家 2-商家 3-平台 4-系统 |
| remark | varchar(200) | 备注 |
| ctime | bigint | |

**refund_orders 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| order_id | bigint | 原订单ID |
| refund_no | varchar(32) | 退款单号 |
| buyer_id | bigint | 买家ID |
| type | tinyint | 1-仅退款 2-退货退款 |
| status | tinyint | 1-待审核 2-审核通过 3-退款中 4-已退款 5-已拒绝 |
| refund_amount | bigint | 退款金额（分） |
| reason | varchar(500) | 退款原因 |
| ctime / utime | bigint | |

#### Redis 缓存

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `order:info:{order_no}` | JSON(Order) | 15min | 订单详情 |
| `order:create:lock:{buyer_id}` | "1" | 5s | 防重复提交 |

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `order_created` | 生产 | inventory-svc, marketing-svc | 创建订单 |
| `order_close_delay` | 生产/消费 | self | **延迟队列**：30min 超时关单 |
| `order_cancelled` | 生产 | inventory-svc, marketing-svc | 取消订单 |
| `order_completed` | 生产 | product-svc | 更新销量 |

#### 超时关单（Kafka 延迟队列）

```
下单 → 发送延迟消息 order_close_delay (delay=30min)
     → Consumer 消费
     → 查订单状态: pending? → 关闭订单 → 发 order_cancelled
                   paid?    → 忽略
```

---

### 5.6 payment-svc（支付服务）

**职责：** 支付对接、回调处理、幂等性、退款

#### 数据库表（库：`mall_payment`）

**payment_orders 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| payment_no | varchar(32) | 支付单号（唯一索引） |
| order_id | bigint | 业务订单ID |
| order_no | varchar(32) | 业务订单号 |
| channel | varchar(20) | 支付渠道（wechat/alipay/mock） |
| amount | bigint | 支付金额（分） |
| status | tinyint | 1-待支付 2-支付中 3-已支付 4-已关闭 5-退款中 6-已退款 |
| channel_trade_no | varchar(100) | 第三方流水号 |
| pay_time | bigint | 支付时间 |
| expire_time | bigint | 过期时间 |
| notify_url | varchar(500) | 回调地址 |
| ctime / utime | bigint | |

**payment_notify_logs 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| payment_no | varchar(32) | 支付单号 |
| channel | varchar(20) | 渠道 |
| notify_body | text | 回调原始报文 |
| status | tinyint | 1-成功 2-失败 |
| ctime | bigint | |

**refund_records 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| payment_no | varchar(32) | 原支付单号 |
| refund_no | varchar(32) | 退款单号（唯一索引） |
| channel | varchar(20) | 渠道 |
| amount | bigint | 退款金额（分） |
| status | tinyint | 1-退款中 2-已退款 3-退款失败 |
| channel_refund_no | varchar(100) | 第三方退款流水号 |
| ctime / utime | bigint | |

#### Redis 缓存

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `payment:idempotent:{payment_no}` | 处理状态 | 24h | 回调幂等 |
| `payment:channel:config:{channel}` | JSON(Config) | 30min | 渠道配置 |

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `order_paid` | 生产 | order-svc, inventory-svc | 支付成功 |
| `refund_completed` | 生产 | order-svc | 退款完成 |

---

### 5.7 cart-svc（购物车服务）

**职责：** 购物车管理（Redis 为主存储）

#### 数据库表（库：`mall_cart`）

**cart_items 表（MySQL 持久化兜底）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户ID |
| tenant_id | bigint | 租户ID |
| sku_id | bigint | SKU ID |
| product_id | bigint | SPU ID |
| quantity | int | 数量 |
| selected | tinyint | 是否选中 |
| ctime / utime | bigint | |
| | | 唯一索引：user_id + sku_id |

#### Redis 设计（主存储）

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `cart:{user_id}` | Hash(sku_id → JSON) | 30天 | 购物车主存储 |

Redis Hash 为主存储，MySQL 异步持久化兜底。

---

### 5.8 search-svc（搜索服务）

**职责：** ES 搜索、MySQL Binlog 数据同步、搜索建议

#### ES 索引（products）

```json
{
  "mappings": {
    "properties": {
      "id":            { "type": "long" },
      "tenant_id":     { "type": "long" },
      "name":          { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
      "subtitle":      { "type": "text", "analyzer": "ik_max_word" },
      "category_id":   { "type": "long" },
      "category_name": { "type": "keyword" },
      "brand_id":      { "type": "long" },
      "brand_name":    { "type": "keyword" },
      "price":         { "type": "long" },
      "sales":         { "type": "long" },
      "main_image":    { "type": "keyword", "index": false },
      "status":        { "type": "integer" },
      "shop_id":       { "type": "long" },
      "shop_name":     { "type": "keyword" },
      "suggest":       { "type": "completion" },
      "ctime":         { "type": "long" },
      "utime":         { "type": "long" }
    }
  }
}
```

#### 数据库表（库：`mall_search`）

**search_hot_words 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| word | varchar(100) | 搜索词 |
| count | bigint | 搜索次数 |
| status | tinyint | 1-展示 2-屏蔽 |
| ctime / utime | bigint | |

**search_history 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户ID |
| keyword | varchar(100) | 搜索关键词 |
| ctime | bigint | |

#### 数据同步架构

```
MySQL (products) → Canal (CDC) → Kafka (product_binlog) → search-svc → ES
```

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `product_binlog` | 消费 | Canal | Binlog CDC 同步 |
| `product_status_changed` | 消费 | product-svc | 上下架同步 |

---

### 5.9 marketing-svc（营销服务）

**职责：** 优惠券、秒杀活动、满减规则

#### 数据库表（库：`mall_marketing`）

**coupons 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| name | varchar(100) | 优惠券名 |
| type | tinyint | 1-满减 2-折扣 3-无门槛 |
| threshold | bigint | 使用门槛（分） |
| discount_value | bigint | 优惠值 |
| total_count | int | 发放总量 |
| received_count | int | 已领取数 |
| used_count | int | 已使用数 |
| per_limit | int | 每人限领 |
| start_time | bigint | 有效期开始 |
| end_time | bigint | 有效期结束 |
| scope_type | tinyint | 1-全店 2-指定分类 3-指定商品 |
| scope_ids | text | 适用范围 JSON |
| status | tinyint | 1-未开始 2-进行中 3-已结束 4-已停用 |
| ctime / utime | bigint | |

**user_coupons 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户ID |
| coupon_id | bigint | 优惠券ID |
| tenant_id | bigint | 租户ID |
| status | tinyint | 1-未使用 2-已使用 3-已过期 |
| order_id | bigint | 使用的订单ID |
| receive_time | bigint | 领取时间 |
| use_time | bigint | 使用时间 |
| ctime / utime | bigint | |

**seckill_activities 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| name | varchar(100) | 活动名称 |
| start_time | bigint | 开始时间 |
| end_time | bigint | 结束时间 |
| status | tinyint | 1-未开始 2-进行中 3-已结束 |
| ctime / utime | bigint | |

**seckill_items 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| activity_id | bigint | 活动ID |
| sku_id | bigint | SKU ID |
| seckill_price | bigint | 秒杀价（分） |
| seckill_stock | int | 秒杀库存 |
| per_limit | int | 每人限购 |
| ctime / utime | bigint | |

**promotion_rules 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| name | varchar(100) | 规则名 |
| type | tinyint | 1-满减 2-满折 |
| threshold | bigint | 门槛金额（分） |
| discount_value | bigint | 优惠值 |
| start_time | bigint | 开始时间 |
| end_time | bigint | 结束时间 |
| status | tinyint | 1-启用 2-停用 |
| ctime / utime | bigint | |

#### Redis 缓存

| Key | Value | TTL | 用途 |
|-----|-------|-----|------|
| `seckill:stock:{item_id}` | int | 活动结束 | 秒杀库存原子扣减 |
| `seckill:bought:{item_id}:{uid}` | int | 活动结束 | 限购控制 |
| `coupon:received:{coupon_id}:{uid}` | int | 券有效期 | 领券限制 |

#### 秒杀核心流程

```
用户抢购 → BFF 限流（令牌桶）
         → marketing-svc.Seckill(uid, item_id)
         → Redis Lua: 检查限购 + 扣减秒杀库存
         → 成功 → Kafka seckill_order_created（异步下单，削峰）
         → 失败 → 返回"已抢完"
```

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `user_registered` | 消费 | user-svc | 新用户发新人券 |
| `seckill_order_created` | 生产 | order-svc | 秒杀异步下单 |
| `order_cancelled` | 消费 | order-svc | 释放优惠券 |

---

### 5.10 logistics-svc（物流服务）

**职责：** 运费模板、运费计算、物流追踪

#### 数据库表（库：`mall_logistics`）

**freight_templates 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| name | varchar(100) | 模板名 |
| charge_type | tinyint | 1-按件 2-按重量 |
| free_threshold | bigint | 包邮门槛（分） |
| ctime / utime | bigint | |

**freight_rules 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| template_id | bigint | 模板ID |
| regions | text | 适用地区 JSON |
| first_unit | int | 首件/首重 |
| first_price | bigint | 首费（分） |
| additional_unit | int | 续件/续重 |
| additional_price | bigint | 续费（分） |
| ctime / utime | bigint | |

**shipments 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 租户ID |
| order_id | bigint | 订单ID |
| carrier_code | varchar(20) | 物流公司编码 |
| carrier_name | varchar(50) | 物流公司名 |
| tracking_no | varchar(50) | 物流单号 |
| status | tinyint | 1-已发货 2-运输中 3-已签收 |
| ctime / utime | bigint | |

**shipment_tracks 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| shipment_id | bigint | 物流单ID |
| description | varchar(500) | 轨迹描述 |
| location | varchar(200) | 位置 |
| track_time | bigint | 轨迹时间 |
| ctime | bigint | |

#### Kafka 事件

| Topic | 方向 | 对端 | 用途 |
|-------|------|------|------|
| `order_shipped` | 生产 | order-svc, notification-svc | 发货通知 |

---

### 5.11 notification-svc（通知服务）

**职责：** 短信、邮件、站内信统一发送

#### 数据库表（库：`mall_notification`）

**notification_templates 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| tenant_id | bigint | 0=平台模板 |
| code | varchar(50) | 模板编码 |
| channel | tinyint | 1-短信 2-邮件 3-站内信 |
| title | varchar(200) | 标题 |
| content | text | 模板内容 |
| status | tinyint | 1-启用 2-停用 |
| ctime / utime | bigint | |

**notifications 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| user_id | bigint | 用户ID |
| tenant_id | bigint | 租户ID |
| channel | tinyint | 1-短信 2-邮件 3-站内信 |
| title | varchar(200) | 标题 |
| content | text | 内容 |
| is_read | tinyint | 是否已读 |
| status | tinyint | 1-待发送 2-已发送 3-发送失败 |
| ctime / utime | bigint | |

#### Kafka 消费（核心角色）

| 消费 Topic | 行为 |
|------------|------|
| `user_registered` | 发送欢迎消息 |
| `order_paid` | 通知商家新订单 |
| `order_shipped` | 通知买家已发货 |
| `inventory_alert` | 通知商家库存不足 |
| `tenant_approved` | 通知商家审核通过 |

---

## 6. Kafka Topic 全景图

| Topic | 生产者 | 消费者 |
|-------|--------|--------|
| `user_registered` | user-svc | notification-svc, marketing-svc |
| `user_login` | user-svc | 审计 |
| `tenant_approved` | tenant-svc | user-svc, notification-svc |
| `tenant_plan_changed` | tenant-svc | product-svc |
| `product_status_changed` | product-svc | search-svc |
| `product_updated` | product-svc | search-svc, cart-svc |
| `product_binlog` | Canal | search-svc |
| `inventory_deducted` | inventory-svc | order-svc |
| `inventory_alert` | inventory-svc | notification-svc |
| `order_created` | order-svc | inventory-svc, marketing-svc |
| `order_close_delay` | order-svc | order-svc |
| `order_cancelled` | order-svc | inventory-svc, marketing-svc |
| `order_completed` | order-svc | product-svc |
| `order_paid` | payment-svc | order-svc, inventory-svc, notification-svc |
| `refund_completed` | payment-svc | order-svc |
| `seckill_order_created` | marketing-svc | order-svc |
| `order_shipped` | logistics-svc | order-svc, notification-svc |

---

## 7. BFF 网关层详设

> C 端采用**独立商城 SaaS 模式**（类似 Shopify/有赞），每个商家拥有独立域名/子域名的 C 端入口，消费者在单一店铺内浏览购买，无跨店购物车和拆单逻辑。

### 7.1 consumer-bff（C 端网关）— Port 8080

**gRPC 客户端依赖（11 个）：**

| gRPC 服务 | 用途 |
|-----------|------|
| tenant-svc | 域名→tenant_id 解析、店铺首页信息 |
| user-svc | 注册、登录、个人信息、收货地址 |
| product-svc | 商品列表/详情、分类、品牌（限 tenant_id） |
| cart-svc | 购物车 CRUD（限 tenant_id） |
| order-svc | 下单、订单列表/详情、取消、确认收货、退款 |
| payment-svc | 发起支付、支付回调 |
| inventory-svc | 查库存（购物车/详情页聚合） |
| search-svc | 店内搜索（tenant_id=N）、建议、热搜 |
| marketing-svc | 店铺优惠券、秒杀、优惠计算 |
| logistics-svc | 物流轨迹、运费预计算 |
| notification-svc | 站内信 |

**核心中间件：**

| 中间件 | 说明 |
|--------|------|
| TenantResolve | 从 Host 提取子域名 → 调 tenant-svc 查 tenant_id（结果缓存 Redis） |
| LoginJWT | Bearer Token 校验 + Redis 黑名单检查，注入 uid 到 context |
| RateLimit | 全局限流（滑动窗口），秒杀接口独立限流 |

**HTTP API（44 个接口）：**

> 所有接口自动限定在当前店铺（tenant_id 由 TenantResolve 中间件注入）

```
# 店铺
GET    /api/v1/shop                            → tenant.GetShop

# 用户
POST   /api/v1/users/signup                    → user.Signup
POST   /api/v1/users/login                     → user.Login
POST   /api/v1/users/logout                    → JWT 黑名单
POST   /api/v1/users/refresh-token             → 刷新 Token
GET    /api/v1/users/profile                   → user.FindById
PUT    /api/v1/users/profile                   → user.UpdateProfile

# 收货地址
POST   /api/v1/addresses                       → user.CreateAddress
GET    /api/v1/addresses                       → user.ListAddresses
PUT    /api/v1/addresses/:id                   → user.UpdateAddress
DELETE /api/v1/addresses/:id                   → user.DeleteAddress

# 商品（店内）
GET    /api/v1/products                        → product.ListProducts(tenant_id)
GET    /api/v1/products/:id                    → product.GetProduct（聚合 SKU+库存）
GET    /api/v1/categories                      → product.ListCategories(tenant_id)
GET    /api/v1/brands                          → product.ListBrands(tenant_id)

# 购物车（单店铺）
POST   /api/v1/cart/items                      → cart.AddItem
GET    /api/v1/cart                             → cart.GetCart（聚合最新价格+库存）
PUT    /api/v1/cart/items/:sku_id              → cart.UpdateItem
DELETE /api/v1/cart/items/:sku_id              → cart.RemoveItem
DELETE /api/v1/cart                             → cart.ClearCart

# 订单（单商家，无拆单）
POST   /api/v1/orders                          → order.CreateOrder(tenant_id)
GET    /api/v1/orders                          → order.ListOrders(buyer_id, tenant_id)
GET    /api/v1/orders/:order_no                → order.GetOrder
POST   /api/v1/orders/:order_no/cancel         → order.CancelOrder
POST   /api/v1/orders/:order_no/confirm        → order.ConfirmReceive
POST   /api/v1/orders/:order_no/refund         → order.ApplyRefund
GET    /api/v1/orders/:order_no/refund         → order.GetRefundOrder
GET    /api/v1/orders/:order_no/logistics      → logistics.GetShipmentByOrder
POST   /api/v1/orders/preview                  → CalculateDiscount + CalculateFreight

# 支付
POST   /api/v1/payments                        → payment.CreatePayment
GET    /api/v1/payments/:payment_no            → payment.GetPayment
POST   /api/v1/payments/notify/:channel        → payment.HandleNotify（无需登录）

# 搜索（店内）
GET    /api/v1/search                          → search.SearchProducts(tenant_id)
GET    /api/v1/search/suggestions              → search.GetSuggestions
GET    /api/v1/search/hot                      → search.GetHotWords
GET    /api/v1/search/history                  → search.GetSearchHistory
DELETE /api/v1/search/history                  → search.ClearSearchHistory

# 营销（店铺级）
GET    /api/v1/coupons                         → marketing.ListCoupons(tenant_id)
POST   /api/v1/coupons/:id/receive             → marketing.ReceiveCoupon
GET    /api/v1/coupons/mine                    → marketing.ListUserCoupons(tenant_id)
GET    /api/v1/seckill/activities              → marketing.ListSeckillActivities(tenant_id)
GET    /api/v1/seckill/activities/:id          → marketing.GetSeckillActivity
POST   /api/v1/seckill/:item_id               → marketing.Seckill

# 通知
GET    /api/v1/notifications                   → notification.ListNotifications
GET    /api/v1/notifications/unread-count      → notification.GetUnreadCount
PUT    /api/v1/notifications/:id/read          → notification.MarkRead
PUT    /api/v1/notifications/read-all          → notification.MarkAllRead
```

**聚合接口（面试亮点）：**

| 接口 | 聚合逻辑 |
|------|---------|
| `GET /products/:id` | 并发调用 product.GetProduct + inventory.BatchGetStock，聚合 SKU 级库存 |
| `GET /cart` | cart.GetCart → product.BatchGetProducts + inventory.BatchGetStock，填充最新价格+库存 |
| `POST /orders/preview` | 并发调用 marketing.CalculateDiscount + logistics.CalculateFreight，展示优惠+运费 |

---

### 7.2 merchant-bff（商家端网关）— Port 8180

**gRPC 客户端依赖（9 个）：**

| gRPC 服务 | 用途 |
|-----------|------|
| user-svc | 商家员工登录、RBAC 权限校验、角色管理 |
| tenant-svc | 店铺信息、配额查询 |
| product-svc | 商品 CRUD、分类管理、品牌管理 |
| inventory-svc | 设置/查询库存、库存日志 |
| order-svc | 本店订单列表、处理退款 |
| payment-svc | 查询支付/退款状态 |
| marketing-svc | 优惠券/秒杀/满减 CRUD |
| logistics-svc | 运费模板 CRUD、发货、物流追踪 |
| notification-svc | 站内信 |

> 不依赖 cart-svc（C 端功能）、search-svc（搜索同步通过 Kafka 事件驱动）。

**核心中间件：**

| 中间件 | 说明 |
|--------|------|
| LoginJWT | 必须携带 tenant_id claim，中间件自动注入 |
| RBAC | 校验当前用户在 tenant 内的角色权限 |

**HTTP API（44 个接口）：**

```
# 认证
POST   /api/v1/auth/login                      → user.Login（+ tenant_id 绑定）
POST   /api/v1/auth/logout                     → JWT 黑名单
POST   /api/v1/auth/refresh-token              → 刷新 Token

# 店铺
GET    /api/v1/shop                            → tenant.GetShop
PUT    /api/v1/shop                            → tenant.UpdateShop
GET    /api/v1/shop/quota                      → tenant.CheckQuota

# 商品管理
POST   /api/v1/products                        → product.CreateProduct
PUT    /api/v1/products/:id                    → product.UpdateProduct
GET    /api/v1/products/:id                    → product.GetProduct
GET    /api/v1/products                        → product.ListProducts(tenant_id)
PUT    /api/v1/products/:id/status             → product.UpdateProductStatus

# 分类管理
POST   /api/v1/categories                      → product.CreateCategory
PUT    /api/v1/categories/:id                  → product.UpdateCategory
GET    /api/v1/categories                      → product.ListCategories(tenant_id)

# 品牌管理
POST   /api/v1/brands                          → product.CreateBrand
PUT    /api/v1/brands/:id                      → product.UpdateBrand
GET    /api/v1/brands                          → product.ListBrands(tenant_id)

# 库存管理
PUT    /api/v1/inventory/:sku_id               → inventory.SetStock
GET    /api/v1/inventory/:sku_id               → inventory.GetStock
GET    /api/v1/inventory/batch                 → inventory.BatchGetStock
GET    /api/v1/inventory/:sku_id/logs          → inventory.ListLogs

# 订单管理
GET    /api/v1/orders                          → order.ListOrders(tenant_id)
GET    /api/v1/orders/:order_no                → order.GetOrder
POST   /api/v1/orders/:order_no/ship           → logistics.CreateShipment + order.UpdateOrderStatus
GET    /api/v1/orders/:order_no/logistics      → logistics.GetShipmentByOrder
PUT    /api/v1/orders/:order_no/refund         → order.HandleRefund
GET    /api/v1/orders/:order_no/refund         → order.GetRefundOrder
GET    /api/v1/orders/:order_no/payment        → payment.GetPayment

# 营销管理
POST   /api/v1/coupons                         → marketing.CreateCoupon
GET    /api/v1/coupons                         → marketing.ListCoupons(tenant_id)
POST   /api/v1/seckill/activities              → marketing.CreateSeckillActivity
GET    /api/v1/seckill/activities              → marketing.ListSeckillActivities(tenant_id)
GET    /api/v1/seckill/activities/:id          → marketing.GetSeckillActivity
POST   /api/v1/promotions                      → marketing.CreatePromotionRule
GET    /api/v1/promotions                      → marketing.ListPromotionRules(tenant_id)

# 运费模板
POST   /api/v1/freight-templates               → logistics.CreateFreightTemplate
PUT    /api/v1/freight-templates/:id           → logistics.UpdateFreightTemplate
GET    /api/v1/freight-templates/:id           → logistics.GetFreightTemplate
GET    /api/v1/freight-templates               → logistics.ListFreightTemplates(tenant_id)
DELETE /api/v1/freight-templates/:id           → logistics.DeleteFreightTemplate

# 员工 & 角色（RBAC）
POST   /api/v1/staff/:user_id/role             → user.AssignRole
GET    /api/v1/roles                           → user.ListRoles(tenant_id)
GET    /api/v1/staff/:user_id/permissions      → user.GetPermissions

# 通知
GET    /api/v1/notifications                   → notification.ListNotifications
GET    /api/v1/notifications/unread-count      → notification.GetUnreadCount
PUT    /api/v1/notifications/:id/read          → notification.MarkRead
PUT    /api/v1/notifications/read-all          → notification.MarkAllRead
```

**聚合接口（面试亮点）：**

| 接口 | 聚合逻辑 |
|------|---------|
| `POST /orders/:order_no/ship` | 编排：创建物流单 + 更新订单状态为已发货，一次 HTTP 编排两个 gRPC |
| `GET /products/:id` | 聚合商品详情 + 各 SKU 库存 |

---

### 7.3 admin-bff（平台管理端网关）— Port 8280

**gRPC 客户端依赖（6 个）：**

| gRPC 服务 | 用途 |
|-----------|------|
| user-svc | 管理员登录、用户管理（冻结/解冻）、RBAC |
| tenant-svc | 商家审核/冻结、套餐 CRUD |
| product-svc | 平台分类管理、品牌管理 |
| order-svc | 全平台订单监管 |
| payment-svc | 全平台支付/退款查看 |
| notification-svc | 通知模板 CRUD、发送通知 |

> 不依赖 cart-svc、search-svc、inventory-svc、marketing-svc、logistics-svc — 这些是商家自管理或 C 端功能，平台只做监管和基础数据维护。

**核心中间件：**

| 中间件 | 说明 |
|--------|------|
| LoginJWT | 限 tenant_id=0（平台角色） |
| RBAC | 平台角色 + 权限校验 |

**HTTP API（30 个接口）：**

```
# 认证
POST   /api/v1/auth/login                          → user.Login（限 tenant_id=0）
POST   /api/v1/auth/logout                         → JWT 黑名单
POST   /api/v1/auth/refresh-token                  → 刷新 Token

# 商家管理
GET    /api/v1/tenants                             → tenant.ListTenants
GET    /api/v1/tenants/:id                         → tenant.GetTenant
PUT    /api/v1/tenants/:id/approve                 → tenant.ApproveTenant
PUT    /api/v1/tenants/:id/freeze                  → tenant.FreezeTenant

# 套餐管理
POST   /api/v1/plans                               → tenant.CreatePlan
PUT    /api/v1/plans/:id                           → tenant.UpdatePlan
GET    /api/v1/plans                               → tenant.ListPlans

# 平台分类
POST   /api/v1/categories                          → product.CreateCategory(tenant_id=0)
PUT    /api/v1/categories/:id                      → product.UpdateCategory
GET    /api/v1/categories                          → product.ListCategories(tenant_id=0)

# 品牌管理
POST   /api/v1/brands                              → product.CreateBrand(tenant_id=0)
PUT    /api/v1/brands/:id                          → product.UpdateBrand
GET    /api/v1/brands                              → product.ListBrands(tenant_id=0)

# 用户管理
GET    /api/v1/users/:id                           → user.FindById
PUT    /api/v1/users/:id/status                    → user.UpdateUserStatus

# 权限管理（RBAC）
POST   /api/v1/users/:user_id/role                 → user.AssignRole(tenant_id=0)
GET    /api/v1/users/:user_id/permissions           → user.GetPermissions
GET    /api/v1/roles                               → user.ListRoles(tenant_id=0)

# 订单监管
GET    /api/v1/orders                              → order.ListOrders（全平台）
GET    /api/v1/orders/:order_no                    → order.GetOrder

# 支付监管
GET    /api/v1/payments/:payment_no                → payment.GetPayment
GET    /api/v1/refunds/:refund_no                  → payment.GetRefund

# 通知管理
POST   /api/v1/notification-templates              → notification.CreateTemplate
PUT    /api/v1/notification-templates/:id           → notification.UpdateTemplate
GET    /api/v1/notification-templates              → notification.ListTemplates(tenant_id=0)
POST   /api/v1/notifications/send                  → notification.SendNotification
POST   /api/v1/notifications/batch-send            → notification.BatchSendNotification
```

---

### 7.4 BFF 汇总对比

| 维度 | consumer-bff | merchant-bff | admin-bff |
|------|-------------|-------------|-----------|
| gRPC 客户端数 | 11 | 9 | 6 |
| HTTP 接口数 | 44 | 44 | 30 |
| JWT 策略 | uid（tenant_id 从域名中间件注入） | uid + tenant_id（JWT claim） | uid, tenant_id=0 |
| 租户识别 | 域名解析中间件 | JWT claim | 固定 tenant_id=0 |
| RBAC 鉴权 | 无（登录即可） | 角色权限校验 | 平台角色 + 权限校验 |
| 面试亮点 | 多租户域名路由、租户解析缓存、数据聚合 | 发货编排、RBAC 鉴权 | 平台级权限管控 |
| 端口 | 8080 | 8180 | 8280 |

---

## 8. 核心业务流程

### 8.1 下单流程

```
C端用户 → consumer-bff → [选商品/加购物车] → 提交订单
  → order-svc.CreateOrder()
    → gRPC 调用 product-svc 校验商品信息和价格
    → gRPC 调用 marketing-svc 计算优惠
    → gRPC 调用 logistics-svc 计算运费
    → gRPC 调用 inventory-svc 预扣库存
    → 创建订单（MySQL）
    → 发 Kafka: order_created
    → 发 Kafka: order_close_delay (延迟30min)
    → 返回订单号 + 支付信息
  → consumer-bff 调用 payment-svc 发起支付
  → 用户支付 → 第三方回调 → payment-svc 处理
    → 发 Kafka: order_paid
    → order-svc 消费 → 更新状态为 paid
    → inventory-svc 消费 → 确认库存扣减
```

### 8.2 秒杀流程

```
C端用户 → consumer-bff (令牌桶限流)
  → marketing-svc.Seckill(uid, item_id)
    → Redis Lua: 检查限购 + 扣减秒杀库存
    → 失败 → 返回"已抢完"
    → 成功 → 发 Kafka: seckill_order_created
  → order-svc Consumer 异步消费
    → 创建秒杀订单
    → 后续同普通下单流程
```

### 8.3 搜索数据同步

```
商家操作商品 → product-svc → MySQL (products 表)
  → Canal 监听 Binlog
  → 发 Kafka: product_binlog
  → search-svc Consumer 消费
  → 解析变更类型 (INSERT/UPDATE/DELETE)
  → 更新 ES products 索引
```

---

## 9. 面试亮点总结

| 亮点 | 涉及服务 | 技术深度 |
|------|----------|----------|
| Redis+Lua 原子库存扣减 | inventory-svc, marketing-svc | 高并发、原子性、Lua 脚本 |
| 预扣/确认/回滚三阶段库存 | inventory-svc | 分布式一致性、最终一致性 |
| 秒杀全链路 | consumer-bff, marketing-svc, order-svc | 限流→扣减→Kafka削峰→异步下单 |
| 订单状态机 | order-svc | 状态模式、状态流转日志 |
| Kafka 延迟队列超时关单 | order-svc | 比定时扫表更优雅 |
| 支付回调幂等性 | payment-svc | Redis去重 + DB状态校验 |
| Canal Binlog CDC → ES | search-svc | 数据同步不侵入业务代码 |
| IK 中文分词 + Completion Suggester | search-svc | ES 搜索能力 |
| JWT 双Token + Redis 黑名单 | user-svc | 无状态认证 + 主动失效 |
| 多租户 RBAC | user-svc, tenant-svc | 平台角色与商家角色共存 |
| SaaS 套餐配额控制 | tenant-svc | 多租户商业模型 |
| 雪花算法分布式ID | order-svc, payment-svc | 分布式唯一ID |
| Redis Hash 购物车 | cart-svc | 存储策略选型 |
| 商品快照冗余 | order-svc | 数据一致性设计 |
| BFF 多租户域名路由 | consumer-bff, tenant-svc | 域名解析→tenant_id，Redis 缓存，租户级隔离 |
| BFF 数据聚合 | consumer-bff | 并发 gRPC 调用聚合商品+库存+优惠+运费 |

---

## 10. 项目目录结构

```
mall/
├── api/proto/                    # Proto 定义
│   ├── user/v1/
│   ├── tenant/v1/
│   ├── product/v1/
│   ├── inventory/v1/
│   ├── order/v1/
│   ├── payment/v1/
│   ├── cart/v1/
│   ├── search/v1/
│   ├── marketing/v1/
│   ├── logistics/v1/
│   ├── notification/v1/
│   └── gen/                      # buf 生成代码
│
├── admin-bff/                    # 平台管理端 BFF
├── merchant-bff/                 # 商家端 BFF
├── consumer-bff/                 # C 端 BFF
│
├── user/                         # 用户服务
├── tenant/                       # 租户服务
├── product/                      # 商品服务
├── inventory/                    # 库存服务
├── order/                        # 订单服务
├── payment/                      # 支付服务
├── cart/                         # 购物车服务
├── search/                       # 搜索服务
├── marketing/                    # 营销服务
├── logistics/                    # 物流服务
├── notification/                 # 通知服务
│
├── pkg/                          # 共享工具包
│   ├── logger/
│   ├── grpcx/
│   ├── ginx/
│   ├── gormx/
│   ├── saramax/
│   ├── redisx/
│   ├── ratelimit/
│   ├── snowflake/                # 雪花ID生成
│   ├── tenantx/                  # 多租户中间件
│   ├── cronjobx/
│   ├── migrator/
│   └── canalx/
│
├── config/
├── script/
├── docs/
│
├── go.mod
├── Makefile
├── docker-compose.yaml
├── buf.gen.yaml
├── prometheus.yaml
└── k8s-*.yaml
```

每个微服务内部遵循 template.md 定义的 DDD 标准结构。
