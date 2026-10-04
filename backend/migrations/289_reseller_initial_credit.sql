ALTER TABLE reseller_profiles
    ADD COLUMN initial_credit NUMERIC(20,8) NOT NULL DEFAULT 0;

ALTER TABLE reseller_credit_entries
    ADD COLUMN deduct_all BOOLEAN NOT NULL DEFAULT FALSE;
