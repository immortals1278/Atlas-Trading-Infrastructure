package engine

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderSide int16

const (
	SideBuy  OrderSide = 1
	SideSell OrderSide = 2
)

type Order struct {
	Id       uuid.UUID
	UserID   uuid.UUID
	Side     OrderSide
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

type OrderBookLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

type OrderBookSnapshot struct {
	Symbol       string           `json:"symbol"`
	Bids         []OrderBookLevel `json:"bids"`
	Asks         []OrderBookLevel `json:"asks"`
	FencingToken int64            `json:"fencing_token"` // 防脑裂令牌
}

func NewOrder(Id uuid.UUID, UserID uuid.UUID, Side OrderSide, Price decimal.Decimal, Quantity decimal.Decimal) *Order {
	return &Order{
		Id:       Id,
		UserID:   UserID,
		Side:     Side,
		Price:    Price,
		Quantity: Quantity,
	}
}
