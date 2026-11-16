-- A model name is a client-facing alias and can be mapped across OpenAI/Grok
-- groups. Migration 338's prefix inference cannot prove the original platform.
-- Clear only pre-338 records; submissions after that migration explicitly
-- record their resolved target platform and retain that evidence.
UPDATE async_image_tasks
SET platform = ''
WHERE created_at < (
    SELECT applied_at FROM schema_migrations
    WHERE filename = '338_async_image_task_user_history.sql'
)
AND platform <> '';
