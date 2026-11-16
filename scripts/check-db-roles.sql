\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE etheyl_api;
SELECT user_id, display_name FROM creator_profiles LIMIT 0;
SELECT user_id, object_key FROM profile_images LIMIT 0;
SELECT id, author_id, price_kopecks FROM subscription_tiers LIMIT 0;
SELECT id, body_markdown FROM posts LIMIT 0;
SELECT post_id,user_id FROM post_likes LIMIT 0;
SELECT id,body FROM post_comments LIMIT 0;
SELECT id, object_key FROM post_media LIMIT 0;
SELECT subscriber_id, tier_id FROM subscriptions LIMIT 0;
SELECT id,status FROM payments LIMIT 0;
SELECT id,auto_renew FROM billing_agreements LIMIT 0;
UPDATE subscriptions SET expires_at=expires_at WHERE false;
UPDATE creator_profiles SET updated_at=now() WHERE false;
UPDATE profile_images SET updated_at=now() WHERE false;
UPDATE subscription_tiers SET name=name WHERE false;
DO $$
BEGIN
 BEGIN
  CREATE TABLE public.forbidden_api_ddl (id integer);
  RAISE EXCEPTION 'API unexpectedly has DDL permission';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  PERFORM body FROM mail_outbox LIMIT 1;
  RAISE EXCEPTION 'API unexpectedly can read outbox bodies';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $$;
RESET ROLE;
SET LOCAL ROLE etheyl_worker;
SELECT object_key FROM profile_images LIMIT 0;
SELECT id,email,blocked FROM users LIMIT 0;
SELECT id,status FROM payments LIMIT 0;
SELECT id,auto_renew FROM billing_agreements LIMIT 0;
UPDATE subscriptions SET expires_at=expires_at WHERE false;
SELECT object_key, thumbnail_key FROM post_media LIMIT 0;
UPDATE post_media SET state=state WHERE false;
DO $$
BEGIN
 BEGIN
  CREATE TABLE public.forbidden_worker_ddl (id integer);
  RAISE EXCEPTION 'worker unexpectedly has DDL permission';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  PERFORM password_hash FROM users LIMIT 1;
  RAISE EXCEPTION 'worker unexpectedly can read password hashes';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  PERFORM body_markdown FROM posts LIMIT 1;
  RAISE EXCEPTION 'worker unexpectedly can read post bodies';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  UPDATE users SET is_admin=true WHERE false;
  RAISE EXCEPTION 'worker unexpectedly can change users';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $$;
ROLLBACK;
