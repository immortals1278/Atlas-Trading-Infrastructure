package order

import (
	"atlas-trading-infrastructure/internal/domain"
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type EventSubscriber struct {
	orderRepo   OrderRepository
	accountRepo AccountRepository
	tradeRepo   TradeRepository
	txManager   DBTransaction // 差防脑裂机制没实现
	eventBus    domain.EventPublisher
}

func NewEventSubscriber(
	orderRepo OrderRepository,
	accountRepo AccountRepository,
	tradeRepo TradeRepository,
	txManager DBTransaction,
	eventBus domain.EventPublisher,
) *EventSubscriber {
	return &EventSubscriber{
		orderRepo:   orderRepo,
		accountRepo: accountRepo,
		tradeRepo:   tradeRepo,
		txManager:   txManager,
		eventBus:    eventBus,
	}
}

func (s *EventSubscriber) HandleEvents(ctx context.Context, key, value []byte) (err error) {
	var envelope struct {
		EventType domain.EventType `json:"event_type"`
	}

	if err := json.Unmarshal(value, &envelope); err != nil {
		return fmt.Errorf("解析事件失败:%w", err)
	}

	switch envelope.EventType {
	case domain.EventSettlementRequested: //撮合成功后在db内结算
	case domain.EventOrderCanceled:
		var event domain.OrderCanceledEvent
		if err := json.Unmarshal(value, &event); err != nil {
			return fmt.Errorf("解析orderCanceledEvent失败: %w", err)
		}
		return s.handleOrderCanceled(ctx, &event)
	default:
		logger.Warn("收到未知EventType", zap.String("event_type", string(envelope.EventType)))
		return nil
	}
}

func (s *EventSubscriber) handleOrderCanceled(ctx context.Context, event *domain.OrderCanceledEvent) error {

	var copyOrder *domain.Order
	var orderSymbol string // 用来发updated event

	err := s.txManager.ExecTx(ctx, func(ctx context.Context) error {
		if event.FencingToken > 0 {

		} // 撮合引擎相关

		order, err := s.orderRepo.GetOrderForUpdate(ctx, event.OrderID)
		if err != nil {
			return fmt.Errorf("锁定订单失败: %w", err)
		}

		if order.Status == domain.StatusFilled || order.Status == domain.StatusCanceled {
			logger.Info("订单已取消或完成", zap.String("order_id", order.ID.String()))
			return nil
		}

		remain := order.Quantity.Sub(order.FilledQuantity)

		currency, amount, err := calculateAmount(order, remain)
		if err != nil {
			return fmt.Errorf("计算解锁资金失败: %w", err)
		}

		if amount.GreaterThan(decimal.Zero) {
			if err := s.accountRepo.UnlockFunds(ctx, order.UserID, currency, amount); err != nil { //没定义
				return fmt.Errorf("解锁资金失败: %w", err)
			}
		}

		order.Status = domain.StatusCanceled

		if err := s.orderRepo.UpdateOrder(ctx, order); err != nil {
			return fmt.Errorf("更新订单状态失败: %w", err)
		}

		orderSymbol = order.Symbol
		copyOrder = *order
		return nil
	})
	// 过时错误不处理逻辑没写
	// 取消成功发kafka
	if err == nil && copyOrder != nil && s.eventBus != nil {
		updateEvent := &domain.OrderUpdatedEvent{
			EventType: domain.EventOrderUpdated,
			Symbol:    orderSymbol,
			Order:     copyOrder,
		}

		publishCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if publishErr := s.eventBus.Publish(publishCtx, domain.TopicOrderUpdates, orderSymbol, updateEvent); publishErr != nil {
			logger.Error("kafka发布OrderUpdatedEvent(取消)失败", zap.Error(publishErr))
		}
		cancel()
	}

	return err
}

func calculateAmount(order *domain.Order, remain decimal.Decimal) (currency string, amount decimal.Decimal, err error) {
	base, quote, err := splitSymbol(order.Symbol)
	if err != nil {
		return "", decimal.Zero, err
	}
	if order.Side == domain.SideBuy {
		return quote, order.Price.Mul(remain), nil
	}
	return base, remain, nil
}
