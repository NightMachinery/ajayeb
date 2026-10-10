// Reconnect delays: exponential backoff with jitter, capped.
//
// The n-th retry (n from 0) waits a random time in
// [ceiling / 2, ceiling], where ceiling = min(max, base * 2^n). Half of the
// delay is fixed, so a retry never fires immediately after a failure, and
// half is random, so many clients that lost the same server do not all
// come back at the same instant ("equal jitter").

export interface BackoffOptions {
  /** The ceiling of the first retry, in ms. */
  baseMs: number;
  /** The largest ceiling, in ms. */
  maxMs: number;
}

export const DEFAULT_BACKOFF: BackoffOptions = { baseMs: 500, maxMs: 15_000 };

/** The ceiling of retry n: base * 2^n, capped at max. */
export function backoffCeiling(attempt: number, opts: BackoffOptions = DEFAULT_BACKOFF): number {
  const n = Math.max(0, Math.floor(attempt));
  // 2^n overflows to Infinity long after it passes any sane cap, so cap n too.
  return Math.min(opts.maxMs, opts.baseMs * 2 ** Math.min(n, 30));
}

/** The delay before retry n, given a random number in [0, 1). */
export function backoffDelay(
  attempt: number,
  random: () => number = Math.random,
  opts: BackoffOptions = DEFAULT_BACKOFF
): number {
  const ceiling = backoffCeiling(attempt, opts);
  const r = Math.min(Math.max(random(), 0), 1);
  return Math.round(ceiling / 2 + (ceiling / 2) * r);
}
