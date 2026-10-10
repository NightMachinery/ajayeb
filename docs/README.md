# Ajayeb documentation

These docs explain how Ajayeb works and how it is built. Files marked "to come" are planned and arrive with the milestone named next to them.

## Suggested reading order

1. [architecture.md](architecture.md): the big picture. The Go server owns the rules and the truth; the browser renders what it is sent. Read this first.
2. [rules/](rules/README.md): the rules of the game in our own words, the rulings on ambiguous points ([rules/decisions.md](rules/decisions.md)), the provenance of the card data ([rules/sources.md](rules/sources.md)) and the data rulings ([rules/data-rulings.md](rules/data-rulings.md)).
3. [protocol.md](protocol.md): the game log and scenario formats, the per-seat view and its event redaction, and (as M4 lands) the WebSocket message contract.
4. [ai.md](ai.md): the bots, their difficulty presets, measured strength and cost per move, and how to rerun the tournaments.
5. [testing.md](testing.md): the test layers, from engine unit tests to full games played over a real WebSocket against a real server binary.
6. [dev/decisions.md](dev/decisions.md): why things are the way they are, and what was rejected.

## Package guides

- [../web/README.md](../web/README.md): the web client: its layout, design tokens and Tailwind utilities, i18n, fonts and the `PUBLIC_DEV_ROUTES` flag.
- [self-hosting.md](self-hosting.md): running Ajayeb locally for development and in production with `self_host.py`.
- [../tools/chrome/README.md](../tools/chrome/README.md): the headless Chrome tools for screenshots, layout probes and SVG contact sheets.

## Project and legal

- [legal.md](legal.md): code and art licenses, the trademark notice, and what we never copy.
- `self-hosting.md` (to come, M8): running your own instance with `self_host.py`, Caddy and tmux.
- [art/](art/README.md): the original art: the hand-drawn vector pipeline (icons, ornaments, logo) and the Codex bitmap pipeline ([art/bitmaps.md](art/bitmaps.md)).

## Development process

- [dev/roadmap.md](dev/roadmap.md): milestones M0 to M11 with their acceptance criteria.
- [dev/orchestration.md](dev/orchestration.md): how the long-running, agent-driven work is organized: run modes, scheduled procedures, bounded slices with independent review.
- [dev/ram-policy.md](dev/ram-policy.md): the memory budget and the `tools/heavy.py` guard that every heavy job goes through.
- [dev/decisions.md](dev/decisions.md): a short record of each major decision, its rationale and the options we rejected.

Contributors and coding agents should also read [../AGENTS.md](../AGENTS.md), the working agreement for this repository.
