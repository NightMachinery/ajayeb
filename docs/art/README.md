# Art in Ajayeb

Ajayeb's art is original and comes from two independent pipelines. Both follow the "Assets" rules of [AGENTS.md](../../AGENTS.md) and the limits in [legal.md](../legal.md): nothing is copied or traced from the official game.

- **Bitmaps** (card, wonder and token illustrations, card backs, textures and the other pictures) are generated one at a time by Codex `image_gen`, reviewed, and shipped as AVIF. [bitmaps.md](bitmaps.md) describes that pipeline.
- **Vectors** (the icons, the ornaments, the military track parts and the logo) are hand-authored SVG, drawn by Opus agents in a render, look and critique loop. Codex is never used for them. The rest of this page describes that pipeline.

## The vector pipeline

### Where things are

- [art/vector/STYLE.md](../../art/vector/STYLE.md): the rules every vector follows (grid, minimum sizes, the two-tone paint convention, the halo, budgets, the visual language) and the iteration loop. Start there.
- `art/vector/specs/<set>.md`: one brief per set: per icon its id, its meaning in the game, the metaphor, the required silhouette, what it must not be confused with, its sizes and the places it appears.
- `art/vector/sets.json`: which files make up each set, and how the specimen shows them in context (their ink, detail tone, tile, chip, badge and band).
- `art/vector/log/<set>.md`: the iteration log of each set: what each round saw and changed, the blind-check scores, and which icons are marked `needs-human`.
- `web/src/lib/icons/svg/<id>.svg`: the icon sources, the single source of truth. `web/src/lib/icons/names.ts` lists their ids; `sources.ts` validates and imports them at build time. `Icon.svelte` renders the original shapes directly, with clipping ids scoped to each instance. This avoids the shared sprite and SVG use shadow trees during native modal restyling.
- `tools/art/vector/`: the tools, documented in [tools/art/README.md](../../tools/art/README.md#vector-pipeline-vector).

The five original sets are `resources-economy-military`, `science`, `chain`, `ui-category` and `ornaments-track-logo`. They were drawn in that order: the most frequent and most critical glyphs first. Two small sets followed in M10: `hints` (the hint pip glyphs) and `tt2-glyphs` (`play-again`, `build-from-discard` and `destroy-card`, which replaced the last Lucide outlines on cards; spec in `art/vector/specs/tt2-glyphs-diagrams.md`). `crest-moon-sm`, drawn with them, belongs to `ui-category` as the `-sm` of `crest-moon`.

Haftgane's own drawings (M19) live in `web/src/lib/games/haftgane/icons/svg/`, outside the directory that `sources.ts` imports, so they never ship to Duel players; `names.ts` and Duel's icon byte budget do not count them. A Duel drawing that Haftgane reuses is referenced from `sets.json` (an entry's `file` or `smFile`), never copied there. `make vec-lint` lints that directory by the same rules. `haftgane-resources-science` holds ore and cloth, the compass and the tablet with their `-sm` variants, and `science-gear-sm`, whose md is Duel's `chain-gear` under the science id.

Haftgane's chain symbols are the set `haftgane-chain` (M19): 24 symbols, of which 14 are Duel drawings referenced from `sets.json` (eleven chain drawings and three science drawings under chain aliases) and 10 are new files in `web/src/lib/games/haftgane/icons/svg/`, outside Duel's bundle. Its spec and log are `art/vector/specs/haftgane-chain.md` and `art/vector/log/haftgane-chain.md`.

The ornaments reached the app in M11 (slice `M11.ornament-integration`): the girih as a faint inlay behind the card structure, the track cells and the city gates on the military track's full bar, and the wordmark above Home's about footer and in the GameOver dialog. `web/README.md` (section Ornaments) says how each is drawn. In the same slice `crest-moon-sm` was redrawn as a plain tilted crescent, because at the 11 px of the compact track the crescent face read as the same round blob as `crest-sun-sm` without colour (log: `art/vector/log/tt2-glyphs.md`).

Haftgane's glyphs (M19) live in `web/src/lib/games/haftgane/icons/svg/`, outside Duel's icon directory, so none of them reaches a Duel player's bundle; Duel drawings they reuse are referenced, never copied. The set `haftgane-conflicts-passing-abilities` holds the conflict victory and defeat tokens, the pass mark and five wonder ability glyphs (spec in `art/vector/specs/haftgane-conflicts-passing-abilities.md`). Its blind check also shuffles in the Duel glyphs each new one must not be confused with (the set's `neighbours` in `sets.json`).

### How one set is drawn

An agent owns one set at a time and loops over it:

1. It reads the set's spec, STYLE.md and the set's log.
2. It draws or edits the SVGs.
3. `lint.mjs` checks what a program can: the viewBox, forbidden elements, the paint convention, raw colours, implicit black fills, snapping to the 0.5 grid, the 2-unit minimum feature, ids, and the byte budgets.
4. `fast.mjs` renders each changed icon at 12 to 96 px through librsvg in milliseconds, for the small steps.
5. Once per batch, `shoot.mjs` photographs the set's specimen page in a headless Chrome: light and dark at DPR 1 and 2, plus greyscale and the deuteranopia, protanopia and tritanopia emulations. The specimen shows every icon at 12 to 48 px on paper and on the night surface, on all seven card band colours, and in mock-ups of a 54x83 mini card, a strip chip, an affordability badge and a track cell. `peep.mjs` enlarges the 12 to 16 px strip pixel by pixel.
6. The agent looks at every PNG, writes its critique and changes in the log, and repeats. An icon gets at most six rounds; then the best candidate is kept and marked `needs-human`.
7. Before the set is done: a blind identification test (`blind.mjs`, shuffled and unlabelled sheets at 12 and 16 px, scored against a key kept outside the repository), a family check across all sets (`family.mjs`), and screenshots of `/dev/gallery` in the real app.

### Where outputs go

Only sources are committed: the SVGs, `STYLE.md`, the specs, `sets.json`, the logs and the tools. Every render, specimen, peep, blind sheet, blind key and family sheet goes to the private run directory, `~/tmp/agents/ajayeb/art-work/vector/` (or `$AJAYEB_RUN_DIR/art-work/vector/`):

- `<set>/` (or `<set>/iter-<n>/`): `specimen.html`, `lint.txt`, the light, dark, greyscale and colour-vision PNGs, `shots.json` and the peeps.
- `blind/<set>-<stamp>/`: the two sheets, `names.txt`, `answers.txt` and the key.
- `family/`: the family sheets.
- `fast/`: the quick renders.

### Browsers and memory

Every tool that takes a screenshot starts one headless Chrome of its own (through `tools/chrome/launch.mjs`), with a throwaway profile, and closes it before it exits. Run those tools inside the guard's browser lock, `python3 tools/heavy.py run --kind browser -- node tools/art/vector/<tool>.mjs ...`, so the machine never runs two Chromes at once. `lint.mjs`, `fast.mjs`, `specimen.mjs` and `peep.mjs` need no browser.
