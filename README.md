# Ajayeb

Ajayeb (Persian عجایب, "wonders") is a browser version of a two-player card-drafting civilization game. It is an unofficial fan implementation of the rules of 7 Wonders Duel, with original art and an original interface, built mobile-first so a full game is comfortable on a phone as well as on a desktop.

If you like it, please buy the real game from its publisher: <https://www.rprod.com/en/games/7-wonders-duel>.

All three game modes are implemented, with original art, a guided tutorial, in-game rules, hints and replay downloads. The remaining review work is tracked in [docs/dev/roadmap.md](docs/dev/roadmap.md). The public instance is <https://ajayeb.lilf.ir>.

## Modes

Every mode is a room on the server. The browser shows the room and sends moves; it never runs the rules itself.

- **Versus the computer.** Play against a bot at one of three levels (Easy, Medium, Hard). The stronger levels use information-set Monte Carlo tree search, and every bot sees only what a human in its seat would see. You can undo your last move.
- **Hot-seat.** Two people share one device and pass it between turns. A short pass screen tells the next player what just happened. Undo works until the other player acts.
- **Online.** Create a room, send the invite link (`/play/<code>`) to a friend, and play from two devices. Reconnects and server restarts restore the persisted game. Finished rooms expire after 7 days and unfinished rooms after 14 days without an update.

## Stack

- **Server:** Go 1.26, one small binary. It holds the only rules engine, the bots, the room actors and the WebSocket endpoint (`github.com/coder/websocket`). Rooms are saved as JSON files (seed plus action log). In production the same binary serves the built web app.
- **Web:** SvelteKit 2 with Svelte 5 runes, Tailwind CSS 4, Vite 8 and TypeScript 6.0, built as a static single-page app (adapter-static). Wire types are generated from the Go structs with `tygo`.
- **Tests:** Go unit, property, fuzz, golden-replay and protocol tests; Vitest for client logic; Playwright full playthroughs in headless Chrome.
- **Tooling:** a pnpm 10 workspace (`web`, `e2e`) on Node 24, a root `Makefile`, and Python 3 standard-library scripts (`tools/heavy.py`, `self_host.py`).

## Quick start

You need Go 1.26, Node 24 (see `.nvmrc`), pnpm 10, Python 3 and GNU Make. The end-to-end tests also need Google Chrome.

```sh
make install       # download the Go modules, then install the web and e2e dependencies
make verify-quick  # fast checks: gofmt, Go vet and tests, web lint, svelte-check, unit tests, wire-type check, icon lint
make e2e           # Playwright playthroughs against a test build, in headless Chrome
make dev           # explains how to start the dev servers (python3 self_host.py dev-start)
```

`make dev` starts nothing itself. `python3 self_host.py dev-start` runs the Go server on 127.0.0.1:38951 and the Vite dev server on port 38952 in tmux; then open <http://localhost:38952>.

Every target that compiles, tests, builds, installs or starts a browser runs through `tools/heavy.py`, a guard for memory, shared job slots and the current run mode. If it refuses a step, it exits with code 75 and prints the reason; `make` reports `Error 75` and exits 2. [docs/dev/ram-policy.md](docs/dev/ram-policy.md) explains the guard and its modes.

To self-host on a machine with Caddy and tmux, use `./self_host.py setup <url>`; to build on a dev machine and deploy a prebuilt binary to a small production host, use `./self_host.py ship <target> <url>` or `./self_host.py ship-all` for all hosts in a private targets file (see [self-hosting](docs/self-hosting.md)).

## Repository layout

```
ajayeb/
  README.md  LICENSE  NOTICE.md  AGENTS.md  CLAUDE.md  Makefile  self_host.py
  package.json  pnpm-workspace.yaml  .nvmrc  .editorconfig  .gitignore
  server/     Go module: cmd/{ajayeb, ajayeb-sim, ajayeb-drive, ajayeb-gen}
              internal/{engine, view, ai, room, store, ident, wire, ws, httpapi, webdist}
  web/        SvelteKit single-page app (the thin client)
  e2e/        Playwright: specs/ (playthroughs and UI checks), fixtures/ (drivers and probes)
  tools/      heavy.py (memory and run-mode guard), chrome/ (headless Chrome tools), art/ (art scripts)
  art/        art direction, asset manifest, style references, approved originals, vector specs and logs
  docs/       architecture, protocol, testing, self-hosting, legal, rules/, art/, dev/
  tests/      Python unittest suites for self_host.py and heavy.py
```

The server commands:

- `ajayeb` is the game server.
- `ajayeb-sim` runs bot tournaments, the AI benchmark, level calibration and the replay verifier.
- `ajayeb-drive` plays full games over WebSocket against a running server, for smoke tests.
- `ajayeb-gen` writes the display catalog (including typed id lists) and contract fixtures. `make gen` runs it after `tygo` generates the wire types. Its `catalog -format csv|json|prose|ts` subcommand also exports catalog data for review.

## Documentation

- [docs/README.md](docs/README.md): the index and a suggested reading order.
- [docs/architecture.md](docs/architecture.md): how the server, the engine and the thin client fit together, how a move flows, and how hidden information stays hidden.
- [docs/legal.md](docs/legal.md): licenses, the trademark notice and what we never copy.
- [docs/dev/roadmap.md](docs/dev/roadmap.md): milestones and their acceptance criteria.
- [docs/dev/decisions.md](docs/dev/decisions.md): the main design decisions, with reasons and rejected options.
- [docs/dev/orchestration.md](docs/dev/orchestration.md) and [docs/dev/ram-policy.md](docs/dev/ram-policy.md): how the long-running, agent-driven development is organized.
- [AGENTS.md](AGENTS.md): the guide for contributors and coding agents.

## License

Code is MIT ([LICENSE](LICENSE)). Original art is CC BY 4.0. See [NOTICE.md](NOTICE.md) and [docs/legal.md](docs/legal.md).

---

Ajayeb is an unofficial, non-commercial fan implementation of the rules of 7 Wonders Duel by Antoine Bauza and Bruno Cathala (Repos Production). It is not affiliated with or endorsed by Repos Production or Asmodee; 7 Wonders Duel is their trademark. [Buy the real game](https://www.rprod.com/en/games/7-wonders-duel).
