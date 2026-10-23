#!/usr/bin/env node
// Writes the specimen page of one icon set: a self-contained HTML file that
// inlines web/src/lib/styles/tokens.css (and the game token sheets of
// GAME_TOKENS_CSS in lib.mjs), one shipped font and the set's SVGs
// (as <symbol>s used through <use>, the way the app's sprite works), so it
// needs no server and no file:// fetches. shoot.mjs photographs it.
//
//   node tools/art/vector/specimen.mjs --set resources-economy-military
//   node tools/art/vector/specimen.mjs --set chain --out /some/dir/specimen.html
//
// Flags:
//   --set NAME   a set from art/vector/sets.json (required)
//   --out PATH   the HTML to write (default <run dir>/art-work/vector/<set>/specimen.html)
//
// The page has, per file of the set (md and -sm rows):
//   A  light paper, at 12, 14, 16, 20, 24, 32 and 48 px, in the icon's
//      natural ink (sets.json "ink" and "i2"; "cardInk" on card mocks), plus a 24 px silhouette with
//      --i2 unset (the transparent fallback)
//   B  the same on the night surface, with the dark theme's tokens
//   C  all 7 card band colours (with their CSS patterns) at 14 px
//   D  in-context mocks, on md rows: the cost row and the effect band of a
//      54x83 mini card (font 10 px, as the structure draws it at 360 px),
//      a 16 px strip chip, the affordability badge at 10 and 14 px, and a
//      32 px track cell
// and at the top a compact peep strip (12, 14 and 16 px on paper and on the
// night surface, -sm variants at 14 px and below) that peep.mjs enlarges.
// The page's own theme (data-theme on <html>) drives the mocks; shoot.mjs
// flips it. Elements with data-rect are measured by shoot.mjs, so peep.mjs
// knows where to crop.

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import {
  ART_WORK,
  aspectOf,
  ensureDir,
  escapeHtml,
  fontFace,
  getSet,
  isMain,
  printHeader,
  runMain,
  setFiles,
  shown,
  symbolsFor,
  tokenSheets
} from './lib.mjs';

export const SIZES = [12, 14, 16, 20, 24, 32, 48];
export const PEEP_SIZES = [12, 14, 16];
export const BANDS = ['brown', 'grey', 'blue', 'green', 'yellow', 'red', 'purple'];

// The band patterns of web/src/lib/components/card/card-style.ts
// (patternLayers), copied so the specimen needs no build. Keep in step.
const PATTERN = {
  brown: 'repeating-linear-gradient(45deg, var(--pat) 0 1.5px, transparent 1.5px 6px)',
  grey: 'radial-gradient(circle, var(--pat) 0 1.3px, transparent 1.6px) 0 0 / 6px 6px',
  blue: 'repeating-linear-gradient(0deg, var(--pat) 0 1.5px, transparent 1.5px 5px)',
  green:
    'repeating-linear-gradient(45deg, var(--pat) 0 1px, transparent 1px 7px), ' +
    'repeating-linear-gradient(-45deg, var(--pat) 0 1px, transparent 1px 7px)',
  yellow:
    'linear-gradient(var(--pat), var(--pat)) 0 2px / 100% 1px no-repeat, ' +
    'linear-gradient(var(--pat), var(--pat)) 0 4px / 100% 1px no-repeat, ' +
    'linear-gradient(var(--pat), var(--pat)) 0 calc(100% - 2px) / 100% 1px no-repeat, ' +
    'linear-gradient(var(--pat), var(--pat)) 0 calc(100% - 4px) / 100% 1px no-repeat',
  red:
    'linear-gradient(135deg, var(--pat) 25%, transparent 25%) -4px 0 / 8px 8px, ' +
    'linear-gradient(225deg, var(--pat) 25%, transparent 25%) -4px 0 / 8px 8px',
  purple: 'repeating-conic-gradient(var(--pat) 0 25%, transparent 0 50%) 0 0 / 8px 8px'
};
export const bandBackground = (c) => `${PATTERN[c]}, var(--card-${c})`;
export const bandInk = (c) =>
  c === 'yellow' || c === 'grey' ? 'var(--card-ink)' : 'var(--paper-bright)';

// The affordability badge styles of AffordBadge.svelte, copied. Keep in step.
const BADGE = {
  chain: 'border-color:var(--gold-deep);background:var(--gold);color:var(--card-ink)',
  free: 'border-color:oklch(0.36 0.08 160);background:oklch(0.47 0.1 160);color:var(--paper-bright)',
  own: 'border-color:oklch(0.36 0.08 160);background:oklch(0.47 0.1 160);color:var(--paper-bright)',
  trade: 'border-color:oklch(0.6 0.13 70);background:var(--saffron);color:var(--card-ink)',
  short:
    'border-style:dashed;border-color:oklch(0.47 0.12 25);background:oklch(0.93 0.015 40);color:oklch(0.47 0.12 25)',
  neutral: 'border-color:var(--rule-strong);background:var(--paper-bright);color:var(--card-ink)'
};

/** The body of the first `selector { ... }` block in a stylesheet. */
function blockBody(css, selector) {
  const at = css.indexOf(selector);
  if (at < 0) return '';
  const open = css.indexOf('{', at);
  let depth = 0;
  for (let k = open; k < css.length; k++) {
    if (css[k] === '{') depth++;
    else if (css[k] === '}' && --depth === 0) return css.slice(open + 1, k);
  }
  return '';
}

/**
 * The tokens (tokens.css and the game token sheets), plus .theme-light and
 * .theme-dark classes that pin a subtree to one theme's values whatever the
 * page theme is (columns A and B).
 */
export function themedTokens() {
  const sheets = tokenSheets();
  const light = sheets.map((css) => blockBody(css, ':root {')).join('');
  const dark = sheets.map((css) => blockBody(css, ":root[data-theme='dark'] {")).join('');
  return `${sheets.join('\n')}\n.theme-light{${light}}\n.theme-dark{${light}${dark}}\n`;
}

/** One <svg><use/></svg> at a size, like Icon.svelte. Wide files keep their aspect. */
export function use(id, size, aspect = 1, extra = '') {
  const w = Math.round(size * aspect * 100) / 100;
  return `<svg class="ic" width="${w}" height="${size}" style="${extra}" aria-hidden="true"><use href="#i-${escapeHtml(id)}"/></svg>`;
}

/** The --i2 to use when the set gives none: the ink mixed into the surface. */
const i2For = (entry, surface) => entry.i2 ?? `color-mix(in oklch, currentColor 45%, ${surface})`;

function inkStyle(entry, surface) {
  return `color:${entry.ink ?? 'var(--fg)'};--i2:${i2For(entry, surface)};--i-halo:${surface}`;
}

/** The variant to show at a size: the -sm file at 14 px and below, when it exists. */
export function variantAt(entry, size, existing) {
  return entry.sm && size <= 14 && existing.has(entry.sm) ? entry.sm : entry.id;
}

function miniCard(entry, existing) {
  const id = variantAt(entry, 10, existing);
  const band = entry.band ?? 'brown';
  let cost;
  let effect;
  if (entry.tile) {
    const tile = (size) =>
      `<span class="tile" style="background:${entry.tile.bg};color:${entry.tile.ink};--i2:color-mix(in oklch, ${entry.tile.ink} 55%, ${entry.tile.bg});width:${size};height:${size}">${use(id, 9)}</span>`;
    cost = `${tile('1em')}<b class="num">2</b>`;
    effect = `${tile('1em')}${tile('1em')}`;
  } else if (entry.chip) {
    const chip = (bg) =>
      `<span class="chip" style="background:${bg};--i2:var(--gold-deep)">${use(id, 12)}</span>`;
    cost = `<span class="coin">3</span><span style="margin-inline-start:auto">${chip('var(--gold)')}</span>`;
    // The chain-out chip is outlined on paper: the same symbol, other fill.
    effect = chip('var(--paper)');
  } else {
    // On a card the ink is the card's (cards stay paper in both themes).
    const onCard = { ...entry, ink: entry.cardInk ?? entry.ink };
    cost = `<span style="${inkStyle(onCard, 'var(--card-paper)')}">${use(id, 11)}</span><b class="num">2</b>`;
    effect = `<span style="${inkStyle(onCard, 'var(--paper)')}">${use(id, 12)}</span><b class="num">2</b>`;
  }
  const badge = entry.badge ? badgeMock(entry, 10) : '';
  return (
    `<div class="card" style="--band:var(--card-${band})">` +
    `<div class="cost">${cost}</div>` +
    `<div class="band" style="background:${bandBackground(band)}"><span class="inset">${effect}</span></div>` +
    `<div class="art">${badge}</div></div>`
  );
}

function badgeMock(entry, size) {
  const style = BADGE[entry.badge ?? 'neutral'];
  // The badge shows this row's icon, whatever state it stands for.
  const ink = entry.badge
    ? 'color:inherit'
    : `color:${entry.cardInk ?? entry.ink ?? 'var(--card-ink)'}`;
  const i2 = entry.i2 ? `;--i2:${entry.i2}` : '';
  return `<span class="badge" style="${style};font-size:${size}px">${use(entry.id, size * 1.05, 1, ink + i2)}<span>3</span></span>`;
}

/** Builds the specimen HTML of a set (see the header comment). */
export function buildSpecimen(set, { stamp = new Date().toISOString() } = {}) {
  const files = setFiles(set);
  const existing = new Set(files.filter((f) => f.exists).map((f) => f.id));
  const missing = files.filter((f) => !f.exists).map((f) => f.id);
  const symbols = symbolsFor(files);
  const aspect = Object.fromEntries(files.map((f) => [f.id, f.exists ? aspectOf(f.file) : 1]));
  const concepts = set.icons.filter((e) => existing.has(e.id));

  const peepRows = [];
  for (const [surface, cls] of [
    ['paper', 'theme-light'],
    ['night', 'theme-dark']
  ]) {
    const bg = surface === 'paper' ? 'var(--paper)' : 'var(--night)';
    for (const size of PEEP_SIZES) {
      const cells = concepts
        .map((e) => {
          const id = variantAt(e, size, existing);
          return `<span class="pc" style="${inkStyle(e, bg)}">${use(id, size, aspect[id])}</span>`;
        })
        .join('');
      peepRows.push(
        `<div class="prow ${cls}" style="background:${bg};color:var(--fg)" data-rect="peep-${size}-${surface}"><span class="plabel">${size}</span>${cells}</div>`
      );
    }
  }

  const rows = files
    .filter((f) => f.exists)
    .map((f) => {
      const e = f.entry;
      const a = aspect[f.id];
      const sizes = (bg) =>
        SIZES.map((s) => `<td class="sz" style="${inkStyle(e, bg)}">${use(f.id, s, a)}</td>`).join(
          ''
        );
      const bands = BANDS.map((c) => {
        const ink = bandInk(c);
        return `<td class="bandcell" style="background:${bandBackground(c)};color:${ink};--i2:color-mix(in oklch, ${ink} 45%, var(--card-${c}));--i-halo:var(--card-${c})">${use(f.id, 14, a)}</td>`;
      }).join('');
      const mocks =
        f.variant === 'md'
          ? `<td class="mock">${miniCard(e, existing)}</td>` +
            `<td class="mock"><span class="stripchip" style="--i2:${i2For(e, 'var(--panel)')}"><span style="color:${e.ink ?? 'var(--fg)'}">${use(f.id, 16, a)}</span><b>2</b><span class="muted">·</span><span class="muted">3</span></span></td>` +
            `<td class="mock">${badgeMock(e, 10)}<br>${badgeMock(e, 14)}</td>` +
            `<td class="mock"><span class="trackcell" style="${inkStyle(e, 'var(--panel-sunken)')}">${use(f.id, 20, a)}</span></td>`
          : '<td class="mock muted" colspan="4">(-sm: see the mini card on the row above)</td>';
      return (
        `<tr><th class="id">${escapeHtml(f.id)}</th>` +
        `<td class="surf theme-light"><table><tr>${sizes('var(--paper)')}<td class="sz mono" style="color:var(--fg)">${use(f.id, 24, a)}</td></tr></table></td>` +
        `<td class="surf theme-dark night"><table><tr>${sizes('var(--night)')}</tr></table></td>` +
        `<td class="surf"><table><tr>${bands}</tr></table></td>` +
        `${mocks}</tr>`
      );
    })
    .join('');

  const head =
    '<tr><th>file</th>' +
    `<th>A: paper, ${SIZES.join(' / ')} px, then a 24 px silhouette (--i2 unset)</th>` +
    `<th>B: night surface, ${SIZES.join(' / ')} px</th>` +
    `<th>C: bands at 14 px (${BANDS.join(', ')})</th>` +
    '<th>mini card 54x83</th><th>strip chip 16 px</th><th>badge 10 and 14 px</th><th>track cell 32 px</th></tr>';

  return `<!doctype html>
<html lang="en" data-theme="light"><head><meta charset="utf-8">
<title>${escapeHtml(set.name)} specimen</title>
<style>
${themedTokens()}
${fontFace('Inter', 'inter-700.woff2', 700)}
html, body { margin: 0; }
body { display: inline-block; background: var(--table); color: var(--fg);
  font: 11px/1.3 ui-monospace, Menlo, Consolas, monospace; padding: 12px; }
h1 { font: 600 15px/1.2 ui-sans-serif, system-ui, sans-serif; margin: 0 0 4px; }
.meta { color: var(--fg-muted); margin-bottom: 10px; }
.peep { display: inline-flex; flex-direction: column; margin-bottom: 14px; border: 1px solid var(--rule); }
.prow { display: flex; align-items: center; gap: 6px; padding: 4px 8px; }
.plabel { width: 18px; color: var(--fg-muted); font-size: 9px; }
.pc { display: inline-grid; place-items: center; min-width: 18px; }
.ic { display: block; flex: none; overflow: visible; }
table { border-collapse: collapse; }
.main > tbody > tr > * { border: 1px solid var(--rule); padding: 4px 6px; vertical-align: middle; }
.main th { text-align: start; font-weight: 600; }
.main tr:first-child th { white-space: normal; max-width: 220px; vertical-align: bottom; }
.main th.id { white-space: nowrap; }
.surf { padding: 0 !important; }
.surf.theme-light { background: var(--paper); }
.surf.night { background: var(--night); }
.sz { padding: 4px 5px; }
.bandcell { width: 24px; height: 24px; padding: 0; --pat: color-mix(in oklch, var(--card-ink) 20%, transparent); }
.bandcell .ic { margin: auto; }
.muted { color: var(--fg-muted); }
.mock { background: var(--table); }
.card { position: relative; display: flex; flex-direction: column; width: 54px; height: 83px; overflow: hidden;
  font: 700 10px/1 Inter, ui-sans-serif, sans-serif; font-variant-numeric: tabular-nums;
  background: var(--card-paper); color: var(--card-ink); border-radius: 7% / 4.5%;
  border: 1.5px solid color-mix(in oklch, var(--band) 75%, var(--card-ink)); box-sizing: border-box; }
.cost { height: 17%; display: flex; align-items: center; gap: 0.12em; padding: 0 2.5%; }
.band { height: 23%; display: flex; align-items: center; justify-content: center; --pat: color-mix(in oklch, var(--card-ink) 20%, transparent); }
.inset { display: inline-flex; align-items: center; gap: 0.12em; padding: 0.12em 0.15em; border-radius: 0.35em;
  background: color-mix(in oklch, var(--paper) 92%, transparent); }
.art { flex: 1; position: relative; background: color-mix(in oklch, var(--band) 14%, var(--card-paper));
  display: flex; align-items: flex-end; justify-content: center; padding-bottom: 0.3em; }
.tile { display: inline-grid; place-items: center; border-radius: 0.25em; }
.chip { display: inline-grid; place-items: center; width: max(1.3em, 14px); height: max(1.3em, 14px); border-radius: 0.35em;
  border: 1px solid var(--gold-deep); color: var(--card-ink); box-sizing: border-box; }
.coin { display: inline-grid; place-items: center; width: 1.25em; height: 1.25em; border-radius: 50%;
  background: var(--gold); border: 1px solid var(--gold-deep); box-sizing: border-box; }
.num { font-weight: 700; }
.stripchip { display: inline-flex; align-items: center; gap: 2px; padding: 2px 6px; border-radius: var(--shape-control);
  border: 1px solid var(--rule); background: var(--panel); color: var(--fg);
  font: 600 16px/1 Inter, ui-sans-serif, sans-serif; font-variant-numeric: tabular-nums; }
.badge { display: inline-flex; align-items: center; gap: 0.2em; height: 1.5em; padding: 0 0.4em; margin: 2px 0;
  border: 1.5px solid; border-radius: 999px; box-sizing: border-box;
  font: 700 1em/1 Inter, ui-sans-serif, sans-serif; font-variant-numeric: tabular-nums; }
.trackcell { display: grid; place-items: center; width: 32px; height: 32px; box-sizing: border-box;
  background: var(--panel-sunken); border: 1px solid var(--rule-strong); border-radius: 4px; }
.missing { color: var(--cinnabar); margin: 6px 0; }
</style></head>
<body>
<svg aria-hidden="true" style="position:absolute;width:0;height:0;overflow:hidden" xmlns="http://www.w3.org/2000/svg">${symbols}</svg>
<h1>${escapeHtml(set.title ?? set.name)} (${escapeHtml(set.name)})</h1>
<div class="meta">${existing.size} files | ${escapeHtml(stamp)} | page theme drives the mocks</div>
${missing.length ? `<div class="missing">not drawn yet: ${missing.map(escapeHtml).join(', ')}</div>` : ''}
<div class="peep" data-rect="peep">${peepRows.join('')}</div>
<table class="main" data-rect="main"><tbody>${head}${rows}</tbody></table>
</body></html>
`;
}

/** The default output directory of a set's renders. */
export const setOutDir = (name) => path.join(ART_WORK, name);

async function main() {
  const { values: o } = parseArgs({
    options: {
      set: { type: 'string' },
      out: { type: 'string' },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (o.help) {
    printHeader(import.meta.url);
    return;
  }
  if (!o.set) throw new Error('--set NAME is required');
  const set = getSet(o.set);
  const out = o.out ?? path.join(setOutDir(set.name), 'specimen.html');
  ensureDir(path.dirname(path.resolve(out)));
  fs.writeFileSync(out, buildSpecimen(set));
  console.error(`specimen: wrote ${shown(out)}`);
}

if (isMain(import.meta.url)) runMain(main);
