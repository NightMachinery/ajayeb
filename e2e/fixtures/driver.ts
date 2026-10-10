// The shared game driver for full playthroughs (hot-seat now; versus the bot
// and online later). It plays through the visible UI only: it reads the
// legal moves from window.__ajayeb (docs/testing.md, "The client test
// hook"), picks one with a policy, performs it by clicking the controls a
// player would click (a structure slot, a sheet button, a panel button, the
// draft screen, the pass-device screen), and waits for the room's version to
// move before it looks again. It never sends a frame itself and never
// decides what is legal: every key it clicks is a [data-act] the page shows
// for one of the view's choices.

import { expect, type Locator, type Page } from '@playwright/test';
import type { Choice, PendingKind, Phase, RoomInfo, View, YouInfo } from '../../web/src/lib/wire';

/** window.__ajayeb as the page defines it (web/src/lib/session/test-hook.ts). */
export interface AjayebHook {
  view(): View | null;
  choices(): Choice[];
  room():
    | (RoomInfo & { code: string; version: number; canUndo: boolean; thinking: number | null })
    | null;
  you(): YouInfo | null;
  conn(): string;
  ready(): boolean;
}

declare global {
  interface Window {
    __ajayeb?: AjayebHook;
  }
}

/** What the driver reads before each move. */
export interface Snapshot {
  version: number;
  ply: number;
  phase: Phase;
  age: number;
  toMove: number | null;
  pending: PendingKind | null;
  choices: Choice[];
  result: View['result'] | null;
  /** The hot-seat pass-device screen is up. */
  pass: boolean;
}

/** How a move is performed on screen. */
export type Flow =
  'draft' | 'pending' | 'construct' | 'discard' | 'wonderCardFirst' | 'wonderFirst';

export interface Move {
  flow: Flow;
  choice: Choice;
}

export interface DriverStats {
  /** Moves sent (each one a server action). */
  actions: number;
  /** Moves by flow. */
  flows: Record<Flow, number>;
  /** Pending choices answered, by kind. */
  pendingKinds: Partial<Record<PendingKind, number>>;
  /** Pass-device screens dismissed. */
  passScreens: number;
}

export type Policy = (snap: Snapshot, stats: DriverStats) => Move;

/**
 * The hot-seat policy: deterministic, and chosen to walk every flow at
 * least once rather than to play well.
 *
 * - The draft takes the first wonder on offer, on the draft screen.
 * - A pending choice (a token, a destroy, a revive, the starter of an age)
 *   takes its first option, through its panel.
 * - In play, a wonder is built whenever one is legal until both wonder
 *   flows have been used (card first, then wonder first), and after that on
 *   every fourth ply; every fifth ply discards; otherwise the first legal
 *   construct, else the first discard.
 */
export const hotseatPolicy: Policy = (snap, stats) => {
  const first = (kind: Choice['kind']) => snap.choices.find((c) => c.kind === kind);
  const fallback = snap.choices[0];
  if (!fallback) throw new Error(`no choices at ply ${snap.ply} (phase ${snap.phase})`);

  if (snap.phase === 'wonderDraft') return { flow: 'draft', choice: first('draft') ?? fallback };
  if (snap.pending) return { flow: 'pending', choice: fallback };

  const wonder = first('wonder');
  const { wonderCardFirst, wonderFirst } = stats.flows;
  if (wonder && (wonderCardFirst === 0 || wonderFirst === 0 || snap.ply % 4 === 0)) {
    const flow =
      wonderCardFirst === 0
        ? 'wonderCardFirst'
        : wonderFirst === 0
          ? 'wonderFirst'
          : snap.ply % 8 === 0
            ? 'wonderFirst'
            : 'wonderCardFirst';
    return { flow, choice: wonder };
  }
  const discard = first('discard');
  if (discard && snap.ply % 5 === 4) return { flow: 'discard', choice: discard };
  const construct = first('construct');
  if (construct) return { flow: 'construct', choice: construct };
  if (discard) return { flow: 'discard', choice: discard };
  throw new Error(`no play move at ply ${snap.ply}: ${snap.choices.map((c) => c.key).join(', ')}`);
};

const q = (value: string) => JSON.stringify(value);

export class GameDriver {
  readonly page: Page;
  readonly stats: DriverStats = {
    actions: 0,
    flows: {
      draft: 0,
      pending: 0,
      construct: 0,
      discard: 0,
      wonderCardFirst: 0,
      wonderFirst: 0
    },
    pendingKinds: {},
    passScreens: 0
  };

  constructor(page: Page) {
    this.page = page;
  }

  /** Waits until the page shows the latest state and takes input. */
  async waitReady(timeout = 20_000): Promise<void> {
    await this.page.waitForFunction(() => window.__ajayeb?.ready() === true, null, { timeout });
  }

  /** Waits until the room's version passes `version` and the page is ready again. */
  async waitAdvance(version: number, timeout = 20_000): Promise<void> {
    await this.page.waitForFunction(
      (v) => {
        const hook = window.__ajayeb;
        return !!hook && hook.ready() && (hook.room()?.version ?? -1) > v;
      },
      version,
      { timeout }
    );
  }

  async snapshot(): Promise<Snapshot> {
    return this.page.evaluate(() => {
      const hook = window.__ajayeb;
      const view = hook?.view();
      const room = hook?.room();
      if (!hook || !view || !room) throw new Error('no game on screen');
      return {
        version: room.version,
        ply: view.ply,
        phase: view.phase,
        age: view.age,
        toMove: view.toMove ?? null,
        pending: view.pending?.kind ?? null,
        choices: hook.choices(),
        result: view.result ?? null,
        pass: document.querySelector('[data-pass-device]') !== null
      };
    });
  }

  /** The one visible element of a selector (sheets keep hidden copies in the DOM). */
  visible(selector: string): Locator {
    return this.page.locator(selector).filter({ visible: true }).first();
  }

  /**
   * Closes any toast on screen with its dismiss button (never its Undo),
   * as a player would before tapping what it covers.
   */
  async dismissToasts(): Promise<void> {
    const closes = this.page.locator('[data-toast] > button:last-child').filter({ visible: true });
    for (let n = await closes.count(); n > 0; n--) {
      await closes
        .first()
        .click()
        .catch(() => {});
    }
  }

  /** "I'm ready" on the pass-device screen. */
  async passDevice(): Promise<void> {
    await this.visible('[data-pass-device] button').click();
    await expect(this.page.locator('[data-pass-device]')).toHaveCount(0);
    this.stats.passScreens += 1;
  }

  /** The draft screen: Take straight away where tiles show it, or open the tile's sheet first. */
  async draft(choice: Choice): Promise<void> {
    const take = this.page.locator(`[data-act=${q(choice.key)}]`).filter({ visible: true });
    if ((await take.count()) === 0) {
      await this.visible(`[data-draft] [data-pick=${q(choice.key)}]`).click();
    }
    await this.visible(`[data-act=${q(choice.key)}]`).click();
  }

  /** A pending choice's panel: select the option when the panel has tiles, then commit. */
  async pending(choice: Choice): Promise<void> {
    const host = '[data-choice-host]';
    await expect(this.page.locator(`${host}[data-mine="true"]`)).toHaveCount(1);
    const pick = this.page
      .locator(`${host} [data-pick=${q(choice.key)}]`)
      .filter({ visible: true });
    if ((await pick.count()) > 0) await pick.first().click();
    await this.visible(`${host} [data-act=${q(choice.key)}]`).click();
  }

  /** Taps a structure card and waits for its sheet. */
  async openCard(card: string): Promise<void> {
    await this.dismissToasts();
    await this.visible(`.card-structure [data-card=${q(card)}]`).click();
    await expect(this.page.locator(`dialog[open] [data-card-sheet=${q(card)}]`)).toBeVisible();
  }

  /** Build or discard from the card sheet. */
  async cardAction(choice: Choice): Promise<void> {
    if (!choice.card) throw new Error(`${choice.key} names no card`);
    await this.openCard(choice.card);
    await this.visible(`dialog[open] [data-act=${q(choice.key)}]`).click();
  }

  /** Card first: the card sheet, "Build a wonder", then the wonder's button. */
  async wonderCardFirst(choice: Choice): Promise<void> {
    if (!choice.card) throw new Error(`${choice.key} names no card`);
    await this.openCard(choice.card);
    await this.visible('dialog[open] [data-control="wonder-step"]').click();
    await expect(this.page.locator('dialog[open] [data-wonder-step]')).toBeVisible();
    await this.visible(`dialog[open] [data-act=${q(choice.key)}]`).click();
  }

  /**
   * Wonder first: the wonder in the mover's panel, "Build this wonder", a
   * marked card in the structure, then the confirmation in the card sheet.
   */
  async wonderFirst(choice: Choice, seat: number): Promise<void> {
    if (!choice.card || !choice.wonder) throw new Error(`${choice.key} is not a wonder move`);
    await this.dismissToasts();
    await this.visible(
      `.seat-panel[data-seat="${seat}"] button[data-wonder=${q(choice.wonder)}]`
    ).click();
    await expect(
      this.page.locator(`dialog[open] [data-wonder-sheet=${q(choice.wonder)}]`)
    ).toBeVisible();
    await this.visible('dialog[open] [data-control="wonder-build"]').click();
    await expect(this.page.locator(`[data-wonder-pick-bar=${q(choice.wonder)}]`)).toBeVisible();
    await this.visible(`.card-structure [data-card=${q(choice.card)}][data-marked]`).click();
    await expect(
      this.page.locator(`dialog[open] [data-wonder-pick=${q(choice.wonder)}]`)
    ).toBeVisible();
    await this.visible(`dialog[open] [data-act=${q(choice.key)}]`).click();
  }

  /** Performs one move by its flow. */
  async perform(move: Move, snap: Snapshot): Promise<void> {
    const { choice } = move;
    switch (move.flow) {
      case 'draft':
        await this.draft(choice);
        break;
      case 'pending':
        await this.pending(choice);
        if (snap.pending) {
          this.stats.pendingKinds[snap.pending] = (this.stats.pendingKinds[snap.pending] ?? 0) + 1;
        }
        break;
      case 'construct':
      case 'discard':
        await this.cardAction(choice);
        break;
      case 'wonderCardFirst':
        await this.wonderCardFirst(choice);
        break;
      case 'wonderFirst':
        if (snap.toMove === null) throw new Error('a wonder move with no seat to move');
        await this.wonderFirst(choice, snap.toMove);
        break;
    }
    this.stats.flows[move.flow] += 1;
    this.stats.actions += 1;
  }

  /**
   * Plays the game on screen to its end and returns the final snapshot.
   * Dismisses the pass-device screen whenever it is up.
   */
  async playToEnd(policy: Policy = hotseatPolicy, maxActions = 300): Promise<Snapshot> {
    for (;;) {
      await this.waitReady();
      const snap = await this.snapshot();
      if (snap.pass) {
        await this.passDevice();
        continue;
      }
      if (snap.phase === 'gameOver') return snap;
      if (this.stats.actions >= maxActions) {
        throw new Error(`the game did not end within ${maxActions} actions`);
      }
      await this.perform(policy(snap, this.stats), snap);
      await this.waitAdvance(snap.version);
    }
  }
}
