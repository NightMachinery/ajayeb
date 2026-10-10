# tools/art: the art pipelines' scripts

Two independent pipelines produce Ajayeb's original art (see AGENTS.md
"Assets"): a hand-authored SVG icon set made by Opus subagents in a
render-screenshot-critique loop, and Codex-generated bitmap art for cards,
wonders, tokens and the rest. `render-fast.sh` (the shared inner-loop SVG
rasterizer, used by both the icon loop and the `chrome-cdp` skill) has its
own usage comment at the top of the file and is covered by the
`.claude/skills/chrome-cdp/SKILL.md` skill. This README covers both
pipelines' scripts: the bitmap pipeline first, then the vector tools in
`tools/art/vector/`.

## Bitmap pipeline

Dependency-free Node 24 ESM scripts (`.mjs`, standard library only; no
`package.json`, no npm install) that take an asset from a text prompt to a
shipped AVIF file. [docs/art/bitmaps.md](../../docs/art/bitmaps.md) is the
full walk-through, with the review criteria and file layout;
[art/style.md](../../art/style.md) is the prompt source they all read.
This section is each script's reference.

All of them resolve the repository root from their own file location, so
they can be run from anywhere with `node tools/art/<script>.mjs ...` as
long as the path is correct; in practice, run them from the repository
root. `AJAYEB_RUN_DIR` (default `~/tmp/agents/ajayeb`, same as
`tools/heavy.py`) is where the private, not-committed working state lives:
raw attempts and contact sheets.

### `lib.mjs`

Not a script; the shared module the other six import. Holds the repo and
run-dir paths, the `<kind>/<slug>` id helpers, the presentation-catalog
reader (parses `web/src/lib/wire/catalog.gen.ts`, so no Go build is
needed), the fixed list of non-catalog assets (backs, textures, the hero,
the AI avatars, the victory scenes, the app icon), the per-kind generation
size / crop / shipped sizes / byte budgets table (`KIND_SPEC`), and small
JSON read/write helpers.

### `manifest.mjs`

```
node tools/art/manifest.mjs           # write art/manifest.json
node tools/art/manifest.mjs --check   # exit 1 if that would change it
```

Rebuilds `art/manifest.json` from the catalog and `art/subjects.json`.
Never writes `subjects.json`. Keeps every existing asset's `status`,
`attempt`, `promptAmend` and `bytes`; only adds ids that are missing and
drops (with a warning on stderr) ids that no longer correspond to a
catalog entry or a static asset.

### `generate.mjs`

```
node tools/art/generate.mjs --ids card/lumber-yard,card/stone-pit
node tools/art/generate.mjs --kind token --limit 3
node tools/art/generate.mjs --ids card/lumber-yard --amend "warmer light"
node tools/art/generate.mjs --kind card --status rejected --limit 5   # only rejected ones
node tools/art/generate.mjs --ids card/lumber-yard --dry-run
```

Generates one Codex `image_gen` attempt per listed asset, strictly
sequentially (never in parallel, even across ids in one invocation). It
always runs under the codex lane; started without the wrapper below, it
re-executes itself under it:

```
python3 tools/heavy.py run --kind codex -- node tools/art/generate.mjs --ids card/lumber-yard
```

Flags: `--ids a,b,c` (exact ids, in order); `--kind KIND` with `--limit N`
(default 1) and `--status S` (default: `todo` and `rejected`) to pick the next matching
assets instead of naming ids; `--amend TEXT` to set this run's prompt
amendment (only meaningful with `--ids`; `--kind` runs use each asset's own
stored `promptAmend`); `--dry-run` to print the prompt and the exact
`codex exec` command without calling Codex or touching the manifest.

`CODEX_BIN` defaults to `~/.local/bin/codex`; `CODEX_HOME` defaults to
`~/.codex` (where `generated_images/` is watched for the new PNG). A
successful attempt is copied to
`${AJAYEB_RUN_DIR}/art-work/raw/<kind>/<slug>/<attempt>.png` and the
manifest's `status` becomes `generated`.

### `sheet.mjs`

```
node tools/art/sheet.mjs --ids card/lumber-yard,card/stone-pit --name pilot
node tools/art/sheet.mjs --kind token
```

Builds a numbered, labelled contact sheet (ImageMagick `montage`) of chosen
attempts (each id's latest attempt by default, or `--attempt N`), written
to `${AJAYEB_RUN_DIR}/art-work/sheets/<name>.jpg`, never into the
repository. Each cell is labelled `<n>. <id> #<attempt>`. `--tile` and `--cell` override the montage grid and cell size
(`4x` and `384x288` by default).

### `apply-review.mjs`

```
node tools/art/apply-review.mjs path/to/decisions.jsonl
node tools/art/apply-review.mjs path/to/decisions.jsonl --dry-run
```

Applies a reviewer's `decisions.jsonl` (one JSON object per line: `{id,
attempt, verdict: "approve"|"reject", reasons?, promptAmend?}`) to
`art/manifest.json`. `approve` sets `status: "approved"`. `reject` sets
`status: "rejected"` and stores `promptAmend` for the next `generate.mjs` run,
unless the asset is already at 3 attempts, in which case it becomes
`needs-human`. A decision whose `attempt` does not match the manifest's
current attempt is skipped as stale. `--dry-run` prints the summary
without writing the manifest.

### `process.mjs`

```
node tools/art/process.mjs --id token/agriculture --src art/originals/token/agriculture.png
node tools/art/process.mjs --id card/lumber-yard \
  --src ~/tmp/agents/ajayeb/art-work/raw/card/lumber-yard/1.png --out /scratch/dir
```

Turns one original into the shipped AVIF sizes from `KIND_SPEC`
(ImageMagick to crop to the exact pixel size, `avifenc` to encode, quality
stepped down from 55 until the byte budget holds or 35 is reached; tokens
get a circular mask composited onto the paper colour `#f4ecd8` first).
Without `--out`, the id **must already be `"approved"`** in the manifest
(set by `apply-review.mjs`); the source is copied to
`art/originals/<kind>/<slug>.png`, the AVIFs go to
`web/static/art/<kind>/`, and the shipped byte counts are recorded in the
manifest. With `--out DIR`, that approval check is skipped and everything
is written under `DIR` instead: nothing is shipped, nothing is copied to
`art/originals/`, and the manifest is left untouched. This is how a
pipeline change (or `M9.B0`'s pilot) is proved end to end without
approving or shipping anything for real.

### `index.mjs`

```
node tools/art/index.mjs
```

Regenerates `web/src/lib/art/available.gen.ts` from every `"approved"`
asset in `art/manifest.json`: an id, its kind, and its shipped variants'
paths, pixel sizes and byte counts. Run it after any `apply-review.mjs` or
`process.mjs` call that changes which assets are approved, so the app's
index never claims a file that was not actually shipped.

## Vector pipeline (`vector/`)

The vector icons, ornaments and logo are hand-authored SVG, drawn by Opus
agents in a render, look and critique loop; Codex is never used for them.
[art/vector/STYLE.md](../../art/vector/STYLE.md) holds the rules and the
loop, [docs/art/README.md](../../docs/art/README.md) the overview. These
scripts are dependency-free Node 24, plus `rsvg-convert` (librsvg) and
ImageMagick 7 (`magick`) where noted. Their outputs go under
`${AJAYEB_RUN_DIR:-~/tmp/agents/ajayeb}/art-work/vector/`, never into the
repository. `render-fast.sh` resolves only the fallbacks of `var()` paints,
so it shows an icon's `--i2` detail as transparent; `vector/fast.mjs`
substitutes both tones.

Run from the repository root. Commands that launch Chrome must run inside the guard's browser lock (`python3 tools/heavy.py run --kind browser -- ...`), which keeps the machine to one headless Chrome; each of them starts one Chrome and closes it before it exits. The others need no lock.

- `lint.mjs`: checks icon SVGs against STYLE.md and prints one line per problem, `path:line: error|warn rule: <element> #n: message`, then a summary on stderr. It exits 1 on any error (with `--strict`, on any warning too) and 2 when it cannot run.
  - `node tools/art/vector/lint.mjs` lints the whole sprite directory, checks it against `names.ts`, and checks the sprite budget.
  - `node tools/art/vector/lint.mjs --set chain` lints one set from `art/vector/sets.json`.
  - `node tools/art/vector/lint.mjs path/to/a.svg path/to/dir` lints files and directories.
- `fast.mjs` (no browser; needs `rsvg-convert` and `magick`): the inner-loop render. `node tools/art/vector/fast.mjs a.svg b.svg` writes one PNG per file, sizes 12 to 96 px on paper above the same on night, with the two tones substituted (`--i1`, `--i2`, and their `-dark` twins).
- `specimen.mjs`: `node tools/art/vector/specimen.mjs --set <set>` writes the self-contained specimen HTML of a set (tokens, a font and the set's SVGs inlined). `shoot.mjs` calls it for you.
- `shoot.mjs` (browser): `python3 tools/heavy.py run --kind browser -- node tools/art/vector/shoot.mjs --set <set> [--iter N]` writes `specimen.html`, `lint.txt`, `light@1x.png`, `light@2x.png`, `dark@1x.png`, `dark@2x.png`, `grey-light@1x.png`, `deuteranopia-light@1x.png`, `protanopia-light@1x.png`, `tritanopia-light@1x.png` and `shots.json` into `art-work/vector/<set>/` (or `iter-N/` under it). `--cvd-themes light,dark` adds the dark theme to the greyscale and colour-vision passes.
- `peep.mjs` (no browser; needs `magick`): `node tools/art/vector/peep.mjs --set <set> [--iter N]` crops the 12 to 16 px strip of the shots and enlarges it 800% with point filtering, into `peep-light@1x.png`, `peep-light@2x.png` and `peep-grey-light@1x.png`.
- `blind.mjs` (browser for `make`): `python3 tools/heavy.py run --kind browser -- node tools/art/vector/blind.mjs make --set <set>` writes shuffled, numbered, unlabelled sheets at 12 px (DPR 2) and 16 px (DPR 1), the candidate names, an answer template and the key into `art-work/vector/blind/<set>-<stamp>/`. Answer every tile before you open the key, then `node tools/art/vector/blind.mjs score --dir <dir> --answers <dir>/answers.txt`. `--mono` draws plain ink shapes only.
- `family.mjs` (browser): `python3 tools/heavy.py run --kind browser -- node tools/art/vector/family.mjs` writes `art-work/vector/family/family@1x.png` and `family@2x.png`: every drawn icon of every set at 16 and 24 px, on paper and on night.
- `lib.mjs`: shared helpers (paths, `sets.json`, the XML reader, sprite symbols, the Chrome page). Not a command.

Every script prints its usage with `--help`.

### Make targets

These wrap the commands above in the guard:

- `make vec-lint` runs `lint.mjs` over the whole sprite directory (guard kind `light`).
- `make vec-shot SET=<set> [ITER=<n>]` runs `shoot.mjs` under `--kind browser`, then `peep.mjs`.
- `make vec-blind SET=<set>` runs `blind.mjs make` under `--kind browser`. Scoring stays a plain `node tools/art/vector/blind.mjs score ...` call.
- `make vec-family` runs `family.mjs` under `--kind browser`.

