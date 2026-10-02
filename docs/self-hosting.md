# Self-hosting

`self_host.py` runs Ajayeb behind Caddy on a single machine, with tmux
keeping every long-running process easy to inspect or restart without
Docker. It is adapted from the snow-white project's `self_host.py`.

## Commands

Run `./self_host.py <command> [url]` from the repository root. The url
argument is optional; it defaults to the site recorded in
`.self-host/config.json`, or `https://ajayeb.pinky.lilf.ir` if nothing is
recorded yet.

- `setup [url]`: first-time install. Installs web dependencies, builds the
  production web bundle, builds the Go server with the bundle embedded,
  writes the Caddy block, reloads Caddy and starts the server.
- `redeploy [url]`: rebuilds and restarts for a new version. Archives the
  current `data/rooms` directory first, then does the same steps as `setup`
  without the dependency dedupe pass.
- `start [url]`: starts the already-built server. Builds it first only if
  `.self-host/bin/ajayeb` does not exist yet.
- `stop`: stops every tmux session this script manages. Leaves the Caddyfile
  and the built binary alone.
- `dev-start [url]`: starts a development setup instead of production. Runs
  the Go server under `go tool air` for live reload, and the Vite dev server
  for the web client, with the Caddyfile pointed at both.
- `status [url]`: prints the recorded configuration, whether each tmux
  session is running (with its last 20 lines of output), whether its port is
  free, and the result of a server health check.

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
`.self-host/config.json` so a later command reuses them. `go tool air`'s
entrypoint is fixed by `server/.air.toml` to the server's default address, so
in dev mode the Go server itself always binds 38951; only the Vite dev
server's port is reselected when busy.

## Configuration file

`.self-host/config.json` records the site URL, the mode (`prod` or `dev`)
and the chosen ports. It is not committed; `.self-host/` is gitignored
entirely, since it holds local build output, the production data directory
and Caddyfile backups.

## Build

`setup`, `redeploy`, and `start` (when no binary exists yet) build the web
client and the Go server:

1. `pnpm install` (offline and frozen, with a dependency dedupe pass on
   `setup` only) and `pnpm -C web build`, both run under `zsh -lc` with
   `nvm-load` and `nvm use 24`.
2. `rsync -a --delete` the built `web/build/` into
   `server/internal/webdist/dist/`.
3. `go build -tags embedweb -trimpath -ldflags "-s -w -X main.version=<version>"`
   to `.self-host/bin/ajayeb.new`, then an atomic rename to
   `.self-host/bin/ajayeb`. The version comes from `git describe --always
   --dirty`. Go is found with `shutil.which('go')`, falling back to
   `/usr/local/go/bin/go` or the Homebrew path.

## Production run

The server starts as:

```
GOMEMLIMIT=96MiB .self-host/bin/ajayeb --addr 127.0.0.1:<port> \
  --data .self-host/data --origin <url> --trust-proxy 127.0.0.1
```

`GOMEMLIMIT` keeps the process well inside its usual 10 to 20 MB RSS.
`--trust-proxy 127.0.0.1` tells the server to believe `X-Forwarded-For` only
from Caddy on the same host. `--origin` restricts which WebSocket origins the
server accepts.

Caddy's managed block, between `# BEGIN ajayeb self_host.py` and `# END
ajayeb self_host.py` in `~/Caddyfile`, is a redirect from the opposite scheme
plus a single `reverse_proxy` to the server port with `encode zstd gzip`.
There is no `try_files` or static file serving in Caddy: the Go binary
embeds and serves the SPA itself.

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

Because `~/Caddyfile` is shared with other sites on the same machine, every
edit goes through a safe path:

1. Copy `~/Caddyfile` to `.self-host/Caddyfile.bak.<timestamp>` before
   writing anything.
2. Write the managed block in place (or append it, if this is the first
   time).
3. Run `caddy validate --config ~/Caddyfile`. If validation fails, restore
   the backup immediately and stop with an error, without reloading.
4. Only if validation passes, run `caddy fmt --overwrite` and `caddy reload`.

After starting the server, the script polls `http://127.0.0.1:<port>/health`
for up to 10 seconds and prints a warning if it never answers, rather than
failing silently.

## Redeploy backups

`redeploy` archives `.self-host/data/rooms` to
`.self-host/backups/rooms-<timestamp>.tar.gz` before rebuilding, and keeps
only the 10 most recent archives. If there is no `rooms` directory yet (a
fresh install), it skips the archive step.

## Testing

`tests/test_self_host.py` covers URL parsing, the rendered Caddy blocks for
both modes, port allocation and reselection, the config file round trip, the
node and pnpm commands, the Go build command, the Caddy validate-then-reload
and restore-on-failure paths, the health poll, and the rooms backup pruning.
It uses temporary files and fake Caddy, tmux and network calls throughout,
so running it never touches the real `~/Caddyfile`, starts a real tmux
session, or reaches the network. Run it with:

```
python3 -m unittest discover -s tests
```
