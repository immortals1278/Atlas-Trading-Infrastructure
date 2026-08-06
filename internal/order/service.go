package order

import (
	"atlas-trading-infrastructure/internal/domain"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct {
	orderRepo   OrderRepository
	accountRepo AccountRepository
	txManager   DBTransaction
}

func NewService(
	orderRepo OrderRepository,
	txManager DBTransaction,
	accountRepo AccountRepository,
) *Service {
	s := &Service{
		orderRepo:   orderRepo,
		txManager:   txManager,
		accountRepo: accountRepo,
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

		return nil
	})

	if err != nil {
		return err
	}
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
