CREATE TABLE IF NOT EXISTS reusable_invitation_codes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    max_uses INTEGER NOT NULL DEFAULT 0,
    used_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT reusable_invitation_codes_status_check CHECK (status IN ('active', 'disabled')),
    CONSTRAINT reusable_invitation_codes_max_uses_check CHECK (max_uses >= 0),
    CONSTRAINT reusable_invitation_codes_used_count_check CHECK (used_count >= 0)
);

CREATE INDEX IF NOT EXISTS reusable_invitation_codes_status_idx
    ON reusable_invitation_codes(status);

CREATE INDEX IF NOT EXISTS reusable_invitation_codes_expires_at_idx
    ON reusable_invitation_codes(expires_at);

CREATE TABLE IF NOT EXISTS reusable_invitation_code_uses (
    id BIGSERIAL PRIMARY KEY,
    code_id BIGINT NOT NULL REFERENCES reusable_invitation_codes(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    email VARCHAR(255) NOT NULL DEFAULT '',
    auth_source VARCHAR(50) NOT NULL DEFAULT '',
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS reusable_invitation_code_uses_code_id_used_at_idx
    ON reusable_invitation_code_uses(code_id, used_at DESC);

CREATE INDEX IF NOT EXISTS reusable_invitation_code_uses_user_id_idx
    ON reusable_invitation_code_uses(user_id);
