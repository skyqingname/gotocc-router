package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/lib/pq"
)

// agentRepository stores LC-024 agent enrollment state. Membership is decided by
// the presence of a row: no row means the user has never applied.
type agentRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewAgentRepository(client *dbent.Client, db *sql.DB) service.AgentRepository {
	return &agentRepository{client, db}
}

func rowsAffected(res sql.Result) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *agentRepository) executor(ctx context.Context) sqlExecutor {
	return clientFromContext(ctx, r.client)
}

// EligibleAmong reports which of the given users may receive a rebate right now.
// Only 'approved' counts: grandfathered users carry an approved status.
func (r *agentRepository) EligibleAmong(ctx context.Context, userIDs []int64) (map[int64]bool, error) {
	eligible := make(map[int64]bool, len(userIDs))
	if len(userIDs) == 0 {
		return eligible, nil
	}
	rows, err := r.executor(ctx).QueryContext(ctx,
		`SELECT user_id FROM agent_profiles WHERE status = 'approved' AND user_id = ANY($1)`, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		eligible[id] = true
	}
	return eligible, rows.Err()
}

func (r *agentRepository) Overview(ctx context.Context, userID int64) (*service.AgentProfile, error) {
	p := &service.AgentProfile{UserID: userID}
	var status, source sql.NullString
	var appliedAt, reviewedAt sql.NullTime
	err := scanSingleRow(ctx, r.executor(ctx),
		`SELECT status, source, applied_at, reviewed_at FROM agent_profiles WHERE user_id = $1`,
		[]any{userID}, &status, &source, &appliedAt, &reviewedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	p.Status = status.String
	p.Source = source.String
	if appliedAt.Valid {
		p.AppliedAt = &appliedAt.Time
	}
	if reviewedAt.Valid {
		p.ReviewedAt = &reviewedAt.Time
	}
	return p, nil
}

// Apply directly activates a membership. An existing approval is permanent and
// retains its original source and effective time on duplicate submissions.
func (r *agentRepository) Apply(ctx context.Context, userID int64) (bool, error) {
	res, err := r.executor(ctx).ExecContext(ctx, `
INSERT INTO agent_profiles (user_id, status, source, applied_at, reviewed_at)
SELECT id, 'approved', 'applied', NOW(), NOW() FROM users WHERE id = $1 AND deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING`, userID)
	if err != nil {
		return false, err
	}
	return rowsAffected(res)
}

// List returns enrolled agents for the read-only admin directory, newest first.
func (r *agentRepository) List(ctx context.Context, search string, page, size int) ([]service.AgentApplication, int64, error) {
	args := []any{}
	where := ""
	if search = strings.TrimSpace(search); search != "" {
		args = append(args, "%"+search+"%")
		where = " AND (u.email ILIKE $1 OR u.username ILIKE $1 OR u.id::text ILIKE $1)"
	}
	q := r.executor(ctx)
	var total int64
	if err := scanSingleRow(ctx, q,
		`SELECT COUNT(*) FROM agent_profiles a JOIN users u ON u.id = a.user_id WHERE u.deleted_at IS NULL`+where,
		args, &total); err != nil {
		return nil, 0, err
	}
	pageArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := q.QueryContext(ctx, `
SELECT a.user_id, u.username, u.email, u.created_at, a.status, a.source, a.applied_at, a.reviewed_at
FROM agent_profiles a JOIN users u ON u.id = a.user_id
WHERE u.deleted_at IS NULL`+where+`
ORDER BY a.applied_at DESC NULLS LAST, a.user_id DESC
LIMIT $`+itoa(len(args)+1)+` OFFSET $`+itoa(len(args)+2), pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []service.AgentApplication{}
	for rows.Next() {
		var item service.AgentApplication
		var appliedAt, reviewedAt sql.NullTime
		if err := rows.Scan(&item.UserID, &item.Username, &item.Email, &item.CreatedAt,
			&item.Status, &item.Source, &appliedAt, &reviewedAt); err != nil {
			return nil, 0, err
		}
		if appliedAt.Valid {
			item.AppliedAt = &appliedAt.Time
		}
		if reviewedAt.Valid {
			item.ReviewedAt = &reviewedAt.Time
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

// EnsureCutoff records the enrollment cutoff exactly once and backfills the
// users already on the platform as grandfathered agents. Both statements run in
// one transaction so a crash cannot leave the cutoff set without its backfill.
//
// The cutoff write is conditional on the sentinel, so concurrent instances
// cannot move the boundary; only the first successful write wins. The advisory
// lock serializes competing boots and is released with the transaction.
func (r *agentRepository) EnsureCutoff(ctx context.Context, sentinel string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext('agent_enrollment_cutoff'))"); err != nil {
		return false, err
	}
	res, err := tx.ExecContext(ctx, `
INSERT INTO settings (key, value, updated_at) VALUES ($1, NOW()::text, NOW())
ON CONFLICT (key) DO UPDATE SET value = NOW()::text, updated_at = NOW()
 WHERE settings.value = $2`, service.SettingKeyAgentEnrollmentCutoff, sentinel)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		// A previous boot already fixed the boundary; keeping it is the contract.
		return false, tx.Commit()
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO agent_profiles (user_id, status, source, applied_at, reviewed_at)
SELECT id, 'approved', 'grandfathered', NOW(), NOW() FROM users
WHERE created_at < (SELECT value::timestamptz FROM settings WHERE key = $1)
ON CONFLICT (user_id) DO NOTHING`, service.SettingKeyAgentEnrollmentCutoff); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
