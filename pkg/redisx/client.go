package redisx

import (
	"github.com/redis/go-redis/v9"
	"github.com/rermrf/mall/pkg/redis/metrics"
	"github.com/rermrf/mall/pkg/redis/tracing"
)

// NewClient 创建带有 metrics 和 tracing hook 的 Redis 客户端
func NewClient(opts *redis.Options) *redis.Client {
	client := redis.NewClient(opts)
	client.AddHook(metrics.NewMetricsHook())
	client.AddHook(tracing.NewTracingHook())
	return client
}
