#!/usr/bin/env node
// tools/art/gate.mjs
//
// An objective pass/fail check a reviewer runs before writing
// decisions.jsonl (see docs/art/bitmaps.md step 4, "Review"): after the
// shipping crop, a full-bleed painting (the subject painted edge to edge,
// with no margin of its own paper colour left anywhere) reads badly once
// the card or wonder frame feathers into it, so this catches one upstream
// of a human ever looking at a contact sheet.
//
// The check: crop the attempt to its kind's main shipped size (the same
// "fill" crop process.mjs ships with), find the image's own paper colour
// (the corner-median, like process.mjs), then look at the outer 4% band
// around all four edges. If at least 50% of that band is within an RGB
// distance of 45 from the paper colour, the margin reads as paper and the
// image passes; otherwise the painting runs to the edge and it fails.
// Both numbers are flags (--band, --dist, --threshold), not hardcoded
// policy, but these are the pilot's agreed defaults (M9.B3).
//
// This is a margin check only, not a full review: it does not know about
// text, emblems, crowds or anything else in art/style.md's Do/Don't list.
// A pass here is necessary, not sufficient, for approval.
//
// Usage:
//   node tools/art/gate.mjs --kind card --attempt all
//   node tools/art/gate.mjs --ids card/theater,wonder/pyramids --attempt all
//   node tools/art/gate.mjs --ids card/theater#1,card/scriptorium#2
//   node tools/art/gate.mjs --all
//   node tools/art/gate.mjs --src some/original.png --id card/lumber-yard
//   node tools/art/gate.mjs --kind card --attempt latest --json
//
// Flags:
//   --ids a,b,c     ids to check (each may carry #attempt, see below)
//   --kind KIND     restrict to assets of one kind, with --attempt
//   --attempt N|latest|all   which attempt(s) per id (default: latest, the
//                   manifest's recorded attempt); 'all' checks every
//                   attempt file found in AJAYEB_RUN_DIR/art-work/raw/
//   --all           ignore --ids/--kind/--attempt; sweep every attempt
//                   file under AJAYEB_RUN_DIR/art-work/raw/**/*.png
//   --src PATH --id ID   check exactly one file (e.g. an approved
//                   original under art/originals/), bypassing raw-dir
//                   resolution entirely
//   --band FRAC     outer band width as a fraction of min(w,h) (default 0.04)
//   --dist N        RGB distance threshold (default 45)
//   --threshold FRAC  required in-band fraction to pass (default 0.5)
//   --json          print one JSON object per line instead of a table
//   --strict        exit 1 if anything failed (default: always exit 0,
//                   since this is a reporting aid, not a CI gate)

import { readdir, stat } from 'node:fs/promises';
import path from 'node:path';

import { rawDir, KIND_SPEC, idKind, idSlug, loadManifest, cropToRawRgb, cornerPaperColor, die } from './lib.mjs';

function parseArgs(argv) {
  const args = { attempt: 'latest', band: 0.04, dist: 45, threshold: 0.5 };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--ids') args.ids = argv[++i].split(',').map((s) => s.trim()).filter(Boolean);
    else if (a === '--kind') args.kind = argv[++i];
    else if (a === '--attempt') args.attempt = argv[++i];
    else if (a === '--all') args.all = true;
    else if (a === '--src') args.src = argv[++i];
    else if (a === '--id') args.id = argv[++i];
    else if (a === '--band') args.band = Number(argv[++i]);
    else if (a === '--dist') args.dist = Number(argv[++i]);
    else if (a === '--threshold') args.threshold = Number(argv[++i]);
    else if (a === '--json') args.json = true;
    else if (a === '--strict') args.strict = true;
    else die(`unknown argument "${a}"`, 2);
  }
  return args;
}

async function listAttemptNumbers(kind, slug) {
  const dir = path.join(rawDir, kind, slug);
  let entries;
  try {
    entries = await readdir(dir);
  } catch {
    return [];
  }
  return entries
    .map((f) => /^(\d+)\.png$/.exec(f))
    .filter(Boolean)
    .map((m) => Number(m[1]))
    .sort((a, b) => a - b);
}

/** Every attempt file under art-work/raw/<kind>/<slug>/<n>.png, as
 * { id, attempt, file } triples, in directory order. */
async function sweepAllRaw() {
  const out = [];
  let kinds;
  try {
    kinds = await readdir(rawDir, { withFileTypes: true });
  } catch {
    return out;
  }
  for (const kindEnt of kinds) {
    if (!kindEnt.isDirectory()) continue;
    const kind = kindEnt.name;
    const kindDir = path.join(rawDir, kind);
    const slugs = await readdir(kindDir, { withFileTypes: true });
    for (const slugEnt of slugs) {
      if (!slugEnt.isDirectory()) continue;
      const slug = slugEnt.name;
      const attempts = await listAttemptNumbers(kind, slug);
      for (const attempt of attempts) {
        out.push({ id: `${kind}/${slug}`, attempt, file: path.join(kindDir, slug, `${attempt}.png`) });
      }
    }
  }
  return out;
}

/** Resolves --ids/--kind/--attempt into a list of { id, attempt, file }.
 * An id in --ids may carry "#<attempt>" (or "#latest") to override
 * --attempt for just that one id, so one call can check several attempts
 * of several ids at once, e.g. "card/theater#1,card/theater#2". */
async function resolveTargets(args, manifest) {
  if (args.all) return sweepAllRaw();

  if (args.src) {
    if (!args.id) die('--src needs --id', 2);
    return [{ id: args.id, attempt: null, file: args.src }];
  }

  let entries; // [{ id, attemptSpec }]
  if (args.ids) {
    entries = args.ids.map((spec) => {
      const hashIdx = spec.indexOf('#');
      return hashIdx === -1
        ? { id: spec, attemptSpec: args.attempt }
        : { id: spec.slice(0, hashIdx), attemptSpec: spec.slice(hashIdx + 1) };
    });
  } else if (args.kind) {
    entries = Object.values(manifest.assets)
      .filter((a) => a.kind === args.kind)
      .map((a) => ({ id: a.id, attemptSpec: args.attempt }));
  } else {
    die('pass --ids a,b,c, --kind KIND, --all, or --src PATH --id ID', 2);
  }

  const targets = [];
  for (const { id, attemptSpec } of entries) {
    const kind = idKind(id);
    const slug = idSlug(id);
    if (attemptSpec === 'all') {
      const attempts = await listAttemptNumbers(kind, slug);
      for (const attempt of attempts) targets.push({ id, attempt, file: path.join(rawDir, kind, slug, `${attempt}.png`) });
      continue;
    }
    const attempt =
      attemptSpec === 'latest' || !attemptSpec ? manifest.assets[id]?.attempt || 0 : Number(attemptSpec);
    if (!attempt) {
      process.stderr.write(`gate.mjs: ${id} has no recorded attempt, skipping\n`);
      continue;
    }
    targets.push({ id, attempt, file: path.join(rawDir, kind, slug, `${attempt}.png`) });
  }
  return targets;
}

/** The outer-band pass/fail check itself: crops to the kind's main shipped
 * size, finds the crop's own paper colour, and measures how much of the
 * outer `band` fraction is within `dist` of that colour. */
async function checkOne(id, file, band, dist, threshold) {
  const kind = idKind(id);
  const spec = KIND_SPEC[kind];
  if (!spec) throw new Error(`gate.mjs: no KIND_SPEC for kind "${kind}" (${id})`);
  const size = spec.sizes[0]; // the main shipped size (not a thumb)

  await stat(file); // throws a clear ENOENT if the attempt file is missing
  const { buf, w, h } = await cropToRawRgb(file, size.w, size.h);
  const paper = cornerPaperColor(buf, w, h);

  const bandWidth = Math.max(1, Math.round(band * Math.min(w, h)));
  let total = 0;
  let within = 0;
  for (let y = 0; y < h; y++) {
    const inBandRow = y < bandWidth || y >= h - bandWidth;
    for (let x = 0; x < w; x++) {
      if (!inBandRow && x >= bandWidth && x < w - bandWidth) continue; // interior
      total++;
      const idx = (y * w + x) * 3;
      const dr = buf[idx] - paper[0];
      const dg = buf[idx + 1] - paper[1];
      const db = buf[idx + 2] - paper[2];
      if (Math.sqrt(dr * dr + dg * dg + db * db) <= dist) within++;
    }
  }
  const frac = total ? within / total : 0;
  return { paper, bandWidth, w, h, frac, pass: frac >= threshold };
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const manifest = await loadManifest();
  const targets = await resolveTargets(args, manifest);
  if (!targets.length) die('nothing to check (no matching ids/attempts found)');

  let passCount = 0;
  let failCount = 0;
  const rows = [];

  for (const { id, attempt, file } of targets) {
    let result;
    try {
      result = await checkOne(id, file, args.band, args.dist, args.threshold);
    } catch (err) {
      process.stderr.write(`gate.mjs: ${id}${attempt ? ` #${attempt}` : ''}: ${err.message}\n`);
      continue;
    }
    if (result.pass) passCount++;
    else failCount++;
    rows.push({ id, attempt, file, ...result });
  }

  if (args.json) {
    for (const r of rows) {
      process.stdout.write(
        JSON.stringify({
          id: r.id,
          attempt: r.attempt,
          pass: r.pass,
          fraction: Number(r.frac.toFixed(4)),
          paper: r.paper,
          bandPx: r.bandWidth,
          size: `${r.w}x${r.h}`,
        }) + '\n',
      );
    }
  } else {
    for (const r of rows) {
      const label = `${r.id}${r.attempt ? ` #${r.attempt}` : ''}`;
      const pct = (r.frac * 100).toFixed(1);
      const paperStr = `rgb(${r.paper.join(',')})`;
      process.stdout.write(
        `${r.pass ? 'PASS' : 'FAIL'}  ${label.padEnd(28)} band=${pct.padStart(5)}%  paper=${paperStr}  (${r.w}x${r.h}, band ${r.bandWidth}px)\n`,
      );
    }
    process.stdout.write(`gate.mjs: ${passCount} pass, ${failCount} fail, ${rows.length} checked\n`);
  }

  if (args.strict && failCount) process.exit(1);
}

main().catch((err) => die(err.stack || String(err)));
