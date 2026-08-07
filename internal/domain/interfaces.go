package domain

import (
	"context"
)

type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload interface{}) error
	Close()
}
