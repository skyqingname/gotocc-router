CREATE TABLE IF NOT EXISTS image_objects (
    id BIGSERIAL PRIMARY KEY,
    object_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    storage_key VARCHAR(1024) NOT NULL,
    content_type VARCHAR(128) NOT NULL DEFAULT 'image/png',
    byte_size BIGINT NOT NULL CHECK (byte_size >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS image_objects_object_id_key ON image_objects (object_id);
CREATE UNIQUE INDEX IF NOT EXISTS image_objects_storage_key_key ON image_objects (storage_key);
CREATE INDEX IF NOT EXISTS image_objects_user_id_created_at_idx ON image_objects (user_id, created_at);
CREATE INDEX IF NOT EXISTS image_objects_task_id_idx ON image_objects (task_id);
CREATE INDEX IF NOT EXISTS image_objects_api_key_id_created_at_idx ON image_objects (api_key_id, created_at);
