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
// hooks, frames are shown as they arrive, except for a bot's answer, which
// is paced so a player can follow it (UX design section 5, "AI thinking"):
//
//   - dwell: a frame that answers a bot's thinking frame (the frame on
//     screen says a bot is thinking, and the next one carries events) is
//     not shown until the thinking frame has been on screen at least
//     minThinkingMs;
//   - pre-roll: when that answer takes something the player can see (a card
//     from the structure, a wonder in the draft), `preview` holds the answer
//     for preRollMs before the swap, so the screen can highlight what the
//     bot is about to take while the old frame is still up.
//
// Both settings may be numbers or functions read at each use, so the app
// can scale them with the animation speed. `fastForwardAfter`: when more
// frames than this are queued (after a reconnect, say), the queue jumps to
// the newest one, and a frame with more frames behind it is never paced.
// hurry() cuts a running dwell or pre-roll short and jumps to the newest
// frame (an undo pressed while a bot answer is held); clear() forgets the
// room on screen.
//
// The UI enables input only when presenter.idle and session.canAct. idle
// stays false through a dwell and a pre-roll, so input stays locked.

import { tick } from 'svelte';
import { playFrameFx } from '$lib/fx';
import type { State, View } from '$lib/wire';
import type { Unsubscribe } from './transport';

export interface PresenterHooks {
  /** Before the swap: read the old DOM. The result is passed to play. */
  capture?(next: State, prev: State | null): unknown;
  /** After the swap and a tick: animate. The next frame waits for it. */
  play?(next: State, prev: State | null, captured: unknown): void | Promise<void>;
}

/** A duration in ms, or a function read each time it is needed. */
export type Millis = number | (() => number);

export interface PresenterOptions {
  hooks?: PresenterHooks;
  /** The shortest time a bot's thinking frame stays up before its answer. */
  minThinkingMs?: Millis;
  /** How long the bot's pick is previewed before its answer shows. */
  preRollMs?: Millis;
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

/** The event kinds of a move that takes something on the board or in the draft. */
const TAKE_KINDS: ReadonlySet<string> = new Set(['build', 'discard', 'wonder', 'draft']);

/**
 * The frame the sound cues of `next` are measured against: `prev`, or null
 * (no cues) when `next` carries a history. Only a join, a resync and the
 * frames after an undo carry one (docs/protocol.md, the state frame's
 * `history`): they replay the game rather than report something that just
 * happened, so a newer version arriving that way never plays a move's cues
 * again, as the client must not animate or toast such a frame either.
 */
export function cuePrev(prev: State | null, next: State): State | null {
  return next.history ? null : prev;
}

/**
 * Whether `next` is a bot's answer to the thinking frame `prev`: the same
 * room, a bot was thinking on screen, and something happened since.
 */
export function isBotAnswer(prev: State | null, next: State): boolean {
  return !!prev && prev.code === next.code && prev.thinking !== undefined && next.events.length > 0;
}

/**
 * Whether a bot's answer takes something the player can see being taken:
 * a card of the structure or a wonder of the draft, by the seat that was
 * thinking.
 */
export function isBotTake(prev: State | null, next: State): boolean {
  if (!prev || !isBotAnswer(prev, next)) return false;
  return next.events.some((e) => e.seat === prev.thinking && TAKE_KINDS.has(e.kind));
}

function millis(value: Millis): number {
  return Math.max(0, typeof value === 'function' ? value() : value);
}

export class Presenter {
  /** The frame on screen. */
  frame = $state.raw<State | null>(null);
  /** The view on screen. */
  view: View | null = $derived(this.frame?.view ?? null);
  /** No frame is waiting and nothing is playing. */
  idle = $state(true);
  /**
   * During a pre-roll: the bot's answer about to be shown, while `frame` is
   * still the one before it. The screen highlights what it takes.
   */
  preview = $state.raw<State | null>(null);

  hooks: PresenterHooks;
  minThinkingMs: Millis;
  preRollMs: Millis;
  fastForwardAfter: number;

  #queue: State[] = [];
  #running = false;
  #thinkingSince: number | null = null;
  #now: () => number;
  #afterSwap: () => Promise<void>;
  #sleep: (ms: number) => Promise<void>;
  #off: Unsubscribe | null = null;
  #destroyed = false;
  // Bumped by clear(): a pump that wakes up to another generation stops.
  #gen = 0;
  // Ends the current dwell or pre-roll early (hurry, clear, destroy).
  #wake: (() => void) | null = null;
  // Set by hurry(): the frames queued now are shown with no pacing.
  #hurried = false;

  constructor(source: FrameSource | null, opts: PresenterOptions = {}) {
    this.hooks = opts.hooks ?? {};
    this.minThinkingMs = opts.minThinkingMs ?? 0;
    this.preRollMs = opts.preRollMs ?? 0;
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

  /**
   * Cuts a running dwell or pre-roll short and shows the newest queued
   * frame at once, unpaced.
   */
  hurry(): void {
    if (this.idle) return;
    this.flush();
    this.#hurried = true;
    this.#wake?.();
  }

  /** Forgets the frame on screen and everything queued (the room was left). */
  clear(): void {
    this.#gen++;
    this.#queue = [];
    this.#thinkingSince = null;
    this.#hurried = false;
    this.#wake?.();
    this.frame = null;
    this.preview = null;
    this.idle = true;
  }

  destroy(): void {
    this.#destroyed = true;
    this.#queue = [];
    this.#wake?.();
    this.#off?.();
    this.#off = null;
  }

  // Waits ms, or less when woken; resolves at once for 0.
  async #wait(ms: number): Promise<void> {
    if (ms <= 0 || this.#hurried) return;
    const woken = new Promise<void>((resolve) => (this.#wake = resolve));
    await Promise.race([this.#sleep(ms), woken]);
    this.#wake = null;
  }

  async #pump(): Promise<void> {
    this.#running = true;
    const gen = this.#gen;
    const live = () => !this.#destroyed && gen === this.#gen;
    try {
      while (this.#queue.length > 0 && live()) {
        const prev = this.frame;
        const head = this.#queue[0];
        // A bot's answer waits until its thinking frame has had its dwell,
        // unless more frames are already queued behind it.
        if (head && this.#queue.length === 1 && isBotAnswer(prev, head)) {
          if (this.#thinkingSince !== null) {
            await this.#wait(this.#thinkingSince + millis(this.minThinkingMs) - this.#now());
          }
          // Then what it takes is previewed while the old frame stays up.
          const still = this.#queue[0];
          if (live() && still && this.#queue.length === 1 && isBotTake(prev, still)) {
            const ms = millis(this.preRollMs);
            if (ms > 0 && !this.#hurried) {
              this.preview = still;
              await this.#wait(ms);
              if (live()) this.preview = null;
            }
          }
        }
        if (!live()) break;
        if (this.#queue.length === 0) break;
        const next = this.#queue.shift();
        if (!next) break;
        if (this.#queue.length === 0) this.#hurried = false;
        this.preview = null;
        const captured = this.hooks.capture?.(next, prev);
        this.frame = next;
        // The bot's thinking starts when a thinking frame first shows, and
        // again after each answer that leaves it thinking (a second turn).
        if (next.thinking === undefined) {
          this.#thinkingSince = null;
        } else if (
          prev?.thinking === undefined ||
          prev.code !== next.code ||
          next.events.length > 0 ||
          this.#thinkingSince === null
        ) {
          this.#thinkingSince = this.#now();
        }
        await this.#afterSwap();
        playFrameFx(next, cuePrev(prev, next));
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
      // After a clear(), frames of the next room may already be waiting.
      if (!this.#destroyed) {
        this.idle = this.#queue.length === 0;
        if (this.#queue.length > 0) void this.#pump();
      }
    }
  }
}
