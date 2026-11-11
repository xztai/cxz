#!/usr/bin/env python3
"""Install pinned Linux amd64 scanner releases after SHA-256 verification."""
import hashlib
import io
import os
from pathlib import Path
import sys
import tarfile
import urllib.request

releases = {
    'grype': ('anchore/grype', '0.121.3', 'linux_amd64', 'e9da18cc1429e786722e32161288103600efffd8e483c90761f90485bd1bec85'),
    'gitleaks': ('gitleaks/gitleaks', '8.30.1', 'linux_x64', '551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb'),
}
destination = Path(sys.argv[1])
destination.mkdir(parents=True, exist_ok=True)
for name, (repo, version, platform, checksum) in releases.items():
    url = f'https://github.com/{repo}/releases/download/v{version}/{name}_{version}_{platform}.tar.gz'
    with urllib.request.urlopen(url, timeout=120) as response:
        data = response.read()
    if hashlib.sha256(data).hexdigest() != checksum:
        raise SystemExit(f'{name}: checksum mismatch')
    with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as archive:
        source = archive.extractfile(name)
        if source is None:
            raise SystemExit(f'{name}: executable missing')
        target = destination / name
        target.write_bytes(source.read())
        os.chmod(target, 0o755)
    print(f'{name} {version}: verified')
