// The order at the start of Ages II and III (M11.age-start-order): the new
// age's deal and its sound come first, then the "who starts" choice. The
// frame that opens the age deals it, and nothing covers the board until
// the deal has ended: not the age dialog with the starter choice, not the
// move summary, not the pass-device screen. The starter's answer deals
// nothing, whoever gives it, and a reload during the choice never deals
// the age again.
//
// A cover is the first moment any of those surfaces is in the DOM, read by
// a MutationObserver against performance.now(), the clock of the runner's
// records (window.__ajayebFx.anims()).
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import {
  clearFx,
  fxCues,
  fxRecords,
  openPosition,
  playThrough,
  useSettings,
  waitSettled,
  type AnimRecord
} from '../fixtures/anim';
import type { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';

const SCENARIO = 'age-transition-chooser';
const LAST_CARD = 'discard:theater';
const AGE = 'dialog[open] [data-age-interstitial]';
const COVERS = {
  dialog: 'dialog[open] [data-age-interstitial]',
  pass: '[data-pass-device]',
  summary: '[data-dismiss-layer] [data-move-summary]'
} as const;

interface Cover {
  what: string;
  at: number;
  /** The dialog holds the starter choice itself (not yet its outcome). */
  choosing: boolean;
}

/** Starts noting the first appearance of each surface that covers the board. */
async function watchCovers(page: Page): Promise<void> {
  await page.evaluate((covers) => {
    const seen: Cover[] = [];
    const note = () => {
      for (const [what, selector] of Object.entries(covers)) {
        if (seen.some((c) => c.what === what) || !document.querySelector(selector)) continue;
        seen.push({
          what,
          at: performance.now(),
          choosing: !!document.querySelector('[data-choice-host="chooseStarter"] dialog[open]')
        });
      }
    };
    new MutationObserver(note).observe(document.body, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ['open']
    });
    Object.assign(window, { __covers: seen });
  }, COVERS);
}

async function covers(page: Page): Promise<Cover[]> {
  return page.evaluate(() => (window as unknown as { __covers: Cover[] }).__covers);
}

/** The game's own settings for this spec, with sound on so cues are recorded. */
async function configure(page: Page, speed: 'normal' | 'off', pass = false): Promise<void> {
  await useSettings(page, speed);
  await page.evaluate((pass) => {
    const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
    localStorage.setItem(
      'ajayeb:settings',
      JSON.stringify({ ...settings, sound: true, hotseatPassDevice: pass })
    );
  }, pass);
}

async function open(page: Page, mode: 'hotseat' | 'ai', baseURL: string): Promise<GameDriver> {
  const driver = await openPosition(page, baseURL, {
    mode,
    seed: '11',
    scenario: SCENARIO,
    keys: []
  });
  await driver.nextTurn();
  return driver;
}

/** The deal's end, after checking that each dealt card landed uninterrupted. */
function dealEnd(anims: AnimRecord[], label: string): number {
  const deals = anims.filter((a) => a.kind === 'deal');
  expect(deals.length, `${label}: the new age was dealt`).toBeGreaterThan(0);
  for (const deal of deals) {
    expect(deal.interrupted, `${label}: a dealt card was cut short`).toBe(false);
    expect(deal.endAt, `${label}: a dealt card never landed`).not.toBeNull();
  }
  return Math.max(...deals.map((d) => d.endAt!));
}

async function ageStartCues(page: Page): Promise<number[]> {
  return (await fxCues(page))
    .filter((c) => c.cue === 'ageStart' && !c.replay)
    .map((c) => c.version);
}

test.describe('the deal comes before the starter choice', { tag: '@mobile' }, () => {
  for (const speed of ['normal', 'off'] as const) {
    test(`the viewer chooses: deal and sound, then the choice (Animation ${speed})`, async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('no baseURL');
      const problems = watchHygiene(page);
      await configure(page, speed);
      const driver = await open(page, 'hotseat', baseURL);
      await clearFx(page);
      await watchCovers(page);
      await playThrough(driver, LAST_CARD);
      await expect(page.locator(AGE)).toBeVisible();
      const opening = await driver.snapshot();
      expect(opening.phase).toBe('chooseStarter');
      await expect(page.locator('dialog[open] [data-act="starter:0"]')).toBeVisible();
      const [first] = await covers(page);
      expect(first, 'the board was covered').toBeDefined();
      expect(first!.what, 'the age dialog is the first thing over the board').toBe('dialog');
      expect(first!.choosing, 'the dialog opens on the choice').toBe(true);
      const { anims } = await fxRecords(page);
      if (speed === 'normal') {
        const end = dealEnd(anims, 'the opening');
        expect(first!.at, 'the deal ended before the choice opened').toBeGreaterThanOrEqual(end);
      } else {
        expect(anims, 'Animation off deals at once').toEqual([]);
      }
      expect(await ageStartCues(page), 'the age sounds with its deal').toEqual([opening.version]);

      // The answer deals nothing and does not sound the age again.
      await clearFx(page);
      await page.locator('dialog[open] [data-act="starter:0"]').click();
      await driver.waitAdvance(opening.version);
      await waitSettled(page);
      expect((await fxRecords(page)).anims.filter((a) => a.kind === 'deal')).toEqual([]);
      expect(await ageStartCues(page)).toEqual([]);
      await expect(page.locator(AGE), 'the summary stays open').toBeVisible();
      expect(problems).toEqual([]);
    });
  }

  test('the bot chooses: the viewer sees the deal, then the outcome', async ({ page, baseURL }) => {
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await configure(page, 'normal');
    const driver = await open(page, 'ai', baseURL);
    await clearFx(page);
    await watchCovers(page);
    await playThrough(driver, LAST_CARD);
    await driver.waitTurn();
    await expect(page.locator(AGE)).toBeVisible();
    const view = await driver.snapshot();
    expect(view.age).toBe(2);
    expect(view.phase, 'the bot has chosen').toBe('play');
    const [first] = await covers(page);
    expect(first?.what).toBe('dialog');
    expect(first!.at, 'the deal ended before the dialog').toBeGreaterThanOrEqual(
      dealEnd((await fxRecords(page)).anims, 'the bot room')
    );
    expect(first!.choosing, 'the dialog opens on the bot’s outcome').toBe(false);
    await expect(page.locator('dialog[open] [data-act^="starter:"]')).toHaveCount(0);
    expect(await ageStartCues(page), 'the age sounds once').toHaveLength(1);
    expect(problems).toEqual([]);
  });

  test('hot-seat pass-device: the pass screen waits for the deal', async ({ page, baseURL }) => {
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await configure(page, 'normal', true);
    const driver = await open(page, 'hotseat', baseURL);
    await clearFx(page);
    await watchCovers(page);
    await playThrough(driver, LAST_CARD);
    await expect(page.locator(`${COVERS.dialog}, ${COVERS.pass}`).first()).toBeVisible();
    const [first] = await covers(page);
    expect(first, 'something covers the board after the deal').toBeDefined();
    test.info().annotations.push({ type: 'first cover', description: first!.what });
    expect(first!.at, `the deal ended before the ${first!.what}`).toBeGreaterThanOrEqual(
      dealEnd((await fxRecords(page)).anims, 'pass device')
    );
    expect(problems).toEqual([]);
  });

  test('a reload during the choice does not deal the age again', async ({ page, baseURL }) => {
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await configure(page, 'normal');
    const driver = await open(page, 'hotseat', baseURL);
    await playThrough(driver, LAST_CARD);
    await expect(page.locator('dialog[open] [data-act="starter:0"]')).toBeVisible();
    await page.reload();
    await driver.waitReady();
    await expect(page.locator('dialog[open] [data-act="starter:0"]')).toBeVisible();
    await waitSettled(page);
    expect((await fxRecords(page)).anims, 'nothing replays after the reload').toEqual([]);
    expect(await ageStartCues(page)).toEqual([]);
    const opening = await driver.snapshot();
    await page.locator('dialog[open] [data-act="starter:0"]').click();
    await driver.waitAdvance(opening.version);
    await waitSettled(page);
    expect(
      (await fxRecords(page)).anims.filter((a) => a.kind === 'deal'),
      'the answer deals nothing'
    ).toEqual([]);
    expect(await ageStartCues(page)).toEqual([]);
    expect(problems).toEqual([]);
  });
});
