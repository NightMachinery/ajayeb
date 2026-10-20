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
// records (window.__ajayebFx.anims()). Two harder paths: a swap whose render
// eats the whole nominal hold, and an opening frame the viewer never sees
// (its deal recovered on the first play frame) with the move summary on.
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
import { GameDriver } from '../fixtures/driver';
import { newDeviceContext } from '../fixtures/online';
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
  anims: AnimRecord[];
}

/**
 * Starts noting the first appearance of each surface that covers the board.
 * `slow` blocks the main thread for that long when the starter host first
 * mounts, in the swap's own DOM update: a render that eats the whole nominal
 * hold before the runner starts the deal.
 */
async function watchCovers(page: Page, slow = 0): Promise<void> {
  await page.evaluate(
    ({ covers, slow }) => {
      const seen: Cover[] = [];
      let slowed = false;
      const animations = () =>
        (window as unknown as { __ajayebFx?: { anims(): AnimRecord[] } }).__ajayebFx?.anims() ?? [];
      const native = HTMLDialogElement.prototype.showModal;
      HTMLDialogElement.prototype.showModal = function () {
        if (
          this.querySelector('[data-age-interstitial]') &&
          !seen.some((c) => c.what === 'dialog')
        ) {
          seen.push({
            what: 'dialog',
            at: performance.now(),
            choosing: !!this.closest('[data-choice-host="chooseStarter"]'),
            anims: animations()
          });
        }
        native.call(this);
      };
      const note = () => {
        if (slow > 0 && !slowed && document.querySelector('[data-choice-host="chooseStarter"]')) {
          slowed = true;
          const end = performance.now() + slow;
          while (performance.now() < end) {
            // Busy: the swap's render takes this long.
          }
        }
        for (const [what, selector] of Object.entries(covers)) {
          if (seen.some((c) => c.what === what) || !document.querySelector(selector)) continue;
          seen.push({
            what,
            at: performance.now(),
            choosing: !!document.querySelector('[data-choice-host="chooseStarter"] dialog[open]'),
            anims: animations()
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
    },
    { covers: COVERS, slow }
  );
}

async function covers(page: Page): Promise<Cover[]> {
  return page.evaluate(() => (window as unknown as { __covers: Cover[] }).__covers);
}

/** The game's own settings for this spec, with sound on so cues are recorded. */
async function configure(
  page: Page,
  speed: 'normal' | 'off',
  pass = false,
  summaries = false
): Promise<void> {
  await useSettings(page, speed);
  await page.evaluate(
    ({ pass, summaries }) => {
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({
          ...settings,
          sound: true,
          hotseatPassDevice: pass,
          showOpponentMove: summaries
        })
      );
    },
    { pass, summaries }
  );
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

/** The opening's end, including counters and pulses as well as every dealt card. */
function openingEnd(anims: AnimRecord[], label: string): number {
  const deals = anims.filter((a) => a.kind === 'deal');
  expect(deals.length, `${label}: the new age was dealt`).toBeGreaterThan(0);
  for (const deal of deals) {
    expect(deal.interrupted, `${label}: a dealt card was cut short`).toBe(false);
    expect(deal.endAt, `${label}: a dealt card never landed`).not.toBeNull();
  }
  const versions = new Set(deals.map((d) => d.version));
  const opening = anims.filter((a) => versions.has(a.version));
  for (const run of opening) {
    expect(run.interrupted, `${label}: ${run.kind} was cut short`).toBe(false);
    expect(run.endAt, `${label}: ${run.kind} still ran at the cover`).not.toBeNull();
  }
  return Math.max(...opening.map((a) => a.endAt!));
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
        const end = openingEnd(first!.anims, 'the opening at native modal entry');
        expect(
          first!.at,
          'every opening effect ended before the choice opened'
        ).toBeGreaterThanOrEqual(end);
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
      openingEnd((await fxRecords(page)).anims, 'the bot room')
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
      openingEnd((await fxRecords(page)).anims, 'pass device')
    );
    expect(problems).toEqual([]);
  });

  test('a slow swap that eats the hold still waits for the deal', async ({ page, baseURL }) => {
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await configure(page, 'normal');
    const driver = await open(page, 'hotseat', baseURL);
    await clearFx(page);
    await watchCovers(page, 1400);
    await playThrough(driver, LAST_CARD);
    await expect(page.locator(AGE)).toBeVisible();
    const [first] = await covers(page);
    expect(first?.what).toBe('dialog');
    expect(first!.at, 'the deal ended before the dialog').toBeGreaterThanOrEqual(
      openingEnd((await fxRecords(page)).anims, 'slow swap')
    );
    expect(problems).toEqual([]);
  });

  test('a skipped opening frame: the recovered deal comes before the move summary', async ({
    browser,
    baseURL
  }, info) => {
    if (!baseURL) throw new Error('no baseURL');
    const a = await newDeviceContext(browser, info),
      b = await newDeviceContext(browser, info);
    try {
      const pageA = await a.newPage(),
        pageB = await b.newPage();
      const problems = [watchHygiene(pageA), watchHygiene(pageB)];
      // Both devices: Animation normal, the move summary on, two names.
      for (const [page, name] of [
        [pageA, 'First'],
        [pageB, 'Second']
      ] as const) {
        await configure(page, 'normal', false, true);
        await page.evaluate((name) => localStorage.setItem('ajayeb:name', name), name);
      }
      // The viewer is whoever takes the last card of Age I. Its opening
      // frame of Age II is dropped at the socket and its request's ack
      // handed to the next frame, so the deal is recovered on the first
      // play frame, the one that answers the other device's starter choice.
      let viewer: Page | null = null;
      let dropped: unknown;
      let skipped = false;
      for (const page of [pageA, pageB]) {
        await page.routeWebSocket('**/ws', (socket) => {
          const server = socket.connectToServer();
          server.onMessage((raw) => {
            const frame = JSON.parse(String(raw)) as {
              t: string;
              ack?: unknown;
              view?: { age: number; phase: string };
            };
            if (page !== viewer || frame.t !== 'state') return socket.send(raw);
            if (frame.view?.age === 2 && frame.view.phase === 'chooseStarter') {
              dropped = frame.ack;
              skipped = true;
              return;
            }
            if (dropped !== undefined) {
              frame.ack = dropped;
              dropped = undefined;
              socket.send(JSON.stringify(frame));
            } else socket.send(raw);
          });
        });
      }
      await pageA.goto(`/dev/new?mode=online&seed=11&scenario=${SCENARIO}`);
      await pageA.waitForURL(/\/play\/[A-Z0-9]+$/);
      await pageB.goto(pageA.url());
      const da = new GameDriver(pageA),
        db = new GameDriver(pageB);
      await da.waitReady();
      await db.waitReady();
      const snap = await da.snapshot();
      viewer = snap.seats.includes(snap.toMove ?? -1) ? pageA : pageB;
      const chooser = viewer === pageA ? pageB : pageA;
      await clearFx(viewer);
      await watchCovers(viewer);
      await viewer.locator('.card-structure [data-card="theater"]').first().click();
      await viewer.locator('[data-act="discard:theater"]:visible').first().click();
      // The chooser may first see the viewer's move in its own summary.
      const starter = chooser.locator('dialog[open] [data-act^="starter:"]').first();
      const popup = chooser.locator(COVERS.summary);
      await expect(starter.or(popup)).toBeVisible();
      if (await popup.isVisible()) await chooser.keyboard.press('Escape');
      await starter.click();
      await expect(viewer.locator(COVERS.summary), 'the summary still opens').toBeVisible();
      expect(skipped, 'the opening frame was dropped').toBe(true);
      const all = await covers(viewer);
      const end = openingEnd((await fxRecords(viewer)).anims, 'recovered deal');
      for (const cover of all) {
        expect(cover.at, `the ${cover.what} waited for the deal`).toBeGreaterThanOrEqual(end);
      }
      expect(await ageStartCues(viewer), 'the age sounds once').toHaveLength(1);
      await viewer.keyboard.press('Escape');
      await expect(viewer.locator(AGE), 'the age dialog follows the summary').toBeVisible();
      expect(problems).toEqual([[], []]);
    } finally {
      await a.close();
      await b.close();
    }
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
