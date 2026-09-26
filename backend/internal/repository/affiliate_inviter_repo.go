package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

func (r *affiliateRepository) GetInviter(ctx context.Context, userID int64) (*service.AffiliateInviterState, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT u.id, ua.inviter_id, COALESCE(p.email, ''), COALESCE(p.username, ''),
 COALESCE(ua.attribution_code_type, ''), COALESCE(ua.attribution_code, ''),
 COALESCE(ua.inviter_version, 0), ua.inviter_effective_at
FROM users u
LEFT JOIN user_affiliates ua ON ua.user_id = u.id
LEFT JOIN users p ON p.id = ua.inviter_id
WHERE u.id = $1 AND u.deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrUserNotFound
	}
	state := &service.AffiliateInviterState{}
	var inviterID sql.NullInt64
	var effectiveAt sql.NullTime
	var inviter service.AffiliateInviterUser
	if err := rows.Scan(&state.UserID, &inviterID, &inviter.Email, &inviter.Username,
		&state.CodeType, &state.Code, &state.Version, &effectiveAt); err != nil {
		return nil, err
	}
	if inviterID.Valid {
		inviter.ID = inviterID.Int64
		state.Inviter = &inviter
	}
	if effectiveAt.Valid {
		state.EffectiveAt = &effectiveAt.Time
	}
	return state, rows.Err()
}

func (r *affiliateRepository) ResolveInviterCode(ctx context.Context, codeType, code string) (*service.AffiliateInviterUser, error) {
	// The legacy code_type field is accepted by callers but no longer selects a directory.
	query := `SELECT c.owner_user_id, COALESCE(u.email, ''), COALESCE(u.username, ''),
 COALESCE(u.status, ''), u.deleted_at
FROM reusable_invitation_codes c LEFT JOIN users u ON u.id = c.owner_user_id
WHERE UPPER(c.code) = $1`
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, query, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateCodeInvalid
	}
	var id sql.NullInt64
	var deleted sql.NullTime
	var status string
	var result service.AffiliateInviterUser
	if err := rows.Scan(&id, &result.Email, &result.Username, &status, &deleted); err != nil {
		return nil, err
	}
	if !id.Valid {
		return nil, service.ErrAffiliateCodeOwnerMissing
	}
	if deleted.Valid || status != service.StatusActive {
		return nil, service.ErrAffiliateInviterUnavailable
	}
	result.ID = id.Int64
	return &result, rows.Err()
}

func (r *affiliateRepository) ChangeInviter(ctx context.Context, userID int64, input *service.AffiliateInviterChange) error {
	return r.withTx(ctx, func(ctx context.Context, client *dbent.Client) error {
		if err := lockAffiliateBindings(ctx, client); err != nil {
			return err
		}
		current, err := r.GetInviter(ctx, userID)
		if err != nil {
			return err
		}
		if current.Version != *input.ExpectedVersion {
			return service.ErrAffiliateInviterChanged
		}
		target, err := r.ResolveInviterCode(ctx, input.CodeType, input.Code)
		if err != nil {
			return err
		}
		if target.ID != input.ResolvedUserID {
			return service.ErrAffiliateCodeOwnerChanged
		}
		if target.ID == userID {
			return service.ErrAffiliateInviterCycle
		}
		if current.Inviter != nil && current.Inviter.ID == target.ID && current.CodeType == input.CodeType && current.Code == input.Code {
			return nil
		}
		rows, err := client.QueryContext(ctx, `WITH RECURSIVE ancestors AS (
 SELECT user_id, inviter_id FROM user_affiliates WHERE user_id = $1
 UNION
 SELECT ua.user_id, ua.inviter_id FROM user_affiliates ua JOIN ancestors a ON ua.user_id = a.inviter_id
) SELECT EXISTS (SELECT 1 FROM ancestors WHERE user_id = $2)`, target.ID, userID)
		if err != nil {
			return err
		}
		var cycle bool
		if rows.Next() {
			err = rows.Scan(&cycle)
		}
		rowsErr := rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if rowsErr != nil {
			return rowsErr
		}
		if cycle {
			return service.ErrAffiliateInviterCycle
		}
		if _, err := ensureUserAffiliateWithClient(ctx, client, userID); err != nil {
			return err
		}
		if _, err := ensureUserAffiliateWithClient(ctx, client, target.ID); err != nil {
			return err
		}
		if _, err := client.ExecContext(ctx, `UPDATE user_affiliates
SET inviter_id = $2, attribution_code_type = $3, attribution_code = $4,
 inviter_version = inviter_version + 1, inviter_effective_at = clock_timestamp(), updated_at = NOW()
WHERE user_id = $1`, userID, target.ID, input.CodeType, input.Code); err != nil {
			return err
		}
		previousID := int64(0)
		if current.Inviter != nil {
			previousID = current.Inviter.ID
		}
		if _, err := client.ExecContext(ctx, `UPDATE user_affiliates parent
SET aff_count = (SELECT count(*) FROM user_affiliates child JOIN users u ON u.id = child.user_id
 WHERE child.inviter_id = parent.user_id AND u.deleted_at IS NULL), updated_at = NOW()
WHERE parent.user_id IN ($1, $2)`, previousID, target.ID); err != nil {
			return err
		}
		previous, err := json.Marshal(current.Inviter)
		if err != nil {
			return err
		}
		next, err := json.Marshal(target)
		if err != nil {
			return err
		}
		_, err = client.ExecContext(ctx, `INSERT INTO affiliate_inviter_changes
(user_id, version, previous_inviter, inviter, previous_code_type, previous_code, code_type, code, actor_user_id, auth_method)
VALUES ($1,$2,$3::jsonb,$4::jsonb,$5,$6,$7,$8,$9,$10)`, userID, current.Version+1,
			string(previous), string(next), current.CodeType, current.Code, input.CodeType, input.Code,
			nullableInt64Arg(input.ActorUserID), input.AuthMethod)
		return err
	})
}

func (r *affiliateRepository) LockInviterBindings(ctx context.Context) error {
	return lockAffiliateBindings(ctx, clientFromContext(ctx, r.client))
}

func (r *affiliateRepository) GetInviterChain(ctx context.Context, userID int64, generations int) ([]int64, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `WITH RECURSIVE chain AS (
 SELECT inviter_id, 1 AS level, ARRAY[user_id, inviter_id] AS path
 FROM user_affiliates WHERE user_id = $1 AND inviter_id IS NOT NULL AND inviter_id <> $1
 UNION ALL
 SELECT ua.inviter_id, c.level + 1, c.path || ua.inviter_id
 FROM chain c JOIN user_affiliates ua ON ua.user_id = c.inviter_id
 WHERE c.level < $2 AND ua.inviter_id IS NOT NULL AND NOT ua.inviter_id = ANY(c.path)
) SELECT inviter_id FROM chain ORDER BY level`, userID, generations)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, generations)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *affiliateRepository) CapturePaymentInvitersForRedeem(ctx context.Context, code string, userID int64, generations int) error {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `SELECT id FROM payment_orders
 WHERE recharge_code = $1 AND user_id = $2 AND order_type = 'balance' AND paid_at IS NOT NULL AND pay_amount > 0`, code, userID)
	if err != nil {
		return err
	}
	if !rows.Next() {
		err := rows.Err()
		rows.Close()
		return err
	}
	var orderID int64
	err = rows.Scan(&orderID)
	rows.Close()
	if err != nil {
		return err
	}
	if err := r.LockInviterBindings(ctx); err != nil {
		return err
	}
	ids, err := r.GetInviterChain(ctx, userID, generations)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	_, err = clientFromContext(ctx, r.client).ExecContext(ctx, `INSERT INTO affiliate_payment_attributions
(order_id, user_id, inviter_ids, capture_source) VALUES ($1,$2,$3::jsonb,'credit')
ON CONFLICT (order_id) DO NOTHING`, orderID, userID, string(encoded))
	return err
}

func (r *affiliateRepository) GetPaymentInviters(ctx context.Context, orderID int64) ([]int64, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx,
		`SELECT inviter_ids FROM affiliate_payment_attributions WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("payment affiliate attribution missing for order %d", orderID)
	}
	var encoded []byte
	if err := rows.Scan(&encoded); err != nil {
		return nil, err
	}
	var ids []int64
	if err := json.Unmarshal(encoded, &ids); err != nil {
		return nil, err
	}
	return ids, rows.Err()
}
