#!/usr/bin/env python3
"""Run the Go seeder against the private services of the local Compose stack."""

import contextlib
import json
import os
from pathlib import Path
import socket
import socketserver
import subprocess
import sys
import threading
from urllib.parse import quote

ROOT = Path(__file__).resolve().parent.parent
COMPOSE = ["docker", "compose", "-f", "compose.yaml"]


def output(command):
    return subprocess.run(command, cwd=ROOT, check=True, stdout=subprocess.PIPE,
                          stderr=subprocess.DEVNULL).stdout


def service_env(service, names):
    # Values go straight into the child environment. Never print the output.
    values = output(COMPOSE + [
        "exec", "-T", service, "sh", "-c",
        'for name do printenv "$name" || printf "\\n"; done', "seed", *names,
    ]).decode().splitlines()
    if len(values) != len(names):
        raise RuntimeError("неожиданный формат переменных запущенного Compose")
    return dict(zip(names, values, strict=True))


def local_docker():
    endpoint = os.getenv("DOCKER_HOST")
    if os.getenv("DOCKER_CONTEXT") or not endpoint:
        current = json.loads(output(["docker", "context", "inspect"]))[0]
        endpoint = current["Endpoints"]["docker"]["Host"]
    if not endpoint.startswith("unix://"):
        raise RuntimeError("сидеру нужен локальный Docker через Unix-сокет")


class Tunnel(socketserver.ThreadingTCPServer):
    daemon_threads = True

    def __init__(self, target, port):
        self.target = target
        self.port = port
        super().__init__(("127.0.0.1", 0), TunnelConnection)


class TunnelConnection(socketserver.BaseRequestHandler):
    def handle(self):
        command = COMPOSE + ["exec", "-T", "postgres", "nc", self.server.target, self.server.port]
        with subprocess.Popen(command, cwd=ROOT, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                              stderr=subprocess.DEVNULL, bufsize=0) as tunnel:
            def forward():
                try:
                    with tunnel.stdin:
                        while data := self.request.recv(65536):
                            view = memoryview(data)
                            while view:
                                view = view[tunnel.stdin.write(view):]
                except OSError:
                    pass

            sender = threading.Thread(target=forward, daemon=True)
            sender.start()
            try:
                while data := tunnel.stdout.read(65536):
                    self.request.sendall(data)
            except OSError:
                pass
            finally:
                with contextlib.suppress(OSError):
                    self.request.shutdown(socket.SHUT_RDWR)
                if tunnel.poll() is None:
                    tunnel.terminate()
                sender.join()


@contextlib.contextmanager
def local_port(target, port):
    with Tunnel(target, port) as server:
        worker = threading.Thread(target=server.serve_forever, daemon=True)
        worker.start()
        try:
            yield server.server_address[1]
        finally:
            server.shutdown()
            worker.join()


def main(args):
    command = ["go", "run", "./cmd/seed", *args]
    if any(arg in ("-h", "--help") for arg in args):
        return subprocess.call(command, cwd=ROOT / "backend")
    if os.getenv("SEED_ALLOW") != "1":
        raise RuntimeError("для запуска нужен SEED_ALLOW=1")
    if set(args) - {"--reset", "--dry-run"}:
        raise RuntimeError("допустимые параметры: --reset и --dry-run")
    if os.getenv("DATABASE_URL"):
        return subprocess.call(command, cwd=ROOT / "backend")
    local_docker()
    api = service_env("api", (
        "APP_ORIGIN", "COOKIE_SECURE", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY",
        "S3_BUCKET", "S3_SECURE", "YOOKASSA_API_KEY", "APP_ENV", "ENVIRONMENT", "NODE_ENV",
    ))
    if api["S3_ENDPOINT"] != "rustfs:9000" or api["S3_SECURE"] != "false":
        raise RuntimeError("запущенный API должен использовать локальный RustFS из compose.yaml")
    api_key = api.pop("YOOKASSA_API_KEY")
    if api_key and not api_key.startswith(("test_", "secret_test_")):
        raise RuntimeError("запущенный API использует настройки ЮKassa, похожие на production")
    db = service_env("postgres", ("POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"))
    if db["POSTGRES_USER"] != "etheyl" or db["POSTGRES_DB"] != "etheyl":
        raise RuntimeError("ожидается локальный Postgres etheyl из compose.yaml")
    env = os.environ.copy()
    for name, value in api.items():
        if name in ("APP_ENV", "ENVIRONMENT", "NODE_ENV"):
            if env.get(name):
                # The Go command checks both host and Compose environment.
                env[name] += " " + value
            else:
                env[name] = value
        elif name != "S3_ENDPOINT":
            env[name] = value
    if "--dry-run" not in args:
        running = output(COMPOSE + ["ps", "--status", "running", "--quiet", "worker"]).strip()
        if running:
            worker = service_env("worker", ("YOOKASSA_SHOP_ID", "YOOKASSA_API_KEY"))
            if worker["YOOKASSA_SHOP_ID"] and worker["YOOKASSA_API_KEY"]:
                raise RuntimeError(
                    "у воркера включён биллинг. Для запуска сида выполните:\n"
                    "YOOKASSA_SHOP_ID= YOOKASSA_API_KEY= docker compose up -d --no-deps worker"
                )
        env["SEED_BILLING_WORKER_DISABLED"] = "1"

    with local_port("127.0.0.1", "5432") as pgport, local_port("rustfs", "9000") as s3port:
        password = quote(db["POSTGRES_PASSWORD"], safe="")
        env["DATABASE_URL"] = f"postgres://etheyl:{password}@127.0.0.1:{pgport}/etheyl?sslmode=disable"
        env["S3_ENDPOINT"] = f"127.0.0.1:{s3port}"
        return subprocess.call(command, cwd=ROOT / "backend", env=env)


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1:]))
    except (OSError, subprocess.CalledProcessError, RuntimeError) as exc:
        message = str(exc) if isinstance(exc, RuntimeError) else "проверьте локальный Docker, запущенный Compose и Go"
        print(f"Сидер: {message}", file=sys.stderr)
        sys.exit(1)
