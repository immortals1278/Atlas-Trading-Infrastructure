package repository

import (
	"atlas-trading-infrastructure/internal/domain"
	"context"
)

func (r *PostgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	executor := r.GetExecutor(ctx)

	query := `INSERT INTO user (id, email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := executor.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	executor := r.GetExecutor(ctx)
	query := `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = $1`

	row := executor.QueryRow(ctx, query, email)
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
