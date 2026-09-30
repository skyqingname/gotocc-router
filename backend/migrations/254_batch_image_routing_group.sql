ALTER TABLE batch_image_jobs
    ADD COLUMN IF NOT EXISTS group_id BIGINT;

COMMENT ON COLUMN batch_image_jobs.group_id IS
    'Original routing group; NULL for legacy or unassigned fixed-key jobs. Retained after group deletion.';
