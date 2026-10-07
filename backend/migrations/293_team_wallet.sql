-- LC-008: existing personal balances remain personal. Team wallets start empty.
ALTER TABLE teams
    ADD COLUMN balance NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN frozen_balance NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN reseller_owner_id BIGINT REFERENCES users(id);

UPDATE teams t SET reseller_owner_id = c.owner_user_id
FROM team_memberships m JOIN reseller_customers c ON c.user_id = m.user_id
WHERE m.team_id = t.id AND m.role = 'owner' AND m.left_at IS NULL;

CREATE TABLE team_wallet_entries (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id),
    operation_id VARCHAR(160) NOT NULL,
    kind VARCHAR(20) NOT NULL,
    actor_user_id BIGINT NOT NULL REFERENCES users(id),
    amount NUMERIC(20,8) NOT NULL,
    frozen_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    balance_after NUMERIC(20,8) NOT NULL,
    frozen_after NUMERIC(20,8) NOT NULL,
    personal_balance_after NUMERIC(20,8),
    api_key_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, operation_id, api_key_id)
);
CREATE INDEX team_wallet_entries_team_created_idx ON team_wallet_entries(team_id, created_at DESC);

-- Old tasks retain their original personal/customer holds. New tasks opt in explicitly.
ALTER TABLE openai_video_tasks ADD COLUMN team_wallet BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE batch_image_jobs ADD COLUMN team_wallet BOOLEAN NOT NULL DEFAULT FALSE;
