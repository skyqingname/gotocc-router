package service

import (
	"context"
	"time"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
)

// 每日签到：每天一次，奖励立即加到个人余额；本自然月累计签满里程碑天数的那一次签到
// 额外再得里程碑奖励。本月有过任意金额充值才开启，站长客户的额度由站长管理，不能签到。
// 规则来自根目录 daily-checkin.json（tools/generate_daily_checkin.py 生成
// daily_checkin_config_gotocc.go），日期与月份边界按站点时区。

const dailyCheckinDateLayout = "2006-01-02"
const dailyCheckinMonthLayout = "2006-01"

type DailyCheckinMilestone struct {
	Days   int
	Reward float64
}

type DailyCheckinConfig struct {
	DailyReward           float64
	Milestones            []DailyCheckinMilestone
	UnlockRedeemTypes     []string
	UnlockPaymentStatuses []string
}

var (
	ErrDailyCheckinUnavailable      = infraerrors.Forbidden("CHECKIN_UNAVAILABLE", "check-in is not available for reseller customer accounts")
	ErrDailyCheckinRechargeRequired = infraerrors.Forbidden("CHECKIN_RECHARGE_REQUIRED", "recharge any amount this month to unlock daily check-in")
	ErrDailyCheckinAlreadyDone      = infraerrors.Conflict("CHECKIN_ALREADY_DONE", "already checked in today")
	ErrDailyCheckinInvalidMonth     = infraerrors.BadRequest("CHECKIN_INVALID_MONTH", "month must be YYYY-MM and not later than the current month")
)

type DailyCheckinRecord struct {
	Date            string
	DailyReward     float64
	MilestoneDays   int
	MilestoneReward float64
}

type DailyCheckinRepository interface {
	ListRange(ctx context.Context, userID int64, from, until string) ([]DailyCheckinRecord, error)
	TotalReward(ctx context.Context, userID int64) (float64, error)
	HasRecharge(ctx context.Context, userID int64, since, until time.Time, redeemTypes, paymentStatuses []string) (bool, error)
	CountRange(ctx context.Context, userID int64, from, until string) (int, error)
	// Insert 在同一用户同一天已有记录时不写入，返回 false。
	Insert(ctx context.Context, userID int64, record DailyCheckinRecord) (bool, error)
}

type DailyCheckinDay struct {
	Date            string  `json:"date"`
	Reward          float64 `json:"reward"`
	MilestoneDays   int     `json:"milestone_days"`
	MilestoneReward float64 `json:"milestone_reward"`
}

type DailyCheckinMilestoneView struct {
	Days      int     `json:"days"`
	Reward    float64 `json:"reward"`
	Reached   bool    `json:"reached"`
	Remaining int     `json:"remaining"`
	Reachable bool    `json:"reachable"`
}

type DailyCheckinOverview struct {
	Timezone           string                      `json:"timezone"`
	Today              string                      `json:"today"`
	CurrentMonth       string                      `json:"current_month"`
	Month              string                      `json:"month"`
	Unlocked           bool                        `json:"unlocked"`
	LockReason         string                      `json:"lock_reason,omitempty"`
	CheckedInToday     bool                        `json:"checked_in_today"`
	DailyReward        float64                     `json:"daily_reward"`
	TodayReward        float64                     `json:"today_reward"`
	TodayMilestoneDays int                         `json:"today_milestone_days"`
	Days               []DailyCheckinDay           `json:"days"`
	MonthDays          int                         `json:"month_days"`
	MonthReward        float64                     `json:"month_reward"`
	TotalReward        float64                     `json:"total_reward"`
	Milestones         []DailyCheckinMilestoneView `json:"milestones"`
}

type DailyCheckinResult struct {
	Date            string  `json:"date"`
	Reward          float64 `json:"reward"`
	MilestoneDays   int     `json:"milestone_days"`
	MilestoneReward float64 `json:"milestone_reward"`
	Total           float64 `json:"total"`
	MonthDays       int     `json:"month_days"`
	Balance         float64 `json:"balance"`
}

type DailyCheckinService struct {
	repo         DailyCheckinRepository
	users        UserRepository
	resellers    ResellerRepository
	entClient    *dbent.Client
	billingCache *BillingCacheService
	authCache    APIKeyAuthCacheInvalidator
}

func NewDailyCheckinService(repo DailyCheckinRepository, users UserRepository, resellers ResellerRepository, entClient *dbent.Client, billingCache *BillingCacheService, authCache APIKeyAuthCacheInvalidator) *DailyCheckinService {
	return &DailyCheckinService{repo: repo, users: users, resellers: resellers, entClient: entClient, billingCache: billingCache, authCache: authCache}
}

// dailyCheckinCalendar 是站点时区里的今天和本月起止。
type dailyCheckinCalendar struct {
	today     time.Time
	month     time.Time
	nextMonth time.Time
}

func newDailyCheckinCalendar() dailyCheckinCalendar {
	now := timezone.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	month := timezone.StartOfMonth(now)
	return dailyCheckinCalendar{today: today, month: month, nextMonth: month.AddDate(0, 1, 0)}
}

func (s *DailyCheckinService) ensureEligible(ctx context.Context, userID int64) error {
	account, err := s.resellers.CustomerAccount(ctx, userID)
	if err != nil {
		return err
	}
	if account != nil {
		return ErrDailyCheckinUnavailable
	}
	return nil
}

func (s *DailyCheckinService) unlocked(ctx context.Context, userID int64, cal dailyCheckinCalendar) (bool, error) {
	return s.repo.HasRecharge(ctx, userID, cal.month, cal.nextMonth, dailyCheckinConfig.UnlockRedeemTypes, dailyCheckinConfig.UnlockPaymentStatuses)
}

// milestoneAt 返回本月第 days 次签到触发的里程碑。
func dailyCheckinMilestoneAt(days int) DailyCheckinMilestone {
	for _, milestone := range dailyCheckinConfig.Milestones {
		if milestone.Days == days {
			return milestone
		}
	}
	return DailyCheckinMilestone{}
}

func dailyCheckinDays(records []DailyCheckinRecord) ([]DailyCheckinDay, float64) {
	days := make([]DailyCheckinDay, 0, len(records))
	total := 0.0
	for _, record := range records {
		days = append(days, DailyCheckinDay{Date: record.Date, Reward: record.DailyReward + record.MilestoneReward, MilestoneDays: record.MilestoneDays, MilestoneReward: record.MilestoneReward})
		total += record.DailyReward + record.MilestoneReward
	}
	// 金额按余额的八位小数取整，避免浮点累加的尾数。
	return days, QuantizeUsageBillingAmount(total)
}

// Overview 返回 month（YYYY-MM，空为本月）的签到日历，以及本月是否开启、今天是否已签。
func (s *DailyCheckinService) Overview(ctx context.Context, userID int64, month string) (*DailyCheckinOverview, error) {
	if err := s.ensureEligible(ctx, userID); err != nil {
		return nil, err
	}
	cal := newDailyCheckinCalendar()
	viewed := cal.month
	if month != "" {
		parsed, err := time.ParseInLocation(dailyCheckinMonthLayout, month, cal.month.Location())
		if err != nil || parsed.After(cal.month) {
			return nil, ErrDailyCheckinInvalidMonth
		}
		viewed = parsed
	}
	viewedNext := viewed.AddDate(0, 1, 0)
	currentRecords, err := s.repo.ListRange(ctx, userID, cal.month.Format(dailyCheckinDateLayout), cal.nextMonth.Format(dailyCheckinDateLayout))
	if err != nil {
		return nil, err
	}
	viewedRecords := currentRecords
	if !viewed.Equal(cal.month) {
		if viewedRecords, err = s.repo.ListRange(ctx, userID, viewed.Format(dailyCheckinDateLayout), viewedNext.Format(dailyCheckinDateLayout)); err != nil {
			return nil, err
		}
	}
	total, err := s.repo.TotalReward(ctx, userID)
	if err != nil {
		return nil, err
	}
	unlocked, err := s.unlocked(ctx, userID, cal)
	if err != nil {
		return nil, err
	}

	todayKey := cal.today.Format(dailyCheckinDateLayout)
	checkedToday := false
	for _, record := range currentRecords {
		checkedToday = checkedToday || record.Date == todayKey
	}
	days, monthReward := dailyCheckinDays(viewedRecords)
	out := &DailyCheckinOverview{
		Timezone: timezone.Name(), Today: todayKey,
		CurrentMonth: cal.month.Format(dailyCheckinMonthLayout), Month: viewed.Format(dailyCheckinMonthLayout),
		Unlocked: unlocked, CheckedInToday: checkedToday, DailyReward: dailyCheckinConfig.DailyReward,
		Days: days, MonthDays: len(days), MonthReward: monthReward, TotalReward: total,
		Milestones: make([]DailyCheckinMilestoneView, 0, len(dailyCheckinConfig.Milestones)),
	}
	if !unlocked {
		out.LockReason = "recharge_required"
	}
	if !checkedToday {
		next := dailyCheckinMilestoneAt(len(currentRecords) + 1)
		out.TodayReward = QuantizeUsageBillingAmount(dailyCheckinConfig.DailyReward + next.Reward)
		out.TodayMilestoneDays = next.Days
	}
	// 本月剩下还能签的天数：从今天（今天已签则从明天）到月底。
	daysLeft := cal.nextMonth.AddDate(0, 0, -1).Day() - cal.today.Day() + 1
	if checkedToday {
		daysLeft--
	}
	for _, milestone := range dailyCheckinConfig.Milestones {
		view := DailyCheckinMilestoneView{Days: milestone.Days, Reward: milestone.Reward, Reached: out.MonthDays >= milestone.Days}
		view.Remaining = max(milestone.Days-out.MonthDays, 0)
		view.Reachable = view.Reached || (viewed.Equal(cal.month) && out.MonthDays+daysLeft >= milestone.Days)
		out.Milestones = append(out.Milestones, view)
	}
	return out, nil
}

// CheckIn 在一个事务里写下今天的签到并把奖励加到余额；同一天重复签到只成功一次。
func (s *DailyCheckinService) CheckIn(ctx context.Context, userID int64) (*DailyCheckinResult, error) {
	if err := s.ensureEligible(ctx, userID); err != nil {
		return nil, err
	}
	cal := newDailyCheckinCalendar()
	unlocked, err := s.unlocked(ctx, userID, cal)
	if err != nil {
		return nil, err
	}
	if !unlocked {
		return nil, ErrDailyCheckinRechargeRequired
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	// 同一天的并发签到在唯一键上排队，后到的那次不写入；不同日期不会并发，先数后写即可。
	signed, err := s.repo.CountRange(txCtx, userID, cal.month.Format(dailyCheckinDateLayout), cal.nextMonth.Format(dailyCheckinDateLayout))
	if err != nil {
		return nil, err
	}
	milestone := dailyCheckinMilestoneAt(signed + 1)
	record := DailyCheckinRecord{Date: cal.today.Format(dailyCheckinDateLayout), DailyReward: dailyCheckinConfig.DailyReward, MilestoneDays: milestone.Days, MilestoneReward: milestone.Reward}
	inserted, err := s.repo.Insert(txCtx, userID, record)
	if err != nil {
		return nil, err
	}
	if !inserted {
		return nil, ErrDailyCheckinAlreadyDone
	}
	total := QuantizeUsageBillingAmount(record.DailyReward + record.MilestoneReward)
	change, err := s.users.AdjustBalance(txCtx, userID, total)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.authCache.InvalidateAuthCacheByUserID(ctx, userID)
	if err := s.billingCache.InvalidateUserBalance(ctx, userID); err != nil {
		logger.LegacyPrintf("service.daily_checkin", "invalidate user balance cache failed: user_id=%d err=%v", userID, err)
	}
	return &DailyCheckinResult{
		Date: record.Date, Reward: record.DailyReward, MilestoneDays: record.MilestoneDays, MilestoneReward: record.MilestoneReward,
		Total: total, MonthDays: signed + 1, Balance: change.New,
	}, nil
}
