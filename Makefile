.PHONY: grpc
grpc:
	@buf format -w api/proto
	@buf lint api/proto
	@buf generate api/proto

.PHONY: gen
gen:
	@go generate ./...

# ==================== Docker Dev ====================

.PHONY: dev-up dev-down dev-logs dev-up-local dev-build seed

# 启动 Consumer 开发环境 (远程基础设施)
dev-up:
	docker compose --profile consumer up -d --build

# 启动全部服务
dev-up-full:
	docker compose --profile full up -d --build

# 启动 Consumer + 本地基础设施
dev-up-local:
	docker compose -f docker-compose.yml -f docker-compose.local.yml --profile consumer --profile local up -d --build

# 停止所有服务
dev-down:
	docker compose --profile consumer --profile merchant --profile admin --profile full --profile local down

# 查看日志
dev-logs:
	docker compose --profile consumer logs -f

# 重新构建镜像
dev-build:
	docker compose --profile consumer build

# 种子数据 (仅登录账号)
seed:
	go run ./script/seed/

# 清空所有表 + 完整测试数据
seed-reset:
	go run ./script/seed/ -reset

# ==================== Local Dev (go run) ====================

SERVICES := user tenant product inventory order payment cart search \
            marketing logistics notification consumer-bff merchant-bff admin-bff

.PHONY: dev-run-all dev-stop-all dev-status dev-run-logs $(addprefix dev-run-,$(SERVICES)) $(addprefix dev-stop-,$(SERVICES))

# 启动全部后端服务 (go run, 连接远程基础设施)
dev-run-all:
	@script/dev.sh start

# 停止全部后端服务
dev-stop-all:
	@script/dev.sh stop

# 查看服务状态
dev-status:
	@script/dev.sh status

# 查看全部日志
dev-run-logs:
	@script/dev.sh logs

# 单独启动某个服务: make dev-run-order, make dev-run-consumer-bff, ...
$(addprefix dev-run-,$(SERVICES)):
	@script/dev.sh start $(subst dev-run-,,$@)

# 单独停止某个服务: make dev-stop-order, make dev-stop-consumer-bff, ...
$(addprefix dev-stop-,$(SERVICES)):
	@script/dev.sh stop $(subst dev-stop-,,$@)