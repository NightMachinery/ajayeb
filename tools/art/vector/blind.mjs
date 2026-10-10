#!/usr/bin/env node
// The blind identification test of a set: shuffled, numbered, unlabelled
// sheets, and a scorer. The key is written outside the repo, next to the
// sheets, in a file named so nobody opens it by accident.
//
//   python3 tools/heavy.py run --kind browser -- \
//     node tools/art/vector/blind.mjs make --set resources-economy-military [--seed 7] [--mono]
//   node tools/art/vector/blind.mjs score --dir <the dir make printed> --answers answers.txt
//
// make writes <run dir>/art-work/vector/blind/<set>-<stamp>/ with:
//   sheet-12px@2x.png   every icon of the set at 12 px, DPR 2 (-sm variants
//                       where they exist), shuffled and numbered 1..N
//   sheet-16px@1x.png   the same at 16 px, DPR 1 (md variants), shuffled again
//                       with a different order
//   names.txt           the candidate names, sorted, and how to answer
//   answers.txt         a template: one line "<sheet> <tile> " per tile
//   KEY-do-not-open-before-answering.json   the key
// Hand the reader (you, or a fresh agent) the two sheets and names.txt only.
// A reader who has seen the key may not take the test.
//
// Flags for make:
//   --set NAME   the set (required)
//   --seed N     the shuffle seed (default: from the clock); the key records it
//   --mono       draw every icon in plain ink with no tile, so only the shape
//                counts (default: the icon's on-card ink, as on a card)
//   --force      launch Chrome without the heavy.py browser lock
//
// score reads answers lines "<sheet> <tile> <name>" (sheet 12 or 16; the name
// as in names.txt, or "?" for no idea; a unique suffix such as "wood" also
// matches) and prints one line per icon and the totals per sheet.

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import {
  ART_WORK,
  ensureDir,
  fontFace,
  getSet,
  isMain,
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
export const SHEETS = [
  { name: '12', size: 12, dpr: 2, file: 'sheet-12px@2x.png' },
  { name: '16', size: 16, dpr: 1, file: 'sheet-16px@1x.png' }
];

function sheetHtml({ symbols, tiles, size, mono }) {
  const cols = Math.min(10, tiles.length);
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
.grid { display: grid; grid-template-columns: repeat(${cols}, 40px); gap: 8px 6px; }
.cell { display: flex; flex-direction: column; align-items: center; gap: 3px; }
.box { height: 22px; display: grid; place-items: center; }
.tile { display: inline-grid; place-items: center; border-radius: 0.25em; }
.ic { display: block; }
.n { font: 700 10px/1 Inter, sans-serif; color: var(--ink-soft); }
</style></head><body>
<svg aria-hidden="true" style="position:absolute;width:0;height:0;overflow:hidden" xmlns="http://www.w3.org/2000/svg">${symbols}</svg>
<div class="grid">${cells}</div></body></html>`;
}

async function make(o) {
  if (!o.set) throw new Error('--set NAME is required');
  const set = getSet(o.set);
  const files = setFiles(set);
  const existing = new Set(files.filter((f) => f.exists).map((f) => f.id));
  const concepts = set.icons.filter((e) => existing.has(e.id));
  if (concepts.length < 2) throw new Error(`set ${set.name} has fewer than 2 drawn icons`);
  const seed = Number(o.seed ?? Date.now() % 1_000_000_007);
  const stamp = new Date().toISOString().replace(/[:.]/g, '-');
  const dir = ensureDir(path.join(ART_WORK, 'blind', `${set.name}-${stamp}`));
  const symbols = symbolsFor(files);

  const key = { set: set.name, seed, mono: !!o.mono, made: new Date().toISOString(), sheets: {} };
  await withShooter(
    async (shooter) => {
      for (const [k, sheet] of SHEETS.entries()) {
        const order = shuffled(concepts, seed + k);
        const tiles = order.map((entry) => ({ entry, id: variantAt(entry, sheet.size, existing) }));
        await shooter.load(sheetHtml({ symbols, tiles, size: sheet.size, mono: o.mono }));
        await shooter.capture({ out: path.join(dir, sheet.file), dpr: sheet.dpr, theme: 'light' });
        key.sheets[sheet.name] = Object.fromEntries(
          tiles.map((t, i) => [String(i + 1), { concept: t.entry.id, file: t.id }])
        );
      }
    },
    { force: o.force }
  );
  const names = concepts.map((e) => e.id).sort();
  fs.writeFileSync(
    path.join(dir, 'names.txt'),
    `Candidate names for the blind test of the "${set.name}" set (${names.length} icons).\n` +
      'Each sheet shows every icon once, in its own shuffled order. For each tile,\n' +
      'write "<sheet> <tile> <name>" (sheet 12 or 16), or "?" as the name if you\n' +
      'cannot tell. Answer every tile of both sheets before anyone opens the key.\n\n' +
      `${names.join('\n')}\n`
  );
  const template = SHEETS.flatMap((s) => concepts.map((_, i) => `${s.name} ${i + 1} `)).join('\n');
  fs.writeFileSync(path.join(dir, 'answers.txt'), `${template}\n`);
  fs.writeFileSync(path.join(dir, KEY_NAME), `${JSON.stringify(key, null, 2)}\n`);
  console.error(
    `blind: sheets, names.txt and answers.txt in ${shown(dir)} (the key is ${KEY_NAME}; do not read it before answering)`
  );
  console.log(dir);
}

function score(o) {
  if (!o.dir || !o.answers) throw new Error('score needs --dir DIR and --answers FILE');
  const key = JSON.parse(fs.readFileSync(path.join(o.dir, KEY_NAME), 'utf8'));
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
  for (const line of fs.readFileSync(o.answers, 'utf8').split('\n')) {
    const m = /^\s*(\d+)\s+(\d+)\s+(\S+)?/.exec(line);
    if (!m) continue;
    (answers[m[1]] ??= {})[m[2]] = resolve(m[3]);
  }
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
    console.log(`sheet ${sheet} px: ${ok}/${n} (${Math.round((100 * ok) / n)}%)`);
  }
  for (const [name, r] of Object.entries(perIcon)) {
    console.log(
      `${name}: ${Object.entries(r)
        .map(([s, v]) => `${s}px ${v}`)
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
