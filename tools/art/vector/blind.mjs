#!/usr/bin/env node
// The blind identification test of a set: shuffled, numbered, unlabelled
// sheets, and a scorer. The key is written outside the repo, next to the
// sheets, in a file named so nobody opens it by accident.
//
//   python3 tools/heavy.py run --kind browser -- \
//     node tools/art/vector/blind.mjs make --set science [--sizes 12@2,14@2,16@1] \
//     [--reader-dir DIR] [--seed 7] [--mono]
//   node tools/art/vector/blind.mjs score --dir <the dir make printed> --answers answers.txt
//
// make writes <run dir>/art-work/vector/blind/<set>-<stamp>/ with:
//   sheet-<px>px@<dpr>x.png   one sheet per size of --sizes: every icon of the
//                       set at <px> CSS px and device pixel ratio <dpr>,
//                       shuffled and numbered 1..N, each sheet in its own
//                       order. At 14 px and below a sheet uses the -sm
//                       variant where one exists, as the app does. The
//                       default sizes give sheet-12px@2x.png and
//                       sheet-16px@1x.png.
//   names.txt           the candidate names, sorted, and how to answer
//   answers.txt         a template: one line "<sheet> <tile> " per tile, where
//                       the sheet id is "<px>@<dpr>", such as 12@2
//   KEY-do-not-open-before-answering.json   the key
// A set may list "neighbours" in sets.json: ids from other sets that its
// icons must not be confused with. They join the sheets and names.txt as
// ordinary candidates, so a reader cannot tell them from the set's own icons.
// Hand the reader (you, or a fresh agent) the sheets and names.txt only.
// A reader who has seen the key may not take the test. --reader-dir does the
// handing over for you: it copies only the sheets, names.txt and the answers
// template into a directory of their own, so a fresh agent pointed at that
// directory cannot see the key.
//
// Flags for make:
//   --set NAME        the set (required)
//   --sizes LIST      the sheets, as <px>@<dpr> joined by commas, such as
//                     12@2,14@2,16@1 (default 12@2,16@1, the sheets of the
//                     earlier runs, so those stay reproducible from the seed)
//   --reader-dir DIR  also copy the sheets, names.txt and answers.txt (never
//                     the key) into DIR; it is created, and make refuses
//                     before drawing anything if it exists and is not empty
//   --seed N          the shuffle seed (default: from the clock); the key records it
//   --mono            draw every icon in plain ink with no tile, so only the
//                     shape counts (default: the icon's on-card ink, as on a card)
//   --force           launch Chrome without the heavy.py browser lock
//
// score reads answers lines "<sheet> <tile> <name>" and prints one line per
// icon and the totals per sheet. The sheet is "<px>@<dpr>" (12@2); a bare
// "<px>" (12) also works when only one sheet has that size, which is how the
// directories made before --sizes were answered. The name is as in
// names.txt, or "?" for no idea; a unique suffix such as "wood" also matches.

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import {
  ART_WORK,
  ensureDir,
  fontFace,
  getSet,
  isMain,
  list,
  loadSets,
  printHeader,
  runMain,
  setFiles,
  shown,
  shuffled,
  symbolsFor,
  withShooter
} from './lib.mjs';
import { themedTokens, use, variantAt } from './specimen.mjs';

const KEY_NAME = 'KEY-do-not-open-before-answering.json';
/** The sizes of the earlier runs, and of make without --sizes. */
export const DEFAULT_SIZES = '12@2,16@1';
/** The sheet sizes of keys written before --sizes, which named them "12" and "16". */
const LEGACY_DPR = { 12: 2, 16: 1 };

/** Parses "12@2,14@2,16@1" into sheets {id, px, dpr, file}, in that order. */
export function parseSizes(spec) {
  const sheets = list(spec).map((item) => {
    const m = /^(\d+)@(\d+(?:\.\d+)?)$/.exec(item);
    if (!m) throw new Error(`--sizes: expected <px>@<dpr> such as 12@2, got "${item}"`);
    const px = Number(m[1]);
    const dpr = Number(m[2]);
    if (px < 6 || px > 128 || dpr <= 0 || dpr > 4)
      throw new Error(`--sizes: ${item} is out of range (6 to 128 px, DPR up to 4)`);
    return { id: `${px}@${dpr}`, px, dpr, file: `sheet-${px}px@${dpr}x.png` };
  });
  if (sheets.length === 0) throw new Error('--sizes: give at least one <px>@<dpr>');
  const ids = sheets.map((sh) => sh.id);
  const dup = ids.find((id, i) => ids.indexOf(id) !== i);
  if (dup) throw new Error(`--sizes: ${dup} is listed twice`);
  return sheets;
}

/** The px of a key's sheet id: "12@2" and the legacy "12" both give 12. */
const pxOf = (id) => Number(String(id).split('@')[0]);
/** A key's sheet id with its DPR spelled out ("12" becomes "12@2"). */
const fullId = (id) => (String(id).includes('@') ? String(id) : `${id}@${LEGACY_DPR[id] ?? '?'}`);

/**
 * Maps an answer's sheet word onto one of the key's sheet ids, or null.
 * Accepts the exact id, "<px>@<dpr>" for a legacy bare key id, and a bare
 * "<px>" when exactly one sheet of the key has that size.
 */
export function sheetFor(word, keyIds) {
  if (keyIds.includes(word)) return word;
  const hit = keyIds.find((id) => fullId(id) === word);
  if (hit) return hit;
  if (/^\d+$/.test(word)) {
    const same = keyIds.filter((id) => pxOf(id) === Number(word));
    if (same.length === 1) return same[0];
  }
  return null;
}

/**
 * The entries of a set's "neighbours": icons of other sets that its glyphs
 * must not be confused with. Each is looked up in the set that owns it, so it
 * keeps its own ink, and is shuffled into the sheets as one more candidate.
 */
export function neighbourEntries(set, sets = loadSets().sets) {
  return (set.neighbours ?? []).map((id) => {
    const entry = Object.values(sets)
      .flatMap((s) => s.icons)
      .find((e) => e.id === id);
    if (!entry) throw new Error(`set ${set.name}: neighbour "${id}" is in no set`);
    if (set.icons.some((e) => e.id === id))
      throw new Error(`set ${set.name}: neighbour "${id}" is one of its own icons`);
    return entry;
  });
}

/** Refuses a reader directory that already holds something. */
function checkReaderDir(dir) {
  if (!fs.existsSync(dir)) return;
  if (!fs.statSync(dir).isDirectory()) throw new Error(`--reader-dir: ${dir} is not a directory`);
  if (fs.readdirSync(dir).length > 0)
    throw new Error(`--reader-dir: ${dir} exists and is not empty; give a new or empty directory`);
}

function sheetHtml({ symbols, tiles, size, mono }) {
  const cols = Math.min(10, tiles.length);
  // Cells grow with the icon, so a large size never crowds its neighbours.
  // Up to 16 px they stay 40 by 22 px, the geometry of the earlier sheets.
  const cell = Math.max(40, Math.ceil(size * 1.6));
  const box = Math.max(22, Math.ceil(size * 1.15) + 2);
  const cells = tiles
    .map((t, k) => {
      const ink = mono ? 'var(--ink)' : (t.entry.cardInk ?? t.entry.ink ?? 'var(--ink)');
      const i2 = mono
        ? 'color-mix(in oklch, var(--ink) 45%, var(--paper))'
        : (t.entry.i2 ?? 'color-mix(in oklch, currentColor 45%, var(--paper))');
      const icon =
        t.entry.tile && !mono
          ? `<span class="tile" style="background:${t.entry.tile.bg};color:${t.entry.tile.ink};--i2:color-mix(in oklch, ${t.entry.tile.ink} 55%, ${t.entry.tile.bg});width:${size * 1.15}px;height:${size * 1.15}px">${use(t.id, size)}</span>`
          : `<span style="color:${ink};--i2:${i2}">${use(t.id, size)}</span>`;
      return `<div class="cell"><div class="box">${icon}</div><div class="n">${k + 1}</div></div>`;
    })
    .join('');
  return `<!doctype html><html data-theme="light"><head><meta charset="utf-8"><style>
${themedTokens()}
${fontFace('Inter', 'inter-700.woff2', 700)}
html, body { margin: 0; }
body { display: inline-block; background: var(--paper); color: var(--ink); padding: 10px; }
.grid { display: grid; grid-template-columns: repeat(${cols}, ${cell}px); gap: 8px 6px; }
.cell { display: flex; flex-direction: column; align-items: center; gap: 3px; }
.box { height: ${box}px; display: grid; place-items: center; }
.tile { display: inline-grid; place-items: center; border-radius: 0.25em; }
.ic { display: block; }
.n { font: 700 10px/1 Inter, sans-serif; color: var(--ink-soft); }
</style></head><body>
<svg aria-hidden="true" style="position:absolute;width:0;height:0;overflow:hidden" xmlns="http://www.w3.org/2000/svg">${symbols}</svg>
<div class="grid">${cells}</div></body></html>`;
}

async function make(o) {
  if (!o.set) throw new Error('--set NAME is required');
  const sheets = parseSizes(o.sizes ?? DEFAULT_SIZES);
  const readerDir = o['reader-dir'] ? path.resolve(o['reader-dir']) : null;
  if (readerDir) checkReaderDir(readerDir);
  const set = getSet(o.set);
  const near = neighbourEntries(set);
  const files = [...setFiles(set), ...setFiles({ icons: near })];
  const existing = new Set(files.filter((f) => f.exists).map((f) => f.id));
  const concepts = [...set.icons, ...near].filter((e) => existing.has(e.id));
  if (concepts.length < 2) throw new Error(`set ${set.name} has fewer than 2 drawn icons`);
  const seed = Number(o.seed ?? Date.now() % 1_000_000_007);
  const stamp = new Date().toISOString().replace(/[:.]/g, '-');
  const dir = ensureDir(path.join(ART_WORK, 'blind', `${set.name}-${stamp}`));
  if (readerDir && readerDir === dir) throw new Error('--reader-dir must not be the key directory');
  const symbols = symbolsFor(files);

  const key = {
    set: set.name,
    seed,
    mono: !!o.mono,
    made: new Date().toISOString(),
    sizes: sheets.map((sh) => sh.id),
    sheets: {}
  };
  await withShooter(
    async (shooter) => {
      // Sheet k is shuffled with seed + k, as before --sizes, so the default
      // sizes and a given seed still give the same sheets.
      for (const [k, sheet] of sheets.entries()) {
        const order = shuffled(concepts, seed + k);
        const tiles = order.map((entry) => ({ entry, id: variantAt(entry, sheet.px, existing) }));
        await shooter.load(sheetHtml({ symbols, tiles, size: sheet.px, mono: o.mono }));
        await shooter.capture({ out: path.join(dir, sheet.file), dpr: sheet.dpr, theme: 'light' });
        key.sheets[sheet.id] = Object.fromEntries(
          tiles.map((t, i) => [String(i + 1), { concept: t.entry.id, file: t.id }])
        );
      }
    },
    { force: o.force }
  );
  const names = concepts.map((e) => e.id).sort();
  const sheetList = sheets.map((sh) => `  ${sh.file}: sheet ${sh.id}`).join('\n');
  fs.writeFileSync(
    path.join(dir, 'names.txt'),
    `Candidate names for the blind test of the "${set.name}" set (${names.length} icons).\n` +
      'The sheets, named by size in CSS px and device pixel ratio:\n' +
      `${sheetList}\n` +
      'Each sheet shows every icon once, in its own shuffled order. For each tile,\n' +
      'fill in its line of answers.txt as "<sheet> <tile> <name>" (such as\n' +
      `"${sheets[0].id} 1 ${names[0]}"), or "?" as the name if you cannot tell.\n` +
      'Answer every tile of every sheet before anyone opens the key.\n\n' +
      `${names.join('\n')}\n`
  );
  const template = sheets.flatMap((sh) => concepts.map((_, i) => `${sh.id} ${i + 1} `)).join('\n');
  fs.writeFileSync(path.join(dir, 'answers.txt'), `${template}\n`);
  fs.writeFileSync(path.join(dir, KEY_NAME), `${JSON.stringify(key, null, 2)}\n`);
  console.error(
    `blind: ${sheets.length} sheets (${sheets.map((sh) => sh.id).join(', ')}), names.txt and answers.txt in ${shown(dir)}` +
      ` (the key is ${KEY_NAME}; do not read it before answering)`
  );
  if (readerDir) {
    ensureDir(readerDir);
    for (const f of [...sheets.map((sh) => sh.file), 'names.txt', 'answers.txt'])
      fs.copyFileSync(path.join(dir, f), path.join(readerDir, f));
    console.error(
      `blind: reader copy (sheets, names.txt, answers.txt; no key) in ${shown(readerDir)}`
    );
    console.error(
      `blind: score with: node tools/art/vector/blind.mjs score --dir ${shown(dir)} --answers ${shown(path.join(readerDir, 'answers.txt'))}`
    );
    console.log(`key dir: ${dir}`);
    console.log(`reader dir: ${readerDir}`);
  } else {
    console.log(dir);
  }
}

function score(o) {
  if (!o.dir || !o.answers) throw new Error('score needs --dir DIR and --answers FILE');
  const key = JSON.parse(fs.readFileSync(path.join(o.dir, KEY_NAME), 'utf8'));
  const keyIds = Object.keys(key.sheets);
  const names = [
    ...new Set(Object.values(key.sheets).flatMap((s) => Object.values(s).map((t) => t.concept)))
  ];
  const resolve = (word) => {
    if (!word || word === '?') return null;
    if (names.includes(word)) return word;
    const hits = names.filter((n) => n.endsWith(`-${word}`) || n.endsWith(word));
    return hits.length === 1 ? hits[0] : word;
  };
  const answers = {};
  const unknown = new Set();
  for (const line of fs.readFileSync(o.answers, 'utf8').split('\n')) {
    const m = /^\s*(\d+(?:@\d+(?:\.\d+)?)?)\s+(\d+)\s+(\S+)?/.exec(line);
    if (!m) continue;
    const sheet = sheetFor(m[1], keyIds);
    if (!sheet) {
      unknown.add(m[1]);
      continue;
    }
    (answers[sheet] ??= {})[m[2]] = resolve(m[3]);
  }
  if (unknown.size > 0)
    console.error(
      `blind: ignored answers for sheets ${[...unknown].join(', ')}: this key has ${keyIds.map(fullId).join(', ')}`
    );
  const perIcon = Object.fromEntries(names.map((n) => [n, {}]));
  for (const [sheet, tiles] of Object.entries(key.sheets)) {
    let ok = 0;
    for (const [tile, t] of Object.entries(tiles)) {
      const given = answers[sheet]?.[tile] ?? null;
      const right = given === t.concept;
      if (right) ok++;
      perIcon[t.concept][sheet] = right ? 'ok' : `MISS (tile ${tile} read as ${given ?? '?'})`;
    }
    const n = Object.keys(tiles).length;
    console.log(`sheet ${fullId(sheet)}: ${ok}/${n} (${Math.round((100 * ok) / n)}%)`);
  }
  for (const [name, r] of Object.entries(perIcon)) {
    console.log(
      `${name}: ${Object.entries(r)
        .map(([s, v]) => `${fullId(s)} ${v}`)
        .join(', ')}`
    );
  }
}

async function main() {
  const [cmd, ...rest] = process.argv.slice(2);
  const { values: o } = parseArgs({
    args: rest,
    options: {
      set: { type: 'string' },
      seed: { type: 'string' },
      sizes: { type: 'string' },
      'reader-dir': { type: 'string' },
      mono: { type: 'boolean', default: false },
      force: { type: 'boolean', default: false },
      dir: { type: 'string' },
      answers: { type: 'string' },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (!cmd || cmd === '--help' || cmd === '-h' || o.help) {
    printHeader(import.meta.url);
    return;
  }
  if (cmd === 'make') return make(o);
  if (cmd === 'score') return score(o);
  throw new Error(`unknown command "${cmd}" (make or score)`);
}

if (isMain(import.meta.url)) runMain(main);
