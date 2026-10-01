#!/usr/bin/env python3
"""Run heavy jobs under Ajayeb's RAM, slot and mode guard.

Every command that compiles, tests, builds, lints, installs, launches a
browser or runs codex goes through this script, usually via the Makefile:

  heavy.py run --kind KIND [--wait SECONDS] -- CMD...
  heavy.py status
  heavy.py stop-all [--grace SECONDS]
  heavy.py mode [continuous|nightshift|paused] [--window HH:MM-HH:MM]

Kinds: test, build, check, lint, browser, codex, install, fuzz, bench, light.
"light" is for cheap work: it needs no slot and every mode allows it.

A run passes three gates, checked in this order:

Mode gate
  The mode lives in RUN_DIR/mode.json. A missing file means "continuous".
  "paused" refuses every kind except light. "nightshift" refuses every kind
  except light outside its window (local time, default 00:00-10:00; a window
  such as 22:00-06:00 wraps past midnight). A refusal is immediate.

Pressure gate (heavy slots)
  Memory pressure sets the number of heavy slots: critical 0, warn 1,
  normal 2 (3 inside the nightshift window). macOS reads the sysctls
  kern.memorystatus_vm_pressure_level (1 normal, 2 warn, 4 critical),
  kern.memorystatus_level (percent free) and vm.swapusage. Linux reads
  /proc/meminfo and calls MemAvailable below 15% warn and below 5% critical.
  If pressure cannot be read it counts as warn.

Kind locks
  browser and codex also take their own machine-wide lock of capacity 1.
  A browser also needs free memory >= AJAYEB_MIN_FREE_PCT_BROWSER percent
  (default 14) and free swap >= AJAYEB_MIN_SWAP_FREE_MB_BROWSER MB
  (default 512; skipped when no swap is allocated).

When a slot, lock or threshold is unavailable the job waits with backoff for
up to --wait seconds (default 1800, or AJAYEB_HEAVY_WAIT), then gives up.

Each slot and kind lock is a directory under RUN_DIR/heavy/ made with mkdir,
holding owner.json: the guard's pid, the command's process group and their
start times. A lock whose owner is gone, or whose pid now belongs to another
process, is stale and the next run clears it.

The command runs in its own session. SIGINT, SIGTERM and SIGHUP are forwarded
to its process group, locks are released when it ends, and this script exits
with the command's exit code. The child gets these defaults when unset or
empty: NODE_OPTIONS=--max-old-space-size=1536, GOMEMLIMIT=768MiB,
GOFLAGS=-p=2, GOMAXPROCS=4. A NODE_OPTIONS or GOFLAGS that is set but lacks a
heap or -p flag gets ours appended; a value that has one is kept. The child
also gets AJAYEB_HEAVY_KIND=<kind>, AJAYEB_HEAVY_HELD=1 when a slot is held,
and AJAYEB_HEAVY_LOCKS=<held kind locks>.

Nested runs: when AJAYEB_HEAVY_HELD=1 is already set, an outer heavy.py holds
a slot, so the inner run skips the mode gate and takes no slot (this keeps
"heavy.py run -- make verify" from deadlocking). It still takes a browser or
codex lock that no ancestor holds. fuzz and bench run under nice 10.

Every run appends {ts, kind, cmd, rc, secs, maxrss_kb} to
RUN_DIR/logs/heavy.jsonl. stop-all sends SIGTERM to the process groups this
script started (recorded in RUN_DIR/heavy/running/) and to guards still
waiting for a slot, then SIGKILL to groups that outlive --grace seconds.

RUN_DIR is AJAYEB_RUN_DIR, or ~/tmp/agents/ajayeb. With CI=true and no run
dir, the guard runs in continuous mode with the Linux readers and keeps its
state in RUNNER_TEMP (or the system temp dir). AJAYEB_MAX_SLOTS caps the
slot count.

Exit codes: the command's own code; 75 when refused (EX_TEMPFAIL); 2 for
usage errors; 126 or 127 when the command cannot be started; 128+N when the
command (or this script, before the command started) died from signal N.
"""

from __future__ import annotations

import argparse
import collections
import datetime as _dt
import json
import os
import random
import re
import resource
import shlex
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import uuid
from dataclasses import dataclass
from typing import Callable, Dict, Iterable, List, Mapping, Optional, Sequence, Set, Tuple

EX_TEMPFAIL = 75
EX_USAGE = 2

KINDS = ("test", "build", "check", "lint", "browser", "codex", "install", "fuzz", "bench", "light")
KIND_LOCKS = ("browser", "codex")
NICE_KINDS = ("fuzz", "bench")
MODES = ("continuous", "nightshift", "paused")

DEFAULT_WINDOW = "00:00-10:00"
DEFAULT_WAIT = 1800.0
DEFAULT_GRACE = 10.0
DEFAULT_MIN_FREE_PCT_BROWSER = 14.0
DEFAULT_MIN_SWAP_FREE_MB_BROWSER = 512.0
DEFAULT_CHILD_ENV = (
    ("NODE_OPTIONS", "--max-old-space-size=1536"),
    ("GOMEMLIMIT", "768MiB"),
    ("GOFLAGS", "-p=2"),
    ("GOMAXPROCS", "4"),
)

LEVEL_NORMAL = 1
LEVEL_WARN = 2
LEVEL_CRITICAL = 4
LINUX_WARN_PCT = 15.0
LINUX_CRITICAL_PCT = 5.0

# A lock dir may exist without its owner file for this long (the owner is
# between mkdir and writing the file); after that it counts as stale.
FRESH_LOCK_GRACE = 30.0
BREAK_GRACE = 30.0
BACKOFF_START = 2.0
BACKOFF_MAX = 30.0
OWNER_FILE = "owner.json"
SLOT_RE = re.compile(r"^slot-(\d+)$")
FORWARDED_SIGNALS = (signal.SIGINT, signal.SIGTERM, signal.SIGHUP)


# ----------------------------------------------------------------------------
# Small helpers


def fmt_secs(secs: float) -> str:
    secs = int(round(max(secs, 0)))
    if secs < 60:
        return f"{secs}s"
    minutes, secs = divmod(secs, 60)
    if minutes < 60:
        return f"{minutes}m{secs:02d}s"
    hours, minutes = divmod(minutes, 60)
    return f"{hours}h{minutes:02d}m"


def iso(when: _dt.datetime) -> str:
    try:
        return when.astimezone().isoformat(timespec="seconds")
    except (ValueError, OSError):
        return when.isoformat(timespec="seconds")


def display_cmd(cmd: Sequence[str], limit: int = 0) -> str:
    text = shlex.join(list(cmd))
    if limit and len(text) > limit:
        return text[: limit - 3] + "..."
    return text


def normalize_maxrss(value: int, platform: str) -> int:
    """ru_maxrss is bytes on macOS and kilobytes on Linux; return kilobytes."""
    if platform == "darwin":
        return int(value) // 1024
    return int(value)


def read_json(path: str) -> Optional[dict]:
    try:
        with open(path, encoding="utf-8") as fh:
            data = json.load(fh)
    except (OSError, ValueError):
        return None
    return data if isinstance(data, dict) else None


def write_json_atomic(path: str, data: Mapping) -> None:
    directory = os.path.dirname(path)
    os.makedirs(directory, exist_ok=True)
    fd, tmp = tempfile.mkstemp(prefix=".tmp-", dir=directory)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as fh:
            json.dump(data, fh, sort_keys=True)
            fh.write("\n")
        os.replace(tmp, path)
    except BaseException:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def remove_file(path: Optional[str]) -> None:
    if not path:
        return
    try:
        os.unlink(path)
    except OSError:
        pass


def default_pid_alive(pid: object) -> bool:
    if not isinstance(pid, int) or pid <= 0:
        return False
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    except (OSError, OverflowError):
        return False
    return True


def default_group_alive(pgid: object) -> bool:
    if not isinstance(pgid, int) or pgid <= 1:
        return False
    try:
        os.killpg(pgid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    except (OSError, OverflowError):
        return False
    return True


PS = "/bin/ps" if os.path.exists("/bin/ps") else "ps"


def _ps_field(pid: int, field: str) -> Optional[str]:
    env = dict(os.environ, LC_ALL="C")
    try:
        res = subprocess.run([PS, "-o", f"{field}=", "-p", str(pid)], env=env,
                             capture_output=True, text=True, timeout=5, check=False)
    except (OSError, subprocess.SubprocessError):
        return None
    if res.returncode != 0:
        return None
    return " ".join(res.stdout.split()) or None


def default_proc_command(pid: int) -> Optional[str]:
    """The command line of a live process, or None."""
    return _ps_field(pid, "command")


def default_proc_start(pid: int) -> Optional[str]:
    """When a process started (ps lstart), or None. It survives exec and changes on pid reuse."""
    return _ps_field(pid, "lstart")


class Clock:
    """Wall clock, monotonic clock and sleep, injectable for tests."""

    def now(self) -> _dt.datetime:
        return _dt.datetime.now()

    def monotonic(self) -> float:
        return time.monotonic()

    def sleep(self, secs: float) -> None:
        time.sleep(secs)


# ----------------------------------------------------------------------------
# Window and mode

_WINDOW_RE = re.compile(r"^\s*(\d{1,2}):(\d{2})\s*-\s*(\d{1,2}):(\d{2})\s*$")


@dataclass(frozen=True)
class Window:
    """A daily local-time window [start, end) in minutes after midnight.

    When start > end the window wraps past midnight (22:00-06:00).
    The end may be 24:00, so 00:00-24:00 is the whole day.
    """

    start: int
    end: int

    @classmethod
    def parse(cls, text: str) -> "Window":
        match = _WINDOW_RE.match(text or "")
        if not match:
            raise ValueError(f"window must look like HH:MM-HH:MM, got {text!r}")
        sh, sm, eh, em = (int(g) for g in match.groups())
        if sh > 23 or sm > 59 or em > 59 or eh > 24 or (eh == 24 and em != 0):
            raise ValueError(f"window has an invalid time: {text!r}")
        start, end = sh * 60 + sm, eh * 60 + em
        if start == end:
            raise ValueError(f"window is empty: {text!r}")
        return cls(start, end)

    def contains(self, minute: int) -> bool:
        if self.start < self.end:
            return self.start <= minute < self.end
        return minute >= self.start or minute < self.end

    def contains_time(self, when: _dt.datetime) -> bool:
        return self.contains(when.hour * 60 + when.minute)

    def __str__(self) -> str:
        return (f"{self.start // 60:02d}:{self.start % 60:02d}-"
                f"{self.end // 60:02d}:{self.end % 60:02d}")


@dataclass
class ModeState:
    mode: str
    window: Window
    error: Optional[str] = None
    source: str = "file"  # "file", "default" (no file) or "ci"

    def in_window(self, now: _dt.datetime) -> bool:
        return self.mode == "nightshift" and self.error is None and self.window.contains_time(now)


def load_mode(path: str) -> ModeState:
    window = Window.parse(DEFAULT_WINDOW)
    try:
        with open(path, encoding="utf-8") as fh:
            text = fh.read()
    except FileNotFoundError:
        return ModeState("continuous", window, source="default")
    except OSError as exc:
        return ModeState("invalid", window, error=f"cannot read mode.json: {exc.strerror or exc}")
    try:
        data = json.loads(text)
    except ValueError as exc:
        return ModeState("invalid", window, error=f"mode.json is not JSON: {exc}")
    if not isinstance(data, dict) or data.get("mode") not in MODES:
        mode = data.get("mode") if isinstance(data, dict) else None
        return ModeState("invalid", window, error=f"mode.json has unknown mode {mode!r}")
    raw_window = data.get("window") or DEFAULT_WINDOW
    try:
        window = Window.parse(str(raw_window))
    except ValueError as exc:
        return ModeState("invalid", window, error=f"mode.json: {exc}")
    return ModeState(data["mode"], window)


def gate_mode(kind: str, state: ModeState, now: _dt.datetime) -> Optional[str]:
    """Why the mode forbids this kind right now, or None when it is allowed."""
    if kind == "light":
        return None
    if state.error:
        return f"{state.error}; fix it with: heavy.py mode continuous|nightshift|paused"
    if state.mode == "paused":
        return "mode is paused; only --kind light may run"
    if state.mode == "nightshift" and not state.in_window(now):
        return (f"mode is nightshift and {now:%H:%M} is outside the window {state.window}; "
                "only --kind light may run")
    return None


# ----------------------------------------------------------------------------
# Memory readers


@dataclass
class Memory:
    level: Optional[int] = None  # 1 normal, 2 warn, 4 critical, None unknown
    free_pct: Optional[float] = None
    swap_total_mb: Optional[float] = None
    swap_free_mb: Optional[float] = None
    source: str = "unknown"


def level_name(level: Optional[int]) -> str:
    if level is None:
        return "unknown"
    return {LEVEL_NORMAL: "normal", LEVEL_WARN: "warn", LEVEL_CRITICAL: "critical"}.get(level, f"level {level}")


_SWAP_RE = re.compile(r"(total|used|free)\s*=\s*([\d.]+)\s*([KMGT]?)", re.IGNORECASE)
_UNIT_MB = {"": 1.0 / (1024 * 1024), "K": 1.0 / 1024, "M": 1.0, "G": 1024.0, "T": 1024.0 * 1024}


def parse_swapusage(text: str) -> Tuple[Optional[float], Optional[float]]:
    """Parse macOS `sysctl -n vm.swapusage` into (total MB, free MB)."""
    values: Dict[str, float] = {}
    for name, number, unit in _SWAP_RE.findall(text or ""):
        values[name.lower()] = float(number) * _UNIT_MB[unit.upper()]
    return values.get("total"), values.get("free")


def default_sysctl(key: str) -> Optional[str]:
    exe = "/usr/sbin/sysctl" if os.path.exists("/usr/sbin/sysctl") else "sysctl"
    try:
        res = subprocess.run([exe, "-n", key], capture_output=True, text=True, timeout=5, check=False)
    except (OSError, subprocess.SubprocessError):
        return None
    if res.returncode != 0:
        return None
    return res.stdout.strip()


def _to_int(text: Optional[str]) -> Optional[int]:
    try:
        return int(str(text).strip())
    except (TypeError, ValueError):
        return None


def read_darwin(sysctl: Callable[[str], Optional[str]] = default_sysctl) -> Memory:
    level = _to_int(sysctl("kern.memorystatus_vm_pressure_level"))
    free = _to_int(sysctl("kern.memorystatus_level"))
    swap_total, swap_free = parse_swapusage(sysctl("vm.swapusage") or "")
    return Memory(level=level, free_pct=None if free is None else float(free),
                  swap_total_mb=swap_total, swap_free_mb=swap_free, source="darwin sysctl")


def level_from_free_pct(free_pct: Optional[float]) -> Optional[int]:
    if free_pct is None:
        return None
    if free_pct < LINUX_CRITICAL_PCT:
        return LEVEL_CRITICAL
    if free_pct < LINUX_WARN_PCT:
        return LEVEL_WARN
    return LEVEL_NORMAL


def parse_meminfo(text: str) -> Memory:
    kb: Dict[str, int] = {}
    for line in (text or "").splitlines():
        name, _, rest = line.partition(":")
        parts = rest.split()
        if parts and parts[0].isdigit():
            kb[name.strip()] = int(parts[0])
    total = kb.get("MemTotal")
    avail = kb.get("MemAvailable")
    free_pct = (avail * 100.0 / total) if total and avail is not None else None
    swap_total = kb["SwapTotal"] / 1024.0 if "SwapTotal" in kb else None
    swap_free = kb["SwapFree"] / 1024.0 if "SwapFree" in kb else None
    return Memory(level=level_from_free_pct(free_pct), free_pct=free_pct,
                  swap_total_mb=swap_total, swap_free_mb=swap_free, source="linux /proc/meminfo")


def read_linux(path: str = "/proc/meminfo") -> Memory:
    try:
        with open(path, encoding="utf-8") as fh:
            return parse_meminfo(fh.read())
    except OSError:
        return Memory(source=f"unreadable {path}")


def default_reader(platform: str) -> Callable[[], Memory]:
    if platform == "darwin":
        return read_darwin
    if platform.startswith("linux"):
        return read_linux
    return lambda: Memory(source=f"no reader for {platform}")


def slot_capacity(level: Optional[int], nightshift_window: bool, max_slots: Optional[int] = None) -> int:
    """Heavy slots for a pressure level: critical 0, warn 1, normal 2 (3 in the nightshift window)."""
    if level is None:
        cap = 1
    elif level >= LEVEL_CRITICAL:
        cap = 0
    elif level >= LEVEL_WARN:
        cap = 1
    else:
        cap = 3 if nightshift_window else 2
    if max_slots is not None:
        cap = min(cap, max_slots)
    return max(cap, 0)


def browser_block_reason(mem: Memory, min_free_pct: float, min_swap_mb: float) -> Optional[str]:
    """Why a browser may not launch now, or None. Unknown readings do not block."""
    if mem.free_pct is not None and mem.free_pct < min_free_pct:
        return f"free memory {mem.free_pct:.0f}% is below the browser floor of {min_free_pct:g}%"
    if mem.swap_total_mb and mem.swap_free_mb is not None and mem.swap_free_mb < min_swap_mb:
        return f"free swap {mem.swap_free_mb:.0f} MB is below the browser floor of {min_swap_mb:g} MB"
    return None


# ----------------------------------------------------------------------------
# Lock dirs


class Locks:
    """Machine-wide locks: one directory each, created atomically with mkdir.

    The directory holds owner.json with the guard's pid and, once the command
    has started, its process group, each with its start time (ps lstart) when
    it could be read. A lock is stale when neither is alive. A pid that is
    alive but started at a different time has been reused by an unrelated
    process, so it does not keep the lock alive; an unreadable start time
    does (when unsure, a lock is kept rather than broken).
    """

    def __init__(self, root: str, pid_alive: Callable[[object], bool],
                 group_alive: Callable[[object], bool],
                 proc_start: Optional[Callable[[int], Optional[str]]] = None):
        self.root = root
        self.pid_alive = pid_alive
        self.group_alive = group_alive
        self.proc_start = proc_start

    def path(self, name: str) -> str:
        return os.path.join(self.root, name)

    def owner(self, name: str) -> Optional[dict]:
        return read_json(os.path.join(self.path(name), OWNER_FILE))

    def _not_reused(self, pid: int, recorded: object) -> bool:
        if not recorded or self.proc_start is None:
            return True
        current = self.proc_start(pid)
        return current is None or current == recorded

    def info_alive(self, info: Mapping) -> bool:
        pid = info.get("pid")
        if self.pid_alive(pid) and self._not_reused(pid, info.get("lstart")):
            return True
        pgid = info.get("pgid")
        return (isinstance(pgid, int) and self.group_alive(pgid)
                and self._not_reused(pgid, info.get("pgid_lstart")))

    def state(self, name: str) -> str:
        """"free", "live" or "stale"."""
        path = self.path(name)
        if not os.path.isdir(path):
            return "free"
        info = self.owner(name)
        if info is None:
            try:
                age = time.time() - os.stat(path).st_mtime
            except OSError:
                return "free"
            return "live" if age < FRESH_LOCK_GRACE else "stale"
        return "live" if self.info_alive(info) else "stale"

    def slot_names(self) -> List[str]:
        try:
            names = [n for n in os.listdir(self.root) if SLOT_RE.match(n)]
        except OSError:
            return []
        return sorted(names, key=lambda n: int(SLOT_RE.match(n).group(1)))

    def try_acquire(self, name: str, info: Mapping) -> bool:
        os.makedirs(self.root, exist_ok=True)
        path = self.path(name)
        for attempt in (0, 1):
            try:
                os.mkdir(path)
            except FileExistsError:
                if attempt == 0 and self.state(name) == "stale" and self.break_stale(name):
                    continue
                return False
            write_json_atomic(os.path.join(path, OWNER_FILE), info)
            return True
        return False

    def update(self, name: str, token: str, **fields: object) -> None:
        info = self.owner(name)
        if not info or info.get("token") != token:
            return
        info.update(fields)
        write_json_atomic(os.path.join(self.path(name), OWNER_FILE), info)

    def release(self, name: str, token: str) -> None:
        info = self.owner(name)
        if info is not None and info.get("token") != token:
            return  # not ours any more; never remove another holder's lock
        shutil.rmtree(self.path(name), ignore_errors=True)

    def break_stale(self, name: str) -> bool:
        """Remove a stale lock. A sibling NAME.break dir serializes breakers."""
        path = self.path(name)
        breaker = path + ".break"
        try:
            os.mkdir(breaker)
        except FileExistsError:
            try:
                if time.time() - os.stat(breaker).st_mtime > BREAK_GRACE:
                    os.rmdir(breaker)
            except OSError:
                pass
            return False
        except OSError:
            return False
        try:
            if self.state(name) != "stale":
                return False
            shutil.rmtree(path, ignore_errors=True)
            return not os.path.exists(path)
        finally:
            try:
                os.rmdir(breaker)
            except OSError:
                pass

    def sweep(self) -> None:
        for name in self.slot_names() + list(KIND_LOCKS):
            if self.state(name) == "stale":
                self.break_stale(name)


# ----------------------------------------------------------------------------
# The guard


class Refused(Exception):
    def __init__(self, reason: str):
        super().__init__(reason)
        self.reason = reason


class _Interrupted(BaseException):
    """Raised by the signal handler. A BaseException, like KeyboardInterrupt,
    so that `except Exception` (around the memory reader) cannot swallow it."""

    def __init__(self, signum: int):
        super().__init__(signum)
        self.signum = signum


class Guard:
    def __init__(self, env: Optional[Mapping[str, str]] = None, platform: Optional[str] = None,
                 reader: Optional[Callable[[], Memory]] = None, clock: Optional[Clock] = None,
                 pid_alive: Optional[Callable[[object], bool]] = None,
                 group_alive: Optional[Callable[[object], bool]] = None,
                 proc_command: Optional[Callable[[int], Optional[str]]] = None,
                 proc_start: Optional[Callable[[int], Optional[str]]] = None,
                 rng: Optional[random.Random] = None, stdout=None, stderr=None):
        self.env: Dict[str, str] = dict(os.environ if env is None else env)
        self.platform = platform or sys.platform
        self.clock = clock or Clock()
        self.pid_alive = pid_alive or default_pid_alive
        self.group_alive = group_alive or default_group_alive
        self.proc_command = proc_command or default_proc_command
        self.proc_start = proc_start or default_proc_start
        self.rng = rng or random.Random()
        self.out = stdout or sys.stdout
        self.err = stderr or sys.stderr

        self.home = self.env.get("HOME") or os.path.expanduser("~")
        self.configured_run_dir = self.env.get("AJAYEB_RUN_DIR") or os.path.join(
            self.home, "tmp", "agents", "ajayeb")
        ci = self.env.get("CI", "").lower() in ("true", "1")
        self.ci_mode = ci and not os.path.isdir(self.configured_run_dir)
        if self.ci_mode:
            base = self.env.get("RUNNER_TEMP") or tempfile.gettempdir()
            self.run_dir = os.path.join(base, "ajayeb-heavy-ci")
            self.reader_platform = "linux"
        else:
            self.run_dir = self.configured_run_dir
            self.reader_platform = self.platform
        self.reader = reader or default_reader(self.reader_platform)
        self.heavy_dir = os.path.join(self.run_dir, "heavy")
        self.log_path = os.path.join(self.run_dir, "logs", "heavy.jsonl")
        self.mode_path = os.path.join(self.configured_run_dir, "mode.json")
        self.locks = Locks(self.heavy_dir, self.pid_alive, self.group_alive, self.proc_start)

        self._child: Optional[subprocess.Popen] = None
        self._spawning = False
        self._pending: Optional[int] = None
        self._done = False
        self._old_handlers: Dict[int, object] = {}

    # -- configuration -------------------------------------------------------

    def _say(self, msg: str) -> None:
        print(msg, file=self.out)

    def _warn(self, msg: str) -> None:
        print(msg, file=self.err)
        try:
            self.err.flush()
        except (OSError, ValueError):
            pass

    def _env_float(self, name: str, default: float) -> float:
        raw = self.env.get(name)
        if raw is None or raw == "":
            return default
        try:
            return float(raw)
        except ValueError:
            self._warn(f"heavy: ignoring {name}={raw!r}: not a number")
            return default

    def max_slots(self) -> Optional[int]:
        raw = self.env.get("AJAYEB_MAX_SLOTS")
        if raw is None or raw == "":
            return None
        try:
            return max(int(raw), 0)
        except ValueError:
            self._warn(f"heavy: ignoring AJAYEB_MAX_SLOTS={raw!r}: not an integer")
            return None

    def browser_floors(self) -> Tuple[float, float]:
        return (self._env_float("AJAYEB_MIN_FREE_PCT_BROWSER", DEFAULT_MIN_FREE_PCT_BROWSER),
                self._env_float("AJAYEB_MIN_SWAP_FREE_MB_BROWSER", DEFAULT_MIN_SWAP_FREE_MB_BROWSER))

    def mode_state(self) -> ModeState:
        if self.ci_mode:
            return ModeState("continuous", Window.parse(DEFAULT_WINDOW), source="ci")
        return load_mode(self.mode_path)

    def read_memory(self) -> Memory:
        try:
            mem = self.reader()
        except Exception as exc:  # a broken reader must not break every job
            return Memory(source=f"reader failed: {exc}")
        return mem if isinstance(mem, Memory) else Memory(source="reader returned nothing")

    def capacity(self, mem: Memory, state: Optional[ModeState] = None,
                 now: Optional[_dt.datetime] = None) -> int:
        state = state or self.mode_state()
        now = now or self.clock.now()
        return slot_capacity(mem.level, state.in_window(now), self.max_slots())

    def tilde(self, path: str) -> str:
        home = self.home.rstrip("/")
        if home and (path == home or path.startswith(home + "/")):
            return "~" + path[len(home):]
        return path

    # -- describing holders ----------------------------------------------------

    def _describe(self, info: Optional[Mapping]) -> str:
        if not info:
            return "a starting job"
        age = time.time() - float(info.get("t0") or time.time())
        return f"pid {info.get('pid')} {info.get('kind')} {fmt_secs(age)}: {info.get('cmd', '')}"

    def _live_slots(self) -> List[str]:
        return [n for n in self.locks.slot_names() if self.locks.state(n) == "live"]

    # -- acquiring ---------------------------------------------------------------

    def _try_slot(self, cap: int, mem: Memory, info: Mapping) -> Tuple[Optional[str], Optional[str]]:
        if cap <= 0:
            return None, f"memory pressure is {level_name(mem.level)}, which allows no heavy slots"
        live = self._live_slots()
        busy = (f"all {cap} heavy slot(s) are busy at {level_name(mem.level)} pressure"
                f" ({'; '.join(self._describe(self.locks.owner(n)) for n in live)})")
        if len(live) >= cap:
            return None, busy
        for index in range(1, cap + 1):
            name = f"slot-{index}"
            if self.locks.try_acquire(name, info):
                if len(self._live_slots()) > cap:  # lost a race while capacity shrank
                    self.locks.release(name, info["token"])
                    return None, busy
                return name, None
        return None, busy

    def _attempt(self, kind: str, nested: bool, inherited: Set[str],
                 info: Mapping) -> Tuple[List[str], Optional[str], bool]:
        """Try every gate once. Returns (held lock names, reason, final)."""
        state = None
        if not nested:
            state = self.mode_state()
            reason = gate_mode(kind, state, self.clock.now())
            if reason:
                return [], reason, True
        if kind == "light":
            return [], None, False
        needs_kind_lock = kind in KIND_LOCKS and kind not in inherited
        if nested and not needs_kind_lock:
            return [], None, False
        self.locks.sweep()
        mem = self.read_memory()
        held: List[str] = []
        try:
            if needs_kind_lock:
                if kind == "browser":
                    min_free, min_swap = self.browser_floors()
                    reason = browser_block_reason(mem, min_free, min_swap)
                    if reason:
                        return [], reason, False
                if not self.locks.try_acquire(kind, info):
                    holder = self._describe(self.locks.owner(kind))
                    return [], f"the {kind} lock is held by {holder}", False
                held.append(kind)
            if not nested:
                slot, reason = self._try_slot(self.capacity(mem, state), mem, info)
                if slot is None:
                    self._release(held, info["token"])
                    return [], reason, False
                held.append(slot)
            return held, None, False
        except BaseException:
            self._release(held, info["token"])
            raise

    def _acquire(self, kind: str, wait: float, nested: bool, inherited: Set[str],
                 info: Mapping) -> List[str]:
        deadline = self.clock.monotonic() + max(wait, 0.0)
        delay = BACKOFF_START
        last_reason = None
        while True:
            held, reason, final = self._attempt(kind, nested, inherited, info)
            if reason is None:
                return held
            if final:
                raise Refused(reason)
            remaining = deadline - self.clock.monotonic()
            if remaining <= 0:
                if wait > 0:
                    reason = f"{reason}; gave up after waiting {fmt_secs(wait)}"
                raise Refused(reason)
            if reason != last_reason:
                self._warn(f"heavy: {kind} is waiting: {reason} (up to {fmt_secs(remaining)} more)")
                last_reason = reason
            self.clock.sleep(min(remaining, delay * self.rng.uniform(0.8, 1.2)))
            delay = min(delay * 2, BACKOFF_MAX)

    def _release(self, held: Iterable[str], token: str) -> None:
        for name in reversed(list(held)):
            self.locks.release(name, token)

    # -- signals -----------------------------------------------------------------

    def _on_signal(self, signum, frame) -> None:
        if self._done:
            return
        if self._child is not None:
            self._forward(signum)
            return
        if self._spawning:
            self._pending = signum
            return
        raise _Interrupted(signum)

    def _forward(self, signum: int) -> None:
        child = self._child
        if child is None or child.pid <= 1:
            return
        try:
            os.killpg(child.pid, signum)
        except (ProcessLookupError, PermissionError):
            pass

    def _install_signals(self) -> None:
        self._old_handlers = {}
        for sig in FORWARDED_SIGNALS:
            try:
                self._old_handlers[sig] = signal.signal(sig, self._on_signal)
            except (ValueError, OSError):  # not the main thread
                pass

    def _restore_signals(self) -> None:
        for sig, handler in self._old_handlers.items():
            try:
                signal.signal(sig, handler)
            except (ValueError, OSError, TypeError):
                pass
        self._old_handlers = {}

    # -- records and log -----------------------------------------------------------

    def _record(self, sub: str, pid: int, data: Mapping) -> Optional[str]:
        path = os.path.join(self.heavy_dir, sub, f"{pid}.json")
        try:
            write_json_atomic(path, data)
        except OSError as exc:
            self._warn(f"heavy: cannot write {sub} record: {exc}")
            return None
        return path

    def _records(self, sub: str) -> List[Tuple[str, dict]]:
        directory = os.path.join(self.heavy_dir, sub)
        try:
            names = sorted(n for n in os.listdir(directory) if n.endswith(".json"))
        except OSError:
            return []
        out = []
        for name in names:
            path = os.path.join(directory, name)
            data = read_json(path)
            if data is not None:
                out.append((path, data))
        return out

    def _log(self, kind: str, cmd: Sequence[str], rc: int, secs: float, maxrss_kb: int,
             when: Optional[_dt.datetime] = None, **extra: object) -> None:
        entry = {"ts": iso(when or self.clock.now()), "kind": kind, "cmd": display_cmd(cmd),
                 "rc": rc, "secs": round(secs, 2), "maxrss_kb": int(maxrss_kb)}
        entry.update({k: v for k, v in extra.items() if v not in (None, False, "")})
        line = (json.dumps(entry) + "\n").encode("utf-8")
        try:
            os.makedirs(os.path.dirname(self.log_path), exist_ok=True)
            fd = os.open(self.log_path, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o644)
            try:
                os.write(fd, line)
            finally:
                os.close(fd)
        except OSError as exc:
            self._warn(f"heavy: cannot append to heavy.jsonl: {exc}")

    # -- run -------------------------------------------------------------------------

    def child_env(self, kind: str, slot_held: bool, locks: Set[str]) -> Dict[str, str]:
        env = dict(self.env)
        for key, value in DEFAULT_CHILD_ENV:
            if not env.get(key):
                env[key] = value
        # NODE_OPTIONS and GOFLAGS hold several flags. A value that is set but
        # has no heap or parallelism flag gets ours appended; one that already
        # has such a flag is kept as it is.
        node = env["NODE_OPTIONS"]
        if "--max-old-space-size" not in node and "--max_old_space_size" not in node:
            env["NODE_OPTIONS"] = f"{node} --max-old-space-size=1536"
        goflags = env["GOFLAGS"].split()
        if not any(flag in ("-p", "--p") or flag.startswith(("-p=", "--p=")) for flag in goflags):
            env["GOFLAGS"] = f"{env['GOFLAGS']} -p=2"
        env["AJAYEB_HEAVY_KIND"] = kind
        if slot_held:
            env["AJAYEB_HEAVY_HELD"] = "1"
        else:
            env.pop("AJAYEB_HEAVY_HELD", None)
        if locks:
            env["AJAYEB_HEAVY_LOCKS"] = ",".join(sorted(locks))
        else:
            env.pop("AJAYEB_HEAVY_LOCKS", None)
        return env

    def run(self, kind: str, cmd: Sequence[str], wait: Optional[float] = None) -> int:
        if kind not in KINDS:
            self._warn(f"heavy: unknown kind {kind!r}; use one of {', '.join(KINDS)}")
            return EX_USAGE
        cmd = list(cmd)
        if not cmd:
            self._warn("heavy: run needs a command after --")
            return EX_USAGE
        if wait is None:
            wait = self._env_float("AJAYEB_HEAVY_WAIT", DEFAULT_WAIT)
        nested = self.env.get("AJAYEB_HEAVY_HELD") == "1"
        inherited = {x for x in self.env.get("AJAYEB_HEAVY_LOCKS", "").split(",") if x}
        info = {"pid": os.getpid(), "kind": kind, "cmd": display_cmd(cmd, 160),
                "since": iso(self.clock.now()), "t0": time.time(), "token": uuid.uuid4().hex}
        if kind != "light" and (not nested or kind in KIND_LOCKS):
            lstart = self.proc_start(os.getpid())  # this run may take a lock
            if lstart:
                info["lstart"] = lstart
        held: List[str] = []
        self._child, self._spawning, self._pending, self._done = None, False, None, False
        self._install_signals()
        try:
            if kind != "light":
                waiting = self._record("waiting", os.getpid(), {
                    "guard": os.getpid(), "kind": kind, "cmd": info["cmd"],
                    "since": info["since"], "t0": info["t0"]})
                try:
                    held = self._acquire(kind, wait, nested, inherited, info)
                except Refused as refusal:
                    self._warn(f"heavy: refused --kind {kind}: {refusal.reason}")
                    self._log(kind, cmd, EX_TEMPFAIL, 0.0, 0, refused=refusal.reason, nested=nested)
                    return EX_TEMPFAIL
                finally:
                    remove_file(waiting)
            slot_held = nested or any(SLOT_RE.match(h) for h in held)
            locks = inherited | {h for h in held if h in KIND_LOCKS}
            env = self.child_env(kind, slot_held, locks)
            return self._spawn(kind, cmd, env, held, info, nested)
        except _Interrupted as exc:
            self._warn(f"heavy: stopped by signal {exc.signum} before the command started")
            return 128 + exc.signum
        finally:
            self._done = True
            self._release(held, info["token"])
            self._restore_signals()

    def _spawn(self, kind: str, cmd: List[str], env: Dict[str, str], held: List[str],
               info: Mapping, nested: bool) -> int:
        started = self.clock.now()
        t0 = self.clock.monotonic()
        preexec = _nice10 if kind in NICE_KINDS else None
        self._spawning = True
        try:
            child = subprocess.Popen(cmd, env=env, start_new_session=True, preexec_fn=preexec)
        except OSError as exc:
            self._spawning = False
            rc = 127 if isinstance(exc, FileNotFoundError) else 126
            self._warn(f"heavy: cannot start {cmd[0]!r}: {exc.strerror or exc}")
            self._log(kind, cmd, rc, 0.0, 0, when=started, nested=nested)
            return rc
        self._child = child
        self._spawning = False
        record_data = {
            "pid": child.pid, "pgid": child.pid, "guard": os.getpid(), "kind": kind,
            "cmd": info["cmd"], "argv0": cmd[0], "since": iso(started), "t0": time.time(),
            "nested": nested}
        record = self._record("running", child.pid, record_data)
        lstart = self.proc_start(child.pid)  # lets stop-all recognize the group if we die
        if lstart and record:
            record_data["lstart"] = lstart
            record = self._record("running", child.pid, record_data)
        try:
            for name in held:
                if lstart:
                    self.locks.update(name, info["token"], pgid=child.pid, pgid_lstart=lstart)
                else:
                    self.locks.update(name, info["token"], pgid=child.pid)
            if self._pending is not None:
                self._forward(self._pending)
                self._pending = None
            rc = child.wait()
        finally:
            self._done = True
            remove_file(record)
        secs = self.clock.monotonic() - t0
        if rc < 0:
            rc = 128 - rc
        maxrss = normalize_maxrss(resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss, self.platform)
        self._log(kind, cmd, rc, secs, maxrss, when=started, nested=nested)
        return rc

    # -- stop-all ----------------------------------------------------------------------

    def _is_heavy_guard(self, pid: object) -> bool:
        if not isinstance(pid, int) or pid <= 1 or not self.pid_alive(pid):
            return False
        return "heavy.py" in (self.proc_command(pid) or "")

    def stop_all(self, grace: float = DEFAULT_GRACE) -> int:
        my_pgid = os.getpgrp()
        guards: List[Tuple[int, str]] = []
        groups: List[Tuple[int, str]] = []
        # Waiting guards first, so they cannot grab a slot that is about to free up.
        for path, rec in self._records("waiting"):
            pid = rec.get("guard")
            if pid == os.getpid():
                continue
            if not self._is_heavy_guard(pid):
                remove_file(path)
                continue
            try:
                os.kill(pid, signal.SIGTERM)
            except OSError:
                remove_file(path)
                continue
            self._say(f"stop-all: SIGTERM waiting guard pid {pid} ({rec.get('kind')}: {rec.get('cmd')})")
            guards.append((pid, path))
        for path, rec in self._records("running"):
            pgid = rec.get("pgid")
            if not isinstance(pgid, int) or pgid <= 1:
                remove_file(path)
                continue
            if pgid == my_pgid:
                self._say(f"stop-all: skipping pgid {pgid}: it is this process's own group")
                continue
            if not self.group_alive(pgid):
                remove_file(path)
                continue
            trusted = self._is_heavy_guard(rec.get("guard"))
            if not trusted and rec.get("lstart"):
                trusted = self.proc_start(pgid) == rec.get("lstart")
            elif not trusted:
                leader = self.proc_command(pgid) or ""
                argv0 = os.path.basename(str(rec.get("argv0") or ""))
                trusted = bool(argv0) and argv0 in leader
            if not trusted:
                self._say(f"stop-all: skipping pgid {pgid}: its guard is gone and its leader "
                          "is not the process we started; check it by hand")
                remove_file(path)
                continue
            try:
                os.killpg(pgid, signal.SIGTERM)
            except OSError:
                remove_file(path)
                continue
            self._say(f"stop-all: SIGTERM process group {pgid} ({rec.get('kind')}: {rec.get('cmd')})")
            groups.append((pgid, path))
        if not guards and not groups:
            self._say("stop-all: nothing to stop")
            return 0
        deadline = self.clock.monotonic() + max(grace, 0.0)
        while self.clock.monotonic() < deadline:
            if not any(self.pid_alive(p) for p, _ in guards) and \
                    not any(self.group_alive(g) for g, _ in groups):
                break
            self.clock.sleep(0.2)
        for pgid, path in groups:
            if self.group_alive(pgid):
                try:
                    os.killpg(pgid, signal.SIGKILL)
                    self._say(f"stop-all: SIGKILL process group {pgid}: still alive after {fmt_secs(grace)}")
                except OSError:
                    pass
            remove_file(path)
        for pid, path in guards:
            if self.pid_alive(pid):
                self._say(f"stop-all: waiting guard pid {pid} is still alive after {fmt_secs(grace)}")
            else:
                remove_file(path)
        return 0

    # -- mode ----------------------------------------------------------------------------

    def set_mode(self, mode: Optional[str], window_text: Optional[str]) -> int:
        now = self.clock.now()
        if mode is None:
            if window_text:
                self._warn("heavy: --window needs a mode, e.g. heavy.py mode nightshift --window 22:00-06:00")
                return EX_USAGE
            self._say(self._mode_line(self.mode_state(), now))
            return 0
        if mode not in MODES:
            self._warn(f"heavy: unknown mode {mode!r}; use one of {', '.join(MODES)}")
            return EX_USAGE
        if window_text and mode != "nightshift":
            self._warn("heavy: --window only applies to nightshift")
            return EX_USAGE
        try:
            window = Window.parse(window_text or DEFAULT_WINDOW)
        except ValueError as exc:
            self._warn(f"heavy: {exc}")
            return EX_USAGE
        data: Dict[str, object] = {"mode": mode, "since": iso(now)}
        if mode == "nightshift":
            data["window"] = str(window)
        try:
            write_json_atomic(self.mode_path, data)
        except OSError as exc:
            self._warn(f"heavy: cannot write {self.tilde(self.mode_path)}: {exc}")
            return 1
        self._say(self._mode_line(ModeState(mode, window), now))
        return 0

    def _mode_line(self, state: ModeState, now: _dt.datetime) -> str:
        if state.error:
            return f"mode: invalid ({state.error})"
        if state.mode == "nightshift":
            inside = "yes" if state.in_window(now) else "no"
            return f"mode: nightshift, window {state.window}, inside the window now ({now:%H:%M}): {inside}"
        return f"mode: {state.mode}"

    # -- status --------------------------------------------------------------------------

    def status(self) -> int:
        now = self.clock.now()
        state = self.mode_state()
        mem = self.read_memory()
        say = self._say

        say(f"run dir:   {self.tilde(self.run_dir)}"
            + (" (CI mode: no run dir, continuous, Linux readers)" if self.ci_mode else ""))
        if state.error:
            say(f"mode:      invalid: {state.error} (non-light kinds are refused)")
        else:
            say(f"mode:      {state.mode}" + (" (no mode.json)" if state.source == "default" else ""))
        if state.mode == "nightshift" and not state.error:
            say(f"window:    {state.window}")
            say(f"in window: {'yes' if state.in_window(now) else 'no'} (local time {now:%H:%M})")
        else:
            say("window:    none (only nightshift has one)")
            say("in window: n/a")

        say(f"pressure:  {level_name(mem.level)}"
            + (f" (level {mem.level})" if mem.level is not None else " (treated as warn)")
            + f" [{mem.source}]")
        say("free mem:  " + ("unknown" if mem.free_pct is None else f"{mem.free_pct:.0f}%"))
        if mem.swap_total_mb is None:
            say("swap:      unknown")
        else:
            say(f"swap:      {mem.swap_free_mb or 0:.0f} MB free of {mem.swap_total_mb:.0f} MB")

        cap = self.capacity(mem, state, now)
        why = f"{level_name(mem.level)} pressure"
        if state.in_window(now):
            why += ", nightshift window"
        if self.max_slots() is not None:
            why += f", AJAYEB_MAX_SLOTS={self.max_slots()}"
        slots = self.locks.slot_names()
        live = [n for n in slots if self.locks.state(n) == "live"]
        say(f"slots:     capacity {cap} ({why}), {len(live)} held")
        for name in slots:
            lock_state = self.locks.state(name)
            mark = "" if lock_state == "live" else " (stale, cleared on the next run)"
            say(f"  {name:<8} {self._describe(self.locks.owner(name))}{mark}")

        min_free, min_swap = self.browser_floors()
        for kind in KIND_LOCKS:
            lock_state = self.locks.state(kind)
            if lock_state == "free":
                line = "free"
            else:
                line = self._describe(self.locks.owner(kind))
                if lock_state == "stale":
                    line += " (stale, cleared on the next run)"
            if kind == "browser":
                block = browser_block_reason(mem, min_free, min_swap)
                line += (f"; launch needs free mem >= {min_free:g}% and free swap >= {min_swap:g} MB: "
                         + ("ok" if block is None else f"blocked, {block}"))
            say(f"{kind + ':':<10} {line}")

        waiting = [rec for _, rec in self._records("waiting") if self.pid_alive(rec.get("guard"))]
        say(f"waiting:   {len(waiting)}")
        for rec in waiting:
            age = time.time() - float(rec.get("t0") or time.time())
            say(f"  pid {rec.get('guard')} {rec.get('kind')} {fmt_secs(age)}: {rec.get('cmd')}")
        running = self._records("running")
        say(f"running:   {len(running)}")
        for _, rec in running:
            age = time.time() - float(rec.get("t0") or time.time())
            gone = "" if self.group_alive(rec.get("pgid")) else " (gone)"
            nested = " nested" if rec.get("nested") else ""
            say(f"  pgid {rec.get('pgid')} {rec.get('kind')}{nested} {fmt_secs(age)}: {rec.get('cmd')}{gone}")

        recent = self._recent(5)
        say(f"recent:    {'none' if not recent else f'last {len(recent)} of heavy.jsonl'}")
        for entry in recent:
            rss = entry.get("maxrss_kb") or 0
            extra = f" refused: {entry['refused']}" if entry.get("refused") else ""
            say(f"  {entry.get('ts')} {entry.get('kind')} rc={entry.get('rc')} "
                f"{entry.get('secs')}s {rss / 1024:.0f} MB: {entry.get('cmd')}{extra}")
        return 0

    def _recent(self, n: int) -> List[dict]:
        try:
            with open(self.log_path, encoding="utf-8") as fh:
                lines = collections.deque(fh, maxlen=n)
        except OSError:
            return []
        out = []
        for line in lines:
            try:
                entry = json.loads(line)
            except ValueError:
                continue
            if isinstance(entry, dict):
                out.append(entry)
        return out


def _nice10() -> None:
    try:
        os.nice(10)
    except OSError:
        pass


# ----------------------------------------------------------------------------
# CLI


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="heavy.py", description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", metavar="COMMAND")
    sub.required = True

    run = sub.add_parser("run", help="run CMD under the guard: heavy.py run --kind KIND -- CMD...",
                         description="Run CMD under the guard. Put -- before the command.")
    run.add_argument("--kind", required=True, choices=KINDS)
    run.add_argument("--wait", type=float, default=None, metavar="SECONDS",
                     help="how long to wait for a slot or lock (default 1800 or $AJAYEB_HEAVY_WAIT)")

    sub.add_parser("status", help="show mode, window, pressure, slots, locks and recent runs")

    stop = sub.add_parser("stop-all", help="stop every job this script started")
    stop.add_argument("--grace", type=float, default=DEFAULT_GRACE, metavar="SECONDS",
                      help="seconds between SIGTERM and SIGKILL (default 10)")

    mode = sub.add_parser("mode", help="show or set the mode")
    mode.add_argument("mode", nargs="?", choices=MODES)
    mode.add_argument("--window", metavar="HH:MM-HH:MM",
                      help=f"nightshift window, local time (default {DEFAULT_WINDOW})")
    return parser


def main(argv: Optional[Sequence[str]] = None) -> int:
    args_in = list(sys.argv[1:] if argv is None else argv)
    cmd: Optional[List[str]] = None
    if "--" in args_in:
        cut = args_in.index("--")
        args_in, cmd = args_in[:cut], args_in[cut + 1:]
    parser = build_parser()
    args = parser.parse_args(args_in)
    if args.command != "run" and cmd is not None:
        parser.error("only the run command takes -- CMD")
    guard = Guard()
    if args.command == "run":
        if cmd is None:
            parser.error("run needs -- before the command, e.g. heavy.py run --kind test -- go test ./...")
        return guard.run(args.kind, cmd, args.wait)
    if args.command == "status":
        return guard.status()
    if args.command == "stop-all":
        return guard.stop_all(args.grace)
    if args.command == "mode":
        return guard.set_mode(args.mode, args.window)
    parser.error(f"unknown command {args.command!r}")
    return EX_USAGE


if __name__ == "__main__":
    sys.exit(main())
