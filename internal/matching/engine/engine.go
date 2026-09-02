package engine

import (
	"atlas-trading-infrastructure/internal/domain"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Engine struct {
	orderbook *Orderbook
	mu        sync.Mutex
}

func NewEngine(Symbol string) *Engine {
	return &Engine{
		orderbook: NewOrderbook(Symbol),
	}
}

// 处理新订单，返回成交结果
func (e *Engine) Process(order *Order) []*Trade {
	e.mu.Lock()
	defer e.mu.Unlock()

	var trades []*Trade

	if order.Side == domain.SideBuy {
		trades = e.matchBuyOrder(order)
	}
	if order.Side == domain.SideSell {
		trades = e.matchSellOrder(order)
	}

	if order.Quantity.IsPositive() {
		e.orderbook.AddOrder(order)
	}

	return trades
}

// 买单撮合逻辑
func (e *Engine) matchBuyOrder(buyOrder *Order) []*Trade {
	var trades []*Trade
	for {
		bestAsk := e.orderbook.BestAsk()
		if bestAsk == nil {
			break
		}

		// 避免左手换右手
		if buyOrder.Side == bestAsk.Side {
			buyOrder.Quantity = decimal.Zero
			break
		}

		// 检查价格是否匹配
		if buyOrder.Price.LessThan(bestAsk.Price) {
			break
		}

		// 计算成交数量
		matchQty := buyOrder.Quantity
		if bestAsk.Quantity.LessThan(matchQty) {
			matchQty = bestAsk.Quantity
		}

		// 记录成交
		tradeID, _ := uuid.NewV7()
		trade := &Trade{
			ID:           tradeID,
			Symbol:       e.orderbook.Symbol,
			MakerOrderID: bestAsk.Id,
			TakerOrderID: buyOrder.Id,
			Price:        bestAsk.Price, // price用挂单的价格
			Quantity:     matchQty,
			CreatedAt:    time.Now().UnixMilli(),
		}
		trades = append(trades, trade)

		bestAsk.Quantity.Sub(matchQty)
		buyOrder.Quantity.Sub(matchQty)

		// 如果卖单完全成交，从orderbook移除
		if bestAsk.Quantity.IsZero() {
			e.orderbook.RemoveAskOrder()
		}

		// 如果买单完全成交，结束
		if buyOrder.Quantity.IsZero() {
			break
		}
	}

	return trades
}

// 卖单撮合逻辑
func (e *Engine) matchSellOrder(order *Order) []*Trade {}
