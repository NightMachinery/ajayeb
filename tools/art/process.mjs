#!/usr/bin/env node
// tools/art/process.mjs
//
// Turns one approved (or, with --scratch, not-yet-approved) original into
// the shipped AVIF sizes from the asset table in art/style.md /
// docs/art/bitmaps.md, and records their byte counts in art/manifest.json.
//
//   magick raw.png -resize WxH^ -gravity center -extent WxH -strip \
//     <crop/mask steps> ppm:- | avifenc --stdin -q 55 ... out.avif
//
// Quality steps down (55 -> 45 -> 35) until each file's budget holds, or
// the lowest quality is kept with a warning. Progress tokens additionally
// get a circular mask onto the paper colour before resizing, so a round
// medallion sits on square card art with no visible corners.
//
// Usage:
//   node tools/art/process.mjs --id token/agriculture \
//     --src art/originals/token/agriculture.png
//   node tools/art/process.mjs --id card/lumber-yard \
//     --src ~/tmp/agents/ajayeb/art-work/raw/card/lumber-yard/1.png \
//     --out /scratch/dir            # write AVIFs there instead of
//                                    # web/static/art, and skip the
//                                    # "copy to art/originals" step
//
// With --out, nothing is copied to art/originals/ or web/static/art/: this
// is how M9.B0's pilot proves the pipeline without shipping or approving
// anything. Without --out, the source is copied to
// art/originals/<kind>/<slug>.png and the AVIFs go to web/static/art/.

import { spawn } from 'node:child_process';
import { mkdir, copyFile, stat } from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';

import {
  KIND_SPEC,
  shippedFileName,
  idKind,
  idSlug,
  originalsDir,
  shippedRoot,
  loadManifest,
  saveManifest,
  die,
} from './lib.mjs';

// Paper colour the token mask composites onto (tokens/CatalogSymbol medallion
// ground), matching --paper in web/src/lib/styles/tokens.css (ivory,
// oklch(0.96 0.025 85) ~= #f4ecdc).
const PAPER_HEX = '#f4ecdc';

function parseArgs(argv) {
  const args = { qualities: [55, 45, 35] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--id') args.id = argv[++i];
    else if (a === '--src') args.src = argv[++i];
    else if (a === '--out') args.out = argv[++i];
    else die(`unknown argument "${a}"`, 2);
  }
  if (!args.id || !args.src) die('usage: process.mjs --id <id> --src <original.png> [--out DIR]', 2);
  return args;
}

function run(cmd, args, opts = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(cmd, args, { stdio: ['ignore', 'pipe', 'pipe'], ...opts });
    let stderr = '';
    child.stderr.on('data', (d) => (stderr += d));
    child.on('error', reject);
    child.on('close', (code) => {
      if (code !== 0) reject(new Error(`${cmd} ${args.join(' ')}\n${stderr}`));
      else resolve();
    });
  });
}

/** Builds the ImageMagick arguments that take the raw original to a
 * WxH-exact, center-cropped PNG, with a circular mask onto the paper colour
 * for tokens. Returns an argv array for `magick <src> ...args... <dest>`. */
function magickArgs(src, dest, w, h, mask) {
  const args = [src, '-resize', `${w}x${h}^`, '-gravity', 'center', '-extent', `${w}x${h}`, '-strip'];
  if (mask === 'circle') {
    // Draw a white-filled circle on black, use it as an alpha mask, then
    // flatten onto the paper colour so there is no transparency in the
    // shipped file (AVIF alpha costs bytes the budget does not have).
    args.push(
      '(',
      '+clone',
      '-alpha',
      'extract',
      '(',
      '-size',
      `${w}x${h}`,
      'xc:black',
      '-fill',
      'white',
      '-draw',
      `circle ${w / 2},${h / 2} ${w / 2},0`,
      ')',
      '-compose',
      'multiply',
      '-composite',
      ')',
      '-alpha',
      'off',
      '-compose',
      'copy_opacity',
      '-composite',
      '-background',
      PAPER_HEX,
      '-flatten',
    );
  }
  args.push(dest);
  return args;
}

async function encodeAvif(pngPath, outPath, budgetBytes, qualities) {
  await mkdir(path.dirname(outPath), { recursive: true });
  let lastSize = Infinity;
  for (const q of qualities) {
    await run('avifenc', ['-q', String(q), '--speed', '6', pngPath, outPath]);
    const { size } = await stat(outPath);
    lastSize = size;
    if (size <= budgetBytes) return { bytes: size, quality: q, overBudget: false };
  }
  return { bytes: lastSize, quality: qualities[qualities.length - 1], overBudget: true };
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const kind = idKind(args.id);
  const slug = idSlug(args.id);
  const spec = KIND_SPEC[kind];
  if (!spec) die(`no KIND_SPEC for kind "${kind}"`);

  await stat(args.src).catch(() => die(`no such file: ${args.src}`));

  if (!args.out) {
    // The real (shipping) path processes "an approved original" (docs/art/
    // bitmaps.md): apply-review.mjs is what sets status to "approved",
    // from a reviewer's decision on the contact sheet. --out (scratch)
    // skips this check, which is how the pipeline is proved end to end
    // without approving or shipping anything.
    const manifestCheck = await loadManifest();
    const assetCheck = manifestCheck.assets[args.id];
    if (!assetCheck || assetCheck.status !== 'approved') {
      die(
        `${args.id} is not "approved" in art/manifest.json yet; run tools/art/apply-review.mjs first, ` +
          `or pass --out DIR to process into a scratch directory without approving`,
      );
    }
  }

  // The crop scratch dir is always a true OS temp dir, never a sibling of
  // --src: --src can point inside the repo (e.g. an already-copied
  // original), and this must never leave untracked files there.
  const scratchRoot = args.out || path.join(os.tmpdir(), `ajayeb-art-process-${slug}-${Date.now()}`);
  await mkdir(scratchRoot, { recursive: true });

  const shippedDir = args.out ? path.join(args.out, spec.outDir) : path.join(shippedRoot, spec.outDir);
  const bytes = {};

  for (const size of spec.sizes) {
    const fileName = shippedFileName(kind, slug, size);
    const croppedPng = path.join(scratchRoot, `_crop-${fileName}.png`);
    await run('magick', magickArgs(args.src, croppedPng, size.w, size.h, spec.mask));

    const outPath = path.join(shippedDir, fileName);
    const result = await encodeAvif(croppedPng, outPath, size.budgetBytes, args.qualities ?? [55, 45, 35]);
    bytes[fileName] = result.bytes;

    const kb = (result.bytes / 1024).toFixed(1);
    const budgetKb = (size.budgetBytes / 1024).toFixed(0);
    const flag = result.overBudget ? ' OVER BUDGET' : '';
    process.stdout.write(`process.mjs: ${fileName} ${size.w}x${size.h} q${result.quality} -> ${kb} KB (budget ${budgetKb} KB)${flag}\n`);
  }

  if (!args.out) {
    const originalDest = path.join(originalsDir, kind, `${slug}.png`);
    await mkdir(path.dirname(originalDest), { recursive: true });
    await copyFile(args.src, originalDest);
    process.stdout.write(`process.mjs: copied original to ${path.relative(process.cwd(), originalDest)}\n`);
  }

  if (args.out) {
    // Scratch run (the M9.B0 pilot proves the pipeline this way): bytes are
    // reported above, but the manifest is left untouched, so nothing looks
    // approved or shipped from this run.
    process.stdout.write('process.mjs: scratch output only; art/manifest.json left unchanged\n');
  } else {
    const manifest = await loadManifest();
    const asset = manifest.assets[args.id];
    if (asset) {
      asset.bytes = { ...asset.bytes, ...bytes };
      await saveManifest(manifest);
    }
    process.stdout.write('process.mjs: shipped and recorded in art/manifest.json\n');
  }
}

main().catch((err) => die(err.stack || String(err)));
