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

# 单独运行 seed
seed:
	go run ./script/seed/