-- +goose Up
CREATE TABLE creator_profiles (
    user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name text NOT NULL CHECK (char_length(btrim(display_name)) BETWEEN 1 AND 80),
    description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 5000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE profile_images (
    user_id bigint NOT NULL REFERENCES creator_profiles(user_id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('avatar', 'cover')),
    object_key text NOT NULL UNIQUE,
    content_type text NOT NULL CHECK (content_type IN ('image/png','image/jpeg')),
    size_bytes bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 5242880),
    width integer NOT NULL CHECK (width BETWEEN 1 AND 8192),
    height integer NOT NULL CHECK (height BETWEEN 1 AND 8192),
    sha256 text NOT NULL CHECK (sha256 ~ '^[a-f0-9]{64}$'),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, kind)
);
CREATE TABLE subscription_tiers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id bigint NOT NULL REFERENCES creator_profiles(user_id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 80),
    description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 5000),
    price_kopecks bigint NOT NULL CHECK (price_kopecks BETWEEN 0 AND 100000000),
    position integer NOT NULL CHECK (position > 0),
    used_at timestamptz,
    archived_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (author_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX subscription_tiers_author_idx ON subscription_tiers(author_id, position);
-- A reference from a publication/subscription must mark used_at under the author lock.
-- Preserve history even when writes originate outside the HTTP API.
-- +goose StatementBegin
CREATE FUNCTION protect_used_tier() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.used_at IS NOT NULL THEN
            RAISE EXCEPTION 'used tier must be archived' USING ERRCODE = '23514';
        END IF;
        RETURN OLD;
    END IF;
    IF NEW.author_id <> OLD.author_id OR (OLD.used_at IS NOT NULL AND
       (NEW.position <> OLD.position OR NEW.used_at IS DISTINCT FROM OLD.used_at)) OR
       (OLD.archived_at IS NOT NULL AND NEW.archived_at IS DISTINCT FROM OLD.archived_at) THEN
        RAISE EXCEPTION 'tier history is immutable' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER protect_used_tier BEFORE UPDATE OR DELETE ON subscription_tiers
    FOR EACH ROW EXECUTE FUNCTION protect_used_tier();

-- +goose Down
DROP TABLE subscription_tiers;
DROP FUNCTION protect_used_tier();
DROP TABLE profile_images, creator_profiles;
