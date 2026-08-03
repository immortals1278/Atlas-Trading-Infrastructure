package repository

import (
	"atlas-trading-infrastructure/internal/domain"
	"context"
)

func (r *PostgresRepository) CreatOrder(ctx context.Context, order *domain.Order) error {
	executor := r.GetExecutor(ctx)
	query := `
		INSERT INTO orders (id, symbol, price, quantity, side, status, filled_quantity)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := executor.Exec(ctx, query, order.ID, order.Symbol, order.Side,
		order.Price, order.Quantity, order.FilledQuantity, order.Status)

	return err
}
