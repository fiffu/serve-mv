import argparse
import contextlib
import hashlib
import http.server
import json
import os
from pathlib import Path
import shutil
import socket
import datetime

DEFAULT_DNS = "serve-mv.local"
DEFAULT_DNS_SUBDOMAIN = '(generated from System.json["gameTitle"])'
DEFAULT_PORT = 9001


def log(*args, **kwargs):
    print("[pymv]", *args, **kwargs)


class SystemJson:
    def __init__(self, path: Path) -> None:
        self.path = path
        self.cached = None

    @classmethod
    def load_from(cls, path: Path):
        instance = cls(path)
        instance._load()
        return instance

    @property
    def game_title(self) -> str:
        return self.get("gameTitle")

    def get(self, *keys):
        cursor = self._load()

        for i, key in enumerate(keys):
            try:
                cursor = cursor[key]
            except (IndexError, AttributeError) as exc:
                path = ".".join(keys[: i + 1])
                cls = exc.__class__
                msg = str(exc)
                raise AttributeError(f"{cls} at {path}: {msg}")
        return cursor

    def _load(self):
        with open(self.path) as file:
            self.cached = json.load(file)
        return self.cached


class TempDNS:
    def __init__(self, cwd: str, dns: str, dns_prefix: str):
        self.system_json: SystemJson = self.detect_system_json(cwd)
        self.dns = dns
        self.dns_prefix = (
            self.calculate_dns_prefix()
            if dns_prefix == DEFAULT_DNS_SUBDOMAIN
            else dns_prefix
        )

        now = int(datetime.datetime.now().timestamp() * 1e6)
        self.hosts_file = "/etc/hosts"
        self.backup_file = f"/etc/hosts.pymv.{now}.bk"

    @property
    def hostname(self):
        return f"{self.dns_prefix}.{self.dns}"

    @property
    def hosts_record(self):
        return f"\n127.0.0.1\t{self.hostname}\n"

    @staticmethod
    def detect_system_json(cwd: str):
        path = Path(cwd) / "www" / "data" / "System.json"
        if not path.exists():
            log(f"Could not find {path.absolute()}")
            raise FileNotFoundError(path)
        return SystemJson.load_from(path)

    def calculate_dns_prefix(self):
        title = self.system_json.game_title
        return hashlib.md5(title.encode("utf-8")).hexdigest()

    @contextlib.contextmanager
    def context(self) -> str:
        host = self.hostname
        try:
            self.hosts_setup(self.hosts_file, self.backup_file, self.hosts_record)
            yield host
        finally:
            self.hosts_teardown(self.hosts_file, self.backup_file, self.hosts_record)

    @staticmethod
    def hosts_setup(hosts_file, backup_file, hostname_record):
        log(f"Backing up {hosts_file} to {backup_file}")
        shutil.copy(hosts_file, backup_file)

        log(f"Updating in {hosts_file}")
        with open(hosts_file, "a") as file:
            file.write(hostname_record)

    @staticmethod
    def hosts_teardown(hosts_file, backup_file, hostname_record):
        log(f"Removing in {hosts_file} - {hostname_record}")
        with open(hosts_file, "r+") as file:
            content = file.read()
            if hostname_record not in content:
                return

            replacement = content.replace(hostname_record, "")
            file.truncate(0)
            file.write(replacement)

        log(f"Removing backup {backup_file}")
        os.remove(backup_file)


def serve(directory: str, port: int, tmpdns: TempDNS):
    # Adapted from https://github.com/python/cpython/blob/3.12/Lib/http/server.py#L1314
    class Server(http.server.ThreadingHTTPServer):
        def log_message(self, format, **args):
            log(format % args)

        def server_bind(self):
            # suppress exception when protocol is IPv4
            with contextlib.suppress(Exception):
                self.socket.setsockopt(socket.IPPROTO_IPV6, socket.IPV6_V6ONLY, 0)
            return super().server_bind()

        def finish_request(self, request, client_address):
            self.RequestHandlerClass(request, client_address, self, directory=directory)

    with tmpdns.context() as host:
        log(f"Starting server - - - http://{tmpdns.hostname}:{port}/www/index.html")
        http.server.test(
            HandlerClass=http.server.SimpleHTTPRequestHandler,
            ServerClass=Server,
            bind=tmpdns.hostname,
            port=port,
        )


if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    parser.add_argument(
        "--dir",
        help="serve this directory (default: current directory)",
        default=os.getcwd(),
    )
    parser.add_argument("--domain", help="the DNS domain to use", default=DEFAULT_DNS)
    parser.add_argument(
        "--subdomain", help="the subdomain to use", default=DEFAULT_DNS_SUBDOMAIN
    )
    parser.add_argument(
        "--port", help="network port to use", type=int, default=DEFAULT_PORT
    )

    args = parser.parse_args()

    dns = TempDNS(args.dir, args.domain, args.subdomain)
    serve(args.dir, args.port, dns)
