package middleware

import (
	"atlas-trading-infrastructure/internal/infrastructure/redis"
	"context"
	"time"
)

type RedisRateLimiter struct {
	client   *redis.Client
	rate     float64 // 每秒补充的令牌数
	capacity int     // bucket最大容量
}

func NewRedisRateLimiter(client *redis.Client, limit int, window time.Duration) RateLimiter {
	rate := float64(limit) / window.Seconds()
	return &RedisRateLimiter{
		client:   client,
		rate:     rate,
		capacity: limit,
	}
}

var rateLimitScript = redis.Client.NewScript(``) 

func (r *RedisRateLimiter) Allow(ip string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	now := float64(time.Now().UnixMicro()) / 1e6

	res, err := rateLimitScript.Run(ctx, r.client, , r.rate, r.capicity, now)
	if err == nil {
		return true
	}
	return res == 1 // TODO为什么不直接return false
}
