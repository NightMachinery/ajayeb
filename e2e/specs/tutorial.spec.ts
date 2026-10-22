// The guided first game (M10.tutorial-client), on the desktop and mobile
// projects, through the DOM contract only (fixtures/tutorial.ts):
//
// - the whole walk from Home through the real create, with the first-time
//   tips on and the move summary on as a real player has them: the lesson
//   ids in the script's order (GET /api/dev/tutorial), the guided keys sent
//   in order, every coach step's target on screen and unobscured, the
//   bubble on screen and off an element target, no tip during the guided
//   part, the pawn's tip (taught by the military lesson) and the new age's
//   (taught by the completion card) never, then the completion card and
//   New game opening the vs-AI sheet; in an ordinary game, the new age's
//   tip does show;
// - Play on my own and Resume tutorial (the rails off, then the same step);
// - a different move while paused (the path is left, leftPath shows once
//   and no guided step follows) and Undo back onto it (the lesson returns);
// - a reload in the middle of a lesson (the same step);
// - Home's No thanks (gone after a reload) and the Settings reset;
// - right to left in Persian at 390x844 and 1280x800;
// - Animation off (no animation over 10 ms) and Animation normal (the
//   bubble only once the presenter is idle, its target box final).

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { playKey } from '../fixtures/scenario';
import { motionLog, recordMotion, waitSettled } from '../fixtures/anim';
import { watchHygiene } from '../fixtures/hygiene';
import {
  freshDevice,
  openTutorialRoom,
  pressOn,
  recordSent,
  snapshot,
  tutorialScript,
  walkTutorial,
  waitLeft,
  type Script
} from '../fixtures/tutorial';

const BUBBLE = '[data-coach][data-placed]';

/** The first-time tips but the new age's (TIP_IDS in web/src/lib/ui/tips.svelte.ts). */
const OTHER_TIPS = [
  'win',
  'pair',
  'denyPair',
  'winOpp',
  'loot',
  'lawLaurel',
  'ghostLoot',
  'hourglass',
  'freeChain',
  'guildBack',
  'track',
  'science'
];

let script: Promise<Script> | null = null;
function scriptOf(origin: string): Promise<Script> {
  script ??= tutorialScript(origin);
  return script;
}

/** The script's keys up to (not including) the first guided turn of an action id. */
function keysBefore(s: Script, actionId: string): string[] {
  const i = s.raw.entries.findIndex((e) => e.action?.id === actionId);
  if (i < 0) throw new Error(`no ${actionId} in the script`);
  return s.keys.slice(0, i);
}

/** Opens a wonder's Take where it is: on the page on a wide screen, in its sheet on a phone. */
async function showTake(page: Page, wonder: string, touch: boolean): Promise<void> {
  const tile = page.locator(`[data-pick="draft:${wonder}"]`).locator('visible=true');
  if ((await tile.count()) > 0) await pressOn(page, `[data-pick="draft:${wonder}"]`, touch);
}

async function closeSheet(page: Page): Promise<void> {
  if ((await page.locator('dialog[open]').count()) > 0) await page.keyboard.press('Escape');
  await expect(page.locator('dialog[open]')).toHaveCount(0);
}

async function menu(page: Page, item: string, touch: boolean): Promise<void> {
  await pressOn(page, '[data-control="menu"]', touch);
  await pressOn(page, `[data-control="${item}"]`, touch);
}

test.describe('the tutorial', { tag: '@mobile' }, () => {
  test('from Home to the completion card, in the script order', async ({
    page,
    baseURL,
    isMobile
  }) => {
    test.setTimeout(240_000);
    if (!baseURL) throw new Error('no baseURL');
    const s = await scriptOf(baseURL);
    const problems = watchHygiene(page);
    await freshDevice(page, { tips: true, summary: true });
    const sent = recordSent(page);
    await page.goto('/');
    await expect(page.locator('[data-tutorial-offer]')).toBeVisible();
    await pressOn(page, '[data-control="tutorial-start"]', isMobile);
    await page.waitForURL(/\/play\//);

    const walk = await walkTutorial(
      page,
      {
        touch: isMobile,
        onStep: async (_seen, step) => {
          if (step.step === 'playOn') {
            await expect(page.locator('[data-coach-pause]').locator('visible=true')).toHaveCount(0);
          }
        }
      },
      sent
    );
    expect(walk.done, 'the completion card').toBe(true);
    expect(walk.observed.slice(0, s.order.length)).toEqual(s.order);
    expect(sent.slice(0, s.guided.length), 'the guided keys sent').toEqual(s.guided);
    expect(walk.guidedTips, 'no first-time tip during the guided part').toEqual([]);
    expect(walk.tips, 'the military lesson taught the pawn').not.toContain('track');
    expect(walk.tips, 'the completion card taught the new age').not.toContain('newAge');

    // Nothing else is open beside the card (a first-time tip would swallow the tap).
    await expect(page.locator('[data-dismiss-layer]')).toHaveCount(1);
    await pressOn(page, '[data-control="tutorial-new"]', isMobile);
    await expect(page).toHaveURL((u) => u.pathname === '/');
    await expect(page.locator('[data-control="ai-tutorial"]'), 'the vs-AI sheet').toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('[data-control="home-tutorial"]')).toHaveAttribute(
      'data-tutorial-replay',
      ''
    );
    expect(problems).toEqual([]);
  });

  test("outside the tutorial, the new age's tip shows", async ({ page, baseURL }) => {
    if (!baseURL) throw new Error('no baseURL');
    // Every other tip seen: one tip shows per ply, and the pawn's would come first.
    await freshDevice(page, { tips: true, seen: OTHER_TIPS, summary: false });
    // Seat 0 takes Age I's last card; the bot, behind on the pawn, chooses
    // who starts Age II.
    const res = await fetch(`${baseURL}/api/dev/rooms`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        mode: 'ai',
        bot: 'easy',
        scenario: 'age-transition-chooser'
      })
    });
    const room = (await res.json()) as { code: string; path: string };
    await page.goto(room.path);
    const driver = new GameDriver(page);
    await driver.waitReady();
    // Cross the boundary on the live Duel connection, after discovery. A cold
    // join of an already-advanced age intentionally does not replay its events.
    await playKey(driver, 'discard:theater');
    // The age-start sheet comes first. No tip opens over a modal dialog,
    // where its dismissal would swallow the player's first tap in it.
    const begin = page.locator('[data-age-start] dialog[open] [data-age-continue]');
    await expect(begin).toBeVisible({ timeout: 15_000 });
    await expect(page.locator('[data-first-tip]')).toHaveCount(0);
    await begin.click();
    await expect(page.locator('[data-first-tip="newAge"]')).toBeVisible({ timeout: 15_000 });
  });

  test('Play on my own turns the rails off; Resume tutorial brings the step back', async ({
    page,
    baseURL,
    isMobile
  }) => {
    if (!baseURL) throw new Error('no baseURL');
    await freshDevice(page, { tips: false, summary: false });
    await openTutorialRoom(page, baseURL);
    await expect(page.locator(BUBBLE)).toHaveAttribute('data-coach-step', 'draftIntro');

    // The rails: only the guided wonder can be taken, and a reason says so.
    await showTake(page, 'mausoleum', isMobile);
    const take = page.locator('[data-act="draft:mausoleum"]').locator('visible=true');
    await expect(take).toBeDisabled();
    await expect(page.locator('[data-tutorial-reason]').locator('visible=true')).toHaveCount(1);
    await closeSheet(page);

    await pressOn(page, '[data-coach-pause]', isMobile);
    await expect(page.locator('[data-coach]')).toBeHidden();
    await showTake(page, 'mausoleum', isMobile);
    await expect(take).toBeEnabled();
    await expect(page.locator('[data-tutorial-reason]')).toHaveCount(0);
    await closeSheet(page);

    await menu(page, 'menu-tutorial', isMobile);
    await expect(page.locator(BUBBLE)).toHaveAttribute('data-coach-step', 'draftIntro');
  });

  test('a different move leaves the path; Undo goes back onto it', async ({
    page,
    baseURL,
    isMobile
  }) => {
    if (!baseURL) throw new Error('no baseURL');
    await freshDevice(page, { tips: false, summary: false });
    await openTutorialRoom(page, baseURL);
    await expect(page.locator(BUBBLE)).toBeVisible();
    await pressOn(page, '[data-coach-pause]', isMobile);
    await showTake(page, 'mausoleum', isMobile);
    await pressOn(page, '[data-act="draft:mausoleum"]', isMobile);
    await waitLeft(page, true);

    await menu(page, 'menu-tutorial', isMobile);
    await expect(page.locator(BUBBLE)).toHaveAttribute('data-coach-step', 'leftPath');
    await pressOn(page, '[data-coach-next]', isMobile);
    await expect(page.locator('[data-coach]')).toBeHidden();
    expect((await snapshot(page)).tutorial?.status).toBe('free');
    // Once only: a reload does not bring it back.
    await page.reload();
    await page.waitForFunction(() => window.__ajayeb?.ready() === true);
    await page.waitForTimeout(500);
    await expect(page.locator('[data-coach]')).toBeHidden();

    // The top bar's Undo, or the menu's where a phone keeps it.
    if ((await page.locator('[data-control="undo"]').locator('visible=true').count()) > 0) {
      await pressOn(page, '[data-control="undo"]', isMobile);
    } else {
      await menu(page, 'menu-undo', isMobile);
    }
    await waitLeft(page, false);
    await expect(page.locator(BUBBLE)).toHaveAttribute('data-coach-turn', '1');
  });

  test('a reload in the middle of a lesson shows the same step', async ({
    page,
    baseURL,
    isMobile
  }) => {
    if (!baseURL) throw new Error('no baseURL');
    const s = await scriptOf(baseURL);
    await freshDevice(page, { tips: false, summary: false });
    await openTutorialRoom(page, baseURL, keysBefore(s, 'buildFree'));
    await expect(page.locator(BUBBLE)).toHaveAttribute('data-coach-step', 'ageOne');
    await pressOn(page, '[data-coach-next]', isMobile);
    await expect(page.locator(BUBBLE)).toHaveAttribute('data-coach-step', 'turn');
    await page.reload();
    await expect(page.locator(BUBBLE)).toHaveAttribute('data-coach-step', 'turn');
  });

  test("Home's No thanks stays after a reload; Settings brings the offer back", async ({
    page,
    isMobile
  }) => {
    await freshDevice(page, { offer: 'new' });
    await page.goto('/');
    const offer = page.locator('[data-tutorial-offer]');
    await expect(offer).toBeVisible();
    await pressOn(page, '[data-control="tutorial-decline"]', isMobile);
    await expect(offer).toHaveCount(0);
    await expect(page.locator('[data-control="home-tutorial"]')).toContainText('Apprentice');
    await page.reload();
    await expect(page.locator('[data-control="home-tutorial"]')).toBeVisible();
    await expect(offer).toHaveCount(0);

    await page.getByRole('button', { name: 'Settings' }).first().click();
    await pressOn(page, '[data-control="tutorial-reset"]', isMobile);
    await page.keyboard.press('Escape');
    await expect(offer).toBeVisible();
  });

  for (const viewport of [
    { width: 390, height: 844 },
    { width: 1280, height: 800 }
  ]) {
    test.describe(`in Persian at ${viewport.width}x${viewport.height}`, () => {
      test.use({ viewport, appLocale: 'fa' });

      test('the bubble stays on screen and points at its target', async ({
        page,
        baseURL,
        isMobile
      }) => {
        if (!baseURL) throw new Error('no baseURL');
        const problems = watchHygiene(page);
        await freshDevice(page, { tips: false, summary: false });
        await openTutorialRoom(page, baseURL);
        await expect(page.locator('html')).toHaveAttribute('dir', 'rtl');
        const walk = await walkTutorial(page, { touch: isMobile, until: 'turn' });
        expect(walk.steps.length).toBeGreaterThan(5);
        expect(problems).toEqual([]);
      });
    });
  }

  test('with Animation off, nothing animates during the guided turns', async ({
    page,
    baseURL,
    isMobile
  }) => {
    if (!baseURL) throw new Error('no baseURL');
    await recordMotion(page);
    await freshDevice(page, { tips: false, summary: false, speed: 'off' });
    await openTutorialRoom(page, baseURL);
    await walkTutorial(page, { touch: isMobile, until: 'military' });
    const long = (await motionLog(page)).filter((m) => m.ms > 10);
    expect(long, 'animations over 10 ms').toEqual([]);
  });

  test('with Animation normal, the bubble waits for the board to settle', async ({
    page,
    baseURL,
    isMobile
  }) => {
    test.setTimeout(120_000);
    if (!baseURL) throw new Error('no baseURL');
    const s = await scriptOf(baseURL);
    await freshDevice(page, { tips: false, summary: false, speed: 'normal' });
    await openTutorialRoom(page, baseURL, keysBefore(s, 'buildFree'));
    let checked = 0;
    await walkTutorial(page, {
      touch: isMobile,
      until: 'military',
      onStep: async (seen, step) => {
        if (step.surface !== 'bubble' || !seen.target || seen.target.region) return;
        expect.soft(seen.ready, `${step.step}: the presenter is idle under the bubble`).toBe(true);
        await waitSettled(page);
        const after = await snapshot(page);
        const a = seen.target.box;
        const b = after.target?.box;
        expect.soft(b, `${step.step}: the target is still there`).toBeTruthy();
        if (b) {
          const moved = Math.max(
            Math.abs(a.x - b.x),
            Math.abs(a.y - b.y),
            Math.abs(a.width - b.width),
            Math.abs(a.height - b.height)
          );
          expect
            .soft(moved, `${step.step}: the target box moved after the bubble`)
            .toBeLessThanOrEqual(2);
        }
        checked++;
      }
    });
    expect(checked, 'element targets checked').toBeGreaterThan(2);
  });
});
