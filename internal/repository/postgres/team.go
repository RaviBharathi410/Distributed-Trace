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

type TeamRepository struct {
	db *pgxpool.Pool
}

func NewTeamRepository(db *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]domain.User, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "ListTeamMembers")
	defer timer.ObserveDuration()

	q := `SELECT id, email, org_id, role, created_at, updated_at 
	      FROM users WHERE org_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	rows, err := r.db.Query(ctx, q, orgID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "ListTeamMembers").Inc()
		return nil, fmt.Errorf("failed to query team members: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var roleStr string
		err := rows.Scan(&u.ID, &u.Email, &u.OrgID, &roleStr, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		u.Role = domain.Role(roleStr)
		users = append(users, u)
	}
	return users, nil
}

func (r *TeamRepository) InviteMember(ctx context.Context, invite *domain.TeamInvite) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "InviteTeamMember")
	defer timer.ObserveDuration()

	q := `INSERT INTO team_invites (id, org_id, email, role, invited_by, expires_at, token)
	      VALUES ($1, $2, $3, $4, $5, $6, $7) /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	_, err := r.db.Exec(ctx, q, invite.ID, invite.OrgID, invite.Email, string(invite.Role), invite.InvitedBy, invite.ExpiresAt, invite.Token)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "InviteTeamMember").Inc()
		return fmt.Errorf("failed to create team invite: %w", err)
	}
	return nil
}

func (r *TeamRepository) GetInviteByToken(ctx context.Context, token string) (*domain.TeamInvite, error) {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "GetInviteByToken")
	defer timer.ObserveDuration()

	q := `SELECT id, org_id, email, role, invited_by, accepted_at, expires_at, token 
	      FROM team_invites WHERE token = $1 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	
	invite := &domain.TeamInvite{}
	var roleStr string
	err := r.db.QueryRow(ctx, q, token).Scan(&invite.ID, &invite.OrgID, &invite.Email, &roleStr, &invite.InvitedBy, &invite.AcceptedAt, &invite.ExpiresAt, &invite.Token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		observability.DbErrorsTotal.WithLabelValues("postgres", "GetInviteByToken").Inc()
		return nil, fmt.Errorf("failed to get invite: %w", err)
	}
	invite.Role = domain.Role(roleStr)
	return invite, nil
}

func (r *TeamRepository) AcceptInvite(ctx context.Context, inviteID uuid.UUID) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "AcceptInvite")
	defer timer.ObserveDuration()

	q := `UPDATE team_invites SET accepted_at = $1 WHERE id = $2 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	_, err := r.db.Exec(ctx, q, time.Now(), inviteID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "AcceptInvite").Inc()
		return fmt.Errorf("failed to accept team invite: %w", err)
	}
	return nil
}

func (r *TeamRepository) RemoveMember(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) error {
	timer := observability.DbQueryDuration.WithLabelValues("postgres", "RemoveTeamMember")
	defer timer.ObserveDuration()

	q := `UPDATE users SET deleted_at = $1 WHERE id = $2 AND org_id = $3 /* request_id=` + observability.GetRequestID(ctx) + ` */`
	_, err := r.db.Exec(ctx, q, time.Now(), userID, orgID)
	if err != nil {
		observability.DbErrorsTotal.WithLabelValues("postgres", "RemoveTeamMember").Inc()
		return fmt.Errorf("failed to soft delete team user: %w", err)
	}
	return nil
}
