package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PgxPool defines the minimal interface satisfied by *pgxpool.Pool and test mocks.
type PgxPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const (
	NominalBaseMonthlyInfraUSD = 15.00
	HourlySpendCapUSD          = 1.00
)

type CostRepository struct {
	db PgxPool
}

func NewCostRepository(db PgxPool) *CostRepository {
	return &CostRepository{db: db}
}

// RecordCost stores an LLM attribution event for a tenant.
func (r *CostRepository) RecordCost(ctx context.Context, event *domain.LLMCostEvent) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	timer := observability.DbQueryDuration.WithLabelValues("postgres", "RecordCost")
	defer timer.ObserveDuration()

	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	q := `INSERT INTO tenant_llm_costs (id, org_id, anomaly_id, model, input_tokens, output_tokens, estimated_cost_usd, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8) /* request_id=` + observability.GetRequestID(ctx) + ` */`

	_, err := r.db.Exec(ctx, q, event.ID, event.OrgID, event.AnomalyID, event.Model, event.InputTokens, event.OutputTokens, event.EstimatedCostUSD, event.CreatedAt)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "RecordCost").Inc()
		return fmt.Errorf("failed to insert tenant cost event: %w", err)
	}
	return nil
}

// GetCostSummary computes the tenant's current spend, prorated infrastructure baseline,
// total explained incidents, and hourly spend status.
func (r *CostRepository) GetCostSummary(ctx context.Context, orgID string) (*domain.CostSummary, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetCostSummary")
	defer timer.ObserveDuration()

	// 1. Query tenant LLM spend aggregations with strict org_id isolation
	spendQuery := `
		SELECT 
			COALESCE(SUM(estimated_cost_usd), 0.0),
			COUNT(*),
			COALESCE(AVG(estimated_cost_usd), 0.0)
		FROM tenant_llm_costs
		WHERE org_id = $1 /* request_id=` + observability.GetRequestID(ctx) + ` */`

	var llmSpend, avgCost float64
	var totalIncidents int
	err := r.db.QueryRow(ctx, spendQuery, orgID).Scan(&llmSpend, &totalIncidents, &avgCost)
	if err != nil && err != pgx.ErrNoRows {
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetCostSummary").Inc()
		return nil, fmt.Errorf("failed to query tenant cost summary: %w", err)
	}

	// 2. Query tenant rolling 1-hour spend for storm circuit breaker status
	hourlyQuery := `
		SELECT COALESCE(SUM(estimated_cost_usd), 0.0)
		FROM tenant_llm_costs
		WHERE org_id = $1 AND created_at >= $2 /* request_id=` + observability.GetRequestID(ctx) + ` */`

	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	var hourlySpend float64
	err = r.db.QueryRow(ctx, hourlyQuery, orgID, oneHourAgo).Scan(&hourlySpend)
	if err != nil && err != pgx.ErrNoRows {
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetCostSummaryHourly").Inc()
		return nil, fmt.Errorf("failed to query tenant hourly spend: %w", err)
	}

	// 3. Count active organizations to compute prorated nominal infra allocation
	orgsCountQuery := `SELECT COUNT(*) FROM organizations /* request_id=` + observability.GetRequestID(ctx) + ` */`
	var activeOrgs int
	err = r.db.QueryRow(ctx, orgsCountQuery).Scan(&activeOrgs)
	if err != nil || activeOrgs < 1 {
		activeOrgs = 1
	}

	infraShare := NominalBaseMonthlyInfraUSD / float64(activeOrgs)
	totalCost := infraShare + llmSpend

	return &domain.CostSummary{
		OrgID:                   orgID,
		InfraShareUSD:           infraShare,
		LLMSpendUSD:             llmSpend,
		TotalCostUSD:            totalCost,
		TotalIncidentsExplained: totalIncidents,
		AvgCostPerIncidentUSD:   avgCost,
		HourlySpendUSD:          hourlySpend,
		HourlyCapUSD:            HourlySpendCapUSD,
		ActiveOrgsCount:         activeOrgs,
	}, nil
}

// GetCostBreakdown returns the per-incident cost ledger for a tenant with strict org_id isolation.
func (r *CostRepository) GetCostBreakdown(ctx context.Context, orgID string, limit, offset int) ([]domain.CostBreakdownItem, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetCostBreakdown")
	defer timer.ObserveDuration()

	q := `SELECT id, anomaly_id, model, input_tokens, output_tokens, estimated_cost_usd, created_at
	      FROM tenant_llm_costs
	      WHERE org_id = $1
	      ORDER BY created_at DESC
	      LIMIT $2 OFFSET $3 /* request_id=` + observability.GetRequestID(ctx) + ` */`

	rows, err := r.db.Query(ctx, q, orgID, limit, offset)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetCostBreakdown").Inc()
		return nil, fmt.Errorf("failed to query tenant cost breakdown: %w", err)
	}
	defer rows.Close()

	items := make([]domain.CostBreakdownItem, 0)
	for rows.Next() {
		var item domain.CostBreakdownItem
		if err := rows.Scan(&item.ID, &item.AnomalyID, &item.Model, &item.InputTokens, &item.OutputTokens, &item.EstimatedCostUSD, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cost item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// GetHourlySpend retrieves the total amount spent by an organization in the last hour.
func (r *CostRepository) GetHourlySpend(ctx context.Context, orgID string) (float64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	q := `SELECT COALESCE(SUM(estimated_cost_usd), 0.0)
	      FROM tenant_llm_costs
	      WHERE org_id = $1 AND created_at >= $2 /* request_id=` + observability.GetRequestID(ctx) + ` */`

	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	var hourlySpend float64
	err := r.db.QueryRow(ctx, q, orgID, oneHourAgo).Scan(&hourlySpend)
	if err != nil {
		return 0, fmt.Errorf("failed to get hourly spend: %w", err)
	}
	return hourlySpend, nil
}
