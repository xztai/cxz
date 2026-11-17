-- +goose Up
-- One read predicate for text, media, feed and interactions. SECURITY INVOKER:
-- application roles retain their existing table privileges and row visibility.
-- +goose StatementBegin
CREATE FUNCTION post_readable(p posts, viewer bigint) RETURNS boolean
LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM users author WHERE author.id=p.author_id AND NOT author.blocked AND author.email_verified)
 AND (p.published_at IS NOT NULL OR p.author_id=viewer)
 AND (viewer=0 OR EXISTS(SELECT 1 FROM users u WHERE u.id=viewer AND NOT u.blocked AND u.email_verified))
 AND (p.author_id=viewer OR p.minimum_tier_id IS NULL OR EXISTS(
  SELECT 1 FROM subscriptions sub JOIN subscription_tiers granted ON granted.id=sub.tier_id
  JOIN subscription_tiers required ON required.id=p.minimum_tier_id
  WHERE sub.subscriber_id=viewer AND sub.author_id=p.author_id
   AND granted.author_id=p.author_id AND required.author_id=p.author_id
   AND granted.position>=required.position AND sub.revoked_at IS NULL
   AND sub.starts_at<=now() AND sub.expires_at>now()
 ));
$$;
-- +goose StatementEnd
CREATE TABLE post_likes (
 post_id bigint NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(post_id,user_id)
);
CREATE TABLE post_comments (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 post_id bigint NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
 user_id bigint NOT NULL REFERENCES users(id),
 body text NOT NULL CHECK(char_length(btrim(body)) BETWEEN 1 AND 2000),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX post_comments_page ON post_comments(post_id,id DESC);
CREATE INDEX posts_feed ON posts(published_at DESC,id DESC) WHERE published_at IS NOT NULL;
CREATE INDEX paid_author_analytics ON payments(agreement_id,completed_at) WHERE status='succeeded';
CREATE INDEX subscriptions_author_active ON subscriptions(author_id,expires_at) WHERE revoked_at IS NULL;

-- +goose Down
DROP INDEX subscriptions_author_active,paid_author_analytics,posts_feed;
DROP TABLE post_comments,post_likes;
DROP FUNCTION post_readable(posts,bigint);
