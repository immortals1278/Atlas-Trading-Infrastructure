package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKeyType struct{}

var TxKey = txKeyType{}

// 从context 中取得事务pgx.Tx
func GetTx(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(TxKey).(pgx.Tx)
	return tx
}
