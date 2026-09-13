package middleware

import (
	"atlas-trading-infrastructure/internal/infrastructure/redis"
	"context"
	"fmt"
	"time"

	redisclient "github.com/redis/go-redis/v9"
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

var rateLimitScript = redisclient.NewScript(`
	local tokens_key = KEYS[1]
	local timestamp_key = KEYS[2]

	local rate = tonumber(ARGV[1])
	local capacity = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	local requested = 1

	local fill_time = capacity / rate
	local ttl = math.floor(fill_time * 2)
	if ttl < 10 then
		ttl = 10
	end

	local last_tokens = tonumber(redis.call("get", tokens_key) or capacity)
	local last_refreshed = tonumber(redis.call("get", timestamp_key) or 0)

	local delta = math.max(0, now - last_refreshed)
	local filled_tokens = math.min(capacity, last_tokens + (delta * rate))

	local allowed = 0
	if filled_tokens >= requested then
		allowed = 1
		filled_tokens = filled_tokens - requested
	end

	redis.call("setex", tokens_key, ttl, filled_tokens)
	-- 更新时间为now
	redis.call("setex", timestamp_key, ttl, now)

	return allowed`)

func (r *RedisRateLimiter) Allow(ip string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	now := float64(time.Now().UnixMicro()) / 1e6
	tokensKey := fmt.Sprintf("exchange:ratelimit:tokens:%s", ip)
	timestampKey := fmt.Sprintf("exchange:ratelimit:ts:%s", ip)

	res, err := rateLimitScript.Run(ctx, r.client.Client, []string{tokensKey, timestampKey}, r.rate, r.capacity, now).Int()
	if err != nil {
		return true // 出错放行 避免redis故障导致全站不可用
	}
	return res == 1 // 本次请求拿到令牌
}
