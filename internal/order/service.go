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

// 在哪启动？
func (s *Service) batchMarkPublishedWorker() {
	ticker := time.NewTicker(50 * time.Millisecond) // 每50ms发一个信号
	defer ticker.Stop()

	var batch []uuid.UUID
	for {
		select {
		case id := <-s.publishedMsgChan:
			batch = append(batch, id)
			if len(batch) >= 200 { //消息数量足够多也触发
				s.flushBatch(&batch)
			}
		case <-ticker.C:
			s.flushBatch(&batch)
		}
	}
}

func (s *Service) flushBatch(batch *[]uuid.UUID) {
	if len(*batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) //两秒超时
	defer cancel()

	if err := s.outboxRepo.MarkPublishedWorker(ctx, *batch); err != nil {
		logger.Log.Warn("批量标记outbox失败",
			zap.Error(err),
		)
	}

	*batch = (*batch)[:0] //清空切片但保留容量，避免内存再次分配
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
				logger.Log.Warn("PlaceOrder kafka发送失败", zap.String("outbox_id", msgId.String()), zap.Error(pubErr))
				return
			}

			select {
			case s.publishedMsgChan <- msgId:
			default:
				logger.Log.Warn("通道满了", zap.String("outbox_id", msgId.String()))
			}
		}(outboxMsg.ID, outboxMsg.Payload, outboxMsg.Topic, outboxMsg.PartitionKey)
	}

	return nil
}

func (s *Service) BacthPlaceOrders(ctx context.Context, orders []*domain.Order) error {
	if len(orders) == 0 {
		return nil
	}

	lockFunds := make(map[uuid.UUID]map[string]decimal.Decimal) // userID -> currency -> amount
	var outboxMsgs []*outbox.Message

	for _, order := range orders {
		order.Symbol = strings.ToUpper(order.Symbol)
		order.Price = order.Price.Round(8)
		order.Quantity = order.Quantity.Round(8)

		if order.Quantity.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("订单数量无效")
		}

		currencyToLock, amountToLock, err := s.calculateLockAmount(order)
		if lockFunds[order.UserID] == nil {
			lockFunds[order.UserID] = make(map[string]decimal.Decimal)
		}
		lockFunds[order.UserID][currencyToLock] = amountToLock

		newID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("创建订单ID失败: %w", err)
		}

		order.ID = newID
		order.Status = domain.StatusNew
		order.FilledQuantity = decimal.Zero

		// 将place order和给outbox发消息的操作一起原子性执行
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

				msg := &outbox.Message{
					AggregateID:   order.ID.String(),
					AggregateType: "order_placed",
					Topic:         domain.TopicOrders,
					PartitionKey:  order.Symbol,
					Payload:       payload,
				}
				outboxMsgs = append(outboxMsgs, msg)

				if insertErr := s.outboxRepo.InsertMsg(ctx, *msg); insertErr != nil {
					return fmt.Errorf("写入outbox失败: %w", insertErr)
				}
			}
			return nil
		})
	}

	// 写这几个数据库逻辑
	//单事务批量提交换取高吞吐和强一致
	err := s.txManager.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.accountRepo.BatchLockFunds(txCtx, lockFunds); err != nil {
			return fmt.Errorf("批次鎖定期資金失敗: %w", err)
		}
		if err := s.orderRepo.BatchCreateOrders(txCtx, orders); err != nil {
			return fmt.Errorf("批次建立訂單失敗: %w", err)
		}
		if len(outboxMsgs) > 0 {
			if err := s.outboxRepo.BatchInsert(txCtx, outboxMsgs); err != nil {
				return fmt.Errorf("批次寫入 Outbox 失敗: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	if s.rawPublisher != nil {
		for _, outboxMsg := range outboxMsgs {
			go func(msgId uuid.UUID, msgPayload []byte, topic, partitionKey string) {
				hotCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				if pubErr := s.rawPublisher.PublishRaw(hotCtx, topic, partitionKey, msgPayload); pubErr != nil {
					logger.Log.Warn("PlaceOrder kafka发送失败", zap.String("outbox_id", msgId.String()), zap.Error(pubErr))
					return
				}

				select {
				case s.publishedMsgChan <- msgId:
				default:
					logger.Log.Warn("通道满了", zap.String("outbox_id", msgId.String()))
				}
			}(outboxMsg.ID, outboxMsg.Payload, outboxMsg.Topic, outboxMsg.PartitionKey)
		}
	}

	return nil
}

func (s *Service) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) (err error) {
	orderPreCheck, err := s.orderRepo.GetOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("订单不存在: %w", err)
	}
	if orderPreCheck.UserID != userID {
		return fmt.Errorf(("没有权限"))
	}
	if orderPreCheck.Status == domain.StatusFilled || orderPreCheck.Status == domain.StatusCanceled {
		return fmt.Errorf("订单已完成或已取消")
	}

	var cancelOutboxMsg *outbox.Message
	err = s.txManager.ExecTx(ctx, func(ctx context.Context) error {
		order, err := s.orderRepo.GetOrderForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("锁定订单失败: %w", err)
		}

		if order.Status == domain.StatusFilled || order.Status == domain.StatusCanceled {
			return fmt.Errorf("订单已完成或取消，无法再次取消")
		}

		if s.outboxRepo != nil {
			event := &domain.OrderCancelRequestedEvent{
				EventType: domain.EventOrderCancelRequested,
				Symbol:    order.Symbol,
				OrderID:   orderID,
				Side:      order.Side,
				UserID:    order.UserID,
			}

			payload, marshalErr := outbox.MarshalPayload(event)
			if marshalErr != nil {
				return fmt.Errorf("序列化 OrderCancelRequestedvent 失败: %w", marshalErr)
			}

			outboxMsg := &outbox.Message{
				AggregateID:   order.ID.String(),
				AggregateType: "order_canceled_request",
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

	if s.rawPublisher != nil && cancelOutboxMsg != nil {
		go func(msgId uuid.UUID, msgPayload []byte, topic, partitionKey string) {
			hotCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if pubErr := s.rawPublisher.PublishRaw(hotCtx, topic, partitionKey, msgPayload); pubErr != nil {
				logger.Log.Warn("CancelOrder kafka发送失败", zap.String("outbox_id", msgId.String()), zap.Error(pubErr))
				return
			}

			select {
			case s.publishedMsgChan <- msgId:
			default:
				logger.Log.Warn("通道满了", zap.String("outbox_id", msgId.String()))
			}
		}(cancelOutboxMsg.ID, cancelOutboxMsg.Payload, cancelOutboxMsg.Topic, cancelOutboxMsg.PartitionKey)
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
