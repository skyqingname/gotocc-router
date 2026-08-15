CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_team_id_active
    ON api_keys (team_id) WHERE team_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_billing_user_created
    ON usage_logs (billing_user_id, created_at DESC) WHERE billing_user_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_team_created
    ON usage_logs (team_id, created_at DESC) WHERE team_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_batch_image_jobs_billing_user_created
    ON batch_image_jobs (billing_user_id, created_at DESC) WHERE billing_user_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_batch_image_jobs_team_created
    ON batch_image_jobs (team_id, created_at DESC) WHERE team_id IS NOT NULL;
