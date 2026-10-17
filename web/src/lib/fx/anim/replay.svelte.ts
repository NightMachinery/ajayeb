// "Replay last move" (UX design section 5, "Log"): whether the animation
// runner holds a move it can play again, and the way to ask for it. The
// runner (runner.ts) offers the last move it animated, with the copies of
// the old screen it read before that move's swap; the log sheet's control
// reads `available` and calls play(). A move that played nothing (Animation
// off, a resync, an undo, a frame with nothing to animate) withdraws the
// offer. A replay changes no state: it plays the move's animations again on
// the screen as it is now, through the same runner path as the move did.

class Replay {
  /** The version of the move on offer, or null. */
  version = $state<number | null>(null);
  #play: (() => boolean) | null = null;

  /** Whether there is a move to replay. */
  get available(): boolean {
    return this.version !== null;
  }

  /** The runner offers the move of `version`; `play` replays it. */
  offer(version: number, play: () => boolean): void {
    this.version = version;
    this.#play = play;
  }

  withdraw(): void {
    this.version = null;
    this.#play = null;
  }

  /** Replays the move on offer; false when there is none, or Animation is off. */
  play(): boolean {
    return this.#play?.() ?? false;
  }
}

/** The app's one replay offer. */
export const replay = new Replay();
