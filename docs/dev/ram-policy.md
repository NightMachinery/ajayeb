# RAM policy and the heavy-job guard

Ajayeb is developed on a laptop that is also in daily use and often runs under memory pressure. Builds, test runs, linters, browsers and image generation can each take hundreds of megabytes, and several agents may want them at the same moment. So every heavy job goes through one guard, `tools/heavy.py`, which decides whether it may start now. The guard is also the hard enforcement of the run modes described in [orchestration.md](orchestration.md).

## The guard

`tools/heavy.py` is a Python standard-library script, tested by `tests/test_heavy.py` with injected readers and an injected clock.

```
python3 tools/heavy.py run --kind test|build|check|lint|browser|codex|install|fuzz|bench|light [--wait 1800] [--priority normal|high] -- CMD...
python3 tools/heavy.py status     # pressure, free level, swap, mode, window, slot holders, ordered queue, recent peak RSS per kind
python3 tools/heavy.py stop-all   # SIGTERM the process groups the guard started (recorded PIDs only)
python3 tools/heavy.py mode continuous|nightshift|paused [--window 00:00-10:00]
```

`python3 tools/heavy.py --help` is the authoritative reference for flags and kinds.

### The mode gate

- The mode is read from `mode.json` in the private run directory (`~/tmp/agents/ajayeb/`, or `AJAYEB_RUN_DIR`). If the file is missing, the mode is `continuous`.
- `paused` refuses every kind except `light`.
- `nightshift` refuses every kind except `light` outside its window.
- A refusal exits with code 75 (`EX_TEMPFAIL`) and prints a one-line reason.

### The pressure gate

On macOS the guard reads the kernel's memory pressure level, the free-memory level and swap usage (`kern.memorystatus_vm_pressure_level`, `kern.memorystatus_level`, `vm.swapusage`). On Linux it reads `MemAvailable` from `/proc/meminfo`. The number of heavy jobs allowed at once follows the pressure:

- critical pressure: no heavy slots;
- warning pressure: one slot;
- normal pressure: two slots in continuous mode, three inside the nightshift window.

On top of the general slots there are two separate locks of capacity one. A `browser` or `codex` run takes its own lock and a general slot as well:

- `browser`: one headless Chrome on the machine at a time. A browser also needs the free-memory level to be at least 14%, and on Linux at least 512 MB of swap free. macOS adds swap files as it needs them, so its free swap is only the unused room in the files made so far and is not checked there. Both thresholds can be tuned through environment variables.
- `codex`: one image-generation job at a time.

### The admission queue

Jobs publish atomic tickets under `heavy/queue/` before attempting admission.
Tickets are ordered by priority, then creation time, giving FIFO order among
jobs with equal priority. A brief admission mutex serializes ticket creation,
queue checks and acquisition. Jobs retry at a fixed one-second interval, so a
freed resource normally passes to the next eligible job within one or two
seconds. The mutex itself uses short, deadline-aware retries. `--wait` (or
`AJAYEB_HEAVY_WAIT`, default 1800 seconds) bounds waiting; expiration exits 75.
Mode refusals remain immediate.

The queue follows the resource that a job can actually use:

- An earlier job waiting for a general slot blocks later jobs needing slots.
- An earlier browser waiting for the browser lock blocks later browsers, while
  tests and linters can pass. The same rule applies to the codex lock.
- A browser held by its memory floor reserves nothing. A mode-blocked ticket
  also reserves nothing and is removed as soon as its guard observes refusal.
- Nested runs never publish tickets or join the queue. A nested browser or
  codex run retries only the physical kind lock it does not yet hold. This is
  a fairness exception: it can pass top-level tickets, which may themselves
  need its parent's held slot. Queuing behind them could deadlock.

Every ticket retains its browser floor settings. Every eligibility check
recomputes the earlier tickets' own gates and resources,
so a sleeping waiter's last recorded reason cannot cause head-of-line blocking.
Memory samples are reused for at most one second. Process start times come
from native libproc on macOS and procfs on Linux, avoiding a `ps` subprocess
per ticket per poll. The existing conservative rule applies when identity
cannot be read: retain the live process's ticket.

Only the coordinator grants high priority. It can use `--priority high` for
one run or `HEAVY_PRIORITY=high` for a team's environment; explicit CLI priority
overrides the environment. High-priority jobs go before normal jobs, FIFO
among themselves. Sustained high-priority traffic can delay normal work, so
priority is for coordinated exceptions.

Use the queue instead of manually checking status and yielding to other
teams. Do not build agent-side retry loops. `status` lists queue position,
priority, kind, age and the last blocked resource or gate. Tickets disappear
when jobs start, fail, time out, are refused or receive a handled signal.
Any waiter or status check removes dead or PID-reused tickets.

Worktrees with older guard copies still share the same atomic slot and kind
locks, so the queue protocol does not double-allocate resources with them.
They ignore tickets and can repeatedly overtake waiting jobs; FIFO applies
only among updated guards. `status` marks their waiting records as
`NO TICKET: legacy guard, may overtake queue`. Updated guards temporarily
without a ticket (starting or inheriting resources) are identified separately.
Rebase old worktrees to gain fairness. Automatic delegation to a newer main
checkout guard is not implemented.

### Locks and processes

- Each slot is a lock directory created atomically with `os.mkdir` (`heavy/slot-N/`, `heavy/browser/`, `heavy/codex/` in the run directory), holding `owner.json`: the guard's PID, the command's process group, and the start time of each. A lock is stale, and the next run clears it, when its owner no longer exists or when its PID now belongs to another process (the start time differs). A start time that cannot be read keeps the lock, so a live lock is never broken.
- The lock directories are machine-wide, so agents in different worktrees share the same slots.
- The child runs in its own session (process group). Signals are forwarded to the group, and the slot is released in a `finally` block, so a crash or an interrupt cannot leak a slot.
- These defaults are set for the child when unset or empty: `NODE_OPTIONS=--max-old-space-size=1536`, `GOMEMLIMIT=768MiB`, `GOFLAGS=-p=2`, `GOMAXPROCS=4`. A `NODE_OPTIONS` or `GOFLAGS` that is set but has no heap or `-p` flag gets ours appended.
- Fuzzing and other long CPU-bound runs get `nice 10`.
- Nested calls pass straight through: the child of a run that holds a slot gets `AJAYEB_HEAVY_HELD=1`, so a wrapped `make verify` that calls other guarded targets cannot deadlock on its own slot. A nested `browser` or `codex` run still takes its own lock unless an ancestor already holds it. A `light` run holds no slot, so it does not set `AJAYEB_HEAVY_HELD`.
- Every run appends `{ts, kind, cmd, rc, secs, maxrss_kb}` to `logs/heavy.jsonl` in the run directory, with the peak RSS taken from `getrusage(RUSAGE_CHILDREN)`. The budgets below are corrected from these measurements over time.
- With `CI=true` and no run directory, the guard runs in continuous mode with the Linux readers, so CI needs no setup.

## Enforcement

- Every `Makefile` target that compiles, tests, builds, lints, installs, launches a browser or runs image generation is wrapped in the guard.
- [AGENTS.md](../../AGENTS.md) tells every agent to use `make <target>`, never raw `go test ./...`, `pnpm build` or `playwright`.
- The screenshot and SVG render tools refuse to launch Chrome unless they run inside the guard's browser lock (`AJAYEB_HEAVY_KIND=browser`) or are given `--force`.

## Budgets

These are planning estimates of transient peaks, to be replaced by the measurements in `heavy.jsonl`:

- the Go game server: about 15 MB;
- `go build` or `go test -p 2`: 300 to 800 MB, and more with `-race` (which runs only in `make verify`);
- golangci-lint: 300 to 700 MB;
- svelte-check: 400 to 800 MB;
- `vite build`: 400 to 800 MB;
- the Vite dev server: 300 to 500 MB;
- a Playwright run: about 150 MB for the runner, plus 250 to 400 MB for Chrome, plus about 100 MB per page;
- `codex exec`: 150 to 300 MB;
- `pnpm install`: 200 to 400 MB.

The target ceiling for all of our processes together is about 1.5 GB in continuous mode and about 2.5 GB inside the nightshift window.

## Other policies

- No Vite dev server during unattended work. End-to-end tests and screenshot matrices use the test build, served by the Go binary.
- Browsers always close at the end of a run. Check with `/bin/ps -axo pid=,command= | grep -F -e ajayeb-chrome -e playwright_chromiumdev_profile | grep -v grep`, which prints nothing when none of ours is left. (`pgrep -f` also matches the shell that runs it, and a `--headless` pattern matches other projects' browsers.)
- Image-generation jobs run strictly one after another.
- At most three worktrees exist at once. Web worktrees install with `make install-offline` (an offline, frozen-lockfile pnpm install from the shared store, which hardlinks instead of copying). Go worktrees share the module and build caches and need nothing.
- No dev servers in worktrees, ever.
- The coordinator's own process grows with every in-process agent, so the number of live workflow agents is capped: four in continuous mode, six in nightshift. Workflow scripts enforce this with a small concurrency limiter.
- The game server runs in production with `GOMEMLIMIT=96MiB`.
- The heaviest suites (race tests, long sweeps, fuzzing, the full Playwright run) also run in GitHub Actions, which takes them off the laptop entirely.
- Renovate (`renovate.json`) opens dependency update PRs once a week, grouped into one PR for all minor, patch, pin and digest updates plus a separate PR per major update, at most three open and two opened per hour. TypeScript is held below 7 (`allowedVersions: "<7"`) because svelte-check 4 does not support it yet. Every Renovate PR runs the same CI as any other PR and is merged by the coordinator once it is green.
