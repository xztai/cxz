#!/usr/bin/env python3
"""Scan tracked and non-ignored source, including untracked work, without local secrets/dependencies."""
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parent.parent
files = subprocess.check_output(['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'], cwd=root)
with tempfile.TemporaryDirectory(prefix='etheyl-source-scan-') as directory:
    stage = Path(directory)
    for item in files.split(b'\0'):
        if not item:
            continue
        name = Path(item.decode())
        source = root / name
        if not source.is_file() or source.is_symlink():
            continue
        target = stage / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, target)
    subprocess.run([sys.argv[1], 'dir', str(stage), '--redact'], check=True)
