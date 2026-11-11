#!/usr/bin/env python3
"""Create local secrets or append newly required settings, preserving existing values."""
import base64
import os
from pathlib import Path
import secrets

path = Path(__file__).resolve().parent.parent / '.env'
settings = {
    'APP_ORIGIN': 'http://localhost:8080', 'APP_PORT': '8080', 'COOKIE_SECURE': 'false',
    'POSTGRES_PASSWORD': secrets.token_hex(24), 'MINIO_ROOT_USER': 'etheyl',
    'MINIO_ROOT_PASSWORD': secrets.token_hex(24), 'MAILPIT_PORT': '8025',
    'MIGRATE_DATABASE_PASSWORD': secrets.token_hex(24),
    'API_DATABASE_PASSWORD': secrets.token_hex(24),
    'WORKER_DATABASE_PASSWORD': secrets.token_hex(24),
    'OUTBOX_KEYS': base64.b64encode(secrets.token_bytes(32)).decode('ascii'),
}
try:
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
except FileExistsError:
    existing = path.read_text()
    names = {line.partition('=')[0] for line in existing.splitlines()}
    missing = {key: value for key, value in settings.items() if key not in names}
    if missing:
        fd = os.open(path, os.O_WRONLY | os.O_APPEND)
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, 'w') as stream:
            if existing and not existing.endswith('\n'):
                stream.write('\n')
            for key, value in missing.items():
                stream.write(f'{key}={value}\n')
        print('Добавлены настройки: ' + ', '.join(missing))
    else:
        print('.env уже содержит необходимые настройки.')
else:
    with os.fdopen(fd, 'w') as stream:
        for key, value in settings.items():
            stream.write(f'{key}={value}\n')
    print('Создан .env со случайными локальными секретами.')
