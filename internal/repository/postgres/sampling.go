package postgres

import (
	"context"
	"fmt"

	"github.com/RaviBharathi410/distributedtrace/internal/domain"
	"github.com/RaviBharathi410/distributedtrace/internal/observability"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SamplingRepository struct {
	db *pgxpool.Pool
}

func NewSamplingRepository(db *pgxpool.Pool) *SamplingRepository {
	return &SamplingRepository{db: db}
}

func (r *SamplingRepository) GetRules(ctx context.Context, orgID uuid.UUID) ([]domain.SamplingRule, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetSamplingRules")
	defer timer.ObserveDuration()

	q := `SELECT id, org_id, rule_type, threshold_ms, sample_rate, enabled, created_at 
	      FROM sampling_rules WHERE org_id = $1 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	rows, err := r.db.Query(ctx, q, orgID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetSamplingRules").Inc()
		return nil, fmt.Errorf("failed to query sampling rules: %w", err)
	}
	defer rows.Close()

	var rules []domain.SamplingRule
	for rows.Next() {
		var sr domain.SamplingRule
		err := rows.Scan(&sr.ID, &sr.OrgID, &sr.RuleType, &sr.ThresholdMs, &sr.SampleRate, &sr.Enabled, &sr.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sampling rule: %w", err)
		}
		rules = append(rules, sr)
	}
	return rules, nil
}

func (r *SamplingRepository) UpdateRules(ctx context.Context, orgID uuid.UUID, rules []domain.SamplingRule) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "UpdateSamplingRules")
	defer timer.ObserveDuration()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clean existing rules
	dq := `DELETE FROM sampling_rules WHERE org_id = $1`
	_, err = tx.Exec(ctx, dq, orgID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "DeleteSamplingRules").Inc()
		return fmt.Errorf("failed to clear old rules: %w", err)
	}

	// Insert new rules
	iq := `INSERT INTO sampling_rules (id, org_id, rule_type, threshold_ms, sample_rate, enabled, created_at)
	       VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	for _, rule := range rules {
		_, err = tx.Exec(ctx, iq, rule.ID, orgID, rule.RuleType, rule.ThresholdMs, rule.SampleRate, rule.Enabled, rule.CreatedAt)
		if err != nil {
			observability.DbErrorsTotal.WithLabelValues("postgres", "InsertSamplingRules").Inc()
			return fmt.Errorf("failed to insert sampling rule: %w", err)
		}
	}

	return tx.Commit(ctx)
}
