#!/usr/bin/env python3
"""Verify data and application roles in a disposable PostgreSQL recovery."""
import os
from pathlib import Path
import subprocess
import tempfile
import time


ROOT = Path(__file__).resolve().parent.parent
ADMIN = 'etheyl_test'
SOURCE = 'etheyl_test'
RESTORED = 'restore_drill_test'
TABLES = (
    'users', 'sessions', 'action_tokens', 'mail_outbox', 'rate_limits',
    'creator_profiles', 'profile_images', 'subscription_tiers', 'posts', 'post_media',
    'subscriptions', 'billing_agreements', 'payments', 'post_likes', 'post_comments',
    'goose_db_version',
)


def docker(*args, **kwargs):
    return subprocess.run(['docker', *args], **kwargs)


def output(*args, **kwargs):
    return docker(*args, check=True, stdout=subprocess.PIPE, text=True, **kwargs).stdout.strip()


def execute(container, *args, **kwargs):
    return docker('exec', '-i', container, *args, **kwargs)


def psql(container, database, *args, **kwargs):
    return execute(container, 'psql', '-X', '-U', ADMIN, '-d', database,
                   '-v', 'ON_ERROR_STOP=1', *args, **kwargs)


def fingerprint(container, database):
    statements = [
        f"SELECT '{table}', count(*), "
        "md5(coalesce(string_agg(row_to_json(t)::text,'' ORDER BY row_to_json(t)::text),'')) "
        f"FROM public.{table} t;"
        for table in TABLES
    ]
    return psql(container, database, '-At', '-c', '\n'.join(statements),
                check=True, stdout=subprocess.PIPE).stdout


def wait_for_postgres(container):
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        # The image initializes with a temporary server that only listens on a socket.
        try:
            ready = execute(container, 'pg_isready', '-h', '127.0.0.1',
                            '-U', ADMIN, '-d', 'postgres', timeout=2, capture_output=True)
        except subprocess.TimeoutExpired:
            continue
        if ready.returncode == 0:
            return
        time.sleep(1)
    raise SystemExit('Disposable PostgreSQL did not become ready within 60 seconds')


def apply_roles(container, phase):
    result = docker('exec', '-i', '-e', f'ROLE_PHASE={phase}', '-e', f'POSTGRES_DB={RESTORED}',
                    container, 'sh', input=(ROOT / 'infra/db-roles.sh').read_bytes(),
                    capture_output=True)
    if result.returncode:
        raise SystemExit(f'Database role setup failed during {phase}')


def check_roles(container):
    return psql(container, RESTORED, '-q', '-f', '-',
                input=(ROOT / 'scripts/check-db-roles.sql').read_bytes(), capture_output=True)


def require_roles(container):
    result = check_roles(container)
    if result.returncode:
        raise SystemExit(result.stderr.decode(errors='replace').strip())


def recover(container, dump, before):
    docker('start', container, check=True, stdout=subprocess.DEVNULL)
    wait_for_postgres(container)
    execute(container, 'createdb', '-U', ADMIN, RESTORED, check=True)
    apply_roles(container, 'bootstrap')
    print('Empty database bootstrap passed.', flush=True)

    execute(container, 'pg_restore', '-U', ADMIN, '--no-owner', '--no-acl',
            '--exit-on-error', '-d', RESTORED, check=True, stdin=dump)
    psql(container, RESTORED, '-q', '-c', 'CREATE TABLE public.restore_sentinel (id integer)',
         check=True)
    apply_roles(container, 'bootstrap')
    apply_roles(container, 'bootstrap')
    apply_roles(container, 'grants')
    require_roles(container)

    psql(container, RESTORED, '-q', '-c', f'ALTER TABLE public.payments OWNER TO {ADMIN}',
         check=True)
    mismatch = check_roles(container)
    expected = b'Application table public.payments must be owned by etheyl_migrator'
    if mismatch.returncode == 0 or expected not in mismatch.stderr:
        raise SystemExit('Ownership check did not report the deliberate payments owner mismatch')
    print('Ownership check rejected the deliberate payments owner mismatch.', flush=True)

    apply_roles(container, 'bootstrap')
    apply_roles(container, 'grants')
    require_roles(container)
    owner = psql(container, RESTORED, '-At', '-c',
                 "SELECT relowner::regrole FROM pg_class WHERE oid = 'public.restore_sentinel'::regclass",
                 check=True, stdout=subprocess.PIPE, text=True).stdout.strip()
    if owner != ADMIN:
        raise SystemExit('Bootstrap changed the unrelated sentinel table owner')
    if before != fingerprint(container, RESTORED):
        raise SystemExit('Restore fingerprints differ')


def main():
    source = os.environ.get('RESTORE_CONTAINER') or output(
        'compose', '-p', 'etheyl-tests', '-f', str(ROOT / 'compose.test.yaml'), 'ps', '-q', 'test-postgres'
    )
    if not source:
        raise SystemExit('Integration-test PostgreSQL container is not running')

    image = output('container', 'inspect', '--format', '{{.Image}}', source)
    before = fingerprint(source, SOURCE)
    with tempfile.TemporaryFile() as dump:
        execute(source, 'pg_dump', '-U', ADMIN, '-d', SOURCE, '--format=custom',
                check=True, stdout=dump)
        dump.seek(0)
        settings = {
            'POSTGRES_USER': ADMIN,
            'POSTGRES_PASSWORD': 'restore_admin_test',
            'MIGRATE_DATABASE_PASSWORD': 'restore_migrator_test',
            'API_DATABASE_PASSWORD': 'restore_api_test',
            'WORKER_DATABASE_PASSWORD': 'restore_worker_test',
            'TZ': 'UTC',
        }
        environment_args = [arg for name in settings for arg in ('-e', name)]
        destination = output(
            'create', '--pull=never', '--network', 'none', '--tmpfs', '/var/lib/postgresql',
            *environment_args, image, env={**os.environ, **settings},
        )
        try:
            recover(destination, dump, before)
        finally:
            docker('rm', '-fv', destination, check=True, stdout=subprocess.DEVNULL)
    print('Restore drill passed. Data fingerprints, application roles, and sentinel ownership match.')


if __name__ == '__main__':
    main()
