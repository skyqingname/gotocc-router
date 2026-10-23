-- Record the public entry host (gotocc.xyz, api.gotocc.xyz, customer domains)
-- that received each request. Historical rows stay unknown.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS service_host VARCHAR(255);
COMMENT ON COLUMN usage_logs.service_host IS 'Public entry host that received the request; NULL unrecorded';
