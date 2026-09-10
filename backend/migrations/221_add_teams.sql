-- 单团队功能：团队、成员、邀请、所有权转让和计费归因字段。
-- 历史 usage_logs / batch_image_jobs 不做全表回填；旧行由读取层回退 user_id。
CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    member_limit INTEGER NOT NULL DEFAULT 10,
    default_daily_limit_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    default_weekly_limit_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    default_monthly_limit_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT teams_status_check CHECK (status IN ('active', 'suspended')),
    CONSTRAINT teams_member_limit_check CHECK (member_limit >= 0),
    CONSTRAINT teams_default_member_limits_check CHECK (
        default_daily_limit_usd >= 0
        AND default_weekly_limit_usd >= 0
        AND default_monthly_limit_usd >= 0
    )
);

CREATE TABLE IF NOT EXISTS team_memberships (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    daily_limit_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    weekly_limit_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    monthly_limit_usd DECIMAL(20,8) NOT NULL DEFAULT 0,
    daily_usage_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    weekly_usage_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    monthly_usage_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    daily_window_start TIMESTAMPTZ NULL,
    weekly_window_start TIMESTAMPTZ NULL,
    monthly_window_start TIMESTAMPTZ NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT team_memberships_role_check CHECK (role IN ('owner', 'member')),
    CONSTRAINT team_memberships_limits_check CHECK (
        daily_limit_usd >= 0 AND weekly_limit_usd >= 0 AND monthly_limit_usd >= 0
    ),
    CONSTRAINT team_memberships_usage_check CHECK (
        daily_usage_usd >= 0 AND weekly_usage_usd >= 0 AND monthly_usage_usd >= 0
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS team_memberships_active_user_uq
    ON team_memberships (user_id) WHERE left_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS team_memberships_active_team_user_uq
    ON team_memberships (team_id, user_id) WHERE left_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS team_memberships_active_owner_uq
    ON team_memberships (team_id) WHERE left_at IS NULL AND role = 'owner';
CREATE INDEX IF NOT EXISTS team_memberships_team_idx
    ON team_memberships (team_id) WHERE left_at IS NULL;

CREATE TABLE IF NOT EXISTS team_invitations (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    inviter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_by_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    accepted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT team_invitations_status_check CHECK (
        status IN ('pending', 'accepted', 'declined', 'revoked', 'expired')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS team_invitations_token_hash_uq ON team_invitations (token_hash);
CREATE UNIQUE INDEX IF NOT EXISTS team_invitations_pending_email_uq
    ON team_invitations (team_id, email) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS team_invitations_email_status_idx
    ON team_invitations (email, status, expires_at);

CREATE TABLE IF NOT EXISTS team_ownership_transfers (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    from_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT team_ownership_transfers_status_check CHECK (
        status IN ('pending', 'accepted', 'declined', 'cancelled', 'expired')
    ),
    CONSTRAINT team_ownership_transfers_distinct_users_check CHECK (from_user_id <> to_user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS team_ownership_transfers_token_hash_uq
    ON team_ownership_transfers (token_hash);
CREATE UNIQUE INDEX IF NOT EXISTS team_ownership_transfers_pending_team_uq
    ON team_ownership_transfers (team_id) WHERE status = 'pending';

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS team_id BIGINT NULL;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS team_owner_disabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS billing_user_id BIGINT NULL;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS team_id BIGINT NULL;
ALTER TABLE batch_image_jobs ADD COLUMN IF NOT EXISTS billing_user_id BIGINT NULL;
ALTER TABLE batch_image_jobs ADD COLUMN IF NOT EXISTS team_id BIGINT NULL;
ALTER TABLE batch_image_jobs ADD COLUMN IF NOT EXISTS allowance_reserved BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'api_keys_team_id_fkey') THEN
        ALTER TABLE api_keys ADD CONSTRAINT api_keys_team_id_fkey
            FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'usage_logs_billing_user_id_fkey') THEN
        ALTER TABLE usage_logs ADD CONSTRAINT usage_logs_billing_user_id_fkey
            FOREIGN KEY (billing_user_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'usage_logs_team_id_fkey') THEN
        ALTER TABLE usage_logs ADD CONSTRAINT usage_logs_team_id_fkey
            FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'batch_image_jobs_billing_user_id_fkey') THEN
        ALTER TABLE batch_image_jobs ADD CONSTRAINT batch_image_jobs_billing_user_id_fkey
            FOREIGN KEY (billing_user_id) REFERENCES users(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'batch_image_jobs_team_id_fkey') THEN
        ALTER TABLE batch_image_jobs ADD CONSTRAINT batch_image_jobs_team_id_fkey
            FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- 兼容仍未显式写入 billing_user_id 的旧 SQL 路径；新路径应直接写入完整归因。
CREATE OR REPLACE FUNCTION fill_usage_log_team_attribution()
RETURNS TRIGGER AS $$
DECLARE
    resolved_team_id BIGINT;
    resolved_billing_user_id BIGINT;
BEGIN
    IF NEW.billing_user_id IS NOT NULL THEN
        RETURN NEW;
    END IF;
    SELECT ak.team_id INTO resolved_team_id FROM api_keys ak WHERE ak.id = NEW.api_key_id;
    NEW.team_id := COALESCE(NEW.team_id, resolved_team_id);
    IF NEW.team_id IS NOT NULL THEN
        SELECT tm.user_id INTO resolved_billing_user_id
        FROM team_memberships tm
        WHERE tm.team_id = NEW.team_id AND tm.left_at IS NULL AND tm.role = 'owner'
        LIMIT 1;
    END IF;
    NEW.billing_user_id := COALESCE(resolved_billing_user_id, NEW.user_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS usage_logs_fill_team_attribution ON usage_logs;
CREATE TRIGGER usage_logs_fill_team_attribution
    BEFORE INSERT ON usage_logs
    FOR EACH ROW EXECUTE FUNCTION fill_usage_log_team_attribution();
