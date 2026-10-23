#!/usr/bin/env node
// tools/art/index.mjs
//
// Writes web/src/lib/art/available.gen.ts from art/manifest.json's
// *approved* assets only, so the running app can check whether real art
// exists for an id before trying to load it, and never requests a file
// that was never shipped. An id missing from this module has no approved
// art yet; the app's procedural placeholder (CardPlaceholderArt and
// friends, per AGENTS.md "Assets") draws for it instead.
//
// For each approved id it also prints which attempt was approved (the
// manifest's approvedAttempt, which may be earlier than the latest
// attempt; M9.B4) and its committed original, art/originals/<kind>/
// <slug>.webp (lossless WebP since M9.B4; a legacy .png is recognised
// too), so the log says exactly which image the app now shows. An
// approved id with no original is warned about: process.mjs has not
// shipped it yet.
//
// Every variant path carries its content hash, `?v=` and the first 10 hex
// digits of the shipped file's SHA-256 (the file names stay as they are).
// The Go server caches an /art/ request whose v matches the file for a
// year with no revalidation (webdist.CacheImmutable), so a returning
// browser asks for nothing it already has, and replaced art gets a new URL
// at once. The byte counts come from the shipped files themselves, so a
// size written by `process.mjs --batch` is counted too. An approved id
// whose shipped file is missing is an error: the app would request it.
//
// The module is formatted with the web package's Prettier (its own config,
// so `make lint-web` accepts the file as written), which needs
// `make install`. Same manifest and files in, same bytes out.
//
// It then prints a report: the shipped total, the full idle prefetch plan
// of web/src/lib/art/prefetch.ts per part and in total on a phone (390 px
// wide, DPR 3) and a desktop (1280 px wide, DPR 1), and the high-DPI check
// of each kind (lib.mjs hiDpiCheck), warning when a kind's largest display
// needs a wider size than it ships.
//
// With --game haftgane (lib.mjs "Games") it indexes that game's manifest
// into web/src/lib/art/available-haftgane.gen.ts, with paths under
// /art/haftgane/. Each alias in art/haftgane/aliases.json whose target is
// approved gets the target's own entry, the same content-hashed paths of
// the same files (an alias never copies or re-encodes a painting); an
// alias whose target is not approved yet is left out, so its placeholder
// draws. The prefetch plan report is Duel's only, until another game's
// screens prefetch.
//
// Usage:
//   node tools/art/index.mjs
//   node tools/art/index.mjs --check
//   node tools/art/index.mjs --manifest /scratch/manifest.json --out /scratch/available.gen.ts
//   node tools/art/index.mjs --game haftgane --out /scratch/available-haftgane.gen.ts
//
// Flags:
//   --game NAME      the game to index (default duel)
//   --check          write nothing; exit 1 when the module on disk differs
//                    from what this would write (the art index check)
//   --manifest PATH  read this manifest instead of the game's own (it must
//                    be that game's)
//   --out FILE       write the module here instead of the game's index,
//                    web/src/lib/art/available.gen.ts for Duel (a scratch proof)
//   --quiet          print only the summary lines, not one line per id
//   --help           print this usage and exit

import { spawn } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';

import {
  KIND_SPEC,
  idKind,
  loadManifest,
  loadGameManifest,
  loadAliases,
  availableGenPath,
  approvedAttemptOf,
  findOriginal,
  setManifestPath,
  setGame,
  game,
  gameDirs,
  qualifiedId,
  shippedPath,
  shippedUrlPath,
  manifestLabel,
  displayPath,
  printUsage,
  die,
  repoRoot,
  readCatalog,
  hiDpiCheck,
  HIDPI_DPR,
  HIDPI_SLACK,
} from './lib.mjs';

/** Hex digits of the SHA-256 in each path's ?v= (the server accepts 8 or more). */
const HASH_DIGITS = 10;

function parseArgs(argv) {
  const args = {};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--help' || a === '-h') printUsage(import.meta.url);
    else if (a === '--manifest') setManifestPath(argv[++i]);
    else if (a === '--game') setGame(argv[++i]);
    else if (a === '--out') args.out = path.resolve(argv[++i]);
    else if (a === '--check') args.check = true;
    else if (a === '--quiet') args.quiet = true;
    else die(`unknown argument "${a}"`, 2);
  }
  args.out ??= availableGenPath;
  return args;
}

/** The first HASH_DIGITS hex digits of a file's SHA-256. */
function contentHash(buf) {
  return createHash('sha256').update(buf).digest('hex').slice(0, HASH_DIGITS);
}

/** The index entries of a manifest's approved assets, from the shipped
 *  files of the game `dirs` (lib.mjs gameDirs). */
function buildEntries(manifest, dirs = game) {
  const entries = {};
  const missing = [];
  const ids = Object.keys(manifest.assets).sort();
  for (const id of ids) {
    const asset = manifest.assets[id];
    if (asset.status !== 'approved') continue;
    const kind = idKind(id);
    const spec = KIND_SPEC[kind];
    const variants = {};
    for (const size of spec.sizes) {
      const file = shippedPath(id, size, dirs.shipped);
      if (!existsSync(file)) {
        missing.push(displayPath(file));
        continue;
      }
      const buf = readFileSync(file);
      const suffix = size.suffix || 'default';
      variants[suffix] = {
        path: `${shippedUrlPath(dirs, id, size)}?v=${contentHash(buf)}`,
        width: size.w,
        height: size.h,
        bytes: buf.length,
      };
    }
    entries[id] = { kind, variants };
  }
  if (missing.length) {
    die(
      `${missing.length} shipped file(s) of approved art are missing (run node tools/art/process.mjs --batch):\n  ` +
        missing.join('\n  '),
    );
  }
  return entries;
}

/**
 * The active game's aliases with approved targets, as index entries that
 * are the targets' own (same files, same hashed paths), plus a line per
 * alias for the log. A target still waiting for approval gives no entry.
 */
async function aliasEntries() {
  const entries = {};
  const lines = [];
  const targets = new Map();
  for (const [id, target] of await loadAliases()) {
    if (!targets.has(target.game)) {
      targets.set(target.game, buildEntries(await loadGameManifest(target.game), gameDirs(target.game)));
    }
    const q = qualifiedId(target.game, target.id);
    const entry = targets.get(target.game)[target.id];
    if (entry) entries[id] = entry;
    lines.push(`${id}: alias of ${q}${entry ? '' : ', not approved yet, so no entry (its placeholder draws)'}`);
  }
  return { entries, lines };
}

/** Own and alias entries in one id order (an id is never both). */
function mergeSorted(own, aliases) {
  const out = {};
  for (const id of [...Object.keys(own), ...Object.keys(aliases)].sort()) out[id] = own[id] ?? aliases[id];
  return out;
}

function renderTs(entries) {
  const count = Object.keys(entries).length;
  const body = JSON.stringify(entries, null, 2)
    .split('\n')
    .map((line, i) => (i === 0 ? line : `  ${line}`))
    .join('\n');

  return `${game.name === 'duel' ? duelHeader(count) : gameHeader(count)}
export interface ArtVariant {
  readonly path: string;
  readonly width: number;
  readonly height: number;
  readonly bytes: number;
}

/** variants is keyed by the shipped size's suffix: 'default' for a kind's
 * main size, otherwise the asset table's suffix ('thumb', '1440', '512', …). */
export interface ArtEntry {
  readonly kind: string;
  readonly variants: { readonly [suffix: string]: ArtVariant };
}

export const art: { readonly [id: string]: ArtEntry } = ${body};

export function hasArt(id: string): boolean {
  return id in art;
}

export function artVariant(id: string, suffix: string = 'default'): ArtVariant | undefined {
  return art[id]?.variants[suffix];
}
`;
}

/** The index's header comment for Duel, as it has always been. */
function duelHeader(count) {
  return `// Code generated by tools/art/index.mjs. DO NOT EDIT.
//
// Approved bitmap art only (${count} id${count === 1 ? '' : 's'} right now):
// every id here has a real file under web/static/art/, written by
// tools/art/process.mjs from an original a review round approved
// (art/manifest.json, status "approved"). An id with no entry has no
// approved art yet; the app draws its procedural placeholder instead
// (CardPlaceholderArt and friends never look here first). See
// docs/art/bitmaps.md for the pipeline that fills this in.
//
// Ids are "<kind>/<slug>", e.g. "card/lumber-yard", "wonder/pyramids",
// "token/agriculture", "back/age1", "texture/paper", "hero/hero",
// "avatar/apprentice", "victory/military", "icon/app".
//
// Each path ends in ?v= and the start of the file's SHA-256, so it names
// one exact file: the server caches it for a year, and new art gets a new
// URL. bytes is the shipped file's size.
`;
}

/** The index's header comment for another game. */
function gameHeader(count) {
  const g = game.name;
  return `// Code generated by tools/art/index.mjs --game ${g}. DO NOT EDIT.
//
// ${g}'s approved bitmap art only (${count} id${count === 1 ? '' : 's'} right now): each id
// here is either ${g}'s own art, a real file under web/static/art/${g}/
// written by tools/art/process.mjs --game ${g} from an original a review
// round approved (art/${g}/manifest.json, status "approved"), or an alias
// (art/${g}/aliases.json) that carries the paths of an approved painting it
// reuses, possibly another game's. An id with no entry draws its procedural
// placeholder. See docs/art/bitmaps.md "Games".
//
// Ids are "<kind>/<slug>" in ${g}'s own namespace. Each path ends in ?v= and
// the start of the file's SHA-256, so it names one exact file: the server
// caches it for a year, and new art gets a new URL. bytes is the shipped
// file's size.
`;
}

/**
 * Formats the module with the web package's Prettier, as `make lint-web`
 * checks it: the config is found from the module's real path whatever
 * --out says, so a scratch copy comes out the same.
 */
function prettierFormat(src) {
  const webDir = path.join(repoRoot, 'web');
  const bin = path.join(webDir, 'node_modules', '.bin', 'prettier');
  if (!existsSync(bin)) die(`no Prettier at ${displayPath(bin)}; run make install first`);
  return new Promise((resolve, reject) => {
    const child = spawn(bin, ['--stdin-filepath', path.relative(webDir, path.join(repoRoot, game.indexRel))], {
      cwd: webDir,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    let out = '';
    let err = '';
    child.stdout.on('data', (d) => (out += d));
    child.stderr.on('data', (d) => (err += d));
    child.on('error', reject);
    child.on('close', (code) => (code === 0 ? resolve(out) : reject(new Error(`prettier exited ${code}\n${err}`))));
    child.stdin.end(src);
  });
}

// --- The report ------------------------------------------------------------
//
// A mirror of web/src/lib/art/prefetch.ts's full plan and of the sizes
// strings in lookup.ts, for the byte counts only (prefetch.test.ts checks
// the plan itself). A frame with a sizes string gets the narrowest variant
// at least as wide as its CSS width times the pixel ratio, as a browser
// picks from a srcset; a frame without one gets its fixed variant.

const DEVICES = [
  { name: 'phone (390 px, DPR 3)', vw: 390, dpr: 3 },
  { name: 'desktop (1280 px, DPR 1)', vw: 1280, dpr: 1 },
];

/** The CSS widths of lookup.ts's sizes strings on a device. */
const FRAME_CSS = {
  cardFull: () => 140, // cardArtSizes(FULL_CARD_WIDTH)
  wonderLarge: (d) => (d.vw >= 1024 ? 280 : d.vw / 2), // WONDER_ART_SIZES
  token: () => 36, // tokenArtSizes('md')
};

function variantFor(entry, frame, device) {
  const vs = Object.entries(entry.variants);
  if (frame.fixed) {
    const own = entry.variants[frame.fixed] ?? entry.variants.default;
    return own ?? vs.map(([, v]) => v).sort((a, b) => b.width - a.width)[0];
  }
  const need = FRAME_CSS[frame.sizes](device) * device.dpr;
  const sorted = vs.map(([, v]) => v).sort((a, b) => a.width - b.width);
  return sorted.find((v) => v.width >= need) ?? sorted[sorted.length - 1];
}

async function report(entries, quiet) {
  const shipped = Object.values(entries).flatMap((e) => Object.values(e.variants));
  const total = shipped.reduce((n, v) => n + v.bytes, 0);
  const out = (line) => process.stdout.write(`index.mjs: ${line}\n`);
  out(`shipped: ${shipped.length} files, ${(total / 1024 / 1024).toFixed(2)} MB (per-image budgets only; no total cap)`);
  if (game.name === 'duel') await duelPlanReport(entries, out);
  else out(`prefetch plan: not reported for ${game.name}; only Duel's screens prefetch so far`);

  for (const kind of Object.keys(KIND_SPEC)) {
    const c = hiDpiCheck(kind);
    if (c.displayCss === null) continue;
    const verdict = c.ok
      ? 'enough'
      : `NEEDS A WIDER SIZE (more than ${HIDPI_SLACK} times ${c.widest}); add one to KIND_SPEC and run process.mjs --batch`;
    if (!quiet || !c.ok) {
      out(`high-DPI ${kind}: shown up to ${c.displayCss} CSS px, ${c.needed} px at DPR ${HIDPI_DPR}; widest shipped ${c.widest}: ${verdict}`);
    }
  }
}

/** Duel's full prefetch plan per part, on each device (see above). */
async function duelPlanReport(entries, out) {
  const { cards, wonders, tokens } = await readCatalog();
  const ageCards = (back) => cards.filter((c) => c.data.back === back).map((c) => `card/${c.slug}`);
  const parts = [
    { name: 'Age I cards', ids: ageCards('age1'), frames: [{ fixed: 'thumb' }, { sizes: 'cardFull' }] },
    { name: 'Age II cards', ids: ageCards('age2'), frames: [{ fixed: 'thumb' }, { sizes: 'cardFull' }] },
    { name: 'Age III cards', ids: ageCards('age3'), frames: [{ fixed: 'thumb' }, { sizes: 'cardFull' }] },
    { name: 'guilds', ids: ageCards('guild'), frames: [{ fixed: 'thumb' }, { sizes: 'cardFull' }] },
    { name: 'wonders', ids: wonders.map((w) => `wonder/${w.slug}`), frames: [{ fixed: 'thumb' }, { sizes: 'wonderLarge' }] },
    { name: 'tokens', ids: tokens.map((t) => `token/${t.slug}`), frames: [{ fixed: 'thumb' }, { sizes: 'token' }] },
    { name: 'backs', ids: ['back/age1', 'back/age2', 'back/age3', 'back/guild'], frames: [{ fixed: 'default' }] },
    {
      name: 'victory scenes',
      ids: ['victory/military', 'victory/science', 'victory/civilian'],
      frames: [{ fixed: 'default' }],
    },
  ];
  for (const device of DEVICES) {
    const seen = new Set();
    let sum = 0;
    const lines = [];
    for (const part of parts) {
      let bytes = 0;
      let files = 0;
      for (const id of part.ids) {
        const entry = entries[id];
        if (!entry) continue;
        for (const frame of part.frames) {
          const v = variantFor(entry, frame, device);
          if (!v || seen.has(v.path)) continue;
          seen.add(v.path);
          bytes += v.bytes;
          files++;
        }
      }
      sum += bytes;
      lines.push(`${part.name} ${(bytes / 1024).toFixed(0)} KB (${files})`);
    }
    out(`full prefetch plan, ${device.name}: ${(sum / 1024).toFixed(0)} KB in ${seen.size} files: ${lines.join(', ')}`);
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const manifest = await loadManifest();
  const own = buildEntries(manifest);
  const aliases = await aliasEntries();
  const entries = mergeSorted(own, aliases.entries);
  if (!args.quiet && !args.check) {
    for (const id of Object.keys(own)) {
      const asset = manifest.assets[id];
      const approved = approvedAttemptOf(asset);
      const earlier = approved < asset.attempt ? ` (earlier than the latest, ${asset.attempt})` : '';
      const recorded = asset.approvedAttempt ? '' : ', no approvedAttempt recorded, so the latest';
      const original = findOriginal(id);
      const origText = original ? `original ${displayPath(original)}` : 'WARNING: no original under art/originals/ yet (run process.mjs)';
      process.stdout.write(`index.mjs: ${id}: approved attempt ${approved}${earlier}${recorded}; ${origText}\n`);
    }
    for (const line of aliases.lines) process.stdout.write(`index.mjs: ${line}\n`);
  }
  const ts = await prettierFormat(renderTs(entries));
  if (args.check) {
    const current = existsSync(args.out) ? await readFile(args.out, 'utf8') : '';
    if (current !== ts) {
      die(
        `${displayPath(args.out)} is out of date or not formatted; run node tools/art/index.mjs${game.name === 'duel' ? '' : ` --game ${game.name}`}`,
      );
    }
    process.stdout.write(`index.mjs: ${displayPath(args.out)} is up to date (${Object.keys(entries).length} ids)\n`);
    return;
  }
  await mkdir(path.dirname(args.out), { recursive: true });
  await writeFile(args.out, ts, 'utf8');
  process.stdout.write(
    `index.mjs: wrote ${Object.keys(entries).length} approved id(s) from ${manifestLabel()} to ${displayPath(args.out)}\n`,
  );
  await report(entries, args.quiet);
}

main().catch((err) => die(err.stack || String(err)));
