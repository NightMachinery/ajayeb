#!/usr/bin/env node
// The inner loop: renders one or more icon files at several sizes on paper
// and on the night surface with rsvg-convert, in milliseconds and without a
// browser, into one PNG strip per file. librsvg ignores CSS custom
// properties (it only uses the var() fallbacks), so this script substitutes
// the two-tone colours itself before rendering; tools/art/render-fast.sh
// alone would show --i2 as its transparent fallback.
//
//   node tools/art/vector/fast.mjs web/src/lib/icons/svg/coin.svg
//   node tools/art/vector/fast.mjs a.svg b.svg --sizes 12,16,24,96 --i1 '#8a5a1c' --i2 '#e0b860'
//
// Flags:
//   --sizes LIST   pixel sizes (default 12,14,16,24,48,96)
//   --i1 COLOUR    the main tone on paper (default #2b2f45, the ink)
//   --i2 COLOUR    the detail tone on paper (default #a9a49a)
//   --i1-dark, --i2-dark   the same on the night surface (default #f1ead9, #6f7690)
//   --out DIR      output directory (default <run dir>/art-work/vector/fast)
//
// Writes <out>/<file name>.png: the paper row above the night row, sizes
// left to right, each at 1 device pixel per CSS pixel. Then look at it with
// the Read tool. Shoot the browser specimen (shoot.mjs) once per batch:
// Chrome is the reference, librsvg only the quick look.

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { parseArgs } from 'node:util';
import { ART_WORK, ensureDir, isMain, list, printHeader, shown } from './lib.mjs';

const PAPER = '#f6efe0';
const NIGHT = '#0f1322';

function run(cmd, args) {
  const r = spawnSync(cmd, args, { encoding: 'utf8' });
  if (r.error) throw new Error(`fast: cannot run ${cmd} (${r.error.message})`);
  if (r.status !== 0) throw new Error(`fast: ${cmd} failed: ${r.stderr.trim()}`);
}

/** Replaces every var(<name>, ...) in src, nested parentheses included, by `value`. */
function replaceVar(src, name, value) {
  let out = '';
  let i = 0;
  const re = new RegExp(`var\\(\\s*${name}\\b`, 'g');
  for (let m = re.exec(src); m; m = re.exec(src)) {
    let depth = 0;
    let k = m.index + 3;
    for (; k < src.length; k++) {
      if (src[k] === '(') depth++;
      else if (src[k] === ')' && --depth === 0) break;
    }
    out += src.slice(i, m.index) + value;
    i = k + 1;
    re.lastIndex = i;
  }
  return out + src.slice(i);
}

/** The source with the two-tone and halo var() paints replaced by fixed colours. */
export function substitute(src, { i1, i2, halo }) {
  let s = replaceVar(src, '--i1', i1);
  s = replaceVar(s, '--i2', i2);
  s = replaceVar(s, '--i-halo', halo);
  return s.replace(/currentColor/g, i1);
}

async function main() {
  const { values: o, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      sizes: { type: 'string', default: '12,14,16,24,48,96' },
      i1: { type: 'string', default: '#2b2f45' },
      i2: { type: 'string', default: '#a9a49a' },
      'i1-dark': { type: 'string', default: '#f1ead9' },
      'i2-dark': { type: 'string', default: '#6f7690' },
      out: { type: 'string' },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (o.help || positionals.length === 0) {
    printHeader(import.meta.url);
    return;
  }
  const sizes = list(o.sizes).map(Number);
  if (sizes.some((n) => !(n > 0))) throw new Error(`--sizes: bad list ${o.sizes}`);
  const out = ensureDir(o.out ?? path.join(ART_WORK, 'fast'));
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'ajayeb-vec-fast-'));
  try {
    for (const file of positionals) {
      const src = fs.readFileSync(file, 'utf8');
      const base = path.basename(file, '.svg');
      const rows = [];
      for (const [surface, bg, i1, i2] of [
        ['paper', PAPER, o.i1, o.i2],
        ['night', NIGHT, o['i1-dark'], o['i2-dark']]
      ]) {
        const svg = path.join(tmp, `${base}-${surface}.svg`);
        fs.writeFileSync(svg, substitute(src, { i1, i2, halo: bg }));
        const cells = [];
        for (const size of sizes) {
          const png = path.join(tmp, `${base}-${surface}-${size}.png`);
          run('rsvg-convert', [
            '--width',
            String(size),
            '--height',
            String(size),
            '--keep-aspect-ratio',
            '--background-color',
            bg,
            '--output',
            png,
            svg
          ]);
          cells.push(png);
        }
        // Centre every cell in a box as tall as the largest size, with a
        // gutter, then join the row.
        const row = path.join(tmp, `${base}-${surface}.png`);
        const tall = Math.max(...sizes) + 12;
        const boxed = cells.flatMap((c, k) => [
          '(',
          c,
          '-background',
          bg,
          '-gravity',
          'center',
          '-extent',
          `${sizes[k] + 12}x${tall}`,
          ')'
        ]);
        run('magick', [...boxed, '-background', bg, '+append', row]);
        rows.push(row);
      }
      const dest = path.join(out, `${base}.png`);
      run('magick', [...rows, '-append', dest]);
      console.error(`fast: ${shown(dest)}`);
    }
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

if (isMain(import.meta.url)) {
  main().catch((err) => {
    console.error(`${err.stack ?? err}`);
    process.exit(1);
  });
}
