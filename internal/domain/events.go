package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EventType string

// 定义kafka常数
const (
	TopicOrders       = "exchange.orders"
	TopicSettlements  = "exchange.settlements"
	TopicTrades       = "exchange.trades"
	TopicOrderBook    = "exchange.orderbook"
	TopicOrderUpdates = "exchange.order_updates"
)

const (
	EventOrderPlaced          EventType = "order.placed"
	EventOrderCancelRequested EventType = "order.cancel_requested"
	EventOrderCanceled        EventType = "order.canceled"
	EventSettlementRequested  EventType = "settlement.requested"
	EventTradeExecuted        EventType = "trade.executed"
	EventOrderBookUpdated     EventType = "orderbook.updated"
	EventOrderUpdated         EventType = "order.updated"
)

type OrderPlaceEvent struct {
	EventType      EventType       `json:"event_type"`
	Symbol         string          `json:"symbol"`
	OrderID        uuid.UUID       `json:"order_id"`
	UserID         uuid.UUID       `json:"user_id"`
	Side           OrderSide       `json:"side"`
	Price          decimal.Decimal `json:"price"`
	Quantity       decimal.Decimal `json:"quantity"`
	AmountLocked   decimal.Decimal `json:"amount_locked"`
	LockedCurrency string          `json:"locked_currency"`
}

type OrderCancelRequestedEvent struct {
	EventType EventType `json:"event_type"`
	Symbol    string    `json:"symbol"`
	OrderID   uuid.UUID `json:"order_id"`
	Side      OrderSide `json:"side"`
	UserID    uuid.UUID `json:"user_id"`
}
