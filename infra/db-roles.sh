#!/bin/sh
set -eu
# psql variables quote identifiers/literals; never interpolate passwords into shell SQL.
if [ "${ROLE_PHASE}" = bootstrap ]; then
 psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -v migrate_password="$MIGRATE_DATABASE_PASSWORD" -v api_password="$API_DATABASE_PASSWORD" -v worker_password="$WORKER_DATABASE_PASSWORD" <<'SQL'
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
-- Upgrade old local volumes: transfer only application tables, never system objects.
SELECT format('ALTER TABLE public.%I OWNER TO etheyl_migrator', tablename)
 FROM pg_tables WHERE schemaname='public' AND tablename IN
 ('users','sessions','action_tokens','mail_outbox','rate_limits','goose_db_version') \gexec
SQL
else
 psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM etheyl_api, etheyl_worker;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM etheyl_api, etheyl_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON users, sessions, action_tokens, rate_limits TO etheyl_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON creator_profiles, profile_images, subscription_tiers TO etheyl_api;
GRANT USAGE ON SEQUENCE subscription_tiers_id_seq TO etheyl_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON posts, post_media TO etheyl_api;
GRANT SELECT ON subscriptions TO etheyl_api;
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
