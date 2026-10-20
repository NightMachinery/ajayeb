// Helpers for the online specs: browser contexts that behave like separate
// devices, the frame capture the leak check reads, the Home and invite
// flows, and the server's room file.
//
// Each player and the spectator get their own browser context in the one
// browser: their own storage (so their own identity token and name), their
// own cookies and their own network switch (context.setOffline), which is
// what two phones or two laptops would have.

import { readFileSync } from 'node:fs';
import path from 'node:path';
import {
  expect,
  type Browser,
  type BrowserContext,
  type Page,
  type TestInfo
} from '@playwright/test';
import { installMotionOff } from './base';
import { hotseatPolicy, type Policy } from './driver';
import { dataDir } from './global-setup';
import type { CapturedFrame, ViewerStream } from './leak-check';

/** The e2e server's origin (playwright.config.ts). */
export const ORIGIN = 'http://127.0.0.1:38954';

/**
 * A new context with the project's device (viewport, touch, user agent), as
 * the test's own `context` fixture would have (`browser.newContext` does not
 * apply the project's options by itself), and the game's own Animation
 * setting forced off (installMotionOff; this context bypasses fixtures/
 * base.ts's `context` fixture override, which only covers the default
 * context a test gets without calling this function).
 */
export async function newDeviceContext(
  browser: Browser,
  testInfo: TestInfo,
  opts: { clipboard?: boolean } = {}
): Promise<BrowserContext> {
  const use = testInfo.project.use;
  const context = await browser.newContext({
    baseURL: use.baseURL ?? ORIGIN,
    viewport: use.viewport,
    userAgent: use.userAgent,
    deviceScaleFactor: use.deviceScaleFactor,
    isMobile: use.isMobile,
    hasTouch: use.hasTouch
  });
  await installMotionOff(context);
  if (opts.clipboard) {
    // The permission must be scoped to the origin the context actually
    // navigates to (use.baseURL, the same one passed to newContext above),
    // not the fixture's own ORIGIN constant: for every local run they are
    // the same value, but against a real site (M8.S02's production smoke,
    // playwright.config.ts's E2E_BASE_URL) only use.baseURL is right, and
    // granting the wrong origin fails the read silently until used.
    await context.grantPermissions(['clipboard-read', 'clipboard-write'], {
      origin: use.baseURL ?? ORIGIN
    });
  }
  return context;
}

export interface LoggedConnection {
  socket: number;
  openedAt: number;
  closedAt: number | null;
  helloNamed: boolean | null;
  states: number;
}

/**
 * Every WebSocket frame a page receives from the server, in order, with its
 * room version. It follows the page across reloads (each new socket counts
 * up `sockets`). Lifecycle metadata records observed close events; a missing
 * close event during navigation does not prove a connection is still open.
 * Attach it before the page's first navigation.
 */
export class FrameLog implements ViewerStream {
  readonly name: string;
  readonly frames: CapturedFrame[] = [];
  sockets = 0;
  readonly connections: LoggedConnection[] = [];
  readonly #started = Date.now();

  constructor(page: Page, name: string) {
    this.name = name;
    page.on('websocket', (ws) => {
      if (!new URL(ws.url()).pathname.endsWith('/ws')) return;
      this.sockets += 1;
      const socket = this.sockets;
      const connection: LoggedConnection = {
        socket,
        openedAt: Date.now() - this.#started,
        closedAt: null,
        helloNamed: null,
        states: 0
      };
      this.connections.push(connection);
      ws.on('close', () => {
        connection.closedAt = Date.now() - this.#started;
      });
      ws.on('framesent', ({ payload }) => {
        try {
          const text = typeof payload === 'string' ? payload : payload.toString('utf8');
          const message = JSON.parse(text) as { t?: string; name?: string };
          if (message.t === 'hello') connection.helloNamed = !!message.name;
        } catch {
          /* Only lifecycle metadata is needed; frame capture below is unchanged. */
        }
      });
      ws.on('framereceived', ({ payload }) => {
        const text = typeof payload === 'string' ? payload : payload.toString('utf8');
        let version: number | null = null;
        try {
          const msg = JSON.parse(text) as { t?: string; version?: unknown };
          if (msg.t === 'state' && typeof msg.version === 'number') version = msg.version;
        } catch {
          // Kept as it came: the leak check reports a frame that is not JSON.
        }
        if (version !== null) connection.states += 1;
        this.frames.push({ text, version, at: Date.now() - this.#started, socket });
      });
    });
  }

  /** The state frames received so far. */
  get states(): number {
    return this.frames.filter((f) => f.version !== null).length;
  }
}

export type Crest = 'Sun' | 'Moon';

/** Home, Play online, a name and a crest, Create: the lobby of a new room. Returns its code. */
export async function createOnlineGame(page: Page, name: string, crest: Crest): Promise<string> {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
  await page.getByRole('button', { name: /Play online/ }).click();
  const sheet = page.locator('dialog[open]').filter({ has: page.locator('[data-online-sheet]') });
  await expect(sheet.getByRole('heading', { name: 'Play online' })).toBeVisible();
  await sheet.locator('[data-name-input]').fill(name);
  await sheet.getByText(crest, { exact: true }).click();
  await expect(sheet.getByRole('radio', { name: crest, exact: true })).toBeChecked();
  await sheet.locator('[data-control="online-create"]').click();
  await page.waitForURL(/\/play\/[A-Z0-9]+$/);
  const lobby = page.locator('[data-lobby]');
  await expect(lobby).toBeVisible();
  const code = await lobby.getAttribute('data-lobby');
  if (!code) throw new Error('the lobby names no room');
  return code;
}

/**
 * Reads the invite from the lobby the way a player passes it on: Copy
 * link, then the clipboard. It must be the link the lobby shows, and end in
 * the code it shows in large type.
 */
export async function readInvite(page: Page): Promise<{ code: string; link: string }> {
  const lobby = page.locator('[data-lobby]');
  const code = (await lobby.locator('[data-room-code]').getAttribute('data-room-code')) ?? '';
  await expect(lobby.locator('[data-room-code]')).toHaveText(code);
  const shown = await lobby.locator('[data-invite-link]').inputValue();
  await lobby.locator('[data-control="copy-link"]').click();
  await expect(page.locator('[data-toast]').filter({ hasText: 'Link copied.' })).toBeVisible();
  const link = await page.evaluate(() => navigator.clipboard.readText());
  expect(link, 'the copied link is the one the lobby shows').toBe(shown);
  expect(new URL(link).pathname).toBe(`/play/${code}`);
  return { code, link };
}

/** Opens an invite link in a fresh context, answers the name prompt and waits for the room. */
export async function joinByLink(page: Page, link: string, name: string): Promise<void> {
  await page.goto(link);
  const prompt = page.locator('[data-name-prompt]');
  await expect(prompt).toBeVisible();
  await prompt.locator('[data-name-input]').fill(name);
  await expect(prompt.locator('[data-control="name-submit"]')).toHaveText(`Play as ${name}`);
  await prompt.locator('[data-control="name-submit"]').click();
  await expect(prompt).toHaveCount(0);
  await page.waitForFunction(() => window.__ajayeb?.ready() === true, null, { timeout: 20_000 });
}

/** The server's room file (server/internal/store), read once the game is over. */
export interface RoomFile {
  code: string;
  mode: string;
  seed: string;
  actions: string[];
  version: number;
  status: string;
  /** The engine's result: kind (the victory), winner, scores. */
  result?: { kind?: string; winner?: number } & Record<string, unknown>;
}

export function readRoomFile(code: string): RoomFile {
  return JSON.parse(readFileSync(path.join(dataDir, 'rooms', `${code}.json`), 'utf8')) as RoomFile;
}

/**
 * The online policy: the hot-seat policy (every flow at least once), except
 * that it drafts and builds the Great Library whenever it can, so that the
 * game's one asymmetric secret is part of the leak check whenever the deal
 * allows it.
 */
export const onlinePolicy: Policy = (snap, stats) => {
  if (snap.phase === 'wonderDraft') {
    const library = snap.choices.find((c) => c.kind === 'draft' && c.wonder === 'great-library');
    if (library) return { flow: 'draft', choice: library };
  } else if (snap.phase === 'play' && !snap.pending) {
    const library = snap.choices.find((c) => c.kind === 'wonder' && c.wonder === 'great-library');
    if (library) return { flow: 'wonderCardFirst', choice: library };
  }
  return hotseatPolicy(snap, stats);
};
