package domain

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 1=BUY,2=SELL
type OrderSide int16

// 1=NEW,2=PARTIALLY_FILLED,3=FILLED,4=CANCELLED
type OrderStatus int16

const (
	SideBuy  OrderSide = 1
	SideSell OrderSide = 2

	StatusNew             OrderStatus = 1 // 新订单
	StatusPartiallyFilled OrderStatus = 2 // 部分成交
	StatusFilled          OrderStatus = 3 // 完全成交
	StatusCanceled        OrderStatus = 4 // 已取消
)

type Order struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"userid"`
	Symbol         string          `json:"symbol"`
	Price          decimal.Decimal `json:"price"`
	Quantity       decimal.Decimal `json:"quantity"`
	Side           OrderSide       `json:"side"` // 不用string确保类型安全
	Status         OrderStatus     `json:"status"`
	FilledQuantity decimal.Decimal `json:"filledQuantity"`
}

var (
	ErrInsufficientFunds = fmt.Errorf("insufficient funds")
	ErrIdempotencySkip   = fmt.Errorf("idempotency skip: event already processed")
)

var allowedSymbol = map[string]bool{
	"BTC-USD": true,
	"ETH-USD": true,
}

func IsSymbolAllowed(symbol string) bool {
	return allowedSymbol[symbol]
}
