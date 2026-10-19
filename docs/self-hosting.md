# Self-hosting

`self_host.py` runs Ajayeb behind Caddy on a single machine, with tmux
keeping every long-running process easy to inspect or restart without
Docker. It is adapted from the snow-white project's `self_host.py`.

## Commands

Run `./self_host.py <command> [url]` from the repository root. The url
argument is optional; it defaults to the site recorded in
`.self-host/config.json`, or `https://ajayeb.lilf.ir` if nothing is
recorded yet.

- `setup [url] [--prebuilt PATH]`: first-time install. Installs web
  dependencies, builds the production web bundle, builds the Go server with
  the bundle embedded, writes the Caddy block, has Caddy load it and starts
  the server. With `--prebuilt PATH`, skips the dependency install, the web
  build, the webdist sync and the Go build, and installs `PATH` instead; see
  "Deploying from another machine" below.
- `redeploy [url] [--prebuilt PATH]`: rebuilds and restarts for a new
  version. Does the same steps as `setup` without the dependency dedupe
  pass, and archives the current `data/rooms` directory after stopping the
  server and before installing the new binary. `--prebuilt`
  works the same way as for `setup`.
- `start [url]`: starts the already-built server. Builds it first only if
  `.self-host/bin/ajayeb` does not exist yet. Safe to run at boot: an
  already-correct Caddyfile is left untouched, and the server still
  starts even if Caddy configuration fails or Caddy is not running yet.
  A known Caddy failure still makes the command exit nonzero.
  See "Starting at boot" below.
- `stop`: stops every tmux session this script manages. Leaves the Caddyfile
  and the built binary alone.
- `dev-start [url]`: starts a development setup instead of production. Runs
  the Go server under `go tool air` for live reload, and the Vite dev server
  for the web client, with the Caddyfile pointed at both.
- `status [url]`: prints the recorded configuration, whether each tmux
  session is running (with its last 20 lines of output), whether its port is
  free, and the result of a server health check.
- `ship-all`: builds once and deploys every host in the private targets
  file; see "Deploying to several hosts" below.
- `ship TARGET [url] [--remote-dir DIR]`: run on the dev machine, not on the
  production host. Cross-compiles for the target and installs the result
  there over SSH; see "Deploying from another machine" below. Once a `TARGET`
  has been remembered, `ship URL` (a bare `http://` or `https://` argument in
  `TARGET`'s position) is read as the url and reuses the remembered target.

`setup`, `redeploy`, `start`, `dev-start` and `ship` also take
`--both-schemes` or `--no-both-schemes`; see "Serving both schemes" below.
Without either flag they keep the recorded choice, which is off for a new
install. `ship` passes the flag on to the remote `setup` or `redeploy`.

The same commands take `--caddyfile PATH`, the Caddyfile the running Caddy
reads. In this document, **the Caddyfile** means that file: `~/Caddyfile`
unless `--caddyfile` has named another. `PATH` is absolute or relative to
the home directory (write `scripts/Caddyfile`, not `~/scripts/Caddyfile`, so
your own shell does not expand the `~` before `ship` forwards it to the
remote host). The choice is recorded like the others, so later runs and the
boot-time `start` keep it. A file that `--caddyfile` names must already
exist; see "Caddy safety" below for why it is never created.

## Processes

Three tmux sessions, never more than two running at once:

- `ajayeb-server`: the production Go server, started by `setup`, `redeploy`
  and `start`.
- `ajayeb-dev-server`: the Go server under `go tool air`, started by
  `dev-start`. Air rebuilds `server/cmd/ajayeb` on save, from
  `server/.air.toml`, with `--dev --dev-endpoints`.
- `ajayeb-dev-web`: the Vite dev server, started by `dev-start`.

`stop` kills all three sessions by name, so it is always safe to run even if
only one of them is up.

## Ports

Ports are scanned upward from 38951 (the Go server) and 38952 (the Vite dev
server) if the preferred port is busy, and the chosen ports are written to
`.self-host/config.json` so a later command reuses them. `setup`, `redeploy`
and `start` choose the ports only after stopping the running server (waiting
up to 5 seconds for its socket to close), so a redeploy keeps the server's
port and leaves the Caddyfile alone instead of moving to the next free port.
`go tool air`'s
entrypoint is fixed by `server/.air.toml` to the server's default address, so
in dev mode the Go server itself always binds 38951; only the Vite dev
server's port is reselected when busy.

## Configuration file

`.self-host/config.json` records the site URL, the mode (`prod` or `dev`),
the chosen ports, whether both schemes serve the site (`both_schemes`), the
Caddyfile when it is not `~/Caddyfile` (`caddyfile`), and on the dev machine
the remembered ship target. It is not committed; `.self-host/` is gitignored
entirely, since it holds local build output, the production data directory
and Caddyfile backups.

## Build

`setup`, `redeploy`, and `start` (when no binary exists yet) build the web
client and the Go server, unless `--prebuilt PATH` is given (see "Deploying
from another machine"):

1. `CI=true pnpm install --frozen-lockfile` (with a dependency dedupe pass on
   `setup` only) and `pnpm -C web build`, both run under `zsh -lc` with
   `nvm-load` and `nvm use 24`.
2. `rsync -a --delete` the built `web/build/` into
   `server/internal/webdist/dist/`.
3. `go build -tags embedweb -trimpath -ldflags "-s -w -X main.version=<version>"`
   to a new binary file. The version comes from `git describe --always
   --dirty`. Go is found with `shutil.which('go')`, falling back to
   `/usr/local/go/bin/go` or the Homebrew path.

Once the new binary is built (or, with `--prebuilt`, checked), the script
runs the candidate room preflight, stops the running server, repeats the check,
backs up the current `.self-host/bin/ajayeb` to
`.self-host/bin/ajayeb.prev`, and atomically installs the new one in its
place. Stopping happens only at this point, after every build and check, so
a failing build or a bad `--prebuilt` path never takes a running server
down.

## Room compatibility preflight

`setup` and `redeploy`, including the remote commands used by `ship` and `ship-all`, run the candidate binary on the target data before stopping the old server:

```
.self-host/bin/ajayeb.upload --check-rooms --data .self-host/data
```

The candidate emits one JSON report with format `ajayeb-room-check/1` and counts for room statuses, playable rooms, hashes it could refresh, rooms it would archive and read/schema failures. It opens no socket, initializes no identity, makes no directories and changes no room bytes. The check scans both `data/rooms` and `data/rooms-archive`; either missing folder is empty. Records in the archive folder must have archived status and never count as live rooms at risk. Directories and JSON files without valid room-code names are ignored, matching the server's room listing; permission, decoding and schema errors in room files fail closed. Unknown newer schemas are preserved. Exit 0 means the check permits deployment; exit 3 means in-progress rooms (lobby or playing) would archive; exit 1 means an operational/read/schema failure. A finished room that would archive is counted but does not require the live-room override.

A clean replay in the same `engine.RulesVersion` can refresh an old state hash without ending the game or resetting its idle clock. Incompatible setup/RNG/catalog/rules changes must bump that version. Invalid action logs or a different rules version archive on ordinary load with a durable reason and timestamp; an archive never reopens automatically. See [D28](dev/decisions.md#d28-saved-rooms-are-canonical-within-an-explicit-rules-version) for the integrity tradeoff and [the CLI contract](protocol.md#read-only-room-check-cli) for the report fields.

Live-room loss refuses before the stop. To deliberately accept it for this invocation, pass `--force-room-archive` to `setup`, `redeploy`, `ship` or `ship-all`. Shipping forwards the flag to the target; `ship-all` applies it to every target in that invocation. It is never remembered in configuration. Force does not bypass schema, read or operational failures. The check prints the affected count and an explicit force warning; raw action keys, hashes and engine errors are absent from the report.

The first scan can race active moves and the reaper. After stopping, the script waits for the old port to be released and repeats the candidate scan before any data backup, configuration save or binary swap. If this second check fails, the old binary and data stay in place and the script attempts to restart that binary on its prior port/configuration, checking health and reporting any restart failure. It retains the candidate for inspection. This assumes stopping the service has quiesced room writers; it does not lock out an unrelated process that edits files afterward. The normal `.prev` binary recovery remains unchanged, and later startup/publication failures still need the existing operator recovery procedure.

An archived join, including a reconnect over a board already on screen, shows a neutral English/Persian unavailable-game message and a Home action to start a new game. It clears pending requests and forgets the unavailable Home entry. Temporary restoration storage failures keep that entry and allow a later retry. Recovering an old record requires the operator to retain a backup and use a compatible old binary against a separate copy of its data. Do not point an old binary at newer-schema production data or reset an archive marker merely to force it open.

## Production run

The server starts as:

```
GOMEMLIMIT=96MiB .self-host/bin/ajayeb --addr 127.0.0.1:<port> \
  --data .self-host/data --origin <url> --trust-proxy 127.0.0.1
```

`GOMEMLIMIT` sets Go's soft memory target to 96 MiB; it is not a hard
limit on the process's resident memory.
`--trust-proxy 127.0.0.1` tells the server to believe `X-Forwarded-For` only
from Caddy on the same host; the per-address limits of the socket and of the
replay download (`GET /api/rooms/{code}/log`) count the real client this way
instead of counting every player as Caddy. `--origin` restricts which WebSocket origins the
server accepts.

Caddy's managed block, between `# BEGIN ajayeb self_host.py` and `# END
ajayeb self_host.py` in the Caddyfile, is a redirect from the opposite scheme
plus a single `reverse_proxy` to the server port with `encode zstd gzip`.
There is no `try_files` or static file serving in Caddy: the Go binary
embeds and serves the SPA itself.

### Security defaults

The server needs no security settings from Caddy:

- It sends its own security headers and a content security policy that
  allows same-origin scripts plus the hash of each inline script in the
  served HTML. In production the policy is enforced. Under `--dev` it is
  report-only, so a Vite page on another port is not blocked.
- The identity secret, `.self-host/data/secret.key`, must be a regular file
  of exactly 32 bytes. If its group or other users may read or write it, the
  server sets its mode to 0600 and logs a warning, and it refuses to start
  if it cannot.
- Bot searches are capped at 2 at once across the server (`--bot-searches`)
  and at 2 per client address (`--bot-searches-per-address`), counted by the
  same address as the other per-address limits, so behind `--trust-proxy`
  each player's own address.

[protocol.md](protocol.md#limits) has every limit.

### Caching and offline

The service worker needs no reverse-proxy configuration either. The server
marks the HTML, `/service-worker.js`, the web manifest, the favicon and the
home-screen icons `no-cache`, and marks content-hashed assets and versioned
art immutable. A layer in front that rewrites `Cache-Control` (some CDNs
give scripts a fixed lifetime) would delay updates, so check the headers
through it with `curl -sI <url>/service-worker.js`. The cached app shell
opens offline; playing still needs the server.

### Serving both schemes

With `--both-schemes` the managed block has no redirect: one site block
answers on both `https://<host>` and `http://<host>` and proxies both to the
server. Use it when a network between the players and the host blocks or
stalls one of the two ports, or to test a deploy over port 80. The server
needs nothing more: its WebSocket origin check accepts the request's own
host whatever the scheme, and the client picks `ws:` or `wss:` from the page.

What it costs:

- Over plain HTTP everything crosses the network unencrypted. The identity
  token a device sends in `hello` is that player's whole identity on that
  scheme, not a key to one room: whoever reads it can act as that player in
  every room the identity has joined or will join, until the device's storage
  is cleared. A device link's claim, sent in `join`, gives that one seat until
  the link is reset. Someone who can change traffic on the path can also
  rewrite the page itself.
- A browser keeps `http://` and `https://` storage apart, so a device that
  has played over one scheme is a new player over the other: its hot-seat
  and bot games do not open there, and an online room sees a different
  player. Invite and device links carry the scheme of the page that made
  them, so an invite made over `http://` sends guests to `http://`; to move a
  seat to the other scheme, change `http` or `https` in its device link
  before opening it.
- Features that need a secure context are missing over HTTP. The Share
  button does not appear, and copying a link to the clipboard can fail: the
  device link sheet then selects the link for copying by hand, and the
  lobby's Copy invite shows a warning.
- The redirect block answers with a permanent redirect (301), which browsers
  cache. A browser that was redirected before the switch keeps going to
  `https://` on its own until its cache is cleared; a private window does
  not have that cache.

`--no-both-schemes` puts the redirect back.

## Development run

`dev-start` writes a different Caddy block, one that routes `/api`,
`/api/*`, `/ws` and `/health` to the Go server and everything else to the
Vite dev server. The backend path matcher is handled before the catch-all
frontend handler, so API and WebSocket traffic never falls through to Vite.
The Vite dev server is started with `AJAYEB_SERVER` pointed at the Go
server's address, which its proxy configuration reads.

Never start these dev servers in a worktree, and never leave them running
after a session. They use ports from the 38951 to 38959 range reserved for
Ajayeb's own local services.

## Caddy safety

Because the Caddyfile is shared with other sites on the same machine, every
edit goes through a safe path. An unchanged managed block never needs a
backup, write or load:

1. If the Caddyfile already contains exactly the managed block this command
   would write, stop here: no backup, no write, no load. This is what makes
   `start` safe to run at boot, before Caddy is necessarily up yet; see
   "Starting at boot" below.
2. If the Caddyfile does not exist, refuse to create it when a new file could
   cost a running Caddy its other sites. A new file would hold the managed
   block alone, and loading it replaces everything that Caddy serves. So a
   missing file that `--caddyfile` named is taken as a wrong path, and a
   missing `~/Caddyfile` is created only when no Caddy on its default admin
   address (`localhost:2019`) is already serving a config, which it would be
   reading from some other file. This check happens before any Caddyfile
   edit. Production commands record its failure and still start the game
   server without editing or loading the file; `dev-start` stops on failure.
3. Copy the Caddyfile to `.self-host/Caddyfile.bak.<timestamp>` before
   writing anything.
4. Write the managed block in place (or append it, if this is the first
   time), already in Caddy's own tab-indented format.
5. Hand the Caddyfile to the running Caddy. `caddy adapt` finds the admin
   address the file's global options name (Caddy's default when they name
   none); when the file does not adapt in the script's environment (an
   unset `{$VAR}` can make an option invalid there), the address is read
   from the global options block as text instead. Everything else goes
   through that Caddy's admin API, so Caddy adapts the file in its own
   process, and a `{$VAR}` placeholder in the shared file resolves from
   Caddy's environment, as it did when Caddy started, whatever the shell
   running the script exports (an ssh session often exports less than the
   shell that started Caddy):
   - **The check.** Caddy adapts the file as it was before the edit
     (`/adapt`, which loads nothing), and the result must equal the config
     Caddy is running. If it does not, the file is not what this Caddy
     serves: the path is wrong, another Caddy owns that admin address, the
     file imports relative paths (Caddy resolves those against its own
     working directory when it adapts text, not against the file's), or
     the file holds edits Caddy has not loaded yet. Hide lists are compared
     as sets, since Caddy orders the config files in them differently from
     one version to the next. Restore the backup and stop the Caddy step without loading
     anything, with a message naming those causes.
     For a Caddyfile this command is creating, the check is that the Caddy
     serves no config at all.
   - **The hidden config file.** Caddy's `file_server` hides the Caddyfile
     it was started from, by that file's path. Text adapted with no file
     name gets a relative stand-in (`./Caddyfile`) instead, so the script
     reads the real path from the running config and puts it back, both
     for the check and in what it loads. Loading the text as it comes would
     otherwise weaken that protection for every site on the shared Caddy.
   - **The load.** Caddy adapts the edited file the same way, the hidden
     path goes back in, and the result goes to `/load` as JSON. Caddy
     switches to it only if all of it loads. If Caddy refuses it, restore
     the backup and stop the Caddy step with Caddy's error; Caddy keeps serving the old
     config. Caddy restarts its admin listener after a load, so if the
     reply is lost, the script waits a moment and compares the running
     config with what it sent: a match means the load took, and anything
     else (a different config, or no answer to that question) restores the
     backup and stops the Caddy step, saying the outcome is unknown.
6. If nothing listens on that address (the connection is refused, or a name
   such as `localhost` resolves to an address this host cannot use), Caddy
   is not running: run `caddy
   validate --config` on the Caddyfile instead, restore the backup and stop
   the Caddy step if it fails, and otherwise keep the validated edit and print a warning.
   Caddy reads the Caddyfile itself the next time it starts. Production
   commands still start the game server, but exit nonzero because Caddy
   has not published it. `caddy
   validate` reads `{$VAR}` placeholders from the script's own environment,
   so it can fail on a file that Caddy itself would load; the error says so
   when the file has placeholders.
7. If the Caddyfile turns the admin API off, puts it on a Unix socket or
   gives its address as a `{$VAR}` placeholder, fall back to `caddy validate` and then `caddy reload --config`, where a
   failed reload keeps the valid edit and prints a warning. Production
   commands still start the game server and exit nonzero on this failure.

Any other error after the write, a Ctrl-C included, also restores the
backup before the error leaves the Caddy step. Production commands catch
ordinary errors there and still attempt game startup; a Ctrl-C interrupts
the command. Left behind unloaded, the block would
make the next run believe there was nothing to do.

`setup`, `redeploy` and `start` also run the check of step 5 as a
preflight, before they stop the game server, whenever the block will need
writing and a Caddy answers. A Caddy that does not serve the file as it
stands is then reported before the game is stopped. The production
command proceeds with the game restart, skipping the Caddy edit and load.
The preflight loads nothing, and a block that already matches skips it.

Every `caddy` command gets `--adapter caddyfile`, so a Caddyfile under any
file name is read as one (Caddy only guesses the adapter from names that
start with `Caddyfile`).

`self_host.py` never runs `caddy fmt`: the Caddyfile is shared with other
sites, and reformatting the whole file on every deploy is not ours to do. The
managed block is already written in Caddy's own tab-indented format, so a
load never has anything of ours to normalize.

Production commands treat Caddy configuration and game startup separately.
A Caddy refusal, missing or invalid Caddyfile, missing Caddy executable,
failed reload or stopped Caddy never prevents the server start attempt.
The existing backup, refusal and check-then-load rules above still apply.
After startup, the script polls `http://127.0.0.1:<port>/health` for up to
10 seconds. Only a successful health check permits a message saying the
game server is running. If Caddy failed, the command reports its reason,
says the server is running on its loopback port but is not published at the
configured URL, and exits nonzero. This conservative status also covers an
unknown load outcome; the previous Caddy config may still serve a site.
A failed server start or health check exits nonzero without claiming that
the server runs or that the site is serving.

For an unchanged managed block, production commands also check the running
config through Caddy's TCP admin API when its address is available. The
file is still never written or reloaded. If the file disables that API or
uses a Unix socket, the unchanged path remains untouched, exits zero after
successful game health, and reports publication as unverified. It cannot
detect a stopped Caddy on that path. Even a successful admin check confirms
the config, not public DNS, TLS or reachability.

## Starting at boot

Run `python3 self_host.py start` from the deploy directory in the host's own
boot script, after it brings up Caddy (for example in tmux, with `caddy run
--config` on the Caddyfile). A `@reboot` crontab line works just as well:

```
@reboot cd ~/ajayeb && python3 self_host.py start
```

Cron runs with a minimal `PATH`, so when `caddy` or `tmux` live elsewhere
(`~/bin`, Homebrew on Linux), set `PATH` on the line itself, and send the
output to a log so a failed boot can be read afterward.

`start` is idempotent and safe to run unattended this way:

- An already-correct Caddyfile is left completely untouched (see "Caddy
  safety" above), so a reboot never rewrites or reformats the shared file.
- The server still starts even when Caddy is not up yet: if the block
  does need writing (say the stored port was taken and `start` chose
  another), it leaves a valid edit ready for Caddy's next start. If
  `caddy validate` fails instead, `start` restores the previous file and
  still starts the game server. Both cases exit nonzero and report that
  the game runs on its port but is not published. A Caddyfile with `{$VAR}`
  placeholders can fail validation unless the boot environment has those
  variables, so use the same environment that starts Caddy. An unchanged
  block uses only the read-only publication check described above.
- `start` only builds when `.self-host/bin/ajayeb` does not exist yet, so a
  normal reboot restarts the existing binary without rebuilding anything.

## Redeploy backups

`redeploy` archives `.self-host/data/rooms` to
`.self-host/backups/rooms-<timestamp>.tar.gz` once the server has stopped,
right before installing the new binary, so the archive never captures a
file mid-write. It keeps at most 10 backups and removes backups older than
30 days, by filesystem modification time, oldest first. Copying or touching
a backup can change that age. If `rooms` is missing or empty, it skips
creating a backup but still prunes existing copies. Unrelated files and
directories are ignored. These snapshots contain live/finished `rooms/`
records only; they do not duplicate `rooms-archive/` on every deployment.

The replay download reads the same room files, so a finished game's log can
be downloaded, unless `hideAtEnd` is set, until the reaper deletes the
room. Finished rooms become eligible after 7 days without an update;
unfinished rooms expire after 14 days. The menu shows the download
as disabled during play and for a hidden record; eligible finished games
also offer it on the result screen. See [the download contract](protocol.md#downloading-a-games-log).

Archived records live separately at `.self-host/data/rooms-archive/<code>.json`,
with the archive reason and original archive time for bug reports. New
archives move there after the closed status is durably saved. Legacy archives
still under `rooms/` migrate on load or the next sweep without resetting their
clock. The directories must be on the same filesystem; a move failure is
reported and retried without overwriting another diagnostic record. Code
lookups still report an archived room, and its code stays reserved while the
record is retained.

The reaper runs at startup and hourly. It removes archives more than 30 days
after `archivedAt` (`updated` for legacy archives), then evicts the oldest
until at most 1000 files and 64 MiB remain. Limits can be exceeded between
sweeps, and a burst can shorten the 30-day window. Malformed, unreadable,
unknown-schema or misplaced nonarchived records are left for operator repair
outside those managed limits; preflight still refuses them. Copy a needed
record outside these folders before expiry if an investigation needs longer.
See [D29](dev/decisions.md#d29-keep-bounded-diagnostic-archives-outside-the-live-room-directory).

## Deploying from another machine

The production host is usually small and shared: one vCPU, a little RAM, a
nearly full disk, an older Go than `server/go.mod` needs, and other services
running on it. Building there is slow and risks the disk and those other
services, so a real deploy builds on the dev machine instead and ships one
static binary.

Run `./self_host.py ship TARGET [url] [--remote-dir DIR]` on the dev
machine, where `TARGET` is an SSH destination such as `user@example.host`
and `DIR` is a directory relative to the remote home (default `ajayeb`).
`ship`:

1. Runs `ssh TARGET uname -sm` and maps the result to a Go `GOOS`/`GOARCH`
   pair (Linux x86_64 to `linux/amd64`, Linux aarch64 to `linux/arm64`;
   anything else is refused).
2. Builds the production web bundle and syncs it into
   `server/internal/webdist/dist/`, the same as `setup`/`redeploy`, then
   cross-compiles with `CGO_ENABLED=0`, the detected `GOOS`/`GOARCH`,
   `-tags embedweb -trimpath` and the version ldflags, to
   `.self-host/dist/ajayeb-<goos>-<goarch>`. When this checkout has
   `tools/heavy.py`, these build steps run under it (`--kind build`); its
   refusal (exit 75, meaning memory pressure or a busy slot) is reported
   clearly rather than as a generic failure.
3. Uploads `self_host.py` and the binary to the remote `DIR` and
   `DIR/.self-host/bin/ajayeb.upload` with `rsync`.
4. Runs `setup --prebuilt .self-host/bin/ajayeb.upload [url]` over SSH on
   the remote side, or `redeploy` instead of `setup` when the remote
   already has a `.self-host/config.json`. `--both-schemes`,
   `--no-both-schemes` and `--caddyfile PATH` are passed on when given, and
   the remote host records them in its own config, so a later `ship` can
   leave them out. `--prebuilt` installs by moving
   the uploaded binary into `.self-host/bin/ajayeb` rather than copying it,
   so the upload does not also linger on disk as a second ~8.5 MB copy once
   it is installed (falling back to copying and then removing it only when
   the two paths are on different filesystems).
5. Polls `<url>/health` from the dev machine for up to 30 seconds and
   prints the version it reports, with a warning if it differs from the one
   just built, or if it never answers.

`ship` remembers the last `TARGET` and `DIR` it used in the local, gitignored
`.self-host/config.json`, so a later `ship` can omit them and reuse what was
remembered. A later `ship url` (just the url, with no target) reuses the
remembered target too; it is treated as the url rather than the target
whenever the first argument starts with `http://` or `https://`, and it is
an error if no target has been remembered yet. Every argument that reaches a
remote shell (the directory, the url, the Caddyfile path) is shell-quoted. `ship` refuses a
`TARGET` or `DIR` that starts with `-`, and a `DIR` with characters outside
letters, digits and `. _ - /`, because rsync would otherwise pass it to the
remote differently from ssh.

`--prebuilt PATH` can also be used directly, without `ship`, by anyone who
builds the binary some other way: `setup`/`redeploy --prebuilt PATH` check
that `PATH` is a regular, executable file and that `PATH --version` prints a
version, before anything is stopped. The candidate must also pass the room compatibility
preflight above; a binary without `--check-rooms` support refuses safely.
`PATH` is then consumed: it is moved
into place (or copied and removed, across filesystems), so pass a path you
do not need to keep afterward.

## Deploying to several hosts

Run `./self_host.py ship-all` to build one release and deploy it sequentially
on every host in a private targets file. The default path is
`.self-host/deploy-targets.json`, inside the gitignored `.self-host/`
directory. Set `AJAYEB_DEPLOY_TARGETS` to use another file, keeping it outside
git too. For example:

```json
{
  "targets": [
    {"target": "deploy@one.example.host", "url": "https://one.example.host"},
    {"target": "deploy@two.example.host", "url": "http://two.example.host:8080", "remote_dir": "apps/ajayeb"}
  ]
}
```

Each entry requires `target` (an SSH alias or `user@host`) and `url` (an
HTTP or HTTPS origin with a DNS name or IPv4 host, without credentials).
`remote_dir` is optional and defaults to `ajayeb`; it has the same character
restrictions as `ship --remote-dir`. Duplicate target/directory pairs and
unknown fields are refused. Missing or malformed config gives its expected
path and an example format before any subprocess, build or SSH command runs.
Store authentication in SSH configuration, never in the targets file.

`ship-all` checks every host's architecture before building. Every target
must use the same supported platform, either Linux x86_64 or Linux aarch64.
If a host cannot be checked, or the targets mix architectures, the whole
batch stops before building or uploading and reports every target. This
keeps the command's build-once guarantee and prevents an incompatible binary
from reaching a host. Use separate target files for different architectures.

After preflight, it reuses `ship`'s build, upload and setup/redeploy steps.
Each remote keeps its own remembered ports, `both_schemes` and Caddyfile path.
The one batch safety override, `--force-room-archive`, explicitly accepts
live-room archiving on every target for that invocation only. The local remembered single-target
`ship` destination is left alone. A new host uses the normal setup defaults;
configure host-specific options with single-target `ship` first if needed.

For each target it checks `<url>/health` twice, even if its deployment or
first health check failed:

- On the host, SSH runs curl with `--resolve HOST:PORT:127.0.0.1` and proxies
  disabled, preserving the origin's Host header and TLS name. Scheme and port
  come from the configured URL, defaulting to 443 for HTTPS or 80 for HTTP.
  SSH is noninteractive; both SSH and curl have time limits. Curl must be
  installed on each host.
- On the dev machine, a separate batch poll checks the public URL using
  local curl, which must also be installed. Each attempt has a total curl
  timeout of at most five seconds. An independent pipe reader enforces the
  same absolute deadline across DNS, connection, headers and body transfer,
  even if the endpoint trickles bytes indefinitely. It accepts at most
  16 KiB per response, regardless of the reported Content-Length, and kills
  and reaps curl after timeout or early rejection. Curl configuration files
  are ignored; proxy environment variables retain their normal effect.

Each probe gets a total 30-second polling window, including attempts and
retry delays. Success requires the health endpoint to report the version just
built. The per-target report includes deployment,
host-local health and public health results. A failed deployment, unavailable
health endpoint or version mismatch makes the overall exit status nonzero,
while later targets still deploy and get their own reports. Failures are
reported by stage and exception type without echoing potentially sensitive
exception messages. Public timeouts report `TimeoutError`, malformed or
wrong-type health JSON reports `InvalidHealthResponse`, and oversized bodies
report `HealthResponseTooLarge`. A stale version reports a version mismatch.
The single-target `ship` health poll is unchanged. There is no automatic
batch rollback; inspect the report before retrying failed hosts.

## Testing

`tests/test_self_host.py` covers URL parsing, the rendered Caddy blocks for
both modes, port allocation and reselection, the config file round trip
(including the remembered ship target), the node and pnpm commands, the Go
build command, the Caddy paths against a local stand-in for Caddy's admin
API (a stateful fake that serves a config, adapts and loads: the edit
loaded as JSON with the config file hidden by the running config's path,
past any proxy; hide lists in any order; refusing a Caddy that does not
serve the file as it stood, or that hides two different paths, and the
same check as a preflight before setup, redeploy and start stop the game;
the backup restored when Caddy refuses
the edit or cannot adapt it; a lost reply accepted only when the running
config shows the load took, and an unknown outcome when the recheck gets no
answer; a new Caddyfile loaded only into a Caddy with no config; `caddy
validate` only when nothing listens (a refused connection, or `localhost`
resolving to an unusable address), with a
hint about `{$VAR}` placeholders when it
fails, the validate-then-reload fallback, and the backup restored on any
other error after the write, a Ctrl-C included), the admin address read from the adapted
Caddyfile or, when it does not adapt, from its global options as text,
`--adapter caddyfile` on every `caddy` command,
never creating a named Caddyfile that does not exist (refused by every
Caddy edit, with production still starting the game) or a `~/Caddyfile`
over a Caddy that serves a config, skipping an already-matching Caddyfile
entirely (no backup, no write, no load), continuing game startup with a
nonzero unpublished result after Caddy failures, preserving valid pending
edits after validate-only or reload failures, verifying unchanged configs
through an observable TCP admin API and reporting unobservable configs as
unverified, refusing a running-server claim after startup or health failure,
that `reload_caddy` never runs `caddy fmt`, the
`--caddyfile` path (resolved from home, remembered, forwarded quoted by
`ship`, and used by every Caddy step), the health poll, the
rooms backup pruning, the `uname` to `GOOS`/`GOARCH` mapping and its refusal
of unsupported platforms, the cross-compile command and environment, the
quoting of every remote command, the `setup` versus `redeploy` choice,
`--prebuilt` skipping every build step and being checked before anything
stops, the `.prev` backup, installing a `--prebuilt` binary by moving it
(falling back to copy-then-remove across filesystems), `ship` reading a
url-looking first argument as the url instead of the target, and the remote
health poll and version comparison. It uses temporary files and fakes for
Caddy, tmux, ssh, rsync, go, pnpm and network calls throughout, so running it
never touches the real `~/Caddyfile`, starts a real tmux session, or reaches
the network. Run it with:

```
make test-py
```

## Production smoke

After `setup`, `redeploy` or `ship` brings up or updates the real site, run
the Playwright production smoke from the repository root:

```
make e2e-prod
```

This is the M8 acceptance check ("the Playwright `@smoke` run (create,
join, five moves) passes against production"), played through the real UI
of the deployed site itself: `GET /health`, Home in both themes, a short
online game between two isolated browser contexts (at least five moves,
through the confirm step), and a short game against the bot. It builds
nothing and starts nothing local; it only drives a browser against
`PROD_URL` (default `https://ajayeb.lilf.ir`, the site `self_host.py`
targets by default too):

```
make e2e-prod PROD_URL=https://staging.example
```

It never touches `/api/dev` or a `/dev` route (production never serves
them; see "Development endpoints" in [protocol.md](protocol.md)), and the
few rooms it creates (at most two per run) are left for the server's own
reaper, the same as any other player's abandoned room. See
[testing.md](testing.md), "Production smoke", for what each test checks and
how `e2e/playwright.config.ts` keeps every other spec off a real site even
when `E2E_BASE_URL` is set by hand instead of through this target.

A failure here after a `redeploy` is the signal to roll back: stop the
server, restore `.self-host/bin/ajayeb.prev` to `.self-host/bin/ajayeb`, and
`start` it again.

A rollback to a pre-D28 binary still checks stored hashes strictly: rooms whose hashes the new binary refreshed or saved may archive on load. Preserve a copy of current data and pair that old binary with a matching pre-upgrade rooms backup when recovering; restoring that backup also gives up moves made afterward.
