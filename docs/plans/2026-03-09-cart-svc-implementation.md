# Cart Service + Consumer BFF Cart 接口实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 cart-svc 购物车微服务（6 个 gRPC RPC）+ consumer-bff 购物车 HTTP 接口（6 个端点），BFF 聚合 product-svc 商品信息。

**Architecture:** DDD 分层（domain → dao → cache → repository → service → grpc → ioc → wire），MySQL + Redis Cache-Aside 存储，gRPC + etcd 服务发现。cart-svc 是纯 CRUD 服务（无 Kafka、无 Snowflake、无幂等）。consumer-bff GetCart 聚合 product-svc 商品信息。

**Tech Stack:** Go, gRPC, GORM, Redis, Wire DI, etcd, Gin (BFF), protobuf

---

## Task 1: Domain + DAO + Init（cart-svc 数据层）

**Files:**
- Create: `cart/domain/cart.go`
- Create: `cart/repository/dao/cart.go`
- Create: `cart/repository/dao/init.go`

**Step 1: 创建 domain 模型**

Create `cart/domain/cart.go`:

```go
package domain

import "time"

type CartItem struct {
	ID        int64
	UserID    int64
	SkuID     int64
	ProductID int64
	TenantID  int64
	Quantity  int32
	Selected  bool
	Ctime     time.Time
	Utime     time.Time
}
```

**Step 2: 创建 DAO 层**

Create `cart/repository/dao/cart.go`:

```go
package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CartItemModel struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	UserId    int64 `gorm:"uniqueIndex:uk_user_sku;index:idx_user"`
	SkuId     int64 `gorm:"uniqueIndex:uk_user_sku"`
	ProductId int64
	TenantId  int64
	Quantity  int32
	Selected  bool
	Ctime     int64
	Utime     int64
}

func (CartItemModel) TableName() string { return "cart_items" }

type CartDAO interface {
	Upsert(ctx context.Context, item CartItemModel) error
	Update(ctx context.Context, userId, skuId int64, updates map[string]any) error
	Delete(ctx context.Context, userId, skuId int64) error
	FindByUser(ctx context.Context, userId int64) ([]CartItemModel, error)
	DeleteByUser(ctx context.Context, userId int64) error
	BatchDelete(ctx context.Context, userId int64, skuIds []int64) error
}

type GORMCartDAO struct {
	db *gorm.DB
}

func NewCartDAO(db *gorm.DB) CartDAO {
	return &GORMCartDAO{db: db}
}

func (d *GORMCartDAO) Upsert(ctx context.Context, item CartItemModel) error {
	now := time.Now().UnixMilli()
	item.Ctime = now
	item.Utime = now
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "sku_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"quantity": gorm.Expr("quantity + ?", item.Quantity),
			"utime":    now,
		}),
	}).Create(&item).Error
}

func (d *GORMCartDAO) Update(ctx context.Context, userId, skuId int64, updates map[string]any) error {
	updates["utime"] = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&CartItemModel{}).
		Where("user_id = ? AND sku_id = ?", userId, skuId).
		Updates(updates).Error
}

func (d *GORMCartDAO) Delete(ctx context.Context, userId, skuId int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND sku_id = ?", userId, skuId).
		Delete(&CartItemModel{}).Error
}

func (d *GORMCartDAO) FindByUser(ctx context.Context, userId int64) ([]CartItemModel, error) {
	var items []CartItemModel
	err := d.db.WithContext(ctx).Where("user_id = ?", userId).
		Order("id DESC").Find(&items).Error
	return items, err
}

func (d *GORMCartDAO) DeleteByUser(ctx context.Context, userId int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Delete(&CartItemModel{}).Error
}

func (d *GORMCartDAO) BatchDelete(ctx context.Context, userId int64, skuIds []int64) error {
	return d.db.WithContext(ctx).
		Where("user_id = ? AND sku_id IN ?", userId, skuIds).
		Delete(&CartItemModel{}).Error
}
```

**Step 3: 创建 DAO init**

Create `cart/repository/dao/init.go`:

```go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(&CartItemModel{})
}
```

**Step 4: 验证编译**

```bash
go build ./cart/...
```

---

## Task 2: Cache + Repository（cart-svc 缓存与仓储层）

**Files:**
- Create: `cart/repository/cache/cart.go`
- Create: `cart/repository/cart.go`

**Step 1: 创建 Redis 缓存**

Create `cart/repository/cache/cart.go`:

```go
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CartCache interface {
	Get(ctx context.Context, userId int64) ([]byte, error)
	Set(ctx context.Context, userId int64, data []byte) error
	Delete(ctx context.Context, userId int64) error
}

type RedisCartCache struct {
	client redis.Cmdable
}

func NewCartCache(client redis.Cmdable) CartCache {
	return &RedisCartCache{client: client}
}

func cartKey(userId int64) string {
	return fmt.Sprintf("cart:items:%d", userId)
}

func (c *RedisCartCache) Get(ctx context.Context, userId int64) ([]byte, error) {
	return c.client.Get(ctx, cartKey(userId)).Bytes()
}

func (c *RedisCartCache) Set(ctx context.Context, userId int64, data []byte) error {
	return c.client.Set(ctx, cartKey(userId), data, 30*time.Minute).Err()
}

func (c *RedisCartCache) Delete(ctx context.Context, userId int64) error {
	return c.client.Del(ctx, cartKey(userId)).Err()
}
```

**Step 2: 创建 Repository**

Create `cart/repository/cart.go`:

```go
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rermrf/mall/cart/domain"
	"github.com/rermrf/mall/cart/repository/cache"
	"github.com/rermrf/mall/cart/repository/dao"
)

type CartRepository interface {
	AddItem(ctx context.Context, item domain.CartItem) error
	UpdateItem(ctx context.Context, userId, skuId int64, updates map[string]any) error
	RemoveItem(ctx context.Context, userId, skuId int64) error
	GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error)
	ClearCart(ctx context.Context, userId int64) error
	BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error
}

type cartRepository struct {
	dao   dao.CartDAO
	cache cache.CartCache
}

func NewCartRepository(d dao.CartDAO, c cache.CartCache) CartRepository {
	return &cartRepository{dao: d, cache: c}
}

func (r *cartRepository) AddItem(ctx context.Context, item domain.CartItem) error {
	err := r.dao.Upsert(ctx, r.toModel(item))
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, item.UserID)
	return nil
}

func (r *cartRepository) UpdateItem(ctx context.Context, userId, skuId int64, updates map[string]any) error {
	err := r.dao.Update(ctx, userId, skuId, updates)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) RemoveItem(ctx context.Context, userId, skuId int64) error {
	err := r.dao.Delete(ctx, userId, skuId)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error) {
	// 尝试缓存
	data, err := r.cache.Get(ctx, userId)
	if err == nil {
		var items []domain.CartItem
		if json.Unmarshal(data, &items) == nil {
			return items, nil
		}
	}
	if err != nil && err != redis.Nil {
		// Redis 错误只记录，不阻塞
	}
	// 回源数据库
	models, err := r.dao.FindByUser(ctx, userId)
	if err != nil {
		return nil, err
	}
	items := make([]domain.CartItem, 0, len(models))
	for _, m := range models {
		items = append(items, r.toDomain(m))
	}
	// 回填缓存
	if jsonData, e := json.Marshal(items); e == nil {
		_ = r.cache.Set(ctx, userId, jsonData)
	}
	return items, nil
}

func (r *cartRepository) ClearCart(ctx context.Context, userId int64) error {
	err := r.dao.DeleteByUser(ctx, userId)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error {
	err := r.dao.BatchDelete(ctx, userId, skuIds)
	if err != nil {
		return err
	}
	_ = r.cache.Delete(ctx, userId)
	return nil
}

func (r *cartRepository) toModel(item domain.CartItem) dao.CartItemModel {
	return dao.CartItemModel{
		UserId:    item.UserID,
		SkuId:     item.SkuID,
		ProductId: item.ProductID,
		TenantId:  item.TenantID,
		Quantity:  item.Quantity,
		Selected:  item.Selected,
	}
}

func (r *cartRepository) toDomain(m dao.CartItemModel) domain.CartItem {
	return domain.CartItem{
		ID:        m.ID,
		UserID:    m.UserId,
		SkuID:     m.SkuId,
		ProductID: m.ProductId,
		TenantID:  m.TenantId,
		Quantity:  m.Quantity,
		Selected:  m.Selected,
		Ctime:     time.UnixMilli(m.Ctime),
		Utime:     time.UnixMilli(m.Utime),
	}
}
```

**Step 3: 验证编译**

```bash
go build ./cart/...
```

---

## Task 3: Service + gRPC Handler（cart-svc 业务与接口层）

**Files:**
- Create: `cart/service/cart.go`
- Create: `cart/grpc/cart.go`

**Step 1: 创建 Service**

Create `cart/service/cart.go`:

```go
package service

import (
	"context"
	"fmt"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/cart/domain"
	"github.com/rermrf/mall/cart/repository"
)

type CartService interface {
	AddItem(ctx context.Context, item domain.CartItem) error
	UpdateItem(ctx context.Context, userId, skuId int64, quantity int32, selected bool, updateSelected bool) error
	RemoveItem(ctx context.Context, userId, skuId int64) error
	GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error)
	ClearCart(ctx context.Context, userId int64) error
	BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error
}

type cartService struct {
	repo repository.CartRepository
	l    logger.Logger
}

func NewCartService(repo repository.CartRepository, l logger.Logger) CartService {
	return &cartService{repo: repo, l: l}
}

func (s *cartService) AddItem(ctx context.Context, item domain.CartItem) error {
	if item.Quantity <= 0 {
		return fmt.Errorf("数量必须大于 0")
	}
	return s.repo.AddItem(ctx, item)
}

func (s *cartService) UpdateItem(ctx context.Context, userId, skuId int64, quantity int32, selected bool, updateSelected bool) error {
	updates := make(map[string]any)
	if quantity > 0 {
		updates["quantity"] = quantity
	}
	if updateSelected {
		updates["selected"] = selected
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateItem(ctx, userId, skuId, updates)
}

func (s *cartService) RemoveItem(ctx context.Context, userId, skuId int64) error {
	return s.repo.RemoveItem(ctx, userId, skuId)
}

func (s *cartService) GetCart(ctx context.Context, userId int64) ([]domain.CartItem, error) {
	return s.repo.GetCart(ctx, userId)
}

func (s *cartService) ClearCart(ctx context.Context, userId int64) error {
	return s.repo.ClearCart(ctx, userId)
}

func (s *cartService) BatchRemoveItems(ctx context.Context, userId int64, skuIds []int64) error {
	if len(skuIds) == 0 {
		return nil
	}
	return s.repo.BatchRemoveItems(ctx, userId, skuIds)
}
```

**Step 2: 创建 gRPC Handler**

Create `cart/grpc/cart.go`:

```go
package grpc

import (
	"context"

	"google.golang.org/grpc"

	cartv1 "github.com/rermrf/mall/api/proto/gen/cart/v1"
	"github.com/rermrf/mall/cart/domain"
	"github.com/rermrf/mall/cart/service"
)

type CartGRPCServer struct {
	cartv1.UnimplementedCartServiceServer
	svc service.CartService
}

func NewCartGRPCServer(svc service.CartService) *CartGRPCServer {
	return &CartGRPCServer{svc: svc}
}

func (s *CartGRPCServer) Register(server *grpc.Server) {
	cartv1.RegisterCartServiceServer(server, s)
}

func (s *CartGRPCServer) AddItem(ctx context.Context, req *cartv1.AddItemRequest) (*cartv1.AddItemResponse, error) {
	err := s.svc.AddItem(ctx, domain.CartItem{
		UserID:    req.GetUserId(),
		SkuID:     req.GetSkuId(),
		ProductID: req.GetProductId(),
		TenantID:  req.GetTenantId(),
		Quantity:  req.GetQuantity(),
	})
	if err != nil {
		return nil, err
	}
	return &cartv1.AddItemResponse{}, nil
}

func (s *CartGRPCServer) UpdateItem(ctx context.Context, req *cartv1.UpdateItemRequest) (*cartv1.UpdateItemResponse, error) {
	err := s.svc.UpdateItem(ctx, req.GetUserId(), req.GetSkuId(), req.GetQuantity(), req.GetSelected(), req.GetUpdateSelected())
	if err != nil {
		return nil, err
	}
	return &cartv1.UpdateItemResponse{}, nil
}

func (s *CartGRPCServer) RemoveItem(ctx context.Context, req *cartv1.RemoveItemRequest) (*cartv1.RemoveItemResponse, error) {
	err := s.svc.RemoveItem(ctx, req.GetUserId(), req.GetSkuId())
	if err != nil {
		return nil, err
	}
	return &cartv1.RemoveItemResponse{}, nil
}

func (s *CartGRPCServer) GetCart(ctx context.Context, req *cartv1.GetCartRequest) (*cartv1.GetCartResponse, error) {
	items, err := s.svc.GetCart(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	pbItems := make([]*cartv1.CartItem, 0, len(items))
	for _, item := range items {
		pbItems = append(pbItems, s.toDTO(item))
	}
	return &cartv1.GetCartResponse{Items: pbItems}, nil
}

func (s *CartGRPCServer) ClearCart(ctx context.Context, req *cartv1.ClearCartRequest) (*cartv1.ClearCartResponse, error) {
	err := s.svc.ClearCart(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &cartv1.ClearCartResponse{}, nil
}

func (s *CartGRPCServer) BatchRemoveItems(ctx context.Context, req *cartv1.BatchRemoveItemsRequest) (*cartv1.BatchRemoveItemsResponse, error) {
	err := s.svc.BatchRemoveItems(ctx, req.GetUserId(), req.GetSkuIds())
	if err != nil {
		return nil, err
	}
	return &cartv1.BatchRemoveItemsResponse{}, nil
}

func (s *CartGRPCServer) toDTO(item domain.CartItem) *cartv1.CartItem {
	return &cartv1.CartItem{
		SkuId:     item.SkuID,
		ProductId: item.ProductID,
		TenantId:  item.TenantID,
		Quantity:  item.Quantity,
		Selected:  item.Selected,
	}
}
```

**Step 3: 验证编译**

```bash
go build ./cart/...
```

---

## Task 4: IoC + Wire + Config + Main（cart-svc 基础设施）

**Files:**
- Create: `cart/ioc/db.go`
- Create: `cart/ioc/redis.go`
- Create: `cart/ioc/logger.go`
- Create: `cart/ioc/grpc.go`
- Create: `cart/wire.go`
- Create: `cart/app.go`
- Create: `cart/main.go`
- Create: `cart/config/dev.yaml`
- Generate: `cart/wire_gen.go`

**Step 1: 创建 IoC — DB**

Create `cart/ioc/db.go`:

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/mall/cart/repository/dao"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	var cfg Config
	err := viper.UnmarshalKey("db", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取数据库配置失败: %w", err))
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("连接数据库失败: %w", err))
	}
	err = dao.InitTables(db)
	if err != nil {
		panic(fmt.Errorf("数据库表初始化失败: %w", err))
	}
	return db
}
```

**Step 2: 创建 IoC — Redis**

Create `cart/ioc/redis.go`:

```go
package ioc

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	type Config struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	}
	var cfg Config
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Redis 配置失败: %w", err))
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return client
}
```

**Step 3: 创建 IoC — Logger**

Create `cart/ioc/logger.go`:

```go
package ioc

import (
	"github.com/rermrf/emo/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.NewZapLogger(l)
}
```

**Step 4: 创建 IoC — gRPC + etcd**

Create `cart/ioc/grpc.go`:

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	cgrpc "github.com/rermrf/mall/cart/grpc"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func InitEtcdClient() *clientv3.Client {
	var cfg struct {
		Addrs []string `yaml:"addrs"`
	}
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{Endpoints: cfg.Addrs})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitGRPCServer(cartServer *cgrpc.CartGRPCServer, l logger.Logger) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 gRPC 配置失败: %w", err))
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(tenantx.GRPCUnaryServerInterceptor()))
	cartServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "cart",
		L:         l,
	}
}
```

**Step 5: 创建 App**

Create `cart/app.go`:

```go
package main

import "github.com/rermrf/mall/pkg/grpcx"

type App struct {
	Server *grpcx.Server
}
```

**Step 6: 创建 Wire DI**

Create `cart/wire.go`:

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	cgrpc "github.com/rermrf/mall/cart/grpc"
	"github.com/rermrf/mall/cart/ioc"
	"github.com/rermrf/mall/cart/repository"
	"github.com/rermrf/mall/cart/repository/cache"
	"github.com/rermrf/mall/cart/repository/dao"
	"github.com/rermrf/mall/cart/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitDB,
	ioc.InitRedis,
	ioc.InitLogger,
	ioc.InitEtcdClient,
)

var cartSet = wire.NewSet(
	dao.NewCartDAO,
	cache.NewCartCache,
	repository.NewCartRepository,
	service.NewCartService,
	cgrpc.NewCartGRPCServer,
	ioc.InitGRPCServer,
)

func InitApp() *App {
	wire.Build(thirdPartySet, cartSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

**Step 7: 创建 main.go**

Create `cart/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
	app := InitApp()

	go func() {
		if err := app.Server.Serve(); err != nil {
			fmt.Println("gRPC 服务启动失败:", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("正在关闭服务...")
	app.Server.Close()
}

func initViper() {
	configFile := pflag.String("config", "config/dev.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}
}
```

**Step 8: 创建配置文件**

Create `cart/config/dev.yaml`:

```yaml
db:
  dsn: "root:wen123...@tcp(rermrf.icu:3306)/mall_cart?charset=utf8mb4&parseTime=true&loc=Local"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 6

grpc:
  port: 8087
  etcdAddrs:
    - "rermrf.icu:2379"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

**Step 9: 生成 Wire 代码并验证**

```bash
cd cart && wire && cd ..
go build ./cart/...
go vet ./cart/...
```

---

## Task 5: Consumer BFF Cart 接口（6 个端点 + 商品聚合）

**Files:**
- Create: `consumer-bff/handler/cart.go`
- Modify: `consumer-bff/ioc/grpc.go` — +InitCartClient +InitProductClient
- Modify: `consumer-bff/ioc/gin.go` — +cartHandler 参数 + 6 路由
- Modify: `consumer-bff/wire.go` — +cart/product client + handler

**Step 1: 创建 CartHandler**

Create `consumer-bff/handler/cart.go`:

```go
package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	cartv1 "github.com/rermrf/mall/api/proto/gen/cart/v1"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type CartHandler struct {
	cartClient    cartv1.CartServiceClient
	productClient productv1.ProductServiceClient
	l             logger.Logger
}

func NewCartHandler(
	cartClient cartv1.CartServiceClient,
	productClient productv1.ProductServiceClient,
	l logger.Logger,
) *CartHandler {
	return &CartHandler{
		cartClient:    cartClient,
		productClient: productClient,
		l:             l,
	}
}

type AddCartItemReq struct {
	SkuID     int64 `json:"sku_id" binding:"required"`
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int32 `json:"quantity" binding:"required,min=1"`
}

func (h *CartHandler) AddItem(ctx *gin.Context, req AddCartItemReq) (ginx.Result, error) {
	uid, _ := ctx.Get("user_id")
	tenantId, _ := ctx.Get("tenant_id")
	_, err := h.cartClient.AddItem(ctx.Request.Context(), &cartv1.AddItemRequest{
		UserId:    uid.(int64),
		SkuId:     req.SkuID,
		ProductId: req.ProductID,
		TenantId:  tenantId.(int64),
		Quantity:  req.Quantity,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("加入购物车失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type UpdateCartItemReq struct {
	Quantity       int32 `json:"quantity"`
	Selected       bool  `json:"selected"`
	UpdateSelected bool  `json:"update_selected"`
}

func (h *CartHandler) UpdateItem(ctx *gin.Context, req UpdateCartItemReq) (ginx.Result, error) {
	uid, _ := ctx.Get("user_id")
	skuIdStr := ctx.Param("skuId")
	skuId, err := strconv.ParseInt(skuIdStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的 SKU ID"}, nil
	}
	_, err = h.cartClient.UpdateItem(ctx.Request.Context(), &cartv1.UpdateItemRequest{
		UserId:         uid.(int64),
		SkuId:          skuId,
		Quantity:       req.Quantity,
		Selected:       req.Selected,
		UpdateSelected: req.UpdateSelected,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新购物车失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *CartHandler) RemoveItem(ctx *gin.Context) {
	uid, _ := ctx.Get("user_id")
	skuIdStr := ctx.Param("skuId")
	skuId, err := strconv.ParseInt(skuIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的 SKU ID"})
		return
	}
	_, err = h.cartClient.RemoveItem(ctx.Request.Context(), &cartv1.RemoveItemRequest{
		UserId: uid.(int64),
		SkuId:  skuId,
	})
	if err != nil {
		h.l.Error("删除购物车商品失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type CartItemVO struct {
	SkuID        int64  `json:"sku_id"`
	ProductID    int64  `json:"product_id"`
	Quantity     int32  `json:"quantity"`
	Selected     bool   `json:"selected"`
	ProductName  string `json:"product_name"`
	ProductImage string `json:"product_image"`
	SkuSpec      string `json:"sku_spec"`
	Price        int64  `json:"price"`
	Stock        int32  `json:"stock"`
}

func (h *CartHandler) GetCart(ctx *gin.Context) {
	uid, _ := ctx.Get("user_id")
	// 1. 获取购物车基础数据
	cartResp, err := h.cartClient.GetCart(ctx.Request.Context(), &cartv1.GetCartRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("获取购物车失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	items := cartResp.GetItems()
	if len(items) == 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: []CartItemVO{}})
		return
	}
	// 2. 收集 productIds，批量查询商品信息
	productIdSet := make(map[int64]struct{})
	for _, item := range items {
		productIdSet[item.GetProductId()] = struct{}{}
	}
	productIds := make([]int64, 0, len(productIdSet))
	for id := range productIdSet {
		productIds = append(productIds, id)
	}
	productResp, err := h.productClient.BatchGetProducts(ctx.Request.Context(), &productv1.BatchGetProductsRequest{
		Ids: productIds,
	})
	if err != nil {
		h.l.Error("批量查询商品信息失败", logger.Error(err))
		// 降级：返回不含商品详情的购物车
		vos := make([]CartItemVO, 0, len(items))
		for _, item := range items {
			vos = append(vos, CartItemVO{
				SkuID:     item.GetSkuId(),
				ProductID: item.GetProductId(),
				Quantity:  item.GetQuantity(),
				Selected:  item.GetSelected(),
			})
		}
		ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: vos})
		return
	}
	// 3. 构建 productId → Product 映射，skuId → SKU 映射
	productMap := make(map[int64]*productv1.Product)
	skuMap := make(map[int64]*productv1.ProductSKU)
	for _, p := range productResp.GetProducts() {
		productMap[p.GetId()] = p
		for _, sku := range p.GetSkus() {
			skuMap[sku.GetId()] = sku
		}
	}
	// 4. 聚合返回
	vos := make([]CartItemVO, 0, len(items))
	for _, item := range items {
		vo := CartItemVO{
			SkuID:     item.GetSkuId(),
			ProductID: item.GetProductId(),
			Quantity:  item.GetQuantity(),
			Selected:  item.GetSelected(),
		}
		if p, ok := productMap[item.GetProductId()]; ok {
			vo.ProductName = p.GetName()
			vo.ProductImage = p.GetMainImage()
		}
		if sku, ok := skuMap[item.GetSkuId()]; ok {
			vo.SkuSpec = sku.GetSpecValues()
			vo.Price = sku.GetPrice()
		}
		vos = append(vos, vo)
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: vos})
}

func (h *CartHandler) ClearCart(ctx *gin.Context) {
	uid, _ := ctx.Get("user_id")
	_, err := h.cartClient.ClearCart(ctx.Request.Context(), &cartv1.ClearCartRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("清空购物车失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type BatchRemoveReq struct {
	SkuIDs []int64 `json:"sku_ids" binding:"required,min=1"`
}

func (h *CartHandler) BatchRemove(ctx *gin.Context, req BatchRemoveReq) (ginx.Result, error) {
	uid, _ := ctx.Get("user_id")
	_, err := h.cartClient.BatchRemoveItems(ctx.Request.Context(), &cartv1.BatchRemoveItemsRequest{
		UserId: uid.(int64),
		SkuIds: req.SkuIDs,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("批量删除购物车商品失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}
```

**Step 2: 修改 consumer-bff/ioc/grpc.go — 添加 InitCartClient + InitProductClient**

在文件末尾添加：

```go
import cartv1 "github.com/rermrf/mall/api/proto/gen/cart/v1"
import productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"

func InitCartClient(etcdClient *clientv3.Client) cartv1.CartServiceClient {
	conn := initServiceConn(etcdClient, "cart")
	return cartv1.NewCartServiceClient(conn)
}

func InitProductClient(etcdClient *clientv3.Client) productv1.ProductServiceClient {
	conn := initServiceConn(etcdClient, "product")
	return productv1.NewProductServiceClient(conn)
}
```

**Step 3: 修改 consumer-bff/ioc/gin.go — 添加 cartHandler 参数 + 6 路由**

`InitGinServer` 函数签名添加 `cartHandler *handler.CartHandler` 参数。

在 `auth` 路由组中添加购物车路由：

```go
// 购物车
auth.POST("/cart/items", ginx.WrapBody[handler.AddCartItemReq](l, cartHandler.AddItem))
auth.PUT("/cart/items/:skuId", ginx.WrapBody[handler.UpdateCartItemReq](l, cartHandler.UpdateItem))
auth.DELETE("/cart/items/:skuId", cartHandler.RemoveItem)
auth.GET("/cart", cartHandler.GetCart)
auth.DELETE("/cart", cartHandler.ClearCart)
auth.POST("/cart/batch-remove", ginx.WrapBody[handler.BatchRemoveReq](l, cartHandler.BatchRemove))
```

**Step 4: 修改 consumer-bff/wire.go — 添加依赖注入**

`thirdPartySet` 添加 `ioc.InitCartClient`, `ioc.InitProductClient`。
`handlerSet` 添加 `handler.NewCartHandler`。

**Step 5: 重新生成 Wire 代码并验证**

```bash
cd consumer-bff && wire && cd ..
go build ./consumer-bff/...
go vet ./consumer-bff/...
```

---

## 文件清单总览

| # | 文件路径 | 操作 | Task |
|---|---------|------|------|
| 1 | `cart/domain/cart.go` | 新建 | 1 |
| 2 | `cart/repository/dao/cart.go` | 新建 | 1 |
| 3 | `cart/repository/dao/init.go` | 新建 | 1 |
| 4 | `cart/repository/cache/cart.go` | 新建 | 2 |
| 5 | `cart/repository/cart.go` | 新建 | 2 |
| 6 | `cart/service/cart.go` | 新建 | 3 |
| 7 | `cart/grpc/cart.go` | 新建 | 3 |
| 8 | `cart/ioc/db.go` | 新建 | 4 |
| 9 | `cart/ioc/redis.go` | 新建 | 4 |
| 10 | `cart/ioc/logger.go` | 新建 | 4 |
| 11 | `cart/ioc/grpc.go` | 新建 | 4 |
| 12 | `cart/wire.go` | 新建 | 4 |
| 13 | `cart/app.go` | 新建 | 4 |
| 14 | `cart/main.go` | 新建 | 4 |
| 15 | `cart/config/dev.yaml` | 新建 | 4 |
| 16 | `cart/wire_gen.go` | 生成 | 4 |
| 17 | `consumer-bff/handler/cart.go` | 新建 | 5 |
| 18 | `consumer-bff/ioc/grpc.go` | 修改 | 5 |
| 19 | `consumer-bff/ioc/gin.go` | 修改 | 5 |
| 20 | `consumer-bff/wire.go` | 修改 | 5 |
| 21 | `consumer-bff/wire_gen.go` | 重新生成 | 5 |

共 21 个文件（15 新建 + 3 修改 + 1 生成 + 2 重新生成）

## 验证

```bash
go build ./cart/...
go vet ./cart/...
go build ./consumer-bff/...
go vet ./consumer-bff/...
```
