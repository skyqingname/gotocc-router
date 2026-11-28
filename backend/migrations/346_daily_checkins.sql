-- Daily check-in. One row per user and site-local calendar day; the unique
-- pair makes a repeated check-in on the same day a no-op. The reward is added
-- to users.balance in the same transaction that inserts the row.
CREATE TABLE IF NOT EXISTS daily_checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    daily_reward DECIMAL(20,8) NOT NULL,
    milestone_days INT NOT NULL DEFAULT 0,
    milestone_reward DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT daily_checkins_user_date_unique UNIQUE (user_id, checkin_date)
);

COMMENT ON TABLE daily_checkins IS 'Daily check-in rewards credited to the user balance';
COMMENT ON COLUMN daily_checkins.checkin_date IS 'Calendar day in the site timezone';
COMMENT ON COLUMN daily_checkins.milestone_days IS 'Monthly milestone reached by this check-in; 0 when none';
