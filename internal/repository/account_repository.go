package repository

import (
	"atlas-trading-infrastructure/internal/domain"
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (r *PostgresRepository) LockFunds(ctx context.Context, userid uuid.UUID, currency string, amount decimal.Decimal) error {
	executor := r.GetExecutor(ctx)
	query := `
	UPDATE accounts
	WHERE userid = $2 AND currency = $3 AND balance >= $1`

	tag, err := executor.Exec(ctx, query, amount, userid, currency)
	if err != nil {
		return err
	}

	//返回query操作多少行数据
	if tag.RowsAffected() == 0 {
		return domain.ErrInsufficientFunds
	}

	return nil
}
