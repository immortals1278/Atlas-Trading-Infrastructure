package outbox

import (
	"atlas-trading-infrastructure/internal/infrastructure/db"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) getExecutor(ctx context.Context) dbExecutor {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *Repository) MarkPublishedWorker(ctx context.Context, batch []uuid.UUID) error {
	if len(batch) == 0 {
		return nil
	}
	_, err := r.getExecutor(ctx).Exec(ctx, `
	DELETE FROM outbox_messages
	WHERE id = ANY($1)`,
		batch)

	return err
}

func (r *Repository) InsertMsg(ctx context.Context, msg Message) error {
	msg.ID, _ = uuid.NewV7()
	msg.CreatedAt = time.Now().UnixMilli()
	msg.Status = StatusPending

	_, err := r.getExecutor(ctx).Exec(ctx, `
		INSERT INTO outbox_messages
			(id, aggregate_id, aggregate_type, topic, partition_key, payload, status, retry_count, created_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		msg.ID,
		msg.AggregateID,
		msg.AggregateType,
		msg.Topic,
		msg.PartitionKey,
		msg.Payload,
		msg.Status,
		msg.RetryCount,
		msg.CreatedAt,
		msg.PublishedAt,
	)
	return err
}
