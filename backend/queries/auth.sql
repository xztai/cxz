-- name: CreateUser :one
INSERT INTO users (email, name, nickname, password_hash, email_verified, is_admin)
VALUES (sqlc.arg(email), sqlc.arg(nickname), sqlc.arg(nickname), sqlc.arg(password_hash), sqlc.arg(email_verified), sqlc.arg(is_admin)) RETURNING *;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: LockUserByEmail :one
SELECT * FROM users WHERE email = $1 FOR UPDATE;

-- name: LockUser :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- name: VerifyEmail :exec
UPDATE users SET email_verified = true WHERE id = $1;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: UpdateUnverifiedEmail :one
UPDATE users SET email = $2 WHERE id = $1 AND NOT email_verified RETURNING *;

-- name: PutActionToken :exec
INSERT INTO action_tokens (token_hash, user_id, purpose, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, purpose) DO UPDATE SET token_hash = EXCLUDED.token_hash, expires_at = EXCLUDED.expires_at;

-- name: FindActionToken :one
SELECT * FROM action_tokens WHERE token_hash = $1 AND purpose = $2 AND expires_at > now();

-- name: VerificationEmail :one
SELECT users.email FROM action_tokens JOIN users ON users.id = action_tokens.user_id
WHERE action_tokens.token_hash = $1 AND action_tokens.purpose = 'verify'
AND action_tokens.expires_at > now() AND NOT users.blocked AND NOT users.email_verified;

-- name: ConsumeActionToken :one
DELETE FROM action_tokens WHERE token_hash = $1 AND purpose = $2 AND expires_at > now() RETURNING user_id;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: SessionUser :one
SELECT users.* FROM users JOIN sessions ON sessions.user_id = users.id
WHERE sessions.token_hash = $1 AND sessions.expires_at > now() AND NOT users.blocked AND users.email_verified;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: RevokeSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: RateLimit :one
INSERT INTO rate_limits (key, hits, expires_at) VALUES ($1, 1, now() + interval '15 minutes')
ON CONFLICT (key) DO UPDATE SET
hits = CASE WHEN rate_limits.expires_at <= now() THEN 1 ELSE rate_limits.hits + 1 END,
expires_at = CASE WHEN rate_limits.expires_at <= now() THEN now() + interval '15 minutes' ELSE rate_limits.expires_at END
RETURNING hits;

-- name: QueueMail :exec
INSERT INTO mail_outbox (recipient, subject, body, token_hash, expires_at) VALUES ($1, $2, $3, $4, $5);

-- name: LockMail :one
SELECT * FROM mail_outbox WHERE available_at <= now() AND failed_at IS NULL ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1;

-- name: DeleteMail :exec
DELETE FROM mail_outbox WHERE id = $1;

-- name: RetryMail :exec
UPDATE mail_outbox SET attempts = attempts + 1,
 available_at = now() + interval '1 minute' * power(2, attempts),
 failed_at = CASE WHEN attempts >= 7 THEN now() ELSE NULL END,
 failure_code = 'smtp' WHERE id = $1;

-- name: CleanupSessions :exec
DELETE FROM sessions WHERE expires_at <= now();

-- name: CleanupTokens :exec
DELETE FROM action_tokens WHERE expires_at <= now();

-- name: CleanupRateLimits :exec
DELETE FROM rate_limits WHERE expires_at <= now();

-- name: CleanupMail :exec
DELETE FROM mail_outbox WHERE expires_at <= now() OR (kind='action' AND NOT EXISTS
 (SELECT 1 FROM action_tokens WHERE token_hash = mail_outbox.token_hash AND expires_at > now()));

-- name: MailTokenActive :one
SELECT EXISTS(SELECT 1 FROM action_tokens JOIN users ON users.id = action_tokens.user_id
 WHERE token_hash = $1 AND expires_at > now() AND NOT users.blocked
 AND ((purpose = 'verify' AND NOT users.email_verified) OR (purpose = 'reset' AND users.email_verified)));

-- name: FailMail :exec
UPDATE mail_outbox SET failed_at = now(), failure_code = 'decrypt' WHERE id = $1;

-- name: RequeueMail :execrows
UPDATE mail_outbox SET attempts = 0, failed_at = NULL, failure_code = '', available_at = now()
WHERE mail_outbox.id = $1 AND failed_at IS NOT NULL AND expires_at > now()
AND ((mail_outbox.kind='billing' AND EXISTS (SELECT 1 FROM users WHERE id=mail_outbox.recipient_user_id AND NOT blocked AND email_verified)) OR EXISTS (SELECT 1 FROM action_tokens JOIN users ON users.id = action_tokens.user_id
 WHERE token_hash = mail_outbox.token_hash AND action_tokens.expires_at > now() AND NOT users.blocked
 AND ((purpose = 'verify' AND NOT users.email_verified) OR (purpose = 'reset' AND users.email_verified))));

-- name: QueueStats :one
SELECT count(*) FILTER (WHERE failed_at IS NULL) AS pending,
 count(*) FILTER (WHERE failed_at IS NOT NULL) AS failed,
 coalesce(extract(epoch FROM now() - min(created_at) FILTER (WHERE failed_at IS NULL)), 0)::bigint AS oldest_seconds
FROM mail_outbox WHERE expires_at > now();

-- name: FailedLoginHits :one
SELECT coalesce((SELECT hits FROM rate_limits WHERE key=$1 AND expires_at > now()), 0)::integer;

-- name: ClearRateLimit :exec
DELETE FROM rate_limits WHERE key=$1;
