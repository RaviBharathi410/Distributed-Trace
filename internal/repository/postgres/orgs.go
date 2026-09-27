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

type OrganizationRepository struct {
	db *pgxpool.Pool
}

func NewOrganizationRepository(db *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "CreateOrganization")
	defer timer.ObserveDuration()

	q := `INSERT INTO organizations (id, name, plan, created_at)
	      VALUES ($1, $2, $3, $4) /* request_id=` + observability.GetRequestID(ctx) + ` */`

	_, err := r.db.Exec(ctx, q, org.ID, org.Name, org.Plan, org.CreatedAt)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "CreateOrganization").Inc()
		return fmt.Errorf("failed to create organization: %w", err)
	}
	return nil
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetOrganizationByID")
	defer timer.ObserveDuration()

	q := `SELECT id, name, plan, created_at 
	      FROM organizations WHERE id = $1 /* request_id=` + observability.GetRequestID(ctx) + ` */`

	org := &domain.Organization{}
	err := r.db.QueryRow(ctx, q, id).Scan(&org.ID, &org.Name, &org.Plan, &org.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetOrganizationByID").Inc()
		return nil, fmt.Errorf("failed to get organization by id: %w", err)
	}
	return org, nil
}
