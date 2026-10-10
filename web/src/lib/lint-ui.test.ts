// Unit tests for the two UI lints (web/scripts/lint-ui.mjs), run against
// small strings rather than real files. The script itself walks web/src and
// is wired into `pnpm -C web lint` (package.json); these tests cover the
// pure rule functions it exports. Living here (not under web/scripts/) is
// what gets it picked up by vitest's include glob (vite.config.ts).
import { describe, expect, it } from 'vitest';
import { findHardcodedTextViolations, findPhysicalCssViolations } from '../../scripts/lint-ui.mjs';

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
    ['<div class="-ml-2">', 'negative ml-']
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
