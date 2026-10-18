# RAM policy and the heavy-job guard

Ajayeb is developed on a laptop that is also in daily use and often runs under memory pressure. Builds, test runs, linters, browsers and image generation can each take hundreds of megabytes, and several agents may want them at the same moment. So every heavy job goes through one guard, `tools/heavy.py`, which decides whether it may start now. The guard is also the hard enforcement of the run modes described in [orchestration.md](orchestration.md).

## The guard

`tools/heavy.py` is a Python standard-library script, tested by `tests/test_heavy.py` with injected readers and an injected clock.

```
python3 tools/heavy.py run --kind test|build|check|lint|browser|codex|install|fuzz|bench|light [--wait 1800] [--priority normal|high] [--notify UUID|auto] -- CMD...
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
queue checks, acquisition, release and the status resource snapshot. Jobs retry at a fixed one-second interval, so a
freed resource normally passes to the next eligible job within one or two
seconds. The mutex itself uses short, deadline-aware retries. `--wait` (or
`AJAYEB_HEAVY_WAIT`, default 1800 seconds) bounds waiting; expiration exits 75.
Mode refusals remain immediate. The slot cap and browser floors are resolved
once when the guard is created, and malformed settings warn before any
admission mutex is held.
Eligibility checks and the status resource snapshot use silent configuration
getters; all terminal output stays outside admission.

The queue follows the resource that a job can actually use:

- An earlier job waiting for a general slot blocks later jobs needing slots,
  even if two or three slots have just become free. Later jobs wait for that
  ticket to acquire its slot. This strict FIFO rule trades up to one polling
  interval of spare capacity for predictable admission order.
- An earlier browser waiting for the browser lock blocks later browsers, while
  tests and linters can pass. The same rule applies to the codex lock.
- A browser held by its memory floor reserves nothing. A mode-blocked ticket
  also reserves nothing and is removed as soon as its guard observes refusal.
- Nested runs never publish tickets or join the queue. A nested browser or
  codex run retries the physical kind lock it does not yet hold. A nested
  browser must also pass the browser memory and swap floors before taking a
  new browser lock; inheriting a browser lock skips that check. This safety
  gate can keep the parent's slot occupied until memory recovers or the wait
  expires. It is intentional. Bypassing the queue is a fairness exception:
  it can pass top-level tickets, which may themselves need its parent's held slot. Queuing behind them could deadlock.

Every ticket retains its browser floor settings. Every eligibility check
recomputes the earlier tickets' own gates and resources,
so a sleeping waiter's last recorded reason cannot cause head-of-line blocking.
Memory samples are reused within one admission poll; every retry, including
its final fractional wait, reads memory again. Process identities come from
native birth seconds and microseconds on macOS and boot ID plus process start
ticks on Linux. They are independent of timezone and realtime clock changes,
and avoid a `ps` subprocess per ticket per poll. If identity cannot be read,
retain the live process's ticket.

Only the coordinator grants high priority. It can use `--priority high` for
one run or `HEAVY_PRIORITY=high` for a team's environment; explicit CLI priority
overrides the environment. High-priority jobs go before normal jobs, FIFO
among themselves. Sustained high-priority traffic can delay normal work, so
priority is for coordinated exceptions.

Use the queue instead of manually checking status and yielding to other
teams. Do not build agent-side retry loops. `status` lists queue position,
priority, kind, age and the last blocked resource or gate. Tickets disappear
when jobs start, fail, time out, are refused or receive a handled signal.
Any waiter or status check removes dead or strongly identified PID-reused
tickets. Status also removes dead or strongly identified PID-reused waiting
records. Legacy records without immutable identities are retained while their
PID is alive, because their local start timestamp cannot distinguish PID
reuse from a timezone or clock change.

Worktrees with older guard copies still share the same atomic slot and kind
locks. New lock, ticket and waiting records deliberately omit legacy local
start timestamps: old readers then retain them while the PID is alive, rather
than mistaking a different timezone for PID reuse. New readers also retain
live legacy holders conservatively. This makes old/new pairs safe across
timezones, at the cost of retaining a legacy record if its PID was reused by
a live unrelated process. Rebase old worktrees to restore strong reuse checks;
two old copies with different timezones can still break each other's locks.
Old copies ignore tickets and do not serialize release with admission. They
can repeatedly overtake waiters or disrupt handover order. Strict FIFO needs
updated guards for both admission and release. `status` marks their waiting records as
`NO TICKET: legacy guard, may overtake queue`. Updated guards temporarily
without a ticket (starting or inheriting resources) are identified separately.
Rebase old worktrees to gain fairness. Automatic delegation to a newer main
checkout guard is not implemented.

### Locks and processes

- Each slot is a lock directory created atomically with `os.mkdir` (`heavy/slot-N/`, `heavy/browser/`, `heavy/codex/` in the run directory), holding `owner.json`: the guard's PID, the command's process group, and their immutable birth identities. A lock is stale when neither process is alive or when a strong identity proves PID reuse. An unreadable or legacy identity keeps a live PID's lock.
- The lock directories are machine-wide, so agents in different worktrees share the same slots.
- The child runs in its own session (process group). Signals are forwarded to the group.
- Before spawning, handled signals are deferred across acquisition and ownership
  handoff, then stop the run before its child starts. Deferral ends before the
  final pending-signal check, so a signal after that check also stops startup.
- Waiting and refusal diagnostics are interruptible. Warnings write directly
  to a file descriptor when available, so an interrupted buffered write cannot
  be retried at interpreter shutdown. A warning that cannot be written (a
  closed pipe, a full disk) is dropped, so it can never crash a run or orphan
  a started child. A signal deferred during acquisition is raised before any
  diagnostic or poll sleep, and one deferred during a failed `Popen` still
  cancels the run. Pre-start cancellation exits without a
  final terminal diagnostic, because another write to a blocked stderr pipe
  could prevent shutdown.
- Cleanup ignores repeated handled signals and releases all locks carrying
  the run's ownership token, including locks acquired before a return was
  assigned. Release is serialized with admission and removes kind locks before
  slots. An unhandled kill can leave records for the next stale sweep.
- `stop-all` requires a matching immutable identity when a running or waiting record carries one. An unreadable or mismatching strong identity never falls back to a command name. Legacy running records keep their local start timestamp for old `stop-all` readers; a mismatch after the guard dies skips the group, so a cross-timezone legacy stop request can require manual inspection.
- These defaults are set for the child when unset or empty: `NODE_OPTIONS=--max-old-space-size=1536`, `GOMEMLIMIT=768MiB`, `GOFLAGS=-p=2`, `GOMAXPROCS=4`. A `NODE_OPTIONS` or `GOFLAGS` that is set but has no heap or `-p` flag gets ours appended.
- Fuzzing and other long CPU-bound runs get `nice 10`.
- Nested calls pass straight through: the child of a run that holds a slot gets `AJAYEB_HEAVY_HELD=1`, so a wrapped `make verify` that calls other guarded targets cannot deadlock on its own slot. A nested `browser` or `codex` run still takes its own lock unless an ancestor already holds it. A `light` run holds no slot, so it does not set `AJAYEB_HEAVY_HELD`.
- Every run appends `{ts, kind, cmd, rc, secs, maxrss_kb}` to `logs/heavy.jsonl` in the run directory, with the peak RSS taken from `getrusage(RUSAGE_CHILDREN)`. The budgets below are corrected from these measurements over time.
- With `CI=true` and no run directory, the guard runs in continuous mode with the Linux readers, so CI needs no setup.

### Detached runs that notify their agent

An agent waiting on `heavy.py run --wait 3600` keeps its turn open, and its harness checks on it with model turns that reread the whole context (Codex teams measured about one turn a minute at about 100k cached tokens each). A **detached run** costs no turns while it waits: the agent starts the job, ends its turn, and is woken by one Paseo message when the job ends.

- Ask for it with `HEAVY_NOTIFY=<Paseo agent UUID>` in the environment or `--notify <UUID>`. The value `auto` reads `PASEO_AGENT_ID`, so `HEAVY_NOTIFY=auto` is all a Paseo agent needs. An unset or empty value (or `0`, `off`, `no`, `false`) keeps the blocking run; any other value that is not a UUID is a usage error (exit 2). Put it on the outermost command only, never in an exported environment: a script that runs several guarded steps would see each one detach and return 0 at once.
- The guard first applies the mode gate, so a paused or out-of-window mode still refuses at once with exit 75. Otherwise it starts a copy of itself in a new session, with standard input from `/dev/null` and its output and the command's output in `logs/heavy-notify/<time>-<kind>-<id>.log` in the run directory. The caller waits only until that copy has published its queue ticket (a `light` run, which takes no ticket: until the copy is about to start the command), for at most a minute, prints the copy's pid, the ticket and its place in the queue, the log path and "end your turn; you will be messaged", and exits 0. The ticket is taken before the caller returns, so the job keeps its place in line.
- The copy is an ordinary guard: same queue, same tickets, same `waiting` and `running` records, so `status` shows it and `stop-all` stops it. The queue semantics are unchanged.
- When the job ends, whether it succeeded, failed, was refused (75), timed out in the queue or was stopped by a signal, the copy sends one message with `paseo send <UUID> "<text>" --no-wait --json` (`PASEO_BIN`, default `paseo`). The text gives the kind, a short form of the command, the outcome and exit code, the time spent queued and running, the log path and the last 12 lines of the log, with terminal colours removed.
- If the copy is refused before it publishes a ticket (the mode changed in the moment between the caller's check and the copy's), the caller prints the refusal and exits 75 itself, and no message is sent. The caller acknowledges that report on a second pipe; if no acknowledgement arrives within 10 seconds (the caller has gone or stopped waiting), the copy messages the agent instead.
- **Keep-warm pings.** While a job waits or runs, the copy sends a one-line ping asking for a one-word reply when the agent has heard nothing from it for the provider's interval: 25 minutes for a Codex agent (OpenAI keeps a GPT-6.1 cached prefix for 30 minutes after its last use) and 50 minutes for any other (Claude's prompt cache lasts about an hour). The provider is read once from the `Provider` field of `paseo inspect <UUID> --json`; when it cannot be read, the 50-minute interval applies and the guard asks again five minutes later. A ping that falls due while the agent's `Status` is `running` is skipped and counts as contact: the agent's cache is warm, and a message arriving mid-turn cuts off its current tool call. For the same reason the outcome waits for a busy agent to go idle, checking every 15 seconds for up to 5 minutes, and is then sent anyway. Pings stop 3.5 hours after the job was started, because past that point a cold reread costs less than further pings. A job that ends within its interval sends no ping at all. `HEAVY_WARM_CODEX`, `HEAVY_WARM_CLAUDE` and `HEAVY_WARM_MAX` override the seconds (1500, 3000, 12600). This is the policy of the integration lock bell.
- **The outcome is never silently lost.** A failed send is retried three times, after 10, 30 and 90 seconds. If all four attempts fail, the copy writes `alerts/quota-<epoch>-heavy-notify-failed.md` in the run directory (the coordinator's waiter watches `quota-*.md`), naming the agent, the command, the outcome, the log and the text it tried to send. A failed ping is only logged, and retried five minutes later. No ping is sent once the outcome is being sent. While sending the outcome the copy ignores SIGINT, SIGTERM and SIGHUP (by then `stop-all` no longer lists it), so only a `SIGKILL` of the copy itself loses the message.
- The `paseo` calls are children of the copy, so a job that ran long enough to be pinged can show the Paseo CLI's peak RSS in its `heavy.jsonl` line when that is larger than the job's own.
- Nested runs (`AJAYEB_HEAVY_HELD=1`) ignore `HEAVY_NOTIFY` and run inline, because their ancestor holds the slot they run under. The command never sees `HEAVY_NOTIFY`, so the guarded steps inside a detached `make` block as usual.
- **`make` targets.** Most targets run several guarded steps, and each step would detach on its own, so a `HEAVY_NOTIFY` that reaches the guard under `make` (`MAKELEVEL` set) is a usage error. Detach the whole target with one `light` run instead, which holds no slot while the steps inside queue one by one:

```
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind light -- make verify-quick
HEAVY_PRIORITY=high HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind light -- make verify-quick
```

  An explicit `--notify` is allowed under `make`, for a recipe that wraps a sub-make on purpose.

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
