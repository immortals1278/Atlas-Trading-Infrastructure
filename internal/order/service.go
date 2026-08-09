package order

import (
	"atlas-trading-infrastructure/internal/domain"
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"atlas-trading-infrastructure/internal/infrastructure/outbox"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Service struct {
	orderRepo        OrderRepository
	accountRepo      AccountRepository
	txManager        DBTransaction
	eventBus         domain.EventPublisher
	rawPublisher     outbox.Publisher
	outboxRepo       *outbox.Repository
	publishedMsgChan chan uuid.UUID
}

func NewService(
	orderRepo OrderRepository,
	txManager DBTransaction,
	accountRepo AccountRepository,
	eventBus domain.EventPublisher,
	rawPublisher outbox.Publisher,
	outboxRepo *outbox.Repository,
) *Service {
	s := &Service{
		orderRepo:        orderRepo,
		txManager:        txManager,
		accountRepo:      accountRepo,
		eventBus:         eventBus,
		rawPublisher:     rawPublisher,
		outboxRepo:       outboxRepo,
		publishedMsgChan: make(chan uuid.UUID, 5000),
	}
	return s
}

func (s *Service) PlaceOrder(ctx context.Context, order *domain.Order) (err error) {
	order.Symbol = strings.ToUpper(order.Symbol)
	order.Price = order.Price.Round(8)
	order.Quantity = order.Quantity.Round(8)

	if order.Quantity.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("订单数量无效")
	}

	currencyToLock, amountToLock, err := s.calculateLockAmount(order)

	newID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("创建订单ID失败: %w", err)
	}

	order.ID = newID
	order.Status = domain.StatusNew
	order.FilledQuantity = decimal.Zero

	// 将place order和给outbox发消息的操作一起原子性执行
	var outboxMsg *outbox.Message
	err = s.txManager.ExecTx(ctx, func(ctx context.Context) error {
		if err := s.accountRepo.LockFunds(ctx, order.UserID, currencyToLock, amountToLock); err != nil {
			return fmt.Errorf("冻结失败：%w", err)
		}

		if err := s.orderRepo.CreateOrder(ctx, order); err != nil {
			return fmt.Errorf("创建订单失败: %w", err)
		}

		if !domain.IsSymbolAllowed(order.Symbol) {
			return fmt.Errorf("不允许的交易对：%w", err)
		}

		if s.outboxRepo != nil && s.eventBus != nil {
			event := &domain.OrderPlaceEvent{
				EventType:      domain.EventOrderPlaced,
				Symbol:         order.Symbol,
				OrderID:        order.ID,
				UserID:         order.UserID,
				Side:           order.Side,
				Price:          order.Price,
				Quantity:       order.Quantity,
				AmountLocked:   amountToLock,
				LockedCurrency: currencyToLock,
			}
			payload, marshalErr := outbox.MarshalPayload(event)
			if marshalErr != nil {
				return fmt.Errorf("序列化 OrderPlacedEvent 失败: %w", marshalErr)
			}
			outboxMsg := &outbox.Message{
				AggregateID:   order.ID.String(),
				AggregateType: "order_placed",
				Topic:         domain.TopicOrders,
				PartitionKey:  order.Symbol,
				Payload:       payload, // 包含整个事件
			}

			if insertErr := s.outboxRepo.InsertMsg(ctx, *outboxMsg); insertErr != nil {
				return fmt.Errorf("写入outbox失败: %w", insertErr)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 发消息给kafka，失败则使用上面发给outbox的
	if s.rawPublisher != nil && outboxMsg != nil {
		go func(msgId uuid.UUID, msgPayload []byte, topic, partitionKey string) {
			hotCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if pubErr := s.rawPublisher.PublishRaw(hotCtx, topic, partitionKey, msgPayload); pubErr != nil {
				logger.Log.Warn("kafka发送失败", zap.String("outbox_id", msgId.String()), zap.Error(pubErr))
				return
			}

			select {
			case s.publishedMsgChan <- msgId: //发给通道之后？
			default:
				logger.Log.Warn("通道满了", zap.String("outbox_id", msgId.String()))
			}
		}(outboxMsg.ID, outboxMsg.Payload, outboxMsg.Topic, outboxMsg.PartitionKey)
	}

	return nil
}

func splitSymbol(symbol string) (base string, quote string, err error) {
	parts := strings.Split(symbol, "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("交易对错误")
	}
	return parts[0], parts[1], err
}

func (S *Service) calculateLockAmount(order *domain.Order) (currency string, amount decimal.Decimal, err error) {
	base, quote, err := splitSymbol(order.Symbol)
	if err != nil {
		return "", decimal.Zero, err
	}
	if order.Side == domain.SideBuy {
		return quote, order.Price.Mul(order.Quantity), nil
	}
	return base, order.Quantity, nil

}
