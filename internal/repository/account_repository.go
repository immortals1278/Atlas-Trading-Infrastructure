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
	SET balance = balance - $1, locked = locked + $1,
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

func (r *PostgresRepository) BatchLockFunds(ctx context.Context, lockFunds map[uuid.UUID]map[string]decimal.Decimal) error {
	executor := r.GetExecutor(ctx)

	UserID := make([]uuid.UUID, 0, len(lockFunds))

	for uid := range lockFunds {
		UserID = append(UserID, uid)
	}
	// 排序防死锁
	for i := 0; i < len(lockFunds); i++ {
		for j := i + 1; j < len(lockFunds); j++ {
			if UserID[i].String() > UserID[j].String() {
				UserID[i], UserID[j] = UserID[j], UserID[i]
			}
		}
	}

	for _, uid := range UserID {
		currency := lockFunds[uid] //cur -> amount
		curkey := make([]string, 0, len(currency))
		for cur := range currency {
			curkey = append(curkey, cur)
		}
		//给币种排序防死锁
		for i := 0; i < len(curkey); i++ {
			for j := i + 1; j < len(curkey); j++ {
				if curkey[i] > curkey[j] {
					curkey[i], curkey[j] = curkey[j], curkey[i]
				}
			}
		}

		for _, cur := range curkey {
			amount := currency[cur]
			query := `
				UPDATE accounts 
				SET balance = balance - $1, locked = locked + $1, 
				WHERE user_id = $2 AND currency = $3 AND balance >= $1`
			tag, err := executor.Exec(ctx, query, amount, uid, cur)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return domain.ErrInsufficientFunds
			}
		}
	}

	return nil
}

func (r *PostgresRepository) UnlockFunds(ctx context.Context, userID uuid.UUID, currency string, amount decimal.Decimal) error {
	executor := r.GetExecutor(ctx)
	query := `
	UPDATE accounts
	SET balance = balance + $1, locked = locked - $1,
	WHERE userid = $2 AND currency = $3 AND balance >= $1`

	_, err := executor.Exec(ctx, query, amount, userID, currency)
	if err != nil {
		return err
	}

	return nil
}
