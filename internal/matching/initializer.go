package matching

import (
	"atlas-trading-infrastructure/internal/domain"
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"atlas-trading-infrastructure/internal/matching/engine"
	"context"
	"fmt"
)

func RestoreEngineSnapshot(ctx context.Context, orderRepo OrderRepository, engineManager *engine.Manager) error {
	orders, err := orderRepo.GetActiveOrders(ctx)
	if err != nil {
		return fmt.Errorf("载入活跃订单失败: %w", err)
	}

	for _, order := range orders {

		eng := engineManager.GetEngine(order.Symbol)
		if eng != nil {
			var matchSide engine.OrderSide
			if order.Side == domain.SideBuy {
				matchSide = engine.SideBuy
			} else {
				matchSide = engine.SideSell
			}

			matchingOrder := &engine.Order{
				Id:       order.ID,
				UserID:   order.UserID,
				Side:     matchSide,
				Price:    order.Price,
				Quantity: order.Quantity.Sub(order.FilledQuantity),
			}
			eng.RestoreOrder(matchingOrder)
		}
	}
	logger.Info(fmt.Sprintf("成功从db恢复 %d 笔订单到撮合引擎", len(orders)))
	return nil
}
