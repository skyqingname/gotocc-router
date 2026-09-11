ALTER TABLE openai_video_tasks
    ADD COLUMN billing_mode VARCHAR(32) NOT NULL DEFAULT 'video';

ALTER TABLE openai_video_tasks
    ALTER COLUMN billing_mode DROP DEFAULT;

ALTER TABLE openai_video_tasks
    ADD CONSTRAINT chk_openai_video_tasks_billing_mode
    CHECK (billing_mode IN ('per_request', 'video'));
