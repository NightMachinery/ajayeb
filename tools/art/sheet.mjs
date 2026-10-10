#!/usr/bin/env node
// tools/art/sheet.mjs
//
// Builds a numbered, labelled contact sheet of chosen attempts with
// ImageMagick's montage, for a reviewer to look at before writing
// decisions.jsonl (see apply-review.mjs). Sheets are scratch output: they
// go under <AJAYEB_RUN_DIR>/art-work/sheets/, never into the repository.
//
// Usage:
//   node tools/art/sheet.mjs --ids card/lumber-yard,card/stone-pit
//   node tools/art/sheet.mjs --kind token --attempt latest
//   node tools/art/sheet.mjs --ids card/lumber-yard --name lumber-pilot
//   node tools/art/sheet.mjs --ids card/theater#1,card/theater#2,card/theater#3
//   node tools/art/sheet.mjs --ids card/theater --attempt all
//
// Each id contributes its chosen attempt's PNG from
// <AJAYEB_RUN_DIR>/art-work/raw/<kind>/<slug>/<attempt>.png. By default
// that is the manifest's recorded `attempt` (the latest one), but any
// --ids entry can carry its own "#<attempt>" (or "#latest", or "#all" for
// every attempt found on disk), so one call can put several attempts of
// several ids on the same sheet, e.g. comparing three rejected variants of
// one card next to the attempt that was approved. --attempt sets the
// default for entries with no "#", and alone (with no --ids) still works
// as a per-id override via --attempt all. Labels are "<n>. <id> #<attempt>"
// (n counts from 1 across the sheet), drawn by montage itself (not baked
// into the raw PNGs), so they never leak into a shipped file.
//
// Flags:
//   --ids a,b,c     ids to put on the sheet, each optionally "#<attempt>",
//                   "#latest" or "#all"
//   --kind KIND     every asset of one kind that has an attempt, when
//                   --ids is not given
//   --attempt N|latest|all   the default attempt for entries with no "#"
//                   (default: latest)
//   --name NAME     the sheet's file name, art-work/sheets/NAME.jpg
//                   (default: sheet-<timestamp>)
//   --tile COLSxROWS  montage grid (default: up to 4 columns)
//   --cell WxH      each cell's size (default 384x288)
//   --help          print this usage and exit

import { spawn } from 'node:child_process';
import { mkdir, readdir, stat } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';

import { rawDir, sheetsDir, idKind, idSlug, loadManifest, printUsage, die } from './lib.mjs';

// montage's -label needs a font it can rasterize; a bare install (notably
// this repo's macOS machines) registers none by default, so point at a
// system font directly instead of making every caller pass --font.
const FONT_CANDIDATES = [
  '/System/Library/Fonts/Supplemental/Arial.ttf',
  '/System/Library/Fonts/Helvetica.ttc',
  '/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf',
  '/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf',
];

function findFont() {
  return FONT_CANDIDATES.find((f) => existsSync(f)) ?? null;
}

function parseArgs(argv) {
  const args = { tile: null, cell: '384x288' };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--help' || a === '-h') printUsage(import.meta.url);
    else if (a === '--ids') args.ids = argv[++i].split(',').map((s) => s.trim()).filter(Boolean);
    else if (a === '--kind') args.kind = argv[++i];
    else if (a === '--attempt') args.attempt = argv[++i]; // 'latest' or a number
    else if (a === '--name') args.name = argv[++i];
    else if (a === '--tile') args.tile = argv[++i];
    else if (a === '--cell') args.cell = argv[++i];
    else die(`unknown argument "${a}" (see --help)`, 2);
  }
  return args;
}

async function latestAttemptFile(kind, slug) {
  const dir = path.join(rawDir, kind, slug);
  let entries;
  try {
    entries = await readdir(dir);
  } catch {
    return null;
  }
  const nums = entries
    .map((f) => /^(\d+)\.png$/.exec(f))
    .filter(Boolean)
    .map((m) => Number(m[1]))
    .sort((a, b) => b - a);
  return nums.length ? path.join(dir, `${nums[0]}.png`) : null;
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

/** Resolves --ids/--kind into a list of { id, attemptSpec } entries. An id
 * in --ids may carry "#<attempt>" (a number, "latest" or "all") to
 * override --attempt for just that one id. */
async function resolveEntries(args, manifest) {
  if (args.ids) {
    return args.ids.map((spec) => {
      const hashIdx = spec.indexOf('#');
      return hashIdx === -1
        ? { id: spec, attemptSpec: args.attempt || 'latest' }
        : { id: spec.slice(0, hashIdx), attemptSpec: spec.slice(hashIdx + 1) };
    });
  }
  if (!args.kind) die('pass --ids a,b,c or --kind KIND', 2);
  return Object.values(manifest.assets)
    .filter((a) => a.kind === args.kind && a.attempt > 0)
    .map((a) => ({ id: a.id, attemptSpec: args.attempt || 'latest' }));
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const manifest = await loadManifest();
  const entries = await resolveEntries(args, manifest);
  if (!entries.length) die('no ids resolved (nothing generated yet for this selection?)');

  // Expand each entry to one or more concrete (id, attempt) picks: "all"
  // becomes every attempt found on disk, a bare id with no "#" falls back
  // to latestAttemptFile when the manifest has no recorded attempt.
  const picks = [];
  for (const { id, attemptSpec } of entries) {
    const kind = idKind(id);
    const slug = idSlug(id);
    if (attemptSpec === 'all') {
      const nums = await listAttemptNumbers(kind, slug);
      for (const n of nums) picks.push({ id, attempt: n, file: path.join(rawDir, kind, slug, `${n}.png`) });
      continue;
    }
    const asset = manifest.assets[id];
    let attempt = attemptSpec === 'latest' ? asset?.attempt || 0 : Number(attemptSpec);
    let file = attempt > 0 ? path.join(rawDir, kind, slug, `${attempt}.png`) : null;
    if (!file) {
      file = await latestAttemptFile(kind, slug);
      attempt = 0;
    }
    if (file) picks.push({ id, attempt, file });
  }

  const files = [];
  const labels = [];
  for (const { id, attempt, file } of picks) {
    try {
      await stat(file);
    } catch {
      process.stderr.write(`sheet.mjs: ${file} does not exist, skipping ${id}${attempt ? ` #${attempt}` : ''}\n`);
      continue;
    }
    files.push(file);
    labels.push(`${files.length}. ${id} #${attempt || '?'}`);
  }
  if (!files.length) die('no attempt files found for any requested id');

  await mkdir(sheetsDir, { recursive: true });
  const name = args.name || `sheet-${Date.now()}`;
  const outPath = path.join(sheetsDir, `${name}.jpg`);

  // montage needs each input individually -label'd, so pair file N with
  // label N through -set and the %t token is not enough here (our labels
  // are not filenames); build the arg list as [-label L file]*.
  const montageArgs = [];
  const font = findFont();
  if (font) {
    montageArgs.push('-font', font);
    for (let i = 0; i < files.length; i++) montageArgs.push('-label', labels[i], files[i]);
  } else {
    process.stderr.write('sheet.mjs: no label font found on this machine; the sheet will render without labels\n');
    montageArgs.push(...files);
  }
  montageArgs.push(
    '-resize',
    args.cell,
    '-tile',
    // Up to 4 columns; fewer when there are fewer attempts, so a short
    // sheet is not padded with empty cells.
    args.tile || `${Math.min(4, files.length)}x`,
    '-geometry',
    '+6+6',
    '-background',
    '#f4ecd8',
    outPath,
  );

  await new Promise((resolve, reject) => {
    const child = spawn('magick', ['montage', ...montageArgs], { stdio: ['ignore', 'pipe', 'pipe'] });
    let stderr = '';
    child.stderr.on('data', (d) => (stderr += d));
    child.on('error', reject);
    child.on('close', (code) => (code === 0 ? resolve() : reject(new Error(`magick montage exited ${code}\n${stderr}`))));
  });

  process.stdout.write(`sheet.mjs: wrote ${files.length} attempt(s) to ${outPath}\n`);
}

main().catch((err) => die(err.stack || String(err)));
