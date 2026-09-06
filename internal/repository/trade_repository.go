package repository

import (
	"atlas-trading-infrastructure/internal/matching/engine"
	"context"
)

func (r *PostgresRepository) CreateTrade(ctx context.Context, trade *engine.Trade) error {
	executor := r.GetExecutor(ctx)

	query := `
	INSERT INTO trades (id, symbol, maker_order_id, taker_order_id, price, quantity, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := executor.Exec(ctx, query, trade.ID, trade.Symbol, trade.MakerOrderID, trade.TakerOrderID,
		trade.Price, trade.Quantity, trade.CreatedAt)
	return err
}
