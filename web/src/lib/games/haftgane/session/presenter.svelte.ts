// Haftgane's Presenter paces what the table screen shows, as Duel's does
// ($lib/session/presenter.svelte.ts). The Session holds the latest frame;
// the Presenter holds the frame on screen and moves it forward one queued
// frame at a time:
//
//   1. capture: a hook reads what it needs from the old DOM;
//   2. swap: the presented frame is replaced, whole;
//   3. tick: Svelte updates the DOM;
//   4. play: a hook plays the new frame's events.
//
// Three kinds of frame are shown without playing anything:
//
//   - a frame with `history` (a join, a resync, an undo, or this
//     connection's first frame after its seat changed): it restates the
//     game for the viewer's place rather than reporting something that just
//     happened, and it replaces the old view whole, events and all;
//   - a frame whose `you.seat` differs from the one on screen, for the same
//     reason, should its history have been merged away in the queue;
//   - a cosmetic refresh (the same revision and no events, such as a
//     presence change), which replaces the frame on screen in place.
//
// When more frames than fastForwardAfter wait (after a reconnect, say),
// the queue jumps to the newest. The hooks come with the reveal and motion
// slices (M18.reveal-and-recap, M18.sound-and-motion); without them frames
// show as they arrive. The UI takes input only while `idle`.

import { tick } from 'svelte';
import type { Unsubscribe } from '$lib/session/transport';
import type { TableEvent, TableState } from '$lib/wire/platform.gen';

export interface TablePresenterHooks {
  /** Before the swap: read the old DOM. The result is passed to play. */
  capture?(next: TableState, prev: TableState | null): unknown;
  /** After the swap and a tick: animate. The next frame waits for it. */
  play?(next: TableState, prev: TableState | null, captured: unknown): void | Promise<void>;
  /** Ends every running animation at once (hurry, clear, destroy). */
  finish?(): void;
}

export interface TablePresenterOptions {
  hooks?: TablePresenterHooks;
  fastForwardAfter?: number;
  /** Waits for the DOM update after a swap; Svelte's tick by default. */
  afterSwap?: () => Promise<void>;
}

/** A source of table frames: the Session. */
export interface TableFrameSource {
  subscribe(fn: (frame: TableState) => void): Unsubscribe;
}

/** Whether `next` restates the game instead of reporting a change since `prev`. */
export function restates(prev: TableState | null, next: TableState): boolean {
  return (
    next.history !== undefined ||
    !prev ||
    prev.code !== next.code ||
    (prev.you.seat ?? null) !== (next.you.seat ?? null)
  );
}

/** Whether `next` only refreshes the position on screen. */
export function cosmetic(prev: TableState | null, next: TableState): boolean {
  return (
    !!prev && !restates(prev, next) && prev.revision === next.revision && next.events.length === 0
  );
}

export class TablePresenter {
  /** The frame on screen. */
  frame = $state.raw<TableState | null>(null);
  /** The events of the change on screen; empty after a restating frame. */
  shownEvents = $state.raw<readonly TableEvent[]>([]);
  /** No frame is waiting and nothing is playing. */
  idle = $state(true);

  hooks: TablePresenterHooks;
  fastForwardAfter: number;

  #queue: TableState[] = [];
  // Queued frames that restate the game because a frame skipped before them
  // did: shown, never played.
  #quiet = new WeakSet<TableState>();
  // A cosmetic refresh of a queued frame shows the refresh, with the events
  // and history of the frame it replaced (its effect source), as Duel's does.
  #effects = new WeakMap<TableState, TableState>();
  #running = false;
  #destroyed = false;
  #gen = 0;
  #afterSwap: () => Promise<void>;
  #off: Unsubscribe | null = null;

  constructor(source: TableFrameSource | null, opts: TablePresenterOptions = {}) {
    this.hooks = opts.hooks ?? {};
    this.fastForwardAfter = opts.fastForwardAfter ?? 2;
    this.#afterSwap = opts.afterSwap ?? tick;
    if (source) this.#off = source.subscribe((frame) => this.enqueue(frame));
  }

  /** The frames waiting behind the one being shown. */
  get queued(): number {
    return this.#queue.length;
  }

  /** Queues a frame to present. */
  enqueue(frame: TableState): void {
    if (this.#destroyed) return;
    const last = this.#queue.at(-1);
    // A cosmetic follow-up of a queued frame takes its place, keeping the
    // queued frame's events and history.
    if (last && cosmetic(this.#effects.get(last) ?? last, frame)) {
      this.#queue[this.#queue.length - 1] = frame;
      this.#effects.set(frame, this.#effects.get(last) ?? last);
      if (this.#quiet.has(last)) this.#quiet.add(frame);
      return;
    }
    this.#queue.push(frame);
    if (this.#queue.length > this.fastForwardAfter) this.#skipToNewest();
    this.idle = false;
    if (!this.#running) void this.#pump();
  }

  /** Ends what plays now and shows the newest queued frame at once. */
  hurry(): void {
    this.hooks.finish?.();
    this.#skipToNewest();
  }

  // Drops every queued frame but the newest, which then shows without
  // playing: its events alone would not tell what the skipped ones did.
  #skipToNewest(): void {
    if (this.#queue.length <= 1) return;
    this.#queue.splice(0, this.#queue.length - 1);
    this.#quiet.add(this.#queue[0]!);
  }

  /** Forgets the frame on screen and everything queued. */
  clear(): void {
    this.#gen++;
    this.#queue = [];
    this.hooks.finish?.();
    this.frame = null;
    this.shownEvents = [];
    this.idle = true;
  }

  destroy(): void {
    this.#destroyed = true;
    this.#queue = [];
    this.hooks.finish?.();
    this.#off?.();
    this.#off = null;
  }

  async #pump(): Promise<void> {
    this.#running = true;
    const gen = this.#gen;
    const live = () => !this.#destroyed && gen === this.#gen;
    try {
      while (this.#queue.length > 0 && live()) {
        const next = this.#queue.shift()!;
        const effects = this.#effects.get(next) ?? next;
        const prev = this.frame;
        if (cosmetic(prev, effects)) {
          this.frame = next;
          continue;
        }
        const restating = restates(prev, effects) || this.#quiet.has(next);
        const quiet = restating || effects.events.length === 0;
        const captured = quiet ? undefined : this.hooks.capture?.(effects, prev);
        this.frame = next;
        this.shownEvents = restating ? [] : effects.events;
        await this.#afterSwap();
        if (!quiet && this.hooks.play && live()) {
          try {
            await this.hooks.play(effects, prev, captured);
          } catch (err) {
            // An animation that fails must never stall the table.
            console.error(err);
          }
        }
      }
    } finally {
      this.#running = false;
      if (!this.#destroyed) {
        this.idle = this.#queue.length === 0;
        if (this.#queue.length > 0) void this.#pump();
      }
    }
  }
}
