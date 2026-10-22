// The generic invite is an active seat join only after today's name gate.
import { readFileSync } from 'node:fs';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { createOnlineGame, newDeviceContext, readRoomFile } from '../fixtures/online';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';

const manifest = JSON.parse(
  readFileSync(
    new URL('../../web/.svelte-kit/output/client/.vite/manifest.json', import.meta.url),
    'utf8'
  )
) as Record<string, { file: string }>;
const haftganeChunk = '/' + manifest['src/lib/games/haftgane/entry.ts']!.file;

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

test('cold invite sends no hello or join and claims no seat before name submission @mobile', async ({
  browser
}, info) => {
  const creator = await newDeviceContext(browser, info),
    visitor = await newDeviceContext(browser, info);
  try {
    const owner = await creator.newPage();
    const ownerFrames = sentFrames(owner);
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
    expect(hellos.map((m) => m.v)).toEqual([2, 1]);
    expect(hellos.every((m) => m.name === 'Visitor')).toBe(true);
    expect(hellos[0]?.token).toBe(hellos[1]?.token);
    expect(hellos[0]?.clientId).toBe(hellos[1]?.clientId);
    expect(log.frames.filter((m) => m.t === 'join').map((m) => m.code)).toEqual([code, code]);
    expect(log.frames.some((m) => m.t === 'leave')).toBe(false);
    await expect.poll(() => log.closed).toContain(1);
    expect(await page.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([1]);
    expect(readRoomFile(code).status).toBe('playing');
    expect(requested).not.toContain(haftganeChunk);
    expect(problems).toEqual([]);
  } finally {
    await visitor.close();
    await creator.close();
  }
});

test('generic discovery, duplicate tabs and reconnect preserve the same Duel seat @mobile', async ({
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
    expect(log.frames.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([2, 1]);
    await second.evaluate(() => {
      window.dispatchEvent(new Event('offline'));
      window.dispatchEvent(new Event('online'));
    });
    await duplicate.waitReady();
    await expect
      .poll(() => log.frames.filter((m) => m.t === 'hello').map((m) => m.v))
      .toEqual([2, 1, 1]);
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

test('authoritative Haftgane snapshot overrides a stale Duel hint without loading Duel gameplay @mobile', async ({
  page
}) => {
  const sent: { t: string; v?: number }[] = [];
  const scripts: Promise<string>[] = [];
  page.on('response', (r) => {
    if (r.request().resourceType() === 'script') scripts.push(r.text());
  });
  await page.addInitScript(() => {
    localStorage.setItem('ajayeb:name', 'Player');
    localStorage.setItem(
      'ajayeb:rooms',
      JSON.stringify([{ code: 'REGAAA', mode: 'online', lastSeen: 1, game: 'duel' }])
    );
  });
  await page.routeWebSocket('**/ws', (socket) =>
    socket.onMessage((raw) => {
      const m = JSON.parse(String(raw)) as { t: string; v?: number };
      sent.push(m);
      if (m.t === 'hello')
        socket.send(
          JSON.stringify({ t: 'welcome', v: m.v, server: { version: 'test', rules: 1 } })
        );
      if (m.t === 'join' || m.t === 'resync')
        socket.send(
          JSON.stringify({
            t: 'table-state',
            game: 'haftgane',
            code: 'REGAAA',
            revision: 1,
            seats: [],
            humans: [],
            you: { userId: 'player' },
            view: {}
          })
        );
    })
  );
  await page.goto('/play/REGAAA');
  await expect.poll(() => sent.map((m) => m.t)).toEqual(['hello', 'join', 'resync']);
  await expect(page.locator('.card-structure')).toHaveCount(0);
  expect(sent.filter((m) => m.t === 'hello').map((m) => m.v)).toEqual([2]);
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem('ajayeb:rooms')!)[0].game)).toBe(
    'haftgane'
  );
  const downloaded = (await Promise.all(scripts)).join('\n');
  expect(downloaded).not.toContain('class SessionError');
  expect(await page.evaluate(() => !!window.__ajayeb)).toBe(false);
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
    ).toEqual([
      { v: 2, name: 'Visitor' },
      { v: 1, name: 'Visitor' }
    ]);
    expect(await page.evaluate(() => localStorage.getItem('ajayeb:name'))).toBeNull();
    expect(await page.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([1]);
  } finally {
    await visitor.close();
    await creator.close();
  }
});
