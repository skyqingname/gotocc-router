-- LC-028: business ownership and customer credits are independent of affiliates
-- and platform wallets. Legacy balances and in-flight task snapshots stay intact.
ALTER TABLE reseller_customers
    ADD COLUMN owner_user_id BIGINT REFERENCES users(id),
    ADD COLUMN credit_balance NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN frozen_credit NUMERIC(20,8) NOT NULL DEFAULT 0;

UPDATE reseller_customers c SET owner_user_id = a.inviter_id
FROM user_affiliates a WHERE a.user_id = c.user_id;

ALTER TABLE reseller_customers ALTER COLUMN owner_user_id SET NOT NULL;
CREATE INDEX reseller_customers_owner_idx ON reseller_customers(owner_user_id, user_id);

CREATE TABLE reseller_credit_entries (
    id BIGSERIAL PRIMARY KEY,
    customer_user_id BIGINT NOT NULL REFERENCES users(id),
    owner_user_id BIGINT NOT NULL REFERENCES users(id),
    operator_user_id BIGINT REFERENCES users(id),
    actor_user_id BIGINT REFERENCES users(id),
    operation_id VARCHAR(255) NOT NULL,
    kind VARCHAR(32) NOT NULL,
    amount NUMERIC(20,8) NOT NULL,
    frozen_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    balance_after NUMERIC(20,8) NOT NULL,
    frozen_after NUMERIC(20,8) NOT NULL,
    platform_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    api_key_id BIGINT NOT NULL DEFAULT 0,
    model VARCHAR(255) NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(customer_user_id, operation_id, api_key_id)
);
CREATE INDEX reseller_credit_entries_customer_time_idx
    ON reseller_credit_entries(customer_user_id, created_at DESC, id DESC);
CREATE INDEX reseller_credit_entries_owner_time_idx
    ON reseller_credit_entries(owner_user_id, created_at DESC, id DESC);

ALTER TABLE reseller_earnings ADD COLUMN settlement_type VARCHAR(32) NOT NULL DEFAULT 'legacy_direct';
