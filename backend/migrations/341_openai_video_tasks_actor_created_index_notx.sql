-- Task center lists a user's own video tasks across every API key, newest first.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_openai_video_tasks_actor_created
    ON openai_video_tasks (actor_user_id, created_at DESC, id DESC);
