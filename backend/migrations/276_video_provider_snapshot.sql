ALTER TABLE openai_video_tasks ADD COLUMN provider_config JSONB;

-- Existing rows remain NULL and continue through the original OpenAI protocol.
-- New Video tasks preserve their channel model contract across admin edits.
