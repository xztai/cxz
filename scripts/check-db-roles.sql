\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE etheyl_api;
SELECT user_id, display_name FROM creator_profiles LIMIT 0;
SELECT user_id, object_key FROM profile_images LIMIT 0;
SELECT id, author_id, price_kopecks FROM subscription_tiers LIMIT 0;
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
  PERFORM object_key FROM profile_images LIMIT 1;
  RAISE EXCEPTION 'worker unexpectedly can read media keys';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  UPDATE users SET is_admin=true WHERE false;
  RAISE EXCEPTION 'worker unexpectedly can change users';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $$;
ROLLBACK;
