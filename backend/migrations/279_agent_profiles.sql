-- LC-024: self-service agent enrollment gating invite rebates.
-- Cutoff and grandfathered backfill run at first boot (repository/agent_repo.go),
-- because the effective enrollment time is only knowable once the new build is
-- actually serving. This migration creates schema only.
CREATE TABLE IF NOT EXISTS agent_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    source VARCHAR(24) NOT NULL DEFAULT 'applied',
    applied_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ,
    reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT agent_profiles_status_check CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT agent_profiles_source_check CHECK (source IN ('applied', 'grandfathered'))
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_status ON agent_profiles(status);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_applied_at ON agent_profiles(applied_at DESC NULLS LAST);

COMMENT ON TABLE agent_profiles IS 'LC-024 代理中心：自助申请与审核状态；无行表示未申请';
COMMENT ON COLUMN agent_profiles.status IS 'pending 待审核 / approved 已通过 / rejected 已驳回；通过后不可退出';
COMMENT ON COLUMN agent_profiles.source IS 'applied 用户自助申请 / grandfathered 存量用户按 cutoff 一次性回填';
COMMENT ON COLUMN agent_profiles.applied_at IS '首次提交申请的时间';
COMMENT ON COLUMN agent_profiles.reviewed_by IS '审核管理员用户ID';
