package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/google/uuid"
)

type resellerRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewResellerRepository(client *dbent.Client, db *sql.DB) service.ResellerRepository {
	return &resellerRepository{client, db}
}
func (r *resellerRepository) executor(ctx context.Context) sqlExecutor {
	return clientFromContext(ctx, r.client)
}
func resellerProfile(ctx context.Context, q sqlExecutor, userID int64) (*service.ResellerProfile, error) {
	p := &service.ResellerProfile{}
	err := scanSingleRow(ctx, q, `SELECT u.id,COALESCE(p.enabled,FALSE),COALESCE(p.invitation_code,''),COALESCE(p.default_multiplier,$2)::float8 FROM users u LEFT JOIN reseller_profiles p ON p.user_id=u.id WHERE u.id=$1`, []any{userID, reseller.DefaultMultiplier}, &p.UserID, &p.Enabled, &p.InvitationCode, &p.DefaultMultiplier)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
func (r *resellerRepository) Profile(ctx context.Context, id int64) (*service.ResellerProfile, error) {
	return resellerProfile(ctx, r.executor(ctx), id)
}

// rebate_rates is intentionally neither read nor written since LC-024. The
// column is left in place: dropping it would not change behaviour and would make
// the change destructive.
func (r *resellerRepository) SaveProfile(ctx context.Context, id int64, enabled bool) (*service.ResellerProfile, error) {
	code := "RS-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
	result, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO reseller_profiles(user_id,enabled,invitation_code,default_multiplier) SELECT id,$2,$3,$4 FROM users WHERE id=$1 AND deleted_at IS NULL ON CONFLICT(user_id) DO UPDATE SET enabled=EXCLUDED.enabled,updated_at=NOW()`, id, enabled, code, reseller.DefaultMultiplier)
	if err != nil {
		return nil, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, service.ErrUserNotFound
	}
	return r.Profile(ctx, id)
}
func (r *resellerRepository) Invitation(ctx context.Context, code string) (*service.ResellerProfile, error) {
	var id int64
	err := scanSingleRow(ctx, r.executor(ctx), `SELECT p.user_id FROM reseller_profiles p JOIN users u ON u.id=p.user_id WHERE p.invitation_code=$1 AND p.enabled AND u.status='active' AND u.deleted_at IS NULL`, []any{strings.ToUpper(strings.TrimSpace(code))}, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrInvitationCodeInvalid
	}
	if err != nil {
		return nil, err
	}
	return r.Profile(ctx, id)
}
func (r *resellerRepository) BindCustomer(ctx context.Context, userID, ownerID int64) error {
	q := r.executor(ctx)
	var enabled bool
	err := scanSingleRow(ctx, q, `SELECT enabled FROM reseller_profiles WHERE user_id=$1 FOR SHARE`, []any{ownerID}, &enabled)
	if err != nil {
		return err
	}
	if !enabled {
		return service.ErrInvitationCodeInvalid
	}
	_, err = q.ExecContext(ctx, `INSERT INTO reseller_customers(user_id) VALUES($1)`, userID)
	return err
}
func (r *resellerRepository) CustomerOwned(ctx context.Context, ownerID, customerID int64) (bool, error) {
	var ok bool
	err := scanSingleRow(ctx, r.executor(ctx), `SELECT EXISTS(SELECT 1 FROM reseller_customers c JOIN user_affiliates a ON a.user_id=c.user_id JOIN users u ON u.id=c.user_id WHERE c.user_id=$2 AND a.inviter_id=$1 AND u.deleted_at IS NULL)`, []any{ownerID, customerID}, &ok)
	return ok, err
}
func (r *resellerRepository) Customers(ctx context.Context, ownerID int64, search string, page, size int) ([]service.ResellerCustomer, int64, error) {
	q := r.executor(ctx)
	pattern := "%" + strings.TrimSpace(search) + "%"
	where := ` FROM reseller_customers c JOIN user_affiliates a ON a.user_id=c.user_id JOIN users u ON u.id=c.user_id WHERE a.inviter_id=$1 AND u.deleted_at IS NULL AND (u.email ILIKE $2 OR u.username ILIKE $2 OR c.notes ILIKE $2)`
	var total int64
	if err := scanSingleRow(ctx, q, `SELECT COUNT(*)`+where, []any{ownerID, pattern}, &total); err != nil {
		return nil, 0, err
	}
	rows, err := q.QueryContext(ctx, `SELECT c.user_id,u.username,u.email,u.status,c.notes,c.created_at,COALESCE((SELECT SUM(charged_amount) FROM reseller_earnings e WHERE e.owner_user_id=$1 AND e.customer_user_id=c.user_id),0)::float8,COALESCE((SELECT SUM(profit_amount) FROM reseller_earnings e WHERE e.owner_user_id=$1 AND e.customer_user_id=c.user_id),0)::float8,(SELECT MAX(created_at) FROM reseller_earnings e WHERE e.owner_user_id=$1 AND e.customer_user_id=c.user_id)`+where+` ORDER BY c.created_at DESC,c.user_id DESC LIMIT $3 OFFSET $4`, ownerID, pattern, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []service.ResellerCustomer{}
	for rows.Next() {
		var c service.ResellerCustomer
		if err = rows.Scan(&c.UserID, &c.Username, &c.Email, &c.Status, &c.Notes, &c.CreatedAt, &c.Charged, &c.Profit, &c.LastUsageAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}
func (r *resellerRepository) UpdateNotes(ctx context.Context, ownerID, customerID int64, notes string) error {
	result, err := r.executor(ctx).ExecContext(ctx, `UPDATE reseller_customers c SET notes=$3 FROM user_affiliates a WHERE c.user_id=$2 AND a.user_id=c.user_id AND a.inviter_id=$1`, ownerID, customerID, strings.TrimSpace(notes))
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return infraerrors.NotFound("RESELLER_CUSTOMER_NOT_FOUND", "客户不属于当前站长")
	}
	return nil
}
func (r *resellerRepository) Prices(ctx context.Context, ownerID int64) ([]service.ResellerPrice, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT customer_user_id,group_id,multiplier::float8 FROM reseller_prices WHERE owner_user_id=$1 ORDER BY customer_user_id NULLS FIRST,group_id NULLS FIRST`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.ResellerPrice{}
	for rows.Next() {
		var p service.ResellerPrice
		if err = rows.Scan(&p.CustomerID, &p.GroupID, &p.Multiplier); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *resellerRepository) SetPrices(ctx context.Context, ownerID int64, customerID *int64, overall *float64, prices []service.ResellerPrice) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var enabled bool
	if err = tx.QueryRowContext(ctx, `SELECT enabled FROM reseller_profiles WHERE user_id=$1 FOR UPDATE`, ownerID).Scan(&enabled); err != nil {
		return err
	}
	if !enabled {
		return infraerrors.Forbidden("RESELLER_DISABLED", "站长中心未开放")
	}
	if customerID != nil {
		var id int64
		if err = tx.QueryRowContext(ctx, `SELECT a.user_id FROM user_affiliates a JOIN reseller_customers c ON c.user_id=a.user_id WHERE a.user_id=$2 AND a.inviter_id=$1 FOR SHARE OF a`, ownerID, *customerID).Scan(&id); err != nil {
			return infraerrors.NotFound("RESELLER_CUSTOMER_NOT_FOUND", "客户不属于当前站长")
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM reseller_prices WHERE owner_user_id=$1 AND customer_user_id IS NOT DISTINCT FROM $2::bigint AND group_id IS NULL`, ownerID, customerID); err != nil {
		return err
	}
	if customerID == nil {
		if overall == nil {
			return infraerrors.BadRequest("INVALID_RESELLER_MULTIPLIER", "默认倍率不能为空")
		}
		if _, err = tx.ExecContext(ctx, `UPDATE reseller_profiles SET default_multiplier=$2,updated_at=NOW() WHERE user_id=$1`, ownerID, *overall); err != nil {
			return err
		}
	} else if overall != nil {
		if _, err = tx.ExecContext(ctx, `INSERT INTO reseller_prices(owner_user_id,customer_user_id,multiplier) VALUES($1,$2,$3)`, ownerID, *customerID, *overall); err != nil {
			return err
		}
	}
	for _, p := range prices {
		if _, err = tx.ExecContext(ctx, `DELETE FROM reseller_prices WHERE owner_user_id=$1 AND customer_user_id IS NOT DISTINCT FROM $2::bigint AND group_id=$3`, ownerID, customerID, p.GroupID); err != nil {
			return err
		}
		if p.Multiplier == nil {
			continue
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO reseller_prices(owner_user_id,customer_user_id,group_id,multiplier) VALUES($1,$2,$3,$4)`, ownerID, customerID, p.GroupID, *p.Multiplier); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (r *resellerRepository) Summary(ctx context.Context, ownerID int64) (*service.ResellerSummary, error) {
	p := &service.ResellerSummary{}
	err := scanSingleRow(ctx, r.executor(ctx), `SELECT (SELECT COUNT(*) FROM reseller_customers c JOIN user_affiliates a ON a.user_id=c.user_id JOIN users u ON u.id=c.user_id WHERE a.inviter_id=$1 AND u.deleted_at IS NULL),COALESCE(SUM(charged_amount),0)::float8,COALESCE(SUM(profit_amount),0)::float8 FROM reseller_earnings WHERE owner_user_id=$1`, []any{ownerID}, &p.CustomerCount, &p.Charged, &p.Profit)
	return p, err
}
func (r *resellerRepository) Earnings(ctx context.Context, ownerID int64, page, size int) ([]service.ResellerEarning, int64, error) {
	q := r.executor(ctx)
	var count int64
	if err := scanSingleRow(ctx, q, `SELECT COUNT(*) FROM reseller_earnings WHERE owner_user_id=$1`, []any{ownerID}, &count); err != nil {
		return nil, 0, err
	}
	rows, err := q.QueryContext(ctx, `SELECT e.id,e.customer_user_id,u.username,e.group_id,g.name,e.model,e.charged_amount::float8,e.cost_amount::float8,e.profit_amount::float8,e.multiplier::float8,e.created_at FROM reseller_earnings e JOIN users u ON u.id=e.customer_user_id JOIN groups g ON g.id=e.group_id WHERE e.owner_user_id=$1 ORDER BY e.created_at DESC,e.id DESC LIMIT $2 OFFSET $3`, ownerID, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []service.ResellerEarning{}
	for rows.Next() {
		var e service.ResellerEarning
		if err = rows.Scan(&e.ID, &e.CustomerID, &e.Username, &e.GroupID, &e.GroupName, &e.Model, &e.Charged, &e.Cost, &e.Profit, &e.Multiplier, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, count, rows.Err()
}

type resellerPricingNode struct {
	customerID, ownerID int64
	defaultRate         float64
	prices              map[[2]int64]float64
}

func (n resellerPricingNode) rate(groupID int64) float64 {
	for _, key := range [][2]int64{{n.customerID, groupID}, {n.customerID, 0}, {0, groupID}} {
		if rate, ok := n.prices[key]; ok {
			return rate
		}
	}
	return n.defaultRate
}
func (r *resellerRepository) Pricing(ctx context.Context, userID int64) (map[int64]*reseller.Snapshot, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	chain := []resellerPricingNode{}
	current := userID
	for {
		var n resellerPricingNode
		n.customerID = current
		n.prices = map[[2]int64]float64{}
		err = tx.QueryRowContext(ctx, `SELECT p.user_id,p.default_multiplier::float8 FROM reseller_customers c JOIN user_affiliates a ON a.user_id=c.user_id JOIN reseller_profiles p ON p.user_id=a.inviter_id WHERE c.user_id=$1`, current).Scan(&n.ownerID, &n.defaultRate)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return nil, err
		}
		rows, e := tx.QueryContext(ctx, `SELECT COALESCE(customer_user_id,0),COALESCE(group_id,0),multiplier::float8 FROM reseller_prices WHERE owner_user_id=$1 AND (customer_user_id IS NULL OR customer_user_id=$2)`, n.ownerID, current)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var a, b int64
			var v float64
			if e = rows.Scan(&a, &b, &v); e != nil {
				rows.Close()
				return nil, e
			}
			n.prices[[2]int64{a, b}] = v
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		chain = append(chain, n)
		current = n.ownerID
	}
	out := map[int64]*reseller.Snapshot{}
	if len(chain) == 0 {
		return out, tx.Commit()
	}
	rows, err := tx.QueryContext(ctx, `SELECT g.id,COALESCE(u.rate_multiplier,g.rate_multiplier)::float8,g.image_rate_independent,g.image_rate_multiplier::float8,g.video_rate_independent,g.video_rate_multiplier::float8 FROM groups g LEFT JOIN user_group_rate_multipliers u ON u.group_id=g.id AND u.user_id=$1 WHERE g.deleted_at IS NULL`, current)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var groupID int64
		var text, image, video float64
		var imageIndependent, videoIndependent bool
		if err = rows.Scan(&groupID, &text, &imageIndependent, &image, &videoIndependent, &video); err != nil {
			rows.Close()
			return nil, err
		}
		if !imageIndependent {
			image = text
		}
		if !videoIndependent {
			video = text
		}
		for i := len(chain) - 1; i >= 0; i-- {
			m := chain[i].rate(groupID)
			text *= m
			image *= m
			video *= m
		}
		out[groupID] = &reseller.Snapshot{UserID: userID, OwnerID: chain[0].ownerID, GroupID: groupID, Multiplier: chain[0].rate(groupID), TextRate: text, ImageRate: image, VideoRate: video}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	return out, tx.Commit()
}

// Billing owns the transaction; the frozen snapshot owns attribution and rates.
func recordResellerEarning(ctx context.Context, tx *sql.Tx, snapshot *reseller.Snapshot, requestID string, keyID int64, model string, charged float64) error {
	if snapshot == nil || charged <= 0 {
		return nil
	}
	charged = service.QuantizeUsageBillingAmount(charged)
	cost := service.QuantizeUsageBillingAmount(charged / snapshot.Multiplier)
	profit := service.QuantizeUsageBillingAmount(charged - cost)
	if profit < 0 {
		return fmt.Errorf("reseller snapshot produces a negative margin")
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO reseller_earnings(owner_user_id,customer_user_id,group_id,request_id,api_key_id,model,charged_amount,cost_amount,profit_amount,multiplier) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(request_id,api_key_id) DO NOTHING`, snapshot.OwnerID, snapshot.UserID, snapshot.GroupID, requestID, keyID, model, charged, cost, profit, snapshot.Multiplier)
	return err
}
