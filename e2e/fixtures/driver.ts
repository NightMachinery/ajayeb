// The shared game driver for full playthroughs (hot-seat, versus the bot and
// online). It plays through the visible UI only: it reads the
// legal moves from window.__ajayeb (docs/testing.md, "The client test
// hook"), picks one with a policy, performs it by clicking the controls a
// player would click (a structure slot, a sheet button, a panel button, the
// draft screen, the pass-device screen), and waits for the room's version to
// move before it looks again. It never sends a frame itself and never
// decides what is legal: every key it clicks is a [data-act] the page shows
// for one of the view's choices.
//
// Against the bot it waits out the bot's turns the way a player does: until
// the page is ready again with a move of this client's to make (the bot's
// thinking frame, the Presenter's dwell and the bot's move have all been
// shown by then), or the game is over. It never sleeps a fixed time, except
// for the confirm step's double-tap guard (see commit).

import { expect, type Locator, type Page } from '@playwright/test';
import type {
  Choice,
  HistoryEntry,
  PendingKind,
  Phase,
  RoomInfo,
  View,
  YouInfo
} from '../../web/src/lib/wire';

/** window.__ajayeb as the page defines it (web/src/lib/session/test-hook.ts). */
export interface AjayebHook {
  view(): View | null;
  choices(): Choice[];
  room():
    | (RoomInfo & { code: string; version: number; canUndo: boolean; thinking: number | null })
    | null;
  you(): YouInfo | null;
  conn(): string;
  /** The game's history as the Session holds it: one entry per action. */
  history(): HistoryEntry[];
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
  /** The seats this connection acts for (you.seats). */
  seats: number[];
  /** The seat whose bot is searching, if any (the state frame's thinking). */
  thinking: number | null;
  /** The server would accept an undo now (the state frame's canUndo). */
  canUndo: boolean;
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
  /** Undos sent from the top bar. */
  undos: number;
  /** Moves sent with the confirm step (arm, then a second tap). */
  confirms: number;
}

/**
 * How long a move button ignores a second tap after the first one armed it
 * (ConfirmButton's double-tap guard is 350 ms), plus a margin. The armed
 * state lasts 3 s, so the second tap lands well inside it.
 */
const CONFIRM_GUARD_MS = 450;

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
    passScreens: 0,
    undos: 0,
    confirms: 0
  };
  /**
   * Moves take the confirm step (the settings' "Confirm moves", on in online
   * rooms by default): a move button's first tap arms it and a second one
   * sends. The pending choice panels' commit buttons take it too (one rule
   * for every act button); Undo never does.
   */
  confirm: boolean;

  constructor(page: Page, opts: { confirm?: boolean } = {}) {
    this.page = page;
    this.confirm = opts.confirm ?? false;
  }

  /**
   * Taps a move button. With the confirm step, the first tap must arm it
   * (data-armed, the label asks to tap again), and a second tap after the
   * double-tap guard sends the move.
   */
  async commit(button: Locator): Promise<void> {
    await button.click();
    if (!this.confirm) return;
    await expect(button).toHaveAttribute('data-armed', '');
    await this.page.waitForTimeout(CONFIRM_GUARD_MS);
    await button.click();
    this.stats.confirms += 1;
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

  /**
   * Waits until this client has a move to make (the page ready, a seat of
   * its own to move, and you.canAct), or the game is over. In a bot room
   * that is after the bot's thinking frame, its dwell and its move; in a
   * hot-seat room it is at once.
   */
  async waitTurn(timeout = 60_000): Promise<void> {
    await this.page.waitForFunction(
      () => {
        const hook = window.__ajayeb;
        if (!hook || !hook.ready()) return false;
        const view = hook.view();
        const you = hook.you();
        if (!view || !you) return false;
        if (view.phase === 'gameOver') return true;
        return you.canAct && view.toMove !== undefined && you.seats.includes(view.toMove);
      },
      null,
      { timeout, polling: 50 }
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
        pass: document.querySelector('[data-pass-device]') !== null,
        seats: hook.you()?.seats ?? [],
        thinking: room.thinking,
        canUndo: room.canUndo
      };
    });
  }

  /** The one visible element of a selector (sheets keep hidden copies in the DOM). */
  visible(selector: string): Locator {
    return this.page.locator(selector).filter({ visible: true }).first();
  }

  /**
   * Closes any toast on screen with its dismiss button,
   * and the bar that says who starts a new age (its close button, never
   * Summary), as a player would before tapping what they cover. On a phone
   * both float over the player's own strip, wonders included.
   */
  async dismissToasts(): Promise<void> {
    const closes = this.page
      .locator('[data-toast] > button:last-child, [data-peek-bar] button[aria-label]')
      .filter({ visible: true });
    // Toasts stack, and closing one can uncover another (an online game's
    // presence toast under an age's summary), so look again after each.
    for (let tries = 0; tries < 10 && (await closes.count()) > 0; tries++) {
      await closes
        .first()
        .click({ timeout: 2_000 })
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
    await this.commit(this.visible(`[data-act=${q(choice.key)}]`));
  }

  /** A pending choice's panel: select the option when the panel has tiles, then commit. */
  async pending(choice: Choice): Promise<void> {
    const host = '[data-choice-host]';
    await expect(this.page.locator(`${host}[data-mine="true"]`)).toHaveCount(1);
    const pick = this.page
      .locator(`${host} [data-pick=${q(choice.key)}]`)
      .filter({ visible: true });
    if ((await pick.count()) > 0) await pick.first().click();
    await this.commit(this.visible(`${host} [data-act=${q(choice.key)}]`));
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
    await this.commit(this.visible(`dialog[open] [data-act=${q(choice.key)}]`));
  }

  /** Card first: the card sheet, "Build a wonder", then the wonder's button. */
  async wonderCardFirst(choice: Choice): Promise<void> {
    if (!choice.card) throw new Error(`${choice.key} names no card`);
    await this.openCard(choice.card);
    await this.visible('dialog[open] [data-control="wonder-step"]').click();
    await expect(this.page.locator('dialog[open] [data-wonder-step]')).toBeVisible();
    await this.commit(this.visible(`dialog[open] [data-act=${q(choice.key)}]`));
  }

  /**
   * Wonder first: the wonder in the mover's panel (on a phone, through the
   * strip's wonder button and the wonder list sheet it opens), "Build this
   * wonder", a marked card in the structure, then the confirmation in the
   * card sheet.
   */
  async wonderFirst(choice: Choice, seat: number): Promise<void> {
    if (!choice.card || !choice.wonder) throw new Error(`${choice.key} is not a wonder move`);
    await this.dismissToasts();
    const panel = `.seat-panel[data-seat="${seat}"]`;
    const listButton = this.visible(`${panel} [data-control="wonders"]`);
    if ((await listButton.count()) > 0) {
      await listButton.click();
      await expect(this.page.locator(`dialog[open] [data-wonder-list="${seat}"]`)).toBeVisible();
      await this.visible(`dialog[open] button[data-wonder=${q(choice.wonder)}]`).click();
    } else {
      await this.visible(`${panel} button[data-wonder=${q(choice.wonder)}]`).click();
    }
    await expect(
      this.page.locator(`dialog[open] [data-wonder-sheet=${q(choice.wonder)}]`)
    ).toBeVisible();
    await this.visible('dialog[open] [data-control="wonder-build"]').click();
    await expect(this.page.locator(`[data-wonder-pick-bar=${q(choice.wonder)}]`)).toBeVisible();
    await this.visible(`.card-structure [data-card=${q(choice.card)}][data-marked]`).click();
    await expect(
      this.page.locator(`dialog[open] [data-wonder-pick=${q(choice.wonder)}]`)
    ).toBeVisible();
    await this.commit(this.visible(`dialog[open] [data-act=${q(choice.key)}]`));
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
   * Waits for this client's next move (see waitTurn), dismissing the
   * pass-device screen whenever it is up, and returns the snapshot there:
   * a move to make, or the game over.
   */
  async nextTurn(): Promise<Snapshot> {
    for (;;) {
      await this.waitTurn();
      const snap = await this.snapshot();
      if (!snap.pass) return snap;
      await this.passDevice();
    }
  }

  /** Performs the policy's move in `snap` and waits for the room to move on. */
  async play(snap: Snapshot, policy: Policy = hotseatPolicy): Promise<Move> {
    const move = policy(snap, this.stats);
    await this.perform(move, snap);
    await this.waitAdvance(snap.version);
    return move;
  }

  /**
   * Plays the game on screen to its end and returns the final snapshot.
   * Dismisses the pass-device screen whenever it is up, and waits through
   * the bot's turns in a bot room.
   */
  async playToEnd(policy: Policy = hotseatPolicy, maxActions = 300): Promise<Snapshot> {
    for (;;) {
      const snap = await this.nextTurn();
      if (snap.phase === 'gameOver') return snap;
      if (this.stats.actions >= maxActions) {
        throw new Error(`the game did not end within ${maxActions} actions`);
      }
      await this.play(snap, policy);
    }
  }

  /**
   * Undo from the top bar (after closing any toast that could cover it),
   * then waits for the room to move on. Returns the snapshot after it.
   */
  async undo(): Promise<Snapshot> {
    const before = await this.snapshot();
    await this.dismissToasts();
    const button = this.visible('[data-control="undo"]');
    await expect(button).toBeEnabled();
    await button.click();
    await this.waitAdvance(before.version);
    this.stats.undos += 1;
    return this.snapshot();
  }

  /**
   * Starts counting how often the bot's thinking indicator appears: the
   * turn bar's thinking state (`[data-turn="aiThinking"]`) and a seat
   * panel's thinking dots (`.seat-panel[data-thinking]`), each counted on
   * the change from absent to shown. It runs in the page from its next
   * load on, so call it before page.goto; read it with thinkingSeen().
   */
  async watchThinking(): Promise<void> {
    await this.page.addInitScript(() => {
      const seen = { turnBar: 0, panel: 0 };
      (window as unknown as { __e2eThinking: typeof seen }).__e2eThinking = seen;
      let turnBar = false;
      let panel = false;
      const look = () => {
        const nowTurnBar = document.querySelector('[data-turn="aiThinking"]') !== null;
        const nowPanel = document.querySelector('.seat-panel[data-thinking]') !== null;
        if (nowTurnBar && !turnBar) seen.turnBar += 1;
        if (nowPanel && !panel) seen.panel += 1;
        turnBar = nowTurnBar;
        panel = nowPanel;
      };
      new MutationObserver(look).observe(document, {
        subtree: true,
        childList: true,
        attributes: true,
        attributeFilter: ['data-turn', 'data-thinking']
      });
    });
  }

  /** How many times the thinking indicator has appeared (see watchThinking). */
  async thinkingSeen(): Promise<{ turnBar: number; panel: number }> {
    return this.page.evaluate(
      () =>
        (window as unknown as { __e2eThinking?: { turnBar: number; panel: number } })
          .__e2eThinking ?? { turnBar: 0, panel: 0 }
    );
  }
}
