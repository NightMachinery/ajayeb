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
//      border-r, scroll-ml/mr/pl/pr-...) and no physical CSS property
//      (margin-left, margin-right, padding-left, padding-right, left:,
//      right:, border-left, border-right), unless the line carries
//      "physical-ok".
//   2. no hard-coded English: every template text node and every
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
  { detail: 'scroll-pr- (use scroll-pe-)', pattern: classPattern('scroll-pr-') }
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

const TEXT_ATTRS = ['aria-label', 'title', 'alt', 'placeholder'];
// A literal string value: attr="...", not attr={...} (an expression, which
// may well call t() and is out of this rule's reach by design: it cannot
// tell what the expression returns).
const ATTR_RE = new RegExp(`\\b(${TEXT_ATTRS.join('|')})="([^"{][^"]*)"`, 'g');
const LETTER_RUN = /[A-Za-z]{3,}/;

/** Strips the same-line content of <code> and <kbd>, the allow-listed tags. */
function stripAllowedTags(line) {
  return line
    .replace(/<code\b[^>]*>.*?<\/code>/gi, '<code></code>')
    .replace(/<kbd\b[^>]*>.*?<\/kbd>/gi, '<kbd></kbd>');
}

/** Hard-coded-text findings on one line, or [] when clean or marked i18n-ok. */
export function findHardcodedTextViolations(rawLine) {
  if (rawLine.includes('i18n-ok')) return [];
  const found = [];

  let attrMatch;
  ATTR_RE.lastIndex = 0;
  while ((attrMatch = ATTR_RE.exec(rawLine))) {
    const [full, attr, value] = attrMatch;
    if (LETTER_RUN.test(value)) {
      found.push({ rule: 'hardcoded-text', detail: `${attr} attribute`, match: full });
    }
  }

  const line = stripAllowedTags(rawLine);
  const textRe = />([^<>{}]+)</g;
  let textMatch;
  while ((textMatch = textRe.exec(line))) {
    const text = textMatch[1];
    if (LETTER_RUN.test(text) && text.trim().length > 0) {
      found.push({ rule: 'hardcoded-text', detail: 'template text', match: text.trim() });
    }
  }
  return found;
}

// --- File walking and the CLI -------------------------------------------

/** Strips <script ...>...</script> blocks: neither rule reaches script code
 *  (TS string literals are not UI text; a variable named for a CSS side
 *  is not a CSS declaration), keeping line numbers intact for reporting. */
function stripScriptBlocks(content) {
  return content.replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, (block) =>
    block.replace(/[^\n]/g, ' ')
  );
}

function lintFile(path, content) {
  const isSvelte = extname(path) === '.svelte';
  const scannable = isSvelte ? stripScriptBlocks(content) : content;
  const lines = scannable.split('\n');
  const findings = [];
  lines.forEach((line, i) => {
    for (const violation of findPhysicalCssViolations(line)) {
      findings.push({ ...violation, line: i + 1 });
    }
    if (isSvelte) {
      for (const violation of findHardcodedTextViolations(line)) {
        findings.push({ ...violation, line: i + 1 });
      }
    }
  });
  return findings;
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
