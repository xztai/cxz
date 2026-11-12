-- name: UpdateNickname :exec
UPDATE users SET nickname = $2 WHERE id = $1;

-- name: SaveCreatorProfile :one
INSERT INTO creator_profiles (user_id, display_name, description) VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE SET display_name = EXCLUDED.display_name,
 description = EXCLUDED.description, updated_at = now() RETURNING *;

-- name: CreatorProfile :one
SELECT p.*, u.nickname FROM creator_profiles p JOIN users u ON u.id = p.user_id
WHERE p.user_id = $1 AND NOT u.blocked AND u.email_verified;

-- name: PublicCreator :one
SELECT p.*, u.nickname FROM creator_profiles p JOIN users u ON u.id = p.user_id
WHERE u.nickname = $1 AND NOT u.blocked AND u.email_verified;

-- name: ListCreators :many
SELECT p.*, u.nickname FROM creator_profiles p JOIN users u ON u.id = p.user_id
WHERE NOT u.blocked AND u.email_verified
 AND (sqlc.arg(search)::text = '' OR strpos(lower(p.display_name), lower(sqlc.arg(search))) > 0
 OR strpos(u.nickname, lower(sqlc.arg(search))) > 0 OR strpos(lower(p.description), lower(sqlc.arg(search))) > 0)
ORDER BY p.created_at DESC, p.user_id DESC LIMIT $1 OFFSET $2;

-- name: LockCreator :one
SELECT * FROM creator_profiles WHERE user_id = $1 FOR UPDATE;

-- name: ProfileImages :many
SELECT * FROM profile_images WHERE user_id = $1 ORDER BY kind;

-- name: ProfileImage :one
SELECT i.* FROM profile_images i JOIN users u ON u.id = i.user_id
WHERE u.nickname = $1 AND i.kind = $2 AND NOT u.blocked AND u.email_verified;

-- name: SaveProfileImage :exec
INSERT INTO profile_images (user_id,kind,object_key,content_type,size_bytes,width,height,sha256)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (user_id,kind) DO UPDATE SET object_key=EXCLUDED.object_key,
 content_type=EXCLUDED.content_type,size_bytes=EXCLUDED.size_bytes,width=EXCLUDED.width,
 height=EXCLUDED.height,sha256=EXCLUDED.sha256,updated_at=now();

-- name: DeleteProfileImage :exec
DELETE FROM profile_images WHERE user_id = $1 AND kind = $2;

-- name: ListTiers :many
SELECT * FROM subscription_tiers WHERE author_id = $1 ORDER BY position DESC;

-- name: CreateTier :one
INSERT INTO subscription_tiers (author_id,name,description,price_kopecks,position)
SELECT $1,$2,$3,$4,COALESCE(MAX(position),0)+1 FROM subscription_tiers WHERE author_id=$1 RETURNING *;

-- name: UpdateTier :one
UPDATE subscription_tiers SET name=$3,description=$4,price_kopecks=$5
WHERE id=$1 AND author_id=$2 AND archived_at IS NULL RETURNING *;

-- name: SetTierPosition :execrows
UPDATE subscription_tiers SET position=$3 WHERE id=$1 AND author_id=$2 AND archived_at IS NULL;

-- name: ArchiveTier :execrows
UPDATE subscription_tiers SET archived_at=COALESCE(archived_at,now()) WHERE id=$1 AND author_id=$2;

-- name: DeleteUnusedTier :execrows
DELETE FROM subscription_tiers WHERE id=$1 AND author_id=$2 AND used_at IS NULL;

-- name: MarkTierUsed :execrows
UPDATE subscription_tiers SET used_at=COALESCE(used_at,now()) WHERE id=$1 AND author_id=$2 AND archived_at IS NULL;

-- name: TierIncludes :one
SELECT EXISTS(SELECT 1 FROM subscription_tiers granted JOIN subscription_tiers required
ON granted.author_id=required.author_id WHERE granted.id=sqlc.arg(granted_id)
AND required.id=sqlc.arg(required_id) AND granted.position >= required.position)::boolean AS includes;
