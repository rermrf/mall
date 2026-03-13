# Go 微服务项目模板

> 基于 WeBook 项目总结的 Go 微服务架构模板，适用于中大型 Web 应用的快速搭建。

---

## 1. 项目总览

### 技术栈

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
| 容器化 | Docker + Kubernetes |
| Proto 生成 | Buf |
| Mock 生成 | mockgen |

### 架构风格

- **DDD（领域驱动设计）** 分层架构
- **微服务** 拆分，gRPC 通信
- **BFF（Backend-for-Frontend）** 聚合网关
- **事件驱动** Kafka 异步解耦
- **Cache-Aside** 缓存策略

---

## 2. 顶层目录结构

```
project/
├── api/                          # Proto 定义和生成代码
│   └── proto/
│       ├── {service}/v1/         # 各服务 .proto 文件
│       │   └── {service}.proto
│       └── gen/                  # buf 生成的 Go 代码
│           └── {service}/v1/
│               ├── {service}.pb.go
│               ├── {service}_grpc.pb.go
│               └── mocks/
│
├── bff/                          # BFF 聚合网关（HTTP 入口）
├── {service}/                    # 各微服务（gRPC 服务）
│   ├── user/
│   ├── article/
│   ├── interactive/
│   ├── comment/
│   ├── search/
│   ├── payment/
│   ├── ...
│
├── pkg/                          # 共享工具包
│   ├── logger/                   # 日志接口与实现
│   ├── grpcx/                    # gRPC 扩展（服务注册）
│   ├── ginx/                     # Gin 中间件扩展
│   ├── gormx/                    # GORM 扩展
│   ├── saramax/                  # Kafka 消费/生产封装
│   ├── redisx/                   # Redis 扩展
│   ├── ratelimit/                # 限流
│   ├── netx/                     # 网络工具
│   ├── cronjobx/                 # 定时任务封装
│   ├── migrator/                 # 数据库迁移框架
│   └── canalx/                   # MySQL Binlog CDC
│
├── config/                       # 全局默认配置
│   └── dev.yaml
├── script/                       # 脚本（SQL初始化、压测等）
│
├── go.mod                        # Go 模块定义
├── Makefile                      # 构建命令
├── Dockerfile                    # 容器镜像
├── docker-compose.yaml           # 本地开发环境
├── buf.gen.yaml                  # Proto 代码生成配置
├── prometheus.yaml               # Prometheus 配置
└── k8s-*.yaml                    # Kubernetes 部署清单
```

---

## 3. 微服务标准结构（gRPC 服务模板）

每个微服务遵循统一的 DDD 分层结构：

```
{service}/
├── config/                       # 服务配置
│   ├── dev.yaml                  # 开发环境配置
│   ├── k8s.go                    # K8s 环境配置
│   └── types.go                  # 配置结构体
│
├── domain/                       # 领域层（实体定义）
│   └── {entity}.go
│
├── service/                      # 业务逻辑层
│   ├── {entity}.go               # 接口定义 + 实现
│   ├── {entity}_test.go          # 单元测试
│   └── mocks/                    # mockgen 生成
│       └── {entity}_mock.go
│
├── repository/                   # 数据访问层
│   ├── {entity}.go               # Repository 接口 + CachedRepository
│   ├── cache/                    # Redis 缓存实现
│   │   ├── {entity}.go
│   │   └── mocks/
│   ├── dao/                      # 数据库访问对象
│   │   ├── {entity}.go           # DAO 接口 + GORM 实现
│   │   ├── init.go               # 表初始化（AutoMigrate）
│   │   └── mocks/
│   └── mocks/                    # Repository mock
│
├── grpc/                         # gRPC 服务端
│   └── {service}.go              # gRPC Handler（proto → domain 转换）
│
├── events/                       # Kafka 事件
│   ├── producer.go               # 事件生产者
│   ├── consumer.go               # 事件消费者
│   └── types.go                  # 事件结构体
│
├── ioc/                          # 依赖初始化（IoC 容器）
│   ├── db.go                     # MySQL 初始化
│   ├── redis.go                  # Redis 初始化
│   ├── kafka.go                  # Kafka 初始化
│   ├── logger.go                 # Logger 初始化
│   └── grpc.go                   # gRPC Server 初始化
│
├── integration/                  # 集成测试
│   ├── startup/                  # 测试专用 Wire 注入
│   │   ├── wire.go
│   │   └── wire_gen.go
│   ├── init.sql                  # 测试数据库初始化
│   └── {service}_test.go
│
├── main.go                       # 服务入口
├── wire.go                       # Wire 依赖注入定义
└── wire_gen.go                   # Wire 自动生成
```

### 调用链路

```
gRPC Request
    → grpc/{service}.go           (协议转换: proto ↔ domain)
    → service/{entity}.go         (业务逻辑)
    → repository/{entity}.go      (缓存+数据访问)
    → cache/{entity}.go           (Redis 缓存)
    → dao/{entity}.go             (MySQL 数据库)
```

---

## 4. BFF 网关结构

```
bff/
├── handler/                      # HTTP 请求处理器
│   ├── user.go                   # 用户相关接口
│   ├── article.go                # 文章相关接口
│   ├── {resource}.go             # 其他资源
│   ├── middleware/               # HTTP 中间件
│   │   ├── login_jwt.go          # JWT 认证
│   │   └── validate_biz.go       # 业务校验
│   └── jwt/                      # JWT 工具
│       └── redis_jwt.go          # Redis 存储 JWT
│
├── client/                       # gRPC 客户端封装
│   ├── {service}_local.go        # 本地直连
│   └── grey_scale_{service}.go   # 灰度发布客户端
│
├── ioc/                          # 依赖初始化
│   ├── web.go                    # Gin 路由注册
│   ├── {service}.go              # 各 gRPC 客户端初始化
│   └── ...
│
├── job/                          # 定时任务
│   ├── {job_name}_job.go
│   └── robfig_adapter.go         # Cron 适配器
│
├── errs/                         # 错误码定义
├── app.go                        # App 结构体
├── main.go                       # 入口（含 Viper 热加载）
├── wire.go
└── wire_gen.go
```

### BFF App 结构体

```go
type App struct {
    Server    *gin.Engine
    Consumers []saramax.Consumer
    cron      *cron.Cron
}
```

---

## 5. Proto/gRPC 定义模板

### 目录规范

```
api/proto/{service}/v1/{service}.proto    # Proto 定义
api/proto/gen/{service}/v1/               # 生成代码
```

### Proto 文件模板

```protobuf
syntax = "proto3";

package {service}.v1;
option go_package = "/{service}/v1;{service}v1";

import "google/protobuf/timestamp.proto";

// 领域实体消息
message {Entity} {
  int64 id = 1;
  string name = 2;
  // ... 其他字段
  google.protobuf.Timestamp ctime = 10;
  google.protobuf.Timestamp utime = 11;
}

// 服务定义
service {Entity}Service {
  rpc Create(Create{Entity}Request) returns (Create{Entity}Response);
  rpc GetById(Get{Entity}ByIdRequest) returns (Get{Entity}ByIdResponse);
  rpc Update(Update{Entity}Request) returns (Update{Entity}Response);
  rpc List(List{Entity}Request) returns (List{Entity}Response);
}

// 请求/响应消息
message Create{Entity}Request {
  {Entity} {entity} = 1;
}

message Create{Entity}Response {}

message Get{Entity}ByIdRequest {
  int64 id = 1;
}

message Get{Entity}ByIdResponse {
  {Entity} {entity} = 1;
}
```

### buf.gen.yaml

```yaml
version: v1
managed:
  enabled: true
  go_package_prefix:
    default: "{module}/api/proto/gen"
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: api/proto/gen
    opt: paths=source_relative
  - plugin: buf.build/grpc/go
    out: api/proto/gen
    opt:
      - paths=source_relative
```

### 生成命令

```makefile
.PHONY: grpc
grpc:
	@buf generate ./api/proto

.PHONY: grpc_mock
grpc_mock:
	@mockgen -source=./api/proto/gen/{service}/v1/{service}_grpc.pb.go \
		-package={service}mocks \
		-destination=./api/proto/gen/{service}/v1/mocks/{service}_grpc.mock.go
```

---

## 6. Wire 依赖注入模板

### wire.go（服务端）

```go
//go:build wireinject

package main

import (
    "github.com/google/wire"
    "{module}/pkg/grpcx"
    "{module}/{service}/grpc"
    "{module}/{service}/ioc"
    "{module}/{service}/repository"
    "{module}/{service}/repository/cache"
    "{module}/{service}/repository/dao"
    "{module}/{service}/service"
)

// 第三方基础设施
var thirdPartySet = wire.NewSet(
    ioc.InitDB,
    ioc.InitLogger,
    ioc.InitRedis,
    ioc.InitKafka,
    ioc.InitProducer,
)

// 业务依赖链：gRPC → Service → Repository → DAO + Cache
var {service}Set = wire.NewSet(
    grpc.New{Service}GRPCServer,
    service.New{Service}Service,
    repository.NewCached{Service}Repository,
    dao.New{Service}Dao,
    cache.New{Service}Cache,
)

func Init{Service}GRPCServer() *grpcx.Server {
    wire.Build(
        thirdPartySet,
        {service}Set,
        ioc.InitGRPCServer,
    )
    return new(grpcx.Server)
}
```

### 生成方式

```bash
# 安装 wire
go install github.com/google/wire/cmd/wire@latest

# 在服务目录下生成
cd {service} && wire
```

---

## 7. 各层代码模板

### 7.1 Domain（领域层）

```go
package domain

import "time"

// {Entity} 领域对象（DDD Entity）
type {Entity} struct {
    Id      int64
    // ... 业务字段
    Status  {Entity}Status
    Ctime   time.Time
    Utime   time.Time
}

// 领域方法（业务逻辑附着在实体上）
func (e {Entity}) Abstract() string {
    // 业务计算
}

type {Entity}Status uint8

const (
    {Entity}StatusDraft {Entity}Status = iota + 1
    {Entity}StatusPublished
)
```

### 7.2 Service（业务逻辑层）

```go
package service

import (
    "context"
    "{module}/pkg/logger"
    "{module}/{service}/domain"
    "{module}/{service}/events"
    "{module}/{service}/repository"
)

//go:generate mockgen -source=./{entity}.go -package=svcmocks -destination=mocks/{entity}_mock.go
type {Entity}Service interface {
    Create(ctx context.Context, e domain.{Entity}) error
    GetById(ctx context.Context, id int64) (domain.{Entity}, error)
    List(ctx context.Context, offset, limit int) ([]domain.{Entity}, error)
}

type {Entity}ServiceImpl struct {
    repo     repository.{Entity}Repository
    l        logger.LoggerV1
    producer events.Producer
}

func New{Entity}Service(
    repo repository.{Entity}Repository,
    l logger.LoggerV1,
    producer events.Producer,
) {Entity}Service {
    return &{Entity}ServiceImpl{
        repo:     repo,
        l:        l,
        producer: producer,
    }
}

func (svc *{Entity}ServiceImpl) Create(ctx context.Context, e domain.{Entity}) error {
    err := svc.repo.Create(ctx, e)
    if err != nil {
        return err
    }
    // 异步事件（可选）
    go func() {
        er := svc.producer.ProduceSyncEvent(ctx, events.SyncEvent{Id: e.Id})
        if er != nil {
            svc.l.Error("发送事件失败", logger.Error(er))
        }
    }()
    return nil
}
```

### 7.3 Repository（数据访问层，含缓存）

```go
package repository

import (
    "context"
    "{module}/{service}/domain"
    "{module}/{service}/repository/cache"
    "{module}/{service}/repository/dao"
)

//go:generate mockgen -source=./{entity}.go -package=repomocks -destination=mocks/{entity}_mock.go
type {Entity}Repository interface {
    Create(ctx context.Context, e domain.{Entity}) error
    FindById(ctx context.Context, id int64) (domain.{Entity}, error)
}

type Cached{Entity}Repository struct {
    dao   dao.{Entity}Dao
    cache cache.{Entity}Cache
}

func NewCached{Entity}Repository(dao dao.{Entity}Dao, c cache.{Entity}Cache) {Entity}Repository {
    return &Cached{Entity}Repository{dao: dao, cache: c}
}

func (repo *Cached{Entity}Repository) FindById(ctx context.Context, id int64) (domain.{Entity}, error) {
    // 1. 先查缓存
    res, err := repo.cache.Get(ctx, id)
    if err == nil {
        return res, nil
    }
    // 2. 缓存未命中，查数据库
    e, err := repo.dao.FindById(ctx, id)
    if err != nil {
        return domain.{Entity}{}, err
    }
    result := repo.entityToDomain(e)
    // 3. 异步回写缓存
    go func() {
        _ = repo.cache.Set(ctx, result)
    }()
    return result, nil
}

// Domain ↔ DAO Entity 转换
func (repo *Cached{Entity}Repository) entityToDomain(e dao.{Entity}) domain.{Entity} {
    return domain.{Entity}{
        Id: e.Id,
        // ... 字段映射
    }
}

func (repo *Cached{Entity}Repository) domainToEntity(e domain.{Entity}) dao.{Entity} {
    return dao.{Entity}{
        Id: e.Id,
        // ... 字段映射
    }
}
```

### 7.4 DAO（数据库访问对象）

```go
package dao

import (
    "context"
    "errors"
    "time"
    "github.com/go-sql-driver/mysql"
    "gorm.io/gorm"
)

var (
    ErrDuplicate = errors.New("数据已存在")
    ErrNotFound  = gorm.ErrRecordNotFound
)

//go:generate mockgen -source=./{entity}.go -package=daomocks -destination=mocks/{entity}_dao_mock.go
type {Entity}Dao interface {
    Insert(ctx context.Context, e {Entity}) error
    FindById(ctx context.Context, id int64) ({Entity}, error)
    FindByIds(ctx context.Context, ids []int64) ([]{Entity}, error)
    UpdateNonZeroFields(ctx context.Context, e {Entity}) error
}

type Gorm{Entity}Dao struct {
    db *gorm.DB
}

func New{Entity}Dao(db *gorm.DB) {Entity}Dao {
    return &Gorm{Entity}Dao{db: db}
}

func (d *Gorm{Entity}Dao) Insert(ctx context.Context, e {Entity}) error {
    now := time.Now().UnixMilli()
    e.Ctime = now
    e.Utime = now
    err := d.db.WithContext(ctx).Create(&e).Error
    var mysqlErr *mysql.MySQLError
    if errors.As(err, &mysqlErr) {
        const uniqueConflictsErrNo = 1062
        if mysqlErr.Number == uniqueConflictsErrNo {
            return ErrDuplicate
        }
    }
    return err
}

func (d *Gorm{Entity}Dao) FindById(ctx context.Context, id int64) ({Entity}, error) {
    var e {Entity}
    err := d.db.WithContext(ctx).Where("id = ?", id).First(&e).Error
    return e, err
}

// {Entity} 数据库表结构（对应数据库表）
type {Entity} struct {
    Id    int64  `gorm:"primaryKey;autoIncrement"`
    // ... 数据库字段
    Ctime int64  // 毫秒时间戳
    Utime int64
}

// InitTables 自动建表
func InitTables(db *gorm.DB) error {
    return db.AutoMigrate(&{Entity}{})
}
```

### 7.5 Cache（Redis 缓存层）

```go
package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
    "{module}/{service}/domain"
)

var ErrKeyNotExist = redis.Nil

//go:generate mockgen -source=./{entity}.go -package=cachemocks -destination=mocks/{entity}_cache_mock.go
type {Entity}Cache interface {
    Get(ctx context.Context, id int64) (domain.{Entity}, error)
    Set(ctx context.Context, e domain.{Entity}) error
    Delete(ctx context.Context, id int64) error
}

type Redis{Entity}Cache struct {
    client     redis.Cmdable
    expiration time.Duration
}

func New{Entity}Cache(client redis.Cmdable) {Entity}Cache {
    return &Redis{Entity}Cache{
        client:     client,
        expiration: 15 * time.Minute,
    }
}

func (c *Redis{Entity}Cache) Get(ctx context.Context, id int64) (domain.{Entity}, error) {
    key := c.key(id)
    val, err := c.client.Get(ctx, key).Bytes()
    if err != nil {
        return domain.{Entity}{}, err
    }
    var e domain.{Entity}
    err = json.Unmarshal(val, &e)
    return e, err
}

func (c *Redis{Entity}Cache) Set(ctx context.Context, e domain.{Entity}) error {
    val, err := json.Marshal(e)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, c.key(e.Id), val, c.expiration).Err()
}

func (c *Redis{Entity}Cache) Delete(ctx context.Context, id int64) error {
    return c.client.Del(ctx, c.key(id)).Err()
}

func (c *Redis{Entity}Cache) key(id int64) string {
    return fmt.Sprintf("{entity}:info:%d", id)
}
```

### 7.6 gRPC Handler

```go
package grpc

import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/protobuf/types/known/timestamppb"
    {service}v1 "{module}/api/proto/gen/{service}/v1"
    "{module}/{service}/domain"
    "{module}/{service}/service"
)

type {Service}GRPCServer struct {
    {service}v1.Unimplemented{Entity}ServiceServer
    svc service.{Entity}Service
}

func New{Service}GRPCServer(svc service.{Entity}Service) *{Service}GRPCServer {
    return &{Service}GRPCServer{svc: svc}
}

func (s *{Service}GRPCServer) Register(server *grpc.Server) {
    {service}v1.Register{Entity}ServiceServer(server, s)
}

func (s *{Service}GRPCServer) GetById(
    ctx context.Context,
    req *{service}v1.Get{Entity}ByIdRequest,
) (*{service}v1.Get{Entity}ByIdResponse, error) {
    e, err := s.svc.GetById(ctx, req.GetId())
    if err != nil {
        return nil, err
    }
    return &{service}v1.Get{Entity}ByIdResponse{
        {Entity}: s.toDTO(e),
    }, nil
}

// Domain → Proto DTO 转换
func (s *{Service}GRPCServer) toDTO(e domain.{Entity}) *{service}v1.{Entity} {
    return &{service}v1.{Entity}{
        Id:    e.Id,
        Ctime: timestamppb.New(e.Ctime),
    }
}
```

---

## 8. IoC（依赖初始化）模板

### ioc/db.go

```go
package ioc

import (
    "github.com/spf13/viper"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/plugin/opentelemetry/tracing"
    "gorm.io/plugin/prometheus"
    "{module}/pkg/logger"
    "{module}/{service}/repository/dao"
)

func InitDB(l logger.LoggerV1) *gorm.DB {
    type Config struct {
        DSN string `yaml:"dsn"`
    }
    var cfg Config = Config{
        DSN: "root:root@tcp(localhost:3306)/webook?charset=utf8mb4&parseTime=True&loc=Local",
    }
    err := viper.UnmarshalKey("db.mysql", &cfg)
    if err != nil {
        panic(err)
    }
    db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
    if err != nil {
        panic("failed to connect database")
    }
    // Prometheus 监控
    db.Use(prometheus.New(prometheus.Config{
        DBName:          "{service}",
        RefreshInterval: 15,
        StartServer:     false,
    }))
    // OpenTelemetry 链路追踪
    db.Use(tracing.NewPlugin(tracing.WithDBName("{service}")))
    // 自动建表
    err = dao.InitTables(db)
    if err != nil {
        panic(err)
    }
    return db
}
```

### ioc/redis.go

```go
package ioc

import (
    "github.com/redis/go-redis/v9"
    "github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
    type Config struct {
        Addr string `yaml:"addr"`
    }
    var cfg Config
    err := viper.UnmarshalKey("redis", &cfg)
    if err != nil {
        panic(err)
    }
    return redis.NewClient(&redis.Options{Addr: cfg.Addr})
}
```

### ioc/kafka.go

```go
package ioc

import (
    "github.com/IBM/sarama"
    "github.com/spf13/viper"
    "{module}/{service}/events"
)

func InitKafka() sarama.Client {
    type Config struct {
        Addrs []string `yaml:"addrs"`
    }
    saramaCfg := sarama.NewConfig()
    saramaCfg.Producer.Return.Successes = true
    saramaCfg.Producer.Partitioner = sarama.NewConsistentCRCHashPartitioner
    var cfg Config
    err := viper.UnmarshalKey("kafka", &cfg)
    if err != nil {
        panic(err)
    }
    client, err := sarama.NewClient(cfg.Addrs, saramaCfg)
    if err != nil {
        panic(err)
    }
    return client
}

func InitProducer(client sarama.Client) events.Producer {
    res, err := events.NewSaraSyncProducer(client)
    if err != nil {
        panic(err)
    }
    return res
}
```

### ioc/logger.go

```go
package ioc

import (
    "go.uber.org/zap"
    "{module}/pkg/logger"
)

func InitLogger() logger.LoggerV1 {
    l, err := zap.NewDevelopment()
    if err != nil {
        panic(err)
    }
    return logger.NewZapLogger(l)
}
```

### ioc/grpc.go

```go
package ioc

import (
    "github.com/spf13/viper"
    "google.golang.org/grpc"
    "{module}/pkg/grpcx"
    "{module}/pkg/logger"
    igrpc "{module}/{service}/grpc"
)

func InitGRPCServer(server *igrpc.{Service}GRPCServer, l logger.LoggerV1) *grpcx.Server {
    type Config struct {
        Port      int      `yaml:"port"`
        EtcdAddrs []string `yaml:"etcdAddrs"`
    }
    var cfg Config
    err := viper.UnmarshalKey("grpc.server", &cfg)
    if err != nil {
        panic(err)
    }
    s := grpc.NewServer()
    server.Register(s)
    return &grpcx.Server{
        Server:    s,
        Port:      cfg.Port,
        EtcdAddrs: cfg.EtcdAddrs,
        Name:      "{service}",
        L:         l,
    }
}
```

---

## 9. 事件驱动模板（Kafka）

### events/types.go

```go
package events

type SyncEvent struct {
    Id   int64  `json:"id"`
    // ... 事件字段
}
```

### events/producer.go

```go
package events

import (
    "context"
    "encoding/json"
    "github.com/IBM/sarama"
)

type Producer interface {
    ProduceSyncEvent(ctx context.Context, evt SyncEvent) error
}

type SaraSyncProducer struct {
    client sarama.SyncProducer
}

func NewSaraSyncProducer(client sarama.Client) (Producer, error) {
    p, err := sarama.NewSyncProducerFromClient(client)
    if err != nil {
        return nil, err
    }
    return &SaraSyncProducer{p}, nil
}

func (s *SaraSyncProducer) ProduceSyncEvent(ctx context.Context, evt SyncEvent) error {
    data, _ := json.Marshal(evt)
    msg := &sarama.ProducerMessage{
        Topic: "sync_{service}_event",
        Value: sarama.ByteEncoder(data),
    }
    _, _, err := s.client.SendMessage(msg)
    return err
}
```

### events/consumer.go

```go
package events

import (
    "context"
    "github.com/IBM/sarama"
    "{module}/pkg/logger"
    "{module}/pkg/saramax"
)

type Consumer struct {
    client sarama.Client
    l      logger.LoggerV1
    // ... 依赖的 service 或 repository
}

func NewConsumer(client sarama.Client, l logger.LoggerV1) *Consumer {
    return &Consumer{client: client, l: l}
}

func (c *Consumer) Start() error {
    cg, err := sarama.NewConsumerGroupFromClient("group_{service}", c.client)
    if err != nil {
        return err
    }
    go func() {
        err := cg.Consume(
            context.Background(),
            []string{"topic_name"},
            saramax.NewHandler[SyncEvent](c.l, c.Consume),
        )
        if err != nil {
            c.l.Error("退出消费循环", logger.Error(err))
        }
    }()
    return nil
}

func (c *Consumer) Consume(msg *sarama.ConsumerMessage, evt SyncEvent) error {
    // 处理消费逻辑
    return nil
}
```

---

## 10. 配置系统模板

### config/dev.yaml（服务配置）

```yaml
db:
  mysql:
    dsn: "root:root@tcp(localhost:13306)/{service}?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "localhost:6379"

kafka:
  addrs:
    - "localhost:9094"

grpc:
  server:
    port: 8091
    etcdAddrs:
      - "localhost:12379"
```

### main.go（Viper 加载）

```go
package main

import (
    "fmt"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

func main() {
    initViper()
    server := Init{Service}GRPCServer()
    err := server.Serve()
    if err != nil {
        panic(err)
    }
}

func initViper() {
    file := pflag.String("config", "config/dev.yaml", "配置文件路径")
    pflag.Parse()
    viper.SetConfigFile(*file)
    err := viper.ReadInConfig()
    if err != nil {
        panic(fmt.Errorf("Fatal error config file: %s \n", err))
    }
}
```

---

## 11. 测试模板

### 单元测试（Table-Driven）

```go
package service

import (
    "context"
    "testing"
    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/assert"
    "{module}/{service}/repository/mocks"
)

func TestCreate(t *testing.T) {
    testCases := []struct {
        name    string
        mock    func(ctrl *gomock.Controller) repository.{Entity}Repository
        wantErr error
    }{
        {
            name: "创建成功",
            mock: func(ctrl *gomock.Controller) repository.{Entity}Repository {
                repo := repomocks.NewMock{Entity}Repository(ctrl)
                repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
                return repo
            },
            wantErr: nil,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            svc := New{Entity}Service(tc.mock(ctrl), nil, nil)
            err := svc.Create(context.Background(), domain.{Entity}{})
            assert.Equal(t, tc.wantErr, err)
        })
    }
}
```

### 集成测试（startup/wire.go）

```go
//go:build wireinject

package startup

import (
    "github.com/google/wire"
    "{module}/{service}/repository"
    "{module}/{service}/repository/cache"
    "{module}/{service}/repository/dao"
    "{module}/{service}/service"
)

func Init{Service}Service() service.{Entity}Service {
    wire.Build(
        InitDB,
        InitRedis,
        InitLog,
        dao.New{Entity}Dao,
        cache.New{Entity}Cache,
        repository.NewCached{Entity}Repository,
        service.New{Entity}Service,
    )
    return nil
}
```

---

## 12. 基础设施模板

### docker-compose.yaml

```yaml
services:
  mysql8:
    image: mysql:8.4.2
    command:
      - --binlog-format=ROW
      - --server-id=1
    environment:
      MYSQL_ROOT_PASSWORD: root
    volumes:
      - ./script/mysql/:/docker-entrypoint-initdb.d/
    ports:
      - "13306:3306"

  redis:
    image: redis:7.4.0
    ports:
      - "6379:6379"

  etcd:
    image: bitnami/etcd:3.4.34
    environment:
      - ALLOW_NONE_AUTHENTICATION=yes
    ports:
      - "12379:2379"

  kafka:
    image: bitnami/kafka:3.6
    ports:
      - "9092:9092"
      - "9094:9094"
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093,EXTERNAL://0.0.0.0:9094
      - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092,EXTERNAL://localhost:9094
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,EXTERNAL:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER

  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yaml:/etc/prometheus/prometheus.yml
    command:
      - "--web.enable-remote-write-receiver"
      - "--config.file=/etc/prometheus/prometheus.yml"
    ports:
      - "9091:9090"

  grafana:
    image: grafana/grafana-enterprise
    ports:
      - "3000:3000"

  zipkin:
    image: openzipkin/zipkin-slim
    ports:
      - "9411:9411"

  elasticsearch:
    image: elasticsearch:8.15.5
    environment:
      - "discovery.type=single-node"
      - "xpack.security.enabled=false"
    ports:
      - "9200:9200"
```

### Makefile

```makefile
# 服务列表
APPS := user article interactive sms code search bff follow comment

# 构建所有服务
.PHONY: build $(APPS)
build: $(APPS)
$(APPS): %:
	@rm ./cmd/$@ || true
	@GOOS=linux GOARCH=arm64 go build -o ./cmd ./$@

# Docker 镜像
.PHONY: docker
docker:
	@rm webook || true
	@go mod tidy
	@GOOS=linux GOARCH=arm64 go build -tags=k8s -o webook
	@docker build -t {registry}/{project}:v0.0.1 .

# Proto 生成
.PHONY: grpc
grpc:
	@buf generate ./api/proto

# Mock 生成
.PHONY: grpc_mock
grpc_mock:
	@mockgen -source=./api/proto/gen/{service}/v1/{service}_grpc.pb.go \
		-package={service}mocks \
		-destination=./api/proto/gen/{service}/v1/mocks/{service}_grpc.mock.go
```

### Dockerfile

```dockerfile
FROM ubuntu:20.04
COPY {binary} /app/{binary}
COPY config /app/config
WORKDIR /app
ENTRYPOINT ["/app/{binary}"]
```

---

## 13. 新建微服务 Checklist

创建新微服务 `{service}` 的步骤：

1. **定义 Proto**
   - [ ] 创建 `api/proto/{service}/v1/{service}.proto`
   - [ ] 运行 `make grpc` 生成代码

2. **创建服务目录**
   - [ ] `{service}/domain/{entity}.go` — 领域实体
   - [ ] `{service}/repository/dao/{entity}.go` — DAO + 表结构
   - [ ] `{service}/repository/dao/init.go` — InitTables
   - [ ] `{service}/repository/cache/{entity}.go` — Redis 缓存
   - [ ] `{service}/repository/{entity}.go` — CachedRepository
   - [ ] `{service}/service/{entity}.go` — Service 接口与实现
   - [ ] `{service}/grpc/{service}.go` — gRPC Handler

3. **事件驱动（可选）**
   - [ ] `{service}/events/types.go` — 事件定义
   - [ ] `{service}/events/producer.go` — Kafka Producer
   - [ ] `{service}/events/consumer.go` — Kafka Consumer

4. **依赖注入**
   - [ ] `{service}/ioc/db.go`
   - [ ] `{service}/ioc/redis.go`
   - [ ] `{service}/ioc/kafka.go`
   - [ ] `{service}/ioc/logger.go`
   - [ ] `{service}/ioc/grpc.go`
   - [ ] `{service}/wire.go` — Wire Build
   - [ ] 运行 `cd {service} && wire` 生成 `wire_gen.go`

5. **配置**
   - [ ] `{service}/config/dev.yaml`
   - [ ] `{service}/main.go`

6. **测试**
   - [ ] `{service}/service/{entity}_test.go` — 单元测试
   - [ ] `{service}/integration/startup/wire.go` — 集成测试 Wire
   - [ ] `{service}/integration/{service}_test.go` — 集成测试

7. **BFF 接入（如需 HTTP 暴露）**
   - [ ] `bff/ioc/{service}.go` — gRPC 客户端初始化
   - [ ] `bff/handler/{service}.go` — HTTP Handler
   - [ ] 在 `bff/ioc/web.go` 注册路由
   - [ ] 更新 `bff/wire.go`

8. **基础设施**
   - [ ] 更新 `Makefile` 的 APPS 列表
   - [ ] 添加 K8s 部署清单（如需）

---

## 14. 核心设计原则

| 原则 | 实践 |
|------|------|
| **面向接口编程** | 每层定义 interface，实现分离 |
| **依赖注入** | 不在内部 new 依赖，全部通过构造函数注入 |
| **依赖反转** | 高层不依赖低层细节，都依赖抽象 |
| **Cache-Aside** | Repository 层管理缓存，先查缓存再查库，异步回写 |
| **Domain ↔ Entity 隔离** | domain 包不引用 dao 包，通过 Repository 做转换 |
| **mockgen 生成 Mock** | 接口文件头部加 `//go:generate mockgen` 注释 |
| **毫秒时间戳存储** | 数据库存 `int64` 毫秒时间戳，domain 用 `time.Time` |
| **错误透传** | DAO 错误通过 Repository 透传到 Service，各层定义语义化别名 |
| **gRPC ↔ Domain 转换** | grpc handler 层负责 proto message 与 domain 的互转 |
