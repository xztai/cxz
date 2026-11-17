-- +goose Up
-- Pending legacy plaintext mail is discarded; users can request a new link.
DELETE FROM mail_outbox;
ALTER TABLE mail_outbox
 ADD COLUMN token_hash text NOT NULL,
 ADD COLUMN expires_at timestamptz NOT NULL,
 ADD COLUMN failed_at timestamptz,
 ADD COLUMN failure_code text NOT NULL DEFAULT '';
CREATE INDEX mail_outbox_token_idx ON mail_outbox(token_hash);

-- +goose Down
-- Older workers cannot decrypt these envelopes. Never turn them into plaintext.
DELETE FROM mail_outbox;
ALTER TABLE mail_outbox DROP COLUMN token_hash, DROP COLUMN expires_at, DROP COLUMN failed_at, DROP COLUMN failure_code;
