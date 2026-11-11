\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE etheyl_api;
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
  UPDATE users SET is_admin=true WHERE false;
  RAISE EXCEPTION 'worker unexpectedly can change users';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $$;
ROLLBACK;
