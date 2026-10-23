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
raw attempts and contact sheets. Every script prints its usage, the
comment block at its top, with `--help`, and rejects an unknown flag with
exit code 2.

### `lib.mjs`

Not a script; the shared module the other scripts import. Holds the repo
and run-dir paths, the `<kind>/<slug>` id helpers, the presentation-catalog
reader (parses `web/src/lib/wire/catalog.gen.ts`, so no Go build is
needed), the fixed list of non-catalog assets (backs, textures, the hero,
the AI avatars, the victory scenes, the app icon), the per-kind generation
size / crop / shipped sizes / byte budgets table (`KIND_SPEC`), small JSON
read/write helpers, and the raw-pixel helpers `process.mjs` and `gate.mjs`
share: `cropToRawRgb`/`rawRgbToPng` (the shipping crop as RGB bytes, and
back to a PNG), `cornerPaperColor` (an image's own paper colour, the
corner-median), `colorDistance`, `hexToRgb`, `PAPER_HEX` (the app's
`--card-paper` token, `#f4ecdc`), `rgbToLab` and `deltaE2000` (the
perceptual colour difference `gate.mjs`'s paper check uses) and
`imageSize` (via `magick identify`).

### `manifest.mjs`

```
node tools/art/manifest.mjs           # write art/manifest.json
node tools/art/manifest.mjs --check   # exit 1 if that would change it
```

Rebuilds `art/manifest.json` (or the file `AJAYEB_ART_MANIFEST` names)
from the catalog and `art/subjects.json`. Never writes `subjects.json`.
Keeps every existing asset's `status`, `attempt`, `promptAmend` and
`bytes`, and the review's `approvedAttempt` and `reasons` when present; only adds ids that are missing and
drops (with a warning on stderr) ids that no longer correspond to a
catalog entry or a static asset.

### `generate.mjs`

```
node tools/art/generate.mjs --ids card/lumber-yard,card/stone-pit
node tools/art/generate.mjs --kind token --limit 3
node tools/art/generate.mjs --ids card/lumber-yard --amend "warmer light"
node tools/art/generate.mjs --kind card --status rejected --limit 5   # only rejected ones
node tools/art/generate.mjs --ids card/lumber-yard --dry-run
node tools/art/generate.mjs --ids wonder/pyramids --ref   # with a style anchor
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
stored `promptAmend`); `--ref` to pass `art/refs/<kind>-anchor.{jpg,png}`
to Codex as a style anchor, when one exists (**off by default**: the
M9.B3 pilot found a same-subject reference makes Codex return a near-copy
of the anchor instead of a new illustration; the style prefix alone keeps
the look consistent across subjects); `--dry-run` to print the prompt and
the exact `codex exec` command without calling Codex or touching the
manifest.

`CODEX_BIN` defaults to `~/.local/bin/codex`; `CODEX_HOME` defaults to
`~/.codex` (where `generated_images/` is watched for the new PNG). The
newest new PNG's pixel size is checked against the kind's requested `gen`
size (`magick identify`), and the attempt is rejected, with the manifest
left untouched, if its aspect ratio is more than 1% off or it is smaller
than requested. A larger image of the same shape is accepted: Codex's
`image_gen` returns every square request as 1254x1254, while a
reference-image run once returned that square for a 1536x1024 card. A
successful attempt is copied to
`${AJAYEB_RUN_DIR}/art-work/raw/<kind>/<slug>/<attempt>.png`, numbered one
past the larger of the manifest's counter and the highest attempt file
already there, and the
manifest's `status` becomes `generated`.

### `sheet.mjs`

```
node tools/art/sheet.mjs --ids card/lumber-yard,card/stone-pit --name pilot
node tools/art/sheet.mjs --kind token
node tools/art/sheet.mjs --ids card/theater#1,card/theater#2,card/theater#3
node tools/art/sheet.mjs --ids card/theater --attempt all
```

Builds a numbered, labelled contact sheet (ImageMagick `montage`) of chosen
attempts, written to `${AJAYEB_RUN_DIR}/art-work/sheets/<name>.jpg`, never
into the repository. By default each id contributes its latest attempt
(`--attempt N`, or `latest` to be explicit), but any `--ids` entry can
carry its own `#<attempt>` (a number, `latest` or `all`), so one call can
put several attempts of several ids on the same sheet, for example three
rejected variants of one card next to the one that was approved. Each cell
is labelled `<n>. <id> #<attempt>`. `--tile` and `--cell` override the
montage grid and cell size (`4x` and `384x288` by default).

### `gate.mjs`

```
node tools/art/gate.mjs --kind card --attempt all
node tools/art/gate.mjs --ids card/theater#1,wonder/pyramids#3
node tools/art/gate.mjs --all                 # every attempt file on disk
node tools/art/gate.mjs --src some.png --id card/lumber-yard
node tools/art/gate.mjs --kind token --attempt latest --json
```

An objective paper and margin check a reviewer runs before `sheet.mjs`,
not a replacement for looking at the images. It crops each attempt to its
kind's main shipped size (the same crop `process.mjs` ships with) and finds
the crop's own paper colour (the corner-median). Then it runs two checks,
and an attempt passes only when both do:

- the band check fails it if less than `--threshold` (default 50%) of the
  outer `--band` fraction (default 4% of `min(w,h)`) is within `--dist`
  (default 45, RGB) of that colour, because a painting with no margin
  anywhere reads badly once the card or wonder frame feathers into it;
- the paper check fails it if that paper colour is more than `--paper-max`
  (default 18, CIEDE2000) from `PAPER_HEX`, because the band check is
  relative and passes an image whose corners are gold leaf as if gold were
  its paper. `--paper-max 0` turns it off. The threshold's calibration on
  the pilot is in `docs/art/bitmaps.md` (step 3) and at the top of the
  script.

Each line shows the band fraction, the detected paper, its `dE00` from
`PAPER_HEX` and, on a failure, which check failed. `--ids` accepts the same
`#<attempt>` syntax as `sheet.mjs`. `--json` prints one JSON object per
line instead of a table; `--strict` exits 1 if anything failed, for
scripting a hard gate.

### `apply-review.mjs`

```
node tools/art/apply-review.mjs path/to/decisions.jsonl
node tools/art/apply-review.mjs path/to/decisions.jsonl --dry-run
node tools/art/apply-review.mjs decisions.jsonl --max-attempts 4
node tools/art/apply-review.mjs decisions.jsonl --manifest /scratch/manifest.json
```

Applies a reviewer's `decisions.jsonl` (one JSON object per line: `{id,
attempt, verdict: "approve"|"reject"|"needs-human", reasons?,
promptAmend?}`) to `art/manifest.json`.

- `approve` sets `status: "approved"` and `approvedAttempt` to the
  decision's attempt, which may be earlier than the latest one (its raw
  file must exist); `process.mjs` and `index.mjs` use that attempt.
- `reject` sets `status: "rejected"` and stores `promptAmend` for the next
  `generate.mjs` run, unless the asset is already at `--max-attempts`
  attempts (default 3), in which case it becomes `needs-human`.
- `needs-human` sets that status directly, with its reasons, at any
  attempt count.

A rejection whose `attempt` does not match the manifest's current attempt
is skipped as stale; an approval or `needs-human` may name an earlier
attempt but never a later one. A decision with no `attempt` is about the
current one. `--manifest PATH` (or `AJAYEB_ART_MANIFEST`) works on a
scratch copy of the manifest; `--dry-run` prints the summary without
writing it.

### `process.mjs`

```
node tools/art/process.mjs --id token/agriculture        # the approved attempt
node tools/art/process.mjs --id token/agriculture --src art/originals/token/agriculture.webp
node tools/art/process.mjs --id card/lumber-yard \
  --src ~/tmp/agents/ajayeb/art-work/raw/card/lumber-yard/1.png --out /scratch/dir
```

Turns one original into the shipped AVIF sizes from `KIND_SPEC`. With no
`--src`, the source is the approved attempt's raw file
(`art-work/raw/<kind>/<slug>/<approvedAttempt>.png`, or the latest attempt
for a `--out` run of an unapproved asset); a `--src` that is another raw
attempt of the same id is refused unless `--out` is given. The first line
of output names the file and the attempt. `--manifest PATH` reads (and
updates) a scratch manifest instead. For each
size: crops to the exact pixel size (ImageMagick) and reads it back as raw
RGB; maps the crop's own paper colour onto `PAPER_HEX` (`#f4ecdc`) with a
smooth per-pixel falloff, so only paper-like pixels shift and the
painting's other colours don't; for a progress token, detects the
medallion's own painted rim (walking inward from each edge for the first
non-paper pixel) and masks to that instead of always touching the square's
edges, with a 1.5px antialiased edge, so a medallion painted with its own
margin doesn't leave a visible ivory "halo" ring; then hands the result to
`avifenc`, quality stepped down from 55 until the byte budget holds or 35
is reached.

First of all it writes the source as the committed original,
`art/originals/<kind>/<slug>.webp`, in lossless WebP (`cwebp -lossless
-exact -z 9`, or ImageMagick's `webp:lossless=true` without `cwebp`), and
checks it: `magick compare -metric AE` over all channels must count 0
differing pixels and the raw 16-bit RGBA bytes must be equal, or the run
fails before shipping anything. It prints the WebP's size next to the
source's. `--src` may be a `.webp` original.

Without `--out`, the id **must already be `"approved"`** in the manifest
(set by `apply-review.mjs`); the AVIFs go to `web/static/art/<kind>/`, and
the shipped byte counts are recorded in the manifest. With `--out DIR`,
that approval check is skipped and everything is written under `DIR`
instead (the original as `DIR/originals/<kind>/<slug>.webp`): nothing is
shipped, nothing is written to `art/originals/`, and the manifest is left
untouched. This is how a
pipeline change (or `M9.B0`'s pilot) is proved end to end without
approving or shipping anything for real.

```
node tools/art/process.mjs --batch                 # every approved asset
node tools/art/process.mjs --batch --kind wonder --force --out /scratch/dir
```

`--batch` re-derives the shipped sizes of every approved asset (or those of
`--kind`, or the comma-separated `--ids`) from its committed original in
`art/originals/`, with the same crop, paper mapping, mask and quality
stepping. By default it writes only the files that do not exist yet, so a
size added to `KIND_SPEC` reaches every approved asset while the files
already shipped stay byte-identical; `--force` rewrites every size, and
`--out DIR` writes under `DIR` instead. It never writes `art/originals/` or
the manifest (`index.mjs` takes the byte counts from the files), and it
fails at the end if an approved id that needed a file has no original. A
`--force --out` run reproduces the shipped files byte for byte with the
same `avifenc`, which is the quick proof that the originals and the
pipeline still agree.

### `index.mjs`

```
node tools/art/index.mjs
node tools/art/index.mjs --check
node tools/art/index.mjs --manifest /scratch/manifest.json --out /scratch/available.gen.ts
```

Regenerates `web/src/lib/art/available.gen.ts` from every `"approved"`
asset in `art/manifest.json`: an id, its kind, and its shipped variants'
paths, pixel sizes and byte counts. Each path ends in `?v=` and the first
10 hex digits of the shipped file's SHA-256 (the file keeps its name), so
the server can cache it for a year; the byte counts are read from the
files, and a missing file is an error. The module is formatted with the
web package's Prettier (so `make install` first), which is what `make
lint-web` checks. `--check` writes nothing and exits 1 when the file on
disk differs from what it would write; `src/lib/art/index-check.test.ts`
runs it in `make test-web`. It prints each approved id's approved attempt
(and notes when it is earlier than the latest) and its committed original
(`.webp`, or a legacy `.png`), and warns about an approved id with no
original yet (`--quiet` drops these lines). Then a report: the shipped
total, the full prefetch plan's bytes per part on a phone at DPR 3 and a
desktop at DPR 1, and the high-DPI check of each kind (its largest display
times 3 against its widest size). `--manifest` and `--out` point it at
scratch files for a proof. Run it after any `apply-review.mjs` or
`process.mjs` call that changes which assets are approved or which files
ship, so the app's index never claims a file that was not actually shipped.

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
  - `node tools/art/vector/lint.mjs` lints the whole icon source directory, checks it against `names.ts`, and checks the aggregate icon source-set budget. It then lints the other games' icon directories (`GAME_ICON_DIRS` in `lib.mjs`, today `web/src/lib/games/haftgane/icons/svg/`) by the same rules, and refuses a file there that is named like a Duel icon, since a reused Duel drawing is referenced from `sets.json`, not copied. Those files count toward neither `names.ts` nor Duel's budget.
  - The legacy constants `SPRITE_RAW_MAX` (40 KB) and `SPRITE_GZIP_MAX` (10 KB) bound the complete source set serialized in the vector tools' specimen SVG envelope, including its symbol wrappers. They do not bound the client bundle or each rendered icon: the client draws local SVG shapes directly.
  - `node tools/art/vector/lint.mjs --set chain` lints one set from `art/vector/sets.json`.
  - `node tools/art/vector/lint.mjs path/to/a.svg path/to/dir` lints files and directories.
- `fast.mjs` (no browser; needs `rsvg-convert` and `magick`): the inner-loop render. `node tools/art/vector/fast.mjs a.svg b.svg` writes one PNG per file, sizes 12 to 96 px on paper above the same on night, with the two tones substituted (`--i1`, `--i2`, and their `-dark` twins).
- `specimen.mjs`: `node tools/art/vector/specimen.mjs --set <set>` writes the self-contained specimen HTML of a set (tokens, a font and the set's SVGs inlined). `shoot.mjs` calls it for you.
- `shoot.mjs` (browser): `python3 tools/heavy.py run --kind browser -- node tools/art/vector/shoot.mjs --set <set> [--iter N]` writes `specimen.html`, `lint.txt`, `light@1x.png`, `light@2x.png`, `dark@1x.png`, `dark@2x.png`, `grey-light@1x.png`, `deuteranopia-light@1x.png`, `protanopia-light@1x.png`, `tritanopia-light@1x.png` and `shots.json` into `art-work/vector/<set>/` (or `iter-N/` under it). `--cvd-themes light,dark` adds the dark theme to the greyscale and colour-vision passes.
- `peep.mjs` (no browser; needs `magick`): `node tools/art/vector/peep.mjs --set <set> [--iter N]` crops the 12 to 16 px strip of the shots and enlarges it 800% with point filtering, into `peep-light@1x.png`, `peep-light@2x.png` and `peep-grey-light@1x.png`.
- `blind.mjs` (browser for `make`): `python3 tools/heavy.py run --kind browser -- node tools/art/vector/blind.mjs make --set <set>` writes shuffled, numbered, unlabelled sheets, the candidate names, an answer template and the key into `art-work/vector/blind/<set>-<stamp>/`. Answer every tile before you open the key, then `node tools/art/vector/blind.mjs score --dir <dir> --answers <dir>/answers.txt`. Its `make` flags:
  - `--sizes LIST` picks the sheets, as `<px>@<dpr>` joined by commas, such as `12@2,14@2,16@1`. Each becomes `sheet-<px>px@<dpr>x.png`, shuffled in its own order; at 14 px and below a sheet uses the `-sm` variant where one exists, as the app does. The default, `12@2,16@1`, gives the two sheets of the earlier runs, so a recorded seed still reproduces them.
  - `--reader-dir DIR` also copies the sheets, `names.txt` and the answer template, and never the key, into `DIR`, and prints both directories and the `score` command. Point a fresh agent at `DIR` and it can take the test truly blind. `DIR` is created; `make` refuses before drawing anything if it exists and is not empty.
  - `--seed N` fixes the shuffle (the key records it), `--mono` draws plain ink shapes only, and `--force` skips the browser lock.
  - A set's `neighbours` in `sets.json` (ids from other sets, such as Duel's `shield` beside a new conflict token) join the sheets and `names.txt` as ordinary candidates, each in its own set's ink, so the test also catches a new glyph read as an existing one.
  - In the answers, a sheet is named `<px>@<dpr>` (`12@2`). `score` also accepts a bare `<px>` (`12`) when only one sheet has that size, so the directories made before `--sizes`, whose sheets were `12` and `16`, still score.
- `family.mjs` (browser): `python3 tools/heavy.py run --kind browser -- node tools/art/vector/family.mjs` writes `art-work/vector/family/family@1x.png` and `family@2x.png`: every drawn icon of every set at 16 and 24 px, on paper and on night.
- `lib.mjs`: shared helpers (paths, `sets.json`, the XML reader, specimen-only symbols, the Chrome page). Not a command.

Every script prints its usage with `--help`. A usage mistake or another expected error prints only its message and exits 1 (a refused browser launch exits 2); set `AJAYEB_STACK=1` to print the stack trace for every error. The `tools/chrome/` scripts follow the same rule.

### Make targets

These wrap the commands above in the guard:

- `make vec-lint` runs `lint.mjs` over the whole icon source directory and the other games' icon directories (guard kind `light`).
- `make vec-shot SET=<set> [ITER=<n>]` runs `shoot.mjs` under `--kind browser`, then `peep.mjs`.
- `make vec-blind SET=<set>` runs `blind.mjs make` under `--kind browser`. Scoring stays a plain `node tools/art/vector/blind.mjs score ...` call.
- `make vec-family` runs `family.mjs` under `--kind browser`.

