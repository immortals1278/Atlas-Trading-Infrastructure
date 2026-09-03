package matching

import (
	"atlas-trading-infrastructure/internal/domain"
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"atlas-trading-infrastructure/internal/matching/engine"
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type Subscriber struct {
	manager      *engine.Manager
	eventBus     domain.EventPublisher
	cacheRepo    domain.CacheRepository
	fencingToken atomic.Int64
}

func NewSubscriber(Manager *engine.Manager, eventBus domain.EventPublisher, cacheRepo domain.CacheRepository) *Subscriber {
	return &Subscriber{
		manager:   Manager,
		eventBus:  eventBus,
		cacheRepo: cacheRepo,
	}
}

// 由leader elector在成为leader时调用
// 失去leader身份传入0
func (s *Subscriber) SetFencingToken(token int64) {
	s.fencingToken.Store(token)
}

func (s *Subscriber) HandleEvent(ctx context.Context, key, value []byte) error {
	if s.fencingToken.Load() == 0 {
		logger.Warn("fencingToken不合法", zap.String("reason", "fencing_token is 0"))
		return nil
	}

	// 解码eventTyp决定路由
	var envelope struct {
		eventType domain.EventType
	}
	if err := json.Unmarshal(value, &envelope); err != nil {
		return fmt.Errorf("matching事件解析失败: %w", err)
	}

	switch envelope.eventType {
	case domain.EventOrderPlaced:
		var event domain.OrderPlacedEvent
		if err := json.Unmarshal(value, event); err != nil {
			return fmt.Errorf("placeOrder事件解析失败: %w", err)
		}
		return s.HandleOrderPlaced(ctx, &event)
	case domain.EventOrderCancelRequested:
		var event domain.OrderCancelRequestedEvent
		if err := json.Unmarshal(value, event); err != nil {
			return fmt.Errorf("CancelOrderRequest事件解析失败: %w", err)
		}
		return s.HandleOrderCancelRequested(ctx, &event)
	default:
		logger.Warn("handleEvent收到未知事件类型")
		return nil
	}
}

func (s *Subscriber) convertToMactchOrder(event *domain.OrderPlacedEvent) *engine.Order {
	var side engine.OrderSide
	if event.Side == domain.SideBuy {
		side = engine.SideBuy
	} else {
		side = engine.SideSell
	}
	return engine.NewOrder(event.OrderID, event.UserID, side, event.Price, event.Quantity)

}

func (s *Subscriber) HandleOrderPlaced(ctx context.Context, event *domain.OrderPlacedEvent) error {
	matchOrder := s.convertToMactchOrder(event)
	engine := s.manager.GetEngine(event.Symbol)
	trades := engine.Process(matchOrder)

	needsSettlement := len(trades) != 0 || matchOrder.Quantity.IsZero()

	if needsSettlement && s.eventBus != nil {
		settlementEvent := &domain.SettlementRequestedEvent{
			EventType:      domain.EventSettlementRequested,
			Symbol:         event.Symbol,
			TakerOrderID:   event.OrderID,
			AmountLocked:   event.AmountLocked,
			LockedCurrency: event.LockedCurrency,
			RemainingQty:   matchOrder.Quantity,
			Trades:         trades,
			FencingToken:   s.fencingToken.Load(),
		}

		// 原地无限重试发布
		for {
			err := s.eventBus.Publish(ctx, domain.TopicSettlements, event.Symbol, settlementEvent)
			if err == nil {
				break
			}

			logger.Warn("发送settlementEvent失败，1秒后尝试",
				zap.String("symbol", event.Symbol),
				zap.Error(err),
			)
			time.Sleep(1 * time.Second)
		}
	}

	// 更新redia，redis相关
	return nil
}

func (s *Subscriber) HandleOrderCancelRequested(ctx context.Context, event *domain.OrderCancelRequestedEvent) error {

}
