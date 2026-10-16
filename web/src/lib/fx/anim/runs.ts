// The running animations of the runner (runner.ts), and the hold on input:
// DOM-free bookkeeping, so the interrupt rules can be unit-tested with
// stand-in animations.
//
//   track   adds a run: its WAAPI animations, and the cleanup that puts the
//           page back in the frame's own state (removes a clone, shows a
//           hidden destination, writes a counter's number back). Its
//           record starts when the animations do (their `ready`);
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
}

export interface Run {
  readonly anims: readonly Playing[];
  readonly record: AnimRecord;
  done: boolean;
  cleanup: () => void;
}

export class Runs {
  #now: () => number;
  #running = new Set<Run>();
  #release: (() => void) | null = null;
  #timer: ReturnType<typeof setTimeout> | null = null;

  constructor(now: () => number) {
    this.#now = now;
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
    // The motion starts at the first frame after the swap's task, when
    // the browser resolves the animations' start time.
    void Promise.all(anims.map((a) => a.ready)).then(
      () => {
        if (!run.done) run.record.startAt = this.#now();
      },
      () => {}
    );
    if (follow) {
      void Promise.all(anims.map((a) => a.finished)).then(
        () => this.end(run, false),
        () => this.end(run, true)
      );
    }
    return run;
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
    for (const run of [...this.#running]) this.end(run, true);
    this.letGo();
  }
}
