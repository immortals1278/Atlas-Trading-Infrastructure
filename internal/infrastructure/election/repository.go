package election

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NerRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// 租约有效时间
var leaseDuration = 15 * time.Second

// 竞选leader
func (r *Repository) AcquireLock(ctx context.Context, partition, instanceID string) (fencingToken int64, acquired bool, err error) {
	now := time.Now().UnixMilli()
	expiresAt := time.Now().Add(leaseDuration).UnixMilli()

	row := r.pool.QueryRow(ctx, ``, partition, instanceID, expiresAt, now)

	err = row.Scan(&fencingToken)
	if err != nil {
		// 目前有人持有租约 竞选失败
		if errors.Is(err, pgx.ErrNoRows) { // pgx 找不到任何符合条件的行
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("取得leader失败 : %w", zap.Error(err))

	}

	return fencingToken, true, nil
}
