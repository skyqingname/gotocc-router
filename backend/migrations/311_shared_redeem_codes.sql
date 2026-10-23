-- Redeem codes may be shared by several users up to an admin-set limit, like
-- promo codes. Existing codes stay single-use; 0 means unlimited.
ALTER TABLE redeem_codes
    ADD COLUMN IF NOT EXISTS max_uses INT NOT NULL DEFAULT 1 CHECK (max_uses >= 0),
    ADD COLUMN IF NOT EXISTS used_count INT NOT NULL DEFAULT 0 CHECK (used_count >= 0);

-- One row per redemption of a shared code; each user redeems a code once.
-- Single-use codes keep recording their redeemer in redeem_codes.used_by.
CREATE TABLE IF NOT EXISTS redeem_code_usages (
    id BIGSERIAL PRIMARY KEY,
    redeem_code_id BIGINT NOT NULL REFERENCES redeem_codes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (redeem_code_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_redeem_code_usages_user_id ON redeem_code_usages(user_id);

COMMENT ON COLUMN redeem_codes.max_uses IS 'Redemption limit: 1 single-use, 0 unlimited';
COMMENT ON COLUMN redeem_codes.used_count IS 'Redemptions of a shared code recorded in redeem_code_usages';
COMMENT ON TABLE redeem_code_usages IS 'Redemptions of shared redeem codes';
