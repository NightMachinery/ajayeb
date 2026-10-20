// Shared helpers for the vector tools in tools/art/vector: repository paths,
// the set definitions in art/vector/sets.json, a small XML reader for the
// icon sources, specimen-only <symbol> markup, and one headless Chrome page
// for the tools. The client (web/src/lib/icons/sources.ts) renders local shapes;
// its runtime has no shared sprite. This module supports the tools
// that take screenshots. Dependency-free: Node 24 only.

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const REPO = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
export const ICON_DIR = path.join(REPO, 'web/src/lib/icons/svg');
export const NAMES_TS = path.join(REPO, 'web/src/lib/icons/names.ts');
export const TOKENS_CSS = path.join(REPO, 'web/src/lib/styles/tokens.css');
export const SETS_JSON = path.join(REPO, 'art/vector/sets.json');
export const FONT_DIR = path.join(REPO, 'web/src/lib/fonts');

/** The private run directory (never inside the repo); AJAYEB_RUN_DIR overrides it. */
export const RUN_DIR = process.env.AJAYEB_RUN_DIR || path.join(os.homedir(), 'tmp/agents/ajayeb');
/** Where renders, specimens, peeps, blind sheets and keys go. */
export const ART_WORK = path.join(RUN_DIR, 'art-work/vector');

/** A path for messages: repo-relative inside the repo, ~-relative under home. */
export function shown(file) {
  const abs = path.resolve(file);
  const rel = path.relative(REPO, abs);
  if (!rel.startsWith('..') && !path.isAbsolute(rel)) return rel;
  const home = os.homedir();
  return abs.startsWith(home + path.sep) ? `~/${path.relative(home, abs)}` : abs;
}

export function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
  return dir;
}

export const list = (s) =>
  String(s ?? '')
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean);

export const escapeHtml = (s) =>
  String(s).replace(
    /[&<>"]/g,
    (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]
  );

// ---------------------------------------------------------------------------
// Sets

/**
 * Reads art/vector/sets.json. Each set has a title, optional lint rules and
 * a list of icons; each icon is { id, sm?, file?, ink?, i2?, tile?, band?,
 * badge?, chip? } (see art/vector/STYLE.md, "sets.json").
 */
export function loadSets() {
  const raw = JSON.parse(fs.readFileSync(SETS_JSON, 'utf8'));
  return raw;
}

/** The repo path of an icon id: its `file` from sets.json, else the icon source directory. */
export function iconFile(id, entry) {
  if (entry?.file && entry.id === id) return path.join(REPO, entry.file);
  if (entry?.smFile && entry.sm === id) return path.join(REPO, entry.smFile);
  return path.join(ICON_DIR, `${id}.svg`);
}

/** Resolves a set by name, or throws listing the names that exist. */
export function getSet(name) {
  const sets = loadSets().sets;
  const set = sets[name];
  if (!set) {
    throw new Error(
      `no set "${name}" in art/vector/sets.json; sets: ${Object.keys(sets).join(', ')}`
    );
  }
  return { name, ...set };
}

/**
 * Every file of a set, md and sm, in sets.json order:
 * [{ id, file, exists, entry, variant: 'md' | 'sm' }].
 */
export function setFiles(set) {
  const out = [];
  for (const entry of set.icons) {
    out.push({ id: entry.id, file: iconFile(entry.id, entry), entry, variant: 'md' });
    if (entry.sm) out.push({ id: entry.sm, file: iconFile(entry.sm, entry), entry, variant: 'sm' });
  }
  for (const f of out) f.exists = fs.existsSync(f.file);
  return out;
}

/** The icon names listed in web/src/lib/icons/names.ts (read as text). */
export function readIconNames() {
  const src = fs.readFileSync(NAMES_TS, 'utf8');
  const body = /ICON_NAMES\s*=\s*\[([\s\S]*?)\]\s*as const/.exec(src)?.[1] ?? '';
  return [...body.replace(/\/\/.*$/gm, '').matchAll(/'([^']+)'/g)].map((m) => m[1]);
}

// ---------------------------------------------------------------------------
// XML

/**
 * A small, forgiving XML reader for hand-written SVG. Returns
 * { root, elements, comments, prolog, errors }, where each element is
 * { name, attrs, line, parent, children, depth }. It does not resolve
 * entities or namespaces; that is enough for linting icon sources.
 */
export function parseXml(src) {
  const elements = [];
  const comments = [];
  const prolog = [];
  const errors = [];
  const stack = [];
  let root = null;
  let i = 0;
  let line = 1;
  const advance = (to) => {
    for (let k = i; k < to; k++) if (src.charCodeAt(k) === 10) line++;
    i = to;
  };
  while (i < src.length) {
    const lt = src.indexOf('<', i);
    if (lt < 0) {
      if (src.slice(i).trim()) errors.push({ line, message: 'text after the root element' });
      break;
    }
    const text = src.slice(i, lt);
    if (text.trim() && stack.length > 0) {
      stack.at(-1).text = (stack.at(-1).text ?? '') + text.trim();
    } else if (text.trim()) {
      errors.push({ line, message: 'text outside any element' });
    }
    advance(lt);
    if (src.startsWith('<!--', i)) {
      const end = src.indexOf('-->', i);
      if (end < 0) {
        errors.push({ line, message: 'unterminated comment' });
        break;
      }
      comments.push({ line, text: src.slice(i + 4, end) });
      advance(end + 3);
    } else if (src.startsWith('<?', i) || src.startsWith('<!', i)) {
      const end = src.indexOf('>', i);
      if (end < 0) {
        errors.push({ line, message: 'unterminated declaration' });
        break;
      }
      prolog.push({ line, text: src.slice(i, end + 1) });
      advance(end + 1);
    } else if (src.startsWith('</', i)) {
      const m = /^<\/\s*([\w:.-]+)\s*>/.exec(src.slice(i));
      if (!m) {
        errors.push({ line, message: 'malformed closing tag' });
        break;
      }
      const open = stack.pop();
      if (!open || open.name !== m[1]) {
        errors.push({ line, message: `</${m[1]}> does not close <${open?.name ?? 'nothing'}>` });
      }
      advance(i + m[0].length);
    } else {
      const m = /^<([\w:.-]+)((?:\s+[\w:.-]+\s*=\s*(?:"[^"]*"|'[^']*'))*)\s*(\/?)>/.exec(
        src.slice(i)
      );
      if (!m) {
        errors.push({ line, message: 'malformed tag' });
        break;
      }
      const attrs = {};
      for (const a of m[2].matchAll(/([\w:.-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')/g)) {
        if (a[1] in attrs) errors.push({ line, message: `<${m[1]}> repeats ${a[1]}` });
        attrs[a[1]] = a[2] ?? a[3];
      }
      const parent = stack.at(-1) ?? null;
      const el = {
        name: m[1],
        attrs,
        line,
        parent,
        children: [],
        depth: stack.length,
        index: elements.length + 1
      };
      if (parent) parent.children.push(el);
      else if (root) errors.push({ line, message: 'more than one root element' });
      else root = el;
      elements.push(el);
      advance(i + m[0].length);
      if (!m[3]) stack.push(el);
    }
  }
  if (stack.length > 0) errors.push({ line, message: `<${stack.at(-1).name}> is never closed` });
  return { root, elements, comments, prolog, errors };
}

/** The value of an attribute on el or its nearest ancestor that has it. */
export function inherited(el, name) {
  for (let e = el; e; e = e.parent) if (name in e.attrs) return e.attrs[name];
  return undefined;
}

// ---------------------------------------------------------------------------
// Specimen-only symbol markup

/** The viewBox and inner markup of a standalone <svg> source, before client rendering optimizations. */
export function svgParts(raw) {
  const open = /<svg\b[^>]*>/.exec(raw);
  const closeAt = raw.lastIndexOf('</svg>');
  if (!open || closeAt < 0) throw new Error('not a standalone <svg>...</svg> document');
  const viewBox = /viewBox="([^"]+)"/.exec(open[0])?.[1] ?? '0 0 24 24';
  return { viewBox, inner: raw.slice(open.index + open[0].length, closeAt).trim() };
}

/** Specimen-only <symbol id="i-<id>"> markup for each original source file. */
export function symbolsFor(files) {
  return files
    .filter((f) => f.exists)
    .map((f) => {
      const { viewBox, inner } = svgParts(fs.readFileSync(f.file, 'utf8'));
      return `<symbol id="i-${f.id}" viewBox="${viewBox}">${inner}</symbol>`;
    })
    .join('');
}

/** The aspect ratio (width / height) of a file's viewBox. */
export function aspectOf(file) {
  try {
    const vb = svgParts(fs.readFileSync(file, 'utf8'))
      .viewBox.split(/[\s,]+/)
      .map(Number);
    return vb[2] > 0 && vb[3] > 0 ? vb[2] / vb[3] : 1;
  } catch {
    return 1;
  }
}

/** tokens.css, for inlining into a specimen page. */
export function tokensCss() {
  return fs.readFileSync(TOKENS_CSS, 'utf8');
}

/** An @font-face rule that inlines one of the shipped WOFF2 fonts. */
export function fontFace(family, file, weight) {
  const data = fs.readFileSync(path.join(FONT_DIR, file)).toString('base64');
  return `@font-face{font-family:'${family}';font-weight:${weight};src:url(data:font/woff2;base64,${data}) format('woff2');}`;
}

/** A stable shuffle (mulberry32), so a blind sheet can be rebuilt from its seed. */
export function shuffled(items, seed) {
  let a = seed >>> 0;
  const rand = () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
  const out = [...items];
  for (let k = out.length - 1; k > 0; k--) {
    const j = Math.floor(rand() * (k + 1));
    [out[k], out[j]] = [out[j], out[k]];
  }
  return out;
}

// ---------------------------------------------------------------------------
// Chrome

/**
 * Launches one headless Chrome (tools/chrome/launch.mjs, under the heavy.py
 * browser lock), opens one isolated page, runs fn(shooter) and always closes
 * both. shooter.load(html) sets the document; shooter.capture({...}) writes
 * one PNG of the whole document.
 */
export async function withShooter(fn, { force = false } = {}) {
  const { launch } = await import('../../chrome/launch.mjs');
  const { CDP, evaluate, openPage } = await import('../../chrome/cdp.mjs');
  const chrome = await launch({ force });
  let browser = null;
  let page = null;
  try {
    browser = await CDP.connect(chrome.wsUrl);
    page = await openPage(browser);
    await page.send('Page.enable');
    const settle = () =>
      evaluate(page, async () => {
        await document.fonts.ready;
        await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
      });
    const shooter = {
      page,
      evaluate: (f, ...args) => evaluate(page, f, ...args),
      async load(html) {
        await page.send('Emulation.setDeviceMetricsOverride', {
          width: 4000,
          height: 1000,
          deviceScaleFactor: 1,
          mobile: false
        });
        const { frameTree } = await page.send('Page.getFrameTree');
        await page.send('Page.setDocumentContent', { frameId: frameTree.frame.id, html });
        await settle();
      },
      /**
       * Writes the whole document to `out` at `dpr`, with the page's theme
       * (data-theme on <html>) and an emulated vision deficiency ('none',
       * 'achromatopsia', 'deuteranopia', 'protanopia', 'tritanopia').
       * Resolves with { width, height, rects } in CSS px, where rects are the
       * boxes of the elements that carry data-rect="<name>".
       */
      async capture({ out, dpr = 1, theme = 'light', vision = 'none' }) {
        await evaluate(
          page,
          (th) => {
            document.documentElement.dataset.theme = th;
          },
          theme
        );
        await page.send('Emulation.setEmulatedVisionDeficiency', { type: vision });
        await page.send('Emulation.setDeviceMetricsOverride', {
          width: 4000,
          height: 1000,
          deviceScaleFactor: dpr,
          mobile: false
        });
        await settle();
        const m = await evaluate(page, () => {
          const r = document.body.getBoundingClientRect();
          const rects = {};
          for (const el of document.querySelectorAll('[data-rect]')) {
            const b = el.getBoundingClientRect();
            rects[el.dataset.rect] = { x: b.x, y: b.y, width: b.width, height: b.height };
          }
          return { width: Math.ceil(r.width), height: Math.ceil(r.height), rects };
        });
        await page.send('Emulation.setDeviceMetricsOverride', {
          width: m.width,
          height: m.height,
          deviceScaleFactor: dpr,
          mobile: false
        });
        await settle();
        const { data } = await page.send('Page.captureScreenshot', {
          format: 'png',
          fromSurface: true,
          clip: { x: 0, y: 0, width: m.width, height: m.height, scale: 1 }
        });
        ensureDir(path.dirname(path.resolve(out)));
        fs.writeFileSync(out, Buffer.from(data, 'base64'));
        await page.send('Emulation.setEmulatedVisionDeficiency', { type: 'none' });
        return m;
      }
    };
    return await fn(shooter);
  } finally {
    await page?.close();
    browser?.close();
    await chrome.close();
  }
}

/** An expected error with a message for the user, such as a bad flag. */
export class UsageError extends Error {
  name = 'UsageError';
}

/**
 * True for an error the tools raise on purpose: a UsageError, a plain Error
 * (every `throw new Error('--set NAME is required')` and the like), or a
 * flag that node:util parseArgs refused. Those are reported by their message
 * alone. Built-in subclasses (TypeError, RangeError, ReferenceError ...)
 * and anything that is not an Error are real crashes and keep the stack.
 */
export function isExpectedError(err) {
  if (!(err instanceof Error)) return false;
  if (err instanceof UsageError) return true;
  if (typeof err.code === 'string' && err.code.startsWith('ERR_PARSE_ARGS_')) return true;
  return Object.getPrototypeOf(err) === Error.prototype;
}

/**
 * Shared CLI error handling: a refused launch exits 2, any other error 1.
 * Expected errors print their message only, crashes their stack;
 * AJAYEB_STACK=1 prints the stack for every error.
 */
export async function runMain(main) {
  try {
    await main();
  } catch (err) {
    const { LaunchRefused } = await import('../../chrome/launch.mjs');
    const refused = err instanceof LaunchRefused;
    const brief = refused || (isExpectedError(err) && process.env.AJAYEB_STACK !== '1');
    console.error(brief ? `${err.message}` : `${err?.stack ?? err}`);
    process.exit(refused ? 2 : 1);
  }
}

/** Prints a tool's header comment (after the shebang) as its --help text. */
export function printHeader(metaUrl) {
  const lines = fs.readFileSync(fileURLToPath(metaUrl), 'utf8').split('\n').slice(1);
  const end = lines.findIndex((l) => !l.startsWith('//'));
  console.log(
    lines
      .slice(0, end)
      .map((l) => l.replace(/^\/\/ ?/, ''))
      .join('\n')
  );
}

/** True when the module at `metaUrl` is the script node was asked to run. */
export function isMain(metaUrl) {
  return !!process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(metaUrl);
}
