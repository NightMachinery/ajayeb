// The animation runner: plays a frame's timeline (plan.ts) on the page,
// through the Presenter's hooks (session/presenter.svelte.ts):
//
//   capture  before the swap, reads the old DOM: the rectangle of the card
//            each take starts from and a copy of it (its slot is gone after
//            the swap), the face-down cards that will flip, where the pawn
//            stood, and which city bands already existed;
//   play     after the swap and a tick, reads the new DOM's rectangles once,
//            then starts every animation (FLIP: from the old rectangle to the
//            new one, transform and opacity only, with WAAPI) and resolves
//            when input may come back (the take's landing);
//   finish   ends everything at once, in its final state: a new frame (from
//            capture), the Presenter's hurry() and clear().
//
// The state is instant and authoritative; animations are visual only. A
// flying card is a clone in the FX layer (FxLayer.svelte, fixed, under
// sheets and tooltips), stripped of data-card, data-act and ids so no test
// or driver can click it; the element it lands on keeps data-fx-hidden
// (visibility hidden, its box intact) until it lands. A counter tween writes
// the old number into the counter's own text node, and writes the server's
// number back the moment it ends or is finished. Nothing else on the page
// is changed, so with every animation finished the DOM is the one the frame
// renders with Animation off.
//
// Animation off (scaleMs gives 0) plans nothing and reads nothing: the
// frame shows at once, exactly as it did before animations existed. So does
// a frame with more frames queued behind it.

import { cardBySlug } from '$lib/catalog';
import { formatNumber, formatSigned } from '$lib/i18n';
import { scaleMs, type AnimationSpeed } from '$lib/ui/motion.svelte';
import { fxLayer } from './layer';
import { timelineFor } from './speed';
import {
  EMPTY_TIMELINE,
  type CoinStep,
  type FlipStep,
  type PawnStep,
  type PlanFrame,
  type Step,
  type TakeStep,
  type Timeline
} from './plan';
import { recordFrame, setPlanned, type FrameRecord } from './records';
import { Runs } from './runs';

/** The attribute that keeps a destination hidden until its card lands. */
export const HIDDEN_ATTR = 'data-fx-hidden';
/** The landing pulse on a destination that was already on screen. */
export const PULSE_MS = 180;

const EASE_FLIGHT = 'cubic-bezier(0.45, 0, 0.2, 1)';
const EASE_OUT = 'cubic-bezier(0.2, 0.7, 0.3, 1)';

export interface AnimRunnerOptions {
  /** The effective animation speed, read once per frame. */
  speed: () => AnimationSpeed;
  /** Frames queued behind the one being shown: any means no animation. */
  backlog?: () => number;
  /** The FX layer; FxLayer.svelte registers it by default. */
  layer?: () => HTMLElement | null;
  /** The game screen the anchors are looked up in. */
  root?: () => ParentNode | null;
  now?: () => number;
}

interface Box {
  x: number;
  y: number;
  w: number;
  h: number;
}

/** A clone of an old element, and where it stood. */
interface Snap {
  box: Box;
  clone: HTMLElement;
}

interface Captured {
  version: number;
  speed: AnimationSpeed;
  timeline: Timeline;
  record: FrameRecord;
  takes: Map<TakeStep, Snap>;
  flips: Map<number, Snap>;
  pawns: { el: HTMLElement; box: Box }[];
  /** The city bands on screen before the swap: `${seat}:${color}`. */
  bands: Set<string>;
}

function boxOf(el: Element): Box {
  const r = el.getBoundingClientRect();
  return { x: r.left, y: r.top, w: r.width, h: r.height };
}

function shown(box: Box): boolean {
  return box.w > 0 && box.h > 0;
}

/** A copy of `el` that nothing can find, click or read aloud. */
export function inertClone(el: Element): HTMLElement {
  const clone = el.cloneNode(true) as HTMLElement;
  const strip = (node: Element) => {
    for (const name of [
      'data-card',
      'data-act',
      'data-slot',
      'id',
      'tabindex',
      'aria-describedby'
    ]) {
      node.removeAttribute(name);
    }
  };
  strip(clone);
  for (const node of clone.querySelectorAll('*')) strip(node);
  // ArtImage shows its art from its own load handler, which a copy does not
  // get: art not yet loaded when it was copied (a card just turned face up)
  // would stay transparent through the whole flip, so the copy shows it.
  for (const img of clone.querySelectorAll<HTMLImageElement>('img.art-image:not(.loaded)')) {
    const show = () => img.classList.add('loaded');
    img.loading = 'eager';
    if (img.complete && img.naturalWidth > 0) show();
    else img.addEventListener('load', show, { once: true });
  }
  clone.setAttribute('aria-hidden', 'true');
  clone.setAttribute('inert', '');
  clone.setAttribute('data-fx-clone', '');
  return clone;
}

/** Places a clone at the layer's origin, sized like its box; transforms move it. */
function place(clone: HTMLElement, box: Box): void {
  const s = clone.style;
  s.position = 'absolute';
  s.insetBlockStart = '0';
  s.insetInlineStart = '0';
  s.margin = '0';
  s.width = `${box.w}px`;
  s.height = `${box.h}px`;
  s.setProperty('--cw', `${box.w}px`);
  s.setProperty('--ch', `${box.h}px`);
  s.transform = `translate(${box.x}px, ${box.y}px)`;
  s.zIndex = '';
}

function cssEscape(value: string): string {
  return typeof CSS !== 'undefined' && CSS.escape ? CSS.escape(value) : value;
}

/** Keeps a point inside a box (a destination scrolled out of its panel). */
function clampInto(x: number, y: number, box: Box): { x: number; y: number } {
  return {
    x: Math.min(Math.max(x, box.x), box.x + box.w),
    y: Math.min(Math.max(y, box.y), box.y + box.h)
  };
}

export class AnimRunner {
  #speed: () => AnimationSpeed;
  #backlog: () => number;
  #layer: () => HTMLElement | null;
  #root: () => ParentNode | null;
  #now: () => number;
  #runs: Runs;

  constructor(opts: AnimRunnerOptions) {
    this.#speed = opts.speed;
    this.#backlog = opts.backlog ?? (() => 0);
    this.#layer = opts.layer ?? fxLayer;
    this.#root =
      opts.root ??
      (() => (typeof document === 'undefined' ? null : document.querySelector('.game-screen')));
    this.#now = opts.now ?? (() => performance.now());
    this.#runs = new Runs(this.#now);
  }

  /** Whether any animation is still running. */
  get busy(): boolean {
    return this.#runs.size > 0;
  }

  /** Ends every running animation now, in its final state, and lets input go. */
  finish(): void {
    this.#runs.finishAll();
  }

  /** The Presenter's capture hook: finishes what runs, then reads the old DOM. */
  capture(next: PlanFrame, prev: PlanFrame | null): Captured {
    this.finish();
    const record = recordFrame({
      version: next.version,
      swapAt: this.#now(),
      releasedAt: null,
      steps: 0
    });
    const speed = this.#speed();
    const root = this.#root();
    const layer = this.#layer();
    setPlanned(next.version, EMPTY_TIMELINE);
    if (speed === 'off' || !root || !layer || this.#backlog() > 0) return this.#none(record);
    const timeline = timelineFor(next, prev, speed);
    if (timeline.steps.length === 0) return this.#none(record);
    record.steps = timeline.steps.length;
    setPlanned(next.version, timeline);

    const takes = new Map<TakeStep, Snap>();
    const flips = new Map<number, Snap>();
    let pawns: Captured['pawns'] = [];
    const bands = new Set<string>();
    for (const step of timeline.steps) {
      if (step.kind === 'take') {
        const el = root.querySelector(`.card-structure [data-card="${cssEscape(step.card)}"]`);
        if (!el) continue;
        const box = boxOf(el);
        if (!shown(box)) continue;
        const clone = inertClone(el);
        // The flight shows the card itself, not the marks of the tap that sent it.
        clone.classList.remove('selected', 'pending');
        takes.set(step, { box, clone });
      } else if (step.kind === 'flip') {
        const el = root.querySelector(`.card-structure [data-slot="${step.slot}"]`);
        if (!el) continue;
        const box = boxOf(el);
        if (shown(box)) flips.set(step.slot, { box, clone: inertClone(el) });
      } else if (step.kind === 'pawn') {
        pawns = [...root.querySelectorAll<HTMLElement>('[data-fx-pawn]')]
          .map((el) => ({ el, box: boxOf(el) }))
          .filter((p) => shown(p.box));
      }
    }
    for (const el of root.querySelectorAll<HTMLElement>('.seat-panel [data-fx-band]')) {
      const seat = el.closest('.seat-panel')?.getAttribute('data-seat');
      bands.add(`${seat}:${el.dataset['fxBand']}`);
    }
    return { version: next.version, speed, timeline, record, takes, flips, pawns, bands };
  }

  /** A frame with nothing to animate: input goes as soon as it shows. */
  #none(record: FrameRecord): Captured {
    return {
      version: record.version,
      speed: 'off',
      timeline: EMPTY_TIMELINE,
      record,
      takes: new Map(),
      flips: new Map(),
      pawns: [],
      bands: new Set()
    };
  }

  /**
   * The Presenter's play hook: starts the frame's animations and resolves
   * when input may come back. Never throws for a missing element: a step
   * whose element is gone is skipped.
   */
  play(captured: unknown): Promise<void> {
    const cap = captured as Captured | null;
    if (!cap) return Promise.resolve();
    const root = this.#root();
    const layer = this.#layer();
    if (cap.timeline.steps.length === 0 || !root || !layer) {
      cap.record.releasedAt = this.#now();
      return Promise.resolve();
    }
    // Read every rectangle first, then write: one layout, not one per step.
    const starts: (() => void)[] = [];
    for (const step of cap.timeline.steps) {
      const start = this.#prepare(step, cap, root, layer);
      if (start) starts.push(start);
    }
    for (const start of starts) start();

    // The hold counts from the swap, so a slow render does not add to it.
    const block = cap.timeline.blockingMs - (this.#now() - cap.record.swapAt);
    if (block <= 0 || this.#runs.size === 0) {
      cap.record.releasedAt = this.#now();
      return Promise.resolve();
    }
    return this.#runs.hold(block, () => (cap.record.releasedAt = this.#now()));
  }

  /** Reads what a step needs and returns the function that starts it. */
  #prepare(step: Step, cap: Captured, root: ParentNode, layer: HTMLElement): (() => void) | null {
    switch (step.kind) {
      case 'take':
        return this.#prepareTake(step, cap, root, layer);
      case 'flip':
        return this.#prepareFlip(step, cap, root, layer);
      case 'coins':
        return this.#prepareCoins(step, cap, root, layer);
      case 'pawn':
        return this.#preparePawn(step, cap);
    }
  }

  #panel(root: ParentNode, seat: number): HTMLElement | null {
    return root.querySelector<HTMLElement>(`.seat-panel[data-seat="${seat}"]`);
  }

  /** The first of the selectors that is on screen inside `scope`. */
  #find(scope: ParentNode, selectors: string[]): { el: HTMLElement; box: Box } | null {
    for (const sel of selectors) {
      for (const el of scope.querySelectorAll<HTMLElement>(sel)) {
        const box = boxOf(el);
        if (shown(box)) return { el, box };
      }
    }
    return null;
  }

  #prepareTake(
    step: TakeStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const from = cap.takes.get(step);
    const panel = this.#panel(root, step.seat);
    if (!from || !panel) return null;
    const panelBox = boxOf(panel);
    let selectors: string[];
    let hide = false;
    if (step.dest === 'city') {
      const color = cardBySlug(step.card)?.color;
      selectors = color ? [`[data-fx-band="${color}"]`, '[data-fx-city]'] : ['[data-fx-city]'];
      // A colour's first card makes its band: it shows when the card lands.
      hide = !!color && !cap.bands.has(`${step.seat}:${color}`);
    } else if (step.dest === 'discard') {
      selectors = ['[data-tally="coins"]'];
    } else {
      const w = step.wonder ? cssEscape(step.wonder) : '';
      selectors = [`[data-fx-wonder="${w}"]`, '[data-fx-wonders]'];
    }
    const dest =
      this.#find(panel, selectors) ?? (shown(panelBox) ? { el: panel, box: panelBox } : null);
    if (!dest) return null;
    hide = hide && dest.el.matches('[data-fx-band]');

    const src = from.box;
    const centre = clampInto(dest.box.x + dest.box.w / 2, dest.box.y + dest.box.h / 2, panelBox);
    // Small enough to sink into a chip, never so small it vanishes in flight.
    const scale = Math.max(0.3, Math.min(dest.box.w / src.w, dest.box.h / src.h, 1));
    const endX = centre.x - (src.w * scale) / 2;
    const endY = centre.y - (src.h * scale) / 2;
    const liftX = src.x + (endX - src.x) * 0.12;
    const liftY = src.y + (endY - src.y) * 0.12 - Math.min(16, src.h * 0.12);
    const clone = from.clone;
    place(clone, src);
    clone.classList.add('fx-flight');
    // Into a chip it fades out; onto a big target (a phone strip) it stays.
    const fade = scale <= 0.5 ? 0 : 0.4;
    const keyframes: Keyframe[] = [
      { transform: `translate(${src.x}px, ${src.y}px) scale(1)`, opacity: 1, offset: 0 },
      {
        transform: `translate(${liftX}px, ${liftY}px) scale(1.06)`,
        opacity: 1,
        offset: 0.22
      },
      { transform: `translate(${endX}px, ${endY}px) scale(${scale})`, opacity: fade, offset: 1 }
    ];
    const destEl = dest.el;
    return () => {
      layer.append(clone);
      if (hide) destEl.setAttribute(HIDDEN_ATTR, '');
      const anim = clone.animate(keyframes, {
        duration: step.dur,
        delay: step.at,
        easing: EASE_FLIGHT,
        fill: 'backwards'
      });
      const run = this.#runs.track(
        cap.version,
        'take',
        [anim],
        () => {
          clone.remove();
          destEl.removeAttribute(HIDDEN_ATTR);
        },
        false
      );
      anim.finished.then(
        () => {
          if (run.done) return;
          this.#runs.end(run, false);
          // A landing nudge on what was already there (an existing band,
          // the coins, the wonder).
          if (!hide && destEl !== panel && destEl.isConnected) this.#pulse(cap, destEl);
        },
        () => this.#runs.end(run, true)
      );
    };
  }

  #pulse(cap: Captured, el: HTMLElement): void {
    const anim = el.animate(
      [{ transform: 'scale(1)' }, { transform: 'scale(1.12)' }, { transform: 'scale(1)' }],
      { duration: scaleMs(PULSE_MS, cap.speed), easing: EASE_OUT }
    );
    this.#runs.track(cap.version, 'take', [anim], () => {});
  }

  #prepareFlip(
    step: FlipStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const back = cap.flips.get(step.slot);
    const slot = root.querySelector<HTMLElement>(`.card-structure [data-slot="${step.slot}"]`);
    if (!back || !slot) return null;
    const box = boxOf(slot);
    if (!shown(box)) return null;
    const front = inertClone(slot);

    const holder = document.createElement('div');
    holder.className = 'fx-flip';
    holder.setAttribute('aria-hidden', 'true');
    place(holder, box);
    holder.style.perspective = `${Math.round(box.w * 6)}px`;
    const spinner = document.createElement('div');
    spinner.className = 'fx-flip-inner';
    for (const [face, turn] of [
      [back.clone, 'none'],
      [front, 'rotateY(180deg)']
    ] as const) {
      place(face, { x: 0, y: 0, w: box.w, h: box.h });
      face.style.transform = turn;
      face.style.backfaceVisibility = 'hidden';
      face.classList.add('fx-face');
      spinner.append(face);
    }
    holder.append(spinner);
    return () => {
      layer.append(holder);
      slot.setAttribute(HIDDEN_ATTR, '');
      const anim = spinner.animate(
        [{ transform: 'rotateY(0deg)' }, { transform: 'rotateY(180deg)' }],
        { duration: step.dur, delay: step.at, easing: EASE_OUT, fill: 'backwards' }
      );
      this.#runs.track(cap.version, 'flip', [anim], () => {
        holder.remove();
        slot.removeAttribute(HIDDEN_ATTR);
      });
    };
  }

  #prepareCoins(
    step: CoinStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const panel = this.#panel(root, step.seat);
    if (!panel) return null;
    const tally = this.#find(panel, ['[data-tally="coins"]']);
    if (!tally) return null;
    const count = tally.el.querySelector<HTMLElement>('[data-fx-count]');
    const text = count?.firstChild instanceof Text ? count.firstChild : null;
    const delta = step.to - step.from;

    const float = document.createElement('span');
    float.className = `fx-float ${delta < 0 ? 'fx-float-loss' : 'fx-float-gain'}`;
    float.setAttribute('aria-hidden', 'true');
    float.textContent = formatSigned(delta, { signDisplay: 'exceptZero' });
    const x = tally.box.x + tally.box.w / 2;
    const y = tally.box.y;
    const at = (dy: number) => `translate(${x}px, ${y + dy}px) translate(-50%, -60%)`;
    const final = text?.nodeValue ?? null;
    const tween = step.tween;
    const delay = step.at;

    return () => {
      layer.append(float);
      const anim = float.animate(
        [
          { transform: at(0), opacity: 0, offset: 0 },
          { transform: at(-6), opacity: 1, offset: 0.15 },
          { transform: at(-26), opacity: 1, offset: 0.65 },
          { transform: at(-34), opacity: 0, offset: 1 }
        ],
        { duration: step.dur, delay, easing: 'linear', fill: 'backwards' }
      );
      let frame = 0;
      const restore = () => {
        if (frame) cancelAnimationFrame(frame);
        frame = 0;
        if (text && final !== null) text.nodeValue = final;
      };
      // The counter: the old number until the float starts, then a tween
      // to the new one, then the server's own text back.
      if (text && final !== null && tween > 0) {
        text.nodeValue = formatNumber(step.from);
        const begin = this.#now() + delay;
        let shownValue = step.from;
        const tick = () => {
          const t = Math.min(1, Math.max(0, (this.#now() - begin) / tween));
          if (t >= 1) {
            restore();
            return;
          }
          const eased = 1 - (1 - t) * (1 - t);
          const value = Math.round(step.from + delta * eased);
          if (value !== shownValue) {
            shownValue = value;
            text.nodeValue = formatNumber(value);
          }
          frame = requestAnimationFrame(tick);
        };
        frame = requestAnimationFrame(tick);
      }
      this.#runs.track(cap.version, 'coins', [anim], () => {
        float.remove();
        restore();
      });
    };
  }

  #preparePawn(step: PawnStep, cap: Captured): (() => void) | null {
    const cells = Math.abs(step.to - step.from);
    if (cells === 0) return null;
    const moves = cap.pawns
      .filter((p) => p.el.isConnected)
      .map((p) => {
        const now = boxOf(p.el);
        return { el: p.el, dx: p.box.x + p.box.w / 2 - (now.x + now.w / 2), h: now.h };
      })
      .filter((m) => Math.abs(m.dx) > 0.5);
    if (moves.length === 0) return null;
    return () => {
      const anims = moves.map(({ el, dx, h }) => {
        const keyframes: Keyframe[] = [];
        const hop = -Math.max(2, h * 0.25);
        for (let k = 0; k < cells; k++) {
          keyframes.push({
            transform: `translate(${dx * (1 - k / cells)}px, 0px)`,
            offset: k / cells,
            easing: EASE_OUT
          });
          keyframes.push({
            transform: `translate(${dx * (1 - (k + 0.5) / cells)}px, ${hop}px)`,
            offset: (k + 0.5) / cells,
            easing: 'ease-in'
          });
        }
        keyframes.push({ transform: 'translate(0px, 0px)', offset: 1 });
        return el.animate(keyframes, {
          duration: step.dur,
          delay: step.at,
          fill: 'backwards'
        });
      });
      this.#runs.track(cap.version, 'pawn', anims, () => {});
    };
  }
}
