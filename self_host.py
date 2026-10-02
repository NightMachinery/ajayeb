#!/usr/bin/env python3
"""Manage Ajayeb self-hosting with Caddy and tmux.

Adapted from the snow-white self_host.py for Ajayeb's architecture: a single
Go server (server/cmd/ajayeb) that embeds the SvelteKit build and owns
/api, /ws and /health, plus a Vite dev server for frontend development.
Caddy terminates TLS and reverse-proxies; tmux keeps every long-running
process easy to inspect or restart without Docker.
"""

from __future__ import annotations

import argparse
import json
import os
import platform
import shutil
import socket
import subprocess
import sys
import tarfile
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from enum import Enum
from pathlib import Path
from typing import Callable
from urllib.parse import urlparse


ROOT = Path(__file__).resolve().parent
SERVER_DIR = ROOT / "server"
WEB_DIR = ROOT / "web"
WEB_BUILD_DIR = WEB_DIR / "build"
WEBDIST_DIST = SERVER_DIR / "internal" / "webdist" / "dist"
CONFIG_DIR = ROOT / ".self-host"
CONFIG_FILE = CONFIG_DIR / "config.json"
BIN_DIR = CONFIG_DIR / "bin"
BIN_PATH = BIN_DIR / "ajayeb"
BIN_NEW_PATH = BIN_DIR / "ajayeb.new"
DATA_DIR = CONFIG_DIR / "data"
BACKUPS_DIR = CONFIG_DIR / "backups"
ROOMS_BACKUP_KEEP = 10
CADDYFILE = Path.home() / "Caddyfile"
DEFAULT_URL = "https://ajayeb.pinky.lilf.ir"
DEFAULT_NODE_VERSION = "24"
HEALTH_POLL_TIMEOUT = 10.0
HEALTH_POLL_INTERVAL = 0.3

DEFAULT_SERVER_PORT = 38951
DEFAULT_WEB_DEV_PORT = 38952

PROD_SESSION = "ajayeb-server"
DEV_SERVER_SESSION = "ajayeb-dev-server"
DEV_WEB_SESSION = "ajayeb-dev-web"

MANAGED_START = "# BEGIN ajayeb self_host.py"
MANAGED_END = "# END ajayeb self_host.py"

PROXY_ENV_NAMES = (
    "ALL_PROXY",
    "all_proxy",
    "http_proxy",
    "https_proxy",
    "HTTP_PROXY",
    "HTTPS_PROXY",
    "npm_config_proxy",
    "npm_config_https_proxy",
    "NO_PROXY",
    "no_proxy",
    "AJAYEB_SERVER",
)

GOMEMLIMIT_PROD = "96MiB"
TRUST_PROXY_PROD = "127.0.0.1"


class Mode(str, Enum):
    PROD = "prod"
    DEV = "dev"


@dataclass(frozen=True)
class Ports:
    server: int = DEFAULT_SERVER_PORT
    web_dev: int = DEFAULT_WEB_DEV_PORT

    def to_json(self) -> dict[str, int]:
        return {"server": self.server, "web_dev": self.web_dev}

    @classmethod
    def from_json(cls, raw: object) -> "Ports":
        if not isinstance(raw, dict):
            return cls()
        return cls(
            server=int(raw.get("server", DEFAULT_SERVER_PORT)),
            web_dev=int(raw.get("web_dev", DEFAULT_WEB_DEV_PORT)),
        )


@dataclass(frozen=True)
class Site:
    scheme: str
    host: str
    port: int | None = None

    @property
    def authority(self) -> str:
        return f"{self.host}:{self.port}" if self.port else self.host

    @property
    def origin(self) -> str:
        return f"{self.scheme}://{self.authority}"

    @property
    def opposite_scheme(self) -> str:
        return "http" if self.scheme == "https" else "https"

    @property
    def opposite_origin(self) -> str:
        return f"{self.opposite_scheme}://{self.authority}"


@dataclass(frozen=True)
class Config:
    site: Site
    mode: Mode
    ports: Ports = Ports()

    def to_json(self) -> str:
        return json.dumps(
            {"url": self.site.origin, "mode": self.mode.value, "ports": self.ports.to_json()},
            indent=2,
            sort_keys=True,
        )

    @classmethod
    def from_json(cls, payload: str) -> "Config":
        raw = json.loads(payload)
        return cls(
            site=parse_site_url(raw.get("url")),
            mode=Mode(raw.get("mode", Mode.PROD.value)),
            ports=Ports.from_json(raw.get("ports")),
        )


def parse_site_url(value: str | None) -> Site:
    parsed = urlparse(value or DEFAULT_URL)
    if parsed.scheme not in {"http", "https"}:
        raise ValueError("URL must start with http:// or https://")
    if not parsed.hostname:
        raise ValueError("URL must include a host")
    if parsed.path not in {"", "/"} or parsed.params or parsed.query or parsed.fragment:
        raise ValueError("URL must be only an origin, for example https://ajayeb.example.test")
    return Site(parsed.scheme, parsed.hostname, parsed.port)


def load_config() -> Config | None:
    if not CONFIG_FILE.exists():
        return None
    return Config.from_json(CONFIG_FILE.read_text())


def save_config(config: Config) -> None:
    CONFIG_DIR.mkdir(exist_ok=True)
    CONFIG_FILE.write_text(config.to_json() + "\n")


def tmux_env_args(env: dict[str, str] | None = None) -> list[str]:
    source = env if env is not None else os.environ
    args: list[str] = []
    for name in PROXY_ENV_NAMES:
        value = source.get(name)
        if value:
            args.extend(["-e", f"{name}={value}"])
    return args


def web_dev_host() -> str:
    return "localhost" if platform.system() == "Darwin" else "0.0.0.0"


def render_caddy_block(site: Site, mode: Mode, ports: Ports = Ports()) -> str:
    """Render the managed Caddyfile block.

    In prod Go serves the embedded SPA itself, so Caddy is a single
    reverse_proxy plus an HTTP-to-HTTPS redirect: no try_files. In dev, the
    backend matcher (/api, /api/*, /ws, /health) must be handled before the
    catch-all that reverse-proxies to the Vite dev server.
    """
    if mode == Mode.PROD:
        site_block = f"""{site.origin} {{
	encode zstd gzip
	reverse_proxy localhost:{ports.server}
}}"""
    else:
        site_block = f"""{site.origin} {{
	@backend path /api /api/* /ws /health
	handle @backend {{
		reverse_proxy localhost:{ports.server}
	}}
	handle {{
		reverse_proxy localhost:{ports.web_dev}
	}}
}}"""

    return f"""{MANAGED_START}
{site.opposite_origin} {{
	redir {site.origin}{{uri}} permanent
}}

{site_block}
{MANAGED_END}
"""


def backup_caddyfile() -> Path | None:
    """Copy ~/Caddyfile to .self-host/Caddyfile.bak.<ts> before editing it.

    Other sites may share this file, so we never touch it without a backup
    we can restore from if `caddy validate` then fails.
    """
    if not CADDYFILE.exists():
        return None
    CONFIG_DIR.mkdir(exist_ok=True)
    ts = time.strftime("%Y%m%dT%H%M%S")
    backup_path = CONFIG_DIR / f"Caddyfile.bak.{ts}"
    shutil.copy2(CADDYFILE, backup_path)
    return backup_path


def update_caddyfile(site: Site, mode: Mode, ports: Ports) -> None:
    block = render_caddy_block(site, mode, ports)
    existing = CADDYFILE.read_text() if CADDYFILE.exists() else ""
    start = existing.find(MANAGED_START)
    end = existing.find(MANAGED_END)
    if (start == -1) != (end == -1):
        raise RuntimeError(f"{CADDYFILE} contains only one Ajayeb managed marker")
    if start == -1:
        text = existing.rstrip() + ("\n\n" if existing.strip() else "") + block
    else:
        end += len(MANAGED_END)
        text = existing[:start].rstrip() + "\n\n" + block + existing[end:].lstrip("\n")
    CADDYFILE.write_text(text)


def run(cmd: list[str], *, cwd: Path = ROOT, check: bool = True) -> subprocess.CompletedProcess[str]:
    print("+", " ".join(cmd))
    return subprocess.run(cmd, cwd=cwd, check=check, text=True)


def ensure_tools(names: list[str]) -> None:
    missing = [name for name in names if shutil.which(name) is None]
    if missing:
        raise RuntimeError("Missing required command(s): " + ", ".join(missing))


def find_go() -> str:
    found = shutil.which("go")
    if found:
        return found
    for candidate in ("/usr/local/go/bin/go", "/opt/homebrew/bin/go"):
        if Path(candidate).exists():
            return candidate
    raise RuntimeError("go binary not found (checked PATH, /usr/local/go/bin, /opt/homebrew/bin)")


def port_free(port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(0.2)
        return sock.connect_ex(("127.0.0.1", port)) != 0


def find_free_port(
    preferred: int,
    *,
    reserved: set[int] | None = None,
    is_free: Callable[[int], bool] = port_free,
) -> int:
    reserved = reserved or set()
    for port in range(preferred, 65536):
        if port not in reserved and is_free(port):
            return port
    raise RuntimeError(f"No free port found at or above {preferred}")


def allocate_ports(preferred: Ports, *, is_free: Callable[[int], bool] = port_free) -> Ports:
    server = find_free_port(preferred.server, is_free=is_free)
    web_dev = find_free_port(preferred.web_dev, reserved={server}, is_free=is_free)
    return Ports(server=server, web_dev=web_dev)


def tmux_kill(session: str) -> None:
    subprocess.run(["tmux", "kill-session", "-t", session], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def stop_sessions() -> None:
    for session in (PROD_SESSION, DEV_SERVER_SESSION, DEV_WEB_SESSION):
        tmux_kill(session)


def tmuxnew(session: str, command: str, *, cwd: Path, env_args: list[str] | None = None) -> None:
    tmux_kill(session)
    args = ["tmux", "new", "-d", "-s", session]
    args.extend(env_args or [])
    args.extend(["-c", str(cwd), command])
    run(args)


def nvmrc_version() -> str:
    nvmrc = ROOT / ".nvmrc"
    return nvmrc.read_text().strip() if nvmrc.exists() else DEFAULT_NODE_VERSION


def node_command(command: str) -> str:
    version = nvmrc_version()
    node_setup = (
        "command -v nvm-load >/dev/null 2>&1 "
        "|| { echo 'nvm-load is required for web commands' >&2; exit 1; }"
        f" && nvm-load && nvm use {version}"
    )
    return node_setup + " && " + command


def shell_with_node(command: str) -> str:
    return f"zsh -lc {json.dumps(node_command(command))}"


def pnpm_install_command(*, dedupe: bool = False) -> str:
    command = "CI=true pnpm install --frozen-lockfile"
    if dedupe:
        command += " && CI=true pnpm dedupe"
    return command


def install_web(*, dedupe: bool = False) -> None:
    # The workspace root holds the lockfile; pnpm install there pulls in web/.
    run(["zsh", "-lc", node_command(pnpm_install_command(dedupe=dedupe))], cwd=ROOT)


def build_web() -> None:
    run(["zsh", "-lc", node_command("pnpm -C web build")], cwd=ROOT)


def sync_webdist() -> None:
    WEBDIST_DIST.mkdir(parents=True, exist_ok=True)
    run(["rsync", "-a", "--delete", "--exclude", ".gitkeep", f"{WEB_BUILD_DIR}/", f"{WEBDIST_DIST}/"])


def git_version() -> str:
    result = subprocess.run(
        ["git", "describe", "--always", "--dirty"], cwd=ROOT, capture_output=True, text=True, check=False
    )
    version = result.stdout.strip()
    return version if result.returncode == 0 and version else "dev"


def go_build_release_command(version: str) -> list[str]:
    return [
        find_go(),
        "build",
        "-tags",
        "embedweb",
        "-trimpath",
        "-ldflags",
        f"-s -w -X main.version={version}",
        "-o",
        str(BIN_NEW_PATH),
        "./cmd/ajayeb",
    ]


def build_server_release() -> None:
    BIN_DIR.mkdir(parents=True, exist_ok=True)
    if BIN_NEW_PATH.exists():
        BIN_NEW_PATH.unlink()
    run(go_build_release_command(git_version()), cwd=SERVER_DIR)
    BIN_NEW_PATH.chmod(0o755)
    # Atomic rename: the server reads BIN_PATH wholesale, never a half-written file.
    BIN_NEW_PATH.replace(BIN_PATH)


def validate_caddyfile() -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["caddy", "validate", "--config", str(CADDYFILE)], text=True, capture_output=True, check=False
    )


def reload_caddy() -> None:
    run(["caddy", "fmt", "--overwrite", str(CADDYFILE)])
    run(["caddy", "reload", "--config", str(CADDYFILE)])


def apply_caddyfile_safely(site: Site, mode: Mode, ports: Ports) -> None:
    """Back up ~/Caddyfile, write the managed block, validate, then reload.

    Restores the backup and raises if `caddy validate` fails, since
    ~/Caddyfile is shared with other sites and must never be left broken.
    """
    backup = backup_caddyfile()
    update_caddyfile(site, mode, ports)
    validated = validate_caddyfile()
    if validated.returncode != 0:
        if backup is not None:
            shutil.copy2(backup, CADDYFILE)
        elif CADDYFILE.exists():
            CADDYFILE.unlink()
        raise RuntimeError(f"caddy validate failed, restored previous Caddyfile:\n{validated.stderr}")
    reload_caddy()


def announce_serving(config: Config) -> None:
    print(f"serving: {config.site.origin}")


def poll_health(port: int, *, timeout: float = HEALTH_POLL_TIMEOUT, interval: float = HEALTH_POLL_INTERVAL) -> bool:
    deadline = time.monotonic() + timeout
    url = f"http://127.0.0.1:{port}/health"
    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=1.0) as resp:
                if resp.status == 200:
                    return True
        except (urllib.error.URLError, OSError, TimeoutError):
            pass
        time.sleep(interval)
    return False


def server_prod_command(ports: Ports, data_dir: Path, site: Site) -> list[str]:
    return [
        str(BIN_PATH),
        "--addr",
        f"127.0.0.1:{ports.server}",
        "--data",
        str(data_dir),
        "--origin",
        site.origin,
        "--trust-proxy",
        TRUST_PROXY_PROD,
    ]


def server_prod_env() -> dict[str, str]:
    return {"GOMEMLIMIT": GOMEMLIMIT_PROD}


def start_server_prod(ports: Ports, site: Site) -> None:
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    cmd = server_prod_command(ports, DATA_DIR, site)
    env_pairs = {**server_prod_env(), **{k: v for k, v in os.environ.items() if k in PROXY_ENV_NAMES}}
    env_args: list[str] = []
    for key, value in env_pairs.items():
        env_args.extend(["-e", f"{key}={value}"])
    tmuxnew(PROD_SESSION, " ".join(cmd), cwd=ROOT, env_args=env_args)


def start_server_dev(ports: Ports) -> None:
    # go tool air rebuilds ./cmd/ajayeb on save (server/.air.toml, frozen and
    # not owned by this slice); its entrypoint binds the default address, so
    # the dev server port tracks DEFAULT_SERVER_PORT rather than the
    # allocated one. We still record the allocated port for status/config.
    tmuxnew(DEV_SERVER_SESSION, "go tool air", cwd=SERVER_DIR, env_args=tmux_env_args())


def web_dev_command(ports: Ports) -> tuple[str, dict[str, str]]:
    command = f"pnpm -C web dev --host {web_dev_host()} --port {ports.web_dev}"
    return command, {"AJAYEB_SERVER": f"http://127.0.0.1:{ports.server}"}


def start_web_dev(ports: Ports) -> None:
    command, env = web_dev_command(ports)
    tmuxnew(DEV_WEB_SESSION, shell_with_node(command), cwd=ROOT, env_args=tmux_env_args(os.environ | env))


def resolve_config(url: str | None, fallback_mode: Mode = Mode.PROD) -> Config:
    if url:
        existing = load_config()
        return Config(parse_site_url(url), fallback_mode, existing.ports if existing else Ports())
    existing = load_config()
    if existing:
        return Config(existing.site, fallback_mode, existing.ports)
    return Config(parse_site_url(None), fallback_mode)


def resolve_runtime_config(url: str | None, mode: Mode) -> Config:
    config = resolve_config(url, mode)
    return Config(config.site, config.mode, allocate_ports(config.ports))


def archive_rooms() -> Path | None:
    """Archive .self-host/data/rooms to .self-host/backups/rooms-<ts>.tar.gz.

    Keeps only the 10 most recent archives.
    """
    rooms_dir = DATA_DIR / "rooms"
    if not rooms_dir.exists() or not any(rooms_dir.iterdir()):
        return None
    BACKUPS_DIR.mkdir(parents=True, exist_ok=True)
    ts = time.strftime("%Y%m%dT%H%M%S")
    archive_path = BACKUPS_DIR / f"rooms-{ts}.tar.gz"
    with tarfile.open(archive_path, "w:gz") as tar:
        tar.add(rooms_dir, arcname="rooms")
    prune_room_backups()
    return archive_path


def prune_room_backups(*, keep: int = ROOMS_BACKUP_KEEP) -> None:
    if not BACKUPS_DIR.exists():
        return
    archives = sorted(BACKUPS_DIR.glob("rooms-*.tar.gz"), key=lambda p: p.name)
    for stale in archives[:-keep] if keep > 0 else archives:
        stale.unlink(missing_ok=True)


def command_setup(url: str | None) -> None:
    command_stop()
    config = resolve_runtime_config(url, Mode.PROD)
    ensure_tools(["caddy", "tmux", "zsh", "rsync"])
    find_go()
    install_web(dedupe=True)
    build_web()
    sync_webdist()
    build_server_release()
    save_config(config)
    apply_caddyfile_safely(config.site, Mode.PROD, config.ports)
    start_server_prod(config.ports, config.site)
    if not poll_health(config.ports.server):
        print("warning: /health did not respond within the poll window", file=sys.stderr)
    announce_serving(config)


def command_redeploy(url: str | None) -> None:
    command_stop()
    config = resolve_runtime_config(url, Mode.PROD)
    ensure_tools(["caddy", "tmux", "zsh", "rsync"])
    find_go()
    archived = archive_rooms()
    if archived is not None:
        print(f"archived rooms: {archived}")
    install_web()
    build_web()
    sync_webdist()
    build_server_release()
    save_config(config)
    apply_caddyfile_safely(config.site, Mode.PROD, config.ports)
    start_server_prod(config.ports, config.site)
    if not poll_health(config.ports.server):
        print("warning: /health did not respond within the poll window", file=sys.stderr)
    announce_serving(config)


def command_start(url: str | None) -> None:
    command_stop()
    config = resolve_runtime_config(url, Mode.PROD)
    ensure_tools(["caddy", "tmux"])
    if not BIN_PATH.exists():
        ensure_tools(["zsh", "rsync"])
        find_go()
        install_web()
        build_web()
        sync_webdist()
        build_server_release()
    save_config(config)
    apply_caddyfile_safely(config.site, Mode.PROD, config.ports)
    start_server_prod(config.ports, config.site)
    if not poll_health(config.ports.server):
        print("warning: /health did not respond within the poll window", file=sys.stderr)
    announce_serving(config)


def command_dev_start(url: str | None) -> None:
    command_stop()
    config = resolve_runtime_config(url, Mode.DEV)
    ensure_tools(["caddy", "tmux", "zsh"])
    find_go()
    install_web()
    save_config(config)
    apply_caddyfile_safely(config.site, Mode.DEV, config.ports)
    start_server_dev(config.ports)
    start_web_dev(config.ports)
    announce_serving(config)


def command_stop() -> None:
    ensure_tools(["tmux"])
    stop_sessions()


def tmux_tail(session: str, lines: int = 20) -> str | None:
    if shutil.which("tmux") is None:
        return None
    found = (
        subprocess.run(
            ["tmux", "has-session", "-t", session], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
        ).returncode
        == 0
    )
    if not found:
        return None
    result = subprocess.run(
        ["tmux", "capture-pane", "-p", "-S", f"-{lines}", "-t", session], text=True, capture_output=True
    )
    return result.stdout if result.returncode == 0 else None


def command_status() -> None:
    config = load_config()
    print(f"config: {CONFIG_FILE if config else 'not written'}")
    if config:
        print(f"url: {config.site.origin}")
        print(f"mode: {config.mode.value}")
        print(f"server port: {config.ports.server}")
        print(f"web dev port: {config.ports.web_dev}")
    for session in (PROD_SESSION, DEV_SERVER_SESSION, DEV_WEB_SESSION):
        if shutil.which("tmux") is None:
            print(f"tmux {session}: tmux not installed")
            continue
        found = (
            subprocess.run(
                ["tmux", "has-session", "-t", session], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
            ).returncode
            == 0
        )
        print(f"tmux {session}: {'running' if found else 'stopped'}")
        if found:
            tail = tmux_tail(session)
            if tail:
                print(f"--- last 20 lines of {session} ---")
                print(tail.rstrip("\n"))
                print("---")
    ports = config.ports if config else Ports()
    for port in (ports.server, ports.web_dev):
        print(f"port {port}: {'free' if port_free(port) else 'in use'}")
    if shutil.which("curl") is None:
        print("server health: curl not installed")
    else:
        health = subprocess.run(
            ["curl", "-fsS", f"http://127.0.0.1:{ports.server}/health"], text=True, capture_output=True
        )
        print(f"server health: {health.stdout.strip() if health.returncode == 0 else 'unavailable'}")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Self-host Ajayeb with Caddy and tmux.")
    parser.add_argument("command", choices=["setup", "redeploy", "start", "stop", "dev-start", "status"])
    parser.add_argument("url", nargs="?", help=f"Site origin, default {DEFAULT_URL}")
    args = parser.parse_args(argv)

    try:
        if args.command == "setup":
            command_setup(args.url)
        elif args.command == "redeploy":
            command_redeploy(args.url)
        elif args.command == "start":
            command_start(args.url)
        elif args.command == "dev-start":
            command_dev_start(args.url)
        elif args.command == "stop":
            command_stop()
        elif args.command == "status":
            command_status()
    except Exception as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
