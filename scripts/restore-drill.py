#!/usr/bin/env python3
"""Exercise pg_dump/pg_restore only on the dedicated integration-test PostgreSQL."""
import os
import subprocess
import tempfile
import uuid

compose = ['docker', 'compose', '-p', 'etheyl-tests', '-f', 'compose.test.yaml', 'exec', '-T', 'test-postgres']
if container := os.environ.get('RESTORE_CONTAINER'):
    compose = ['docker', 'exec', '-i', container]
source = 'etheyl_test'
restored = 'restore_' + uuid.uuid4().hex + '_test'

def run(*args, **kwargs):
    return subprocess.run(compose + list(args), check=True, **kwargs)

def fingerprint(database):
    tables = ('users', 'sessions', 'action_tokens', 'mail_outbox', 'rate_limits', 'creator_profiles', 'profile_images', 'subscription_tiers', 'goose_db_version')
    statements = [f"SELECT '{table}', count(*), md5(coalesce(string_agg(row_to_json(t)::text,'' ORDER BY row_to_json(t)::text),'')) FROM {table} t;" for table in tables]
    return run('psql', '-U', 'etheyl_test', '-d', database, '-At', '-v', 'ON_ERROR_STOP=1', '-c', '\n'.join(statements), capture_output=True).stdout

before = fingerprint(source)
with tempfile.TemporaryFile() as dump:
    run('pg_dump', '-U', 'etheyl_test', '-d', source, '--format=custom', stdout=dump)
    dump.seek(0)
    run('createdb', '-U', 'etheyl_test', restored)
    try:
        run('pg_restore', '-U', 'etheyl_test', '--no-owner', '--no-acl', '--exit-on-error', '-d', restored, stdin=dump)
        if before != fingerprint(restored):
            raise SystemExit('Restore fingerprints differ')
        print('Restore drill passed: all nine application/migration tables match.')
    finally:
        run('dropdb', '-U', 'etheyl_test', restored)
