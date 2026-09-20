CREATE TABLE IF NOT EXISTS openai_video_tasks (
    id BIGSERIAL PRIMARY KEY,
    local_request_id VARCHAR(128) NOT NULL,
    task_id VARCHAR(255),
    actor_user_id BIGINT NOT NULL,
    billing_user_id BIGINT NOT NULL,
    team_id BIGINT,
    api_key_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    channel_id BIGINT,
    account_id BIGINT NOT NULL,
    subscription_id BIGINT,
    requested_model VARCHAR(128) NOT NULL,
    upstream_model VARCHAR(128) NOT NULL,
    request_seconds INTEGER NOT NULL CHECK (request_seconds > 0),
    resolution VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'creating',
    upstream_status VARCHAR(64),
    billing_type SMALLINT NOT NULL,
    billing_status VARCHAR(32) NOT NULL DEFAULT 'none',
    total_cost NUMERIC(20,10) NOT NULL DEFAULT 0,
    actual_cost NUMERIC(20,10),
    hold_amount NUMERIC(20,10) NOT NULL DEFAULT 0,
    group_rate_multiplier NUMERIC(20,10) NOT NULL DEFAULT 1,
    account_rate_multiplier NUMERIC(20,10) NOT NULL DEFAULT 1,
    allowance_reserved BOOLEAN NOT NULL DEFAULT FALSE,
    request_payload_hash VARCHAR(64) NOT NULL,
    inbound_endpoint VARCHAR(255) NOT NULL,
    upstream_endpoint VARCHAR(255) NOT NULL,
    model_mapping_chain VARCHAR(512),
    user_agent TEXT,
    ip_address VARCHAR(64),
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_poll_at TIMESTAMPTZ,
    lease_until TIMESTAMPTZ,
    lease_token VARCHAR(128),
    last_error_code VARCHAR(128),
    last_error_message TEXT,
    usage_recorded BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    settled_at TIMESTAMPTZ,
    usage_recorded_at TIMESTAMPTZ,
    CONSTRAINT uq_openai_video_tasks_local_request_id UNIQUE (local_request_id),
    CONSTRAINT uq_openai_video_tasks_task_id UNIQUE (task_id)
);

CREATE INDEX IF NOT EXISTS idx_openai_video_tasks_due
    ON openai_video_tasks (next_poll_at, id)
    WHERE status IN ('creating', 'pending', 'processing') OR (status = 'completed' AND usage_recorded = FALSE);

CREATE INDEX IF NOT EXISTS idx_openai_video_tasks_api_key_created
    ON openai_video_tasks (api_key_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_openai_video_tasks_billing_status
    ON openai_video_tasks (billing_status, updated_at);
