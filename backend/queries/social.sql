-- name: ListFeed :many
SELECT p.id,p.author_id,p.title,p.teaser,p.minimum_tier_id,p.published_at,u.nickname,cp.display_name,
 post_readable(p,sqlc.arg(viewer_id)::bigint)::boolean AS can_read
FROM posts p JOIN users u ON u.id=p.author_id JOIN creator_profiles cp ON cp.user_id=p.author_id
JOIN subscriptions sub ON sub.author_id=p.author_id AND sub.subscriber_id=sqlc.arg(viewer_id)
WHERE p.published_at IS NOT NULL AND NOT u.blocked AND u.email_verified
 AND sub.revoked_at IS NULL AND sub.starts_at<=now() AND sub.expires_at>now()
 AND (sqlc.arg(before_id)::bigint=0 OR (p.published_at,p.id)<(sqlc.arg(before_time)::timestamptz,sqlc.arg(before_id)))
ORDER BY p.published_at DESC,p.id DESC LIMIT 25;

-- name: LockInteractionPost :one
SELECT p.id,post_readable(p,sqlc.arg(viewer_id)::bigint)::boolean AS can_read
FROM posts p JOIN users author ON author.id=p.author_id
WHERE p.id=$1 AND p.published_at IS NOT NULL AND NOT author.blocked AND author.email_verified FOR SHARE OF p;

-- name: InteractionSummary :one
SELECT
 (SELECT count(*) FROM post_likes l JOIN users u ON u.id=l.user_id WHERE l.post_id=sqlc.arg(post_id) AND NOT u.blocked AND u.email_verified)::bigint AS like_count,
 EXISTS(SELECT 1 FROM post_likes own_like WHERE own_like.post_id=sqlc.arg(post_id) AND own_like.user_id=sqlc.arg(viewer_id))::boolean AS liked,
 (SELECT count(*) FROM post_comments c JOIN users u ON u.id=c.user_id WHERE c.post_id=sqlc.arg(post_id) AND NOT u.blocked AND u.email_verified)::bigint AS comment_count;

-- name: LikePost :exec
INSERT INTO post_likes(post_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING;

-- name: UnlikePost :exec
DELETE FROM post_likes WHERE post_id=$1 AND user_id=$2;

-- name: ListComments :many
SELECT c.id,c.post_id,c.body,c.created_at,u.nickname,c.user_id
FROM post_comments c JOIN users u ON u.id=c.user_id
WHERE c.post_id=$1 AND NOT u.blocked AND u.email_verified AND (sqlc.arg(before_id)::bigint=0 OR c.id<sqlc.arg(before_id))
ORDER BY c.id DESC LIMIT 25;

-- name: CreateComment :one
INSERT INTO post_comments(post_id,user_id,body) VALUES($1,$2,$3) RETURNING *;

-- name: DeleteOwnComment :execrows
DELETE FROM post_comments WHERE id=$1 AND post_id=$2 AND user_id=$3;
