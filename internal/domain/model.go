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
	StatusRejected        OrderStatus = 5 // 已拒绝
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
	UpdatedAt      int64           `json:"updated_at"`
}

type Account struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Currency  string          `json:"currency"`   // 币种 例如 "USD", "BTC"
	Balance   decimal.Decimal `json:"balance"`    // 余额
	Locked    decimal.Decimal `json:"locked"`     // 锁定余额
	CreatedAt int64           `json:"created_at"` // Unix 毫秒
	UpdatedAt int64           `json:"updated_at"` // Unix 毫秒
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

func StatusToString(s OrderStatus) string {
	switch s {
	case StatusNew:
		return "NEW"
	case StatusPartiallyFilled:
		return "PARTIALLY_FILLED"
	case StatusFilled:
		return "FILLED"
	case StatusCanceled:
		return "CANCELED"
	case StatusRejected:
		return "REJECTED"
	default:
		return "UNKNOWN"
	}
}

// SideFromString 字符串转 OrderSide (API 输入层使用)
func SideFromString(s string) (OrderSide, error) {
	switch s {
	case "BUY":
		return SideBuy, nil
	case "SELL":
		return SideSell, nil
	default:
		return 0, fmt.Errorf("无效订单方向: %s", s)
	}
}

func SideToString(s OrderSide) string {
	switch s {
	case SideBuy:
		return "BUY"
	case SideSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}
