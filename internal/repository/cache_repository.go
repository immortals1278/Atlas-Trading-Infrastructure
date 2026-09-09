package repository

import (
	"atlas-trading-infrastructure/internal/domain"
	"atlas-trading-infrastructure/internal/infrastructrue/redis"
	"atlas-trading-infrastructure/internal/matching/engine"
	"context"
	"encoding/json"
	"fmt"
)

type RedisCacheRepository struct {
	client *redis.Client
}

func NewRedisCacheRepository(client *redis.Client) domain.CacheRepository {
	return &RedisCacheRepository{
		client: client,
	}
}

func (r *RedisCacheRepository) SetOrderBookSnapshot(ctx context.Context, snapshot *engine.OrderBookSnapshot) error {
	key := fmt.Sprintf("exchange_orderbook: %s", snapshot.Symbol)

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	// redis lua脚本
	// 解析现有快照中的fencing token，若较小则拒绝传入
	const luaScript := ``

	res, err := r.client.Eval(ctx, luaScript, []string{key}, snapshot.FencingToken, data).Result()
	if err != nil {
		return fmt.Errorf("写入redis快照脚本失败: %w", err)
	}

	// 检查fencing token是否过期
	if res(int64) == 0{
		return fmt.Errorf("fencingtoken已过期: %d", snapshot.FencingToken)
	}

	return nil
}

func (r *RedisCacheRepository) GetOrderBookSnapshot(ctx context.Context, Symbol string) (*engine.OrderBookSnapshot, error) {
}
