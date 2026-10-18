// The running animations of the runner (runner.ts), and the hold on input:
// DOM-free bookkeeping, so the interrupt rules can be unit-tested with
// stand-in animations.
//
//   track   adds a run: its WAAPI animations, and the cleanup that puts the
//           page back in the frame's own state (removes a clone, shows a
//           hidden destination, writes a counter's number back). Its
//           record starts when the animations do (their `ready`, the first
//           frame they move in); a run with no WAAPI animation of its own (a
//           counter's tween, the deal's wait for a structure) at the first
//           animation frame, which is the first it can change the screen in;
//   gate    while a frame's animations are being set up: the runs tracked
//           until the gate opens wait, paused in their first pose (their
//           clones placed, their destinations hidden), until the page has
//           painted the new frame once. Painting a new frame (its new DOM,
//           the clones' layers) is the costliest frame of a move, and
//           motion that starts in it stutters on its first step; started a
//           frame later, it moves on layers that are already drawn;
//   end     ends one run once: finishes its animations, runs its cleanup,
//           and stamps its record (endAt, and whether it was cut short);
//   hold    the play hook's promise: it resolves after `ms`, or at once on
//           finishAll();
//   finishAll  ends every run, cut short, and lets input go. A new frame,
//           hurry() and clear() call it, so the screen never shows two
//           frames at once.

import type { StepKind } from './plan';
import { recordAnim, type AnimRecord } from './records';

/** What the runs need of a WAAPI Animation. */
export interface Playing {
  readonly playState: string;
  readonly ready: Promise<unknown>;
  readonly finished: Promise<unknown>;
  finish(): void;
  cancel(): void;
  pause(): void;
  play(): void;
}

export interface Run {
  readonly anims: readonly Playing[];
  readonly record: AnimRecord;
  done: boolean;
  cleanup: () => void;
}

export class Runs {
  #now: () => number;
  #frame: (run: () => void) => void;
  #running = new Set<Run>();
  #release: (() => void) | null = null;
  #timer: ReturnType<typeof setTimeout> | null = null;
  /** While a gate is shut: what starts each held run when it opens. */
  #held: (() => void)[] | null = null;

  /** `frame` runs a callback at the next animation frame (requestAnimationFrame by default). */
  constructor(now: () => number, frame?: (run: () => void) => void) {
    this.#now = now;
    this.#frame =
      frame ??
      ((run) => {
        if (typeof requestAnimationFrame === 'function') requestAnimationFrame(() => run());
        else setTimeout(run, 0);
      });
  }

  /** How many runs are still playing. */
  get size(): number {
    return this.#running.size;
  }

  /** Adds a run. `follow` ends it when all its animations have finished. */
  track(
    version: number,
    kind: StepKind,
    anims: readonly Playing[],
    cleanup: () => void,
    follow = true
  ): Run {
    const run: Run = {
      anims,
      cleanup,
      done: false,
      record: recordAnim({ version, kind, startAt: this.#now(), endAt: null, interrupted: false })
    };
    this.#running.add(run);
    // The motion starts at the first frame after the animations play, when
    // the browser resolves their start time. With no animation to wait
    // for, that frame is the next one: nothing a run does in the task that
    // tracks it (a counter writing back its old number) is motion.
    const stamp = () => {
      if (!run.done) run.record.startAt = this.#now();
    };
    const go = (paused: boolean) => {
      if (run.done) return;
      if (anims.length === 0) {
        this.#frame(stamp);
        return;
      }
      if (paused) for (const a of anims) a.play();
      void Promise.all(anims.map((a) => a.ready)).then(stamp, () => {});
    };
    if (this.#held) {
      for (const a of anims) a.pause();
      this.#held.push(() => go(true));
    } else {
      go(false);
    }
    if (follow) {
      void Promise.all(anims.map((a) => a.finished)).then(
        () => this.end(run, false),
        () => this.end(run, true)
      );
    }
    return run;
  }

  /**
   * Shuts the gate: runs tracked from now wait, paused, until the returned
   * function runs, which plays them at the second animation frame from
   * then, after the browser has painted the page as it is now. A
   * finishAll() before that opens it at once, ending them as usual.
   */
  gate(): () => void {
    const held: (() => void)[] = [];
    this.#held = held;
    return () => {
      if (this.#held === held) this.#held = null;
      this.#frame(() =>
        this.#frame(() => {
          for (const go of held.splice(0)) go();
        })
      );
    };
  }

  /** Ends a run, once: its animations finish and its cleanup runs. */
  end(run: Run, interrupted: boolean): void {
    if (run.done) return;
    run.done = true;
    this.#running.delete(run);
    for (const anim of run.anims) {
      try {
        if (anim.playState !== 'finished') anim.finish();
      } catch {
        anim.cancel();
      }
    }
    run.cleanup();
    run.record.endAt = this.#now();
    run.record.interrupted = interrupted;
  }

  /** Resolves after `ms`, or as soon as finishAll() runs. */
  hold(ms: number, released: () => void): Promise<void> {
    this.letGo();
    return new Promise<void>((resolve) => {
      this.#release = () => {
        released();
        resolve();
      };
      this.#timer = setTimeout(() => this.letGo(), ms);
    });
  }

  /** Lets input go now, if a hold is waiting. */
  letGo(): void {
    if (this.#timer !== null) clearTimeout(this.#timer);
    this.#timer = null;
    const release = this.#release;
    this.#release = null;
    release?.();
  }

  /** Ends every run, cut short, and lets input go. */
  finishAll(): void {
    this.#held = null;
    for (const run of [...this.#running]) this.end(run, true);
    this.letGo();
  }
}
