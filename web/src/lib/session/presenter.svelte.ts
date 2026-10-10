// The Presenter paces what the UI shows. The Session holds the latest
// authoritative frame; the Presenter holds the frame on screen, and moves
// it forward one queued frame at a time:
//
//   1. capture: a hook reads what it needs from the old DOM (for example
//      the card rectangles a FLIP animation starts from);
//   2. swap: the presented frame is replaced;
//   3. tick: Svelte updates the DOM;
//   4. play: a hook plays the animations of the new frame's events.
//
// There are no animations yet (M11 adds them through the hooks). With no
// hooks, frames are shown as they arrive. Two settings exist for later
// slices:
//   - minThinkingMs: a bot's move is not shown until the "thinking" frame
//     before it has been on screen at least this long (M6 uses ~700 ms);
//   - fastForwardAfter: when more frames than this are queued (after a
//     reconnect, say), the queue jumps to the newest one.
//
// The UI enables input only when presenter.idle and session.canAct.

import { tick } from 'svelte';
import type { State, View } from '$lib/wire';
import type { Unsubscribe } from './transport';

export interface PresenterHooks {
  /** Before the swap: read the old DOM. The result is passed to play. */
  capture?(next: State, prev: State | null): unknown;
  /** After the swap and a tick: animate. The next frame waits for it. */
  play?(next: State, prev: State | null, captured: unknown): void | Promise<void>;
}

export interface PresenterOptions {
  hooks?: PresenterHooks;
  minThinkingMs?: number;
  fastForwardAfter?: number;
  now?: () => number;
  /** Waits for the DOM update after a swap; Svelte's tick by default. */
  afterSwap?: () => Promise<void>;
  /** Waits ms; setTimeout by default. */
  sleep?: (ms: number) => Promise<void>;
}

/** A source of frames: the Session. */
export interface FrameSource {
  subscribe(fn: (frame: State) => void): Unsubscribe;
}

const defaultSleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

export class Presenter {
  /** The frame on screen. */
  frame = $state.raw<State | null>(null);
  /** The view on screen. */
  view: View | null = $derived(this.frame?.view ?? null);
  /** No frame is waiting and nothing is playing. */
  idle = $state(true);

  hooks: PresenterHooks;
  minThinkingMs: number;
  fastForwardAfter: number;

  #queue: State[] = [];
  #running = false;
  #thinkingSince: number | null = null;
  #now: () => number;
  #afterSwap: () => Promise<void>;
  #sleep: (ms: number) => Promise<void>;
  #off: Unsubscribe | null = null;
  #destroyed = false;

  constructor(source: FrameSource | null, opts: PresenterOptions = {}) {
    this.hooks = opts.hooks ?? {};
    this.minThinkingMs = opts.minThinkingMs ?? 0;
    this.fastForwardAfter = opts.fastForwardAfter ?? 2;
    this.#now = opts.now ?? (() => Date.now());
    this.#afterSwap = opts.afterSwap ?? tick;
    this.#sleep = opts.sleep ?? defaultSleep;
    if (source) this.#off = source.subscribe((frame) => this.enqueue(frame));
  }

  /** Queues a frame to present. */
  enqueue(frame: State): void {
    if (this.#destroyed) return;
    this.#queue.push(frame);
    if (this.#queue.length > this.fastForwardAfter) this.#queue.splice(0, this.#queue.length - 1);
    this.idle = false;
    if (!this.#running) void this.#pump();
  }

  /** Shows the newest queued frame now, skipping the rest. */
  flush(): void {
    if (this.#queue.length > 1) this.#queue.splice(0, this.#queue.length - 1);
  }

  destroy(): void {
    this.#destroyed = true;
    this.#queue = [];
    this.#off?.();
    this.#off = null;
  }

  async #pump(): Promise<void> {
    this.#running = true;
    try {
      while (this.#queue.length > 0 && !this.#destroyed) {
        const prev = this.frame;
        // Hold a bot's move until its thinking frame has had its dwell.
        if (this.#thinkingSince !== null && this.minThinkingMs > 0) {
          const wait = this.#thinkingSince + this.minThinkingMs - this.#now();
          if (wait > 0) await this.#sleep(wait);
        }
        const next = this.#queue.shift();
        if (!next || this.#destroyed) break;
        const captured = this.hooks.capture?.(next, prev);
        this.frame = next;
        if (next.thinking !== undefined) {
          if (prev?.thinking === undefined) this.#thinkingSince = this.#now();
        } else {
          this.#thinkingSince = null;
        }
        await this.#afterSwap();
        if (this.hooks.play) {
          try {
            await this.hooks.play(next, prev, captured);
          } catch (err) {
            // An animation that fails must never stall the game.
            console.error(err);
          }
        }
      }
    } finally {
      this.#running = false;
      if (!this.#destroyed) this.idle = this.#queue.length === 0;
      if (this.#queue.length > 0 && !this.#destroyed) void this.#pump();
    }
  }
}
