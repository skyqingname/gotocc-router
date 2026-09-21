CREATE TABLE reseller_profiles (
 user_id BIGINT PRIMARY KEY REFERENCES users(id),
 enabled BOOLEAN NOT NULL DEFAULT FALSE,
 invitation_code VARCHAR(64) NOT NULL UNIQUE,
 default_multiplier NUMERIC(20,8) NOT NULL DEFAULT 1,
 rebate_rates JSONB NOT NULL DEFAULT '[null,null,null]'::jsonb,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Ownership remains the existing unique user_affiliates.inviter_id relation.
-- This table marks registrations through the reseller entry; old AFF customers
-- are not silently enrolled or repriced.
CREATE TABLE reseller_customers (
 user_id BIGINT PRIMARY KEY REFERENCES users(id),
 notes TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE reseller_prices (
 id BIGSERIAL PRIMARY KEY,
 owner_user_id BIGINT NOT NULL REFERENCES users(id),
 customer_user_id BIGINT REFERENCES users(id),
 group_id BIGINT REFERENCES groups(id),
 multiplier NUMERIC(20,8) NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX reseller_prices_scope_idx ON reseller_prices(owner_user_id, COALESCE(customer_user_id,0), COALESCE(group_id,0));
CREATE TABLE reseller_earnings (
 id BIGSERIAL PRIMARY KEY,
 owner_user_id BIGINT NOT NULL REFERENCES users(id),
 customer_user_id BIGINT NOT NULL REFERENCES users(id),
 group_id BIGINT NOT NULL REFERENCES groups(id),
 request_id VARCHAR(255) NOT NULL,
 api_key_id BIGINT NOT NULL DEFAULT 0,
 model VARCHAR(255) NOT NULL DEFAULT '',
 charged_amount NUMERIC(20,8) NOT NULL,
 cost_amount NUMERIC(20,8) NOT NULL,
 profit_amount NUMERIC(20,8) NOT NULL,
 multiplier NUMERIC(20,8) NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(request_id, api_key_id)
);
CREATE INDEX reseller_earnings_owner_time_idx ON reseller_earnings(owner_user_id, created_at DESC, id DESC);
CREATE INDEX reseller_earnings_customer_idx ON reseller_earnings(customer_user_id, created_at DESC);
ALTER TABLE openai_video_tasks ADD COLUMN reseller_snapshot JSONB;
ALTER TABLE batch_image_jobs ADD COLUMN reseller_snapshot JSONB;
