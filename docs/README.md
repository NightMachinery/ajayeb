# Ajayeb documentation

These docs explain the current implementation, its rules and the tools used to build and run it. The roadmap separates completed milestones from remaining work.

## Suggested reading order

1. [architecture.md](architecture.md): the big picture. The Go server owns the rules and the truth; the browser renders what it is sent. Read this first.
2. [rules/](rules/README.md): the rules of the game in our own words, the rulings on ambiguous points ([rules/decisions.md](rules/decisions.md)), the provenance of the card data ([rules/sources.md](rules/sources.md)) and the data rulings ([rules/data-rulings.md](rules/data-rulings.md)).
3. [protocol.md](protocol.md): the game log and scenario formats, the per-seat view and its event redaction, and the WebSocket message contract.
4. [ai.md](ai.md): the bots, their difficulty presets, measured strength and cost per move, and how to rerun the tournaments.
5. [testing.md](testing.md): the test layers, from engine unit tests to full games played over a real WebSocket against a real server binary.
6. [dev/decisions.md](dev/decisions.md): why things are the way they are, and what was rejected.

## Rules reference

- [rules/setup.md](rules/setup.md): components, setup, the wonder draft and card structures.
- [rules/turn.md](rules/turn.md): taking cards, the three actions and extra turns.
- [rules/economy.md](rules/economy.md): production, trading, chains and payment.
- [rules/military-science.md](rules/military-science.md): the military track, science pairs and supremacy.
- [rules/wonders-tokens.md](rules/wonders-tokens.md): wonders, progress tokens and their choices.
- [rules/ages-scoring.md](rules/ages-scoring.md): age transitions, scoring and ties.

## The second game (design)

- [haftgane/](haftgane/README.md): the design of Haftgane, Ajayeb's second game, an unofficial version of 7 Wonders (second edition) for 3 to 7 players, from milestone M12: its rules in our own words and the platform, engine, bot, screen and art designs. The design is accepted: its decisions are D30 to D61 in [dev/decisions.md](dev/decisions.md) and its build plan is M13 to M23 in [dev/roadmap.md](dev/roadmap.md). Nothing in it is built yet.

## Package guides

- [../web/README.md](../web/README.md): the web client: its layout, design tokens and Tailwind utilities, i18n, fonts and the `PUBLIC_DEV_ROUTES` flag.
- [i18n/persian.md](i18n/persian.md): the Persian translation's glossary: the house terms, every card, wonder and token name with the reason for it, and the open questions for a native speaker's check.
- [self-hosting.md](self-hosting.md): running Ajayeb locally for development and in production with `self_host.py`.
- [../tools/chrome/README.md](../tools/chrome/README.md): the headless Chrome tools for screenshots, layout probes and SVG contact sheets.

## Project and legal

- [legal.md](legal.md): code and art licenses, the trademark notice, and what we never copy.
- [art/](art/README.md): the original art: the hand-drawn vector pipeline (icons, ornaments, logo) and the Codex bitmap pipeline ([art/bitmaps.md](art/bitmaps.md)), and the [CC0 sound pipeline](art/sound.md).

## Development process

- [dev/roadmap.md](dev/roadmap.md): milestones M0 to M23 with their acceptance criteria: Duel in M0 to M11, the design of Haftgane in M12 and its build in M13 to M23.
- [dev/orchestration.md](dev/orchestration.md): how the long-running, agent-driven work is organized: run modes, scheduled procedures, bounded slices with independent review.
- [dev/master-e2e-failures.md](dev/master-e2e-failures.md): bot-frame animation, modal restyling, localized age labels and reload connection-count attribution.
- [dev/protocol-fixtures.md](dev/protocol-fixtures.md): fixture coverage, compact storage and regeneration.
- [dev/scenarios.md](dev/scenarios.md): named engine positions and the development-only room endpoint used by tests.
- [dev/ram-policy.md](dev/ram-policy.md): the memory budget and the `tools/heavy.py` guard that every heavy job goes through.
- [dev/decisions.md](dev/decisions.md): a short record of each major decision, its rationale and the options we rejected.

Contributors and coding agents should also read [../AGENTS.md](../AGENTS.md), the working agreement for this repository.
