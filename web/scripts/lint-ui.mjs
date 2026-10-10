#!/usr/bin/env node
// A plain Node tool script, not app code under the strict TypeScript gate
// (src/lib/lint-ui.test.ts imports its pure functions, which would
// otherwise pull this file into svelte-check's program).
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
//
// lint-ui.mjs: two project rules ESLint and Prettier do not check (see
// AGENTS.md "Svelte and the web client"):
//
//   1. logical CSS only: no physical Tailwind utility (ml-, mr-, pl-, pr-,
//      left-, right-, text-left, text-right, rounded-l, rounded-r, border-l,
//      border-r, scroll-ml/mr/pl/pr-..., origin-left/right, float-left/right)
//      and no physical CSS property
//      (margin-left, margin-right, padding-left, padding-right, left:,
//      right:, border-left, border-right), unless the line carries
//      "physical-ok".
//   2. no hard-coded English: every template text node (also one on a line
//      of its own, or next to an expression) and every
//      aria-label/title/alt/placeholder literal must go through t(), unless
//      it is inside <code> or <kbd>, or the line carries "i18n-ok".
//
// Node only, no new dependencies, so this runs as a plain step of
// `pnpm -C web lint` (see package.json). The checker functions below are
// pure (line in, violations out) and are unit-tested directly from
// src/lib/lint-ui.test.ts; this file's own main() just walks web/src and
// reports.
//
// Usage: node scripts/lint-ui.mjs [dir]   (default: src)

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

// --- Rule 1: logical CSS only ------------------------------------------

// Matched against one line. Each pattern requires the physical token to
// start at a class/word boundary (line start, whitespace, a single or double
// quote, '{', '(', ',', or a leading '-' for a negative Tailwind value),
// optionally preceded by Tailwind variants ("sm:", "hover:", "dark:", ...),
// and ends at a word boundary, so it does not fire on an unrelated word that
// merely contains the same letters (e.g. "rounded-large", "border-lime").
// No backtick in the class: Tailwind classes in this codebase are always a
// plain class="..." string, never a template literal.
const BOUNDARY = String.raw`(?:^|[\s"'{(,])`;
const VARIANTS = String.raw`(?:[\w-]+:)*`;
const VALUE = String.raw`[\w./%[\]-]*`;

function classPattern(base) {
  return new RegExp(`${BOUNDARY}-?${VARIANTS}${base}${VALUE}`);
}
function bareClassPattern(base) {
  // rounded-l / rounded-r / border-l / border-r: the bare form has no
  // trailing value of its own kind, but may have a size suffix (-lg, -full).
  return new RegExp(`${BOUNDARY}${VARIANTS}${base}(?:-[\\w-]+)?\\b`);
}

const PHYSICAL_TAILWIND = [
  { detail: 'ml- (use ms-)', pattern: classPattern('ml-') },
  { detail: 'mr- (use me-)', pattern: classPattern('mr-') },
  { detail: 'pl- (use ps-)', pattern: classPattern('pl-') },
  { detail: 'pr- (use pe-)', pattern: classPattern('pr-') },
  { detail: 'left- (use start-)', pattern: classPattern('left-') },
  { detail: 'right- (use end-)', pattern: classPattern('right-') },
  {
    detail: 'text-left (use text-start)',
    pattern: new RegExp(`${BOUNDARY}${VARIANTS}text-left\\b`)
  },
  {
    detail: 'text-right (use text-end)',
    pattern: new RegExp(`${BOUNDARY}${VARIANTS}text-right\\b`)
  },
  { detail: 'rounded-l (use rounded-s)', pattern: bareClassPattern('rounded-l') },
  { detail: 'rounded-r (use rounded-e)', pattern: bareClassPattern('rounded-r') },
  { detail: 'border-l (use border-s)', pattern: bareClassPattern('border-l') },
  { detail: 'border-r (use border-e)', pattern: bareClassPattern('border-r') },
  { detail: 'scroll-ml- (use scroll-ms-)', pattern: classPattern('scroll-ml-') },
  { detail: 'scroll-mr- (use scroll-me-)', pattern: classPattern('scroll-mr-') },
  { detail: 'scroll-pl- (use scroll-ps-)', pattern: classPattern('scroll-pl-') },
  { detail: 'scroll-pr- (use scroll-pe-)', pattern: classPattern('scroll-pr-') },
  {
    detail: 'origin-left/right (no logical keyword: set it in CSS per direction)',
    pattern: new RegExp(`${BOUNDARY}${VARIANTS}origin-(?:top-|bottom-)?(?:left|right)\\b`)
  },
  {
    detail: 'float-/clear-left/right (use float-start, float-end)',
    pattern: new RegExp(`${BOUNDARY}${VARIANTS}(?:float|clear)-(?:left|right)\\b`)
  }
];

const PHYSICAL_CSS = [
  { detail: 'margin-left (use margin-inline-start)', pattern: /\bmargin-left\s*:/ },
  { detail: 'margin-right (use margin-inline-end)', pattern: /\bmargin-right\s*:/ },
  { detail: 'padding-left (use padding-inline-start)', pattern: /\bpadding-left\s*:/ },
  { detail: 'padding-right (use padding-inline-end)', pattern: /\bpadding-right\s*:/ },
  { detail: 'border-left (use border-inline-start)', pattern: /\bborder-left\b\s*[:-]/ },
  { detail: 'border-right (use border-inline-end)', pattern: /\bborder-right\b\s*[:-]/ },
  { detail: 'left: (use inset-inline-start)', pattern: /(?:^|[{;\s])left\s*:/ },
  { detail: 'right: (use inset-inline-end)', pattern: /(?:^|[{;\s])right\s*:/ }
];

const PHYSICAL_RULES = [...PHYSICAL_TAILWIND, ...PHYSICAL_CSS];

/** Physical-CSS findings on one line, or [] when clean or marked physical-ok. */
export function findPhysicalCssViolations(line) {
  if (line.includes('physical-ok')) return [];
  const found = [];
  for (const { detail, pattern } of PHYSICAL_RULES) {
    const match = pattern.exec(line);
    if (match) found.push({ rule: 'physical-css', detail, match: match[0].trim() });
  }
  return found;
}

// --- Rule 2: no hard-coded English --------------------------------------
//
// This rule reads a whole template, not one line at a time, because Prettier
// puts long text on a line of its own between the opening and closing tags,
// and text often sits next to an expression ("{count} coins"). Both would
// slip past a per-line "between > and <" check. The i18n-ok opt-out is still
// per line: it covers the line it sits on.

const TEXT_ATTRS = ['aria-label', 'title', 'alt', 'placeholder'];
// A literal string value: attr="...", not attr={...} (an expression, which
// may well call t() and is out of this rule's reach by design: it cannot
// tell what the expression returns).
const ATTR_RE = new RegExp(`\\b(${TEXT_ATTRS.join('|')})="([^"{][^"]*)"`, 'g');
const LETTER_RUN = /[A-Za-z]{3,}/;
// Elements whose content is literal, not translatable copy.
const ALLOWED_TAGS = new Set(['code', 'kbd']);

/** Replaces every character but newlines with a space, keeping offsets. */
function blank(text) {
  return text.replace(/[^\n]/g, ' ');
}

/**
 * Masks what is not template text, keeping offsets and line breaks: script
 * and style blocks, HTML comments, character entities, and every {...}
 * expression or block tag (nested braces and quoted strings inside them
 * included).
 */
function maskNonText(content) {
  let out = content
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, blank)
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, blank)
    .replace(/<!--[\s\S]*?-->/g, blank)
    .replace(/&#?\w+;/g, blank);
  const chars = [...out];
  let depth = 0;
  let quote = '';
  for (let i = 0; i < chars.length; i++) {
    const c = chars[i];
    if (depth === 0) {
      if (c === '{') {
        depth = 1;
        chars[i] = ' ';
      }
      continue;
    }
    if (quote) {
      if (c === '\\') {
        if (chars[i + 1] !== '\n') chars[i + 1] = ' ';
        chars[i] = ' ';
        i++;
        continue;
      }
      if (c === quote) quote = '';
    } else if (c === '"' || c === "'" || c === '`') {
      quote = c;
    } else if (c === '{') {
      depth++;
    } else if (c === '}') {
      depth--;
    }
    if (c !== '\n') chars[i] = ' ';
  }
  out = chars.join('');
  return out;
}

/**
 * Hard-coded-text findings in a whole .svelte source (1-based line numbers),
 * skipping lines marked i18n-ok.
 */
export function findHardcodedText(content) {
  const rawLines = content.split('\n');
  const optedOut = (line) => rawLines[line - 1]?.includes('i18n-ok') ?? false;
  const found = [];

  // Literal text attributes: checked per line on the masked source, so an
  // attribute inside a script block or a comment does not count.
  const masked = maskNonText(content);
  const maskedLines = masked.split('\n');
  maskedLines.forEach((line, i) => {
    if (optedOut(i + 1)) return;
    // Keep the attribute's own value: masking blanked {...} but left quotes.
    const raw = rawLines[i];
    let attrMatch;
    ATTR_RE.lastIndex = 0;
    while ((attrMatch = ATTR_RE.exec(line))) {
      const value = raw.slice(attrMatch.index, attrMatch.index + attrMatch[0].length);
      if (LETTER_RUN.test(attrMatch[2])) {
        found.push({
          rule: 'hardcoded-text',
          detail: `${attrMatch[1]} attribute`,
          match: value,
          line: i + 1
        });
      }
    }
  });

  // Text nodes: walk the masked source, tracking whether we are inside a tag
  // (where quotes matter) and inside an allow-listed element.
  let line = 1;
  let inTag = false;
  let tagQuote = '';
  let tagText = '';
  let allowedDepth = 0;
  let run = '';
  let runLine = 1;
  const flush = () => {
    const text = run.trim();
    if (allowedDepth === 0 && LETTER_RUN.test(text) && !optedOut(runLine)) {
      found.push({ rule: 'hardcoded-text', detail: 'template text', match: text, line: runLine });
    }
    run = '';
  };
  for (const c of masked) {
    if (inTag) {
      if (tagQuote) {
        if (c === tagQuote) tagQuote = '';
      } else if (c === '"' || c === "'") {
        tagQuote = c;
      } else if (c === '>') {
        inTag = false;
        const m = /^(\/?)([\w:-]+)/.exec(tagText);
        if (m && ALLOWED_TAGS.has(m[2].toLowerCase()) && !tagText.endsWith('/')) {
          allowedDepth += m[1] ? -1 : 1;
          if (allowedDepth < 0) allowedDepth = 0;
        }
        tagText = '';
      } else {
        tagText += c;
      }
    } else if (c === '<') {
      flush();
      inTag = true;
    } else if (c === '\n') {
      flush();
    } else {
      if (run.trim() === '' && c.trim() !== '') runLine = line;
      run += c;
    }
    if (c === '\n') line++;
  }
  flush();
  return found;
}

/** Hard-coded-text findings on one line, or [] when clean or marked i18n-ok. */
export function findHardcodedTextViolations(rawLine) {
  return findHardcodedText(rawLine).map(({ rule, detail, match }) => ({ rule, detail, match }));
}

// --- File walking and the CLI -------------------------------------------

/** Strips <script ...>...</script> blocks: neither rule reaches script code
 *  (TS string literals are not UI text; a variable named for a CSS side
 *  is not a CSS declaration), keeping line numbers intact for reporting. */
function stripScriptBlocks(content) {
  return content.replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, blank);
}

function lintFile(path, content) {
  const isSvelte = extname(path) === '.svelte';
  const scannable = isSvelte ? stripScriptBlocks(content) : content;
  const findings = [];
  scannable.split('\n').forEach((line, i) => {
    for (const violation of findPhysicalCssViolations(line)) {
      findings.push({ ...violation, line: i + 1 });
    }
  });
  if (isSvelte) findings.push(...findHardcodedText(content));
  return findings.sort((a, b) => a.line - b.line);
}

function walk(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const stat = statSync(full);
    if (stat.isDirectory()) {
      if (entry === 'node_modules' || entry === '.svelte-kit') continue;
      walk(full, out);
    } else if (extname(full) === '.svelte' || extname(full) === '.css') {
      out.push(full);
    }
  }
  return out;
}

function main() {
  const root = fileURLToPath(new URL('..', import.meta.url)); // web/
  const target = join(root, process.argv[2] ?? 'src');
  const files = walk(target);
  let total = 0;
  for (const file of files) {
    const content = readFileSync(file, 'utf8');
    const findings = lintFile(file, content);
    for (const finding of findings) {
      total++;
      console.error(
        `${relative(root, file)}:${finding.line}: [${finding.rule}] ${finding.detail}: ${finding.match}`
      );
    }
  }
  if (total > 0) {
    console.error(
      `\nlint-ui: ${total} violation(s) in ${files.length} file(s) under ${relative(root, target)}`
    );
    process.exitCode = 1;
  } else {
    console.log(`lint-ui: clean (${files.length} file(s) checked)`);
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}
