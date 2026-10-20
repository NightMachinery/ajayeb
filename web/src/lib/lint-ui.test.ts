// Unit tests for the two UI lints (web/scripts/lint-ui.mjs), run against
// small strings rather than real files. The script itself walks web/src and
// is wired into `pnpm -C web lint` (package.json); these tests cover the
// pure rule functions it exports. Living here (not under web/scripts/) is
// what gets it picked up by vitest's include glob (vite.config.ts).
import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import {
  findHardcodedText,
  findHardcodedTextViolations,
  findPhysicalCssViolations
} from '../../scripts/lint-ui.mjs';

describe('findPhysicalCssViolations', () => {
  it.each([
    ['<div class="ml-2">', 'ml-'],
    ['<div class="hover:mr-4 text-fg">', 'mr-'],
    ['<div class="pl-3">', 'pl-'],
    ['<div class="sm:pr-6">', 'pr-'],
    ['<div class="left-0 absolute">', 'left-'],
    ['<div class="right-2">', 'right-'],
    ['<p class="text-left">', 'text-left'],
    ['<p class="split:text-right">', 'text-right'],
    ['<div class="rounded-l-lg">', 'rounded-l'],
    ['<div class="rounded-r">', 'rounded-r'],
    ['<div class="border-l-2">', 'border-l'],
    ['<div class="border-r">', 'border-r'],
    ['<div class="scroll-ml-4">', 'scroll-ml-'],
    ['  margin-left: 4px;', 'margin-left'],
    ['  margin-right: 1rem;', 'margin-right'],
    ['  padding-left: 8px;', 'padding-left'],
    ['  padding-right: 8px;', 'padding-right'],
    ['  border-left: 1px solid red;', 'border-left'],
    ['  border-right: 1px solid red;', 'border-right'],
    ['  left: 0;', 'left:'],
    ['  right: 4px;', 'right:'],
    ['<div class="-ml-2">', 'negative ml-'],
    ['<span class="origin-left">', 'origin-left'],
    ['<span class="origin-top-right">', 'origin-top-right'],
    ['<img class="float-left" />', 'float-left']
  ])('flags %s (%s)', (line) => {
    expect(findPhysicalCssViolations(line).length).toBeGreaterThan(0);
  });

  it.each([
    ['<div class="ms-2 pe-4 text-start border-s rounded-s">', 'logical-only line'],
    ['<div class="rounded-large">', 'unrelated word starting with rounded-l'],
    ['<div class="border-lime">', 'unrelated word starting with border-l'],
    ['  color: var(--fg);', 'unrelated CSS property'],
    ['  /* turn left at the fork */', 'prose, not a declaration'],
    ['<div class="ml-2"> <!-- physical-ok: device chrome --> ', 'opted out with physical-ok']
  ])('does not flag %s (%s)', (line) => {
    expect(findPhysicalCssViolations(line)).toEqual([]);
  });
});

describe('findHardcodedTextViolations', () => {
  it.each([
    ['<button aria-label="Close the sheet">', 'literal aria-label'],
    ['<input title="Enter your name" />', 'literal title'],
    ['<img alt="A wonder" />', 'literal alt'],
    ['<input placeholder="Your name" />', 'literal placeholder'],
    ['<p>Your turn</p>', 'template text node'],
    ['<h1>Settings</h1>', 'heading text node']
  ])('flags %s (%s)', (line) => {
    expect(findHardcodedTextViolations(line).length).toBeGreaterThan(0);
  });

  it.each([
    ["<button aria-label={t('shell.close')}>", 'expression aria-label'],
    ["<p>{t('turn.player', { name })}</p>", 'expression text node'],
    ['<p>{count}</p>', 'bare expression, no letters'],
    ['<p>12</p>', 'digits only'],
    ['<code>pnpm install</code>', 'allow-listed code element'],
    ['<kbd>Ctrl</kbd>', 'allow-listed kbd element'],
    ['<p>Hi</p>', 'run of only 2 letters'],
    ['<div class="text-fg-muted">', 'an ordinary class attribute'],
    ['<p>Your turn</p> <!-- i18n-ok: fixture label -->', 'opted out with i18n-ok']
  ])('does not flag %s (%s)', (line) => {
    expect(findHardcodedTextViolations(line)).toEqual([]);
  });
});

describe('findHardcodedText (whole template)', () => {
  it('flags text on a line of its own, with its line number', () => {
    const src = '<p class="text-sm">\n  Your turn to build\n</p>';
    expect(findHardcodedText(src)).toMatchObject([{ line: 2, match: 'Your turn to build' }]);
  });

  it('flags text next to an expression', () => {
    expect(findHardcodedText('<p>{count} coins</p>')).toHaveLength(1);
    expect(findHardcodedText('<p>Score: {score}</p>')).toHaveLength(1);
  });

  it('flags a multi-line literal attribute only on its own line', () => {
    const src = '<button\n  type="button"\n  aria-label="Close it"\n>';
    expect(findHardcodedText(src)).toMatchObject([{ line: 3 }]);
  });

  it('ignores script and style blocks, comments, entities and expressions', () => {
    const src = [
      '<script lang="ts">',
      "  const label = 'Some text';",
      '  const f = (a: number) => a > 1 && a < 3;',
      '</script>',
      '<!-- A comment with words -->',
      "<p>{t('turn.player', { name: 'Ana' })}&nbsp;{n}</p>",
      '{#if open}<span>{label}</span>{/if}',
      "<button onclick={() => (x = x > 1 ? 0 : 1)}>{t('shell.close')}</button>",
      '<style>',
      '  .a > .b { color: red; }',
      '</style>'
    ].join('\n');
    expect(findHardcodedText(src)).toEqual([]);
  });

  it('allows multi-line code and kbd content, and i18n-ok per line', () => {
    const src = '<code>\n  pnpm install\n</code>\n<p>\n  Fixture label <!-- i18n-ok -->\n</p>';
    expect(findHardcodedText(src)).toEqual([]);
  });
});

// --- Rule 3: no native title attribute --------------------------------------
//
// Tooltips go through our own helper (ui/tooltip.svelte.ts, AGENTS.md
// "Tooltips"), never the native title attribute: it shows only on a mouse
// hover after a long delay, never on touch, and cannot be styled. This rule
// lives here rather than in scripts/lint-ui.mjs, so `make test-web` enforces
// it over every .svelte file under src/. SVG <title> elements stay allowed:
// they are elements, not attributes.

/** Native title attributes in one Svelte file: `title="..."`, `title={...}`
 *  and the `{title}` shorthand, at the top level of an element's opening
 *  tag. Script and style blocks and comments are skipped, and so is
 *  anything inside an attribute's own braces (a tooltip's `{ title: ... }`
 *  object, for instance). A component's prop named title (`<HintsSummary
 *  title={...}>`) is not a native attribute and is not flagged; the
 *  component's own file is scanned like any other. Returns the 1-based line
 *  of each. */
function findTitleAttributes(src: string): { line: number; match: string }[] {
  const found: { line: number; match: string }[] = [];
  const blank = (m: string): string => m.replace(/[^\n]/g, ' ');
  const text = src
    .replace(/<script[\s\S]*?<\/script>/g, blank)
    .replace(/<style[\s\S]*?<\/style>/g, blank)
    .replace(/<!--[\s\S]*?-->/g, blank);
  const lineAt = (at: number): number => text.slice(0, at).split('\n').length;

  // The index just past a quoted string or a balanced {...} starting at `at`.
  const skipValue = (at: number): number => {
    const c = text.charAt(at);
    if (c === '"' || c === "'") {
      const close = text.indexOf(c, at + 1);
      return close < 0 ? text.length : close + 1;
    }
    let depth = 0;
    for (let k = at; k < text.length; k++) {
      const d = text.charAt(k);
      if (d === '"' || d === "'" || d === '`') {
        const close = text.indexOf(d, k + 1);
        k = close < 0 ? text.length : close;
      } else if (d === '{') depth++;
      else if (d === '}' && --depth === 0) return k + 1;
    }
    return text.length;
  };

  const tag = /<[a-zA-Z][\w:.-]*/g;
  for (let m = tag.exec(text); m; m = tag.exec(text)) {
    let k = m.index + m[0].length;
    // A component (a capitalised or dotted tag name) takes props, not
    // native attributes.
    const component = /^<(?:[A-Z]|[\w-]+\.)/.test(m[0]);
    while (k < text.length) {
      const c = text.charAt(k);
      if (/[\s/]/.test(c)) {
        k++;
      } else if (c === '>') {
        break;
      } else if (c === '{') {
        // An attachment, a spread or the {name} shorthand.
        const end = skipValue(k);
        const inner = text.slice(k, end);
        if (!component && /^\{\s*title\s*\}$/.test(inner)) {
          found.push({ line: lineAt(k), match: inner });
        }
        k = end;
      } else {
        const name = /^[^\s=>/]+/.exec(text.slice(k))?.[0] ?? c;
        if (!component && name.toLowerCase() === 'title') {
          found.push({ line: lineAt(k), match: name });
        }
        k += name.length;
        while (/\s/.test(text.charAt(k))) k++;
        if (text.charAt(k) === '=') {
          k++;
          while (/\s/.test(text.charAt(k))) k++;
          const v = text.charAt(k);
          if (v === '"' || v === "'" || v === '{') k = skipValue(k);
          else while (k < text.length && !/[\s>]/.test(text.charAt(k))) k++;
        }
      }
    }
    tag.lastIndex = k + 1;
  }
  return found;
}

describe('findTitleAttributes', () => {
  it.each([
    ['<span title="Wood">', 'a literal'],
    ["<span title={t('board.resource.chip')}>", 'an expression'],
    ['<iframe {title} src="/x"></iframe>', 'the shorthand'],
    ['<button\n  type="button"\n  title={label}\n>', 'on a line of its own'],
    ['<div class="a" title="x" />', 'self-closing'],
    ['<svelte:element this="span" title="x">', 'a dynamic element']
  ])('flags %s (%s)', (src) => {
    expect(findTitleAttributes(src)).toHaveLength(1);
  });

  it('reports the line', () => {
    expect(findTitleAttributes('<p>\n</p>\n<span\n  title={x}\n>')).toEqual([
      { line: 4, match: 'title' }
    ]);
  });

  it.each([
    ['<svg><title>Wood</title></svg>', 'an SVG title element'],
    ["<span {@attach tooltip({ text: t('x'), title: 'y' })}>", 'inside an attachment'],
    ['<p>{title}</p>', 'the expression as text'],
    ['<script>\n  const o = { title: 1 };\n  x.title = "a";\n</script>', 'script code'],
    ['<!-- a title="x" comment -->', 'a comment'],
    ['<h2 id={titleId} data-title="a">', 'other names containing title'],
    ['<Sheet titleId="x" aria-label={t(\'a\')}>', 'a component prop named titleId'],
    ["<HintsSummary hints={h} title={t('x')} />", 'a component prop named title'],
    ['<Foo {title} />', 'the shorthand on a component'],
    ['<ui.Card title="x">', 'a dotted component']
  ])('does not flag %s (%s)', (src) => {
    expect(findTitleAttributes(src)).toEqual([]);
  });
});

describe('no native title attribute in the app', () => {
  it('finds none under src/', () => {
    const root = fileURLToPath(new URL('..', import.meta.url));
    const files: string[] = [];
    const walk = (dir: string): void => {
      for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const path = join(dir, entry.name);
        if (entry.isDirectory()) walk(path);
        else if (entry.name.endsWith('.svelte')) files.push(path);
      }
    };
    walk(root);
    expect(files.length).toBeGreaterThan(50);
    const offenders = files.flatMap((file) =>
      findTitleAttributes(readFileSync(file, 'utf8')).map(
        ({ line, match }) => `${relative(root, file)}:${line}: ${match}`
      )
    );
    expect(offenders).toEqual([]);
  });
});

/**
 * Every `@layer` inside a .svelte file's <style> blocks, with its line.
 * Cascade layers are ordered by the first stylesheet in the document that
 * names them, and a component's CSS chunk can be linked before app.css's.
 * A component naming `components` that way put it ahead of Tailwind's
 * `base`, so preflight beat Icon's defaults (M11.icon-size-override). Only
 * app.css names layers.
 */
function findStyleLayers(src: string): { line: number; match: string }[] {
  const out: { line: number; match: string }[] = [];
  for (const block of src.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/g)) {
    const start = block.index + block[0].indexOf('>') + 1;
    for (const m of block[1]!.matchAll(/@layer\b[^{;]*/g)) {
      const at = start + m.index;
      out.push({ line: src.slice(0, at).split('\n').length, match: m[0].trim() });
    }
  }
  return out;
}

describe('findStyleLayers', () => {
  it('finds @layer blocks and statements in a style block', () => {
    const src =
      '<p>a</p>\n<style>\n  @layer components {\n    .a { color: red; }\n  }\n  @layer base;\n</style>';
    expect(findStyleLayers(src)).toEqual([
      { line: 3, match: '@layer components' },
      { line: 6, match: '@layer base' }
    ]);
  });

  it('ignores @layer outside style blocks', () => {
    expect(
      findStyleLayers('<p>We use @layer in prose</p>\n<style>.a { color: red; }</style>')
    ).toEqual([]);
  });
});

describe('no cascade layer named in a component style block', () => {
  it('finds none under src/ (only app.css names layers)', () => {
    const root = fileURLToPath(new URL('..', import.meta.url));
    const files: string[] = [];
    const walk = (dir: string): void => {
      for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const path = join(dir, entry.name);
        if (entry.isDirectory()) walk(path);
        else if (entry.name.endsWith('.svelte')) files.push(path);
      }
    };
    walk(root);
    expect(files.length).toBeGreaterThan(50);
    const offenders = files.flatMap((file) =>
      findStyleLayers(readFileSync(file, 'utf8')).map(
        ({ line, match }) => `${relative(root, file)}:${line}: ${match}`
      )
    );
    expect(
      offenders,
      "A .svelte <style> block names a cascade layer. The first stylesheet that names a layer fixes the order of all layers, and a component chunk can load before app.css, which would put that layer ahead of Tailwind's base (preflight would then override it). Put the rule in app.css inside its @layer block instead."
    ).toEqual([]);
  });
});
