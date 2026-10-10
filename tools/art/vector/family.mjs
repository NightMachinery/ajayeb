#!/usr/bin/env node
// The family sheet: every drawn icon of every set side by side at 16 and
// 24 px, on paper and on the night surface, so stroke weight, optical size,
// corner style, density and the two-tone balance can be compared across
// sets. One Chrome, one page, two shots (DPR 1 and 2):
//
//   python3 tools/heavy.py run --kind browser -- node tools/art/vector/family.mjs
//
// Flags:
//   --sets LIST   only these sets (default: all of art/vector/sets.json)
//   --mono        plain ink everywhere instead of each icon's natural ink
//   --out DIR     output directory (default <run dir>/art-work/vector/family)
//   --force       launch Chrome without the heavy.py browser lock
//
// Writes family@1x.png, family@2x.png and family.html.

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import {
  ART_WORK,
  ensureDir,
  escapeHtml,
  fontFace,
  isMain,
  list,
  loadSets,
  printHeader,
  runMain,
  setFiles,
  shown,
  symbolsFor,
  withShooter
} from './lib.mjs';
import { themedTokens, use } from './specimen.mjs';

export function buildFamily(names, { mono = false } = {}) {
  const sets = loadSets().sets;
  let symbols = '';
  const blocks = [];
  for (const name of names) {
    if (!sets[name]) throw new Error(`no set "${name}"`);
    const files = setFiles({ name, ...sets[name] }).filter((f) => f.exists && f.variant === 'md');
    symbols += symbolsFor(files);
    if (files.length === 0) {
      blocks.push(
        `<section><h2>${escapeHtml(name)}</h2><p class="none">nothing drawn yet</p></section>`
      );
      continue;
    }
    const strip = (size, surface) => {
      const bg = surface === 'paper' ? 'var(--paper)' : 'var(--night)';
      const cls = surface === 'paper' ? 'theme-light' : 'theme-dark';
      const cells = files
        .map((f) => {
          const ink = mono ? 'var(--fg)' : (f.entry.ink ?? 'var(--fg)');
          const i2 =
            f.entry.i2 && !mono ? f.entry.i2 : `color-mix(in oklch, currentColor 45%, ${bg})`;
          return `<span style="color:${ink};--i2:${i2};--i-halo:${bg}">${use(f.id, size)}</span>`;
        })
        .join('');
      return `<div class="strip ${cls}" style="background:${bg}"><span class="lab">${size}</span>${cells}</div>`;
    };
    blocks.push(
      `<section><h2>${escapeHtml(name)} (${files.length})</h2>` +
        strip(16, 'paper') +
        strip(24, 'paper') +
        strip(16, 'night') +
        strip(24, 'night') +
        '</section>'
    );
  }
  return `<!doctype html><html data-theme="light"><head><meta charset="utf-8"><style>
${themedTokens()}
${fontFace('Inter', 'inter-700.woff2', 700)}
html, body { margin: 0; }
body { display: inline-block; background: var(--table); color: var(--fg); padding: 12px; font: 11px/1.3 ui-monospace, Menlo, monospace; }
h2 { font: 600 12px/1.2 ui-sans-serif, system-ui, sans-serif; margin: 10px 0 4px; }
.strip { display: flex; align-items: center; gap: 8px; padding: 5px 8px; color: var(--fg); }
.lab { width: 18px; font-size: 9px; color: var(--fg-muted); }
.ic { display: block; }
.none { color: var(--fg-muted); margin: 0; }
</style></head><body>
<svg aria-hidden="true" style="position:absolute;width:0;height:0;overflow:hidden" xmlns="http://www.w3.org/2000/svg">${symbols}</svg>
${blocks.join('\n')}
</body></html>`;
}

async function main() {
  const { values: o } = parseArgs({
    options: {
      sets: { type: 'string' },
      mono: { type: 'boolean', default: false },
      out: { type: 'string' },
      force: { type: 'boolean', default: false },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (o.help) {
    printHeader(import.meta.url);
    return;
  }
  const names = o.sets ? list(o.sets) : Object.keys(loadSets().sets);
  const out = ensureDir(o.out ?? path.join(ART_WORK, 'family'));
  const html = buildFamily(names, { mono: o.mono });
  fs.writeFileSync(path.join(out, 'family.html'), html);
  await withShooter(
    async (shooter) => {
      await shooter.load(html);
      for (const dpr of [1, 2]) {
        const file = path.join(out, `family@${dpr}x.png`);
        const m = await shooter.capture({ out: file, dpr, theme: 'light' });
        console.error(`family: ${shown(file)} (${m.width * dpr}x${m.height * dpr})`);
      }
    },
    { force: o.force }
  );
}

if (isMain(import.meta.url)) runMain(main);
