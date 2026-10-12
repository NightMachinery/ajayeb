// tools/art/lib.mjs
//
// Shared, dependency-free helpers for the bitmap art pipeline
// (manifest.mjs, generate.mjs, process.mjs, sheet.mjs, apply-review.mjs,
// index.mjs). Node 24, ESM, standard library only (fs/promises, path, os,
// child_process, url).
//
// See docs/art/bitmaps.md for how the pipeline fits together and
// art/style.md for the prompts this reads into.

import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { readFileSync, existsSync } from 'node:fs';
import { spawn } from 'node:child_process';
import path from 'node:path';
import os from 'node:os';
import { fileURLToPath } from 'node:url';

// --- Repo and run-dir paths ------------------------------------------------

const scriptDir = path.dirname(fileURLToPath(import.meta.url)); // tools/art
export const repoRoot = path.resolve(scriptDir, '..', '..');

/**
 * Private run state directory, outside the repo (AGENTS.md "Private run
 * state"). Same default and override as tools/heavy.py, so both tools agree
 * on where it lives.
 */
export const runDir = process.env.AJAYEB_RUN_DIR || path.join(os.homedir(), 'tmp', 'agents', 'ajayeb');

export const artWorkDir = path.join(runDir, 'art-work');
export const rawDir = path.join(artWorkDir, 'raw');
export const sheetsDir = path.join(artWorkDir, 'sheets');

/**
 * art/manifest.json, unless AJAYEB_ART_MANIFEST names another file, or a
 * script's --manifest flag calls setManifestPath(). A scratch copy is how
 * apply-review.mjs, process.mjs and index.mjs are proved without touching
 * the committed manifest. A live binding: importers see the change.
 */
export let manifestPath = process.env.AJAYEB_ART_MANIFEST
  ? path.resolve(process.env.AJAYEB_ART_MANIFEST)
  : path.join(repoRoot, 'art', 'manifest.json');

export function setManifestPath(p) {
  manifestPath = path.resolve(p);
}

/** A path for messages: repo-relative inside the repo, `~/...` under the
 * home directory, else absolute. */
export function displayPath(p) {
  const abs = path.resolve(p);
  const rel = path.relative(repoRoot, abs);
  if (!rel.startsWith('..') && !path.isAbsolute(rel)) return rel;
  const home = os.homedir();
  if (abs === home || abs.startsWith(home + path.sep)) return `~${abs.slice(home.length)}`;
  return abs;
}

/** The manifest path for messages. */
export function manifestLabel() {
  return displayPath(manifestPath);
}
export const subjectsPath = path.join(repoRoot, 'art', 'subjects.json');
export const styleDocPath = path.join(repoRoot, 'art', 'style.md');
export const originalsDir = path.join(repoRoot, 'art', 'originals');
export const refsDir = path.join(repoRoot, 'art', 'refs');
export const shippedRoot = path.join(repoRoot, 'web', 'static', 'art');
export const availableGenPath = path.join(repoRoot, 'web', 'src', 'lib', 'art', 'available.gen.ts');
export const catalogGenPath = path.join(repoRoot, 'web', 'src', 'lib', 'wire', 'catalog.gen.ts');

// --- id helpers --------------------------------------------------------
// A manifest/subjects id is always "<kind>/<slug>", e.g. "card/lumber-yard"
// or "hero/hero". Shipped and raw paths use the bare slug, grouped under a
// kind-named directory, so a reviewer browsing art-work/raw/card/ sees only
// cards.

export function idKind(id) {
  return id.slice(0, id.indexOf('/'));
}

export function idSlug(id) {
  return id.slice(id.indexOf('/') + 1);
}

export function makeId(kind, slug) {
  return `${kind}/${slug}`;
}

/**
 * Approved originals are committed as lossless WebP (M9.B4; the owner
 * chose it over PNG to keep the public repository about 30% smaller),
 * pixel-identical to the generated PNG, which process.mjs verifies. A
 * `.png` original from before that change is still recognised.
 */
export const ORIGINAL_EXT = 'webp';
const ORIGINAL_EXTS = ['webp', 'png'];

/** Where process.mjs writes an id's original: art/originals/<kind>/<slug>.webp,
 * or the same layout under `root` (a scratch --out directory). */
export function originalPath(id, root = originalsDir) {
  return path.join(root, idKind(id), `${idSlug(id)}.${ORIGINAL_EXT}`);
}

/** The committed original for an id, .webp first, else a legacy .png, else null. */
export function findOriginal(id) {
  for (const ext of ORIGINAL_EXTS) {
    const candidate = path.join(originalsDir, idKind(id), `${idSlug(id)}.${ext}`);
    if (existsSync(candidate)) return candidate;
  }
  return null;
}

/** The raw attempt file for an id: art-work/raw/<kind>/<slug>/<n>.png. */
export function rawAttemptPath(id, attempt) {
  return path.join(rawDir, idKind(id), idSlug(id), `${attempt}.png`);
}

/**
 * Which attempt of an approved asset is the approved one: the recorded
 * `approvedAttempt` (apply-review.mjs sets it on every approval, M9.B4),
 * else the latest `attempt` for a manifest written before that field
 * existed. Null when the asset is not approved.
 */
export function approvedAttemptOf(asset) {
  if (!asset || asset.status !== 'approved') return null;
  return asset.approvedAttempt || asset.attempt || null;
}

// --- Catalog (cards, wonders, tokens) ------------------------------------
//
// catalog.gen.ts is generated TypeScript, not JSON, but each entry is
// written on its own line as `'slug': { ...valid JSON... },`, so it can be
// read without a TS/JS parser: find the three `export const X: ... = { ... }`
// blocks and JSON.parse each line's object literal.

function extractBlock(src, constName) {
  const marker = `export const ${constName}`;
  const start = src.indexOf(marker);
  if (start === -1) throw new Error(`lib.mjs: catalog.gen.ts has no "${marker}"`);
  const braceStart = src.indexOf('{', start);
  const end = src.indexOf('\n};', braceStart);
  if (braceStart === -1 || end === -1) {
    throw new Error(`lib.mjs: could not find the body of "${marker}" in catalog.gen.ts`);
  }
  return src.slice(braceStart + 1, end);
}

const ENTRY_LINE = /^\s*'([^']+)':\s*(\{.*\}),?\s*$/;

function parseEntries(block) {
  const out = [];
  for (const line of block.split('\n')) {
    if (!line.trim()) continue;
    const m = ENTRY_LINE.exec(line);
    if (!m) continue; // blank lines, stray braces
    const [, slug, json] = m;
    out.push({ slug, data: JSON.parse(json) });
  }
  return out;
}

/**
 * Reads web/src/lib/wire/catalog.gen.ts (checked in, generated by
 * `make gen`) and returns { cards, wonders, tokens }, each an array of
 * { slug, data } in file order. This is the "generated web catalog" the
 * slice brief names as an id source, and it needs no Go build.
 */
export async function readCatalog() {
  const src = await readFile(catalogGenPath, 'utf8');
  return {
    cards: parseEntries(extractBlock(src, 'cards')),
    wonders: parseEntries(extractBlock(src, 'wonders')),
    tokens: parseEntries(extractBlock(src, 'tokens')),
  };
}

// --- Non-catalog assets --------------------------------------------------
//
// Backs, textures, the hero, the AI avatars, the victory scenes and the app
// icon are not in the engine catalog, so their ids, output paths and
// generation sizes are fixed here.

export const STATIC_ASSETS = [
  { id: 'back/age1', kind: 'back', slug: 'age1' },
  { id: 'back/age2', kind: 'back', slug: 'age2' },
  { id: 'back/age3', kind: 'back', slug: 'age3' },
  { id: 'back/guild', kind: 'back', slug: 'guild' },
  { id: 'texture/paper', kind: 'texture', slug: 'paper' },
  { id: 'texture/night', kind: 'texture', slug: 'night' },
  { id: 'hero/hero', kind: 'hero', slug: 'hero' },
  { id: 'avatar/apprentice', kind: 'avatar', slug: 'apprentice' },
  { id: 'avatar/scholar', kind: 'avatar', slug: 'scholar' },
  { id: 'avatar/master', kind: 'avatar', slug: 'master' },
  { id: 'victory/military', kind: 'victory', slug: 'military' },
  { id: 'victory/science', kind: 'victory', slug: 'science' },
  { id: 'victory/civilian', kind: 'victory', slug: 'civilian' },
  { id: 'icon/app', kind: 'icon', slug: 'app' },
];

// --- Per-kind generation and shipping spec --------------------------------
//
// gen: the size passed to codex image_gen (always landscape 1536x1024,
//   except square kinds, per docs/art/bitmaps.md step 8 of the UX design).
// crop: how process.mjs crops the approved original before resizing.
// outDir: the directory under web/static/art/<kind> shipped files land in.
// mask: 'circle' for progress tokens (onto the paper colour), else none.
// sizes: shipped AVIF outputs, each { suffix, w, h, budgetBytes }. A bare
//   suffix '' means "<slug>.avif"; otherwise "<slug>@<suffix>.avif" to
//   match the asset table ("cards/<id>@thumb.avif"). A size added later
//   reaches every approved asset with `process.mjs --batch`, which writes
//   only the files that are missing.
// displayCss: the largest width, in CSS px, at which the app draws the
//   kind's art, measured in a headless browser over the screens that show
//   it (M10.AD1; see docs/art/bitmaps.md "High-DPI sizes"). index.mjs
//   checks it against the widest shipped size at a device pixel ratio of 3
//   (HIDPI_DPR): a kind needs a wider size once that width is more than
//   HIDPI_SLACK times its widest one. null: not measured.

export const KIND_SPEC = {
  card: {
    gen: '1536x1024',
    crop: '4:3',
    outDir: 'cards',
    mask: 'none',
    sizes: [
      { suffix: '', w: 480, h: 360, budgetBytes: 35 * 1024 },
      { suffix: 'thumb', w: 240, h: 180, budgetBytes: 10 * 1024 },
    ],
    // The card sheet's full-tier card (140 px wide, the art drawn 142 px
    // by its cover crop); the board's cards are smaller.
    displayCss: 142,
  },
  wonder: {
    gen: '1536x1024',
    crop: '3:2',
    outDir: 'wonders',
    mask: 'none',
    sizes: [
      { suffix: '', w: 960, h: 640, budgetBytes: 70 * 1024 },
      { suffix: 'thumb', w: 480, h: 320, budgetBytes: 20 * 1024 },
      // High-DPI (M10.AD1): a large wonder tile is half the viewport below
      // 64rem, up to 486 CSS px on a 1020 px tablet, so 1458 device px at
      // DPR 3. The budget keeps the 960 size's 70 KB per 614,400 pixels.
      { suffix: '1440', w: 1440, h: 960, budgetBytes: 158 * 1024 },
    ],
    displayCss: 486,
  },
  token: {
    gen: '1024x1024',
    crop: '1:1',
    outDir: 'tokens',
    mask: 'circle',
    sizes: [
      { suffix: '', w: 256, h: 256, budgetBytes: 14 * 1024 },
      { suffix: 'thumb', w: 96, h: 96, budgetBytes: 5 * 1024 },
    ],
    // The token sheet's medallion: the board's 36 px one, scaled 1.25.
    displayCss: 45,
  },
  back: {
    gen: '1024x1536',
    crop: '1:1.545',
    outDir: 'backs',
    mask: 'none',
    sizes: [{ suffix: '', w: 320, h: 495, budgetBytes: 30 * 1024 }],
    // A face-down card on a 2560 px desktop board.
    displayCss: 132,
  },
  texture: {
    gen: '1024x1024',
    crop: 'tile',
    outDir: 'ui',
    mask: 'none',
    sizes: [{ suffix: '', w: 512, h: 512, budgetBytes: 60 * 1024 }],
    displayCss: null,
  },
  hero: {
    gen: '1536x1024',
    crop: '16:9',
    outDir: 'ui',
    mask: 'none',
    // Named "hero", not "hero/hero.avif", because it is the one hero image.
    // The 800-wide variant is for phones (the asset table's hero-800).
    sizes: [
      { suffix: '', w: 1600, h: 900, budgetBytes: 120 * 1024, nameOverride: 'hero' },
      { suffix: '800', w: 800, h: 450, budgetBytes: 60 * 1024, nameOverride: 'hero' },
    ],
    // Home's hero above 64rem (28rem).
    displayCss: 448,
  },
  avatar: {
    gen: '1024x1024',
    crop: '1:1',
    outDir: 'avatars',
    mask: 'none',
    sizes: [{ suffix: '', w: 192, h: 192, budgetBytes: 12 * 1024 }],
    displayCss: null,
  },
  victory: {
    gen: '1536x1024',
    crop: '16:9',
    outDir: 'victory',
    mask: 'none',
    sizes: [{ suffix: '', w: 960, h: 540, budgetBytes: 60 * 1024 }],
    // The result screen's scene on a desktop.
    displayCss: 158,
  },
  icon: {
    gen: '1024x1024',
    crop: '1:1',
    outDir: 'icon',
    mask: 'none',
    sizes: [
      { suffix: '512', w: 512, h: 512, budgetBytes: 60 * 1024 },
      { suffix: '192', w: 192, h: 192, budgetBytes: 20 * 1024 },
      { suffix: '180', w: 180, h: 180, budgetBytes: 20 * 1024 },
      { suffix: '32', w: 32, h: 32, budgetBytes: 4 * 1024 },
    ],
    displayCss: null,
  },
};

/** The device pixel ratio the high-DPI check plans for (a phone at DPR 3). */
export const HIDPI_DPR = 3;
/** How far a kind's widest shipped size may fall short of its largest
 *  display at HIDPI_DPR before it needs a wider one: up to 1.25 times
 *  upscaled still looks sharp enough on a dense screen. */
export const HIDPI_SLACK = 1.25;

/** The high-DPI check of one kind: the device pixels its largest display
 *  needs at HIDPI_DPR, its widest shipped size, and whether that is enough. */
export function hiDpiCheck(kind) {
  const spec = KIND_SPEC[kind];
  const widest = Math.max(...spec.sizes.map((s) => s.w));
  if (spec.displayCss == null) return { kind, displayCss: null, needed: null, widest, ok: true };
  const needed = spec.displayCss * HIDPI_DPR;
  return { kind, displayCss: spec.displayCss, needed, widest, ok: needed <= widest * HIDPI_SLACK };
}

export function shippedFileName(kind, slug, size) {
  const base = size.nameOverride || slug;
  return size.suffix ? `${base}@${size.suffix}.avif` : `${base}.avif`;
}

// --- JSON file helpers -----------------------------------------------------

export async function readJson(filePath, fallback = undefined) {
  try {
    return JSON.parse(await readFile(filePath, 'utf8'));
  } catch (err) {
    if (err.code === 'ENOENT' && fallback !== undefined) return fallback;
    throw err;
  }
}

export async function writeJson(filePath, data) {
  await mkdir(path.dirname(filePath), { recursive: true });
  await writeFile(filePath, JSON.stringify(data, null, 2) + '\n', 'utf8');
}

export async function loadSubjects() {
  return readJson(subjectsPath, {});
}

export async function loadManifest() {
  return readJson(manifestPath, { assets: {} });
}

export async function saveManifest(manifest) {
  await writeJson(manifestPath, manifest);
}

// --- Process helpers ---------------------------------------------------

export function die(message, code = 1) {
  process.stderr.write(`${path.basename(process.argv[1])}: ${message}\n`);
  process.exit(code);
}

/**
 * Prints a script's usage, which is its leading comment block (every line
 * after the shebang up to the first line that is not a `//` comment), and
 * exits 0. Each script answers --help with this, so the usage text lives
 * in one place, next to the code it describes.
 */
export function printUsage(metaUrl) {
  const src = readFileSync(fileURLToPath(metaUrl), 'utf8').split('\n');
  const out = [];
  for (const line of src.slice(src[0]?.startsWith('#!') ? 1 : 0)) {
    if (!line.startsWith('//')) break;
    out.push(line.replace(/^\/\/ ?/, ''));
  }
  process.stdout.write(out.join('\n').trim() + '\n');
  process.exit(0);
}

// --- Raw-pixel helpers -----------------------------------------------------
//
// Shared by process.mjs (the token rim mask, the paper-tone mapping) and
// gate.mjs (the edge-band full-bleed check). All of them need the same
// "shipping crop" as plain RGB bytes to do per-pixel math in JS rather than
// chaining several ImageMagick composite calls; ImageMagick itself is only
// used to resize/crop/extent and to convert raw bytes back to a PNG.

// The paper colour the app's own card/token frames use (--paper /
// --card-paper in web/src/lib/styles/tokens.css, ivory
// oklch(0.96 0.025 85) ~= #f4ecdc). process.mjs maps every processed
// image's own paper tone onto this exact colour, so a painting's ivory
// ground never shows a visible tone step against the surrounding frame.
export const PAPER_HEX = '#f4ecdc';

export function hexToRgb(hex) {
  const h = hex.replace('#', '');
  return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
}

function runCapture(cmd, args, input) {
  return new Promise((resolve, reject) => {
    const child = spawn(cmd, args, { stdio: ['pipe', 'pipe', 'pipe'] });
    const chunks = [];
    let stderr = '';
    child.stdout.on('data', (d) => chunks.push(d));
    child.stderr.on('data', (d) => (stderr += d));
    child.on('error', reject);
    child.on('close', (code) => {
      if (code !== 0) reject(new Error(`${cmd} ${args.join(' ')}\n${stderr}`));
      else resolve(Buffer.concat(chunks));
    });
    if (input) child.stdin.end(input);
    else child.stdin.end();
  });
}

/**
 * Crops `src` to an exact `w`x`h`, the same "fill" crop process.mjs ships
 * with (`-resize WxH^ -gravity center -extent WxH`), with no mask and no
 * other post-processing, and returns the raw RGB bytes (`w*h*3`, no alpha,
 * row-major, 8-bit) plus `{ w, h }`. This is "the shipping crop" every
 * objective pixel check in this pipeline runs on.
 */
export async function cropToRawRgb(src, w, h) {
  const args = [src, '-resize', `${w}x${h}^`, '-gravity', 'center', '-extent', `${w}x${h}`, '-strip', 'rgb:-'];
  const buf = await runCapture('magick', args);
  if (buf.length !== w * h * 3) {
    throw new Error(`cropToRawRgb: expected ${w * h * 3} bytes for ${w}x${h}, got ${buf.length} (${src})`);
  }
  return { buf, w, h };
}

/**
 * Every stored pixel value of an image file as raw 16-bit RGBA, including
 * the colour under fully transparent pixels (which `magick compare`
 * ignores) and an opaque alpha for an image without one. Two files with
 * equal bytes here hold exactly the same pixels; process.mjs uses it to
 * prove a lossless WebP original matches its source.
 */
export async function rawRgba16(file) {
  return runCapture('magick', [file, '-depth', '16', 'rgba:-']);
}

/** Encodes raw RGB bytes (as `cropToRawRgb` returns) back to a PNG file. */
export async function rawRgbToPng(buf, w, h, outPath) {
  await mkdir(path.dirname(outPath), { recursive: true });
  const pngBuf = await runCapture('magick', ['-size', `${w}x${h}`, '-depth', '8', 'rgb:-', 'png:-'], buf);
  await writeFile(outPath, pngBuf);
}

/**
 * The image's own paper colour: the per-channel median of small patches at
 * each of the four corners. A median (not a mean) so a stray dark ink line
 * clipping a corner cannot pull the estimate off the actual paper tone.
 */
export function cornerPaperColor(buf, w, h, patch = 8) {
  const px = Math.min(patch, Math.floor(w / 2), Math.floor(h / 2));
  const corners = [
    [0, 0],
    [w - px, 0],
    [0, h - px],
    [w - px, h - px],
  ];
  const rs = [];
  const gs = [];
  const bs = [];
  for (const [cx, cy] of corners) {
    for (let y = cy; y < cy + px; y++) {
      for (let x = cx; x < cx + px; x++) {
        const idx = (y * w + x) * 3;
        rs.push(buf[idx]);
        gs.push(buf[idx + 1]);
        bs.push(buf[idx + 2]);
      }
    }
  }
  const median = (arr) => {
    const sorted = [...arr].sort((a, b) => a - b);
    return sorted[Math.floor(sorted.length / 2)];
  };
  return [median(rs), median(gs), median(bs)];
}

/** The pixel size of an image file, via `magick identify`, as "WxH". Used
 * to verify a Codex `image_gen` attempt actually came back the size it was
 * asked for (M9.B3: the pilot saw a reference-image run come back
 * 1254x1254 against a requested 1024x1024). */
export async function imageSize(file) {
  const buf = await runCapture('magick', ['identify', '-format', '%wx%h', file]);
  const s = buf.toString('utf8').trim();
  const m = /^(\d+)x(\d+)$/.exec(s);
  if (!m) throw new Error(`imageSize: could not parse "magick identify" output "${s}" for ${file}`);
  return { w: Number(m[1]), h: Number(m[2]) };
}

export function colorDistance(a, b) {
  const dr = a[0] - b[0];
  const dg = a[1] - b[1];
  const db = a[2] - b[2];
  return Math.sqrt(dr * dr + dg * dg + db * db);
}

// --- Perceptual colour difference ---------------------------------------
//
// gate.mjs's absolute paper check asks "is this image's paper close enough
// to PAPER_HEX?", which is a question about how different two colours look,
// so it uses CIEDE2000 (the CIE's current colour-difference formula) on
// CIELAB (D65 white) rather than the straight-line RGB distance above. RGB
// distance weighs a step in blue the same as a step in green and treats
// lightness and saturation alike; CIEDE2000 corrects for both, so one
// threshold means roughly the same visible difference anywhere in the warm
// ivory-to-gold range the generated papers fall in.

function srgbToLinear(c) {
  const v = c / 255;
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

/** An 8-bit sRGB triple to CIELAB (D65), as [L, a, b]. */
export function rgbToLab([r8, g8, b8]) {
  const r = srgbToLinear(r8);
  const g = srgbToLinear(g8);
  const b = srgbToLinear(b8);
  const x = (0.4124564 * r + 0.3575761 * g + 0.1804375 * b) / 0.95047;
  const y = 0.2126729 * r + 0.7151522 * g + 0.072175 * b;
  const z = (0.0193339 * r + 0.119192 * g + 0.9503041 * b) / 1.08883;
  const f = (t) => (t > 216 / 24389 ? Math.cbrt(t) : ((24389 / 27) * t + 16) / 116);
  const fx = f(x);
  const fy = f(y);
  const fz = f(z);
  return [116 * fy - 16, 500 * (fx - fy), 200 * (fy - fz)];
}

/**
 * CIEDE2000 colour difference between two 8-bit sRGB triples (kL = kC =
 * kH = 1). About 1 is a just-noticeable difference; the pilot's papers sit
 * between 2 and 28 from PAPER_HEX (see gate.mjs for the calibration).
 */
export function deltaE2000(rgb1, rgb2) {
  return deltaE2000Lab(rgbToLab(rgb1), rgbToLab(rgb2));
}

/** CIEDE2000 between two CIELAB triples (the formula itself; Sharma, Wu
 * and Dalal 2005, whose published test pairs it reproduces). */
export function deltaE2000Lab([L1, a1, b1], [L2, a2, b2]) {
  const rad = Math.PI / 180;
  const C1 = Math.hypot(a1, b1);
  const C2 = Math.hypot(a2, b2);
  const Cbar7 = ((C1 + C2) / 2) ** 7;
  const G = 0.5 * (1 - Math.sqrt(Cbar7 / (Cbar7 + 25 ** 7)));
  const ap1 = (1 + G) * a1;
  const ap2 = (1 + G) * a2;
  const Cp1 = Math.hypot(ap1, b1);
  const Cp2 = Math.hypot(ap2, b2);
  const hue = (x, y) => {
    if (x === 0 && y === 0) return 0;
    const t = Math.atan2(y, x) / rad;
    return t < 0 ? t + 360 : t;
  };
  const hp1 = hue(ap1, b1);
  const hp2 = hue(ap2, b2);
  const dL = L2 - L1;
  const dC = Cp2 - Cp1;
  let dh = 0;
  if (Cp1 * Cp2 !== 0) {
    dh = hp2 - hp1;
    if (dh > 180) dh -= 360;
    else if (dh < -180) dh += 360;
  }
  const dH = 2 * Math.sqrt(Cp1 * Cp2) * Math.sin((dh * rad) / 2);
  const Lbar = (L1 + L2) / 2;
  const Cpbar = (Cp1 + Cp2) / 2;
  let hbar = hp1 + hp2;
  if (Cp1 * Cp2 !== 0) {
    if (Math.abs(hp1 - hp2) > 180) hbar += hbar < 360 ? 360 : -360;
    hbar /= 2;
  }
  const T =
    1 -
    0.17 * Math.cos((hbar - 30) * rad) +
    0.24 * Math.cos(2 * hbar * rad) +
    0.32 * Math.cos((3 * hbar + 6) * rad) -
    0.2 * Math.cos((4 * hbar - 63) * rad);
  const dTheta = 30 * Math.exp(-(((hbar - 275) / 25) ** 2));
  const Cpbar7 = Cpbar ** 7;
  const RC = 2 * Math.sqrt(Cpbar7 / (Cpbar7 + 25 ** 7));
  const SL = 1 + (0.015 * (Lbar - 50) ** 2) / Math.sqrt(20 + (Lbar - 50) ** 2);
  const SC = 1 + 0.045 * Cpbar;
  const SH = 1 + 0.015 * Cpbar * T;
  const RT = -Math.sin(2 * dTheta * rad) * RC;
  const tL = dL / SL;
  const tC = dC / SC;
  const tH = dH / SH;
  return Math.sqrt(tL * tL + tC * tC + tH * tH + RT * tC * tH);
}
