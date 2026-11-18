\set ON_ERROR_STOP on
BEGIN;
DO $$
DECLARE
 migrator oid := 'etheyl_migrator'::regrole;
 table_name text;
 function_signature text;
 table_oid oid;
 owner_oid oid;
 sequence_name text;
BEGIN
 FOREACH table_name IN ARRAY ARRAY[
  'users', 'sessions', 'action_tokens', 'mail_outbox', 'rate_limits',
  'creator_profiles', 'profile_images', 'subscription_tiers', 'posts', 'post_media',
  'subscriptions', 'billing_agreements', 'payments', 'post_likes', 'post_comments',
  'goose_db_version'
 ] LOOP
  SELECT c.oid, c.relowner INTO table_oid, owner_oid
   FROM pg_class c
   WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('r', 'p')
   AND c.relname = table_name;
  IF NOT FOUND THEN
   RAISE EXCEPTION 'Missing application table public.%', table_name;
  END IF;
  IF owner_oid <> migrator THEN
   RAISE EXCEPTION 'Application table public.% must be owned by etheyl_migrator', table_name;
  END IF;

  -- Each application identity column, including goose's, must still have an owned sequence.
  IF table_name IN ('users', 'mail_outbox', 'subscription_tiers', 'posts', 'post_media',
                    'billing_agreements', 'post_comments', 'goose_db_version')
     AND pg_get_serial_sequence(format('public.%I', table_name), 'id') IS NULL THEN
   RAISE EXCEPTION 'Missing owned sequence for public.%.id', table_name;
  END IF;
  SELECT s.oid::regclass::text INTO sequence_name
   FROM pg_depend d JOIN pg_class s ON s.oid = d.objid
   WHERE d.classid = 'pg_class'::regclass AND d.refclassid = 'pg_class'::regclass
   AND d.refobjid = table_oid AND d.refobjsubid > 0 AND d.deptype IN ('a', 'i')
   AND s.relkind = 'S' AND s.relowner <> migrator
   LIMIT 1;
  IF FOUND THEN
   RAISE EXCEPTION 'Sequence % must be owned by etheyl_migrator', sequence_name;
  END IF;
 END LOOP;

 FOREACH function_signature IN ARRAY ARRAY[
  'public.protect_used_tier()', 'public.mark_referenced_tier()',
  'public.stop_billing_for_tier()', 'public.stop_billing_for_user()',
  'public.post_readable(public.posts,bigint)'
 ] LOOP
  SELECT p.proowner INTO owner_oid
   FROM pg_proc p WHERE p.oid = to_regprocedure(function_signature) AND p.prokind = 'f';
  IF NOT FOUND THEN
   RAISE EXCEPTION 'Missing application function %', function_signature;
  END IF;
  IF owner_oid <> migrator THEN
   RAISE EXCEPTION 'Application function % must be owned by etheyl_migrator', function_signature;
  END IF;
 END LOOP;
END $$;
SET LOCAL ROLE etheyl_migrator;
ALTER TABLE public.payments ADD COLUMN restore_ownership_probe boolean;
RESET ROLE;
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
