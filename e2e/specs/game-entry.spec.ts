// The generic invite is an active seat join only after today's name gate.
import { readFileSync } from 'node:fs';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { createOnlineGame, newDeviceContext, readRoomFile } from '../fixtures/online';
import { GameDriver } from '../fixtures/driver';
import { openHotseatRoom, playKey } from '../fixtures/scenario';
import { watchHygiene } from '../fixtures/hygiene';

const manifest = JSON.parse(
  readFileSync(
    new URL('../../web/.svelte-kit/output/client/.vite/manifest.json', import.meta.url),
    'utf8'
  )
) as Record<string, { file: string }>;
const haftganeChunk = '/' + manifest['src/lib/games/haftgane/entry.ts']!.file;

// A spectator's first frame from the real table actor
// (server/internal/tableroom/testdata/client-frames.json), for code.
function spectatorFrame(code: string): Record<string, unknown> {
  const recording = JSON.parse(
    readFileSync(
      new URL('../../server/internal/tableroom/testdata/client-frames.json', import.meta.url),
      'utf8'
    )
  ) as { code: string; clients: Record<string, { log: { recv?: Record<string, unknown> }[] }> };
  const frame = JSON.stringify(recording.clients['watcher']!.log[0]!.recv);
  return JSON.parse(frame.replaceAll(recording.code, code)) as Record<string, unknown>;
}

function sentFrames(page: Page) {
  const frames: {
    socket: number;
    t: string;
    v?: number;
    name?: string;
    token?: string;
    clientId?: string;
    code?: string;
    claim?: string;
  }[] = [];
  const closed: number[] = [];
  let sockets = 0;
  page.on('websocket', (socket) => {
    const index = ++sockets;
    socket.on('framesent', ({ payload }) =>
      frames.push({ socket: index, ...JSON.parse(String(payload)) })
    );
    socket.on('close', () => closed.push(index));
  });
  return { frames, closed };
}

function seatPresence(page: Page) {
  const states: boolean[] = [];
  page.on('websocket', (socket) =>
    socket.on('framereceived', ({ payload }) => {
      const message = JSON.parse(String(payload)) as {
        t: string;
        room?: { seats: { connected: boolean; kind: string }[] };
      };
      if (message.t !== 'state' || message.room?.seats[1]?.kind !== 'human') return;
      const connected = message.room.seats[1].connected;
      if (states.at(-1) !== connected) states.push(connected);
    })
  );
  return {
    states,
    sinceClaim: () => states.slice(states.indexOf(true)),
    reset: () => {
      states.length = 0;
    }
  };
}

test('cold invite sends no hello or join and claims no seat before name submission @mobile', async ({
  browser
}, info) => {
  const creator = await newDeviceContext(browser, info),
    visitor = await newDeviceContext(browser, info);
  try {
    const owner = await creator.newPage();
    const ownerFrames = sentFrames(owner);
    const presence = seatPresence(owner);
    await owner.addInitScript(() => {
      const seen = new WeakMap<Element, string>();
      const toasts: string[] = [];
      Object.defineProperty(window, '__entryToasts', { value: toasts });
      new MutationObserver(() => {
        for (const toast of document.querySelectorAll('[data-toast]')) {
          const text = toast.textContent ?? '';
          if (seen.get(toast) === text) continue;
          seen.set(toast, text);
          toasts.push(text);
        }
      }).observe(document, { subtree: true, childList: true, characterData: true });
    });
    const code = await createOnlineGame(owner, 'Creator', 'Sun');
    expect(ownerFrames.frames.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([1]);
    const page = await visitor.newPage();
    const requested: string[] = [];
    page.on('request', (request) => requested.push(new URL(request.url()).pathname));
    const log = sentFrames(page);
    const problems = watchHygiene(page);
    await page.goto(`/play/${code}`);
    await expect(page.locator('[data-name-prompt]')).toBeVisible();
    await page.waitForLoadState('networkidle');
    expect(log.frames).toEqual([]);
    const before = readRoomFile(code);
    expect(before.status).toBe('lobby');
    expect(
      (before as unknown as { seats: { kind: string }[] }).seats.filter((s) => s.kind === 'open')
    ).toHaveLength(1);
    await expect(owner.locator('[data-lobby]')).toBeVisible();
    await page.locator('[data-name-input]').fill('Visitor');
    await page.locator('[data-control="name-submit"]').click();
    await new GameDriver(page).waitReady();
    const hellos = log.frames.filter((m) => m.t === 'hello');
    expect(hellos.map((m) => m.v)).toEqual([1]);
    expect(hellos.every((m) => m.name === 'Visitor')).toBe(true);
    expect(log.frames.filter((m) => m.t === 'join').map((m) => m.code)).toEqual([code]);
    expect(log.frames.some((m) => m.t === 'leave')).toBe(false);
    expect(log.closed).toEqual([]);
    expect(presence.sinceClaim()).toEqual([true]);
    expect(
      await owner.evaluate(() =>
        Reflect.get(window, '__entryToasts').filter((text: string) =>
          text.includes('Visitor left.')
        )
      )
    ).toEqual([]);
    expect(await owner.locator('[data-toast]').allTextContents()).not.toContain(
      'Visitor left. The game waits for them.'
    );
    expect(await page.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([1]);
    expect(readRoomFile(code).status).toBe('playing');
    expect(requested).not.toContain(haftganeChunk);
    presence.reset();
    await page.reload();
    await new GameDriver(page).waitReady();
    await expect.poll(() => presence.states.at(-1)).toBe(true);
    // A real reload closes the old document once, as on 2912538. After
    // the returning player attaches there must be no discovery detach.
    expect(presence.sinceClaim()).toEqual([true]);
    expect(presence.states.filter((value) => !value).length).toBeLessThanOrEqual(1);
    expect(
      await owner.evaluate(
        () =>
          Reflect.get(window, '__entryToasts').filter((text: string) =>
            text.includes('Visitor left.')
          ).length
      )
    ).toBeLessThanOrEqual(1);
    expect(log.frames.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([1, 1]);
    expect(problems).toEqual([]);
  } finally {
    await visitor.close();
    await creator.close();
  }
});

test('ordinary Duel entry, duplicate tabs and reconnect preserve the same Duel seat @mobile', async ({
  browser
}, info) => {
  const creator = await newDeviceContext(browser, info),
    visitor = await newDeviceContext(browser, info);
  try {
    const owner = await creator.newPage();
    const code = await createOnlineGame(owner, 'Creator', 'Sun');
    const page = await visitor.newPage();
    await page.goto(`/play/${code}`);
    await page.locator('[data-name-input]').fill('Visitor');
    await page.locator('[data-control="name-submit"]').click();
    const driver = new GameDriver(page);
    await driver.waitReady();
    const second = await visitor.newPage(),
      log = sentFrames(second);
    await second.goto(`/play/${code}`);
    const duplicate = new GameDriver(second);
    await duplicate.waitReady();
    expect(await second.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([1]);
    expect(log.frames.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([1]);
    await second.evaluate(() => {
      window.dispatchEvent(new Event('offline'));
      window.dispatchEvent(new Event('online'));
    });
    await duplicate.waitReady();
    await expect
      .poll(() => log.frames.filter((m) => m.t === 'hello').map((m) => m.v))
      .toEqual([1, 1]);
    await page.close();
    expect(await second.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([1]);
    const before = await duplicate.snapshot();
    await second.reload();
    await duplicate.waitReady();
    expect((await duplicate.snapshot()).version).toBe(before.version);
    expect(await second.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([1]);
  } finally {
    await visitor.close();
    await creator.close();
  }
});

test('a table with no saved tag refuses v1 before admission, then discovers Haftgane @mobile', async ({
  page
}) => {
  const sent: { t: string; v?: number }[] = [];
  let admitted = 0;
  await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Player'));
  await page.routeWebSocket('**/ws', (socket) => {
    let version = 0;
    socket.onMessage((raw) => {
      const m = JSON.parse(String(raw)) as { t: string; v?: number };
      sent.push(m);
      if (m.t === 'hello') {
        version = m.v!;
        socket.send(
          JSON.stringify({ t: 'welcome', v: version, server: { version: 'test', rules: 1 } })
        );
      }
      if (m.t === 'join' && version === 1) {
        // The server's v1 admission gate refuses before attaching a seat.
        expect(admitted).toBe(0);
        socket.send(JSON.stringify({ t: 'error', code: 'protocol' }));
      } else if (m.t === 'join' || m.t === 'resync') {
        if (m.t === 'join') admitted++;
        socket.send(JSON.stringify(spectatorFrame('REGAAA')));
      }
    });
  });
  await page.goto('/play/REGAAA');
  await expect
    .poll(() => sent.map((m) => m.t))
    .toEqual(['hello', 'join', 'hello', 'join', 'resync']);
  expect(sent.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([1, 2]);
  expect(admitted).toBe(1);
  await expect(page.locator('.card-structure')).toHaveCount(0);
  await expect(page.locator('[data-entry-fatal], [data-room-error]')).toHaveCount(0);
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem('ajayeb:rooms')!)[0].game)).toBe(
    'haftgane'
  );
  expect(await page.evaluate(() => window.__ajayeb?.room())).toBeNull();
});

test('name-gated handoff keeps the submitted name when name storage is blocked @mobile', async ({
  browser
}, info) => {
  const creator = await newDeviceContext(browser, info),
    visitor = await newDeviceContext(browser, info);
  try {
    const owner = await creator.newPage();
    const code = await createOnlineGame(owner, 'Creator', 'Sun');
    await visitor.addInitScript(() => {
      const write = Storage.prototype.setItem;
      Storage.prototype.setItem = function (key, value) {
        if (key === 'ajayeb:name') throw new DOMException('Blocked', 'SecurityError');
        write.call(this, key, value);
      };
    });
    const page = await visitor.newPage(),
      log = sentFrames(page);
    await page.goto(`/play/${code}`);
    await expect(page.locator('[data-name-prompt]')).toBeVisible();
    expect(log.frames).toEqual([]);
    await page.locator('[data-name-input]').fill('Visitor');
    await page.locator('[data-control="name-submit"]').click();
    await new GameDriver(page).waitReady();
    expect(
      log.frames.filter((m) => m.t === 'hello').map((m) => ({ v: m.v, name: m.name }))
    ).toEqual([{ v: 1, name: 'Visitor' }]);
    expect(await page.evaluate(() => localStorage.getItem('ajayeb:name'))).toBeNull();
    expect(await page.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([1]);
  } finally {
    await visitor.close();
    await creator.close();
  }
});

test('a stale Haftgane tag keeps discovery attached until the first v1 Duel state @mobile', async ({
  browser
}, info) => {
  const creator = await newDeviceContext(browser, info),
    visitor = await newDeviceContext(browser, info);
  try {
    const owner = await creator.newPage(),
      presence = seatPresence(owner);
    const code = await createOnlineGame(owner, 'Creator', 'Sun');
    await visitor.addInitScript((code) => {
      localStorage.setItem('ajayeb:name', 'Visitor');
      localStorage.setItem(
        'ajayeb:rooms',
        JSON.stringify([{ code, mode: 'online', lastSeen: 1, game: 'haftgane' }])
      );
    }, code);
    const page = await visitor.newPage(),
      log = sentFrames(page);
    const events: string[] = [];
    page.on('websocket', (socket) => {
      let version = 0;
      socket.on('framesent', ({ payload }) => {
        const m = JSON.parse(String(payload));
        if (m.t === 'hello') version = m.v;
      });
      socket.on('framereceived', ({ payload }) => {
        if (JSON.parse(String(payload)).t === 'state') events.push(`state:${version}`);
      });
      socket.on('close', () => events.push(`close:${version}`));
    });
    await page.goto(`/play/${code}`);
    await new GameDriver(page).waitReady();
    await expect.poll(() => log.closed).toContain(1);
    const hellos = log.frames.filter((m) => m.t === 'hello');
    expect(hellos.map((m) => m.v)).toEqual([2, 1]);
    expect(hellos[0]?.token).toBe(hellos[1]?.token);
    expect(hellos[0]?.clientId).toBe(hellos[1]?.clientId);
    expect(events.indexOf('close:2')).toBeGreaterThan(events.indexOf('state:1'));
    expect(log.frames.some((m) => m.t === 'leave')).toBe(false);
    expect(presence.sinceClaim()).toEqual([true]);
    await expect(owner.locator('[data-toast]').filter({ hasText: 'Visitor left.' })).toHaveCount(0);
  } finally {
    await visitor.close();
    await creator.close();
  }
});

test('a discovery bad-hello uses the fatal reload notice @mobile', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('ajayeb:name', 'Player');
    localStorage.setItem(
      'ajayeb:rooms',
      JSON.stringify([{ code: 'REGAAA', mode: 'online', lastSeen: 1, game: 'haftgane' }])
    );
  });
  await page.routeWebSocket('**/ws', (socket) =>
    socket.onMessage(() => socket.send(JSON.stringify({ t: 'error', code: 'bad-hello' })))
  );
  await page.goto('/play/REGAAA');
  await expect(page.locator('[data-entry-fatal="bad-hello"] [data-notice]')).toContainText(
    /reload/i
  );
  await expect(page.locator('[data-conn="offline"]')).toBeVisible();
  await expect(page.locator('[data-room-error]')).toHaveCount(0);
});

test('rematch reuses Room and its v1 socket @mobile', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Player'));
  const log = sentFrames(page);
  const driver = await openHotseatRoom(page, 'military-supremacy');
  await playKey(driver, 'construct:walls');
  await expect(page.locator('dialog[open] [data-rematch]')).toBeVisible();
  const hook = await page.evaluateHandle(() => window.__ajayeb);
  // Observe Room's actual GameScreen host: a remount destroys this node.
  const screen = await page.locator('.game-screen').elementHandle();
  const previous = page.url();
  await page.locator('dialog[open] [data-rematch]').click();
  await page.waitForURL((url) => url.href !== previous);
  await driver.waitReady();
  expect(await page.evaluate((old) => old === window.__ajayeb, hook)).toBe(true);
  expect(screen).not.toBeNull();
  expect(await screen!.evaluate((node) => node.isConnected)).toBe(true);
  expect(log.frames.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([1]);
  expect(log.frames.some((m) => m.t === 'leave')).toBe(false);
});

test('a device fragment change reuses Room and its v1 socket @mobile', async ({ page }) => {
  const log = sentFrames(page);
  const code = await createOnlineGame(page, 'Creator', 'Sun');
  const lobby = await page.locator('[data-lobby]').elementHandle();
  await page.locator('[data-lobby] [data-control="device-link"][data-seat="0"]').click();
  const link = await page.locator('dialog[open] input[data-device-link]').inputValue();
  await page.keyboard.press('Escape');
  await page.evaluate((link) => {
    location.hash = new URL(link).hash;
  }, link);
  await expect
    .poll(() => log.frames.filter((m) => m.t === 'join' && m.code === code).length)
    .toBe(2);
  await new GameDriver(page).waitReady();
  expect(log.frames.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([1]);
  expect(log.frames.some((m) => m.t === 'leave')).toBe(false);
  expect(await lobby!.evaluate((node) => node.isConnected)).toBe(true);
});

test('deterministic build versions still detect a deployed update @mobile', async ({ page }) => {
  await openHotseatRoom(page, null);
  expect(await page.evaluate(() => window.__ajayeb!.checkForUpdate())).toBe(false);
  await page.route('**/_app/version.json', (route) =>
    route.fulfill({ json: { version: 'next-build-fixture' } })
  );
  expect(await page.evaluate(() => window.__ajayeb!.checkForUpdate())).toBe(true);
});
