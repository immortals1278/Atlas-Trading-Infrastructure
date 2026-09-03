package domain

import (
	"atlas-trading-infrastructure/internal/matching/engine"
	"context"
)

type CacheRepository interface {
	GetOrderBookSnapshot(ctx context.Context, symbol string) (*engine.OrderBookSnapshot, error)
	SetOrderBookSnapshot(ctx context.Context, snapshot *engine.OrderBookSnapshot) error
}

type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload interface{}) error
	Close()
}
