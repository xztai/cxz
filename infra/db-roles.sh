#!/bin/sh
set -eu
# psql variables quote identifiers/literals; never interpolate passwords into shell SQL.
if [ "${ROLE_PHASE}" = bootstrap ]; then
 psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -v migrate_password="$MIGRATE_DATABASE_PASSWORD" -v api_password="$API_DATABASE_PASSWORD" -v worker_password="$WORKER_DATABASE_PASSWORD" <<'SQL'
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT 'CREATE ROLE etheyl_migrator LOGIN' WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname='etheyl_migrator') \gexec
SELECT 'CREATE ROLE etheyl_api LOGIN' WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname='etheyl_api') \gexec
SELECT 'CREATE ROLE etheyl_worker LOGIN' WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname='etheyl_worker') \gexec
ALTER ROLE etheyl_migrator PASSWORD :'migrate_password';
ALTER ROLE etheyl_api PASSWORD :'api_password' NOSUPERUSER NOCREATEDB NOCREATEROLE;
ALTER ROLE etheyl_worker PASSWORD :'worker_password' NOSUPERUSER NOCREATEDB NOCREATEROLE;
ALTER ROLE etheyl_api SET statement_timeout='4s';
ALTER ROLE etheyl_api SET lock_timeout='2s';
ALTER ROLE etheyl_api SET idle_in_transaction_session_timeout='10s';
ALTER ROLE etheyl_worker SET statement_timeout='4s';
ALTER ROLE etheyl_worker SET lock_timeout='2s';
ALTER ROLE etheyl_worker SET idle_in_transaction_session_timeout='20s';
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA public TO etheyl_migrator;
GRANT USAGE ON SCHEMA public TO etheyl_api, etheyl_worker;
-- Repair only application objects. Table ownership also covers its owned sequences.
SELECT format('ALTER TABLE public.%I OWNER TO etheyl_migrator', c.relname)
 FROM pg_class c
 WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('r', 'p')
 AND c.relname IN (
  'users', 'sessions', 'action_tokens', 'mail_outbox', 'rate_limits',
  'creator_profiles', 'profile_images', 'subscription_tiers', 'posts', 'post_media',
  'subscriptions', 'billing_agreements', 'payments', 'post_likes', 'post_comments',
  'goose_db_version'
 )
 AND NOT EXISTS (
  SELECT FROM pg_depend d
  WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
 ) \gexec
SELECT format('ALTER FUNCTION public.%I(%s) OWNER TO etheyl_migrator',
              p.proname, pg_get_function_identity_arguments(p.oid))
 FROM pg_proc p
 WHERE p.pronamespace = 'public'::regnamespace AND p.prokind = 'f'
 AND (
  (p.proname IN ('protect_used_tier', 'mark_referenced_tier', 'stop_billing_for_tier',
                 'stop_billing_for_user') AND p.pronargs = 0)
  OR (p.proname = 'post_readable' AND p.pronargs = 2
      AND p.proargtypes[0] = to_regtype('public.posts') AND p.proargtypes[1] = 'bigint'::regtype)
 )
 AND NOT EXISTS (
  SELECT FROM pg_depend d
  WHERE d.classid = 'pg_proc'::regclass AND d.objid = p.oid AND d.deptype = 'e'
 ) \gexec
COMMIT;
SQL
else
 psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM etheyl_api, etheyl_worker;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM etheyl_api, etheyl_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON users, sessions, action_tokens, rate_limits TO etheyl_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON creator_profiles, profile_images, subscription_tiers TO etheyl_api;
GRANT USAGE ON SEQUENCE subscription_tiers_id_seq TO etheyl_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON posts, post_media TO etheyl_api;
GRANT SELECT, INSERT, DELETE ON post_likes,post_comments TO etheyl_api;
GRANT USAGE ON SEQUENCE post_comments_id_seq TO etheyl_api;
GRANT SELECT, INSERT, UPDATE ON subscriptions, billing_agreements, payments TO etheyl_api, etheyl_worker;
GRANT USAGE ON SEQUENCE billing_agreements_id_seq TO etheyl_api, etheyl_worker;
GRANT SELECT ON creator_profiles, subscription_tiers TO etheyl_worker;
GRANT UPDATE (updated_at) ON creator_profiles TO etheyl_worker;
GRANT UPDATE (used_at) ON subscription_tiers TO etheyl_worker;
GRANT SELECT (email) ON users TO etheyl_worker;
GRANT INSERT ON mail_outbox TO etheyl_worker;
GRANT USAGE ON SEQUENCE mail_outbox_id_seq TO etheyl_worker;
GRANT USAGE ON SEQUENCE posts_id_seq, post_media_id_seq TO etheyl_api;
GRANT SELECT, UPDATE, DELETE ON post_media TO etheyl_worker;
GRANT SELECT (object_key) ON profile_images TO etheyl_worker;
GRANT INSERT ON mail_outbox TO etheyl_api;
GRANT USAGE ON SEQUENCE users_id_seq, mail_outbox_id_seq TO etheyl_api;
GRANT SELECT, UPDATE, DELETE ON mail_outbox TO etheyl_worker;
GRANT SELECT, DELETE ON action_tokens TO etheyl_worker;
GRANT SELECT (id, blocked, email_verified) ON users TO etheyl_worker;
GRANT SELECT (expires_at), DELETE ON sessions, rate_limits TO etheyl_worker;
SQL
fi
