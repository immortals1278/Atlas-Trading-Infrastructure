package order

import (
	"atlas-trading-infrastructure/internal/domain"
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DBTransaction interface {
	ExecTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) error
	GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	UpdateOrder(ctx context.Context, order *domain.Order) error
	GetOrderForUpdate(ctx context.Context, id uuid.UUID) (*domain.Order, error)
}

type AccountRepository interface {
	LockFunds(ctx context.Context, userID uuid.UUID, currency string, amount decimal.Decimal) error
}
