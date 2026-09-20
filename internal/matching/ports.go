package matching

import (
	"atlas-trading-infrastructure/internal/domain"
	"context"
)

type OrderRepository interface {
	GetActiveOrders(ctx context.Context) ([]*domain.Order, error)
}
