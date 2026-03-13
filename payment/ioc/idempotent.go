package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/rermrf/emo/idempotent"
)

func InitIdempotencyService(client redis.Cmdable) idempotent.IdempotencyService {
	return idempotent.NewBloomIdempotencyService(client, "payment:bloom", 1000000, 0.001)
}
