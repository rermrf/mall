// Package ratelimit 重新导出 github.com/rermrf/emo/ratelimit，
// 项目中统一使用本包引用。
package ratelimit

import "github.com/rermrf/emo/ratelimit"

// 重新导出类型
type (
	Limiter                  = ratelimit.Limiter
	RedisSlidingWindowLimiter = ratelimit.RedisSlidingWindowLimiter
)

// 重新导出构造函数
var NewRedisSlidingWindowLimiter = ratelimit.NewRedisSlidingWindowLimiter
