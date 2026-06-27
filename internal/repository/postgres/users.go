package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "CreateUser")
	defer timer.ObserveDuration()

	q := `INSERT INTO users (id, email, password_hash, org_id, role, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7) /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	_, err := r.db.Exec(ctx, q, u.ID, u.Email, u.PasswordHash, u.OrgID, string(u.Role), u.CreatedAt, u.UpdatedAt)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "CreateUser").Inc()
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetUserByEmail")
	defer timer.ObserveDuration()

	q := `SELECT id, email, password_hash, org_id, role, created_at, updated_at 
	      FROM users WHERE email = $1 AND deleted_at IS NULL /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	u := &domain.User{}
	var roleStr string
	err := r.db.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.OrgID, &roleStr, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetUserByEmail").Inc()
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	u.Role = domain.Role(roleStr)
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetUserByID")
	defer timer.ObserveDuration()

	q := `SELECT id, email, org_id, role, created_at, updated_at 
	      FROM users WHERE id = $1 AND deleted_at IS NULL /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	u := &domain.User{}
	var roleStr string
	err := r.db.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.OrgID, &roleStr, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetUserByID").Inc()
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	u.Role = domain.Role(roleStr)
	return u, nil
}
