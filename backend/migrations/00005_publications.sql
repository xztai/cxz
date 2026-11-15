-- +goose Up
ALTER TABLE subscription_tiers ADD CONSTRAINT tiers_id_author_unique UNIQUE (id, author_id);
CREATE TABLE posts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id bigint NOT NULL REFERENCES creator_profiles(user_id),
    title text NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 200),
    teaser text NOT NULL DEFAULT '' CHECK (char_length(teaser) <= 2000),
    body_markdown text NOT NULL DEFAULT '' CHECK (char_length(body_markdown) <= 100000),
    minimum_tier_id bigint,
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, author_id),
    FOREIGN KEY (minimum_tier_id, author_id) REFERENCES subscription_tiers(id, author_id)
);
CREATE INDEX posts_author_idx ON posts(author_id, id DESC);
-- No subscription mutation endpoint exists before payments. Integration fixtures
-- insert grants here; a future payment service must use the same access predicate.
CREATE TABLE subscriptions (
    subscriber_id bigint NOT NULL REFERENCES users(id),
    author_id bigint NOT NULL REFERENCES creator_profiles(user_id),
    tier_id bigint NOT NULL,
    starts_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    PRIMARY KEY (subscriber_id, author_id),
    CHECK (expires_at > starts_at),
    FOREIGN KEY (tier_id, author_id) REFERENCES subscription_tiers(id, author_id)
);
-- Tier usage is permanent, including references created outside HTTP handlers.
-- +goose StatementBegin
CREATE FUNCTION mark_referenced_tier() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE tier bigint;
BEGIN
    IF TG_TABLE_NAME = 'posts' THEN tier := NEW.minimum_tier_id;
    ELSE tier := NEW.tier_id;
    END IF;
    IF tier IS NOT NULL THEN
        PERFORM 1 FROM creator_profiles WHERE user_id = NEW.author_id FOR UPDATE;
        UPDATE subscription_tiers SET used_at = COALESCE(used_at, now())
            WHERE id = tier AND author_id = NEW.author_id;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER posts_tier_usage BEFORE INSERT OR UPDATE OF minimum_tier_id ON posts
    FOR EACH ROW EXECUTE FUNCTION mark_referenced_tier();
CREATE TRIGGER subscriptions_tier_usage BEFORE INSERT OR UPDATE OF tier_id ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION mark_referenced_tier();

CREATE TABLE post_media (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id bigint NOT NULL REFERENCES creator_profiles(user_id),
    post_id bigint,
    kind text NOT NULL CHECK (kind IN ('image','audio','video','attachment')),
    filename text NOT NULL CHECK (char_length(filename) BETWEEN 1 AND 200),
    content_type text NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 104857600),
    object_key text NOT NULL UNIQUE,
    thumbnail_key text NOT NULL DEFAULT '',
    state text NOT NULL DEFAULT 'uploading' CHECK (state IN ('uploading','ready','deleting')),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (post_id, author_id) REFERENCES posts(id, author_id) ON DELETE SET NULL (post_id)
);
CREATE INDEX post_media_post_idx ON post_media(post_id, id);
CREATE INDEX post_media_cleanup_idx ON post_media(state, created_at);

-- +goose Down
DROP TABLE post_media, subscriptions, posts;
DROP FUNCTION mark_referenced_tier();
ALTER TABLE subscription_tiers DROP CONSTRAINT tiers_id_author_unique;
