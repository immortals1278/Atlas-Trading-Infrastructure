package repository

import (
	"atlas-trading-infrastructure/internal/infrastructure/db"
	"atlas-trading-infrastructure/internal/order"
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// 承诺 PostgresRepository 实现了 OrderRepository 接口，如果没实现则通过不了编译
var _ order.OrderRepository = (*PostgresRepository)(nil)
var _ order.DBTransaction = (*PostgresRepository)(nil)

// 通过BeginFunc来实现成功提交失败回滚
func (r *PostgresRepository) ExecTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		txCtx := context.WithValue(ctx, db.TxKey, tx) //将事务对象存进context
		return fn(txCtx)

	})
}

// pgx.pool和pgx.tx都已经实现了
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// 有些操作需要在事务内执行，有些不用(用pool)
func (r *PostgresRepository) GetExecutor(ctx context.Context) DBExecutor {
	if tx := db.GetTx(ctx); ctx != nil {
		return tx
	}
	return r.db
}

func (r *PostgresRepository) ValidateFencingTokenTx(ctx context.Context, partition string, token int64) (bool, error) {
	var currentToken int64

	err := r.GetExecutor(ctx).QueryRow(ctx, `
		SELECT fencing_token
		FROM partition_leader_locks
		WHERE partition = $1
		FOR SHARE`,
		partition,
	).Scan(&currentToken)

	if err != nil {
		// pgx.ErrNoRows 表示db中没有leader持有锁
		// → 系统刚启动允许通过
		return true, nil
	}

	// 检查是否为最新的fencingToken
	if token < currentToken {
		return false, nil
	}
	return true, nil
}
