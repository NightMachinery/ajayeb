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
//      The ids in OWN_GROUND skip this step: their ground is meant to stay
//      its own colour (texture/night is lapis, not paper), and they name
//      their own AVIF qualities for step 4.
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
// Before any of that, the source is written as the committed original,
// art/originals/<kind>/<slug>.webp: LOSSLESS WebP (`cwebp -lossless -exact
// -z 9`, or ImageMagick's webp:lossless when cwebp is missing), then
// checked with `magick compare -metric AE` against the source, alpha
// included. Any differing pixel is a hard failure and nothing is shipped
// (M9.B4: the owner chose lossless WebP over PNG to keep the public
// repository about 30% smaller). --src may itself be a .webp original.
//
// The source: with no --src, the approved attempt's raw file,
// art-work/raw/<kind>/<slug>/<approvedAttempt>.png (apply-review.mjs
// records approvedAttempt, which may be earlier than the latest attempt;
// M9.B4). A --src that is a raw attempt of this id must be that same
// attempt. With --out and no approval, the latest attempt is used. The
// script prints which file and attempt it processed.
//
// Usage:
//   node tools/art/process.mjs --id token/agriculture
//   node tools/art/process.mjs --id token/agriculture \
//     --src art/originals/token/agriculture.png
//   node tools/art/process.mjs --id card/lumber-yard \
//     --src ~/tmp/agents/ajayeb/art-work/raw/card/lumber-yard/1.png \
//     --out /scratch/dir            # write AVIFs there instead of
//                                    # web/static/art, and skip the
//                                    # "copy to art/originals" step
//
// Flags:
//   --id ID          the asset, "<kind>/<slug>" (required)
//   --src PATH       the original to process (default: the approved
//                    attempt's raw file, see above)
//   --out DIR        scratch run: write everything under DIR, skip the
//                    approval check, touch nothing committed
//   --game NAME      the game whose asset this is (default duel); another
//                    game's original goes to art/originals/<game>/ and its
//                    files to web/static/art/<game>/ (lib.mjs "Games"), so
//                    it never replaces a Duel file of the same slug
//   --manifest PATH  read (and, without --out, update) this manifest
//                    instead of the game's own (art/manifest.json for Duel)
//   --help           print this usage and exit
//
// The app icon PNGs (the one exception to "shipped bitmaps are AVIF only",
// because iOS's apple-touch-icon and the web app manifest need PNG):
//   node tools/art/process.mjs --app-icons [--out DIR]
// reads the committed original art/originals/icon/app.webp (icon/app must
// be approved), crops and paper-maps it exactly as the AVIF sizes are, and
// writes web/static/icons/apple-touch-icon.png (180 px), icon-192.png and
// icon-512.png (or the same names under DIR). It touches neither the
// manifest nor web/static/art/.
//
// Batch mode, for a size added to KIND_SPEC after the art shipped (and for
// any other re-derivation of the shipped files from the committed
// originals):
//   node tools/art/process.mjs --batch [--kind wonder] [--ids a,b] [--force] [--out DIR]
// reprocesses every approved asset (or those of --kind, or the --ids)
// from its committed original, art/originals/<kind>/<slug>.webp, with the
// same crop, paper mapping, mask and quality stepping as above. By default
// it writes only the shipped files that do not exist yet, so the files
// already shipped stay byte-identical and only a new size appears;
// --force rewrites every size. It never writes art/originals/ or the
// manifest (index.mjs reads the byte counts from the shipped files), and
// it prints the files it wrote, with their bytes, and the files it kept.
// An approved id with no committed original is reported and skipped.
//
// With --out, nothing is written to art/originals/ or web/static/art/: the
// WebP original goes to DIR/originals/<kind>/<slug>.webp and the AVIFs to
// DIR/<outDir>/. This is how a pipeline change is proved without shipping
// or approving anything. Without --out, the original is written to
// art/originals/<kind>/<slug>.webp and the AVIFs go to web/static/art/.

import { spawn } from 'node:child_process';
import { mkdir, copyFile, rename, rm, stat } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';
import os from 'node:os';

import {
  repoRoot,
  KIND_SPEC,
  rawDir,
  rawAttemptPath,
  approvedAttemptOf,
  displayPath,
  setManifestPath,
  manifestLabel,
  idKind,
  idSlug,
  originalPath,
  findOriginal,
  originalsDir,
  shippedRoot,
  shippedPath,
  setGame,
  game,
  qualifiedId,
  loadManifest,
  saveManifest,
  cropToRawRgb,
  rawRgbToPng,
  cornerPaperColor,
  rawRgba16,
  hexToRgb,
  PAPER_HEX,
  printUsage,
  die,
} from './lib.mjs';

function parseArgs(argv) {
  const args = { qualities: [55, 45, 35] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--help' || a === '-h') printUsage(import.meta.url);
    else if (a === '--id') args.id = argv[++i];
    else if (a === '--src') args.src = argv[++i];
    else if (a === '--out') args.out = argv[++i];
    else if (a === '--manifest') setManifestPath(argv[++i]);
    else if (a === '--game') setGame(argv[++i]);
    else if (a === '--app-icons') args.appIcons = true;
    else if (a === '--batch') args.batch = true;
    else if (a === '--kind') args.kind = argv[++i];
    else if (a === '--ids') args.ids = argv[++i].split(',').filter(Boolean);
    else if (a === '--force') args.force = true;
    else die(`unknown argument "${a}" (see --help)`, 2);
  }
  if (args.batch) {
    if (args.appIcons || args.id || args.src) die('--batch takes no --id, --src or --app-icons', 2);
    if (args.kind && !KIND_SPEC[args.kind]) die(`no KIND_SPEC for kind "${args.kind}"`, 2);
    return args;
  }
  if (args.kind || args.ids || args.force) die('--kind, --ids and --force go with --batch', 2);
  if (args.appIcons) {
    if (args.id || args.src) die('--app-icons takes no --id or --src (it reads the icon/app original)', 2);
    if (game.name !== 'duel') die('--app-icons makes the app icon, which is Duel\'s icon/app; it takes no --game', 2);
    return args;
  }
  if (!args.id) die('usage: process.mjs --id <id> [--src <original>] [--out DIR] [--manifest PATH] (see --help)', 2);
  return args;
}

/** The attempt number when `file` is one of this id's raw attempt files
 * (art-work/raw/<kind>/<slug>/<n>.png), else null. */
function rawAttemptOfFile(id, file) {
  const dir = path.join(rawDir, idKind(id), idSlug(id));
  const abs = path.resolve(file);
  if (path.dirname(abs) !== dir) return null;
  const m = /^(\d+)\.png$/.exec(path.basename(abs));
  return m ? Number(m[1]) : null;
}

/**
 * Settles which file to process and says why. Without --out the asset
 * must be approved, and the file is the approved attempt's raw file (or a
 * --src that is not a different raw attempt, such as its committed
 * original). With --out it is --src, else the approved attempt, else the
 * latest attempt.
 */
function resolveSource(args, asset) {
  const approved = approvedAttemptOf(asset);
  if (!args.out && !approved) {
    // The real (shipping) path processes "an approved original" (docs/art/
    // bitmaps.md): apply-review.mjs is what sets status to "approved",
    // from a reviewer's decision on the contact sheet. --out (scratch)
    // skips this check, which is how the pipeline is proved end to end
    // without approving or shipping anything.
    die(
      `${args.id} is not "approved" in ${manifestLabel()} yet; run tools/art/apply-review.mjs first, ` +
        `or pass --out DIR to process into a scratch directory without approving`,
    );
  }
  if (args.src) {
    const srcAttempt = rawAttemptOfFile(args.id, args.src);
    if (!args.out && srcAttempt !== null && srcAttempt !== approved) {
      die(`${args.id}: the approved attempt is ${approved}, but --src is attempt ${srcAttempt} (${displayPath(args.src)})`);
    }
    const what =
      srcAttempt !== null
        ? `attempt ${srcAttempt}${srcAttempt === approved ? ', the approved one' : ''}`
        : approved
          ? `given with --src; the approved attempt is ${approved}`
          : 'given with --src';
    return { src: args.src, attempt: srcAttempt ?? approved, label: what };
  }
  if (approved) {
    return { src: rawAttemptPath(args.id, approved), attempt: approved, label: `the approved attempt ${approved}` };
  }
  if (!asset?.attempt) die(`${args.id} has no attempt recorded in ${manifestLabel()}; pass --src`);
  return { src: rawAttemptPath(args.id, asset.attempt), attempt: asset.attempt, label: `the latest attempt ${asset.attempt}, not approved` };
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

function runCapture(cmd, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(cmd, args, { stdio: ['ignore', 'pipe', 'pipe'] });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => (stdout += d));
    child.stderr.on('data', (d) => (stderr += d));
    child.on('error', reject);
    child.on('close', (code) => resolve({ code, stdout, stderr }));
  });
}

/** True when `cmd` runs (used to fall back from cwebp to ImageMagick). */
async function hasCommand(cmd) {
  try {
    const r = await runCapture(cmd, ['-version']);
    return r.code === 0;
  } catch {
    return false;
  }
}

/**
 * Writes `src` as a lossless WebP at `dest` and proves it: `magick compare
 * -metric AE` must count 0 differing pixels between the two, over every
 * channel, alpha included, and their raw 16-bit RGBA values must be equal
 * byte for byte (which also covers the colour under transparent pixels
 * and a source deeper than 8 bits). Encodes to a temporary file first and only
 * then moves it into place, so a failed check (or a --src that is `dest`
 * itself) never leaves a bad or half-written original behind. Returns the
 * encoder used and both byte sizes.
 */
async function writeLosslessWebp(src, dest, tmpDir) {
  const tmp = path.join(tmpDir, `_original-${process.pid}-${Date.now()}.webp`);
  let encoder;
  if (await hasCommand('cwebp')) {
    // -exact keeps the RGB values under fully transparent pixels, which
    // lossless mode would otherwise rewrite; -z 9 is the smallest file.
    encoder = 'cwebp -lossless -exact -z 9';
    await run('cwebp', ['-quiet', '-lossless', '-exact', '-z', '9', src, '-o', tmp]);
  } else {
    encoder = 'magick webp:lossless=true';
    await run('magick', [
      src,
      '-define',
      'webp:lossless=true',
      '-define',
      'webp:exact=true',
      '-define',
      'webp:method=6',
      tmp,
    ]);
  }
  // compare exits 0 when identical, 1 when not, 2 on an error, and prints
  // the differing-pixel count on stderr ("0 (0)" for identical images).
  const cmp = await runCapture('magick', ['compare', '-metric', 'AE', '-channel', 'RGBA', src, tmp, 'null:']);
  const differing = Number.parseFloat(cmp.stderr.trim().split(/\s+/)[0]);
  // compare weighs colour by alpha, so it cannot see a change in the RGB
  // stored under a fully transparent pixel; the raw 16-bit RGBA bytes can.
  const sameRaw = differing === 0 && (await rawRgba16(src)).equals(await rawRgba16(tmp));
  if (cmp.code !== 0 || differing !== 0 || !sameRaw) {
    await rm(tmp, { force: true });
    const why =
      differing !== 0 || cmp.code !== 0
        ? `magick compare -metric AE: "${cmp.stderr.trim()}", exit ${cmp.code}`
        : 'magick compare saw 0 differing pixels, but the raw 16-bit RGBA values differ';
    throw new Error(
      `the lossless WebP of ${displayPath(src)} is NOT pixel-identical to it (${why}); no original written, nothing shipped`,
    );
  }
  const srcBytes = (await stat(src)).size;
  const webpBytes = (await stat(tmp)).size;
  await mkdir(path.dirname(dest), { recursive: true });
  try {
    await rename(tmp, dest);
  } catch (err) {
    if (err.code !== 'EXDEV') throw err;
    await copyFile(tmp, dest);
    await rm(tmp, { force: true });
  }
  return { encoder, differing, srcBytes, webpBytes };
}

function mb(bytes) {
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
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

/**
 * The PNG app icons: file name and square size, under web/static/icons/.
 * A maskable icon draws the art at MASKABLE_ART of its size, centred on
 * paper, so the whole roundel sits inside the safe zone (the centred
 * circle of 80% diameter) that every launcher mask keeps. The roundel
 * spans about 96% of the art; outside a circle of MASKABLE_KEEP of the
 * art (as a radius) the grained paper fades into flat PAPER_HEX, so the
 * padding shows no seam.
 */
const APP_ICON_PNGS = [
  { file: 'apple-touch-icon.png', size: 180 },
  { file: 'icon-192.png', size: 192 },
  { file: 'icon-512.png', size: 512 },
  { file: 'icon-maskable-512.png', size: 512, maskable: true },
];
const MASKABLE_ART = 0.8;
const MASKABLE_KEEP = 0.49;
const APP_ICON_ID = 'icon/app';

/**
 * --app-icons: the PNG icons iOS and the web app manifest need, from the
 * approved app icon's committed original, with the same fill crop and
 * paper-tone mapping as its AVIF sizes. The gouache grain makes a
 * truecolour PNG huge (408 KB at 512 px), so each is quantised to a 256
 * colour palette with Floyd-Steinberg dithering (93 KB at 512 px, 12 KB at
 * 180 px, no visible change at icon sizes) and written with no metadata.
 * The maskable one pads the art with PAPER_HEX, the paper it was mapped to.
 */
async function appIcons(args) {
  const manifest = await loadManifest();
  if (!approvedAttemptOf(manifest.assets[APP_ICON_ID])) {
    die(`${APP_ICON_ID} is not "approved" in ${manifestLabel()}; there is no app icon to derive PNGs from`);
  }
  const src = originalPath(APP_ICON_ID);
  await stat(src).catch(() => die(`no committed original at ${displayPath(src)}`));
  const outDir = args.out ? path.resolve(args.out) : path.join(repoRoot, 'web', 'static', 'icons');
  const scratch = path.join(os.tmpdir(), `ajayeb-art-app-icons-${process.pid}-${Date.now()}`);
  await mkdir(scratch, { recursive: true });
  await mkdir(outDir, { recursive: true });
  const targetPaperRgb = hexToRgb(PAPER_HEX);
  try {
    for (const { file, size, maskable } of APP_ICON_PNGS) {
      const art = maskable ? Math.round(size * MASKABLE_ART) : size;
      const { buf, w, h } = await cropToRawRgb(src, art, art);
      mapPaperTone(buf, w, h, cornerPaperColor(buf, w, h), targetPaperRgb);
      const raw = path.join(scratch, file);
      await rawRgbToPng(buf, w, h, raw);
      const outPath = path.join(outDir, file);
      await run('magick', [
        raw,
        ...(maskable ? maskablePadding(art, size) : []),
        '-dither',
        'FloydSteinberg',
        '-colors',
        '256',
        '-strip',
        '-define',
        'png:compression-level=9',
        `PNG8:${outPath}`,
      ]);
      const { size: bytes } = await stat(outPath);
      process.stdout.write(`process.mjs: ${displayPath(outPath)} ${size}x${size} PNG -> ${(bytes / 1024).toFixed(1)} KB\n`);
    }
  } finally {
    await rm(scratch, { recursive: true, force: true });
  }
}

/** magick arguments that pad an art-sized icon onto a size-sized paper square. */
function maskablePadding(art, size) {
  const c = art / 2;
  const keep = `circle ${c},${c} ${c},${c - Math.round(art * MASKABLE_KEEP)}`;
  return [
    '(', '+clone', '-fill', 'black', '-colorize', '100', '-fill', 'white', '-draw', keep, '-blur', '0x2', ')',
    '-alpha', 'off', '-compose', 'CopyOpacity', '-composite',
    '-background', PAPER_HEX, '-compose', 'Over', '-flatten',
    '-gravity', 'center', '-extent', `${size}x${size}`,
  ];
}

/**
 * The ids whose ground is not paper, so the paper-tone mapping (step 2 at
 * the top) leaves them as painted, each with the AVIF qualities to step
 * through instead of the default. The night table tile is a deep lapis by
 * design: mapped, its lapis corners counted as "paper" and the whole tile
 * turned into a pale ivory grain that the dark theme had to multiply back
 * down. Unmapped, its faint fibres sit so close to the ground that q55
 * smooths most of them away (3.5 KB, half the original's grain), while q75
 * keeps them for 13 KB (M10.F2).
 */
const OWN_GROUND = new Map([[qualifiedId('duel', 'texture/night'), [75, 65, 55]]]);

/**
 * Writes one shipped size of the original of `id`: the fill crop to the
 * size's exact pixels, the paper-tone mapping (unless `id` is in
 * OWN_GROUND), the token rim mask, then AVIF at the first quality of
 * `qualities` (or the id's own, from OWN_GROUND) that meets the size's byte
 * budget (see the steps at the top).
 * Returns encodeAvif's { bytes, quality, overBudget }.
 */
async function shipSize(src, id, size, outPath, scratchRoot, qualities = [55, 45, 35]) {
  const spec = KIND_SPEC[idKind(id)];
  const croppedPng = path.join(scratchRoot, `_crop-${path.basename(outPath)}.png`);
  const { buf, w, h } = await cropToRawRgb(src, size.w, size.h);
  const ownPaperRgb = cornerPaperColor(buf, w, h);
  // The rim is detected on the unmapped pixels, against the crop's own
  // paper colour: after mapPaperTone the paper sits at PAPER_HEX, so a
  // deep generated paper would no longer match ownPaperRgb and the rim
  // scan would stop at the first pixel, leaving the ivory ring in place.
  const radius = spec.mask === 'circle' ? detectTokenRadius(buf, w, h, ownPaperRgb) : 0;
  const targetPaperRgb = hexToRgb(PAPER_HEX);
  const ownGround = OWN_GROUND.get(qualifiedId(game.name, id));
  if (!ownGround) mapPaperTone(buf, w, h, ownPaperRgb, targetPaperRgb);
  if (spec.mask === 'circle') applyCircleMask(buf, w, h, radius, targetPaperRgb);
  await rawRgbToPng(buf, w, h, croppedPng);
  try {
    return await encodeAvif(croppedPng, outPath, size.budgetBytes, ownGround ?? qualities);
  } finally {
    await rm(croppedPng, { force: true });
  }
}

/** "480x360 q55 -> 30.1 KB (budget 35 KB)", flagged when over budget. */
function sizeReport(size, result) {
  const kb = (result.bytes / 1024).toFixed(1);
  const budgetKb = (size.budgetBytes / 1024).toFixed(0);
  const flag = result.overBudget ? ' OVER BUDGET' : '';
  return `${size.w}x${size.h} q${result.quality} -> ${kb} KB (budget ${budgetKb} KB)${flag}`;
}

/**
 * --batch: re-derives shipped sizes from the committed originals (see the
 * usage above). Only missing files are written unless --force.
 */
async function batch(args) {
  const manifest = await loadManifest();
  const ids = Object.keys(manifest.assets)
    .sort()
    .filter((id) => approvedAttemptOf(manifest.assets[id]))
    .filter((id) => !args.kind || idKind(id) === args.kind)
    .filter((id) => !args.ids || args.ids.includes(id));
  for (const id of args.ids ?? []) {
    if (!ids.includes(id)) die(`--ids: ${id} is not an approved asset in ${manifestLabel()}`);
  }
  const outRoot = args.out ? path.resolve(args.out) : shippedRoot;
  const scratchRoot = path.join(os.tmpdir(), `ajayeb-art-batch-${process.pid}-${Date.now()}`);
  await mkdir(scratchRoot, { recursive: true });
  let written = 0;
  let writtenBytes = 0;
  let kept = 0;
  let over = 0;
  const noOriginal = [];
  try {
    for (const id of ids) {
      const kind = idKind(id);
      const spec = KIND_SPEC[kind];
      if (!spec) die(`no KIND_SPEC for kind "${kind}" (${id})`);
      const todo = spec.sizes
        .map((size) => ({ size, outPath: shippedPath(id, size, outRoot) }))
        .filter(({ outPath }) => args.force || !existsSync(outPath));
      kept += spec.sizes.length - todo.length;
      if (todo.length === 0) continue;
      const src = findOriginal(id);
      if (!src) {
        noOriginal.push(id);
        continue;
      }
      for (const { size, outPath } of todo) {
        const result = await shipSize(src, id, size, outPath, scratchRoot, args.qualities);
        written++;
        writtenBytes += result.bytes;
        if (result.overBudget) over++;
        process.stdout.write(`process.mjs: ${displayPath(outPath)} ${sizeReport(size, result)}\n`);
      }
    }
  } finally {
    await rm(scratchRoot, { recursive: true, force: true });
  }
  process.stdout.write(
    `process.mjs: batch: ${ids.length} approved id(s); wrote ${written} file(s), ${(writtenBytes / 1024).toFixed(1)} KB` +
      `${over ? `, ${over} over budget` : ''}; kept ${kept} existing file(s) untouched` +
      `${args.force ? '' : ' (--force rewrites them)'}; ${manifestLabel()} left unchanged\n`,
  );
  if (noOriginal.length) {
    die(`no committed original under ${displayPath(originalsDir)}/ for ${noOriginal.join(', ')}; their missing sizes were not written`);
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.appIcons) return appIcons(args);
  if (args.batch) return batch(args);
  const kind = idKind(args.id);
  const slug = idSlug(args.id);
  const spec = KIND_SPEC[kind];
  if (!spec) die(`no KIND_SPEC for kind "${kind}"`);

  const manifestAtStart = await loadManifest();
  const source = resolveSource(args, manifestAtStart.assets[args.id]);
  args.src = source.src;
  await stat(args.src).catch(() => die(`no such file: ${args.src}`));
  process.stdout.write(`process.mjs: ${args.id}: processing ${displayPath(args.src)} (${source.label})\n`);

  // The crop scratch dir is always a true OS temp dir, never a sibling of
  // --src: --src can point inside the repo (e.g. an already-copied
  // original), and this must never leave untracked files there.
  const scratchRoot = args.out || path.join(os.tmpdir(), `ajayeb-art-process-${slug}-${Date.now()}`);
  await mkdir(scratchRoot, { recursive: true });

  // The original first: if it is not pixel-identical, fail before any
  // shipped file is written.
  const originalDest = args.out ? originalPath(args.id, path.join(args.out, 'originals')) : originalPath(args.id);
  const orig = await writeLosslessWebp(args.src, originalDest, scratchRoot);
  const pct = ((orig.webpBytes / orig.srcBytes) * 100).toFixed(0);
  process.stdout.write(
    `process.mjs: original ${displayPath(originalDest)}: ${mb(orig.webpBytes)} lossless WebP (${orig.encoder}) ` +
      `vs ${mb(orig.srcBytes)} source ${path.extname(args.src).slice(1).toUpperCase()} (${pct}%); ` +
      `magick compare -metric AE: ${orig.differing} differing pixels, raw RGBA identical\n`,
  );

  const outRoot = args.out ?? shippedRoot;
  const bytes = {};

  for (const size of spec.sizes) {
    const outPath = shippedPath(args.id, size, outRoot);
    const fileName = path.basename(outPath);
    const result = await shipSize(args.src, args.id, size, outPath, scratchRoot, args.qualities);
    bytes[fileName] = result.bytes;
    process.stdout.write(`process.mjs: ${fileName} ${sizeReport(size, result)}\n`);
  }

  if (args.out) {
    // Scratch run (the M9.B0 pilot proves the pipeline this way): bytes are
    // reported above, but the manifest is left untouched, so nothing looks
    // approved or shipped from this run.
    process.stdout.write(`process.mjs: scratch output only; ${manifestLabel()} left unchanged\n`);
  } else {
    const manifest = await loadManifest();
    const asset = manifest.assets[args.id];
    if (asset) {
      asset.bytes = { ...asset.bytes, ...bytes };
      await saveManifest(manifest);
    }
    process.stdout.write(`process.mjs: shipped and recorded in ${manifestLabel()}\n`);
  }
}

main().catch((err) => die(err.stack || String(err)));
