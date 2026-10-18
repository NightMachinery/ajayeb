// Tooltip coverage (M10.tooltip-coverage): every board element, symbol on a
// card and icon-only button has a tooltip, or a visible text label it would
// only repeat (AGENTS.md, "Tooltips"). Later work cannot quietly add an
// unexplained glyph.
//
// One seeded hot-seat game is played over the protocol (fixtures/ws-game.ts)
// to learn its keys, as layout.spec.ts does, and the same game is opened at
// the wonder draft, a fresh Age I, a fresh Age III (with face-down guild
// backs) and the result screen, on the desktop project and on the phone.
// At each one, inside the game screen, every visible svg, img and
// [role="img"] element and every icon-only button (no visible letters)
// must be covered:
//
// - by a registered tooltip anchor ([data-tooltip], which the tooltip helper
//   sets) on itself or an ancestor, or, for a button, on a descendant that
//   fills most of it (a mini card's face); or
// - by a visible text label it would only repeat: a letter in the visible
//   text of an immediate sibling (allowing one image-only wrapper).
//
// Elements inside the tooltip layer are skipped, and so is anything in
// ALLOW below, each with its reason. The test prints every offender with a
// selector path.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { settle } from '../fixtures/layout-probes';
import { createDevRoom, recordGame, type GameMarks, type RecordedGame } from '../fixtures/ws-game';

/** Skipped elements: a selector the element or an ancestor matches, and why. */
const ALLOW: readonly { selector: string; reason: string }[] = [
  {
    selector: '[data-card-art]',
    reason: 'a card’s painting is decoration; the card’s name and sheet describe it'
  },
  {
    selector: '.wonder-tile [data-art], .wonder-tile .wonder-art',
    reason: 'a wonder’s painting is decoration beside its printed name'
  },
  {
    selector: '.victory-scene',
    reason: 'the result screen’s painting is decoration under the printed headline'
  },
  {
    selector: '.victory-scene + span',
    reason: 'the winner’s crests under the painting repeat the headline that names the winner'
  }
];

const POSITIONS = [
  'draft',
  'age1',
  'age3',
  'gameOver'
] as const satisfies readonly (keyof GameMarks)[];

/** Seeds tried in turn until one game reaches Age III under the recording
 *  policy with a guild dealt face down. */
const SEEDS = ['11', '12', '13', '14', '15', '16', '17', '18', '19', '20', '21', '22'];

let recorded: Promise<RecordedGame> | null = null;

/** Opens the recorded game at a position, through the server's dev rooms. */
async function open(page: Page, origin: string, rec: RecordedGame, position: keyof GameMarks) {
  const room = await createDevRoom(origin, {
    mode: 'hotseat',
    seed: rec.seed,
    actions: rec.keys.slice(0, rec.marks[position])
  });
  await page.goto(room.path);
  await settle(page);
}

/** The recorded game, played once per worker: the first seed whose fresh
 *  Age III has a face-down guild back. */
function game(page: Page, origin: string): Promise<RecordedGame> {
  recorded ??= (async () => {
    let last: unknown = new Error('no seed deals a guild face down in Age III');
    for (const seed of SEEDS) {
      try {
        const rec = await recordGame(origin, seed);
        await open(page, origin, rec, 'age3');
        const guildBack = await page.evaluate(
          () =>
            window.__ajayeb
              ?.view()
              ?.structure.some((s) => s.state === 'faceDown' && s.back === 'guild') ?? false
        );
        if (guildBack) return rec;
      } catch (err) {
        last = err;
      }
    }
    throw last;
  })();
  return recorded;
}

interface Offender {
  path: string;
  kind: 'image' | 'button';
}

interface Coverage {
  checked: number;
  offenders: Offender[];
  /** The elements passed on a visible label, for a human to review
   *  (printed with TOOLTIP_COVERAGE_DEBUG=1). */
  byLabel: string[];
}

/** Finds the uncovered elements of the game screen (runs in the page). */
async function uncovered(page: Page): Promise<Coverage> {
  return page.evaluate((allow) => {
    const root = document.querySelector('.game-screen');
    if (!root) throw new Error('no game screen');

    const isVisible = (el: Element): boolean => {
      const r = el.getBoundingClientRect();
      if (r.width < 1 || r.height < 1) return false;
      if (r.bottom <= 0 || r.right <= 0 || r.top >= innerHeight || r.left >= innerWidth) {
        return false;
      }
      if (el.closest('.sr-only, [hidden], [inert]')) return false;
      for (let n: Element | null = el; n; n = n.parentElement) {
        const cs = getComputedStyle(n);
        if (cs.display === 'none' || cs.visibility === 'hidden' || cs.opacity === '0') {
          return false;
        }
      }
      return true;
    };

    /** The text a sighted user sees in `el` (screen-reader-only text left out). */
    const visibleText = (el: Element): string => {
      let out = '';
      const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
      for (let node = walker.nextNode(); node; node = walker.nextNode()) {
        const parent = node.parentElement;
        if (!parent || parent.closest('.sr-only')) continue;
        if (!isVisible(parent)) continue;
        out += node.textContent ?? '';
      }
      return out.trim();
    };
    const hasLetter = (s: string): boolean => /\p{L}/u.test(s);

    const allowed = (el: Element): boolean =>
      allow.some(({ selector }) => el.closest(selector) !== null);

    const anchored = (el: Element): boolean => {
      const anchor = el.closest('[data-tooltip]');
      // A whole-card fallback describes its art/frame, not an unexplained
      // symbol. Keep the existing per-symbol coverage gate meaningful.
      return (
        anchor !== null &&
        (anchor === el || !anchor.matches('[data-tooltip-card], [data-tooltip-wonder]'))
      );
    };

    const labelled = (el: Element): boolean => {
      let branch = el;
      for (let depth = 0; depth < 2; depth++) {
        const parent = branch.parentElement;
        if (!parent) return false;
        // Exactly one image-bearing branch belongs to this label. A new
        // unrelated glyph in the same row must provide its own tooltip.
        const images = [...parent.children].filter(
          (child) =>
            child.matches('svg, img, [role="img"]') || child.querySelector('svg, img, [role="img"]')
        );
        if (images.length !== 1) return false;
        for (const sibling of parent.childNodes) {
          if (sibling === branch) continue;
          if (sibling.nodeType === Node.TEXT_NODE && hasLetter(sibling.textContent ?? ''))
            return true;
          if (
            sibling instanceof Element &&
            isVisible(sibling) &&
            !sibling.querySelector('svg, img, [role="img"], button, a, input') &&
            !sibling.matches('svg, img, [role="img"], button, a, input') &&
            hasLetter(visibleText(sibling))
          )
            return true;
        }
        if (hasLetter(visibleText(parent))) return false;
        branch = parent;
      }
      return false;
    };

    /** A descendant anchor that fills most of the button (a card's face). */
    const filled = (button: Element): boolean => {
      const b = button.getBoundingClientRect();
      const area = b.width * b.height;
      return [...button.querySelectorAll('[data-tooltip]')].some((a) => {
        const r = a.getBoundingClientRect();
        return r.width * r.height >= 0.75 * area;
      });
    };

    const describe = (el: Element): string => {
      const parts: string[] = [];
      for (let n: Element | null = el, i = 0; n && n !== root && i < 6; n = n.parentElement, i++) {
        let part = n.tagName.toLowerCase();
        const classes = [...n.classList]
          .filter((c) => !c.includes(':') && !c.includes('['))
          .slice(0, 2);
        if (classes.length > 0) part += `.${classes.join('.')}`;
        const data = [...n.attributes]
          .filter((a) => a.name.startsWith('data-') && a.name !== 'data-tooltip')
          .slice(0, 2)
          .map((a) => (a.value ? `[${a.name}="${a.value.slice(0, 24)}"]` : `[${a.name}]`));
        const aria = n.getAttribute('aria-label');
        parts.unshift(part + data.join('') + (aria ? `[aria-label="${aria.slice(0, 30)}"]` : ''));
      }
      return parts.join(' > ');
    };

    const offenders: { path: string; kind: 'image' | 'button' }[] = [];
    const byLabel: string[] = [];
    let checked = 0;
    for (const el of root.querySelectorAll('svg, img, [role="img"]')) {
      if (el.closest('[data-tooltip-layer]')) continue;
      // Only the outermost of nested images.
      if (el.parentElement?.closest('svg')) continue;
      if (!isVisible(el) || allowed(el)) continue;
      checked++;
      if (anchored(el)) continue;
      if (labelled(el)) {
        byLabel.push(describe(el));
        continue;
      }
      offenders.push({ path: describe(el), kind: 'image' });
    }
    for (const el of root.querySelectorAll('button')) {
      if (!isVisible(el) || allowed(el)) continue;
      if (hasLetter(visibleText(el))) continue;
      checked++;
      if (anchored(el) || filled(el)) continue;
      offenders.push({ path: describe(el), kind: 'button' });
    }
    return { checked, offenders, byLabel };
  }, ALLOW);
}

test.describe('tooltip coverage', { tag: '@mobile' }, () => {
  test('a label beside one glyph does not excuse unrelated glyphs in its row', async ({ page }) => {
    await page.goto('/');
    await page.setContent(`<main class="game-screen">
      <div><svg width="24" height="24"></svg><span>Its own label</span></div>
      <div><span><svg width="24" height="24"></svg></span><span>One wrapped icon</span></div>
      <div><svg width="24" height="24"></svg><span>A row label</span><svg width="24" height="24"></svg></div>
    </main>`);
    const result = await uncovered(page);
    expect(result.byLabel).toHaveLength(2);
    expect(result.offenders).toHaveLength(2);
  });

  test('a whole-card fallback does not excuse a symbol without its own explanation', async ({
    page
  }) => {
    await page.goto('/');
    await page.setContent(`<main class="game-screen">
      <div data-tooltip data-tooltip-card="baths">
        <span data-card-art="baths"><svg width="24" height="24"></svg></span>
        <svg width="24" height="24"></svg>
        <span data-tooltip><svg width="24" height="24"></svg></span>
      </div>
    </main>`);
    const result = await uncovered(page);
    expect(result.offenders).toHaveLength(1);
  });

  test('every image and icon-only button of the game screen has a tooltip', async ({
    page,
    baseURL
  }, info) => {
    test.setTimeout(150_000);
    if (!baseURL) throw new Error('no baseURL');
    // The board itself, without the pass-device screen between turns; a
    // stored name keeps the room page from asking for one (see layout.spec).
    await page.addInitScript(() => {
      try {
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({
            hotseatPassDevice: false,
            animationSpeed: 'off',
            showOpponentMove: false
          })
        );
        localStorage.setItem('ajayeb:name', 'Coverage');
      } catch {
        // Storage may be blocked.
      }
    });
    if (info.project.name === 'desktop') await page.setViewportSize({ width: 1280, height: 800 });
    const recording = await game(page, baseURL);
    const problems = watchHygiene(page);

    const report: string[] = [];
    for (const position of POSITIONS) {
      await test.step(position, async () => {
        await open(page, baseURL, recording, position);
        if (position === 'age3') {
          // Age III deals its guilds face down among the backs.
          await expect(page.locator('.card-structure [data-back="guild"]').first()).toBeVisible();
          const nodes = await page.locator('*').count();
          console.log(`${info.project.name} Age III DOM: ${nodes}`);
          expect(nodes, 'Age III stays within the UX DOM budget').toBeLessThan(2500);
        }
        const { checked, offenders, byLabel } = await uncovered(page);
        console.log(
          `${info.project.name} ${position}: ${checked} checked, ${byLabel.length} by a label, ${offenders.length} uncovered`
        );
        if (process.env['TOOLTIP_COVERAGE_DEBUG']) console.log(byLabel.join('\n'));
        for (const o of offenders) report.push(`${position} ${o.kind}: ${o.path}`);
        expect(checked, 'the probe found elements to check').toBeGreaterThan(5);
      });
    }
    if (report.length > 0) console.log(`Uncovered:\n${report.join('\n')}`);
    expect(report, 'images and icon-only buttons with no tooltip and no label').toEqual([]);
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
