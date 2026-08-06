package repository

import (
	"atlas-trading-infrastructure/internal/domain"
	"context"

	"github.com/google/uuid"
)

func (r *PostgresRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	executor := r.GetExecutor(ctx)
	query := `
		INSERT INTO orders (id, userid, symbol, price, quantity, side, status, filled_quantity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := executor.Exec(ctx, query, order.ID, order.Symbol, order.Side,
		order.Price, order.Quantity, order.FilledQuantity, order.Status)

	return err
}

func (r *PostgresRepository) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	executor := r.GetExecutor(ctx)
	query := `
		SELECT id, userid, symbol, side, price, quantity, filled_quantity, status
		FROM orders WHERE id = $1`

	row := executor.QueryRow(ctx, query, id)
	return scanOrder(row)
}

func (r *PostgresRepository) UpdateOrder(ctx context.Context, order *domain.Order) error {
	executor := r.GetExecutor(ctx)
	query := `
		UPDATE orders 
		SET filled_quantity = $1, status = $2
		WHERE id = $4`

	_, err := executor.Exec(ctx, query,
		order.FilledQuantity, order.Status, order.ID)
	return err
}

// 多行单行查询都实现了这个接口
type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrder(row rowScanner) (*domain.Order, error) {
	var o domain.Order
	err := row.Scan(
		&o.ID, &o.Symbol, &o.Side,
		&o.Price, &o.Quantity, &o.FilledQuantity, &o.Status) // 谁实现接口，谁就能用
	if err != nil {
		return nil, err
	}
	return &o, nil
}
