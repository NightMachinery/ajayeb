#!/usr/bin/env node
// Lints hand-authored icon SVGs against art/vector/STYLE.md. Prints one line
// per problem ("path:line: error|warn rule: message") and a summary on
// stderr; exits 1 when there is an error (or a warning, with --strict).
//
//   node tools/art/vector/lint.mjs                     # the whole icon source directory
//   node tools/art/vector/lint.mjs --set science       # one set from art/vector/sets.json
//   node tools/art/vector/lint.mjs web/src/lib/icons/svg/coin.svg other.svg
//
// Flags:
//   --set NAME    lint the files of a set (repeatable); missing files are warnings
//   --strict      warnings fail too
//   --json        print the problems as one JSON array instead of lines
//   --quiet       print errors only
//
// With no paths and no --set it lints web/src/lib/icons/svg, checks that the
// files match web/src/lib/icons/names.ts, and checks the aggregate source-set budget
// (40 KB raw, 10 KB gzipped in the tools' specimen SVG envelope). It then lints
// the other games' icon directories (GAME_ICON_DIRS in lib.mjs) by the same rules,
// and refuses a file there named like a Duel icon: a reused drawing is referenced,
// not copied. Duel's names.ts and budget checks do not count those files.
// It needs no browser and runs in well under a second, so it does not go through
// tools/heavy.py.
//
// Rules (STYLE.md has the reasons): parse, root, prolog, comment, element,
// attr, paint, raw-colour, convention, implicit-fill, halo, snap, feature,
// live-area, gradient, id-prefix, id-collision, ref, budget, names,
// sprite-budget, missing.

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import zlib from 'node:zlib';
import {
  GAME_ICON_DIRS,
  ICON_DIR,
  TOKENS_CSS,
  inherited,
  isMain,
  loadSets,
  parseXml,
  printHeader,
  readIconNames,
  setFiles,
  shown,
  svgParts
} from './lib.mjs';

// Legacy names: these cap the complete original icon set serialized with the
// tools' specimen symbol/SVG wrappers, not shipped client markup or instances.
const SPRITE_RAW_MAX = 40 * 1024;
const SPRITE_GZIP_MAX = 10 * 1024;

const DEFAULT_RULES = {
  md: {
    viewBox: '0 0 24 24',
    budget: 1228,
    gradients: 0,
    snap: true,
    tokens: false,
    rawColors: false
  },
  sm: {
    viewBox: '0 0 12 12',
    budget: 1228,
    gradients: 0,
    snap: true,
    tokens: false,
    rawColors: false
  }
};

const SHAPES = new Set(['path', 'circle', 'ellipse', 'rect', 'line', 'polyline', 'polygon']);
const ALLOWED = new Set([...SHAPES, 'svg', 'g', 'defs', 'clipPath', 'mask', 'use']);
const GRADIENT = new Set(['linearGradient', 'radialGradient', 'stop']);
const FORBIDDEN = {
  text: 'no text: draw letters and numerals as outlines',
  tspan: 'no text: draw letters and numerals as outlines',
  textPath: 'no text: draw letters and numerals as outlines',
  image: 'no raster images inside a vector',
  filter: 'no filters (blur and shadows smear at icon sizes and cost render time)',
  foreignObject: 'no foreignObject',
  script: 'no scripts',
  style: 'no <style>: styles inside an inline SVG can affect the whole page',
  pattern: 'no SVG patterns: card patterns are CSS gradients',
  title: 'no <title>: accessible names come from the i18n tables',
  desc: 'no <desc>: accessible names come from the i18n tables',
  metadata: 'no metadata: editor leftovers cost bytes',
  symbol: 'no <symbol>: each source is a standalone SVG; only specimens add symbol wrappers',
  a: 'no links'
};
const PAINT_ATTRS = ['fill', 'stroke', 'color', 'stop-color', 'flood-color', 'lighting-color'];
const NAMED = new Set(
  (
    'black white red green blue yellow orange purple gray grey silver maroon olive lime aqua ' +
    'teal navy fuchsia brown pink gold ivory beige tan khaki crimson coral salmon indigo violet'
  ).split(' ')
);
const NUMERIC_ATTRS = [
  'x',
  'y',
  'x1',
  'y1',
  'x2',
  'y2',
  'cx',
  'cy',
  'r',
  'rx',
  'ry',
  'width',
  'height',
  'points',
  'd',
  'stroke-width',
  'transform'
];

const norm = (v) =>
  String(v)
    .replace(/\s+/g, ' ')
    .replace(/\s*,\s*/g, ', ')
    .trim();

function tokenNames() {
  const css = fs.readFileSync(TOKENS_CSS, 'utf8');
  return new Set([...css.matchAll(/(--[\w-]+)\s*:/g)].map((m) => m[1]));
}

/** Lint rules for one file: defaults by grid, then the set's rules, then its override. */
export function rulesFor(id, setName, sets = loadSets().sets) {
  const variant = id.endsWith('-sm') ? 'sm' : 'md';
  const set = setName ? sets[setName] : Object.values(sets).find((s) => ownsId(s, id));
  return {
    variant,
    ...DEFAULT_RULES[variant],
    ...(set?.rules ?? {}),
    ...(set?.overrides?.[id] ?? {})
  };
}

function ownsId(set, id) {
  return set.icons.some((e) => e.id === id || e.sm === id);
}

function rawColour(v) {
  const s = String(v).trim().toLowerCase();
  if (/#[0-9a-f]{3,8}\b/.test(s)) return true;
  if (/\b(rgba?|hsla?|hwb|lab|lch|oklab|oklch|color)\(/.test(s)) return true;
  return NAMED.has(s);
}

/** Why a paint value breaks the convention, or null when it is fine. */
function paintProblem(attr, value, rules, tokens) {
  const v = norm(value);
  if (v === 'none') return null;
  if (rules.rawColors) return null;
  if (v === 'var(--i1, currentColor)' || v === 'var(--i2, transparent)') return null;
  if (attr === 'stroke' && v === 'var(--i-halo, transparent)') return null;
  if (v === 'currentColor' || v === 'currentcolor') {
    return [
      'convention',
      `${attr}="${value}": write var(--i1, currentColor) so a caller can retint it`
    ];
  }
  const m = /^var\((--[\w-]+)(?:, (.+))?\)$/.exec(v);
  if (m) {
    const [, name, fallback] = m;
    if (name === '--i1')
      return ['convention', `--i1 must fall back to currentColor, not ${fallback ?? 'nothing'}`];
    if (name === '--i2')
      return ['convention', `--i2 must fall back to transparent, not ${fallback ?? 'nothing'}`];
    if (name === '--i-halo') {
      return attr === 'stroke'
        ? ['convention', `--i-halo must fall back to transparent, not ${fallback ?? 'nothing'}`]
        : ['halo', '--i-halo is for the halo stroke only, never a fill'];
    }
    if (rules.tokens && tokens.has(name)) {
      if (!fallback || /^(currentColor|transparent|none)$/.test(fallback)) return null;
      return ['raw-colour', `${attr}="${value}": a token needs no raw fallback`];
    }
    return ['paint', `${attr}="${value}": ${name} is not a paint this file may use`];
  }
  if (rawColour(v))
    return [
      'raw-colour',
      `${attr}="${value}": raw colour; use var(--i1, currentColor) or var(--i2, transparent)`
    ];
  if (/^url\(/.test(v)) {
    return rules.gradients > 0
      ? null
      : ['paint', `${attr}="${value}": paint servers are for ornaments only`];
  }
  return ['paint', `${attr}="${value}" is not an allowed paint`];
}

// --- geometry ---------------------------------------------------------------

const nums = (s) =>
  [...String(s).matchAll(/-?(?:\d+\.?\d*|\.\d+)(?:e[-+]?\d+)?/gi)].map((m) => Number(m[0]));

/** Points that bound a path (end and control points, plus arc samples). */
function pathPoints(d) {
  const pts = [];
  const tokens = [
    ...String(d).matchAll(/([MmLlHhVvCcSsQqTtAaZz])|(-?(?:\d+\.?\d*|\.\d+)(?:e[-+]?\d+)?)/g)
  ];
  let cmd = null;
  let x = 0;
  let y = 0;
  let sx = 0;
  let sy = 0;
  let k = 0;
  const next = () => Number(tokens[k++]?.[2]);
  const arity = { M: 2, L: 2, H: 1, V: 1, C: 6, S: 4, Q: 4, T: 2, A: 7, Z: 0 };
  while (k < tokens.length) {
    if (tokens[k][1]) {
      cmd = tokens[k][1];
      k++;
      if (cmd === 'Z' || cmd === 'z') {
        x = sx;
        y = sy;
        continue;
      }
    }
    if (!cmd) return pts;
    const up = cmd.toUpperCase();
    const rel = cmd !== up;
    const n = arity[up];
    if (k + n > tokens.length || tokens.slice(k, k + n).some((t) => t[1])) return pts;
    const a = Array.from({ length: n }, next);
    const ox = rel ? x : 0;
    const oy = rel ? y : 0;
    if (up === 'M' || up === 'L' || up === 'T') {
      x = a[0] + ox;
      y = a[1] + oy;
      if (up === 'M') {
        sx = x;
        sy = y;
        cmd = rel ? 'l' : 'L';
      }
      pts.push([x, y]);
    } else if (up === 'H') {
      x = a[0] + ox;
      pts.push([x, y]);
    } else if (up === 'V') {
      y = a[0] + oy;
      pts.push([x, y]);
    } else if (up === 'C' || up === 'S' || up === 'Q') {
      for (let j = 0; j < n; j += 2) pts.push([a[j] + ox, a[j + 1] + oy]);
      x = a[n - 2] + ox;
      y = a[n - 1] + oy;
    } else if (up === 'A') {
      const ex = a[5] + ox;
      const ey = a[6] + oy;
      pts.push(...arcSamples(x, y, a[0], a[1], a[2], a[3], a[4], ex, ey));
      x = ex;
      y = ey;
    }
  }
  return pts;
}

/** Samples an SVG arc (endpoint form, SVG 1.1 appendix F.6.5). */
function arcSamples(x1, y1, rx, ry, phiDeg, large, sweep, x2, y2) {
  rx = Math.abs(rx);
  ry = Math.abs(ry);
  if (!rx || !ry) return [[x2, y2]];
  const phi = (phiDeg * Math.PI) / 180;
  const cos = Math.cos(phi);
  const sin = Math.sin(phi);
  const dx = (x1 - x2) / 2;
  const dy = (y1 - y2) / 2;
  const x1p = cos * dx + sin * dy;
  const y1p = -sin * dx + cos * dy;
  const lambda = (x1p * x1p) / (rx * rx) + (y1p * y1p) / (ry * ry);
  if (lambda > 1) {
    rx *= Math.sqrt(lambda);
    ry *= Math.sqrt(lambda);
  }
  const num = rx * rx * ry * ry - rx * rx * y1p * y1p - ry * ry * x1p * x1p;
  const den = rx * rx * y1p * y1p + ry * ry * x1p * x1p;
  let c = Math.sqrt(Math.max(0, num / den));
  if (Boolean(large) === Boolean(sweep)) c = -c;
  const cxp = (c * rx * y1p) / ry;
  const cyp = (-c * ry * x1p) / rx;
  const cx = cos * cxp - sin * cyp + (x1 + x2) / 2;
  const cy = sin * cxp + cos * cyp + (y1 + y2) / 2;
  const ang = (ux, uy, vx, vy) => Math.atan2(ux * vy - uy * vx, ux * vx + uy * vy);
  const t1 = ang(1, 0, (x1p - cxp) / rx, (y1p - cyp) / ry);
  let dt = ang((x1p - cxp) / rx, (y1p - cyp) / ry, (-x1p - cxp) / rx, (-y1p - cyp) / ry);
  if (!sweep && dt > 0) dt -= 2 * Math.PI;
  if (sweep && dt < 0) dt += 2 * Math.PI;
  const out = [];
  for (let s = 1; s <= 24; s++) {
    const t = t1 + (dt * s) / 24;
    out.push([
      cx + rx * Math.cos(t) * cos - ry * Math.sin(t) * sin,
      cy + rx * Math.cos(t) * sin + ry * Math.sin(t) * cos
    ]);
  }
  return out;
}

/** The bounding box of one shape in user units, or null when unknown. */
function shapeBox(el) {
  const a = el.attrs;
  const n = (k) => Number(a[k] ?? 0);
  let pts = [];
  switch (el.name) {
    case 'rect':
      pts = [
        [n('x'), n('y')],
        [n('x') + n('width'), n('y') + n('height')]
      ];
      break;
    case 'circle':
      pts = [
        [n('cx') - n('r'), n('cy') - n('r')],
        [n('cx') + n('r'), n('cy') + n('r')]
      ];
      break;
    case 'ellipse':
      pts = [
        [n('cx') - n('rx'), n('cy') - n('ry')],
        [n('cx') + n('rx'), n('cy') + n('ry')]
      ];
      break;
    case 'line':
      pts = [
        [n('x1'), n('y1')],
        [n('x2'), n('y2')]
      ];
      break;
    case 'polyline':
    case 'polygon': {
      const v = nums(a.points ?? '');
      for (let j = 0; j + 1 < v.length; j += 2) pts.push([v[j], v[j + 1]]);
      break;
    }
    case 'path':
      pts = pathPoints(a.d ?? '');
      break;
    default:
      return null;
  }
  if (pts.length === 0) return null;
  const xs = pts.map((p) => p[0]);
  const ys = pts.map((p) => p[1]);
  return { x0: Math.min(...xs), y0: Math.min(...ys), x1: Math.max(...xs), y1: Math.max(...ys) };
}

// --- the linter -------------------------------------------------------------

/**
 * Lints files. Each item is { file, id?, set? }. Returns a list of problems
 * { file, line, level: 'error' | 'warn', rule, message } and per-file byte
 * counts, so other tools (shoot.mjs) can reuse it.
 */
export function lintFiles(items, { sets = loadSets().sets } = {}) {
  const problems = [];
  const tokens = tokenNames();
  const idOwner = new Map(); // id -> file
  // `at` is a line number or an element; an element also names itself as
  // "<name> #n" (its position in document order), since most icon files
  // are a single line.
  const report = (file, at, level, rule, message) =>
    typeof at === 'object'
      ? problems.push({
          file,
          line: at.line,
          level,
          rule,
          message: `<${at.name}> #${at.index}: ${message}`
        })
      : problems.push({ file, line: at, level, rule, message });
  const bytes = {};

  for (const item of items) {
    const file = item.file;
    const id = item.id ?? path.basename(file, '.svg');
    if (!fs.existsSync(file)) {
      report(file, 0, 'warn', 'missing', `no file yet for "${id}"`);
      continue;
    }
    const rules = rulesFor(id, item.set, sets);
    const src = fs.readFileSync(file, 'utf8');
    bytes[file] = Buffer.byteLength(src);
    const doc = parseXml(src);
    for (const e of doc.errors) report(file, e.line, 'error', 'parse', e.message);
    if (!doc.root) continue;
    const sprite = [ICON_DIR, ...GAME_ICON_DIRS].some(
      (d) => path.resolve(path.dirname(file)) === path.resolve(d)
    );

    // Root.
    const root = doc.root;
    if (root.name !== 'svg') report(file, root, 'error', 'root', 'the root element must be <svg>');
    if (root.attrs.xmlns !== 'http://www.w3.org/2000/svg') {
      report(file, root, 'error', 'root', 'the root needs xmlns="http://www.w3.org/2000/svg"');
    }
    const vb = root.attrs.viewBox;
    if (!vb) report(file, root, 'error', 'root', 'no viewBox');
    else if (rules.viewBox !== 'any' && norm(vb) !== norm(rules.viewBox)) {
      report(file, root, 'error', 'root', `viewBox="${vb}", expected "${rules.viewBox}"`);
    }
    if (sprite && ('width' in root.attrs || 'height' in root.attrs)) {
      report(
        file,
        root,
        'error',
        'root',
        'no width or height on the root: icons are sized in em by CSS'
      );
    }
    for (const p of doc.prolog)
      report(file, p.line, 'warn', 'prolog', 'drop the XML declaration or doctype');
    for (const c of doc.comments) {
      report(
        file,
        c.line,
        sprite ? 'error' : 'warn',
        'comment',
        sprite
          ? 'comments ship in inline icon markup; keep notes in art/vector/log/'
          : 'comments ship to the browser; keep notes in art/vector/log/'
      );
    }

    // Grid-derived limits (unknown for viewBox "any").
    const box = vb ? nums(vb) : [0, 0, 24, 24];
    const knownGrid = rules.viewBox !== 'any';
    const unit = knownGrid ? box[2] / 24 : 1; // 1 on the md grid, 0.5 on the 12 grid
    const minFeature = 2 * unit;
    const live = [box[0] + 2 * unit, box[0] + box[2] - 2 * unit];

    const ids = new Set();
    let gradients = 0;
    let usesI1 = false;
    for (const el of doc.elements) {
      for (const [k, v] of Object.entries(el.attrs)) {
        if (k === 'id') {
          ids.add(v);
          if (!v.startsWith(`${id}-`)) {
            report(file, el, 'error', 'id-prefix', `id="${v}" must start with "${id}-"`);
          }
          const owner = idOwner.get(v);
          if (owner && owner !== file) {
            report(file, el, 'error', 'id-collision', `id="${v}" is also used in ${shown(owner)}`);
          } else idOwner.set(v, file);
        }
      }
    }

    for (const el of doc.elements) {
      const name = el.name;
      const inDefs = (() => {
        for (let e = el.parent; e; e = e.parent) {
          if (
            e.name === 'defs' ||
            e.name === 'clipPath' ||
            e.name === 'mask' ||
            GRADIENT.has(e.name)
          )
            return true;
        }
        return false;
      })();
      if (
        FORBIDDEN[name] ||
        name.startsWith('fe') ||
        name.startsWith('animate') ||
        name === 'set'
      ) {
        report(file, el, 'error', 'element', `${FORBIDDEN[name] ?? 'not allowed in icons'}`);
        continue;
      }
      if (GRADIENT.has(name)) {
        if (name !== 'stop') gradients++;
      } else if (!ALLOWED.has(name)) {
        report(file, el, 'error', 'element', `not on the allowed list`);
      }
      if (el.text) report(file, el, 'error', 'element', `has text content`);

      // Attributes.
      for (const [k, v] of Object.entries(el.attrs)) {
        if (k === 'style')
          report(file, el, 'error', 'attr', 'no style attribute: use presentation attributes');
        else if (k === 'class')
          report(file, el, 'error', 'attr', 'no class attribute: the page CSS would reach it');
        else if (/^on/i.test(k)) report(file, el, 'error', 'attr', `no event handler (${k})`);
        else if (k === 'xlink:href') report(file, el, 'error', 'attr', 'use href, not xlink:href');
        else if (k.startsWith('xmlns:') && el === root)
          report(file, el, 'warn', 'attr', `${k}: editor namespace, drop it`);
        else if (/^(sodipodi|inkscape|sketch|serif|data-)/.test(k))
          report(file, el, 'error', 'attr', `${k}: editor leftover`);
        if (PAINT_ATTRS.includes(k)) {
          if (/var\(--i1\b/.test(v)) usesI1 = true;
          const p = paintProblem(k, v, rules, tokens);
          if (p) report(file, el, 'error', p[0], p[1]);
        }
        if (k === 'transform' && /matrix|skew/.test(v)) {
          report(file, el, 'error', 'attr', 'transform: only translate, rotate and scale');
        }
        const refs = [...String(v).matchAll(/url\(\s*['"]?#([^'")]+)['"]?\s*\)/g)].map((m) => m[1]);
        if (k === 'href' && v.startsWith('#')) refs.push(v.slice(1));
        for (const r of refs) {
          if (!ids.has(r)) report(file, el, 'error', 'ref', `#${r} is not defined in this file`);
        }
      }

      if (!SHAPES.has(name) || el === root) continue;
      if (inDefs && name !== 'path' && name !== 'circle' && name !== 'rect') continue;

      // Paint inheritance: a shape with no fill anywhere up the tree is black.
      const fill = inherited(el, 'fill');
      const stroke = inherited(el, 'stroke');
      const halo = stroke !== undefined && norm(stroke) === 'var(--i-halo, transparent)';
      if (!inDefs) {
        if (fill === undefined && name !== 'line') {
          report(
            file,
            el,
            'error',
            'implicit-fill',
            'no fill anywhere up the tree, so any area it encloses paints black; set fill (fill="none" for a stroke-only shape)'
          );
        }
        if (name === 'line' && (stroke === undefined || stroke === 'none')) {
          report(file, el, 'error', 'implicit-fill', 'a line without a stroke draws nothing');
        }
        if (halo && inherited(el, 'paint-order') !== 'stroke') {
          report(
            file,
            el,
            'error',
            'halo',
            'the halo stroke needs paint-order="stroke" so it sits under the fill'
          );
        }
      }

      // Snapping.
      if (rules.snap) {
        for (const k of NUMERIC_ATTRS) {
          if (!(k in el.attrs)) continue;
          let values = nums(el.attrs[k]);
          if (k === 'd') values = values.filter((x) => Number.isFinite(x));
          const off = values.find((x) => Math.abs(x * 2 - Math.round(x * 2)) > 1e-9);
          if (off !== undefined) {
            report(file, el, 'error', 'snap', `${k} has ${off}; snap coordinates to 0.5`);
            break;
          }
        }
      }

      // Feature sizes and the live area (known grids only, no transforms).
      if (!knownGrid || inDefs) continue;
      const stroked = stroke !== undefined && stroke !== 'none' && !halo;
      const sw = Number(inherited(el, 'stroke-width') ?? 1);
      if (stroked && sw + 1e-9 < minFeature) {
        report(
          file,
          el,
          'error',
          'feature',
          `stroke-width ${sw} is under the minimum feature ${minFeature}`
        );
      }
      const a = el.attrs;
      const small = (v) => v !== undefined && Number(v) > 0 && Number(v) + 1e-9 < minFeature;
      if (name === 'rect' && (small(a.width) || small(a.height))) {
        report(
          file,
          el,
          'error',
          'feature',
          `rect ${a.width}x${a.height} is under the minimum feature ${minFeature}`
        );
      }
      if (name === 'circle' && small(2 * Number(a.r))) {
        report(
          file,
          el,
          'error',
          'feature',
          `circle r=${a.r} is under the minimum feature ${minFeature}`
        );
      }
      if (name === 'ellipse' && (small(2 * Number(a.rx)) || small(2 * Number(a.ry)))) {
        report(
          file,
          el,
          'error',
          'feature',
          `ellipse ${a.rx}x${a.ry} is under the minimum feature ${minFeature}`
        );
      }
      let transformed = false;
      for (let e = el; e; e = e.parent) if (e.attrs.transform) transformed = true;
      if (transformed) continue;
      if (rules.liveArea === false) continue;
      const b = shapeBox(el);
      if (!b) continue;
      const pad = stroked ? sw / 2 : 0;
      if (
        b.x0 - pad < live[0] - 1e-9 ||
        b.y0 - pad < live[0] - 1e-9 ||
        b.x1 + pad > live[1] + 1e-9 ||
        b.y1 + pad > live[1] + 1e-9
      ) {
        const r = (v) => Math.round(v * 100) / 100;
        report(
          file,
          el,
          'warn',
          'live-area',
          `reaches ${r(b.x0 - pad)},${r(b.y0 - pad)} to ${r(b.x1 + pad)},${r(b.y1 + pad)}; the live area is ${live[0]} to ${live[1]}`
        );
      }
    }

    if (!usesI1 && !rules.rawColors && !rules.tokens) {
      report(
        file,
        root,
        'error',
        'convention',
        'never uses var(--i1, currentColor): the icon cannot follow the text colour'
      );
    }
    if (gradients > (rules.gradients ?? 0)) {
      report(
        file,
        root,
        'error',
        'gradient',
        `${gradients} gradient(s); this file may have ${rules.gradients ?? 0}`
      );
    }
    if (bytes[file] > rules.budget) {
      report(
        file,
        0,
        'error',
        'budget',
        `${bytes[file]} bytes, over the ${rules.budget}-byte budget`
      );
    }
  }
  return { problems, bytes };
}

/** Aggregate source-set size in the tools' canonical specimen SVG envelope.
 *  Keep this historical API/rule name for tool callers; the client no longer
 *  ships a sprite, and its direct SVG instances are not measured here. */
export function spriteTotals(names = readIconNames()) {
  const symbols = names
    .filter((n) => fs.existsSync(path.join(ICON_DIR, `${n}.svg`)))
    .map((n) => {
      const { viewBox, inner } = svgParts(fs.readFileSync(path.join(ICON_DIR, `${n}.svg`), 'utf8'));
      return `<symbol id="i-${n}" viewBox="${viewBox}">${inner}</symbol>`;
    })
    .join('');
  const markup = `<svg aria-hidden="true" focusable="false" style="position:absolute;width:0;height:0;overflow:hidden" xmlns="http://www.w3.org/2000/svg">${symbols}</svg>`;
  return { raw: Buffer.byteLength(markup), gzip: zlib.gzipSync(markup, { level: 9 }).length };
}

async function main() {
  const { values: o, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      set: { type: 'string', multiple: true, default: [] },
      strict: { type: 'boolean', default: false },
      json: { type: 'boolean', default: false },
      quiet: { type: 'boolean', default: false },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (o.help) {
    printHeader(import.meta.url);
    return;
  }
  const sets = loadSets().sets;
  const items = [];
  for (const name of o.set) {
    if (!sets[name]) throw new Error(`no set "${name}"; sets: ${Object.keys(sets).join(', ')}`);
    for (const f of setFiles({ name, ...sets[name] }))
      items.push({ file: f.file, id: f.id, set: name });
  }
  for (const p of positionals) {
    const st = fs.statSync(p, { throwIfNoEntry: false });
    if (!st) throw new Error(`${p} does not exist`);
    if (st.isDirectory()) {
      for (const f of fs.readdirSync(p).sort())
        if (f.endsWith('.svg')) items.push({ file: path.join(p, f) });
    } else items.push({ file: p });
  }
  const whole = o.set.length === 0 && positionals.length === 0;
  const gameItems = [];
  if (whole) {
    for (const f of fs.readdirSync(ICON_DIR).sort())
      if (f.endsWith('.svg')) items.push({ file: path.join(ICON_DIR, f) });
    for (const dir of GAME_ICON_DIRS.filter((d) => fs.existsSync(d)))
      for (const f of fs.readdirSync(dir).sort())
        if (f.endsWith('.svg')) gameItems.push({ file: path.join(dir, f) });
  }

  const { problems } = lintFiles([...items, ...gameItems], { sets });
  if (whole) {
    const names = readIconNames();
    const onDisk = new Set(items.map((i) => path.basename(i.file, '.svg')));
    for (const { file } of gameItems) {
      if (onDisk.has(path.basename(file, '.svg')))
        problems.push({
          file,
          line: 0,
          level: 'error',
          rule: 'names',
          message: `"${path.basename(file, '.svg')}" copies a Duel icon name; reference the Duel file from sets.json instead`
        });
    }
    for (const n of names) {
      if (!onDisk.has(n))
        problems.push({
          file: path.join(ICON_DIR, `${n}.svg`),
          line: 0,
          level: 'error',
          rule: 'names',
          message: `names.ts lists "${n}" but there is no file`
        });
    }
    for (const n of onDisk) {
      if (!names.includes(n))
        problems.push({
          file: path.join(ICON_DIR, `${n}.svg`),
          line: 0,
          level: 'error',
          rule: 'names',
          message: `"${n}" is not listed in names.ts`
        });
    }
    const t = spriteTotals(names);
    if (t.raw > SPRITE_RAW_MAX)
      problems.push({
        file: ICON_DIR,
        line: 0,
        level: 'error',
        rule: 'sprite-budget',
        message: `icon source set (specimen envelope) is ${t.raw} bytes raw, over ${SPRITE_RAW_MAX}`
      });
    if (t.gzip > SPRITE_GZIP_MAX)
      problems.push({
        file: ICON_DIR,
        line: 0,
        level: 'error',
        rule: 'sprite-budget',
        message: `icon source set (specimen envelope) is ${t.gzip} bytes gzipped, over ${SPRITE_GZIP_MAX}`
      });
    console.error(
      `lint: icon source set (specimen envelope) ${t.raw} bytes raw (max ${SPRITE_RAW_MAX}), ${t.gzip} gzipped (max ${SPRITE_GZIP_MAX})`
    );
  }

  const shownProblems = o.quiet ? problems.filter((p) => p.level === 'error') : problems;
  if (o.json) {
    console.log(
      JSON.stringify(
        shownProblems.map((p) => ({ ...p, file: shown(p.file) })),
        null,
        2
      )
    );
  } else {
    for (const p of shownProblems)
      console.log(`${shown(p.file)}:${p.line}: ${p.level} ${p.rule}: ${p.message}`);
  }
  const errors = problems.filter((p) => p.level === 'error').length;
  const warns = problems.length - errors;
  const byRule = {};
  for (const p of problems) byRule[p.rule] = (byRule[p.rule] ?? 0) + 1;
  const filesWith = new Set(problems.filter((p) => p.level === 'error').map((p) => p.file)).size;
  console.error(
    `lint: ${items.length + gameItems.length} files, ${errors} errors in ${filesWith} files, ${warns} warnings` +
      (problems.length
        ? ` (${Object.entries(byRule)
            .map(([r, n]) => `${r} ${n}`)
            .join(', ')})`
        : '')
  );
  process.exitCode = errors > 0 || (o.strict && warns > 0) ? 1 : 0;
}

if (isMain(import.meta.url)) {
  main().catch((err) => {
    console.error(`${err.stack ?? err}`);
    process.exit(2);
  });
}
