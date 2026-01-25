package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyin-ad/flutter-code-mentor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (int, error)
	GetByID(ctx context.Context, id int) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) (int, error) {
	query := `
		INSERT INTO users (email, password_hash, role, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	var id int
	err := r.pool.QueryRow(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.FirstName,
		user.LastName,
	).Scan(&id, &user.CreatedAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return id, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, first_name, last_name, created_at, last_login
		FROM users
		WHERE id = $1
	`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.LastLogin,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, first_name, last_name, created_at, last_login
		FROM users
		WHERE email = $1
	`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.LastLogin,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
