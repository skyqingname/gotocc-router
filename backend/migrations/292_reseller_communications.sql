ALTER TABLE reseller_profiles
    ADD COLUMN contact_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN contact_info TEXT NOT NULL DEFAULT '',
    ADD COLUMN announcements_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN sync_main_announcements BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE reseller_announcements (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NOT NULL REFERENCES reseller_profiles(user_id),
    source_announcement_id BIGINT REFERENCES announcements(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    notify_mode VARCHAR(20) NOT NULL DEFAULT 'silent',
    targeting JSONB NOT NULL DEFAULT '{}'::jsonb,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(owner_user_id, source_announcement_id)
);
CREATE INDEX reseller_announcements_owner_status_idx
    ON reseller_announcements(owner_user_id, status, id DESC);

CREATE TABLE reseller_announcement_reviews (
    owner_user_id BIGINT NOT NULL REFERENCES reseller_profiles(user_id),
    source_announcement_id BIGINT NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    source_updated_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL,
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(owner_user_id, source_announcement_id)
);

CREATE TABLE reseller_announcement_reads (
    announcement_id BIGINT NOT NULL REFERENCES reseller_announcements(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(announcement_id, user_id)
);
