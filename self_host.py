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
import shlex
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
BIN_PREV_PATH = BIN_DIR / "ajayeb.prev"
DATA_DIR = CONFIG_DIR / "data"
BACKUPS_DIR = CONFIG_DIR / "backups"
DIST_DIR = CONFIG_DIR / "dist"
ROOMS_BACKUP_KEEP = 10
CADDYFILE = Path.home() / "Caddyfile"
DEFAULT_URL = "https://ajayeb.pinky.lilf.ir"
DEFAULT_NODE_VERSION = "24"
HEALTH_POLL_TIMEOUT = 10.0
HEALTH_POLL_INTERVAL = 0.3

# ship: cross-compile on this machine and install a prebuilt binary remotely.
HEAVY_SCRIPT = ROOT / "tools" / "heavy.py"
HEAVY_REFUSED = 75
DEFAULT_REMOTE_DIR = "ajayeb"
REMOTE_UPLOAD_BIN = ".self-host/bin/ajayeb.upload"
SHIP_BUILD_WAIT = 1800.0
SHIP_HEALTH_POLL_TIMEOUT = 30.0
SHIP_HEALTH_POLL_INTERVAL = 1.0
PORT_RELEASE_TIMEOUT = 5.0
UNAME_TO_GOOS_GOARCH: dict[tuple[str, str], tuple[str, str]] = {
    ("Linux", "x86_64"): ("linux", "amd64"),
    ("Linux", "aarch64"): ("linux", "arm64"),
}

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
class ShipTarget:
    """The remembered ship destination: an SSH target and a remote directory."""

    target: str
    remote_dir: str = DEFAULT_REMOTE_DIR

    def to_json(self) -> dict[str, str]:
        return {"target": self.target, "remote_dir": self.remote_dir}

    @classmethod
    def from_json(cls, raw: object) -> "ShipTarget | None":
        if not isinstance(raw, dict):
            return None
        target = raw.get("target")
        if not target:
            return None
        return cls(target=str(target), remote_dir=str(raw.get("remote_dir") or DEFAULT_REMOTE_DIR))


@dataclass(frozen=True)
class Config:
    site: Site
    mode: Mode
    ports: Ports = Ports()
    ship: ShipTarget | None = None

    def to_json(self) -> str:
        payload: dict[str, object] = {
            "url": self.site.origin,
            "mode": self.mode.value,
            "ports": self.ports.to_json(),
        }
        if self.ship is not None:
            payload["ship"] = self.ship.to_json()
        return json.dumps(payload, indent=2, sort_keys=True)

    @classmethod
    def from_json(cls, payload: str) -> "Config":
        raw = json.loads(payload)
        return cls(
            site=parse_site_url(raw.get("url")),
            mode=Mode(raw.get("mode", Mode.PROD.value)),
            ports=Ports.from_json(raw.get("ports")),
            ship=ShipTarget.from_json(raw.get("ship")),
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


def find_managed_block(text: str) -> tuple[int, int] | None:
    """Return (start, end) of the managed block in text, end just past MANAGED_END.

    None means text has no managed block yet. Raises if exactly one marker is
    present, which means the file was hand-edited into a broken state.
    """
    start = text.find(MANAGED_START)
    end = text.find(MANAGED_END)
    if (start == -1) != (end == -1):
        raise RuntimeError(f"{CADDYFILE} contains only one Ajayeb managed marker")
    if start == -1:
        return None
    return start, end + len(MANAGED_END)


def caddyfile_block_matches(site: Site, mode: Mode, ports: Ports) -> bool:
    """True when ~/Caddyfile already has exactly the managed block we would
    write for site/mode/ports, so writing it again would be a no-op."""
    if not CADDYFILE.exists():
        return False
    existing = CADDYFILE.read_text()
    bounds = find_managed_block(existing)
    if bounds is None:
        return False
    start, end = bounds
    return existing[start:end] == render_caddy_block(site, mode, ports).rstrip("\n")


def update_caddyfile(site: Site, mode: Mode, ports: Ports) -> None:
    block = render_caddy_block(site, mode, ports)
    existing = CADDYFILE.read_text() if CADDYFILE.exists() else ""
    bounds = find_managed_block(existing)
    if bounds is None:
        text = existing.rstrip() + ("\n\n" if existing.strip() else "") + block
    else:
        start, end = bounds
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


def build_server_release() -> Path:
    """Compile the release binary to BIN_NEW_PATH. Does not touch BIN_PATH.

    Compiling and swapping are separate steps so the running server is only
    stopped right before the swap (see swap_in_binary): a failing compile
    never takes it down.
    """
    BIN_DIR.mkdir(parents=True, exist_ok=True)
    if BIN_NEW_PATH.exists():
        BIN_NEW_PATH.unlink()
    run(go_build_release_command(git_version()), cwd=SERVER_DIR)
    BIN_NEW_PATH.chmod(0o755)
    return BIN_NEW_PATH


def check_prebuilt_binary(path: Path) -> None:
    """Refuse a --prebuilt path that is not a runnable executable.

    Called before anything is stopped, so a bad --prebuilt path never takes
    down a running server.
    """
    if not path.is_file():
        raise RuntimeError(f"--prebuilt {path} does not exist or is not a regular file")
    if not os.access(path, os.X_OK):
        raise RuntimeError(f"--prebuilt {path} is not executable")
    result = subprocess.run([str(path), "--version"], capture_output=True, text=True, check=False)
    version = result.stdout.strip()
    if result.returncode != 0 or not version:
        detail = result.stderr.strip() or version or f"exit {result.returncode}"
        raise RuntimeError(f"--prebuilt {path} --version did not print a version: {detail}")


def swap_in_binary(source: Path, *, move: bool = False) -> None:
    """Back up the current binary to .prev, then atomically replace it with source.

    source is installed into BIN_NEW_PATH (unless it already is BIN_NEW_PATH,
    the normal case right after build_server_release) and then renamed into
    BIN_PATH, so a half-written copy never becomes the live binary.

    By default source is copied, leaving the caller's own file (which may be
    an arbitrary --prebuilt path) untouched. With move=True, source is
    consumed instead: moved into place with a rename when possible, which is
    what setup/redeploy --prebuilt uses for an uploaded binary that exists
    only to be installed, so it does not sit around as a second ~8.5 MB copy
    once this returns. A rename cannot cross filesystems, so that case falls
    back to copying source and then removing it, which still leaves no
    residual copy.
    """
    BIN_DIR.mkdir(parents=True, exist_ok=True)
    if BIN_PATH.exists():
        shutil.copy2(BIN_PATH, BIN_PREV_PATH)
    source = Path(source)
    if source.resolve() != BIN_NEW_PATH.resolve():
        if BIN_NEW_PATH.exists():
            BIN_NEW_PATH.unlink()
        if move:
            try:
                os.replace(source, BIN_NEW_PATH)
            except OSError:
                shutil.copy2(source, BIN_NEW_PATH)
                source.unlink()
        else:
            shutil.copy2(source, BIN_NEW_PATH)
    BIN_NEW_PATH.chmod(0o755)
    BIN_NEW_PATH.replace(BIN_PATH)


def validate_caddyfile() -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["caddy", "validate", "--config", str(CADDYFILE)], text=True, capture_output=True, check=False
    )


def reload_caddy() -> None:
    """Ask the running Caddy to reload ~/Caddyfile.

    This never runs `caddy fmt --overwrite`: ~/Caddyfile is shared with other
    sites, and reformatting the whole file on every deploy is not ours to
    do. render_caddy_block already renders our block in Caddy's own
    tab-indented format, so there is nothing of ours for a reload to
    complain about.
    """
    run(["caddy", "reload", "--config", str(CADDYFILE)])


def apply_caddyfile_safely(site: Site, mode: Mode, ports: Ports) -> None:
    """Back up ~/Caddyfile, write the managed block, validate, then reload.

    If ~/Caddyfile already has exactly the managed block we would write,
    this does nothing at all: no backup, no write, no validate, no reload.
    That is what makes `start` safe to call at boot, before Caddy is
    necessarily listening yet to take a reload.

    ~/Caddyfile is shared with other sites, so a `caddy validate` failure
    restores the backup and raises: the file must never be left broken. A
    `caddy reload` failure does not raise; it only warns and continues,
    because Caddy reads ~/Caddyfile itself the next time it starts, so a
    reload that fails now (for example because Caddy is not up yet) does not
    need to block the rest of `start`/`setup`/`redeploy`.
    """
    if caddyfile_block_matches(site, mode, ports):
        return
    backup = backup_caddyfile()
    update_caddyfile(site, mode, ports)
    validated = validate_caddyfile()
    if validated.returncode != 0:
        if backup is not None:
            shutil.copy2(backup, CADDYFILE)
        elif CADDYFILE.exists():
            CADDYFILE.unlink()
        raise RuntimeError(f"caddy validate failed, restored previous Caddyfile:\n{validated.stderr}")
    try:
        reload_caddy()
    except subprocess.CalledProcessError as exc:
        print(
            f"warning: caddy reload failed ({exc}); Caddy will read {CADDYFILE} on its own next start",
            file=sys.stderr,
        )


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


def wait_for_port_free(
    port: int,
    *,
    timeout: float = PORT_RELEASE_TIMEOUT,
    is_free: Callable[[int], bool] = port_free,
) -> bool:
    deadline = time.monotonic() + timeout
    while not is_free(port):
        if time.monotonic() >= deadline:
            return False
        time.sleep(0.1)
    return True


def ports_after_stop(config: Config) -> Config:
    """Choose the ports once our own server has stopped.

    Before the stop, the running server's own port looks busy, so choosing
    then moved every redeploy to the next free port and rewrote the shared
    Caddyfile each time. tmux kill-session returns before the server has
    closed its socket, so the old port gets a moment to free up first.
    """
    wait_for_port_free(config.ports.server)
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


# --- ship: build here, install a prebuilt binary on a remote host ----------
#
# The production host is small and shared, so it must never build: `ship`
# cross-compiles on this (dev) machine and uploads one static binary plus
# self_host.py itself, then runs `setup --prebuilt` or `redeploy --prebuilt`
# over ssh on the remote side.


def parse_uname(output: str) -> tuple[str, str]:
    """Map `uname -sm` output to a (GOOS, GOARCH) pair, or raise."""
    parts = output.split()
    if len(parts) != 2:
        raise ValueError(f"unexpected 'uname -sm' output: {output!r}")
    mapped = UNAME_TO_GOOS_GOARCH.get((parts[0], parts[1]))
    if mapped is None:
        raise ValueError(
            f"unsupported remote platform {parts[0]} {parts[1]}: "
            "ship only supports Linux x86_64 and Linux aarch64"
        )
    return mapped


def ssh_command(target: str, remote_command: str) -> list[str]:
    return ["ssh", target, remote_command]


def remote_uname_command(target: str) -> list[str]:
    return ssh_command(target, "uname -sm")


def detect_remote_platform(target: str) -> tuple[str, str]:
    result = subprocess.run(remote_uname_command(target), capture_output=True, text=True, check=True)
    return parse_uname(result.stdout.strip())


def dist_binary_path(goos: str, goarch: str) -> Path:
    return DIST_DIR / f"ajayeb-{goos}-{goarch}"


def go_build_cross_env(goos: str, goarch: str) -> dict[str, str]:
    return {"CGO_ENABLED": "0", "GOOS": goos, "GOARCH": goarch}


def go_build_cross_command(version: str, goos: str, goarch: str) -> list[str]:
    return [
        find_go(),
        "build",
        "-tags",
        "embedweb",
        "-trimpath",
        "-ldflags",
        f"-s -w -X main.version={version}",
        "-o",
        str(dist_binary_path(goos, goarch)),
        "./cmd/ajayeb",
    ]


def heavy_wrap(cmd: list[str], *, kind: str, wait: float = SHIP_BUILD_WAIT) -> list[str]:
    """Wrap cmd in tools/heavy.py's guard when this checkout has one.

    The production host never gets tools/heavy.py (ship only rsyncs
    self_host.py and the binary there), so this is a no-op on the remote
    side and only ever wraps the build steps run here, on the dev machine.
    """
    if not HEAVY_SCRIPT.exists():
        return cmd
    return [sys.executable, str(HEAVY_SCRIPT), "run", "--kind", kind, "--wait", str(wait), "--", *cmd]


def run_build_step(
    cmd: list[str], *, cwd: Path = ROOT, env: dict[str, str] | None = None, kind: str = "build"
) -> None:
    """Run one ship build step, refusing clearly if tools/heavy.py's guard refuses it.

    Exit code 75 from the guard means "wait and retry", not "this command is
    broken", so it gets its own error message instead of a bare
    CalledProcessError.
    """
    wrapped = heavy_wrap(cmd, kind=kind)
    full_env = {**os.environ, **env} if env else None
    print("+", " ".join(wrapped))
    result = subprocess.run(wrapped, cwd=cwd, env=full_env, text=True)
    if result.returncode == HEAVY_REFUSED and wrapped is not cmd:
        raise RuntimeError(
            f"tools/heavy.py refused the {kind} step (exit 75): memory pressure or a busy slot; wait and retry"
        )
    if result.returncode != 0:
        raise subprocess.CalledProcessError(result.returncode, wrapped)


def cross_build(version: str, goos: str, goarch: str) -> Path:
    """Build the production web bundle and a static Go binary for goos/goarch."""
    run_build_step(["zsh", "-lc", node_command(pnpm_install_command())])
    run_build_step(["zsh", "-lc", node_command("pnpm -C web build")])
    sync_webdist()
    DIST_DIR.mkdir(parents=True, exist_ok=True)
    out_path = dist_binary_path(goos, goarch)
    if out_path.exists():
        out_path.unlink()
    run_build_step(
        go_build_cross_command(version, goos, goarch), cwd=SERVER_DIR, env=go_build_cross_env(goos, goarch)
    )
    out_path.chmod(0o755)
    return out_path


def remote_bin_dir(remote_dir: str) -> str:
    return f"{remote_dir}/.self-host/bin"


def remote_mkdir_command(target: str, remote_dir: str) -> list[str]:
    return ssh_command(target, f"mkdir -p {shlex.quote(remote_bin_dir(remote_dir))}")


def rsync_self_host_command(target: str, remote_dir: str) -> list[str]:
    local = str(ROOT / "self_host.py")
    return ["rsync", "-a", local, f"{target}:{shlex.quote(remote_dir)}/"]


def rsync_binary_command(target: str, remote_dir: str, binary_path: Path) -> list[str]:
    dest = f"{target}:{shlex.quote(remote_dir)}/{REMOTE_UPLOAD_BIN}"
    return ["rsync", "-a", str(binary_path), dest]


def upload_to_remote(target: str, remote_dir: str, binary_path: Path) -> None:
    run(remote_mkdir_command(target, remote_dir))
    run(rsync_self_host_command(target, remote_dir))
    run(rsync_binary_command(target, remote_dir, binary_path))


def remote_config_exists_command(target: str, remote_dir: str) -> list[str]:
    remote_path = f"{remote_dir}/.self-host/config.json"
    return ssh_command(target, f"test -e {shlex.quote(remote_path)}")


def determine_deploy_command(target: str, remote_dir: str) -> str:
    """'redeploy' when the remote already has a config, else 'setup'.

    `test -e` exits 1 when the file is missing; any other failure (ssh exits
    255 when it cannot connect) is an error, not a reason to run setup.
    """
    result = subprocess.run(remote_config_exists_command(target, remote_dir), capture_output=True, text=True)
    if result.returncode == 0:
        return "redeploy"
    if result.returncode == 1:
        return "setup"
    detail = result.stderr.strip() or f"exit {result.returncode}"
    raise RuntimeError(f"could not check for a remote config on {target}: {detail}")


def remote_deploy_command(target: str, remote_dir: str, deploy_command: str, url: str | None) -> list[str]:
    parts = [
        "cd",
        shlex.quote(remote_dir),
        "&&",
        "python3",
        "self_host.py",
        deploy_command,
        "--prebuilt",
        shlex.quote(REMOTE_UPLOAD_BIN),
    ]
    if url:
        parts.append(shlex.quote(url))
    return ssh_command(target, " ".join(parts))


def deploy_remote(target: str, remote_dir: str, deploy_command: str, url: str | None) -> None:
    run(remote_deploy_command(target, remote_dir, deploy_command, url))


def poll_remote_version(
    url: str, *, timeout: float = SHIP_HEALTH_POLL_TIMEOUT, interval: float = SHIP_HEALTH_POLL_INTERVAL
) -> str | None:
    """Poll <url>/health and return its reported version, or None on timeout."""
    deadline = time.monotonic() + timeout
    health_url = url.rstrip("/") + "/health"
    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(health_url, timeout=2.0) as resp:
                if resp.status == 200:
                    payload = json.loads(resp.read().decode("utf-8"))
                    version = payload.get("version")
                    return str(version) if version else None
        except (urllib.error.URLError, OSError, TimeoutError, ValueError):
            pass
        time.sleep(interval)
    return None


def remember_ship_target(target: str, remote_dir: str) -> None:
    existing = load_config()
    ship = ShipTarget(target, remote_dir)
    if existing is None:
        config = Config(site=parse_site_url(None), mode=Mode.PROD, ship=ship)
    else:
        config = Config(existing.site, existing.mode, existing.ports, ship)
    save_config(config)


def validate_ship_target(target: str, remote_dir: str) -> None:
    """Refuse a TARGET or DIR that ssh or rsync could misread.

    A leading '-' would be parsed as an ssh or rsync option. DIR must also be
    shell-safe as written: rsync 3.2.4 and later protect remote arguments by
    default, so a shell-quoted DIR would reach the remote with its quotes as
    part of the name, while older rsync passes it through a shell.
    """
    if not target or target.startswith("-") or any(c.isspace() for c in target):
        raise RuntimeError(f"invalid ship TARGET {target!r}: use an SSH destination such as user@example.host")
    if remote_dir.startswith("-") or not all(c.isascii() and (c.isalnum() or c in "._-/") for c in remote_dir or "\0"):
        raise RuntimeError(
            f"invalid --remote-dir {remote_dir!r}: use letters, digits and . _ - / only, relative to the remote home"
        )


def resolve_ship_target(target: str | None, remote_dir: str | None) -> ShipTarget:
    existing = load_config()
    remembered = existing.ship if existing else None
    if target is None:
        if remembered is None:
            raise RuntimeError("no TARGET given and none remembered yet; run: self_host.py ship user@example.host")
        target = remembered.target
    if remote_dir is None:
        remote_dir = remembered.remote_dir if remembered else DEFAULT_REMOTE_DIR
    validate_ship_target(target, remote_dir)
    return ShipTarget(target, remote_dir)


def looks_like_url(value: str) -> bool:
    return value.startswith("http://") or value.startswith("https://")


def command_ship(target: str | None, url: str | None, remote_dir: str | None) -> None:
    if target is not None and looks_like_url(target):
        # `ship URL` with a remembered target: argparse puts URL in the
        # TARGET slot since it's positional and comes first. Without this,
        # TARGET would be used as the SSH destination and fail (ssh exit
        # 255) instead of reusing the remembered target.
        if url is not None:
            raise RuntimeError(
                f"ship got a url-looking TARGET ({target!r}) and a URL ({url!r}); "
                "pass only the url when a target is already remembered"
            )
        target, url = None, target

    resolved = resolve_ship_target(target, remote_dir)
    target, remote_dir = resolved.target, resolved.remote_dir

    goos, goarch = detect_remote_platform(target)
    print(f"remote platform: {goos}/{goarch}")

    version = git_version()
    binary_path = cross_build(version, goos, goarch)
    print(f"built {binary_path} ({version})")

    upload_to_remote(target, remote_dir, binary_path)

    deploy_command = determine_deploy_command(target, remote_dir)
    print(f"remote command: {deploy_command}")
    deploy_remote(target, remote_dir, deploy_command, url)

    remember_ship_target(target, remote_dir)

    if not url:
        print("no url given; skipping the post-deploy health poll")
        return
    remote_version = poll_remote_version(url)
    if remote_version is None:
        print(f"warning: {url}/health did not respond within {SHIP_HEALTH_POLL_TIMEOUT:g}s", file=sys.stderr)
    elif remote_version != version:
        print(f"warning: remote reports version {remote_version}, expected {version}", file=sys.stderr)
    else:
        print(f"remote is running version {remote_version}")


def command_setup(url: str | None, prebuilt: str | None = None) -> None:
    config = resolve_config(url, Mode.PROD)
    ensure_tools(["caddy", "tmux"])
    prebuilt_path = Path(prebuilt).expanduser() if prebuilt else None
    if prebuilt_path is not None:
        check_prebuilt_binary(prebuilt_path)
        new_binary = prebuilt_path
    else:
        ensure_tools(["zsh", "rsync"])
        find_go()
        install_web(dedupe=True)
        build_web()
        sync_webdist()
        new_binary = build_server_release()
    # Everything above only builds and checks; nothing has been stopped yet,
    # so a failing build or a bad --prebuilt path never takes a running
    # server down. The swap below is the first thing that touches it.
    command_stop()
    config = ports_after_stop(config)
    save_config(config)
    swap_in_binary(new_binary, move=prebuilt_path is not None)
    apply_caddyfile_safely(config.site, Mode.PROD, config.ports)
    start_server_prod(config.ports, config.site)
    if not poll_health(config.ports.server):
        print("warning: /health did not respond within the poll window", file=sys.stderr)
    announce_serving(config)


def command_redeploy(url: str | None, prebuilt: str | None = None) -> None:
    config = resolve_config(url, Mode.PROD)
    ensure_tools(["caddy", "tmux"])
    prebuilt_path = Path(prebuilt).expanduser() if prebuilt else None
    if prebuilt_path is not None:
        check_prebuilt_binary(prebuilt_path)
        new_binary = prebuilt_path
    else:
        ensure_tools(["zsh", "rsync"])
        find_go()
        install_web()
        build_web()
        sync_webdist()
        new_binary = build_server_release()
    # See command_setup: stop moves to just before the swap, after every
    # build and check, so a failing one never takes the running server down.
    # The rooms archive still runs with the server already stopped (so it
    # never captures a file mid-write), and still before the swap.
    command_stop()
    config = ports_after_stop(config)
    save_config(config)
    archived = archive_rooms()
    if archived is not None:
        print(f"archived rooms: {archived}")
    swap_in_binary(new_binary, move=prebuilt_path is not None)
    apply_caddyfile_safely(config.site, Mode.PROD, config.ports)
    start_server_prod(config.ports, config.site)
    if not poll_health(config.ports.server):
        print("warning: /health did not respond within the poll window", file=sys.stderr)
    announce_serving(config)


def command_start(url: str | None) -> None:
    command_stop()
    config = ports_after_stop(resolve_config(url, Mode.PROD))
    ensure_tools(["caddy", "tmux"])
    if not BIN_PATH.exists():
        ensure_tools(["zsh", "rsync"])
        find_go()
        install_web()
        build_web()
        sync_webdist()
        swap_in_binary(build_server_release())
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


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Self-host Ajayeb with Caddy and tmux.")
    sub = parser.add_subparsers(dest="command", required=True)

    def add_url(p: argparse.ArgumentParser) -> None:
        p.add_argument("url", nargs="?", help=f"Site origin, default {DEFAULT_URL}")

    def add_prebuilt(p: argparse.ArgumentParser) -> None:
        p.add_argument(
            "--prebuilt",
            metavar="PATH",
            help="use this already-built binary instead of installing, building and syncing here",
        )

    setup_p = sub.add_parser("setup", help="first-time install")
    add_url(setup_p)
    add_prebuilt(setup_p)

    redeploy_p = sub.add_parser("redeploy", help="rebuild and restart for a new version")
    add_url(redeploy_p)
    add_prebuilt(redeploy_p)

    start_p = sub.add_parser("start", help="start the already-built server")
    add_url(start_p)

    sub.add_parser("stop", help="stop every tmux session this script manages")

    dev_p = sub.add_parser("dev-start", help="start a development setup instead of production")
    add_url(dev_p)

    status_p = sub.add_parser("status", help="print the recorded configuration and health")
    add_url(status_p)

    ship_p = sub.add_parser("ship", help="build on this machine and install a prebuilt binary on a remote host")
    ship_p.add_argument("target", nargs="?", default=None, help="SSH target, e.g. user@example.host")
    ship_p.add_argument("url", nargs="?", default=None, help="Remote site origin")
    ship_p.add_argument(
        "--remote-dir",
        default=None,
        help=f"Remote directory, relative to the remote home (default {DEFAULT_REMOTE_DIR})",
    )

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    try:
        if args.command == "setup":
            command_setup(args.url, args.prebuilt)
        elif args.command == "redeploy":
            command_redeploy(args.url, args.prebuilt)
        elif args.command == "start":
            command_start(args.url)
        elif args.command == "dev-start":
            command_dev_start(args.url)
        elif args.command == "stop":
            command_stop()
        elif args.command == "status":
            command_status()
        elif args.command == "ship":
            command_ship(args.target, args.url, args.remote_dir)
    except Exception as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
