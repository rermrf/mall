# syntax=docker/dockerfile:1

# ============================================================
# Universal multi-stage Dockerfile for all mall Go services
# Usage: docker build --build-arg SERVICE=user -t mall-user .
# ============================================================

FROM golang:1.25-alpine AS builder

ARG SERVICE

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the service binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./${SERVICE}/

# ============================================================
# Runtime image
# ============================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai

COPY --from=builder /server /server
COPY --from=builder /app/${SERVICE}/config /config
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/server", "--config", "/config/dev.yaml"]
