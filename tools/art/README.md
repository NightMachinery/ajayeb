# tools/art: the art pipelines' scripts

Two independent pipelines produce Ajayeb's original art (see AGENTS.md
"Assets"): a hand-authored SVG icon set made by Opus subagents in a
render-screenshot-critique loop, and Codex-generated bitmap art for cards,
wonders, tokens and the rest. `render-fast.sh` (the shared inner-loop SVG
rasterizer, used by both the icon loop and the `chrome-cdp` skill) is
documented in [docs/dev/](../../docs/dev/) and the `chrome-cdp` skill, not
here.

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
node tools/art/generate.mjs --kind card --status rejected --limit 5
node tools/art/generate.mjs --ids card/lumber-yard --dry-run
```

Generates one Codex `image_gen` attempt per listed asset, strictly
sequentially (never in parallel, even across ids in one invocation). Always
run it under the codex lane:

```
python3 tools/heavy.py run --kind codex -- node tools/art/generate.mjs --ids card/lumber-yard
```

Flags: `--ids a,b,c` (exact ids, in order); `--kind KIND` with `--limit N`
(default 1) and `--status S` (default `todo`) to pick the next matching
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
to `${AJAYEB_RUN_DIR}/art-work/sheets/<name>.jpg` — never into the
repository. `--tile` and `--cell` override the montage grid and cell size
(`4x` and `384x288` by default).

### `apply-review.mjs`

```
node tools/art/apply-review.mjs path/to/decisions.jsonl
node tools/art/apply-review.mjs path/to/decisions.jsonl --dry-run
```

Applies a reviewer's `decisions.jsonl` (one JSON object per line: `{id,
attempt, verdict: "approve"|"reject", reasons?, promptAmend?}`) to
`art/manifest.json`. `approve` sets `status: "approved"`. `reject` sets
`status: "todo"` and stores `promptAmend` for the next `generate.mjs` run,
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
is written under `DIR` instead — nothing is shipped, nothing is copied to
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
