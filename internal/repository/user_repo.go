package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dedehudianto12/simbu-tech-backend/internal/model"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, email, password_hash, role, created_at
		FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return model.User{}, fmt.Errorf("user_repo.GetByEmail: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, email, password_hash, role, created_at
		FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return model.User{}, fmt.Errorf("user_repo.GetByID: %w", err)
	}
	return u, nil
}
