package engine

import (
	"atlas-trading-infrastructure/internal/domain"
	"sort"
)

type Orderbook struct {
	Symbol string
	bids   []*Order
	asks   []*Order
}

func NewOrderbook(Symbol string) *Orderbook {
	return &Orderbook{
		Symbol: Symbol,
		bids:   make([]*Order, 0),
		asks:   make([]*Order, 0),
	}
}

func (ob *Orderbook) AddOrder(order *Order) {
	if order.Side == domain.SideBuy {
		ob.bids = append(ob.bids, order)
		// 降序
		sort.SliceStable(ob.bids, func(i, j int) bool {
			return ob.bids[i].Price.GreaterThan(ob.bids[j].Price)
		})
	} else {
		ob.asks = append(ob.asks, order)
		// 升序
		sort.SliceStable(ob.asks, func(i, j int) bool {
			return ob.asks[i].Price.LessThan(ob.asks[j].Price)
		})

	}
}

func (ob *Orderbook) BestAsk() (order *Order) {
	if len(ob.asks) == 0 {
		return nil
	}
	return ob.asks[0]
}

func (ob *Orderbook) RemoveAskOrder() {
	if len(ob.asks) > 0 {
		ob.asks[0] = nil // 避免内存泄露
		ob.asks = ob.asks[1:]
	}
}
