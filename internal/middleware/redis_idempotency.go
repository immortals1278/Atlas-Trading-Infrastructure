package middleware

import (
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"atlas-trading-infrastructure/internal/infrastructure/redis"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type RedisIdempotencyStore struct {
	client *redis.Client
}

func NewRedisIdempotencyStore(client *redis.Client) IdempotencyStore { // TODO创建完不用定期清理？
	return &RedisIdempotencyStore{
		client: client,
	}
}

func (s *RedisIdempotencyStore) Get(key string) *idempotencyEntry {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	redisKey := fmt.Sprintf("exchange:idempotency:%s", key)

	val, err := s.client.Client.Get(ctx, redisKey).Bytes()
	if err != nil {
		return nil
	}

	var entry idempotencyEntry
	if err := json.Unmarshal(val, &entry); err != nil {
		logger.Error("反序列化redis幂等性内存失败")
		return nil
	}

	return &entry
}

func (s *RedisIdempotencyStore) Set(key string, statusCode int, body []byte, ttl time.Duration) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	redisKey := fmt.Sprintf("exchange:idempotency:%s", key)

	entry := idempotencyEntry{
		StatusCode: statusCode,
		Body:       body,
		ExpiresAt:  time.Now().Add(ttl),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		logger.Error("序列化幂等性记录失败", zap.Error(err))
		return
	}

	// 写入redis
	if err := s.client.Client.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		logger.Error("寫入 Redis 冪等性快取失敗")
	}
}
