-- name: CreatePost :one
INSERT INTO posts(author_id,title,teaser,body_markdown,minimum_tier_id)
VALUES($1,$2,$3,$4,$5) RETURNING *;

-- name: UpdatePost :one
UPDATE posts SET title=$3,teaser=$4,body_markdown=$5,minimum_tier_id=$6,updated_at=now()
WHERE id=$1 AND author_id=$2 RETURNING *;

-- name: OwnPost :one
SELECT * FROM posts WHERE id=$1 AND author_id=$2;

-- name: PublishPost :execrows
UPDATE posts SET published_at=CASE WHEN sqlc.arg(publish)::boolean THEN COALESCE(published_at,now()) ELSE NULL END,
updated_at=now() WHERE id=$1 AND author_id=$2;

-- name: DeletePost :execrows
DELETE FROM posts WHERE id=$1 AND author_id=$2;

-- name: ListOwnPosts :many
SELECT * FROM posts WHERE author_id=$1 ORDER BY id DESC LIMIT 25 OFFSET $2;

-- name: ListPublishedPosts :many
SELECT id,title,teaser,minimum_tier_id,published_at FROM posts
WHERE author_id=$1 AND published_at IS NOT NULL ORDER BY id DESC LIMIT 25 OFFSET $2;

-- name: PostAccess :one
SELECT p.*, u.nickname,
 post_readable(p,sqlc.arg(viewer_id)::bigint)::boolean AS can_read
FROM posts p JOIN users u ON u.id=p.author_id
WHERE p.id=$1 AND NOT u.blocked AND u.email_verified
 AND (p.published_at IS NOT NULL OR p.author_id=sqlc.arg(viewer_id)::bigint);

-- name: ReservePostMedia :one
INSERT INTO post_media(author_id,post_id,kind,filename,content_type,size_bytes,object_key,thumbnail_key)
VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING *;

-- name: CompletePostMedia :execrows
UPDATE post_media SET state='ready' WHERE id=$1 AND author_id=$2 AND post_id IS NOT NULL AND state='uploading';

-- name: ListPostMedia :many
SELECT * FROM post_media WHERE post_id=$1 AND state='ready' ORDER BY id;

-- name: GetPostMedia :one
SELECT * FROM post_media WHERE id=$1 AND post_id IS NOT NULL AND state='ready';

-- name: DetachPostMedia :execrows
UPDATE post_media SET post_id=NULL,state='deleting' WHERE id=$1 AND post_id=$2 AND author_id=$3;

-- name: ClaimStaleMedia :many
UPDATE post_media SET state='deleting' WHERE id IN (
 SELECT id FROM post_media WHERE state='deleting' OR
 (created_at < now()-interval '1 hour' AND (state='uploading' OR post_id IS NULL))
 ORDER BY id LIMIT 100 FOR UPDATE SKIP LOCKED
) RETURNING *;

-- name: DeleteStaleMedia :exec
DELETE FROM post_media WHERE id=$1 AND state='deleting';

-- name: ReferencedMediaKeys :many
SELECT m.object_key FROM post_media m WHERE m.object_key=ANY(sqlc.arg(keys)::text[])
UNION
SELECT m.thumbnail_key FROM post_media m WHERE m.thumbnail_key=ANY(sqlc.arg(keys)::text[])
UNION
SELECT i.object_key FROM profile_images i WHERE i.object_key=ANY(sqlc.arg(keys)::text[]);
