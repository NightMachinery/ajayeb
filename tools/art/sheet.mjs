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
//
// Each id contributes its chosen attempt's PNG from
// <AJAYEB_RUN_DIR>/art-work/raw/<kind>/<slug>/<attempt>.png (default
// attempt: the manifest's recorded `attempt`, i.e. the latest one).
// Labels are "<n>. <id> #<attempt>" (n counts from 1 across the sheet), drawn by montage itself (not baked into
// the raw PNGs), so they never leak into a shipped file.

import { spawn } from 'node:child_process';
import { mkdir, readdir, stat } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';

import { rawDir, sheetsDir, idKind, idSlug, loadManifest, die } from './lib.mjs';

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
    if (a === '--ids') args.ids = argv[++i].split(',').map((s) => s.trim()).filter(Boolean);
    else if (a === '--kind') args.kind = argv[++i];
    else if (a === '--attempt') args.attempt = argv[++i]; // 'latest' or a number
    else if (a === '--name') args.name = argv[++i];
    else if (a === '--tile') args.tile = argv[++i];
    else die(`unknown argument "${a}"`, 2);
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

async function resolveIds(args, manifest) {
  if (args.ids) return args.ids;
  if (!args.kind) die('pass --ids a,b,c or --kind KIND', 2);
  return Object.values(manifest.assets)
    .filter((a) => a.kind === args.kind && a.attempt > 0)
    .map((a) => a.id);
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const manifest = await loadManifest();
  const ids = await resolveIds(args, manifest);
  if (!ids.length) die('no ids resolved (nothing generated yet for this selection?)');

  const files = [];
  const labels = [];
  for (const id of ids) {
    const kind = idKind(id);
    const slug = idSlug(id);
    const asset = manifest.assets[id];
    const attempt =
      args.attempt === 'latest' || !args.attempt ? asset?.attempt || 0 : Number(args.attempt);
    const file =
      attempt > 0
        ? path.join(rawDir, kind, slug, `${attempt}.png`)
        : await latestAttemptFile(kind, slug);
    if (!file) {
      process.stderr.write(`sheet.mjs: no raw attempt found for ${id}, skipping\n`);
      continue;
    }
    try {
      await stat(file);
    } catch {
      process.stderr.write(`sheet.mjs: ${file} does not exist, skipping ${id}\n`);
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
