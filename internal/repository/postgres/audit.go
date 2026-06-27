package postgres

import (
	"context"
	"fmt"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Log(ctx context.Context, log *domain.AuditLog) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "InsertAuditLog")
	defer timer.ObserveDuration()

	q := `INSERT INTO audit_log (org_id, user_id, action, resource_type, resource_id, ip_address, user_agent, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8) /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	_, err := r.db.Exec(ctx, q, log.OrgID, log.UserID, log.Action, log.ResourceType, log.ResourceID, log.IPAddress, log.UserAgent, log.CreatedAt)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "InsertAuditLog").Inc()
		return fmt.Errorf("failed to write audit log: %w", err)
	}
	return nil
}
