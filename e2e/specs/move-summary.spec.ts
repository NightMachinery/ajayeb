// The opponent move summary (M10.S03b), through the real UI on the desktop
// and mobile projects.
//
// - Against the bot: once the bot has moved, a popup lists its moves (the
//   bot's last run of history entries, consolidated); a tap on a card of
//   the structure closes it without acting (the card's sheet does not
//   open), and Escape closes it too. With Settings' "Show opponent move in
//   detail" off, the popup no longer opens by itself, but the top bar's
//   button opens the latest summary.
// - Hot-seat with the pass-device screen: the popup waits for the next
//   player to take the device; the pass screen replaces its automatic popup.
// - A card or wonder picture's tooltip (M11.track-and-log-polish) has the
//   board's detail: a fine pointer hovers it; on touch a long-press shows it and
//   keeps the popup, the next tap closes only the tooltip, and a tap then
//   closes the popup.
//
// - Phone landscape: the popup takes the first lane clear of the cards to
//   take (those the viewer's move uncovered included), both strips and the
//   turn bar.
//
// The base fixture turns the popup off for every other spec (its taps would
// eat a driver's clicks), so this spec turns it back on for its pages.

import { expect, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver, type Snapshot } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';

const POPUP = '[data-dismiss-layer] [data-move-summary]';

/** A finger resting on the element's centre for `ms`, then lifting. */
async function longPress(page: Page, target: Locator, ms = 800): Promise<void> {
  const box = await target.boundingBox();
  if (!box) throw new Error('element has no box');
  const touchPoints = [{ x: box.x + box.width / 2, y: box.y + box.height / 2 }];
  const cdp = await page.context().newCDPSession(page);
  try {
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints });
    await page.waitForTimeout(ms);
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  } finally {
    await cdp.detach();
  }
}

/** Turns the popup back on (its default) for every page load of this test. */
async function summaryOn(page: Page): Promise<void> {
  // A page-level init script runs after the base fixture's context-level one.
  await page.addInitScript(() => {
    try {
      const raw = localStorage.getItem('ajayeb:settings');
      const current = raw ? (JSON.parse(raw) as Record<string, unknown>) : {};
      current['showOpponentMove'] = true;
      localStorage.setItem('ajayeb:settings', JSON.stringify(current));
    } catch {
      // Storage may be blocked; the default is on anyway.
    }
  });
}

/** The plies of the other side's moves since `seat` last moved, as the page's history has them. */
async function pliesSince(page: Page, seat: number): Promise<string> {
  return page.evaluate((viewer) => {
    const history = window.__ajayeb?.history() ?? [];
    let i = history.length;
    while (i > 0 && history[i - 1]!.seat !== viewer) i--;
    return history
      .slice(i)
      .map((e) => e.ply)
      .join(' ');
  }, seat);
}

/** The seat of the latest history entry, or null before any move. */
async function lastMover(page: Page): Promise<number | null> {
  return page.evaluate(() => {
    const history = window.__ajayeb?.history() ?? [];
    return history[history.length - 1]?.seat ?? null;
  });
}

/**
 * What the element at `over` covers among the elements `targets` matches, by
 * bounding box, as "name WxH" lines; empty when it covers none.
 */
async function coveredBy(page: Page, over: string, targets: string): Promise<string[]> {
  return page.evaluate(
    ([over, targets]) => {
      const top = document.querySelector(over);
      if (!top) return [`nothing matches ${over}`];
      const a = top.getBoundingClientRect();
      const hits: string[] = [];
      for (const el of document.querySelectorAll(targets)) {
        const b = el.getBoundingClientRect();
        if (b.width <= 1 || b.height <= 1) continue;
        const w = Math.min(a.right, b.right) - Math.max(a.left, b.left);
        const h = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top);
        if (w <= 0.5 || h <= 0.5) continue;
        const name = el.getAttribute('data-card') ?? String(el.className).split(' ')[0];
        hits.push(`${name} ${Math.round(w)}x${Math.round(h)}`);
      }
      return hits;
    },
    [over, targets] as const
  );
}

/** Taps (on touch) or clicks a locator; `force` presses even a covered, inert card. */
async function press(
  page: Page,
  isMobile: boolean,
  selector: string,
  force = false
): Promise<void> {
  const target = page.locator(selector).filter({ visible: true }).first();
  if (isMobile) await target.tap({ force });
  else await target.click({ force });
}

/** Prefer a takeable card outside the popup for the swallowed board tap. */
async function dismissalCard(
  page: Page,
  snap: Snapshot
): Promise<{ card: string; force: boolean }> {
  const popup = await page
    .locator('[data-dismiss-layer]')
    .filter({ has: page.locator('[data-move-summary]') })
    .boundingBox();
  if (!popup) throw new Error('no move summary box');
  const cards = new Set(
    snap.choices.filter((c) => c.kind === 'construct' || c.kind === 'discard').map((c) => c.card)
  );
  let fallback: string | undefined;
  for (const card of cards) {
    if (!card) continue;
    const box = await page
      .locator(`.card-structure [data-card="${card}"]`)
      .filter({ visible: true })
      .first()
      .boundingBox();
    if (!box) throw new Error(`no takeable card box for ${card}`);
    fallback ??= card;
    const covered =
      box.x < popup.x + popup.width &&
      box.x + box.width > popup.x &&
      box.y < popup.y + popup.height &&
      box.y + box.height > popup.y;
    if (!covered) return { card, force: false };
  }
  if (!fallback) throw new Error('no card to take');
  // Every takeable card overlaps the popup. Only this fallback is forced;
  // it may land on the popup itself instead of exercising a board tap.
  return { card: fallback, force: true };
}

/** Plays this client's moves until the bot has just moved in play; returns the turn then. */
async function untilBotMoved(driver: GameDriver, bot: number): Promise<Snapshot> {
  for (let guard = 0; guard < 30; guard++) {
    const snap = await driver.nextTurn();
    if (snap.phase === 'play' && snap.pending === null && (await lastMover(driver.page)) === bot) {
      return snap;
    }
    // A popup over a choice of ours (a new age's starter) would eat the click.
    if ((await driver.page.locator(POPUP).count()) > 0) await driver.page.keyboard.press('Escape');
    await driver.play(snap);
  }
  throw new Error('the bot did not move in play within 30 turns');
}

test.describe('opponent move summary', { tag: ['@mobile'] }, () => {
  test('against the bot: shows after its move, a tap closes it without acting, the button reopens it', async ({
    page,
    isMobile
  }) => {
    test.setTimeout(150_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    await page.goto('/');
    await page.getByRole('button', { name: /Play vs AI/ }).click();
    const sheet = page.locator('dialog[open]');
    await sheet.getByText('Apprentice', { exact: true }).click();
    await sheet.getByText('Sun', { exact: true }).click();
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();

    // No popup in the wonder draft, even when the bot picks.
    const first = await driver.nextTurn();
    expect(first.phase).toBe('wonderDraft');
    await expect(page.locator(POPUP)).toHaveCount(0);

    // The bot has just moved: the popup lists its moves, as the history has them.
    const snap = await untilBotMoved(driver, 1);
    const popup = page.locator(POPUP);
    await expect(popup).toBeVisible();
    await expect(popup).toHaveAttribute('data-move-summary', '1');
    await expect(popup).toHaveAttribute('data-plies', await pliesSince(page, 0));
    await expect(popup.locator('[data-summary-line]').first()).toBeVisible();
    await expect(popup.locator('h2')).toContainText("'s move");

    // A tap on a takeable card closes it, and the card's sheet does not open.
    const { card, force } = await dismissalCard(page, snap);
    const cardSelector = `.card-structure [data-card="${card}"]`;
    await press(page, isMobile, cardSelector, force);
    await expect(page.locator(POPUP)).toHaveCount(0);
    await expect(page.locator('dialog[open] [data-card-sheet]')).toHaveCount(0);
    expect((await driver.snapshot()).version).toBe(snap.version);
    // The next tap acts as usual.
    await press(page, isMobile, cardSelector);
    await expect(page.locator(`dialog[open] [data-card-sheet="${card}"]`)).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open] [data-card-sheet]')).toHaveCount(0);

    // The button opens it again; Escape closes it.
    await press(page, isMobile, '[data-control="move-summary"]');
    await expect(page.locator(POPUP)).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator(POPUP)).toHaveCount(0);

    // Option off: the popup no longer opens by itself after the bot's move...
    // Settings through the game menu (a narrow phone's top bar has no gear).
    await press(page, isMobile, '[data-control="menu"]');
    await press(page, isMobile, '[data-control="menu-settings"]');
    const option = page.locator('dialog[open] [data-setting="show-opponent-move"]');
    await expect(option).toBeChecked();
    await option.setChecked(false);
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    const saved = await page.evaluate(() => localStorage.getItem('ajayeb:settings') ?? '{}');
    expect(JSON.parse(saved)).toMatchObject({ showOpponentMove: false });

    await driver.play(await driver.nextTurn());
    const later = await untilBotMoved(driver, 1);
    expect(later.ply).toBeGreaterThan(snap.ply);
    await page.waitForTimeout(300);
    await expect(page.locator(POPUP)).toHaveCount(0);

    // ...but the button near the log still shows the latest one.
    await press(page, isMobile, '[data-control="move-summary"]');
    await expect(page.locator(POPUP)).toBeVisible();
    await expect(page.locator(POPUP)).toHaveAttribute('data-plies', await pliesSince(page, 0));
    await page.keyboard.press('Escape');
    await expect(page.locator(POPUP)).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test('phone landscape: the popup keeps clear of the cards to take, those the move uncovered included, and of both strips', async ({
    page
  }) => {
    // M11.round3-board-fixes: at 844x390 the popup sat over the structure's
    // head and hid a card the viewer's own move had just uncovered. It now
    // takes the first lane clear of every card to take, both strips and the
    // turn bar (ui/toast-lane.ts, summaryLane).
    test.setTimeout(240_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    for (const { width, height } of [
      { width: 844, height: 390 },
      { width: 740, height: 360 }
    ]) {
      const at = `${width}x${height}`;
      await page.setViewportSize({ width, height });
      await page.goto('/dev/new?mode=ai&level=easy&seat=0&seed=7');
      await page.waitForURL(/\/play\/[A-Z0-9]+$/);
      const driver = new GameDriver(page);
      await driver.waitReady();
      let uncovered = 0;
      let before = new Set<string>();
      for (let guard = 0; uncovered < 2; guard++) {
        expect(guard, `${at}: two popups after a move that uncovered a card`).toBeLessThan(24);
        const snap = await driver.nextTurn();
        const takeable = new Set(
          snap.choices.flatMap((c) =>
            (c.kind === 'construct' || c.kind === 'discard') && c.card ? [c.card] : []
          )
        );
        if (snap.phase === 'play' && (await lastMover(page)) === 1) {
          const popup = page.locator(POPUP);
          await expect(popup, at).toBeVisible();
          const fresh = [...takeable].filter((card) => !before.has(card));
          if (fresh.length > 0) uncovered++;
          const targets = [
            '.area-home .player-strip',
            '.area-away .player-strip',
            '.area-turn',
            ...[...takeable].map((card) => `.card-structure [data-card="${card}"]`)
          ].join(', ');
          await expect
            .poll(() => coveredBy(page, '[data-dismiss-layer]:has([data-move-summary])', targets), {
              message: `${at}: covered (newly takeable: ${fresh.join(' ')})`,
              timeout: 3000
            })
            .toEqual([]);
          await expect(page.locator('[data-summary-lane]'), at).not.toHaveAttribute(
            'data-summary-lane',
            'none'
          );
          await page.keyboard.press('Escape');
          await expect(popup, at).toHaveCount(0);
        }
        before = takeable;
        await driver.play(snap);
      }
    }
    expect(problems).toEqual([]);
  });

  test('hot-seat: pass screen replaces the automatic popup, manual summary stays available', async ({
    page,
    isMobile
  }) => {
    test.setTimeout(150_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    await page.goto('/');
    await page.getByRole('button', { name: /^Pass and play/ }).click();
    const sheet = page.locator('dialog[open]');
    await expect(sheet.getByRole('heading', { name: 'Pass and play' })).toBeVisible();
    await sheet.getByRole('checkbox').setChecked(true);
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();

    // Play until a move in Age I hands the device to the other player.
    for (let guard = 0; guard < 30; guard++) {
      await driver.waitTurn();
      const snap = await driver.snapshot();
      if (snap.pass) {
        if (snap.phase === 'play') break;
        await driver.passDevice();
        continue;
      }
      await driver.play(snap);
    }
    const pass = page.locator('[data-pass-device]');
    await expect(pass).toBeVisible();
    // Not while the device is being passed...
    await expect(page.locator(POPUP)).toHaveCount(0);
    const next = Number(await pass.getAttribute('data-pass-device'));
    const previous = next === 0 ? 1 : 0;
    await driver.passDevice();

    // The pass screen already showed the move. The board is ready at once.
    const popup = page.locator(POPUP);
    await expect(popup).toHaveCount(0);
    await press(page, isMobile, '[data-control="move-summary"]');
    await expect(popup).toBeVisible();
    await expect(popup).toHaveAttribute('data-move-summary', String(previous));
    await expect(popup).toHaveAttribute('data-plies', await pliesSince(page, next));

    // Any tap closes it without acting.
    const version = (await driver.snapshot()).version;
    await press(page, isMobile, '.card-structure [data-card]', true);
    await expect(page.locator(POPUP)).toHaveCount(0);
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    expect((await driver.snapshot()).version).toBe(version);
    expect(problems).toEqual([]);
  });
});

test.describe('opponent move summary: card tooltips', { tag: ['@mobile'] }, () => {
  test('a card picture shows the card detail on hover, or on a long-press that keeps the popup', async ({
    page,
    isMobile
  }) => {
    const problems = watchHygiene(page);
    await page.goto('/dev/fixture/state-join-history');
    await press(page, isMobile, '[data-control="move-summary"]');
    const popup = page.locator(POPUP);
    await expect(popup).toBeVisible();
    const thumb = popup.locator('[data-summary-line] [data-card]').first();
    const slug = await thumb.getAttribute('data-card');
    if (isMobile) await longPress(page, thumb);
    else await thumb.hover();

    // The board's card tooltip: the effects, the printed cost, the chains,
    // with the card's face beside it.
    // The group has no box of its own (its parts are placed absolutely).
    const open = page.locator('[data-tooltip-layer] [data-tooltip-open]');
    await expect(open.locator('[data-tooltip-bubble]')).toBeVisible();
    await expect(open.locator('[data-tooltip-lines]')).toContainText('Printed cost');
    await expect(open.locator(`[data-card-preview="${slug}"]`)).toBeVisible();
    await expect(popup).toBeVisible();

    if (isMobile) {
      // The next tap closes only the tooltip; a tap then closes the popup.
      await page.touchscreen.tap(5, 5);
      await expect(open).toHaveCount(0);
      await expect(popup).toBeVisible();
      await press(page, isMobile, `${POPUP} [data-summary-line] [data-card]`);
      await expect(page.locator(POPUP)).toHaveCount(0);
      await expect(open).toHaveCount(0);
    }
    expect(problems).toEqual([]);
  });

  test('a wonder picture shows the wonder detail, not just its name', async ({
    page,
    isMobile
  }) => {
    const problems = watchHygiene(page);
    // The gallery's extra-turn scene opens with Piraeus.
    await page.goto('/dev/gallery');
    await press(page, isMobile, '[data-summary-scene="game.gallery.summary.extra"]');
    const popup = page.locator(POPUP);
    await expect(popup).toBeVisible();
    const thumb = popup.locator('[data-summary-line] [data-wonder="piraeus"]');
    if (isMobile) await longPress(page, thumb);
    else await thumb.hover();

    // The board's wonder tooltip: the effects and the printed cost.
    const open = page.locator('[data-tooltip-layer] [data-tooltip-open]');
    await expect(open.locator('[data-tooltip-bubble]')).toContainText('Piraeus');
    const lines = open.locator('[data-tooltip-lines]');
    await expect(lines).toContainText('glass');
    await expect(lines).toContainText('Printed cost');
    await expect(popup).toBeVisible();
    expect(problems).toEqual([]);
  });
});
