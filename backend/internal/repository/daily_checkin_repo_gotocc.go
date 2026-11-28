package repository

import (
	"context"
	"time"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/lib/pq"
)

type dailyCheckinRepository struct {
	client *dbent.Client
}

func NewDailyCheckinRepository(client *dbent.Client) service.DailyCheckinRepository {
	return &dailyCheckinRepository{client: client}
}

func (r *dailyCheckinRepository) ListRange(ctx context.Context, userID int64, from, until string) ([]service.DailyCheckinRecord, error) {
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
		SELECT to_char(checkin_date, 'YYYY-MM-DD'), daily_reward::float8, milestone_days, milestone_reward::float8
		FROM daily_checkins
		WHERE user_id = $1 AND checkin_date >= $2::date AND checkin_date < $3::date
		ORDER BY checkin_date`, userID, from, until)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	records := []service.DailyCheckinRecord{}
	for rows.Next() {
		var record service.DailyCheckinRecord
		if err := rows.Scan(&record.Date, &record.DailyReward, &record.MilestoneDays, &record.MilestoneReward); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (r *dailyCheckinRepository) TotalReward(ctx context.Context, userID int64) (float64, error) {
	var total float64
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
		SELECT COALESCE(SUM(daily_reward + milestone_reward), 0)::float8 FROM daily_checkins WHERE user_id = $1`,
		[]any{userID}, &total)
	return total, err
}

func (r *dailyCheckinRepository) CountRange(ctx context.Context, userID int64, from, until string) (int, error) {
	var count int
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
		SELECT COUNT(*) FROM daily_checkins WHERE user_id = $1 AND checkin_date >= $2::date AND checkin_date < $3::date`,
		[]any{userID, from, until}, &count)
	return count, err
}

// HasRecharge 判断本月是否有任意金额的入账：单次兑换码与管理员加余额记在 redeem_codes.used_by，
// 共享兑换码的每次兑换记在 redeem_code_usages，在线支付看完成的订单。
func (r *dailyCheckinRepository) HasRecharge(ctx context.Context, userID int64, since, until time.Time, redeemTypes, paymentStatuses []string) (bool, error) {
	var found bool
	err := scanSingleRow(ctx, clientFromContext(ctx, r.client), `
		SELECT EXISTS (
			SELECT 1 FROM redeem_codes
			WHERE used_by = $1 AND type = ANY($4) AND value > 0 AND used_at >= $2 AND used_at < $3
		) OR EXISTS (
			SELECT 1 FROM redeem_code_usages u JOIN redeem_codes c ON c.id = u.redeem_code_id
			WHERE u.user_id = $1 AND c.type = ANY($4) AND c.value > 0 AND u.used_at >= $2 AND u.used_at < $3
		) OR EXISTS (
			SELECT 1 FROM payment_orders
			WHERE user_id = $1 AND status = ANY($5) AND amount > 0 AND completed_at >= $2 AND completed_at < $3
		)`,
		[]any{userID, since, until, pq.Array(redeemTypes), pq.Array(paymentStatuses)}, &found)
	return found, err
}

func (r *dailyCheckinRepository) Insert(ctx context.Context, userID int64, record service.DailyCheckinRecord) (bool, error) {
	result, err := clientFromContext(ctx, r.client).ExecContext(ctx, `
		INSERT INTO daily_checkins (user_id, checkin_date, daily_reward, milestone_days, milestone_reward)
		VALUES ($1, $2::date, $3, $4, $5)
		ON CONFLICT (user_id, checkin_date) DO NOTHING`,
		userID, record.Date, record.DailyReward, record.MilestoneDays, record.MilestoneReward)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}
