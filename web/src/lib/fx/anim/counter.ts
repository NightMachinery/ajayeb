// A coin counter's tween (runner.ts): the number a counter's own text node
// shows while its frame animates. A frame may move one counter more than
// once (a trade's coins, then a looting token, then another), so a counter
// holds segments, each a tween from one number to the next at its own
// time: before the first it shows the first's start, between two the
// earlier's end. When the last segment ends, or the counter is stopped (a
// new frame, hurry, clear), the text node gets the server's own text back,
// the text the frame rendered, so the DOM is the one Animation off shows.
//
// One requestAnimationFrame loop per counter, reading no layout.

import { formatNumber } from '$lib/i18n';

interface Segment {
  /** The start, in performance.now() time, and the length in ms. */
  begin: number;
  dur: number;
  from: number;
  to: number;
}

export class CounterTween {
  readonly text: Text;
  #final: string;
  #now: () => number;
  #segments: Segment[] = [];
  #frame = 0;
  #shown: number | null = null;
  #ended: (() => void) | null = null;
  done = false;

  /** `final` is the text the frame rendered; `ended` runs once the last segment ends. */
  constructor(text: Text, final: string, now: () => number, ended?: () => void) {
    this.text = text;
    this.#final = final;
    this.#now = now;
    this.#ended = ended ?? null;
  }

  /**
   * Shows `value` until the first segment is added, unless something is
   * shown already: the old number, while the frame's animations wait to
   * start (Runs.gate).
   */
  hold(value: number): void {
    if (this.done || this.#shown !== null) return;
    this.#show(value);
  }

  /** Adds a tween from `from` to `to`, starting `delay` ms from now. */
  add(delay: number, dur: number, from: number, to: number): void {
    if (this.done) return;
    this.#segments.push({ begin: this.#now() + delay, dur: Math.max(0, dur), from, to });
    this.#segments.sort((a, b) => a.begin - b.begin);
    this.#show(this.#valueAt(this.#now()));
    if (!this.#frame) this.#frame = requestAnimationFrame(this.#tick);
  }

  #valueAt(t: number): number {
    const first = this.#segments[0];
    if (!first || t < first.begin) return first?.from ?? 0;
    let value = first.from;
    for (const s of this.#segments) {
      if (t < s.begin) break;
      const k = s.dur > 0 ? Math.min(1, (t - s.begin) / s.dur) : 1;
      const eased = 1 - (1 - k) * (1 - k);
      value = Math.round(s.from + (s.to - s.from) * eased);
    }
    return value;
  }

  #show(value: number): void {
    if (value === this.#shown) return;
    this.#shown = value;
    this.text.nodeValue = formatNumber(value);
  }

  #tick = () => {
    this.#frame = 0;
    if (this.done) return;
    const t = this.#now();
    const end = Math.max(...this.#segments.map((s) => s.begin + s.dur));
    if (t >= end) {
      this.stop();
      this.#ended?.();
      return;
    }
    this.#show(this.#valueAt(t));
    this.#frame = requestAnimationFrame(this.#tick);
  };

  /** Ends the tween now and puts the server's text back. */
  stop(): void {
    if (this.#frame) cancelAnimationFrame(this.#frame);
    this.#frame = 0;
    if (this.done) return;
    this.done = true;
    this.text.nodeValue = this.#final;
  }
}
