-- LC-018: one invitation graph shared by AFF and reusable registration codes.
ALTER TABLE reusable_invitation_codes
 ADD COLUMN owner_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;
COMMENT ON COLUMN reusable_invitation_codes.owner_user_id IS '返佣归属用户；后台指定时只补入未绑定的历史注册用户，不补发历史充值';

ALTER TABLE user_affiliates ADD COLUMN invitation_code TEXT NULL;

ALTER TABLE user_affiliate_ledger
 ADD COLUMN rebate_source VARCHAR(32) NULL,
 ADD COLUMN rebate_level INTEGER NULL,
 ADD COLUMN rebate_rate_percent DECIMAL(12,8) NULL,
 ADD COLUMN rebate_base_amount DECIMAL(20,8) NULL;
-- Historical entries retain NULL snapshots; do not infer old rates from current settings.
CREATE UNIQUE INDEX idx_user_affiliate_ledger_order_recipient
 ON user_affiliate_ledger(source_order_id, user_id)
 WHERE action = 'accrue' AND source_order_id IS NOT NULL;
