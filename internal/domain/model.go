package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 1=BUY,2=SELL
type Orderside int16

// 1=NEW,2=PARTIALLY_FILLED,3=FILLED,4=CANCELLED
type OrderStatus int16

type Order struct {
	ID             uuid.UUID       `json:"id"`
	Symbol         string          `json:"symbol"`
	Price          decimal.Decimal `json:"price"`
	Quantity       decimal.Decimal `json:"quantity"`
	Side           Orderside       `json:"side"` // 不用string确保类型安全
	Status         OrderStatus     `json:"status"`
	FilledQuantity decimal.Decimal `json:"filledQuantity"`
}
