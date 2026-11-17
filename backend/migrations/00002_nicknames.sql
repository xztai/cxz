-- +goose Up
ALTER TABLE users ADD COLUMN nickname text;

-- Existing accounts retain their credentials, sessions and original display names.
-- IDs provide a unique URL-safe nickname without guessing a transliteration.
UPDATE users SET nickname = 'user-' || id::text;

ALTER TABLE users
    ALTER COLUMN nickname SET NOT NULL,
    ADD CONSTRAINT users_nickname_format CHECK (nickname ~ '^[a-z0-9_-]{3,30}$'),
    ADD CONSTRAINT users_nickname_key UNIQUE (nickname);

-- +goose Down
ALTER TABLE users DROP COLUMN nickname;
