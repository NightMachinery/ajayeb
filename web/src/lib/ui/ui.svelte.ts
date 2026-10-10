// The game screen's own UI state: which slot is selected, which sheet is
// open and at which step, the wonder-first pick mode, which player holds
// the device in a hot-seat game, and the log of events seen so far. One
// GameUi lives per game screen. It decides nothing about the game: the
// screen reads every action from the view's choices.
//
// Writes happen in event handlers and in the Session's frame callback,
// never inside an $effect.

import type { GameEvent, State } from '$lib/wire';

export type SheetKind = 'card' | 'wonder' | 'city' | 'log' | 'settings';

/** The card sheet's steps: the card itself, then the wonders it could go under. */
export type CardStep = 'card' | 'wonders';

/** What a frame's ack answered, as far as the screen knows. */
export type Acked = 'act' | 'undo' | null;

export type Seat = 0 | 1;

/** Stop growing the log past this many events (oldest go first). */
export const LOG_LIMIT = 2000;

export class GameUi {
  /** The open sheet, if any. */
  sheet = $state<SheetKind | null>(null);
  /** The structure slot the card sheet shows. */
  selected = $state<number | null>(null);
  /** The card sheet's step. */
  step = $state<CardStep>('card');
  /** The wonder the wonder sheet shows. */
  wonder = $state<string | null>(null);
  /** Wonder-first mode: the wonder waiting for a card to go under it. */
  pickFor = $state<string | null>(null);
  /** The seat whose city the city sheet shows. */
  citySeat = $state<Seat>(0);
  /** Hot-seat: the seat that last took the device (null before the first frame). */
  holder = $state<Seat | null>(null);
  /** Every event seen in this room, in order (the view carries only the latest). */
  log = $state.raw<readonly GameEvent[]>([]);

  // Log lengths before each acked act, so an undo can take its events back.
  #marks: number[] = [];

  /** Opens the card sheet on a slot. */
  selectSlot(index: number): void {
    this.selected = index;
    this.step = 'card';
    this.sheet = 'card';
  }

  /** The card sheet's wonder step (card first). */
  showWonders(): void {
    if (this.sheet === 'card') this.step = 'wonders';
  }

  /** Back from the wonder step to the card. */
  backToCard(): void {
    this.step = 'card';
  }

  /** Opens the wonder sheet (wonder first). */
  openWonder(slug: string): void {
    this.wonder = slug;
    this.sheet = 'wonder';
  }

  /** Starts wonder-first mode: the next card tapped goes under this wonder. */
  startWonderPick(slug: string): void {
    this.pickFor = slug;
    this.sheet = null;
    this.selected = null;
    this.step = 'card';
  }

  cancelWonderPick(): void {
    this.pickFor = null;
  }

  openCity(seat: Seat): void {
    this.citySeat = seat;
    this.sheet = 'city';
  }

  openLog(): void {
    this.sheet = 'log';
  }

  openSettings(): void {
    this.sheet = 'settings';
  }

  /** Closes the open sheet; wonder-first mode stays on until a move or Cancel. */
  close(): void {
    this.sheet = null;
    this.selected = null;
    this.step = 'card';
  }

  /** A move was sent: close everything and leave wonder-first mode. */
  sent(): void {
    this.close();
    this.pickFor = null;
  }

  /** Whether a sheet of this kind is open. */
  isOpen(kind: SheetKind): boolean {
    return this.sheet === kind;
  }

  /** Hot-seat: the seat to move has taken the device. */
  ready(seat: number | undefined): void {
    if (seat === 0 || seat === 1) this.holder = seat;
  }

  /**
   * Takes in a state frame from the Session: grows the log, takes an undone
   * move's events back out of it, and, while the pass-device screen is off
   * or before the first frame, hands the device to the seat to move.
   */
  record(frame: State, acked: Acked, passDevice: boolean): void {
    const toMove = frame.view.toMove;
    if (this.holder === null || !passDevice) {
      if (toMove === 0 || toMove === 1) this.holder = toMove;
    }
    let log = this.log;
    if (acked === 'undo') {
      const mark = this.#marks.pop();
      if (mark !== undefined) log = log.slice(0, mark);
    } else if (acked === 'act') {
      this.#marks.push(log.length);
    }
    if (frame.events.length > 0) {
      log = [...log, ...frame.events];
      if (log.length > LOG_LIMIT) {
        const drop = log.length - LOG_LIMIT;
        log = log.slice(drop);
        this.#marks = this.#marks.map((m) => Math.max(0, m - drop));
      }
    }
    if (log !== this.log) this.log = log;
    // A selected slot that has emptied has nothing left to show.
    if (this.selected !== null) {
      const slot = frame.view.structure[this.selected];
      if (!slot || slot.state === 'empty') this.close();
    }
    // Wonder-first mode ends when the turn is no longer this wonder's to take.
    if (this.pickFor !== null) {
      const still = frame.view.choices.some(
        (c) => c.kind === 'wonder' && c.wonder === this.pickFor
      );
      if (!still) this.pickFor = null;
    }
  }

  /** Forgets the room: a new room starts with a clean screen. */
  reset(): void {
    this.close();
    this.pickFor = null;
    this.wonder = null;
    this.holder = null;
    this.log = [];
    this.#marks = [];
  }
}
