#!/usr/bin/env node
// tools/art/process.mjs
//
// Turns one approved (or, with --out, not-yet-approved) original into the
// shipped AVIF sizes from the asset table in art/style.md /
// docs/art/bitmaps.md, and records their byte counts in art/manifest.json.
//
// For each shipped size:
//   1. Crop the original to that exact WxH (`-resize WxH^ -gravity center
//      -extent WxH`, the same "fill" crop every objective check in this
//      pipeline uses) and read it back as raw RGB bytes.
//   2. Detect the image's own paper colour (the corner-median of the crop)
//      and map it onto PAPER_HEX, the app's --card-paper token, with a
//      smooth per-pixel falloff so only paper-like pixels move and the
//      painting's other colours are left alone (M9.B3's "paper tone" fix:
//      a generated gouache ground runs warmer than the CSS token, which
//      used to show as a visible seam against the surrounding card frame).
//   3. For a progress token (`mask: 'circle'`), detect the painted
//      medallion's own rim radius (scanning inward from each edge for the
//      first pixel that isn't paper-coloured) instead of always masking to
//      a circle that touches the square's edges, and feather a soft
//      anti-aliased edge there. A medallion painted with a margin inside
//      the square (M9.B3's pilot tokens) used to leave a visible ivory
//      "halo" ring once masked to the full square; detecting the real rim
//      removes it without needing a second generation.
//   4. Write the result back to a PNG and encode it with `avifenc`,
//      stepping quality down (55 -> 45 -> 35) until the size's byte budget
//      holds, or keeping the lowest quality with a warning.
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
  cropToRawRgb,
  rawRgbToPng,
  cornerPaperColor,
  hexToRgb,
  PAPER_HEX,
  die,
} from './lib.mjs';

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

/**
 * Maps the crop's own paper colour onto `targetRgb`, in place, with a
 * per-pixel falloff: a pixel exactly at the paper colour gets the full
 * shift, a pixel `fadeDist` or further from it (ink, gouache, anything
 * painted) gets none, and pixels in between blend smoothly. This is what
 * keeps "the painting's other colours" from shifting (M9.B3's acceptance:
 * the mapping touches only the ivory ground).
 */
function mapPaperTone(buf, w, h, paperRgb, targetRgb, fadeDist = 60) {
  const delta = [targetRgb[0] - paperRgb[0], targetRgb[1] - paperRgb[1], targetRgb[2] - paperRgb[2]];
  for (let i = 0; i < buf.length; i += 3) {
    const r = buf[i];
    const g = buf[i + 1];
    const b = buf[i + 2];
    const dr = r - paperRgb[0];
    const dg = g - paperRgb[1];
    const db = b - paperRgb[2];
    const dist = Math.sqrt(dr * dr + dg * dg + db * db);
    const weight = Math.max(0, 1 - dist / fadeDist);
    if (weight <= 0) continue;
    buf[i] = Math.min(255, Math.max(0, Math.round(r + delta[0] * weight)));
    buf[i + 1] = Math.min(255, Math.max(0, Math.round(g + delta[1] * weight)));
    buf[i + 2] = Math.min(255, Math.max(0, Math.round(b + delta[2] * weight)));
  }
}

/**
 * Finds the progress token's own painted rim, so the mask can shrink-wrap
 * to it instead of always touching the square's edges. Walks inward from
 * each of the 4 edges along the horizontal/vertical centre lines, looking
 * for the first pixel that is not paper-coloured; the smallest of the 4
 * estimates is the safe radius (whichever direction the margin is
 * narrowest), so no ivory ring from any direction survives the mask.
 * The token template (art/style.md, M9.B4) asks for exactly this case: a
 * crisp gold rim inset by a narrow, flat margin of paper. Falls back to
 * the full `min(w,h)/2` (the old behaviour: mask to the square's edge)
 * when an axis never leaves the paper colour, e.g. a medallion painted
 * edge to edge.
 */
function detectTokenRadius(buf, w, h, paperRgb, distThresh = 30) {
  const cx = (w - 1) / 2;
  const cy = (h - 1) / 2;
  const maxR = Math.min(w, h) / 2;
  const axes = [
    { dx: 1, dy: 0 },
    { dx: -1, dy: 0 },
    { dx: 0, dy: 1 },
    { dx: 0, dy: -1 },
  ];
  const estimates = [];
  for (const { dx, dy } of axes) {
    let r = maxR;
    for (let step = Math.floor(maxR) - 1; step >= 0; step--) {
      const x = Math.round(cx + dx * step);
      const y = Math.round(cy + dy * step);
      if (x < 0 || x >= w || y < 0 || y >= h) continue;
      const idx = (y * w + x) * 3;
      const dist = Math.sqrt(
        (buf[idx] - paperRgb[0]) ** 2 + (buf[idx + 1] - paperRgb[1]) ** 2 + (buf[idx + 2] - paperRgb[2]) ** 2,
      );
      if (dist > distThresh) {
        r = step;
        break;
      }
    }
    estimates.push(r);
  }
  // A floor keeps a noisy detection from crushing the medallion to nothing;
  // half the old default radius is already a very tight margin.
  return Math.max(maxR * 0.5, Math.min(...estimates));
}

/** Flattens every pixel outside `radius` of the crop's centre onto
 * `paperRgb` (no alpha channel, matching the rest of the pipeline), with a
 * 1.5px antialiased edge so the masked circle is not jagged at small sizes
 * (tokens ship as small as 96px). */
function applyCircleMask(buf, w, h, radius, paperRgb) {
  const cx = (w - 1) / 2;
  const cy = (h - 1) / 2;
  const feather = 1.5;
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const dist = Math.sqrt((x - cx) ** 2 + (y - cy) ** 2);
      const t = Math.min(1, Math.max(0, (dist - (radius - feather)) / (feather * 2)));
      if (t <= 0) continue;
      const idx = (y * w + x) * 3;
      buf[idx] = Math.round(buf[idx] * (1 - t) + paperRgb[0] * t);
      buf[idx + 1] = Math.round(buf[idx + 1] * (1 - t) + paperRgb[1] * t);
      buf[idx + 2] = Math.round(buf[idx + 2] * (1 - t) + paperRgb[2] * t);
    }
  }
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
  const targetPaperRgb = hexToRgb(PAPER_HEX);

  for (const size of spec.sizes) {
    const fileName = shippedFileName(kind, slug, size);
    const croppedPng = path.join(scratchRoot, `_crop-${fileName}.png`);

    const { buf, w, h } = await cropToRawRgb(args.src, size.w, size.h);
    const ownPaperRgb = cornerPaperColor(buf, w, h);
    // The rim is detected on the unmapped pixels, against the crop's own
    // paper colour: after mapPaperTone the paper sits at PAPER_HEX, so a
    // deep generated paper would no longer match ownPaperRgb and the rim
    // scan would stop at the first pixel, leaving the ivory ring in place.
    const radius = spec.mask === 'circle' ? detectTokenRadius(buf, w, h, ownPaperRgb) : 0;
    mapPaperTone(buf, w, h, ownPaperRgb, targetPaperRgb);
    if (spec.mask === 'circle') applyCircleMask(buf, w, h, radius, targetPaperRgb);
    await rawRgbToPng(buf, w, h, croppedPng);

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
