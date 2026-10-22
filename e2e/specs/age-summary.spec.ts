// Completed ages are presentation history, independent of who chooses or moves next.
import { expect, type Page, type WebSocketRoute } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { joinByLink, newDeviceContext } from '../fixtures/online';
import { currentView, openHotseatRoom, playKey } from '../fixtures/scenario';
import { createDevRoom } from '../fixtures/ws-game';
import { watchHygiene } from '../fixtures/hygiene';
import { useSettings } from '../fixtures/anim';
import {
  fixtureNames,
  fixtures,
  protocolFixtureNames,
  protocolFixtures
} from '../../web/src/lib/wire/fixtures.gen';
import type { State } from '../../web/src/lib/wire';

const AGE = 'dialog[open] [data-age-interstitial]';
const POPUP = '[data-dismiss-layer] [data-move-summary]';

async function configure(page: Page, summary = false, pass = false): Promise<void> {
  await page.addInitScript(
    ({ summary, pass }) => {
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({
          ...settings,
          showOpponentMove: summary,
          confirmMoves: 'never',
          hotseatPassDevice: pass
        })
      );
    },
    { summary, pass }
  );
}

// Relay unchanged messages to the real server, dropping only this socket
// when a reconnect is under test. HTTP art fetches remain available, and a
// paused reconnect cannot produce an intentional browser network error.
async function socketGate(page: Page) {
  let paused = false;
  let client: WebSocketRoute | undefined;
  let server: WebSocketRoute | undefined;
  const waiting: (() => void)[] = [];
  const frames: State[] = [];
  const reconnect = () => {
    paused = false;
    for (const release of waiting.splice(0)) release();
  };
  page.once('close', reconnect);
  await page.routeWebSocket('**/ws', async (socket) => {
    if (paused) await new Promise<void>((release) => waiting.push(release));
    if (page.isClosed()) return;
    client = socket;
    server = socket.connectToServer();
    server.onMessage((raw) => {
      const frame = JSON.parse(String(raw)) as { t: string; v?: number };
      if (frame.t === 'state') frames.push(frame as State);
      socket.send(raw);
    });
  });
  return {
    frames,
    reconnect,
    async disconnect() {
      paused = true;
      if (!client || !server) throw Error('no real socket to disconnect');
      await Promise.all([
        client.close({ code: 1012, reason: 'test reconnect' }),
        server.close({ code: 1000, reason: 'test reconnect' })
      ]);
    }
  };
}

// Match the existing player-flow wake probe: frames keep arriving while
// the page is hidden, then the real visibility event wakes presentation.
async function visible(page: Page, show: boolean): Promise<void> {
  await page.evaluate((show) => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => (show ? 'visible' : 'hidden')
    });
    document.dispatchEvent(new Event('visibilitychange'));
  }, show);
}

function recordedBoundary(): State {
  const first = structuredClone(
    fixtures[(fixtureNames as readonly string[]).indexOf('state-play')]
  ) as State;
  first.code = 'FIXTUR';
  first.version = 100;
  first.view = structuredClone(
    protocolFixtures[protocolFixtureNames.indexOf('choose-starter-seat0')]
  )!;
  first.events = [];
  delete first.history;
  delete first.ack;
  first.room = { ...first.room, mode: 'online', status: 'playing', allowUndo: false };
  first.you = { seats: [0], canAct: true, admin: true };
  first.room.seats = first.room.seats.map((seat) => ({ ...seat, you: seat.seat === 0 }));
  first.canUndo = false;
  return first;
}

async function recordedRoom(page: Page, first: State): Promise<(frame: State) => void> {
  await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'First'));
  let deliver: ((frame: State) => void) | undefined;
  await page.routeWebSocket('**/ws', (socket) => {
    deliver = (frame) => socket.send(JSON.stringify(frame));
    socket.onMessage((raw) => {
      const message = JSON.parse(String(raw)) as { t: string; v?: number };
      if (message.t === 'hello')
        socket.send(
          JSON.stringify({ t: 'welcome', v: message.v, server: { version: 'test', rules: 1 } })
        );
      if (message.t === 'join') deliver!(first);
    });
  });
  await page.goto('/play/FIXTUR');
  await expect(page.locator(AGE)).toBeVisible();
  return (frame) => deliver!(frame);
}

async function totals(page: Page): Promise<string[]> {
  return page.locator(`${AGE} [data-seat-summary] dd`).allTextContents();
}

async function discard(driver: GameDriver): Promise<void> {
  const snap = await driver.snapshot();
  const choice = snap.choices.find((choice) => choice.kind === 'discard');
  if (!choice) throw Error('scenario has no discard');
  await driver.perform({ flow: 'discard', choice }, snap);
  await driver.waitAdvance(snap.version);
}

async function footerFits(page: Page): Promise<void> {
  const body = page.locator('dialog[open] [data-sheet-body]');
  const footer = page.locator('dialog[open] [data-sheet-footer]');
  await expect(footer).toBeVisible();
  expect((await body.boundingBox())!.height).toBeGreaterThanOrEqual(44);
  for (const control of await footer.locator('button').all()) {
    const geometry = await control.evaluate((element) => {
      const rect = element.getBoundingClientRect();
      const panel = element.closest('.sheet-panel')!.getBoundingClientRect();
      const hit = document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2);
      return {
        width: rect.width,
        height: rect.height,
        inside: rect.top >= panel.top && rect.bottom <= panel.bottom,
        hit: !!hit && element.contains(hit)
      };
    });
    expect(geometry).toMatchObject({ inside: true, hit: true });
    expect(geometry.width).toBeGreaterThanOrEqual(44);
    expect(geometry.height).toBeGreaterThanOrEqual(44);
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
}

async function expectNoAge(page: Page): Promise<void> {
  await expect(page.locator(AGE)).toHaveCount(0);
}

async function continueAge(page: Page): Promise<void> {
  await page.locator('dialog[open] [data-age-continue]').click();
  await expectNoAge(page);
}

test.describe('completed-age summary', { tag: '@mobile' }, () => {
  for (const speed of ['off', 'normal'] as const) {
    test(`a real bot chooses without closing the completed age (${speed})`, async ({ page }) => {
      const problems = watchHygiene(page);
      await configure(page);
      await useSettings(page, speed);
      const network = await socketGate(page);
      await page.goto('/dev/new?mode=ai&seat=0&seed=7&scenario=age-transition-chooser');
      await page.waitForURL(/\/play\/[A-Z0-9]+$/);
      const driver = new GameDriver(page);
      await driver.waitReady();
      await page.evaluate(() => {
        const watch = { seen: false, closed: false };
        Object.assign(window, { __ageDialogWatch: watch });
        const observer = new MutationObserver(() => {
          const open = !!document.querySelector('dialog[open] [data-age-interstitial]');
          if (watch.seen && !open) watch.closed = true;
          if (open) watch.seen = true;
        });
        observer.observe(document.body, {
          subtree: true,
          childList: true,
          attributes: true,
          attributeFilter: ['open']
        });
      });
      await playKey(driver, 'discard:theater');
      await driver.waitTurn();
      await expect(page.locator(AGE)).toBeVisible();
      await expect(page.locator(AGE)).toHaveAttribute('data-age-interstitial', '1');
      expect((await currentView(page)).pending).toBeUndefined();
      expect(
        await page.evaluate(
          () =>
            (window as unknown as { __ageDialogWatch: { closed: boolean } }).__ageDialogWatch.closed
        )
      ).toBe(false);
      const boundary = network.frames.find(
        (frame) => frame.view.age === 2 && frame.view.pending?.kind === 'chooseStarter'
      )?.view;
      expect(boundary, 'the bot really received and resolved a starter choice').toBeTruthy();
      expect((await totals(page)).map((value) => value.trim())).toEqual(
        [0, 1].flatMap((seat) => [
          String(boundary!.players[seat]!.coins),
          String(boundary!.scorePreview[seat]!.total)
        ])
      );
      await footerFits(page);
      await continueAge(page);
      // Reconnect/resync at the same position cannot undo explicit dismissal.
      await network.disconnect();
      await expect.poll(() => page.evaluate(() => window.__ajayeb?.conn())).not.toBe('open');
      network.reconnect();
      await driver.waitReady();
      await expectNoAge(page);
      expect(problems).toEqual([]);
    });
  }

  for (const speed of ['off', 'normal'] as const) {
    test(`summaries on: the recap opens before the summary of the bot's starter choice and its new-age move (${speed})`, async ({
      page
    }) => {
      // M11.round3-board-fixes: the bot chose and played into the new age
      // while the viewer's screen caught up, and its summary opened first,
      // then a recap with "Begin Age II". With seed 1 the bot chooses itself
      // and takes the first card of Age II at once.
      const problems = watchHygiene(page);
      await configure(page, true);
      await useSettings(page, speed);
      await page.goto('/dev/new?mode=ai&seat=0&seed=1&scenario=age-transition-chooser');
      await page.waitForURL(/\/play\/[A-Z0-9]+$/);
      const driver = new GameDriver(page);
      await driver.waitReady();
      await playKey(driver, 'discard:theater');
      await expect
        .poll(async () => {
          const view = await currentView(page);
          return view.age === 2 && !view.pending && view.toMove === 0;
        })
        .toBe(true);
      await driver.waitTurn();
      const history = await page.evaluate(() => window.__ajayeb?.history() ?? []);
      const chosen = history.flatMap((e) => e.events).find((e) => e.kind === 'starterChosen');
      expect(chosen?.starter, 'the bot chose itself').toBe(1);
      expect((await currentView(page)).structure.some((slot) => slot.state === 'empty')).toBe(true);
      await expect(page.locator(AGE)).toBeVisible();
      await expect(page.locator(POPUP)).toHaveCount(0);
      await expect(page.locator('dialog[open] [data-age-continue]')).toHaveText('Continue');
      await continueAge(page);
      const popup = page.locator(POPUP);
      await expect(popup).toBeVisible();
      await expect(popup.locator('[data-summary-line]').first()).toHaveAttribute(
        'data-summary-line',
        'ageStart'
      );
      await expect(popup.locator('[data-summary-line="starterChosen"]')).toHaveCount(1);
      await expect(popup.locator('[data-summary-line]')).toHaveCount(3);
      expect(problems).toEqual([]);
    });

    test(`summaries on: a reload while the recap holds back a summary still opens that summary (${speed})`, async ({
      page
    }) => {
      // M11.round3-board-fixes review: the summary waiting behind the recap
      // was marked seen before it ever opened, so a reload lost it. The
      // recap itself cannot come back (its ended-age totals are not in the
      // history), but the summary must, and the age must not deal again.
      const problems = watchHygiene(page);
      await configure(page, true);
      await useSettings(page, speed);
      await page.goto('/dev/new?mode=ai&seat=0&seed=7&scenario=age-transition-chooser');
      await page.waitForURL(/\/play\/[A-Z0-9]+$/);
      const driver = new GameDriver(page);
      await driver.waitReady();
      await playKey(driver, 'discard:theater');
      await expect
        .poll(async () => {
          const view = await currentView(page);
          return view.age === 2 && !view.pending && view.toMove === 0;
        })
        .toBe(true);
      await driver.waitTurn();
      await expect(page.locator(AGE)).toBeVisible();
      await expect(page.locator(POPUP)).toHaveCount(0);
      await page.waitForTimeout(300);
      const seen = await page.evaluate(() => localStorage.getItem('ajayeb:seen-moves') ?? '[]');
      expect(
        (JSON.parse(seen) as [string, number][]).every(([, ply]) => ply < 1),
        `the starter choice (ply 1) is not seen yet: ${seen}`
      ).toBe(true);
      await page.reload();
      await driver.waitReady();
      const popup = page.locator(POPUP);
      await expect(popup).toBeVisible();
      await expect(popup).toHaveAttribute('data-plies', '1');
      await expect(popup.locator('[data-summary-line="starterChosen"]')).toHaveCount(1);
      const deals = await page.evaluate(
        () =>
          (window as unknown as { __ajayebFx?: { anims(): { kind: string }[] } }).__ajayebFx
            ?.anims()
            .filter((a) => a.kind === 'deal').length ?? 0
      );
      expect(deals, 'the reload deals nothing').toBe(0);
      await page.keyboard.press('Escape');
      await expect(popup).toHaveCount(0);
      expect(problems).toEqual([]);
    });
  }

  test('game over cannot replace an unread completed age, and Continue opens the result', async ({
    page
  }) => {
    // A transport-level presentation regression built from recorded,
    // redacted server snapshots. Real bot/online rules are covered above.
    const problems = watchHygiene(page);
    await configure(page);
    await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'First'));
    const template = structuredClone(
      fixtures[(fixtureNames as readonly string[]).indexOf('state-play')]
    ) as State;
    const boundary = structuredClone(
      protocolFixtures[protocolFixtureNames.indexOf('choose-starter-seat0')]
    )!;
    const first: State = { ...template, code: 'FIXTUR', version: 100, view: boundary, events: [] };
    delete first.history;
    delete first.ack;
    first.room = { ...first.room, mode: 'online', status: 'playing' };
    first.you = { seats: [0], canAct: true, admin: true };
    first.room.seats = first.room.seats.map((seat) => ({ ...seat, you: seat.seat === 0 }));
    first.room.allowUndo = false;
    first.canUndo = false;
    const over = structuredClone(first);
    over.version++;
    over.view = structuredClone(
      protocolFixtures[protocolFixtureNames.indexOf('game-over-military-hidden-seat0')]
    )!;
    over.view.age = boundary.age;
    over.view.ply = boundary.ply + 1;
    over.room.status = 'finished';
    over.you.canAct = false;
    let deliver: ((frame: State) => void) | undefined;
    await page.routeWebSocket('**/ws', (socket) => {
      deliver = (frame) => socket.send(JSON.stringify(frame));
      socket.onMessage((raw) => {
        const message = JSON.parse(String(raw)) as { t: string; v?: number };
        if (message.t === 'hello')
          socket.send(
            JSON.stringify({ t: 'welcome', v: message.v, server: { version: 'test', rules: 1 } })
          );
        if (message.t === 'join') deliver!(first);
      });
    });
    await page.goto('/play/FIXTUR');
    await expect(page.locator(AGE)).toBeVisible();
    const frozen = await totals(page);
    expect(deliver).toBeDefined();
    deliver!(over);
    await expect.poll(() => page.evaluate(() => window.__ajayeb?.view()?.phase)).toBe('gameOver');
    await expect(page.locator(AGE)).toBeVisible();
    expect(await totals(page)).toEqual(frozen);
    await expect(page.locator('dialog[open]')).toHaveCount(1);
    await expect(page.locator('dialog[open] #game-over-title')).toHaveCount(0);
    await expect(page.locator('dialog[open] [data-age-continue]')).toHaveText('Continue');
    await expect(page.locator('dialog[open] [data-begin-age]')).toHaveCount(0);
    await continueAge(page);
    await expect(page.locator('dialog[open] #game-over-title')).toBeVisible();
    await expect(page.locator('dialog[open]')).toHaveCount(1);
    await footerFits(page);
    const home = page.locator('dialog[open] [data-home]');
    await home.focus();
    await page.locator('dialog[open]').evaluate((dialog) => {
      dialog.addEventListener('close', () => dialog.setAttribute('data-unexpected-close', 'true'));
    });
    deliver!({ ...over, version: over.version + 1, events: [] });
    await expect
      .poll(() => page.evaluate(() => window.__ajayeb?.room()?.version))
      .toBe(over.version + 1);
    await expect(home).toBeFocused();
    await expect(page.locator('dialog[open]')).not.toHaveAttribute('data-unexpected-close', 'true');
    // Native close can keep focus inside its closing detail body for a
    // frame. It must return quietly once the result leaves the top layer.
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(page.locator('[data-control="menu"]').filter({ visible: true })).toBeFocused();
    await expect(page.locator('[data-tooltip-open]')).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  for (const finished of [false, true]) {
    test(`queued ages retain older totals and expose only matching controls (finished ${finished})`, async ({
      page
    }, info) => {
      const problems = watchHygiene(page);
      await configure(page);
      const first = recordedBoundary();
      const deliver = await recordedRoom(page, first);
      const frozen = await totals(page);
      const later = structuredClone(first);
      later.version++;
      later.view.age++;
      later.view.ply += 21;
      later.view.players[0].coins += 17;
      deliver(later);
      await expect
        .poll(() => page.evaluate(() => window.__ajayeb?.room()?.version))
        .toBe(later.version);
      await expect(page.locator(AGE)).toHaveAttribute('data-age-interstitial', '1');
      expect(await totals(page)).toEqual(frozen);
      await expect(page.locator('dialog[open] [data-age-continue]')).toHaveText('Continue');
      await expect(page.locator('dialog[open] [data-begin-age]')).toHaveCount(0);
      await expect(page.locator('dialog[open] [data-act^="starter:"]')).toHaveCount(0);
      await expect(page.locator('dialog[open]')).toHaveCount(1);
      await info.attach('older-age-recap', {
        body: await page.screenshot({ path: info.outputPath('older-age-recap.png') }),
        contentType: 'image/png'
      });
      const over = structuredClone(later);
      if (finished) {
        over.version++;
        over.view = structuredClone(
          protocolFixtures[protocolFixtureNames.indexOf('game-over-military-hidden-seat0')]
        )!;
        over.view.age = later.view.age;
        over.view.ply = later.view.ply + 1;
        over.room.status = 'finished';
        over.you.canAct = false;
        deliver(over);
        await expect
          .poll(() => page.evaluate(() => window.__ajayeb?.view()?.phase))
          .toBe('gameOver');
        expect(await totals(page)).toEqual(frozen);
        await expect(page.locator('dialog[open] #game-over-title')).toHaveCount(0);
      }
      await page.locator('dialog[open] [data-age-continue]').click();
      await expect(page.locator(AGE)).toHaveAttribute('data-age-interstitial', '2');
      const second = await totals(page);
      expect(second).not.toEqual(frozen);
      await expect(page.locator('dialog[open]')).toHaveCount(1);
      if (finished) {
        await expect(page.locator('dialog[open] [data-age-continue]')).toHaveText('Continue');
        await expect(page.locator('dialog[open] [data-begin-age]')).toHaveCount(0);
        await page.locator('dialog[open]').evaluate((dialog) => {
          dialog.addEventListener('close', () =>
            dialog.setAttribute('data-unexpected-close', 'true')
          );
        });
        deliver({ ...over, version: over.version + 1, events: [] });
        await expect
          .poll(() => page.evaluate(() => window.__ajayeb?.room()?.version))
          .toBe(over.version + 1);
        await expect(page.locator(AGE)).toHaveAttribute('data-age-interstitial', '2');
        await expect(page.locator('dialog[open]')).not.toHaveAttribute(
          'data-unexpected-close',
          'true'
        );
        await continueAge(page);
        await expect(page.locator('dialog[open] #game-over-title')).toBeVisible();
      } else {
        await expect(page.locator('dialog[open] [data-act="starter:0"]')).toBeVisible();
        await expect(page.locator('dialog[open] [data-age-continue]')).toHaveText(
          'Peek at the board'
        );
        const picked = structuredClone(later);
        picked.version++;
        picked.view.ply++;
        picked.view.phase = 'play';
        picked.view.active = 0;
        picked.view.toMove = 0;
        picked.view.choices = [];
        delete picked.view.pending;
        picked.events = [{ kind: 'starterChosen', seat: 0, starter: 0, private: false }];
        deliver(picked);
        await expect(page.locator('dialog[open] [data-begin-age="3"]')).toBeVisible();
        expect(await totals(page)).toEqual(second);
        await continueAge(page);
      }
      expect(problems).toEqual([]);
    });
  }

  for (const closeBeforeChoice of [false, true]) {
    test(`online opponent chooses: local close is explicit and survives later frames (close early ${closeBeforeChoice})`, async ({
      browser,
      baseURL
    }, info) => {
      if (!baseURL) throw Error('age summary requires local baseURL');
      const a = await newDeviceContext(browser, info),
        b = await newDeviceContext(browser, info);
      try {
        const pageA = await a.newPage(),
          pageB = await b.newPage();
        const problemsA = watchHygiene(pageA),
          problemsB = watchHygiene(pageB);
        const networkA = await socketGate(pageA),
          networkB = await socketGate(pageB);
        await configure(pageA, !closeBeforeChoice);
        await configure(pageB, !closeBeforeChoice);
        await pageA.addInitScript(() => localStorage.setItem('ajayeb:name', 'First'));
        await pageA.goto('/dev/new?mode=online&seed=7&scenario=age-transition-chooser');
        await pageA.waitForURL(/\/play\/[A-Z0-9]+$/);
        await joinByLink(pageB, pageA.url(), 'Second');
        const da = new GameDriver(pageA),
          db = new GameDriver(pageB);
        await da.waitReady();
        await db.waitReady();
        if (!closeBeforeChoice) await visible(pageB, false);
        await discard(da);
        // A hidden boundary cannot open/mark the age summary before wake.
        if (!closeBeforeChoice) {
          await db.waitReady();
          await expect(pageB.locator('dialog[open]')).toHaveCount(0);
          await expect(pageB.locator(POPUP)).toHaveCount(0);
          await visible(pageB, true);
          await expect(pageB.locator(POPUP)).toBeVisible();
          await expect(pageB.locator('dialog[open]')).toHaveCount(0);
          await pageB.keyboard.press('Escape');
        }
        await expect(pageA.locator(AGE)).toBeVisible();
        await expect(pageB.locator(AGE)).toBeVisible();
        const frozen = await totals(pageA);
        if (!closeBeforeChoice) {
          await visible(pageB, false);
          await expect(pageB.locator(AGE)).toBeVisible();
          await visible(pageB, true);
          await expect(pageB.locator(AGE)).toBeVisible();
          await expect(pageB.locator(POPUP)).toHaveCount(0);
        }
        if (closeBeforeChoice) await continueAge(pageA);
        await pageB.locator('dialog[open] [data-act="starter:1"]').click();
        await db.waitReady();
        await da.waitReady();
        await expect(pageB.locator(AGE)).toBeVisible();
        await expect(pageB.locator('[data-begin-age="2"]')).toBeVisible();
        if (closeBeforeChoice) await expectNoAge(pageA);
        else {
          await expect(pageA.locator(AGE)).toBeVisible();
          expect(await totals(pageA)).toEqual(frozen);
          await expect(pageA.locator(POPUP)).toHaveCount(0);
          // A same-position history resync must keep the open summary.
          await networkA.disconnect();
          await expect.poll(() => pageA.evaluate(() => window.__ajayeb?.conn())).not.toBe('open');
          networkA.reconnect();
          await da.waitReady();
          await expect(pageA.locator(AGE)).toBeVisible();
          expect(await totals(pageA)).toEqual(frozen);
        }
        await continueAge(pageB);
        await discard(db);
        await da.waitReady();
        if (!closeBeforeChoice) {
          await expect(pageA.locator(AGE)).toBeVisible();
          expect(await totals(pageA)).toEqual(frozen);
          await continueAge(pageA);
          await expect(pageA.locator(POPUP)).toBeVisible();
          await pageA.keyboard.press('Escape');
        }
        await networkB.disconnect();
        await expect(
          pageA.locator('[data-presence="away"]').filter({ visible: true }).first()
        ).toBeVisible();
        await da.waitReady();
        await expectNoAge(pageA);
        networkB.reconnect();
        await db.waitReady();
        await da.waitReady();
        await expect(pageA.locator('[data-presence="away"]')).toHaveCount(0);
        await expectNoAge(pageA);

        if (!closeBeforeChoice) {
          // A fresh room clears the old cache. At game over no summary
          // popup opens: the result dialog alone carries the final move
          // (M11.gameover-last-move), once the page is visible again.
          const room = await createDevRoom(baseURL, {
            mode: 'online',
            seed: '7',
            scenario: 'civilian-end'
          });
          await pageA.goto(room.path);
          await da.waitReady();
          await pageB.goto(room.path);
          await db.waitReady();
          await expectNoAge(pageA);
          await expectNoAge(pageB);
          await visible(pageB, false);
          await discard(da);
          await db.waitReady();
          await expect(pageB.locator('dialog[open]')).toHaveCount(0);
          await expect(pageB.locator(POPUP)).toHaveCount(0);
          await visible(pageB, true);
          await expect(pageB.locator('dialog[open] #game-over-title')).toBeVisible();
          await expect(pageB.locator(POPUP)).toHaveCount(0);
          await expect(
            pageB.locator('dialog[open] [data-last-move="0"] [data-summary-line="discard"]')
          ).toBeVisible();
          await footerFits(pageB);
        }
        expect(problemsA).toEqual([]);
        expect(problemsB).toEqual([]);
      } finally {
        await a.close();
        await b.close();
      }
    });
  }

  for (const pass of [false, true]) {
    test(`hot-seat retains the summary through undo and explicit dismissal (pass device ${pass})`, async ({
      page
    }) => {
      const problems = watchHygiene(page);
      await configure(page, pass, pass);
      const driver = await openHotseatRoom(page, 'extra-turn-lost');
      await driver.nextTurn();
      await playKey(driver, 'discard:theater');
      await expect(page.locator(AGE)).toBeVisible();
      const frozen = await totals(page);
      await expect(page.locator('dialog[open] [data-age-continue]')).toHaveText(
        'Peek at the board'
      );
      const originalDialog = await page.locator('dialog[open]').elementHandle();
      await originalDialog!.evaluate((dialog) => {
        dialog.addEventListener('close', () =>
          dialog.setAttribute('data-unexpected-close', 'true')
        );
      });
      const starter = page.locator('dialog[open] [data-act="starter:1"]');
      await starter.focus();
      await page.keyboard.press('Enter');
      await driver.waitReady();
      // Choosing the other starter cannot close or replace this holder's
      // already-open dialog. Only their explicit continuation hands it on.
      expect(await originalDialog!.evaluate((dialog) => dialog.isConnected)).toBe(true);
      await expect(page.locator(AGE)).toBeVisible();
      await expect(page.locator('dialog[open]')).not.toHaveAttribute(
        'data-unexpected-close',
        'true'
      );
      expect(await totals(page)).toEqual(frozen);
      if (pass) {
        await expect(page.locator('[data-pass-device]')).toHaveCount(0);
        await expect(page.locator('dialog[open] [data-begin-age]')).toBeFocused();
        await continueAge(page);
        const handoff = page.locator('[data-pass-device]');
        await expect(handoff).toBeVisible();
        await expect(handoff.locator('li').filter({ hasText: 'discarded Theater' })).toHaveCount(1);
        await expect(handoff.locator('li').filter({ hasText: 'chooses' })).toHaveCount(1);
        await expect(page.locator(POPUP)).toHaveCount(0);
        await driver.passDevice();
      }
      await expect(page.locator(AGE)).toBeVisible();
      expect(await totals(page)).toEqual(frozen);
      if (pass) await expect(page.locator(POPUP)).toHaveCount(0);
      await continueAge(page);
      await expect(page.locator('[data-control="menu"]').filter({ visible: true })).toBeFocused();
      if (pass) {
        // Both holders explicitly closed their summaries. Returning the
        // device cannot reopen the chooser's already-read age.
        await discard(driver);
        await driver.passDevice();
        await expectNoAge(page);
      }
      for (let move = 0; move < (pass ? 2 : 1); move++) {
        await driver.visible('[data-control="undo"]').click();
        await driver.waitReady();
        if (await page.locator('[data-pass-device]').count()) {
          // Undo did not make an opponent move. Do not reopen an older run.
          await expect(page.locator('[data-pass-device] li')).toHaveCount(0);
          await driver.passDevice();
        }
        if (pass && move === 0) await expectNoAge(page);
      }
      await expect(page.locator(AGE)).toBeVisible();
      await expect(page.locator('[data-act="starter:0"]')).toBeVisible();
      await page.keyboard.press('Escape');
      await expectNoAge(page);
      await expect(page.locator('[data-control="menu"]').filter({ visible: true })).toBeFocused();
      await expect(page.locator('[data-tooltip-open]')).toHaveCount(0);
      await page.locator('[data-choice-host="chooseStarter"] [data-peek-bar] button').click();
      await expect(page.locator(AGE)).toBeVisible();
      await page.locator('dialog[open] [data-act="starter:0"]').click();
      await driver.waitReady();
      await expect(page.locator('dialog[open] [data-begin-age]')).toBeFocused();
      // Native backdrop dismissal is explicit and cannot act on the board.
      const ply = (await currentView(page)).ply;
      await page.mouse.click(2, 2);
      await expectNoAge(page);
      expect((await currentView(page)).ply).toBe(ply);
      // The chooser closes and takes the first card. With handoff enabled,
      // the next holder still has their own unread completed-age summary.
      await discard(driver);
      if (pass) {
        await driver.passDevice();
        await expect(page.locator(AGE)).toBeVisible();
        expect(await totals(page)).toEqual(frozen);
        await continueAge(page);
      } else await expectNoAge(page);
      await driver.visible('[data-control="menu"]').click();
      await driver.visible('[data-control="menu-home"]').click();
      await page.waitForURL('/');
      await openHotseatRoom(page, 'mausoleum');
      await expectNoAge(page);
      expect(problems).toEqual([]);
    });
  }
});

for (const locale of ['en', 'fa'] as const) {
  test.describe(`age/result footer layout ${locale}`, { tag: '@mobile' }, () => {
    test.use({ appLocale: locale });
    for (const theme of ['light', 'dark'] as const) {
      test(`${theme}: controls stay reachable while details scroll`, async ({ page }) => {
        const problems = watchHygiene(page);
        await configure(page);
        await page.addInitScript((theme) => localStorage.setItem('ajayeb:theme', theme), theme);
        for (const [width, height] of [
          [360, 740],
          [390, 844],
          [740, 360],
          [1280, 800]
        ]) {
          await page.setViewportSize({ width: width!, height: height! });
          for (const fixture of [
            'choose-starter-seat0',
            'online-age-start-seat0',
            'game-over-civilian-seat0'
          ]) {
            await page.goto(`/dev/fixture/${fixture}`);
            await expect(page.locator('html')).toHaveAttribute('data-ready', '1');
            await expect(page.locator('html')).toHaveAttribute('lang', locale);
            await footerFits(page);
            const body = page.locator('dialog[open] [data-sheet-body]');
            await body.focus();
            await page.keyboard.press('End');
            if (await body.evaluate((element) => element.scrollHeight > element.clientHeight)) {
              await expect
                .poll(() => body.evaluate((element) => element.scrollTop))
                .toBeGreaterThan(0);
            }
            await footerFits(page);
            await body.evaluate((element) => {
              element.scrollTop = 0;
            });
            await footerFits(page);
          }
        }
        expect(problems).toEqual([]);
      });
    }
  });
}
