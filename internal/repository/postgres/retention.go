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

type RetentionRepository struct {
	db *pgxpool.Pool
}

func NewRetentionRepository(db *pgxpool.Pool) *RetentionRepository {
	return &RetentionRepository{db: db}
}

func (r *RetentionRepository) Get(ctx context.Context, orgID uuid.UUID) (*domain.RetentionPolicy, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetRetentionPolicy")
	defer timer.ObserveDuration()

	q := `SELECT id, org_id, retention_days, updated_at 
	      FROM retention_policies WHERE org_id = $1 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	rp := &domain.RetentionPolicy{}
	err := r.db.QueryRow(ctx, q, orgID).Scan(&rp.ID, &rp.OrgID, &rp.RetentionDays, &rp.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return default
			return &domain.RetentionPolicy{
				ID:            uuid.New(),
				OrgID:         orgID,
				RetentionDays: 30,
				UpdatedAt:     time.Now(),
			}, nil
		}
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetRetentionPolicy").Inc()
		return nil, fmt.Errorf("failed to get retention policy: %w", err)
	}
	return rp, nil
}

func (r *RetentionRepository) Update(ctx context.Context, rp *domain.RetentionPolicy) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "UpdateRetentionPolicy")
	defer timer.ObserveDuration()

	q := `INSERT INTO retention_policies (id, org_id, retention_days, updated_at)
	      VALUES ($1, $2, $3, $4)
	      ON CONFLICT (org_id) DO UPDATE 
	      SET retention_days = EXCLUDED.retention_days, updated_at = EXCLUDED.updated_at 
	      /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	_, err := r.db.Exec(ctx, q, rp.ID, rp.OrgID, rp.RetentionDays, rp.UpdatedAt)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "UpdateRetentionPolicy").Inc()
		return fmt.Errorf("failed to update retention policy: %w", err)
	}
	return nil
}
