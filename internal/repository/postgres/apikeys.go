package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyRepository struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, k *domain.APIKey) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "CreateAPIKey")
	defer timer.ObserveDuration()

	q := `INSERT INTO api_keys (id, org_id, name, key_hash, prefix, created_by, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7) /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	_, err := r.db.Exec(ctx, q, k.ID, k.OrgID, k.Name, k.KeyHash, k.Prefix, k.CreatedBy, k.CreatedAt)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "CreateAPIKey").Inc()
		return fmt.Errorf("failed to create api key: %w", err)
	}
	return nil
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetAPIKeyByHash")
	defer timer.ObserveDuration()

	q := `SELECT id, org_id, name, prefix, created_by, last_used_at, revoked_at, created_at
	      FROM api_keys WHERE key_hash = $1 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	k := &domain.APIKey{KeyHash: hash}
	err := r.db.QueryRow(ctx, q, hash).Scan(&k.ID, &k.OrgID, &k.Name, &k.Prefix, &k.CreatedBy, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetAPIKeyByHash").Inc()
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return k, nil
}

func (r *APIKeyRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.APIKey, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "ListAPIKeysByOrg")
	defer timer.ObserveDuration()

	q := `SELECT id, org_id, name, prefix, created_by, last_used_at, revoked_at, created_at
	      FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	rows, err := r.db.Query(ctx, q, orgID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "ListAPIKeysByOrg").Inc()
		return nil, fmt.Errorf("failed to query api keys: %w", err)
	}
	defer rows.Close()

	var keys []domain.APIKey
	for rows.Next() {
		var k domain.APIKey
		err := rows.Scan(&k.ID, &k.OrgID, &k.Name, &k.Prefix, &k.CreatedBy, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "RevokeAPIKey")
	defer timer.ObserveDuration()

	q := `UPDATE api_keys SET revoked_at = $1 WHERE id = $2 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	_, err := r.db.Exec(ctx, q, time.Now(), id)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "RevokeAPIKey").Inc()
		return fmt.Errorf("failed to revoke api key: %w", err)
	}
	return nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "UpdateAPIKeyLastUsed")
	defer timer.ObserveDuration()

	q := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	_, err := r.db.Exec(ctx, q, time.Now(), id)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "UpdateAPIKeyLastUsed").Inc()
		return fmt.Errorf("failed to update last used timestamp: %w", err)
	}
	return nil
}
