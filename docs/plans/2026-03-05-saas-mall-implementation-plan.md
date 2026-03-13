# SaaS 多租户商城 — 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 构建一个基于 Go 微服务架构的 SaaS 多租户商城系统，包含 11 个 gRPC 微服务 + 3 个 BFF 网关。

**Architecture:** DDD 分层架构，gRPC 服务间通信，Kafka 事件编舞，etcd 服务发现，共享数据库 + tenant_id 多租户隔离。每个服务遵循 `domain → service → repository → dao/cache` 分层，Wire 依赖注入。

**Tech Stack:** Go 1.23, Gin, GORM, gRPC, Redis, Kafka, etcd, Elasticsearch, Wire, Viper, Zap, Prometheus, OpenTelemetry, Docker

**Design Doc:** `docs/plans/2026-03-05-saas-mall-design.md`
**Template Reference:** `docs/template.md`

---

## Phase 0: 项目脚手架 & 基础设施

> 目标：初始化 Go Module、docker-compose、Makefile、buf 配置、共享工具包骨架。

### Task 0.1: 初始化 Go Module & 根目录结构

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `buf.gen.yaml`
- Create: `buf.yaml`
- Create: `.gitignore`

**Step 1: 初始化 Go module**

```bash
cd mall
go mod init github.com/your-username/mall
```

**Step 2: 创建 .gitignore**

```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
cmd/

# Wire generated
# wire_gen.go  # 需要提交 wire_gen.go

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Env
*.env
```

**Step 3: 创建 buf.yaml**

```yaml
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
```

**Step 4: 创建 buf.gen.yaml**

```yaml
version: v1
managed:
  enabled: true
  go_package_prefix:
    default: "github.com/your-username/mall/api/proto/gen"
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: api/proto/gen
    opt: paths=source_relative
  - plugin: buf.build/grpc/go
    out: api/proto/gen
    opt:
      - paths=source_relative
```

**Step 5: 创建 Makefile**

```makefile
# 服务列表
APPS := user tenant product inventory order payment cart search marketing logistics notification
BFFS := admin-bff merchant-bff consumer-bff

# Proto 生成
.PHONY: grpc
grpc:
	@buf generate ./api/proto

# Mock 生成（用法：make mock SVC=user）
.PHONY: mock
mock:
	@find ./$(SVC) -name "*.go" | xargs grep -l "go:generate" | xargs -I {} go generate {}

# 构建单个服务（用法：make build SVC=user）
.PHONY: build
build:
	@mkdir -p cmd
	@GOOS=linux GOARCH=arm64 go build -o ./cmd/$(SVC) ./$(SVC)

# 构建所有服务
.PHONY: build-all
build-all:
	@for app in $(APPS) $(BFFS); do \
		echo "Building $$app..."; \
		mkdir -p cmd; \
		go build -o ./cmd/$$app ./$$app; \
	done

# 启动基础设施
.PHONY: infra-up
infra-up:
	docker-compose up -d

.PHONY: infra-down
infra-down:
	docker-compose down

# 运行单个服务（用法：make run SVC=user）
.PHONY: run
run:
	go run ./$(SVC)/main.go

# Wire 生成（用法：make wire SVC=user）
.PHONY: wire
wire:
	cd $(SVC) && wire

# 测试
.PHONY: test
test:
	go test ./... -v -count=1

.PHONY: test-svc
test-svc:
	go test ./$(SVC)/... -v -count=1
```

**Step 6: 创建目录骨架**

```bash
# 创建所有顶层目录
mkdir -p api/proto/{user,tenant,product,inventory,order,payment,cart,search,marketing,logistics,notification}/v1
mkdir -p api/proto/gen
mkdir -p pkg/{logger,grpcx,ginx,gormx,saramax,redisx,ratelimit,snowflake,tenantx,cronjobx,migrator,canalx}
mkdir -p config script docs
mkdir -p {user,tenant,product,inventory,order,payment,cart,search,marketing,logistics,notification}/{config,domain,service,repository/{cache,dao},grpc,events,ioc,integration/startup}
mkdir -p {admin-bff,merchant-bff,consumer-bff}/{handler/{middleware,jwt},client,ioc,errs}
```

**Step 7: Commit**

```bash
git init
git add -A
git commit -m "chore: init project structure and build tooling"
```

---

### Task 0.2: Docker Compose 基础设施

**Files:**
- Create: `docker-compose.yaml`
- Create: `prometheus.yaml`
- Create: `script/mysql/init.sql`

**Step 1: 创建 docker-compose.yaml**

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

  elasticsearch:
    image: elasticsearch:8.15.5
    environment:
      - "discovery.type=single-node"
      - "xpack.security.enabled=false"
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ports:
      - "9200:9200"

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
```

**Step 2: 创建 MySQL 初始化脚本**

```sql
-- script/mysql/init.sql
CREATE DATABASE IF NOT EXISTS mall_user;
CREATE DATABASE IF NOT EXISTS mall_tenant;
CREATE DATABASE IF NOT EXISTS mall_product;
CREATE DATABASE IF NOT EXISTS mall_inventory;
CREATE DATABASE IF NOT EXISTS mall_order;
CREATE DATABASE IF NOT EXISTS mall_payment;
CREATE DATABASE IF NOT EXISTS mall_cart;
CREATE DATABASE IF NOT EXISTS mall_search;
CREATE DATABASE IF NOT EXISTS mall_marketing;
CREATE DATABASE IF NOT EXISTS mall_logistics;
CREATE DATABASE IF NOT EXISTS mall_notification;
```

**Step 3: 创建 prometheus.yaml**

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "mall-services"
    static_configs:
      - targets:
          - "host.docker.internal:8081"   # user-svc metrics
          - "host.docker.internal:8082"   # tenant-svc metrics
          - "host.docker.internal:8083"   # product-svc metrics
          - "host.docker.internal:8084"   # inventory-svc metrics
          - "host.docker.internal:8085"   # order-svc metrics
          - "host.docker.internal:8086"   # payment-svc metrics
          - "host.docker.internal:8087"   # cart-svc metrics
          - "host.docker.internal:8088"   # search-svc metrics
          - "host.docker.internal:8089"   # marketing-svc metrics
          - "host.docker.internal:8090"   # logistics-svc metrics
          - "host.docker.internal:8091"   # notification-svc metrics
```

**Step 4: 验证基础设施启动**

```bash
make infra-up
# 等待所有容器就绪后
docker-compose ps
# 验证 MySQL
mysql -h127.0.0.1 -P13306 -uroot -proot -e "SHOW DATABASES;"
```

**Step 5: Commit**

```bash
git add docker-compose.yaml prometheus.yaml script/
git commit -m "chore: add docker-compose infrastructure"
```

---

## Phase 1: 共享工具包 (pkg/)

> 目标：实现所有服务共用的基础库，后续各服务直接引用。

### Task 1.1: Logger 日志接口

**Files:**
- Create: `pkg/logger/types.go`
- Create: `pkg/logger/zap_logger.go`

**Step 1: 定义日志接口**

```go
// pkg/logger/types.go
package logger

type LoggerV1 interface {
    Debug(msg string, args ...Field)
    Info(msg string, args ...Field)
    Warn(msg string, args ...Field)
    Error(msg string, args ...Field)
}

type Field struct {
    Key   string
    Value any
}

func String(key, val string) Field {
    return Field{Key: key, Value: val}
}

func Int64(key string, val int64) Field {
    return Field{Key: key, Value: val}
}

func Error(err error) Field {
    return Field{Key: "error", Value: err}
}
```

**Step 2: 实现 Zap Logger**

```go
// pkg/logger/zap_logger.go
package logger

import "go.uber.org/zap"

type ZapLogger struct {
    l *zap.Logger
}

func NewZapLogger(l *zap.Logger) LoggerV1 {
    return &ZapLogger{l: l}
}

func (z *ZapLogger) Debug(msg string, args ...Field) {
    z.l.Debug(msg, z.toZapFields(args)...)
}

func (z *ZapLogger) Info(msg string, args ...Field) {
    z.l.Info(msg, z.toZapFields(args)...)
}

func (z *ZapLogger) Warn(msg string, args ...Field) {
    z.l.Warn(msg, z.toZapFields(args)...)
}

func (z *ZapLogger) Error(msg string, args ...Field) {
    z.l.Error(msg, z.toZapFields(args)...)
}

func (z *ZapLogger) toZapFields(args []Field) []zap.Field {
    res := make([]zap.Field, 0, len(args))
    for _, arg := range args {
        res = append(res, zap.Any(arg.Key, arg.Value))
    }
    return res
}
```

**Step 3: Commit**

```bash
git add pkg/logger/
git commit -m "feat(pkg): add logger interface and zap implementation"
```

---

### Task 1.2: gRPC 扩展 (etcd 服务注册)

**Files:**
- Create: `pkg/grpcx/server.go`

**Step 1: 实现带 etcd 注册的 gRPC Server 封装**

```go
// pkg/grpcx/server.go
package grpcx

import (
    "context"
    "fmt"
    "net"
    "time"

    "github.com/your-username/mall/pkg/logger"
    "github.com/your-username/mall/pkg/netx"
    clientv3 "go.etcd.io/etcd/client/v3"
    "go.etcd.io/etcd/client/v3/naming/endpoints"
    "google.golang.org/grpc"
)

type Server struct {
    *grpc.Server
    Port      int
    EtcdAddrs []string
    Name      string
    L         logger.LoggerV1
    kaCancel  func()
    em        endpoints.Manager
    client    *clientv3.Client
}

func (s *Server) Serve() error {
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
    if err != nil {
        return err
    }
    err = s.register()
    if err != nil {
        return err
    }
    return s.Server.Serve(lis)
}

func (s *Server) register() error {
    client, err := clientv3.New(clientv3.Config{
        Endpoints: s.EtcdAddrs,
    })
    if err != nil {
        return err
    }
    s.client = client
    em, err := endpoints.NewManager(client, "service/"+s.Name)
    if err != nil {
        return err
    }
    s.em = em

    ip := netx.GetOutboundIP()
    addr := fmt.Sprintf("%s:%d", ip, s.Port)
    key := fmt.Sprintf("service/%s/%s", s.Name, addr)

    ctx, cancel := context.WithCancel(context.Background())
    s.kaCancel = cancel

    // 租约保活
    leaseResp, err := client.Grant(ctx, 30)
    if err != nil {
        return err
    }
    err = em.AddEndpoint(ctx, key, endpoints.Endpoint{Addr: addr}, clientv3.WithLease(leaseResp.ID))
    if err != nil {
        return err
    }

    ch, err := client.KeepAlive(ctx, leaseResp.ID)
    if err != nil {
        return err
    }
    go func() {
        for range ch {
            // 消费 keepalive 响应
        }
    }()

    return nil
}

func (s *Server) Close() error {
    if s.kaCancel != nil {
        s.kaCancel()
    }
    if s.em != nil {
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()
        ip := netx.GetOutboundIP()
        addr := fmt.Sprintf("%s:%d", ip, s.Port)
        key := fmt.Sprintf("service/%s/%s", s.Name, addr)
        _ = s.em.DeleteEndpoint(ctx, key)
    }
    if s.client != nil {
        _ = s.client.Close()
    }
    s.Server.GracefulStop()
    return nil
}
```

**Step 2: 创建网络工具**

```go
// pkg/netx/ip.go
package netx

import "net"

func GetOutboundIP() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        return "127.0.0.1"
    }
    defer conn.Close()
    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String()
}
```

**Step 3: Commit**

```bash
git add pkg/grpcx/ pkg/netx/
git commit -m "feat(pkg): add grpcx server with etcd registration"
```

---

### Task 1.3: Kafka 封装 (saramax)

**Files:**
- Create: `pkg/saramax/handler.go`
- Create: `pkg/saramax/consumer.go`
- Create: `pkg/saramax/batch_handler.go`

**Step 1: 实现泛型消息 Handler**

```go
// pkg/saramax/handler.go
package saramax

import (
    "encoding/json"

    "github.com/IBM/sarama"
    "github.com/your-username/mall/pkg/logger"
)

type Handler[T any] struct {
    l  logger.LoggerV1
    fn func(msg *sarama.ConsumerMessage, event T) error
}

func NewHandler[T any](l logger.LoggerV1, fn func(msg *sarama.ConsumerMessage, event T) error) *Handler[T] {
    return &Handler[T]{l: l, fn: fn}
}

func (h *Handler[T]) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *Handler[T]) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        var t T
        err := json.Unmarshal(msg.Value, &t)
        if err != nil {
            h.l.Error("反序列化消息失败",
                logger.String("topic", msg.Topic),
                logger.Error(err))
            session.MarkMessage(msg, "")
            continue
        }
        err = h.fn(msg, t)
        if err != nil {
            h.l.Error("处理消息失败",
                logger.String("topic", msg.Topic),
                logger.Error(err))
        }
        session.MarkMessage(msg, "")
    }
    return nil
}
```

**Step 2: 定义 Consumer 接口**

```go
// pkg/saramax/consumer.go
package saramax

type Consumer interface {
    Start() error
}
```

**Step 3: Commit**

```bash
git add pkg/saramax/
git commit -m "feat(pkg): add saramax kafka consumer handler"
```

---

### Task 1.4: Gin 中间件扩展 (ginx)

**Files:**
- Create: `pkg/ginx/wrapper.go`
- Create: `pkg/ginx/middleware/cors.go`

**Step 1: 实现统一响应包装**

```go
// pkg/ginx/wrapper.go
package ginx

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/your-username/mall/pkg/logger"
)

type Result struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
    Data any    `json:"data,omitempty"`
}

func WrapBody[Req any](l logger.LoggerV1, fn func(ctx *gin.Context, req Req) (Result, error)) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        var req Req
        if err := ctx.Bind(&req); err != nil {
            ctx.JSON(http.StatusBadRequest, Result{Code: 4, Msg: "参数错误"})
            return
        }
        res, err := fn(ctx, req)
        if err != nil {
            l.Error("业务处理错误", logger.Error(err))
            ctx.JSON(http.StatusOK, Result{Code: 5, Msg: "系统错误"})
            return
        }
        ctx.JSON(http.StatusOK, res)
    }
}

func WrapBodyV2[Req any](fn func(ctx *gin.Context, req Req) (Result, error)) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        var req Req
        if err := ctx.Bind(&req); err != nil {
            ctx.JSON(http.StatusBadRequest, Result{Code: 4, Msg: "参数错误"})
            return
        }
        res, err := fn(ctx, req)
        if err != nil {
            ctx.JSON(http.StatusOK, Result{Code: 5, Msg: "系统错误"})
            return
        }
        ctx.JSON(http.StatusOK, res)
    }
}
```

**Step 2: Commit**

```bash
git add pkg/ginx/
git commit -m "feat(pkg): add ginx response wrapper and middlewares"
```

---

### Task 1.5: 雪花 ID 生成器

**Files:**
- Create: `pkg/snowflake/snowflake.go`

**Step 1: 实现雪花算法（或封装 sony/sonyflake）**

```go
// pkg/snowflake/snowflake.go
package snowflake

import (
    "sync"
    "time"
)

const (
    workerBits     = 10
    sequenceBits   = 12
    workerMax      = -1 ^ (-1 << workerBits)
    sequenceMax    = -1 ^ (-1 << sequenceBits)
    timeShift      = workerBits + sequenceBits
    workerShift    = sequenceBits
)

// epoch: 2024-01-01 00:00:00 UTC
var epoch int64 = 1704067200000

type Node struct {
    mu        sync.Mutex
    timestamp int64
    workerID  int64
    sequence  int64
}

func NewNode(workerID int64) (*Node, error) {
    if workerID < 0 || workerID > workerMax {
        return nil, fmt.Errorf("worker ID must be between 0 and %d", workerMax)
    }
    return &Node{workerID: workerID}, nil
}

func (n *Node) Generate() int64 {
    n.mu.Lock()
    defer n.mu.Unlock()

    now := time.Now().UnixMilli()
    if now == n.timestamp {
        n.sequence = (n.sequence + 1) & sequenceMax
        if n.sequence == 0 {
            for now <= n.timestamp {
                now = time.Now().UnixMilli()
            }
        }
    } else {
        n.sequence = 0
    }
    n.timestamp = now
    return (now-epoch)<<timeShift | n.workerID<<workerShift | n.sequence
}
```

> 注：也可以直接使用 `github.com/bwmarrin/snowflake`，面试时能讲清原理即可。

**Step 2: Commit**

```bash
git add pkg/snowflake/
git commit -m "feat(pkg): add snowflake ID generator"
```

---

### Task 1.6: 多租户中间件

**Files:**
- Create: `pkg/tenantx/middleware.go`
- Create: `pkg/tenantx/context.go`

**Step 1: 实现 tenant_id 提取与注入**

```go
// pkg/tenantx/context.go
package tenantx

import "context"

type tenantKey struct{}

func WithTenantID(ctx context.Context, tenantID int64) context.Context {
    return context.WithValue(ctx, tenantKey{}, tenantID)
}

func GetTenantID(ctx context.Context) int64 {
    val, ok := ctx.Value(tenantKey{}).(int64)
    if !ok {
        return 0
    }
    return val
}
```

```go
// pkg/tenantx/middleware.go
package tenantx

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

// GinMiddleware 从 JWT Claims 中提取 tenant_id 注入到 context
// 实际使用时 tenant_id 从 JWT 中解析，这里简化从 header 读取
func GinMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tidStr := c.GetHeader("X-Tenant-ID")
        if tidStr == "" {
            // 也可以从 JWT claims 中读取
            tidStr = "0"
        }
        tid, err := strconv.ParseInt(tidStr, 10, 64)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"msg": "invalid tenant id"})
            return
        }
        ctx := WithTenantID(c.Request.Context(), tid)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
```

**Step 2: Commit**

```bash
git add pkg/tenantx/
git commit -m "feat(pkg): add multi-tenant context middleware"
```

---

### Task 1.7: 限流器

**Files:**
- Create: `pkg/ratelimit/types.go`
- Create: `pkg/ratelimit/redis_slide_window.go`

**Step 1: 实现 Redis 滑动窗口限流**

```go
// pkg/ratelimit/types.go
package ratelimit

import "context"

type Limiter interface {
    // Limit 返回 true 表示被限流
    Limit(ctx context.Context, key string) (bool, error)
}
```

```go
// pkg/ratelimit/redis_slide_window.go
package ratelimit

import (
    "context"
    _ "embed"
    "time"

    "github.com/redis/go-redis/v9"
)

//go:embed lua/slide_window.lua
var luaSlideWindow string

type RedisSlideWindowLimiter struct {
    client   redis.Cmdable
    interval time.Duration
    rate     int // 窗口内最大请求数
}

func NewRedisSlideWindowLimiter(client redis.Cmdable, interval time.Duration, rate int) Limiter {
    return &RedisSlideWindowLimiter{client: client, interval: interval, rate: rate}
}

func (r *RedisSlideWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
    return r.client.Eval(ctx, luaSlideWindow, []string{key},
        r.interval.Milliseconds(), r.rate, time.Now().UnixMilli()).Bool()
}
```

```lua
-- pkg/ratelimit/lua/slide_window.lua
local key = KEYS[1]
local window = tonumber(ARGV[1])
local threshold = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local cnt = redis.call('ZCARD', key)
if cnt >= threshold then
    return "true"
else
    redis.call('ZADD', key, now, now)
    redis.call('PEXPIRE', key, window)
    return "false"
end
```

**Step 2: Commit**

```bash
git add pkg/ratelimit/
git commit -m "feat(pkg): add redis sliding window rate limiter"
```

---

### Task 1.8: 安装所有依赖 & go mod tidy

**Step 1: 安装核心依赖**

```bash
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/mysql
go get github.com/redis/go-redis/v9
go get google.golang.org/grpc
go get google.golang.org/protobuf
go get github.com/IBM/sarama
go get go.etcd.io/etcd/client/v3
go get github.com/google/wire
go get github.com/spf13/viper
go get github.com/spf13/pflag
go get go.uber.org/zap
go get github.com/prometheus/client_golang
go get go.opentelemetry.io/otel
go get gorm.io/plugin/prometheus
go get gorm.io/plugin/opentelemetry/tracing
go get github.com/golang/mock/gomock
go get github.com/stretchr/testify
go mod tidy
```

**Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add all dependencies"
```

---

## Phase 2: user-svc（用户服务）— 完整示范

> 这是第一个完整实现的服务，后续服务均参考此模式。
> 完整展示 Proto → Domain → DAO → Cache → Repository → Service → gRPC → Events → IoC → Wire → main.go 全流程。

### Task 2.1: 定义 user Proto

**Files:**
- Create: `api/proto/user/v1/user.proto`

**Step 1: 编写 Proto**

```protobuf
syntax = "proto3";

package user.v1;
option go_package = "/user/v1;userv1";

import "google/protobuf/timestamp.proto";

message User {
  int64 id = 1;
  string phone = 2;
  string email = 3;
  string nickname = 4;
  string avatar = 5;
  int32 status = 6;
  google.protobuf.Timestamp ctime = 7;
  google.protobuf.Timestamp utime = 8;
}

service UserService {
  rpc Signup(SignupRequest) returns (SignupResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc FindById(FindByIdRequest) returns (FindByIdResponse);
  rpc FindByPhone(FindByPhoneRequest) returns (FindByPhoneResponse);
  rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileResponse);

  // RBAC
  rpc GetPermissions(GetPermissionsRequest) returns (GetPermissionsResponse);
  rpc AssignRole(AssignRoleRequest) returns (AssignRoleResponse);
}

message SignupRequest {
  string phone = 1;
  string email = 2;
  string password = 3;
}
message SignupResponse { int64 id = 1; }

message LoginRequest {
  string phone = 1;
  string password = 2;
}
message LoginResponse {
  User user = 1;
}

message FindByIdRequest { int64 id = 1; }
message FindByIdResponse { User user = 1; }

message FindByPhoneRequest { string phone = 1; }
message FindByPhoneResponse { User user = 1; }

message UpdateProfileRequest {
  int64 id = 1;
  string nickname = 2;
  string avatar = 3;
}
message UpdateProfileResponse {}

message GetPermissionsRequest {
  int64 user_id = 1;
  int64 tenant_id = 2;
}
message GetPermissionsResponse {
  repeated string permissions = 1;
}

message AssignRoleRequest {
  int64 user_id = 1;
  int64 tenant_id = 2;
  int64 role_id = 3;
}
message AssignRoleResponse {}
```

**Step 2: 生成代码**

```bash
make grpc
```

**Step 3: Commit**

```bash
git add api/proto/
git commit -m "feat(user): define user.proto and generate gRPC code"
```

---

### Task 2.2: User Domain 层

**Files:**
- Create: `user/domain/user.go`

**Step 1: 定义领域实体**

```go
// user/domain/user.go
package domain

import "time"

type User struct {
    Id       int64
    Phone    string
    Email    string
    Password string
    Nickname string
    Avatar   string
    Status   UserStatus
    Ctime    time.Time
    Utime    time.Time
}

type UserStatus uint8

const (
    UserStatusNormal  UserStatus = 1
    UserStatusFrozen  UserStatus = 2
    UserStatusDeleted UserStatus = 3
)

type Role struct {
    Id          int64
    TenantId    int64
    Name        string
    Code        string
    Description string
}

type Permission struct {
    Id       int64
    Code     string
    Name     string
    Type     PermissionType
    Resource string
}

type PermissionType uint8

const (
    PermissionTypeMenu   PermissionType = 1
    PermissionTypeButton PermissionType = 2
    PermissionTypeAPI    PermissionType = 3
)

type UserAddress struct {
    Id        int64
    UserId    int64
    Name      string
    Phone     string
    Province  string
    City      string
    District  string
    Detail    string
    IsDefault bool
}
```

**Step 2: Commit**

```bash
git add user/domain/
git commit -m "feat(user): add domain entities"
```

---

### Task 2.3: User DAO 层

**Files:**
- Create: `user/repository/dao/user.go`
- Create: `user/repository/dao/init.go`

**Step 1: 定义表结构和 DAO 接口**

```go
// user/repository/dao/user.go
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

//go:generate mockgen -source=./user.go -package=daomocks -destination=mocks/user_dao_mock.go
type UserDao interface {
    Insert(ctx context.Context, u User) (int64, error)
    FindByPhone(ctx context.Context, phone string) (User, error)
    FindByEmail(ctx context.Context, email string) (User, error)
    FindById(ctx context.Context, id int64) (User, error)
    UpdateNonZeroFields(ctx context.Context, u User) error
}

type GormUserDao struct {
    db *gorm.DB
}

func NewUserDao(db *gorm.DB) UserDao {
    return &GormUserDao{db: db}
}

func (d *GormUserDao) Insert(ctx context.Context, u User) (int64, error) {
    now := time.Now().UnixMilli()
    u.Ctime = now
    u.Utime = now
    err := d.db.WithContext(ctx).Create(&u).Error
    var mysqlErr *mysql.MySQLError
    if errors.As(err, &mysqlErr) {
        const uniqueConflictsErrNo = 1062
        if mysqlErr.Number == uniqueConflictsErrNo {
            return 0, ErrDuplicate
        }
    }
    return u.Id, err
}

func (d *GormUserDao) FindByPhone(ctx context.Context, phone string) (User, error) {
    var u User
    err := d.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error
    return u, err
}

func (d *GormUserDao) FindByEmail(ctx context.Context, email string) (User, error) {
    var u User
    err := d.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
    return u, err
}

func (d *GormUserDao) FindById(ctx context.Context, id int64) (User, error) {
    var u User
    err := d.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
    return u, err
}

func (d *GormUserDao) UpdateNonZeroFields(ctx context.Context, u User) error {
    u.Utime = time.Now().UnixMilli()
    return d.db.WithContext(ctx).Updates(&u).Error
}

// 数据库表结构
type User struct {
    Id       int64  `gorm:"primaryKey;autoIncrement"`
    Phone    string `gorm:"type:varchar(20);uniqueIndex"`
    Email    string `gorm:"type:varchar(100);uniqueIndex"`
    Password string `gorm:"type:varchar(255)"`
    Nickname string `gorm:"type:varchar(50)"`
    Avatar   string `gorm:"type:varchar(500)"`
    Status   uint8  `gorm:"type:tinyint;default:1"`
    Ctime    int64
    Utime    int64
}

type UserRole struct {
    Id       int64 `gorm:"primaryKey;autoIncrement"`
    UserId   int64 `gorm:"index:idx_user_tenant"`
    TenantId int64 `gorm:"index:idx_user_tenant"`
    RoleId   int64
    Ctime    int64
    Utime    int64
}

type Role struct {
    Id          int64  `gorm:"primaryKey;autoIncrement"`
    TenantId    int64  `gorm:"uniqueIndex:uk_tenant_code"`
    Name        string `gorm:"type:varchar(50)"`
    Code        string `gorm:"type:varchar(50);uniqueIndex:uk_tenant_code"`
    Description string `gorm:"type:varchar(200)"`
    Ctime       int64
    Utime       int64
}

type RolePermission struct {
    Id           int64 `gorm:"primaryKey;autoIncrement"`
    RoleId       int64 `gorm:"index:idx_role"`
    PermissionId int64
    Ctime        int64
}

type Permission struct {
    Id       int64  `gorm:"primaryKey;autoIncrement"`
    Code     string `gorm:"type:varchar(100);uniqueIndex"`
    Name     string `gorm:"type:varchar(100)"`
    Type     uint8  `gorm:"type:tinyint"`
    Resource string `gorm:"type:varchar(200)"`
    Ctime    int64
    Utime    int64
}

type UserAddress struct {
    Id        int64  `gorm:"primaryKey;autoIncrement"`
    UserId    int64  `gorm:"index:idx_user"`
    Name      string `gorm:"type:varchar(50)"`
    Phone     string `gorm:"type:varchar(20)"`
    Province  string `gorm:"type:varchar(50)"`
    City      string `gorm:"type:varchar(50)"`
    District  string `gorm:"type:varchar(50)"`
    Detail    string `gorm:"type:varchar(200)"`
    IsDefault bool   `gorm:"type:tinyint;default:0"`
    Ctime     int64
    Utime     int64
}

type OAuthAccount struct {
    Id          int64  `gorm:"primaryKey;autoIncrement"`
    UserId      int64  `gorm:"index:idx_user"`
    Provider    string `gorm:"type:varchar(20);uniqueIndex:uk_provider_uid"`
    ProviderUid string `gorm:"type:varchar(100);uniqueIndex:uk_provider_uid"`
    AccessToken string `gorm:"type:varchar(500)"`
    Ctime       int64
    Utime       int64
}
```

```go
// user/repository/dao/init.go
package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
    return db.AutoMigrate(
        &User{},
        &UserRole{},
        &Role{},
        &RolePermission{},
        &Permission{},
        &UserAddress{},
        &OAuthAccount{},
    )
}
```

**Step 2: Commit**

```bash
git add user/repository/dao/
git commit -m "feat(user): add DAO layer with table definitions"
```

---

### Task 2.4: User Cache 层

**Files:**
- Create: `user/repository/cache/user.go`

**Step 1: 实现 Redis 缓存**

```go
// user/repository/cache/user.go
package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/your-username/mall/user/domain"
)

var ErrKeyNotExist = redis.Nil

//go:generate mockgen -source=./user.go -package=cachemocks -destination=mocks/user_cache_mock.go
type UserCache interface {
    Get(ctx context.Context, id int64) (domain.User, error)
    Set(ctx context.Context, u domain.User) error
    Delete(ctx context.Context, id int64) error
    GetPermissions(ctx context.Context, uid, tenantId int64) ([]string, error)
    SetPermissions(ctx context.Context, uid, tenantId int64, perms []string) error
}

type RedisUserCache struct {
    client     redis.Cmdable
    expiration time.Duration
}

func NewUserCache(client redis.Cmdable) UserCache {
    return &RedisUserCache{
        client:     client,
        expiration: 15 * time.Minute,
    }
}

func (c *RedisUserCache) Get(ctx context.Context, id int64) (domain.User, error) {
    key := c.key(id)
    val, err := c.client.Get(ctx, key).Bytes()
    if err != nil {
        return domain.User{}, err
    }
    var u domain.User
    err = json.Unmarshal(val, &u)
    return u, err
}

func (c *RedisUserCache) Set(ctx context.Context, u domain.User) error {
    val, err := json.Marshal(u)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, c.key(u.Id), val, c.expiration).Err()
}

func (c *RedisUserCache) Delete(ctx context.Context, id int64) error {
    return c.client.Del(ctx, c.key(id)).Err()
}

func (c *RedisUserCache) GetPermissions(ctx context.Context, uid, tenantId int64) ([]string, error) {
    key := fmt.Sprintf("user:permission:%d:%d", uid, tenantId)
    val, err := c.client.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }
    var perms []string
    err = json.Unmarshal(val, &perms)
    return perms, err
}

func (c *RedisUserCache) SetPermissions(ctx context.Context, uid, tenantId int64, perms []string) error {
    key := fmt.Sprintf("user:permission:%d:%d", uid, tenantId)
    val, err := json.Marshal(perms)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, key, val, 10*time.Minute).Err()
}

func (c *RedisUserCache) key(id int64) string {
    return fmt.Sprintf("user:info:%d", id)
}
```

**Step 2: Commit**

```bash
git add user/repository/cache/
git commit -m "feat(user): add Redis cache layer"
```

---

### Task 2.5: User Repository 层

**Files:**
- Create: `user/repository/user.go`

**Step 1: 实现 CachedRepository（Cache-Aside 模式）**

```go
// user/repository/user.go
package repository

import (
    "context"
    "time"

    "github.com/your-username/mall/user/domain"
    "github.com/your-username/mall/user/repository/cache"
    "github.com/your-username/mall/user/repository/dao"
)

//go:generate mockgen -source=./user.go -package=repomocks -destination=mocks/user_mock.go
type UserRepository interface {
    Create(ctx context.Context, u domain.User) (int64, error)
    FindByPhone(ctx context.Context, phone string) (domain.User, error)
    FindById(ctx context.Context, id int64) (domain.User, error)
    Update(ctx context.Context, u domain.User) error
    GetPermissions(ctx context.Context, uid, tenantId int64) ([]string, error)
}

type CachedUserRepository struct {
    dao   dao.UserDao
    cache cache.UserCache
}

func NewCachedUserRepository(d dao.UserDao, c cache.UserCache) UserRepository {
    return &CachedUserRepository{dao: d, cache: c}
}

func (repo *CachedUserRepository) Create(ctx context.Context, u domain.User) (int64, error) {
    return repo.dao.Insert(ctx, repo.domainToEntity(u))
}

func (repo *CachedUserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
    // 1. 先查缓存
    u, err := repo.cache.Get(ctx, id)
    if err == nil {
        return u, nil
    }
    // 2. 缓存未命中，查数据库
    ue, err := repo.dao.FindById(ctx, id)
    if err != nil {
        return domain.User{}, err
    }
    u = repo.entityToDomain(ue)
    // 3. 异步回写缓存
    go func() {
        _ = repo.cache.Set(ctx, u)
    }()
    return u, nil
}

func (repo *CachedUserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
    ue, err := repo.dao.FindByPhone(ctx, phone)
    if err != nil {
        return domain.User{}, err
    }
    return repo.entityToDomain(ue), nil
}

func (repo *CachedUserRepository) Update(ctx context.Context, u domain.User) error {
    err := repo.dao.UpdateNonZeroFields(ctx, repo.domainToEntity(u))
    if err != nil {
        return err
    }
    return repo.cache.Delete(ctx, u.Id)
}

func (repo *CachedUserRepository) GetPermissions(ctx context.Context, uid, tenantId int64) ([]string, error) {
    // 先查缓存
    perms, err := repo.cache.GetPermissions(ctx, uid, tenantId)
    if err == nil {
        return perms, nil
    }
    // TODO: 从 DAO 查询 user_roles → role_permissions → permissions
    // 这里先返回空，等 DAO 补充 RBAC 查询方法后完善
    return []string{}, nil
}

func (repo *CachedUserRepository) entityToDomain(u dao.User) domain.User {
    return domain.User{
        Id:       u.Id,
        Phone:    u.Phone,
        Email:    u.Email,
        Password: u.Password,
        Nickname: u.Nickname,
        Avatar:   u.Avatar,
        Status:   domain.UserStatus(u.Status),
        Ctime:    time.UnixMilli(u.Ctime),
        Utime:    time.UnixMilli(u.Utime),
    }
}

func (repo *CachedUserRepository) domainToEntity(u domain.User) dao.User {
    return dao.User{
        Id:       u.Id,
        Phone:    u.Phone,
        Email:    u.Email,
        Password: u.Password,
        Nickname: u.Nickname,
        Avatar:   u.Avatar,
        Status:   uint8(u.Status),
    }
}
```

**Step 2: Commit**

```bash
git add user/repository/user.go
git commit -m "feat(user): add cached repository layer"
```

---

### Task 2.6: User Service 层

**Files:**
- Create: `user/service/user.go`

**Step 1: 实现业务逻辑**

```go
// user/service/user.go
package service

import (
    "context"
    "errors"

    "golang.org/x/crypto/bcrypt"

    "github.com/your-username/mall/pkg/logger"
    "github.com/your-username/mall/user/domain"
    "github.com/your-username/mall/user/events"
    "github.com/your-username/mall/user/repository"
    "github.com/your-username/mall/user/repository/dao"
)

var (
    ErrDuplicateUser        = dao.ErrDuplicate
    ErrInvalidUserOrPassword = errors.New("用户名或密码错误")
)

//go:generate mockgen -source=./user.go -package=svcmocks -destination=mocks/user_mock.go
type UserService interface {
    Signup(ctx context.Context, u domain.User) (int64, error)
    Login(ctx context.Context, phone, password string) (domain.User, error)
    FindById(ctx context.Context, id int64) (domain.User, error)
    FindByPhone(ctx context.Context, phone string) (domain.User, error)
    UpdateProfile(ctx context.Context, u domain.User) error
    GetPermissions(ctx context.Context, uid, tenantId int64) ([]string, error)
}

type UserServiceImpl struct {
    repo     repository.UserRepository
    l        logger.LoggerV1
    producer events.Producer
}

func NewUserService(
    repo repository.UserRepository,
    l logger.LoggerV1,
    producer events.Producer,
) UserService {
    return &UserServiceImpl{repo: repo, l: l, producer: producer}
}

func (svc *UserServiceImpl) Signup(ctx context.Context, u domain.User) (int64, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
    if err != nil {
        return 0, err
    }
    u.Password = string(hash)
    id, err := svc.repo.Create(ctx, u)
    if err != nil {
        return 0, err
    }
    // 异步发送注册事件
    go func() {
        er := svc.producer.ProduceUserRegistered(ctx, events.UserRegisteredEvent{
            UserId: id,
            Phone:  u.Phone,
        })
        if er != nil {
            svc.l.Error("发送注册事件失败", logger.Error(er))
        }
    }()
    return id, nil
}

func (svc *UserServiceImpl) Login(ctx context.Context, phone, password string) (domain.User, error) {
    u, err := svc.repo.FindByPhone(ctx, phone)
    if err != nil {
        return domain.User{}, ErrInvalidUserOrPassword
    }
    err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
    if err != nil {
        return domain.User{}, ErrInvalidUserOrPassword
    }
    return u, nil
}

func (svc *UserServiceImpl) FindById(ctx context.Context, id int64) (domain.User, error) {
    return svc.repo.FindById(ctx, id)
}

func (svc *UserServiceImpl) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
    return svc.repo.FindByPhone(ctx, phone)
}

func (svc *UserServiceImpl) UpdateProfile(ctx context.Context, u domain.User) error {
    return svc.repo.Update(ctx, u)
}

func (svc *UserServiceImpl) GetPermissions(ctx context.Context, uid, tenantId int64) ([]string, error) {
    return svc.repo.GetPermissions(ctx, uid, tenantId)
}
```

**Step 2: Commit**

```bash
git add user/service/
git commit -m "feat(user): add service layer with signup/login/RBAC"
```

---

### Task 2.7: User Events (Kafka)

**Files:**
- Create: `user/events/types.go`
- Create: `user/events/producer.go`

**Step 1: 定义事件和生产者**

```go
// user/events/types.go
package events

type UserRegisteredEvent struct {
    UserId int64  `json:"user_id"`
    Phone  string `json:"phone"`
}
```

```go
// user/events/producer.go
package events

import (
    "context"
    "encoding/json"

    "github.com/IBM/sarama"
)

type Producer interface {
    ProduceUserRegistered(ctx context.Context, evt UserRegisteredEvent) error
}

type SaramaProducer struct {
    client sarama.SyncProducer
}

func NewSaramaProducer(client sarama.Client) (Producer, error) {
    p, err := sarama.NewSyncProducerFromClient(client)
    if err != nil {
        return nil, err
    }
    return &SaramaProducer{client: p}, nil
}

func (s *SaramaProducer) ProduceUserRegistered(ctx context.Context, evt UserRegisteredEvent) error {
    data, _ := json.Marshal(evt)
    _, _, err := s.client.SendMessage(&sarama.ProducerMessage{
        Topic: "user_registered",
        Value: sarama.ByteEncoder(data),
    })
    return err
}
```

**Step 2: Commit**

```bash
git add user/events/
git commit -m "feat(user): add Kafka event producer"
```

---

### Task 2.8: User gRPC Handler

**Files:**
- Create: `user/grpc/user.go`

**Step 1: 实现 gRPC 服务端**

```go
// user/grpc/user.go
package grpc

import (
    "context"

    "google.golang.org/grpc"
    "google.golang.org/protobuf/types/known/timestamppb"

    userv1 "github.com/your-username/mall/api/proto/gen/user/v1"
    "github.com/your-username/mall/user/domain"
    "github.com/your-username/mall/user/service"
)

type UserGRPCServer struct {
    userv1.UnimplementedUserServiceServer
    svc service.UserService
}

func NewUserGRPCServer(svc service.UserService) *UserGRPCServer {
    return &UserGRPCServer{svc: svc}
}

func (s *UserGRPCServer) Register(server *grpc.Server) {
    userv1.RegisterUserServiceServer(server, s)
}

func (s *UserGRPCServer) Signup(ctx context.Context, req *userv1.SignupRequest) (*userv1.SignupResponse, error) {
    id, err := s.svc.Signup(ctx, domain.User{
        Phone:    req.GetPhone(),
        Email:    req.GetEmail(),
        Password: req.GetPassword(),
    })
    if err != nil {
        return nil, err
    }
    return &userv1.SignupResponse{Id: id}, nil
}

func (s *UserGRPCServer) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
    u, err := s.svc.Login(ctx, req.GetPhone(), req.GetPassword())
    if err != nil {
        return nil, err
    }
    return &userv1.LoginResponse{User: s.toDTO(u)}, nil
}

func (s *UserGRPCServer) FindById(ctx context.Context, req *userv1.FindByIdRequest) (*userv1.FindByIdResponse, error) {
    u, err := s.svc.FindById(ctx, req.GetId())
    if err != nil {
        return nil, err
    }
    return &userv1.FindByIdResponse{User: s.toDTO(u)}, nil
}

func (s *UserGRPCServer) FindByPhone(ctx context.Context, req *userv1.FindByPhoneRequest) (*userv1.FindByPhoneResponse, error) {
    u, err := s.svc.FindByPhone(ctx, req.GetPhone())
    if err != nil {
        return nil, err
    }
    return &userv1.FindByPhoneResponse{User: s.toDTO(u)}, nil
}

func (s *UserGRPCServer) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
    err := s.svc.UpdateProfile(ctx, domain.User{
        Id:       req.GetId(),
        Nickname: req.GetNickname(),
        Avatar:   req.GetAvatar(),
    })
    return &userv1.UpdateProfileResponse{}, err
}

func (s *UserGRPCServer) GetPermissions(ctx context.Context, req *userv1.GetPermissionsRequest) (*userv1.GetPermissionsResponse, error) {
    perms, err := s.svc.GetPermissions(ctx, req.GetUserId(), req.GetTenantId())
    if err != nil {
        return nil, err
    }
    return &userv1.GetPermissionsResponse{Permissions: perms}, nil
}

func (s *UserGRPCServer) toDTO(u domain.User) *userv1.User {
    return &userv1.User{
        Id:       u.Id,
        Phone:    u.Phone,
        Email:    u.Email,
        Nickname: u.Nickname,
        Avatar:   u.Avatar,
        Status:   int32(u.Status),
        Ctime:    timestamppb.New(u.Ctime),
        Utime:    timestamppb.New(u.Utime),
    }
}
```

**Step 2: Commit**

```bash
git add user/grpc/
git commit -m "feat(user): add gRPC handler"
```

---

### Task 2.9: User IoC + Wire + main.go

**Files:**
- Create: `user/ioc/db.go`
- Create: `user/ioc/redis.go`
- Create: `user/ioc/kafka.go`
- Create: `user/ioc/logger.go`
- Create: `user/ioc/grpc.go`
- Create: `user/wire.go`
- Create: `user/config/dev.yaml`
- Create: `user/main.go`

**Step 1: IoC 初始化文件**

```go
// user/ioc/db.go
package ioc

import (
    "github.com/spf13/viper"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    gormPrometheus "gorm.io/plugin/prometheus"

    "github.com/your-username/mall/pkg/logger"
    "github.com/your-username/mall/user/repository/dao"
)

func InitDB(l logger.LoggerV1) *gorm.DB {
    type Config struct {
        DSN string `yaml:"dsn"`
    }
    var cfg Config
    err := viper.UnmarshalKey("db.mysql", &cfg)
    if err != nil {
        panic(err)
    }
    db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
    if err != nil {
        panic("failed to connect database")
    }
    db.Use(gormPrometheus.New(gormPrometheus.Config{
        DBName:          "user",
        RefreshInterval: 15,
        StartServer:     false,
    }))
    err = dao.InitTables(db)
    if err != nil {
        panic(err)
    }
    return db
}
```

```go
// user/ioc/redis.go
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

```go
// user/ioc/kafka.go
package ioc

import (
    "github.com/IBM/sarama"
    "github.com/spf13/viper"
    "github.com/your-username/mall/user/events"
)

func InitKafka() sarama.Client {
    type Config struct {
        Addrs []string `yaml:"addrs"`
    }
    saramaCfg := sarama.NewConfig()
    saramaCfg.Producer.Return.Successes = true
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
    p, err := events.NewSaramaProducer(client)
    if err != nil {
        panic(err)
    }
    return p
}
```

```go
// user/ioc/logger.go
package ioc

import (
    "go.uber.org/zap"
    "github.com/your-username/mall/pkg/logger"
)

func InitLogger() logger.LoggerV1 {
    l, err := zap.NewDevelopment()
    if err != nil {
        panic(err)
    }
    return logger.NewZapLogger(l)
}
```

```go
// user/ioc/grpc.go
package ioc

import (
    "github.com/spf13/viper"
    "google.golang.org/grpc"

    "github.com/your-username/mall/pkg/grpcx"
    "github.com/your-username/mall/pkg/logger"
    igrpc "github.com/your-username/mall/user/grpc"
)

func InitGRPCServer(userServer *igrpc.UserGRPCServer, l logger.LoggerV1) *grpcx.Server {
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
    userServer.Register(s)
    return &grpcx.Server{
        Server:    s,
        Port:      cfg.Port,
        EtcdAddrs: cfg.EtcdAddrs,
        Name:      "user",
        L:         l,
    }
}
```

**Step 2: Wire 依赖注入**

```go
// user/wire.go
//go:build wireinject

package main

import (
    "github.com/google/wire"

    "github.com/your-username/mall/pkg/grpcx"
    "github.com/your-username/mall/user/grpc"
    "github.com/your-username/mall/user/ioc"
    "github.com/your-username/mall/user/repository"
    "github.com/your-username/mall/user/repository/cache"
    "github.com/your-username/mall/user/repository/dao"
    "github.com/your-username/mall/user/service"
)

var thirdPartySet = wire.NewSet(
    ioc.InitDB,
    ioc.InitLogger,
    ioc.InitRedis,
    ioc.InitKafka,
    ioc.InitProducer,
)

var userSet = wire.NewSet(
    grpc.NewUserGRPCServer,
    service.NewUserService,
    repository.NewCachedUserRepository,
    dao.NewUserDao,
    cache.NewUserCache,
)

func InitUserGRPCServer() *grpcx.Server {
    wire.Build(
        thirdPartySet,
        userSet,
        ioc.InitGRPCServer,
    )
    return new(grpcx.Server)
}
```

**Step 3: 配置文件**

```yaml
# user/config/dev.yaml
db:
  mysql:
    dsn: "root:root@tcp(localhost:13306)/mall_user?charset=utf8mb4&parseTime=True&loc=Local"

redis:
  addr: "localhost:6379"

kafka:
  addrs:
    - "localhost:9094"

grpc:
  server:
    port: 8081
    etcdAddrs:
      - "localhost:12379"
```

**Step 4: main.go**

```go
// user/main.go
package main

import (
    "fmt"

    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

func main() {
    initViper()
    server := InitUserGRPCServer()
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

**Step 5: 生成 Wire**

```bash
cd user && wire && cd ..
```

**Step 6: 验证编译**

```bash
go build ./user/...
```

**Step 7: Commit**

```bash
git add user/
git commit -m "feat(user): complete user-svc with IoC, Wire, and main.go"
```

---

## Phase 3-12: 剩余微服务

> **每个服务都遵循与 user-svc 完全相同的实现模式。** 以下列出每个服务的关键差异点，你需要参考 Phase 2 的完整模式来实现。

### Task 3: tenant-svc（租户服务）

**实现顺序：** Proto → Domain → DAO → Cache → Repository → Service → gRPC → Events → IoC → Wire → main.go

**关键差异：**
- Proto: `api/proto/tenant/v1/tenant.proto` — 定义 Tenant, Shop, TenantPlan 消息和 CRUD + 审核接口
- DAO 表: tenants, tenant_plans, tenant_quota_usage, shops（参考设计文档 5.2）
- 配额控制逻辑在 Service 层：创建商品前 gRPC 调用 tenant-svc 检查配额
- Kafka 事件: `tenant_approved`, `tenant_plan_changed`
- gRPC 端口: 8082
- 数据库: `mall_tenant`

---

### Task 4: product-svc（商品服务）

**关键差异：**
- Proto: `api/proto/product/v1/product.proto` — Product, ProductSKU, Category, Brand, ProductSpec
- DAO 表: categories, brands, products, product_skus, product_specs（参考设计文档 5.3）
- 三级分类树查询：递归查询或一次全量查出后内存构建
- Kafka 事件: `product_status_changed`, `product_updated`
- gRPC 端口: 8083
- 数据库: `mall_product`

---

### Task 5: inventory-svc（库存服务）— 高并发亮点

**关键差异：**
- Proto: `api/proto/inventory/v1/inventory.proto` — SetStock, Deduct (预扣), Confirm (确认), Rollback (回滚)
- DAO 表: inventories, inventory_logs（参考设计文档 5.4）
- **核心亮点 — Redis+Lua 原子扣减：**
  - 创建 `inventory/repository/cache/lua/deduct.lua`
  - Cache 层实现 `Deduct(ctx, skuId, qty)` 调用 Lua 脚本
  - Service 层实现预扣/确认/回滚三阶段：
    1. Deduct: Redis Lua 扣减 → 记录 inventory_logs(type=预扣)
    2. Confirm: 消费 `order_paid` → MySQL 更新 available/sold → 记录 logs(type=确认)
    3. Rollback: 消费 `order_cancelled` → Redis INCRBY 回滚 → MySQL 回滚 → 记录 logs(type=回滚)
- Kafka: 生产 `inventory_deducted`, `inventory_alert`；消费 `order_paid`, `order_cancelled`
- gRPC 端口: 8084

---

### Task 6: order-svc（订单服务）— 状态机亮点

**关键差异：**
- Proto: `api/proto/order/v1/order.proto` — CreateOrder, GetOrder, ListOrders, CancelOrder, ConfirmReceive
- DAO 表: orders, order_items, order_status_logs, refund_orders（参考设计文档 5.5）
- **核心亮点 — 订单状态机：**
  - Domain 层定义状态枚举和允许的状态转换 map
  - Service.CreateOrder 中 gRPC 调用 product-svc/marketing-svc/logistics-svc/inventory-svc
  - 每次状态变更写 order_status_logs
- **核心亮点 — 超时关单：**
  - 创建订单后发 Kafka 延迟消息到 `order_close_delay`
  - Consumer 消费后检查状态，pending 则关闭
- **核心亮点 — 雪花 ID：**
  - 注入 `pkg/snowflake.Node`，生成 order_no
- Kafka: 生产 `order_created`, `order_close_delay`, `order_cancelled`, `order_completed`
- gRPC 端口: 8085

---

### Task 7: payment-svc（支付服务）— 幂等性亮点

**关键差异：**
- Proto: `api/proto/payment/v1/payment.proto` — CreatePayment, HandleNotify, Refund, GetPayment
- DAO 表: payment_orders, payment_notify_logs, refund_records（参考设计文档 5.6）
- **核心亮点 — 支付回调幂等：**
  - Redis `payment:idempotent:{payment_no}` 去重
  - DB 状态校验（只有 pending/paying 才处理回调）
- **核心亮点 — Strategy 模式：**
  - 定义 `PaymentChannel` 接口: `Pay()`, `HandleNotify()`, `Refund()`
  - 实现 `WechatPayChannel`, `AlipayChannel`, `MockChannel`
  - Service 层根据 channel 字段路由到对应实现
- Kafka: 生产 `order_paid`, `refund_completed`
- gRPC 端口: 8086

---

### Task 8: cart-svc（购物车服务）

**关键差异：**
- Proto: `api/proto/cart/v1/cart.proto` — AddItem, UpdateItem, RemoveItem, GetCart, ClearCart
- DAO 表: cart_items（参考设计文档 5.7）
- **Redis Hash 主存储：** `cart:{user_id}` → Hash(sku_id → JSON)
- 读写都走 Redis，定期异步 batch sync 到 MySQL
- gRPC 端口: 8087

---

### Task 9: search-svc（搜索服务）— ES + CDC 亮点

**关键差异：**
- Proto: `api/proto/search/v1/search.proto` — SearchProducts, GetSuggestions, GetHotWords
- DAO 表: search_hot_words, search_history（参考设计文档 5.8）
- **核心亮点 — ES 搜索：**
  - IoC 初始化 ES client (`olivere/elastic` 或 `elastic/go-elasticsearch`)
  - Repository 层直接操作 ES（不走 GORM）
  - IK 分词器配置
  - Completion Suggester 搜索建议
- **核心亮点 — Canal Binlog CDC：**
  - Kafka Consumer 消费 `product_binlog` topic
  - 解析 Canal JSON 格式，转换为 ES 文档
  - 增量更新 ES 索引
- gRPC 端口: 8088

---

### Task 10: marketing-svc（营销服务）— 秒杀亮点

**关键差异：**
- Proto: `api/proto/marketing/v1/marketing.proto` — CreateCoupon, ReceiveCoupon, Seckill, CalculateDiscount
- DAO 表: coupons, user_coupons, seckill_activities, seckill_items, promotion_rules（参考设计文档 5.9）
- **核心亮点 — 秒杀全链路：**
  - Redis Lua: 检查 `seckill:bought:{item_id}:{uid}` 限购 + 扣减 `seckill:stock:{item_id}`
  - 抢购成功 → Kafka `seckill_order_created`（异步下单，削峰）
  - order-svc Consumer 消费创建秒杀订单
- **优惠券领取：** Redis 原子计数 `coupon:received:{coupon_id}:{uid}` 防超领
- Kafka: 消费 `user_registered` (发新人券), `order_cancelled` (释放券)；生产 `seckill_order_created`
- gRPC 端口: 8089

---

### Task 11: logistics-svc（物流服务）

**关键差异：**
- Proto: `api/proto/logistics/v1/logistics.proto` — CalculateFreight, CreateShipment, GetShipment, AddTrack
- DAO 表: freight_templates, freight_rules, shipments, shipment_tracks（参考设计文档 5.10）
- 运费计算: 根据区域 + 首件续件规则计算
- Kafka: 生产 `order_shipped`
- gRPC 端口: 8090

---

### Task 12: notification-svc（通知服务）

**关键差异：**
- Proto: `api/proto/notification/v1/notification.proto` — SendNotification, GetNotifications, MarkRead
- DAO 表: notification_templates, notifications（参考设计文档 5.11）
- **纯消费者角色为主：** 消费 `user_registered`, `order_paid`, `order_shipped`, `inventory_alert`, `tenant_approved`
- 通知渠道 Strategy: SMS / Email / InApp
- gRPC 端口: 8091

---

## Phase 13: BFF 网关层

> 三个 BFF 结构相同，差异在于暴露的路由、鉴权策略和 gRPC 客户端依赖。
> C 端采用**独立商城 SaaS 模式**（类似 Shopify/有赞），每个商家有独立域名入口。
> 完整路由与 gRPC 映射详见设计文档 **Section 7: BFF 网关层详设**。

### Task 13.1: consumer-bff（C 端网关）

**gRPC 客户端依赖（11 个）：** tenant / user / product / cart / order / payment / inventory / search / marketing / logistics / notification

**Files:**
- Create: `consumer-bff/handler/user.go` — 注册/登录/个人信息/收货地址
- Create: `consumer-bff/handler/product.go` — 商品列表/详情/分类/品牌
- Create: `consumer-bff/handler/cart.go` — 购物车操作
- Create: `consumer-bff/handler/order.go` — 下单/订单列表/详情/取消/确认收货/退款/物流
- Create: `consumer-bff/handler/payment.go` — 发起支付/查询/回调
- Create: `consumer-bff/handler/search.go` — 搜索/建议/热搜/历史
- Create: `consumer-bff/handler/marketing.go` — 优惠券/秒杀/下单预览
- Create: `consumer-bff/handler/notification.go` — 站内信/未读数/已读
- Create: `consumer-bff/handler/shop.go` — 店铺信息
- Create: `consumer-bff/handler/middleware/login_jwt.go` — JWT 认证中间件
- Create: `consumer-bff/handler/middleware/tenant_resolve.go` — 域名→tenant_id 解析中间件
- Create: `consumer-bff/handler/jwt/redis_jwt.go` — JWT 双 Token 实现
- Create: `consumer-bff/ioc/web.go` — Gin 路由注册
- Create: `consumer-bff/ioc/{service}.go` — 11 个 gRPC 客户端初始化（etcd 发现）
- Create: `consumer-bff/wire.go`
- Create: `consumer-bff/main.go`
- Create: `consumer-bff/config/dev.yaml`

**关键实现：**

```go
// consumer-bff/handler/middleware/tenant_resolve.go 核心逻辑
// 独立商城 SaaS 模式：从域名解析 tenant_id
// 1. 提取 Host: shop1.mall.com → subdomain = "shop1"
// 2. 先查 Redis 缓存 tenant:domain:shop1 → tenant_id
// 3. 缓存未命中则调 tenant-svc 查询，结果缓存到 Redis（TTL 10min）
// 4. 注入 tenant_id 到 context，后续所有 handler 自动使用
// 5. 若解析失败 → 404 店铺不存在
```

```go
// consumer-bff/handler/jwt/redis_jwt.go 核心逻辑
// 双 Token 方案：
// - access_token: 短有效期（30min），存储 uid
// - refresh_token: 长有效期（7天），用于刷新 access_token
// - Redis 黑名单：登出时将 jti 加入黑名单
// 注意：tenant_id 不存在 JWT 中，而是从域名中间件注入
```

**路由规划（44 个接口）：** 详见设计文档 Section 7.1

**Gin 端口: 8080**

---

### Task 13.2: merchant-bff（商家端网关）

**gRPC 客户端依赖（9 个）：** user / tenant / product / inventory / order / payment / marketing / logistics / notification

**Files:**
- Create: `merchant-bff/handler/auth.go` — 登录/登出/刷新
- Create: `merchant-bff/handler/shop.go` — 店铺信息/配额
- Create: `merchant-bff/handler/product.go` — 商品 CRUD/上下架
- Create: `merchant-bff/handler/category.go` — 分类管理
- Create: `merchant-bff/handler/brand.go` — 品牌管理
- Create: `merchant-bff/handler/inventory.go` — 库存设置/查询/日志
- Create: `merchant-bff/handler/order.go` — 订单列表/详情/发货/退款/物流/支付查询
- Create: `merchant-bff/handler/marketing.go` — 优惠券/秒杀/满减
- Create: `merchant-bff/handler/logistics.go` — 运费模板 CRUD
- Create: `merchant-bff/handler/staff.go` — 员工角色管理（RBAC）
- Create: `merchant-bff/handler/notification.go` — 站内信
- Create: `merchant-bff/handler/middleware/login_jwt.go` — JWT 认证（含 tenant_id claim）
- Create: `merchant-bff/handler/middleware/rbac.go` — RBAC 权限校验
- Create: `merchant-bff/ioc/web.go` — Gin 路由注册
- Create: `merchant-bff/ioc/{service}.go` — 9 个 gRPC 客户端初始化
- Create: `merchant-bff/wire.go`
- Create: `merchant-bff/main.go`
- Create: `merchant-bff/config/dev.yaml`

**路由规划（44 个接口）：** 详见设计文档 Section 7.2

**JWT 鉴权：** 必须携带 tenant_id claim，中间件自动注入；RBAC 中间件校验角色权限
**Gin 端口: 8180**

---

### Task 13.3: admin-bff（平台管理端网关）

**gRPC 客户端依赖（6 个）：** user / tenant / product / order / payment / notification

**Files:**
- Create: `admin-bff/handler/auth.go` — 登录/登出/刷新
- Create: `admin-bff/handler/tenant.go` — 商家列表/详情/审核/冻结
- Create: `admin-bff/handler/plan.go` — 套餐 CRUD
- Create: `admin-bff/handler/category.go` — 平台分类管理
- Create: `admin-bff/handler/brand.go` — 品牌管理
- Create: `admin-bff/handler/user.go` — 用户管理/冻结/RBAC
- Create: `admin-bff/handler/order.go` — 订单监管
- Create: `admin-bff/handler/payment.go` — 支付/退款监管
- Create: `admin-bff/handler/notification.go` — 通知模板 CRUD/发送
- Create: `admin-bff/handler/middleware/login_jwt.go` — JWT 认证（限 tenant_id=0）
- Create: `admin-bff/handler/middleware/rbac.go` — 平台 RBAC 权限校验
- Create: `admin-bff/ioc/web.go` — Gin 路由注册
- Create: `admin-bff/ioc/{service}.go` — 6 个 gRPC 客户端初始化
- Create: `admin-bff/wire.go`
- Create: `admin-bff/main.go`
- Create: `admin-bff/config/dev.yaml`

**路由规划（30 个接口）：** 详见设计文档 Section 7.3

**JWT 鉴权：** 必须为平台角色（tenant_id=0），RBAC 校验权限
**Gin 端口: 8280**

---

## Phase 14: 集成测试 & 端到端验证

### Task 14.1: 启动全部服务并验证

**Step 1: 启动基础设施**

```bash
make infra-up
```

**Step 2: 按依赖顺序启动服务**

```bash
# 终端 1-11: 分别启动各微服务
make run SVC=user
make run SVC=tenant
make run SVC=product
make run SVC=inventory
make run SVC=order
make run SVC=payment
make run SVC=cart
make run SVC=search
make run SVC=marketing
make run SVC=logistics
make run SVC=notification
```

```bash
# 终端 12-14: 启动 BFF
make run SVC=consumer-bff
make run SVC=merchant-bff
make run SVC=admin-bff
```

**Step 3: 端到端验证核心流程**

```bash
# 1. 注册用户
curl -X POST http://localhost:8080/api/v1/users/signup \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800138000","email":"test@test.com","password":"123456"}'

# 2. 登录获取 Token
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800138000","password":"123456"}'
# 记录返回的 access_token

# 3. 搜索商品（需要先通过商家端创建商品）
curl -X GET "http://localhost:8080/api/v1/search?keyword=手机" \
  -H "Authorization: Bearer {token}"

# 4. 加购物车
curl -X POST http://localhost:8080/api/v1/cart/items \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"sku_id":1,"quantity":1}'

# 5. 创建订单
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"sku_ids":[1],"address_id":1}'

# 6. 发起支付
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"order_no":"xxx","channel":"mock"}'
```

---

### Task 14.2: 单元测试（每个服务）

每个服务的 Service 层需要写 Table-Driven 单元测试，参考 template.md 第 11 节。

```bash
# 生成所有 mock
find . -name "*.go" -exec grep -l "go:generate" {} \; | xargs -I {} go generate {}

# 运行所有测试
make test
```

---

## 建议实施顺序总结

| 阶段 | 内容 | 预期产出 |
|------|------|----------|
| Phase 0 | 脚手架 + docker-compose | 项目结构 + 本地基础设施 |
| Phase 1 | pkg/ 共享库 | logger, grpcx, saramax, ginx, snowflake, tenantx, ratelimit |
| Phase 2 | user-svc **（完整示范）** | 第一个可运行的微服务 |
| Phase 3 | tenant-svc | SaaS 基础 |
| Phase 4 | product-svc | 商品 SPU/SKU |
| Phase 5 | **inventory-svc** | **Redis+Lua 高并发亮点** |
| Phase 6 | cart-svc | Redis Hash 购物车 |
| Phase 7 | **order-svc** | **状态机 + 超时关单亮点** |
| Phase 8 | **payment-svc** | **幂等性 + Strategy 亮点** |
| Phase 9 | **search-svc** | **ES + Canal CDC 亮点** |
| Phase 10 | **marketing-svc** | **秒杀全链路亮点** |
| Phase 11 | logistics-svc | 运费计算 |
| Phase 12 | notification-svc | Kafka 消费者 |
| Phase 13 | 3 个 BFF | HTTP 网关 + JWT + 路由 |
| Phase 14 | 集成测试 | 端到端验证 |

> **加粗的是面试重点服务**，优先实现。其余服务可以先实现骨架，逐步完善。
