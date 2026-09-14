package order

import (
	"atlas-trading-infrastructure/internal/domain"
	"atlas-trading-infrastructure/internal/matching/engine"
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DBTransaction interface {
	ExecTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) error
	BatchCreateOrders(ctx context.Context, orders []*domain.Order) error
	GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	UpdateOrder(ctx context.Context, order *domain.Order) error
	GetOrderForUpdate(ctx context.Context, id uuid.UUID) (*domain.Order, error)
}

type TradeRepository interface {
	CreateTrade(ctx context.Context, trade *engine.Trade) error
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
} // TODOrepo没实现

type AccountRepository interface {
	UpdateBalance(ctx context.Context, userID uuid.UUID, currency string, amount decimal.Decimal) error
	LockFunds(ctx context.Context, userID uuid.UUID, currency string, amount decimal.Decimal) error
	BatchLockFunds(ctx context.Context, lockFunds map[uuid.UUID]map[string]decimal.Decimal) error
	UnlockFunds(ctx context.Context, userID uuid.UUID, currency string, amount decimal.Decimal) error
}

type OrderService interface {
	PlaceOrder(ctx context.Context, order *domain.Order) error
	BatchPlaceOrders(ctx context.Context, orders []*domain.Order) error
	CancelOrder(ctx context.Context, orderID, userID uuid.UUID) error
}
