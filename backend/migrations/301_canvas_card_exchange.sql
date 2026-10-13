-- LC-036: external Canvas cards are separate from GoToCC recharge codes.
CREATE TABLE canvas_card_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    operation_id UUID NOT NULL,
    denomination INTEGER NOT NULL,
    funding_source VARCHAR(16) NOT NULL,
    team_id BIGINT REFERENCES teams(id),
    balance_after NUMERIC(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, operation_id)
);
CREATE INDEX canvas_card_orders_user_created_idx ON canvas_card_orders(user_id, id DESC);

CREATE TABLE canvas_card_inventory (
    id BIGSERIAL PRIMARY KEY,
    denomination INTEGER NOT NULL,
    code TEXT NOT NULL UNIQUE,
    imported_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    issued_order_id BIGINT UNIQUE REFERENCES canvas_card_orders(id)
);
CREATE INDEX canvas_card_inventory_available_idx ON canvas_card_inventory(denomination, id)
    WHERE issued_order_id IS NULL;
