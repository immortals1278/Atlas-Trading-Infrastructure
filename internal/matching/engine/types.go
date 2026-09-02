package engine

import (
	"atlas-trading-infrastructure/internal/domain"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Order struct {
	Id       uuid.UUID
	UserID   uuid.UUID
	Side     domain.OrderSide
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type Trade struct {
	ID           uuid.UUID       `json:"id"`
	Symbol       string          `json:"symbol"`
	MakerOrderID uuid.UUID       `json:"maker_order_id"`
	TakerOrderID uuid.UUID       `json:"taker_order_id"`
	Price        decimal.Decimal `json:"price"`
	Quantity     decimal.Decimal `json:"quantity"`
	CreatedAt    int64           `json:"created_at"` //uinx毫秒

}

func NewOrder(Id uuid.UUID, UserID uuid.UUID, Side domain.OrderSide, Price decimal.Decimal, Quantity decimal.Decimal) *Order {
	return &Order{
		Id:       Id,
		UserID:   UserID,
		Side:     Side,
		Price:    Price,
		Quantity: Quantity,
	}
}
