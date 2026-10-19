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
import errno
import http.client
import json
import os
import platform
import re
import select
import shlex
import shutil
import socket
import subprocess
import sys
import tarfile
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, replace
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
DEFAULT_CADDYFILE = Path.home() / "Caddyfile"
# The Caddyfile this run edits and reloads: DEFAULT_CADDYFILE unless the
# config names another one (use_caddyfile sets both at the start of a command).
CADDYFILE = DEFAULT_CADDYFILE
CADDYFILE_NAMED = False
# Caddy's own default admin address, used when the Caddyfile names none.
CADDY_DEFAULT_ADMIN = "localhost:2019"
CADDY_LOAD_TIMEOUT = 60.0
CADDY_PROBE_TIMEOUT = 2.0
CADDY_RECHECK_DELAY = 1.0
# What Caddy's Caddyfile adapter puts in every file_server's hide list for
# the config file when it adapts text that came with no file name (the admin
# API's /adapt and /load). `caddy run --config FILE` and `caddy reload
# --config FILE` hide FILE itself there instead.
ADAPTED_TEXT_HIDE = "./Caddyfile"
DEFAULT_URL = "https://ajayeb.lilf.ir"
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
SHIP_HEALTH_REQUEST_TIMEOUT = 5.0
SHIP_HEALTH_MAX_BYTES = 16 * 1024
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
class DeployTarget:
    destination: ShipTarget
    site: Site


@dataclass(frozen=True)
class Config:
    site: Site
    mode: Mode
    ports: Ports = Ports()
    ship: ShipTarget | None = None
    # Serve the site on both http:// and https:// instead of redirecting the
    # other scheme to the site's own (render_caddy_block).
    both_schemes: bool = False
    # The Caddyfile the running Caddy reads, if not ~/Caddyfile: absolute, or
    # relative to the home directory (caddyfile_path).
    caddyfile: str | None = None

    def to_json(self) -> str:
        payload: dict[str, object] = {
            "url": self.site.origin,
            "mode": self.mode.value,
            "ports": self.ports.to_json(),
            "both_schemes": self.both_schemes,
        }
        if self.ship is not None:
            payload["ship"] = self.ship.to_json()
        if self.caddyfile is not None:
            payload["caddyfile"] = self.caddyfile
        return json.dumps(payload, indent=2, sort_keys=True)

    @classmethod
    def from_json(cls, payload: str) -> "Config":
        raw = json.loads(payload)
        return cls(
            site=parse_site_url(raw.get("url")),
            mode=Mode(raw.get("mode", Mode.PROD.value)),
            ports=Ports.from_json(raw.get("ports")),
            ship=ShipTarget.from_json(raw.get("ship")),
            both_schemes=raw.get("both_schemes") is True,
            caddyfile=str(raw["caddyfile"]) if raw.get("caddyfile") else None,
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


def caddyfile_path(value: str | None) -> Path:
    """The Caddyfile a config names: ~/Caddyfile when it names none, an
    absolute path as is, and anything else relative to the home directory
    (a leading ~/ included), so the same value means the same file whether
    it was typed here or forwarded to a remote host by ship."""
    if not value:
        return DEFAULT_CADDYFILE
    path = Path(value).expanduser()
    return path if path.is_absolute() else Path.home() / path


def use_caddyfile(config: Config) -> None:
    """Point this run's Caddy steps at the Caddyfile config names."""
    global CADDYFILE, CADDYFILE_NAMED
    CADDYFILE = caddyfile_path(config.caddyfile)
    CADDYFILE_NAMED = bool(config.caddyfile)


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


def render_caddy_block(site: Site, mode: Mode, ports: Ports = Ports(), both_schemes: bool = False) -> str:
    """Render the managed Caddyfile block.

    In prod Go serves the embedded SPA itself, so Caddy is a single
    reverse_proxy plus an HTTP-to-HTTPS redirect: no try_files. In dev, the
    backend matcher (/api, /api/*, /ws, /health) must be handled before the
    catch-all that reverse-proxies to the Vite dev server.

    With both_schemes the site block answers on both origins and there is no
    redirect, for a network that blocks or stalls one of the two ports. The
    server needs no second --origin for it: its WebSocket check accepts the
    request's own host whatever the scheme.
    """
    addresses = f"{site.origin}, {site.opposite_origin}" if both_schemes else site.origin
    if mode == Mode.PROD:
        site_block = f"""{addresses} {{
	encode zstd gzip
	reverse_proxy localhost:{ports.server}
}}"""
    else:
        site_block = f"""{addresses} {{
	@backend path /api /api/* /ws /health
	handle @backend {{
		reverse_proxy localhost:{ports.server}
	}}
	handle {{
		reverse_proxy localhost:{ports.web_dev}
	}}
}}"""

    if both_schemes:
        return f"""{MANAGED_START}
{site_block}
{MANAGED_END}
"""
    return f"""{MANAGED_START}
{site.opposite_origin} {{
	redir {site.origin}{{uri}} permanent
}}

{site_block}
{MANAGED_END}
"""


def backup_caddyfile() -> Path | None:
    """Copy the Caddyfile to .self-host/Caddyfile.bak.<ts> before editing it.

    Other sites may share this file, so we never touch it without a backup
    we can restore from if Caddy then refuses it.
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


def caddyfile_block_matches(site: Site, mode: Mode, ports: Ports, both_schemes: bool = False) -> bool:
    """True when the Caddyfile already has exactly the managed block we would
    write for site/mode/ports, so writing it again would be a no-op."""
    if not CADDYFILE.exists():
        return False
    existing = CADDYFILE.read_text()
    bounds = find_managed_block(existing)
    if bounds is None:
        return False
    start, end = bounds
    return existing[start:end] == render_caddy_block(site, mode, ports, both_schemes).rstrip("\n")


def update_caddyfile(site: Site, mode: Mode, ports: Ports, both_schemes: bool = False) -> None:
    block = render_caddy_block(site, mode, ports, both_schemes)
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


def check_candidate_rooms(candidate: Path, force_room_archive: bool = False) -> None:
    """Read-only candidate check. Force permits live archive risk, never read errors."""
    try:
        result = subprocess.run(
            [str(candidate.resolve()), "--check-rooms", "--data", str(DATA_DIR)],
            capture_output=True, text=True, check=False, timeout=120,
        )
        report = json.loads(result.stdout)
        counts = ("rooms", "lobby", "playing", "finished", "archived", "playable", "wouldArchive",
                  "liveWouldArchive", "refreshableHashes", "schemaErrors", "errors")
        if (not isinstance(report, dict) or report.get("format") != "ajayeb-room-check/1"
            or type(report.get("rules")) is not int or report["rules"] < 1
            or any(type(report.get(k)) is not int or report[k] < 0 for k in counts)
            or report["liveWouldArchive"] > report["wouldArchive"]
            or report["wouldArchive"] + report["playable"] + report["archived"] > report["rooms"]
            or report["refreshableHashes"] > report["playable"]
            or report["schemaErrors"] > report["errors"]
            or report["liveWouldArchive"] > report["lobby"] + report["playing"]
            or (report["errors"] == 0 and
                report["wouldArchive"] + report["playable"] + report["archived"] != report["rooms"])):
            raise ValueError("invalid report")
    except (OSError, ValueError, subprocess.TimeoutExpired) as exc:
        raise RuntimeError(f"candidate room check failed ({type(exc).__name__}); existing server and data preserved") from None
    print("room check: " + ", ".join(f"{k}={report[k]}" for k in counts))
    if report["errors"] or result.returncode not in (0, 3):
        raise RuntimeError("candidate room check has read/schema/operational errors; refusing deployment even with --force-room-archive")
    live = report["liveWouldArchive"]
    if (result.returncode == 3) != (live > 0):
        raise RuntimeError("candidate room check returned inconsistent status; refusing deployment")
    if live and not force_room_archive:
        raise RuntimeError(f"candidate would archive {live} in-progress room(s); refused before binary replacement. "
                           "Use --force-room-archive only to explicitly accept ending those games")
    if live:
        print(f"warning: --force-room-archive accepts archiving {live} in-progress room(s)", file=sys.stderr)


def stop_for_candidate(config: Config, candidate: Path, force_room_archive: bool) -> None:
    """Check before stopping, then again before any backup, config write or swap.

    The first scan may race active moves. If the stopped scan fails, keep the
    candidate and data untouched and restart the previous binary on its old port.
    """
    check_candidate_rooms(candidate, force_room_archive)
    previous_binary = BIN_PATH.is_file()
    command_stop()
    try:
        if not wait_for_port_free(config.ports.server):
            raise RuntimeError("previous server did not release its port; candidate not installed")
        check_candidate_rooms(candidate, force_room_archive)
    except BaseException:
        if previous_binary:
            try:
                start_server_prod(config.ports, config.site)
                if not poll_health(config.ports.server):
                    raise RuntimeError("previous server health check failed")
                print("previous binary restarted after stopped room check failed", file=sys.stderr)
            except Exception as exc:
                print(f"warning: previous binary restart failed ({type(exc).__name__}); binary and data preserved", file=sys.stderr)
        raise


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
        ["caddy", "validate", "--config", str(CADDYFILE), "--adapter", "caddyfile"],
        text=True,
        capture_output=True,
        check=False,
    )


def reload_caddy() -> None:
    """Ask the running Caddy to reload the Caddyfile with `caddy reload`.

    Only the fallback for a Caddyfile whose admin API load_through_caddy cannot
    reach (turned off, or on a Unix socket). This never runs `caddy fmt
    --overwrite`: the Caddyfile is shared with other sites, and reformatting
    the whole file on every deploy is not ours to do. render_caddy_block
    already renders our block in Caddy's own tab-indented format, so there is
    nothing of ours for a reload to complain about.
    """
    run(["caddy", "reload", "--config", str(CADDYFILE), "--adapter", "caddyfile"])


class CaddyUnreachable(Exception):
    """No Caddy answered on the admin address."""


# Connect failures that mean no Caddy listens there. A name like localhost
# can resolve to an address this host cannot use (::1 without IPv6), and
# Python reports the error from the last address it tried, which is then
# not ECONNREFUSED.
NOTHING_LISTENS = frozenset(
    {errno.ECONNREFUSED, errno.EADDRNOTAVAIL, errno.ENETUNREACH, errno.EHOSTUNREACH, errno.EINVAL, errno.EAFNOSUPPORT}
)


class CaddyRejected(Exception):
    """The running Caddy refused a request (an error status), changing nothing."""


class CaddyUnsure(Exception):
    """A request reached Caddy but its reply was lost (a reset, a timeout), so
    Caddy may or may not have acted on it."""


class CaddyMismatch(Exception):
    """The Caddy on the admin address does not serve the Caddyfile as it stood
    before our edit, so loading the edited file there is not safe."""


def admin_listen_address(listen: str) -> str | None:
    """host:port for an admin listen address; None for a Unix socket or an
    address that is still a {$VAR} placeholder."""
    if listen.startswith("tcp/"):
        listen = listen[len("tcp/") :]
    return None if "/" in listen or "{" in listen else listen


def scanned_admin_address(text: str) -> str | None:
    """The admin address the global options block of Caddyfile text names,
    read as plain text: Caddy's default when the block names none, None for
    `admin off`, a Unix socket or a placeholder.

    Only for a file that does not adapt in this shell's environment (a
    {$VAR} placeholder the shell lacks can make an option invalid here)
    while the running Caddy, which has the variable, may still load it.
    """
    depth = 0
    for raw in text.splitlines():
        line = raw.split("#", 1)[0].strip()
        if not line:
            continue
        if depth == 0:
            if line != "{":
                return CADDY_DEFAULT_ADMIN
            depth = 1
            continue
        tokens = line.split()
        if depth == 1 and tokens[0] == "admin":
            argument = tokens[1] if len(tokens) > 1 and tokens[1] != "{" else None
            if argument is None:
                return CADDY_DEFAULT_ADMIN
            return None if argument == "off" else admin_listen_address(argument)
        depth += line.count("{") - line.count("}")
        if depth <= 0:
            return CADDY_DEFAULT_ADMIN
    return CADDY_DEFAULT_ADMIN


def caddy_admin_address() -> str | None:
    """The host:port of the admin API the Caddyfile's own global options
    name, or Caddy's default when they name none: the Caddy that `caddy
    reload` would talk to. None when the file turns the admin API off or
    puts it on a Unix socket.

    `caddy adapt` reads the file the way Caddy does. When it fails here,
    which an unset {$VAR} can cause, the global options block is read as
    text instead (scanned_admin_address), so the file still goes to the
    running Caddy, which adapts it with its own environment.
    """
    adapted = subprocess.run(
        ["caddy", "adapt", "--config", str(CADDYFILE), "--adapter", "caddyfile"],
        text=True,
        capture_output=True,
        check=False,
    )
    admin = None
    if adapted.returncode == 0:
        try:
            config = json.loads(adapted.stdout)
            admin = config.get("admin") or {} if isinstance(config, dict) else None
        except ValueError:
            admin = None
    if not isinstance(admin, dict):
        return scanned_admin_address(CADDYFILE.read_text() if CADDYFILE.exists() else "")
    if admin.get("disabled"):
        return None
    return admin_listen_address(str(admin.get("listen") or CADDY_DEFAULT_ADMIN))


def caddy_admin_opener() -> urllib.request.OpenerDirector:
    """An opener that never goes through a proxy: the admin API is local,
    and a host's https_proxy or http_proxy must not catch these requests."""
    return urllib.request.build_opener(urllib.request.ProxyHandler({}))


def caddy_admin_request(
    address: str,
    path: str,
    *,
    data: bytes | None = None,
    headers: dict[str, str] | None = None,
    timeout: float = CADDY_LOAD_TIMEOUT,
) -> object:
    """One request to the admin API on address (a POST when data is given),
    with the reply parsed as JSON (None for an empty reply).

    Raises CaddyRejected with Caddy's reason for an error status,
    CaddyUnreachable when the connection is refused (no Caddy listens), and
    CaddyUnsure for any other transport failure.
    """
    request = urllib.request.Request(
        f"http://{address}{path}", data=data, method="POST" if data is not None else "GET", headers=headers or {}
    )
    try:
        with caddy_admin_opener().open(request, timeout=timeout) as resp:
            body = resp.read()
    except urllib.error.HTTPError as exc:
        with exc:
            detail = exc.read().decode("utf-8", errors="replace").strip()
        raise CaddyRejected(detail or f"HTTP {exc.code}") from exc
    except urllib.error.URLError as exc:
        # urllib wraps what fails while connecting and sending; a connect
        # failure means nothing reached a Caddy.
        if isinstance(exc.reason, OSError) and exc.reason.errno in NOTHING_LISTENS:
            raise CaddyUnreachable(str(exc.reason)) from exc
        raise CaddyUnsure(str(exc.reason)) from exc
    except ConnectionRefusedError as exc:
        raise CaddyUnreachable(str(exc)) from exc
    except (OSError, http.client.HTTPException) as exc:
        raise CaddyUnsure(str(exc) or type(exc).__name__) from exc
    return json.loads(body) if body.strip() else None


def adapt_in_caddy(address: str, text: bytes) -> object:
    """Caddyfile text as the running Caddy adapts it to JSON (/adapt), with
    Caddy's own environment. Loads nothing."""
    reply = caddy_admin_request(address, "/adapt", data=text, headers={"Content-Type": "text/caddyfile"})
    if not isinstance(reply, dict) or "result" not in reply:
        raise CaddyRejected("Caddy's /adapt gave no result")
    return reply["result"]


def hide_paths_for_adapted_text(adapted: object, running: object, found: set[str]) -> set[str]:
    """The values running has wherever adapted has ADAPTED_TEXT_HIDE in a
    file_server's hide list: the path the running Caddy hides its config
    file by."""
    if isinstance(adapted, dict) and isinstance(running, dict):
        if adapted.get("handler") == "file_server":
            ours, theirs = adapted.get("hide"), running.get("hide")
            # Caddy orders the config files in a hide list differently from
            # one version to the next, so match them as sets.
            if isinstance(ours, list) and isinstance(theirs, list) and ADAPTED_TEXT_HIDE in ours:
                found.update(t for t in set(map(str, theirs)) - set(map(str, ours)))
        for key in adapted.keys() & running.keys():
            hide_paths_for_adapted_text(adapted[key], running[key], found)
    elif isinstance(adapted, list) and isinstance(running, list) and len(adapted) == len(running):
        for ours, theirs in zip(adapted, running):
            hide_paths_for_adapted_text(ours, theirs, found)
    return found


def with_config_file_hidden(config: object, path: str) -> object:
    """config with ADAPTED_TEXT_HIDE in every file_server's hide list
    replaced by path, as `caddy reload --config` would have hidden the file."""
    if isinstance(config, list):
        return [with_config_file_hidden(item, path) for item in config]
    if not isinstance(config, dict):
        return config
    out = {key: with_config_file_hidden(value, path) for key, value in config.items()}
    if config.get("handler") == "file_server" and isinstance(config.get("hide"), list):
        out["hide"] = [path if item == ADAPTED_TEXT_HIDE else item for item in config["hide"]]
    return out


def with_hide_lists_sorted(config: object) -> object:
    """config with every file_server hide list sorted, for comparing configs
    whose hide lists hold the same paths in another order."""
    if isinstance(config, list):
        return [with_hide_lists_sorted(item) for item in config]
    if not isinstance(config, dict):
        return config
    out = {key: with_hide_lists_sorted(value) for key, value in config.items()}
    if config.get("handler") == "file_server" and isinstance(config.get("hide"), list):
        out["hide"] = sorted(config["hide"], key=str)
    return out


def same_config(a: object, b: object) -> bool:
    return with_hide_lists_sorted(a) == with_hide_lists_sorted(b)


def check_caddy_serves(address: str, before: bytes | None) -> str:
    """Raise CaddyMismatch unless the running Caddy on address serves the
    Caddyfile text before (None for a file not created yet, which needs a
    Caddy with no config). Loads nothing. Returns the path the running
    Caddy hides its config file by (see load_through_caddy)."""
    running = caddy_admin_request(address, "/config/")
    if before is None:
        if running not in (None, {}):
            raise CaddyMismatch(f"{CADDYFILE} is new, but the Caddy on {address} already serves a config")
        return str(CADDYFILE)
    adapted = adapt_in_caddy(address, before)
    found = hide_paths_for_adapted_text(adapted, running, set())
    hide_path = next(iter(found)) if len(found) == 1 else str(CADDYFILE)
    if len(found) > 1 or not same_config(with_config_file_hidden(adapted, hide_path), running):
        raise CaddyMismatch(
            f"the Caddy on {address} does not serve {CADDYFILE} as it stands: the path may be wrong "
            "(check --caddyfile), another Caddy may own that admin address, the file may import relative "
            "paths, or it may hold edits Caddy has not loaded yet (reload Caddy with it first)"
        )
    return hide_path


def load_through_caddy(address: str, before: bytes | None) -> None:
    """Have the running Caddy on address load the edited Caddyfile, but only
    if it is the Caddy that serves the file as it stood before the edit.

    Caddy adapts both versions itself (/adapt), so a {$VAR} placeholder in
    the shared file resolves from Caddy's environment, as it did when Caddy
    started, whatever the shell running this script exports. Adapted text
    hides the config file by a relative stand-in (ADAPTED_TEXT_HIDE), so the
    path Caddy really hides it by is read from the running config and put
    back. The unedited version must then equal the running config (hide
    lists compared as sets; check_caddy_serves).
    If it does not, the file is not what this Caddy serves: a wrong path,
    another Caddy on the address, imports that resolve differently here, or
    edits not loaded yet. Then this raises CaddyMismatch before loading
    anything. The edited version, with the same hide path, goes to /load as
    JSON, which Caddy applies only if all of it provisions. When the reply
    to that is lost, the running config shows whether the load took.
    """
    hide_path = check_caddy_serves(address, before)
    edited = with_config_file_hidden(adapt_in_caddy(address, CADDYFILE.read_bytes()), hide_path)
    try:
        caddy_admin_request(
            address,
            "/load",
            data=json.dumps(edited).encode(),
            headers={"Content-Type": "application/json", "Cache-Control": "must-revalidate"},
        )
    except CaddyUnsure as exc:
        # Caddy restarts its admin listener after a load, so a lost reply
        # may follow a load that took: look at what Caddy runs now. Any
        # failure here is an unknown outcome, never a reason to treat Caddy
        # as down or as having refused.
        time.sleep(CADDY_RECHECK_DELAY)
        try:
            now = caddy_admin_request(address, "/config/")
        except (CaddyUnreachable, CaddyRejected, CaddyUnsure, ValueError) as again:
            raise RuntimeError(
                f"the reply to the load was lost ({exc}), and Caddy could not be asked whether it took ({again})"
            ) from again
        if not same_config(now, edited):
            raise RuntimeError(f"the reply to the load was lost ({exc}), and Caddy does not run the edited config") from exc


def caddy_serves_a_config(address: str) -> bool:
    """True when a Caddy answers on address with a config already loaded."""
    try:
        return caddy_admin_request(address, "/config/", timeout=CADDY_PROBE_TIMEOUT) not in (None, {})
    except (CaddyUnreachable, CaddyUnsure, CaddyRejected, ValueError):
        return False


def refuse_to_create_caddyfile() -> None:
    """Stop before creating a Caddyfile when a new one could cost a running
    Caddy its other sites.

    A Caddyfile that does not exist yet would be written with our block
    alone, and loading that replaces everything the Caddy serves. So a
    missing file that --caddyfile named is taken as a wrong path, and a
    missing ~/Caddyfile is created only when no Caddy on Caddy's default
    admin address already serves a config (which it would be reading from
    some other file).
    """
    if CADDYFILE.exists():
        return
    if CADDYFILE_NAMED:
        raise RuntimeError(f"the Caddyfile {CADDYFILE} does not exist; check --caddyfile (nothing was changed)")
    if caddy_serves_a_config(CADDY_DEFAULT_ADMIN):
        raise RuntimeError(
            f"{CADDYFILE} does not exist, but the Caddy on {CADDY_DEFAULT_ADMIN} already serves a config, "
            "so it reads another file: name that file with --caddyfile (nothing was changed)"
        )


def preflight_caddyfile(site: Site, mode: Mode, ports: Ports, both_schemes: bool = False) -> None:
    """Before a command stops the game server: when the managed block will
    need writing and a running Caddy answers, check that it serves the
    Caddyfile as it stands (check_caddy_serves), so a refusal that the
    load would meet anyway comes while the game is still up. Changes
    nothing. A block that already matches, a file not created yet, or no
    Caddy answering leaves everything to apply_caddyfile_safely."""
    if caddyfile_block_matches(site, mode, ports, both_schemes) or not CADDYFILE.exists():
        return
    address = caddy_admin_address()
    if address is None:
        return
    try:
        check_caddy_serves(address, CADDYFILE.read_bytes())
    except CaddyUnreachable:
        return
    except CaddyMismatch as exc:
        raise RuntimeError(f"{exc}. Nothing was stopped or changed") from exc


def restore_caddyfile(backup: Path | None) -> None:
    if backup is not None:
        shutil.copy2(backup, CADDYFILE)
    elif CADDYFILE.exists():
        CADDYFILE.unlink()


def apply_caddyfile_safely(site: Site, mode: Mode, ports: Ports, both_schemes: bool = False) -> bool | None:
    """Back up the Caddyfile, write the managed block, then have Caddy load it.

    If the Caddyfile already has exactly the managed block we would write,
    this does nothing at all: no backup, no write, no load. That is what
    makes `start` safe to call at boot, before Caddy is necessarily
    listening yet.

    The Caddyfile is shared with other sites, so it must never be left in a
    state Caddy cannot load. When the running Caddy answers on the admin
    address the file names, and serves the file as it stood before the edit,
    it loads the edited file (load_through_caddy); if it does not serve that
    file, or refuses the edit, the backup is restored and this raises, and
    Caddy keeps its old config. When no Caddy answers, `caddy
    validate` checks the file instead (a failure restores and raises), and
    Caddy reads it on its next start. When the file turns the admin API off
    or puts it on a Unix socket, this validates and then runs `caddy
    reload`, whose failure only warns. Any other error after the write also
    restores the backup before it propagates.

    Returns True after loading, False when only validated or reload failed,
    and None when the managed block already matched.
    """
    if caddyfile_block_matches(site, mode, ports, both_schemes):
        return
    refuse_to_create_caddyfile()
    backup = backup_caddyfile()
    update_caddyfile(site, mode, ports, both_schemes)
    try:
        return load_or_validate_caddyfile(backup)
    except BaseException:
        # Whatever went wrong after the write, a refusal or a Ctrl-C
        # included, the shared file goes back to what it was before the
        # error propagates. Left behind unloaded, our block would make the
        # next run believe there is nothing to do.
        restore_caddyfile(backup)
        raise


def load_or_validate_caddyfile(backup: Path | None) -> bool:
    """The steps of apply_caddyfile_safely after the write. Raising means
    the caller restores the backup."""
    address = caddy_admin_address()
    if address is not None:
        try:
            load_through_caddy(address, backup.read_bytes() if backup is not None else None)
            return True
        except CaddyRejected as exc:
            raise RuntimeError(
                f"Caddy refused the Caddyfile and kept its running config; restored the previous Caddyfile:\n{exc}"
            ) from exc
        except CaddyMismatch as exc:
            raise RuntimeError(f"{exc}. Nothing was loaded; restored the previous Caddyfile") from exc
        except CaddyUnreachable as exc:
            print(f"warning: no Caddy answered on {address} ({exc}); checking the Caddyfile with caddy validate", file=sys.stderr)
    validated = validate_caddyfile()
    if validated.returncode != 0:
        hint = ""
        if CADDYFILE.exists() and "{$" in CADDYFILE.read_text():
            hint = "\n(the Caddyfile uses {$VAR} placeholders: caddy validate reads them from this shell's environment)"
        raise RuntimeError(f"caddy validate failed, restored previous Caddyfile:\n{validated.stderr}{hint}")
    if address is not None:
        print(f"warning: Caddy is not running; it will read {CADDYFILE} on its own next start", file=sys.stderr)
        return False
    try:
        reload_caddy()
    except subprocess.CalledProcessError as exc:
        print(
            f"warning: caddy reload failed ({exc}); Caddy will read {CADDYFILE} on its own next start",
            file=sys.stderr,
        )
        return False
    return True


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


def resolve_config(
    url: str | None,
    fallback_mode: Mode = Mode.PROD,
    both_schemes: bool | None = None,
    caddyfile: str | None = None,
) -> Config:
    """The config to run with: the stored one, overridden by url, and by
    both_schemes and caddyfile unless they are None (None keeps the stored
    choice, so a plain redeploy or the boot-time start never drops it)."""
    existing = load_config()
    site = parse_site_url(url) if url else existing.site if existing else parse_site_url(None)
    if both_schemes is None:
        both_schemes = existing.both_schemes if existing else False
    if caddyfile is None:
        caddyfile = existing.caddyfile if existing else None
    return Config(
        site,
        fallback_mode,
        existing.ports if existing else Ports(),
        existing.ship if existing else None,
        both_schemes,
        caddyfile or None,
    )


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
    return replace(config, ports=allocate_ports(config.ports))


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


def remote_deploy_command(
    target: str,
    remote_dir: str,
    deploy_command: str,
    url: str | None,
    both_schemes: bool | None = None,
    caddyfile: str | None = None,
    force_room_archive: bool = False,
) -> list[str]:
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
    if both_schemes is not None:
        parts.append("--both-schemes" if both_schemes else "--no-both-schemes")
    if caddyfile is not None:
        parts.extend(["--caddyfile", shlex.quote(caddyfile)])
    if force_room_archive:
        parts.append("--force-room-archive")
    if url:
        parts.append(shlex.quote(url))
    return ssh_command(target, " ".join(parts))


def deploy_remote(
    target: str,
    remote_dir: str,
    deploy_command: str,
    url: str | None,
    both_schemes: bool | None = None,
    caddyfile: str | None = None,
    force_room_archive: bool = False,
) -> None:
    run(remote_deploy_command(target, remote_dir, deploy_command, url, both_schemes, caddyfile,
                              **({"force_room_archive": True} if force_room_archive else {})))


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
        config = replace(existing, ship=ship)
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


def command_ship(
    target: str | None,
    url: str | None,
    remote_dir: str | None,
    both_schemes: bool | None = None,
    caddyfile: str | None = None,
    force_room_archive: bool = False,
) -> None:
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
    deploy_remote(target, remote_dir, deploy_command, url, both_schemes, caddyfile,
                  **({"force_room_archive": True} if force_room_archive else {}))

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


def load_deploy_targets() -> list[DeployTarget]:
    """Read the entire private config before any build or remote operation.

    Errors deliberately describe fields, never their supplied values: malformed
    SSH destinations and URLs can contain passwords or other credentials.
    """
    path = Path(os.environ.get("AJAYEB_DEPLOY_TARGETS", str(CONFIG_DIR / "deploy-targets.json"))).expanduser()
    example = '{"targets": [{"target": "deploy@example.host", "url": "https://game.example.host"}]}'

    def invalid(reason: str) -> RuntimeError:
        return RuntimeError(f"deploy targets config {path}: {reason}. Expected format: {example}")

    try:
        raw = json.loads(path.read_text())
    except (OSError, ValueError):
        raise invalid("missing, unreadable, or invalid JSON") from None
    if not isinstance(raw, dict) or set(raw) != {"targets"}:
        raise invalid("expected an object with only a targets field")
    entries = raw["targets"]
    if not isinstance(entries, list) or not entries:
        raise invalid("targets must be a nonempty array")
    targets: list[DeployTarget] = []
    seen: set[tuple[str, str]] = set()
    for index, entry in enumerate(entries, 1):
        label = f"target {index}"
        if not isinstance(entry, dict) or not {"target", "url"} <= set(entry) <= {"target", "url", "remote_dir"}:
            raise invalid(f"{label} needs target and url, with only optional remote_dir")
        target, url = entry["target"], entry["url"]
        remote_dir = entry.get("remote_dir", DEFAULT_REMOTE_DIR)
        if not all(isinstance(value, str) and value for value in (target, url, remote_dir)):
            raise invalid(f"{label} fields must be nonempty strings")
        # Restrict destinations to SSH aliases or user@host. In particular,
        # reject rsync's host:path syntax, credentials and shell metacharacters.
        if not re.fullmatch(r"(?:[A-Za-z0-9_][A-Za-z0-9_.-]*@)?[A-Za-z0-9_][A-Za-z0-9_.-]*", target):
            raise invalid(f"{label} target must be an SSH alias or user@host")
        try:
            validate_ship_target(target, remote_dir)
        except RuntimeError:
            raise invalid(f"{label} remote_dir must use letters, digits and . _ - / only") from None
        try:
            parsed = urlparse(url)
            site = parse_site_url(url)
            if (
                parsed.username is not None or parsed.password is not None
                or any(c.isspace() or ord(c) < 32 for c in url)
                or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.-]*", site.host)
                or parsed.netloc.endswith(":")
                or (site.port is not None and not 1 <= site.port <= 65535)
            ):
                raise ValueError("invalid origin")
        except ValueError:
            raise invalid(f"{label} url must be an http(s) origin with a DNS name or IPv4 host, no credentials") from None
        identity = (target, remote_dir)
        if identity in seen:
            raise invalid(f"{label} repeats a target and remote_dir")
        seen.add(identity)
        targets.append(DeployTarget(ShipTarget(target, remote_dir), site))
    return targets


def poll_host_version(
    target: str,
    site: Site,
    *,
    timeout: float = SHIP_HEALTH_POLL_TIMEOUT,
    interval: float = SHIP_HEALTH_POLL_INTERVAL,
) -> str | None:
    """Probe the public origin on the host itself, bypassing DNS and proxies.

    --resolve preserves the Host header and TLS SNI. Both curl and its SSH
    subprocess have time limits, including when the connection never opens.
    """
    port = site.port or (443 if site.scheme == "https" else 80)
    deadline = time.monotonic() + timeout
    while (remaining := deadline - time.monotonic()) > 0:
        request_timeout = min(SHIP_HEALTH_REQUEST_TIMEOUT, remaining)
        curl = [
            "curl", "--fail", "--silent", "--show-error", "--noproxy", "*",
            "--connect-timeout", str(min(2.0, request_timeout)),
            "--max-time", str(request_timeout),
            "--resolve", f"{site.host}:{port}:127.0.0.1", site.origin + "/health",
        ]
        cmd = ssh_command(target, shlex.join(curl))
        # SSH cannot prompt for a password and hang an unattended run.
        cmd[1:1] = ["-o", "BatchMode=yes", "-o", "ConnectTimeout=5"]
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=remaining)
            if result.returncode == 0:
                payload = json.loads(result.stdout)
                if isinstance(payload, dict) and isinstance(payload.get("version"), str) and payload["version"]:
                    return payload["version"]
        except (OSError, subprocess.TimeoutExpired, ValueError):
            pass
        time.sleep(min(interval, max(0, deadline - time.monotonic())))
    return None


class InvalidHealthResponse(RuntimeError):
    """The public health endpoint did not send the expected version JSON."""


class HealthResponseTooLarge(RuntimeError):
    """The public health endpoint exceeded the batch response-size limit."""


def public_health_version(url: str, deadline: float) -> str:
    """Read at most 16 KiB from curl before an absolute monotonic deadline.

    Curl's own total timeout also covers DNS, headers and trickling bodies.
    The pipe reader independently bounds time and bytes, without relying on
    curl's version-dependent --max-filesize behavior or buffering stderr.
    """
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise TimeoutError("public health deadline expired")
    cmd = [
        "curl", "--disable", "--fail", "--silent", "--show-error",
        "--connect-timeout", str(min(2.0, remaining)),
        "--max-time", str(remaining), url.rstrip("/") + "/health",
    ]
    with subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL) as process:
        try:
            assert process.stdout is not None
            body = bytearray()
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0 or not select.select([process.stdout], [], [], remaining)[0]:
                    raise TimeoutError("public health deadline expired")
                # Read one byte past the cap so an oversized response is
                # rejected even when its Content-Length is absent or false.
                chunk = os.read(process.stdout.fileno(), min(4096, SHIP_HEALTH_MAX_BYTES + 1 - len(body)))
                if not chunk:
                    break
                body.extend(chunk)
                if len(body) > SHIP_HEALTH_MAX_BYTES:
                    raise HealthResponseTooLarge("public health body exceeds the size limit")
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise TimeoutError("public health deadline expired")
            returncode = process.wait(timeout=remaining)
            if time.monotonic() >= deadline:
                raise TimeoutError("public health deadline expired")
            if returncode == 28:  # curl CURLE_OPERATION_TIMEDOUT
                raise TimeoutError("public health deadline expired")
            if returncode != 0:
                raise RuntimeError("public health request failed")
            try:
                payload = json.loads(body.decode("utf-8"))
            except (ValueError, RecursionError):
                raise InvalidHealthResponse("public health body is not valid JSON") from None
            if not isinstance(payload, dict) or not isinstance(payload.get("version"), str) or not payload["version"]:
                raise InvalidHealthResponse("public health body has no version string")
            return payload["version"]
        except subprocess.TimeoutExpired:
            raise TimeoutError("public health deadline expired") from None
        finally:
            # No background reader or curl may survive a timeout, early size
            # rejection, malformed response or interrupted batch.
            if process.poll() is None:
                process.kill()
            process.wait()


def poll_public_version(
    url: str, *, timeout: float = SHIP_HEALTH_POLL_TIMEOUT, interval: float = SHIP_HEALTH_POLL_INTERVAL
) -> str:
    """Batch-only public poll with one total deadline across every attempt."""
    deadline = time.monotonic() + timeout
    failure: Exception = TimeoutError("public health deadline expired")
    while (remaining := deadline - time.monotonic()) > 0:
        try:
            return public_health_version(url, min(deadline, time.monotonic() + SHIP_HEALTH_REQUEST_TIMEOUT))
        except (OSError, RuntimeError) as exc:
            failure = exc
        time.sleep(min(interval, max(0, deadline - time.monotonic())))
    raise failure


def command_ship_all(force_room_archive: bool = False) -> int:
    targets = load_deploy_targets()
    platforms: list[tuple[str, str] | None] = []
    platform_errors: list[str | None] = []
    for entry in targets:
        try:
            platforms.append(detect_remote_platform(entry.destination.target))
            platform_errors.append(None)
        except Exception as exc:
            platforms.append(None)
            platform_errors.append(type(exc).__name__)
    # A single static binary cannot serve mixed architectures. Refuse the
    # whole batch before building or changing any host rather than quietly
    # building twice or deploying an incompatible executable.
    if None in platforms or len(set(platforms)) != 1:
        for index, (entry, platform_pair, error) in enumerate(zip(targets, platforms, platform_errors), 1):
            if platform_pair is None:
                status = f"platform check failed ({error})"
            else:
                status = f"platform={'/'.join(platform_pair)}, skipped: incompatible or unavailable batch platform"
            print(f"target {index} ({entry.site.origin}): {status}")
        print("error: ship-all requires every target to have the same supported platform; nothing deployed", file=sys.stderr)
        return 1
    goos, goarch = platforms[0]
    try:
        version = git_version()
        binary = cross_build(version, goos, goarch)
    except Exception as exc:
        for index, entry in enumerate(targets, 1):
            print(f"target {index} ({entry.site.origin}): skipped: build failed")
        print(f"error: ship-all build failed ({type(exc).__name__}); nothing deployed", file=sys.stderr)
        return 1

    failed = False
    for index, entry in enumerate(targets, 1):
        target, remote_dir = entry.destination.target, entry.destination.remote_dir
        deployment = "ok"
        try:
            upload_to_remote(target, remote_dir, binary)
            deploy_command = determine_deploy_command(target, remote_dir)
            # No flag overrides: each host retains its own both_schemes,
            # caddyfile and ports from its remembered config.
            deploy_remote(target, remote_dir, deploy_command, entry.site.origin,
                          **({"force_room_archive": True} if force_room_archive else {}))
        except Exception as exc:
            deployment = f"failed ({type(exc).__name__})"

        def health_status(probe: Callable[[], str | None]) -> str:
            try:
                observed = probe()
                if observed == version:
                    return "ok"
                return "failed (version mismatch)" if observed else "failed (unavailable)"
            except Exception as exc:
                return f"failed ({type(exc).__name__})"

        remote_health = health_status(lambda: poll_host_version(target, entry.site))
        local_health = health_status(lambda: poll_public_version(entry.site.origin))
        failed |= any(status != "ok" for status in (deployment, remote_health, local_health))
        print(f"target {index} ({entry.site.origin}): deploy={deployment}, remote-health={remote_health}, local-health={local_health}")
    return 1 if failed else 0


def prepare_prod_caddy(config: Config) -> Exception | None:
    """Check Caddy before stopping, but keep its failure separate from startup."""
    try:
        refuse_to_create_caddyfile()
        ensure_tools(["caddy"])
        preflight_caddyfile(config.site, Mode.PROD, config.ports, config.both_schemes)
    except Exception as exc:
        print(f"warning: Caddy configuration failed: {exc}", file=sys.stderr)
        return exc
    return None


def start_prod_with_caddy(config: Config, caddy_error: Exception | None) -> None:
    """Start the game even when the independent Caddy step has failed."""
    publication_verified = True
    if caddy_error is None:
        try:
            loaded = apply_caddyfile_safely(config.site, Mode.PROD, config.ports, config.both_schemes)
            if loaded is False:
                raise RuntimeError("Caddy did not load the Caddyfile")
            if loaded is None:
                # An unchanged block requires no write or reload, but a stopped
                # TCP-admin Caddy must not produce a successful publication claim.
                address = caddy_admin_address()
                if address is not None:
                    check_caddy_serves(address, CADDYFILE.read_bytes())
                else:
                    publication_verified = False
        except Exception as exc:
            caddy_error = exc
            print(f"warning: Caddy configuration failed: {exc}", file=sys.stderr)
    try:
        start_server_prod(config.ports, config.site)
        healthy = poll_health(config.ports.server)
    except Exception as exc:
        raise RuntimeError(f"game server startup failed on port {config.ports.server}: {exc}") from exc
    if not healthy:
        raise RuntimeError(
            f"game server startup could not be verified on port {config.ports.server}: "
            "/health did not respond within the poll window"
        )
    if caddy_error is not None:
        raise RuntimeError(
            f"Caddy configuration failed; game server is running on http://127.0.0.1:{config.ports.server} "
            f"but is not published at {config.site.origin}"
        ) from caddy_error
    if not publication_verified:
        print(
            f"game server is running on http://127.0.0.1:{config.ports.server}; "
            f"Caddyfile unchanged, publication at {config.site.origin} is unverified (no TCP admin API)"
        )
        return
    announce_serving(config)


def command_setup(
    url: str | None, prebuilt: str | None = None, both_schemes: bool | None = None, caddyfile: str | None = None,
    force_room_archive: bool = False,
) -> None:
    config = resolve_config(url, Mode.PROD, both_schemes, caddyfile)
    use_caddyfile(config)
    ensure_tools(["tmux"])
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
    caddy_error = prepare_prod_caddy(config)
    stop_for_candidate(config, new_binary, force_room_archive)
    config = ports_after_stop(config)
    save_config(config)
    swap_in_binary(new_binary, move=prebuilt_path is not None)
    start_prod_with_caddy(config, caddy_error)


def command_redeploy(
    url: str | None, prebuilt: str | None = None, both_schemes: bool | None = None, caddyfile: str | None = None,
    force_room_archive: bool = False,
) -> None:
    config = resolve_config(url, Mode.PROD, both_schemes, caddyfile)
    use_caddyfile(config)
    ensure_tools(["tmux"])
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
    caddy_error = prepare_prod_caddy(config)
    stop_for_candidate(config, new_binary, force_room_archive)
    config = ports_after_stop(config)
    save_config(config)
    archived = archive_rooms()
    if archived is not None:
        print(f"archived rooms: {archived}")
    swap_in_binary(new_binary, move=prebuilt_path is not None)
    start_prod_with_caddy(config, caddy_error)


def command_start(url: str | None, both_schemes: bool | None = None, caddyfile: str | None = None) -> None:
    config = resolve_config(url, Mode.PROD, both_schemes, caddyfile)
    use_caddyfile(config)
    ensure_tools(["tmux"])
    caddy_error = prepare_prod_caddy(config)
    command_stop()
    config = ports_after_stop(config)
    if not BIN_PATH.exists():
        ensure_tools(["zsh", "rsync"])
        find_go()
        install_web()
        build_web()
        sync_webdist()
        swap_in_binary(build_server_release())
    save_config(config)
    start_prod_with_caddy(config, caddy_error)


def command_dev_start(url: str | None, both_schemes: bool | None = None, caddyfile: str | None = None) -> None:
    config = resolve_config(url, Mode.DEV, both_schemes, caddyfile)
    use_caddyfile(config)
    refuse_to_create_caddyfile()
    command_stop()
    config = replace(config, ports=allocate_ports(config.ports))
    ensure_tools(["caddy", "tmux", "zsh"])
    find_go()
    install_web()
    save_config(config)
    apply_caddyfile_safely(config.site, Mode.DEV, config.ports, config.both_schemes)
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
        print(f"both schemes: {'yes, http:// and https:// both serve the site' if config.both_schemes else 'no'}")
        print(f"caddyfile: {caddyfile_path(config.caddyfile)}")
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

    def add_both_schemes(p: argparse.ArgumentParser) -> None:
        # A pair of store_const flags rather than BooleanOptionalAction, which
        # needs Python 3.9: this script runs on whatever python3 a host has.
        group = p.add_mutually_exclusive_group()
        group.add_argument(
            "--both-schemes",
            dest="both_schemes",
            action="store_const",
            const=True,
            default=None,
            help="serve the site on both http:// and https:// instead of redirecting one to the other "
            "(remembered; without either flag the stored choice is kept, off for a new install)",
        )
        group.add_argument(
            "--no-both-schemes",
            dest="both_schemes",
            action="store_const",
            const=False,
            help="redirect the other scheme to the site's own again (remembered)",
        )

    def add_caddyfile(p: argparse.ArgumentParser) -> None:
        p.add_argument(
            "--caddyfile",
            metavar="PATH",
            default=None,
            help="the Caddyfile the running Caddy reads, absolute or relative to the home directory "
            "(remembered; default: keep the stored one, ~/Caddyfile for a new install)",
        )

    def add_prebuilt(p: argparse.ArgumentParser) -> None:
        p.add_argument(
            "--prebuilt",
            metavar="PATH",
            help="use this already-built binary instead of installing, building and syncing here",
        )

    def add_force_room_archive(p: argparse.ArgumentParser) -> None:
        p.add_argument("--force-room-archive", action="store_true",
                       help="explicitly accept candidate ending in-progress rooms; read/schema errors still refuse")

    setup_p = sub.add_parser("setup", help="first-time install")
    add_url(setup_p)
    add_prebuilt(setup_p)
    add_force_room_archive(setup_p)
    add_both_schemes(setup_p)
    add_caddyfile(setup_p)

    redeploy_p = sub.add_parser("redeploy", help="rebuild and restart for a new version")
    add_url(redeploy_p)
    add_prebuilt(redeploy_p)
    add_force_room_archive(redeploy_p)
    add_both_schemes(redeploy_p)
    add_caddyfile(redeploy_p)

    start_p = sub.add_parser("start", help="start the already-built server")
    add_url(start_p)
    add_both_schemes(start_p)
    add_caddyfile(start_p)

    sub.add_parser("stop", help="stop every tmux session this script manages")

    dev_p = sub.add_parser("dev-start", help="start a development setup instead of production")
    add_url(dev_p)
    add_both_schemes(dev_p)
    add_caddyfile(dev_p)

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
    add_force_room_archive(ship_p)
    add_both_schemes(ship_p)
    add_caddyfile(ship_p)

    ship_all_p = sub.add_parser("ship-all", help="build once and deploy every target in the private deploy-targets.json config")
    add_force_room_archive(ship_all_p)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    try:
        if args.command == "setup":
            command_setup(args.url, args.prebuilt, args.both_schemes, args.caddyfile, args.force_room_archive)
        elif args.command == "redeploy":
            command_redeploy(args.url, args.prebuilt, args.both_schemes, args.caddyfile, args.force_room_archive)
        elif args.command == "start":
            command_start(args.url, args.both_schemes, args.caddyfile)
        elif args.command == "dev-start":
            command_dev_start(args.url, args.both_schemes, args.caddyfile)
        elif args.command == "stop":
            command_stop()
        elif args.command == "status":
            command_status()
        elif args.command == "ship-all":
            return command_ship_all(args.force_room_archive)
        elif args.command == "ship":
            command_ship(args.target, args.url, args.remote_dir, args.both_schemes, args.caddyfile, args.force_room_archive)
    except Exception as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
