// The animation runner: plays a frame's timeline (plan.ts) on the page,
// through the Presenter's hooks (session/presenter.svelte.ts):
//
//   capture  before the swap, reads the old DOM: the rectangle and a copy of
//            whatever a step starts from and the swap removes (the taken
//            card's slot, a token on the board or in a choice, a draft tile,
//            a card of the discard pile, the face-down cards that will flip,
//            the looting tokens the pawn will take, the old structure of an
//            age deal), where the pawn stood, and which city bands existed;
//   play     after the swap and a tick, reads the new DOM's rectangles once,
//            then starts every animation (FLIP: from the old rectangle to the
//            new one, transform and opacity only, with WAAPI) and resolves
//            when input may come back (the lead's landing);
//   finish   ends everything at once, in its final state: a new frame (from
//            capture), the Presenter's hurry() and clear().
//
// The state is instant and authoritative; animations are visual only. A
// flying card is a clone in the FX layer (FxLayer.svelte, fixed, under
// sheets and tooltips; its top-layer part over a dialog), stripped of
// data-card, data-act and ids so no test or driver can click it; the
// element it lands on keeps data-fx-hidden (visibility hidden, its box
// intact, so it cannot be clicked either) until it lands. A counter tween
// writes the old number into the counter's own text node, and writes the
// server's number back the moment it ends or is finished (counter.ts).
// Nothing else on the page is changed, so with every animation finished the
// DOM is the one the frame renders with Animation off.
//
// Animation off (scaleMs gives 0) plans nothing and reads nothing: the
// frame shows at once, exactly as it did before animations existed. So does
// a frame with more frames queued behind it.
//
// Replay (replay.svelte.ts): the last frame that animated is kept, with
// copies of what its capture read, and replay() plays it again on the
// screen as it is now, without touching any state.

import { mount, unmount } from 'svelte';
import { cardBySlug } from '$lib/catalog';
import { formatSigned } from '$lib/i18n';
import { scaleMs, type AnimationSpeed } from '$lib/ui/motion.svelte';
import CardFace from '$lib/components/card/CardFace.svelte';
import { planCues, type PlannedCue } from '../cues';
import { playRecordedCues } from '../index';
import { cancelHaptic } from '../haptics';
import { DealCarry } from './backlog';
import { particleOffset, victoryBanner } from './banner';
import { CounterTween } from './counter';
import {
  boxAround,
  boxOf,
  centreOf,
  clampInto,
  cssEscape,
  drawn,
  findShown,
  inertClone,
  makeInert,
  place,
  settled,
  shown,
  type Box
} from './dom';
import { fxLayer, fxTop } from './layer';
import {
  EMPTY_TIMELINE,
  planTimeline,
  type CoinStep,
  type DealStep,
  type DestroyStep,
  type DraftStep,
  type DrawStep,
  type FlipStep,
  type GlowStep,
  type LootStep,
  type PawnStep,
  type PlanFrame,
  type RebuildStep,
  type Step,
  type StepKind,
  type TakeStep,
  type Timeline,
  type TokenStep,
  type VictoryStep
} from './plan';
import { recordFrame, setPlanned, type FrameRecord } from './records';
import { replay } from './replay.svelte';
import { Runs } from './runs';
import { retimeCues, scaleTimeline } from './speed';

export { inertClone } from './dom';

/** The attribute that keeps a destination hidden until its card lands. */
export const HIDDEN_ATTR = 'data-fx-hidden';
/** The landing pulse on a destination that was already on screen. */
export const PULSE_MS = 180;

const EASE_FLIGHT = 'cubic-bezier(0.45, 0, 0.2, 1)';
const EASE_OUT = 'cubic-bezier(0.2, 0.7, 0.3, 1)';
/** A card mounted for a flight that has no element to copy (a destroyed one). */
const MOUNT_W = 56;
const CARD_RATIO = 148 / 96;
/** A Great Library back's size, in px. */
const BACK_PX = 36;
const SVG_NS = 'http://www.w3.org/2000/svg';

export interface AnimRunnerOptions {
  /** The effective animation speed, read once per frame. */
  speed: () => AnimationSpeed;
  /** Frames queued behind the one being shown: any means no animation. */
  backlog?: () => number;
  /** The FX layer; FxLayer.svelte registers it by default. */
  layer?: () => HTMLElement | null;
  /** The FX layer's top-layer part, for drawing over a dialog. */
  top?: () => HTMLElement | null;
  /** The game screen the anchors are looked up in. */
  root?: () => ParentNode | null;
  now?: () => number;
}

/** A copy of an old element (or only its rectangle), and where it stood. */
interface Snap {
  box: Box;
  clone: HTMLElement | null;
}

interface Captured {
  version: number;
  speed: AnimationSpeed;
  timeline: Timeline;
  record: FrameRecord;
  /** By step index: what a lead step starts from (a card, a token, a tile). */
  sources: Map<number, Snap>;
  /** By slot: the face-down card that will flip. */
  flips: Map<number, Snap>;
  pawns: { el: HTMLElement; box: Box }[];
  /** By step index: the looting tokens (one per track on screen). */
  loots: Map<number, Snap[]>;
  /** The structure an age deal replaces. */
  structure: Snap | null;
  /** The city bands on screen before the swap: `${seat}:${color}`. */
  bands: Set<string>;
  /** The counters this frame tweens, by text node. */
  counters: Map<Text, CounterTween>;
}

/** The copies a capture made, which a replay copies again. */
type Copies = Pick<Captured, 'sources' | 'flips' | 'pawns' | 'loots' | 'structure' | 'bands'>;

/** What a replay needs of a frame that animated: its plan and its copies. */
interface Kept {
  version: number;
  plan: Timeline;
  cues: PlannedCue[];
  copies: Copies;
}

interface Dest {
  el: HTMLElement;
  box: Box;
}

interface FlightOpts {
  kind: StepKind;
  clone: HTMLElement;
  from: Box;
  dest: Dest;
  at: number;
  dur: number;
  /** Keep the landing point inside this box (a panel scrolled away). */
  clamp?: Box;
  /** Hide `dest` until the flight lands. */
  hide?: boolean;
  /** Nudge `dest` when the flight lands. */
  pulse?: boolean;
  /** The end opacity; by default the clone fades into a small target. */
  fade?: number;
  /** When in the flight (0 to 1) the fade begins; by default after the lift. */
  fadeFrom?: number;
  /** Appear at the start (a token drawn from the box), not wait in place. */
  fadeIn?: boolean;
  layer: HTMLElement;
  cleanup?: () => void;
}

function overlaps(a: Box, b: Box): boolean {
  return a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;
}

function other(seat: number): number {
  return seat === 0 ? 1 : 0;
}

function copySnap(s: Snap): Snap {
  return { box: s.box, clone: s.clone ? (s.clone.cloneNode(true) as HTMLElement) : null };
}

function copyCopies(c: Copies): Copies {
  return {
    sources: new Map([...c.sources].map(([i, s]) => [i, copySnap(s)] as const)),
    flips: new Map([...c.flips].map(([i, s]) => [i, copySnap(s)] as const)),
    loots: new Map([...c.loots].map(([i, l]) => [i, l.map(copySnap)] as const)),
    structure: c.structure ? copySnap(c.structure) : null,
    pawns: c.pawns.map((p) => ({ ...p })),
    bands: new Set(c.bands)
  };
}

/** A shown element that is not inside a seat panel (a token on the board or in a choice). */
function outsidePanels(el: HTMLElement): boolean {
  return el.closest('.seat-panel') === null;
}

export class AnimRunner {
  #speed: () => AnimationSpeed;
  #backlog: () => number;
  #layer: () => HTMLElement | null;
  #topLayer: () => HTMLElement | null;
  #root: () => ParentNode | null;
  #now: () => number;
  #runs: Runs;
  #topUsers = 0;
  #kept: Kept | null = null;
  #deal = new DealCarry();

  constructor(opts: AnimRunnerOptions) {
    this.#speed = opts.speed;
    this.#backlog = opts.backlog ?? (() => 0);
    this.#layer = opts.layer ?? fxLayer;
    this.#topLayer = opts.top ?? fxTop;
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
    this.#deal.clear();
    this.#finishRuns();
  }

  #finishRuns(): void {
    cancelHaptic();
    this.#runs.finishAll();
  }

  /** The Presenter's capture hook: finishes what runs, then reads the old DOM. */
  capture(next: PlanFrame, prev: PlanFrame | null): Captured {
    this.#finishRuns();
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
    // A move that plays nothing withdraws the replay; a frame with no
    // events (a bot starting to think, a presence change) keeps it.
    const moved = next.events.length > 0 || !prev;
    if (speed === 'off' || !root || !layer) {
      this.#deal.clear();
      if (moved) this.#withdraw();
      return this.#none(record);
    }
    const plan = this.#deal.plan(next, prev, planTimeline(next, prev), this.#backlog());
    const timeline = scaleTimeline(plan, speed);
    if (timeline.steps.length === 0) {
      if (moved) this.#withdraw();
      return this.#none(record);
    }
    record.steps = timeline.steps.length;
    setPlanned(next.version, timeline);
    const cap = this.#read(root, next.version, speed, timeline, record);
    this.#kept = {
      version: next.version,
      plan,
      cues: planCues(next, prev).cues,
      copies: copyCopies(cap)
    };
    replay.offer(next.version, () => this.replay());
    return cap;
  }

  #read(
    root: ParentNode,
    version: number,
    speed: AnimationSpeed,
    timeline: Timeline,
    record: FrameRecord
  ): Captured {
    const cap: Captured = {
      version,
      speed,
      timeline,
      record,
      sources: new Map(),
      flips: new Map(),
      pawns: [],
      loots: new Map(),
      structure: null,
      bands: new Set(),
      counters: new Map()
    };
    const snap = (el: Element | null | undefined): Snap | null => {
      if (!el) return null;
      const box = boxOf(el);
      return shown(box) ? { box, clone: inertClone(el) } : null;
    };
    const taken: string[] = [];
    timeline.steps.forEach((step, i) => {
      switch (step.kind) {
        case 'take': {
          const s = snap(
            root.querySelector(`.card-structure [data-card="${cssEscape(step.card)}"]`)
          );
          // The flight shows the card itself, not the marks of the tap that sent it.
          s?.clone?.classList.remove('selected', 'pending');
          if (s) cap.sources.set(i, s);
          taken.push(step.card);
          break;
        }
        case 'rebuild': {
          const card = cssEscape(step.card);
          const s = snap(findShown(root, [`dialog[open] [data-fx-pile-card="${card}"]`])?.el);
          if (s) cap.sources.set(i, s);
          break;
        }
        case 'destroy': {
          // The destroyer's own choice shows the card; everyone sees the band it leaves.
          const card = cssEscape(step.card);
          const inChoice = snap(
            findShown(root, [`dialog[open] [data-fx-pile-card="${card}"]`])?.el
          );
          if (inChoice) {
            cap.sources.set(i, inChoice);
            break;
          }
          const panel = this.#panel(root, other(step.seat));
          const color = cardBySlug(step.card)?.color;
          const selectors = color
            ? [`[data-fx-band="${color}"]`, '[data-fx-city]']
            : ['[data-fx-city]'];
          const band = panel ? findShown(panel, selectors) : null;
          if (band) cap.sources.set(i, { box: band.box, clone: null });
          break;
        }
        case 'token': {
          const token = cssEscape(step.token);
          const found = findShown(
            root,
            [`dialog[open] [data-fx-token="${token}"]`, `[data-fx-token="${token}"]`],
            outsidePanels
          );
          const s = snap(found?.el);
          if (s) cap.sources.set(i, s);
          break;
        }
        case 'draft': {
          const w = cssEscape(step.wonder);
          const found = findShown(root, [
            `[data-wonder-offer="${w}"] [data-wonder]`,
            `[data-wonder-offer="${w}"]`
          ]);
          const s = snap(found?.el);
          if (s) cap.sources.set(i, s);
          break;
        }
        case 'flip': {
          const s = snap(root.querySelector(`.card-structure [data-slot="${step.slot}"]`));
          if (s) cap.flips.set(step.slot, s);
          break;
        }
        case 'pawn':
          cap.pawns = [...root.querySelectorAll<HTMLElement>('[data-fx-pawn]')]
            .map((el) => ({ el, box: boxOf(el) }))
            .filter((p) => shown(p.box));
          break;
        case 'loot': {
          if (step.cell === undefined) break;
          const tokens = [
            ...root.querySelectorAll(`[data-loot-at="${step.cell}"] .loot-token[data-present]`)
          ]
            .map(snap)
            .filter((s): s is Snap => s !== null);
          if (tokens.length > 0) cap.loots.set(i, tokens);
          break;
        }
        case 'deal': {
          const inner = step.fade > 0 ? root.querySelector('.card-structure > div') : null;
          if (!inner) break;
          const box = boxOf(inner);
          if (!shown(box)) break;
          const copy = inner.cloneNode(true) as HTMLElement;
          // The age's last card leaves by its own flight.
          for (const card of taken) {
            for (const el of copy.querySelectorAll(`[data-card="${cssEscape(card)}"]`)) {
              el.remove();
            }
          }
          makeInert(copy);
          cap.structure = { box, clone: copy };
          break;
        }
        default:
          break;
      }
    });
    for (const el of root.querySelectorAll<HTMLElement>('.seat-panel [data-fx-band]')) {
      const seat = el.closest('.seat-panel')?.getAttribute('data-seat');
      cap.bands.add(`${seat}:${el.dataset['fxBand']}`);
    }
    return cap;
  }

  #withdraw(): void {
    this.#kept = null;
    replay.withdraw();
  }

  /**
   * Plays the last animated move again on the screen as it is now: the
   * same steps, the copies its capture made, the current speed, and its
   * sounds when Sound is on. Changes no state and holds no input; a new
   * frame finishes it like any other. False when there is nothing to replay
   * or Animation is off.
   */
  replay(): boolean {
    const kept = this.#kept;
    const speed = this.#speed();
    if (!kept || speed === 'off' || !this.#root() || !this.#layer()) return false;
    this.finish();
    const timeline = scaleTimeline(kept.plan, speed);
    const cap: Captured = {
      ...copyCopies(kept.copies),
      version: kept.version,
      speed,
      timeline,
      // Not in the frame records: a replay is no frame.
      record: { version: kept.version, swapAt: this.#now(), releasedAt: null, steps: 0 },
      counters: new Map()
    };
    void this.play(cap);
    playRecordedCues(retimeCues(kept.cues, timeline), kept.version, true);
    return true;
  }

  /** A frame with nothing to animate: input goes as soon as it shows. */
  #none(record: FrameRecord): Captured {
    return {
      version: record.version,
      speed: 'off',
      timeline: EMPTY_TIMELINE,
      record,
      sources: new Map(),
      flips: new Map(),
      pawns: [],
      loots: new Map(),
      structure: null,
      bands: new Set(),
      counters: new Map()
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
    cap.timeline.steps.forEach((step, i) => {
      const start = this.#prepare(step, i, cap, root, layer);
      if (start) starts.push(start);
    });
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
  #prepare(
    step: Step,
    i: number,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    switch (step.kind) {
      case 'take':
        return this.#prepareTake(step, i, cap, root, layer);
      case 'rebuild':
        return this.#prepareRebuild(step, i, cap, root, layer);
      case 'destroy':
        return this.#prepareDestroy(step, i, cap, root, layer);
      case 'token':
        return this.#prepareToken(step, i, cap, root, layer);
      case 'draft':
        return this.#prepareDraft(step, i, cap, root, layer);
      case 'flip':
        return this.#prepareFlip(step, cap, root, layer);
      case 'coins':
        return this.#prepareCoins(step, cap, root, layer);
      case 'pawn':
        return this.#preparePawn(step, cap);
      case 'loot':
        return this.#prepareLoot(step, i, cap, root, layer);
      case 'glow':
        return this.#prepareGlow(step, cap, root, layer);
      case 'draw':
        return this.#prepareDraw(step, cap, root, layer);
      case 'deal':
        return this.#prepareDeal(step, cap, root, layer);
      case 'victory':
        return this.#prepareVictory(step, cap, root);
    }
  }

  #panel(root: ParentNode, seat: number): HTMLElement | null {
    return root.querySelector<HTMLElement>(`.seat-panel[data-seat="${seat}"]`);
  }

  /** The top-layer part, shown above any open dialog until released. */
  #useTop(): { layer: HTMLElement; release: () => void } | null {
    const top = this.#topLayer();
    if (!top || typeof top.showPopover !== 'function') return null;
    try {
      // Shown again, so it lands above any dialog opened since.
      if (top.matches(':popover-open')) top.hidePopover();
      top.showPopover();
    } catch {
      return null;
    }
    this.#topUsers += 1;
    let released = false;
    return {
      layer: top,
      release: () => {
        if (released) return;
        released = true;
        this.#topUsers -= 1;
        if (this.#topUsers <= 0 && top.matches(':popover-open')) top.hidePopover();
      }
    };
  }

  /**
   * A clone's flight from `from` to `dest`: lifted, then shrunk into it,
   * fading into a small target. Returns the function that starts it.
   */
  #flight(cap: Captured, o: FlightOpts): () => void {
    const src = o.from;
    const centre = o.clamp
      ? clampInto(o.dest.box.x + o.dest.box.w / 2, o.dest.box.y + o.dest.box.h / 2, o.clamp)
      : centreOf(o.dest.box);
    // Small enough to sink into a chip, never so small it vanishes in flight.
    const scale = Math.max(0.3, Math.min(o.dest.box.w / src.w, o.dest.box.h / src.h, 1));
    const endX = centre.x - (src.w * scale) / 2;
    const endY = centre.y - (src.h * scale) / 2;
    const liftX = src.x + (endX - src.x) * 0.12;
    const liftY = src.y + (endY - src.y) * 0.12 - Math.min(16, src.h * 0.12);
    const clone = o.clone;
    place(clone, src);
    clone.classList.add('fx-flight');
    // Into a chip it fades out; onto a big target (a phone strip) it stays.
    const fade = o.fade ?? (scale <= 0.5 ? 0 : 0.4);
    const keyframes: Keyframe[] = [
      {
        transform: `translate(${src.x}px, ${src.y}px) scale(${o.fadeIn ? 0.6 : 1})`,
        opacity: o.fadeIn ? 0 : 1,
        offset: 0
      },
      {
        transform: `translate(${liftX}px, ${liftY}px) scale(1.06)`,
        opacity: 1,
        offset: 0.22
      },
      ...(o.fadeFrom !== undefined && o.fadeFrom > 0.22 && o.fadeFrom < 1
        ? [{ opacity: 1, offset: o.fadeFrom }]
        : []),
      { transform: `translate(${endX}px, ${endY}px) scale(${scale})`, opacity: fade, offset: 1 }
    ];
    const destEl = o.dest.el;
    return () => {
      o.layer.append(clone);
      if (o.hide) destEl.setAttribute(HIDDEN_ATTR, '');
      const anim = clone.animate(keyframes, {
        duration: o.dur,
        delay: o.at,
        easing: EASE_FLIGHT,
        fill: 'backwards'
      });
      const run = this.#runs.track(
        cap.version,
        o.kind,
        [anim],
        () => {
          clone.remove();
          if (o.hide) destEl.removeAttribute(HIDDEN_ATTR);
          o.cleanup?.();
        },
        false
      );
      anim.finished.then(
        () => {
          if (run.done) return;
          this.#runs.end(run, false);
          // A landing nudge on what was already there (an existing band,
          // the coins, the wonder).
          if (o.pulse && destEl.isConnected) this.#pulse(cap, destEl, o.kind);
        },
        () => this.#runs.end(run, true)
      );
    };
  }

  #pulse(cap: Captured, el: HTMLElement, kind: StepKind): void {
    const anim = el.animate(
      [{ transform: 'scale(1)' }, { transform: 'scale(1.12)' }, { transform: 'scale(1)' }],
      { duration: scaleMs(PULSE_MS, cap.speed), easing: EASE_OUT }
    );
    this.#runs.track(cap.version, kind, [anim], () => {});
  }

  /** A seat's city as a destination: the card colour's band, else the city. */
  #cityDest(
    cap: Captured,
    panel: HTMLElement,
    seat: number,
    card: string
  ): { dest: Dest; hide: boolean } | null {
    const panelBox = boxOf(panel);
    const color = cardBySlug(card)?.color;
    const selectors = color ? [`[data-fx-band="${color}"]`, '[data-fx-city]'] : ['[data-fx-city]'];
    const dest =
      findShown(panel, selectors) ?? (shown(panelBox) ? { el: panel, box: panelBox } : null);
    if (!dest) return null;
    // A colour's first card makes its band: it shows when the card lands.
    const hide = !!color && !cap.bands.has(`${seat}:${color}`) && dest.el.matches('[data-fx-band]');
    return { dest, hide };
  }

  #prepareTake(
    step: TakeStep,
    i: number,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const from = cap.sources.get(i);
    const panel = this.#panel(root, step.seat);
    if (!from?.clone || !panel) return null;
    const panelBox = boxOf(panel);
    let dest: Dest | null;
    let hide = false;
    if (step.dest === 'city') {
      const city = this.#cityDest(cap, panel, step.seat, step.card);
      dest = city?.dest ?? null;
      hide = city?.hide ?? false;
    } else {
      const w = step.wonder ? cssEscape(step.wonder) : '';
      const selectors =
        step.dest === 'discard'
          ? ['[data-tally="coins"]']
          : [`[data-fx-wonder="${w}"]`, '[data-fx-wonders]'];
      dest = findShown(panel, selectors) ?? (shown(panelBox) ? { el: panel, box: panelBox } : null);
    }
    if (!dest) return null;
    return this.#flight(cap, {
      kind: 'take',
      clone: from.clone,
      from: from.box,
      dest,
      clamp: panelBox,
      hide,
      pulse: !hide && dest.el !== panel,
      at: step.at,
      dur: step.dur,
      layer
    });
  }

  /** A card mounted in the FX layer, for a flight with no element to copy. */
  #mountCard(card: string, w: number): { el: HTMLElement; box: Box; release: () => void } {
    const h = Math.round(w * CARD_RATIO);
    const el = drawn('div', 'fx-card');
    const face = mount(CardFace, { target: el, props: { slug: card, tier: 'mini', width: w } });
    makeInert(el);
    return { el, box: { x: 0, y: 0, w, h }, release: () => void unmount(face) };
  }

  #prepareRebuild(
    step: RebuildStep,
    i: number,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const panel = this.#panel(root, step.seat);
    if (!panel) return null;
    const city = this.#cityDest(cap, panel, step.seat, step.card);
    if (!city) return null;
    const from = cap.sources.get(i);
    let clone = from?.clone ?? null;
    let box = from?.box;
    let release: (() => void) | undefined;
    if (!clone || !box) {
      // The pile was not on screen: the card drops in from above the city.
      const card = this.#mountCard(step.card, MOUNT_W);
      const c = centreOf(city.dest.box);
      clone = card.el;
      box = boxAround(c.x, Math.max(card.box.h / 2, c.y - 120), card.box.w, card.box.h);
      release = card.release;
    }
    return this.#flight(cap, {
      kind: 'rebuild',
      clone,
      from: box,
      dest: city.dest,
      clamp: boxOf(panel),
      hide: city.hide,
      pulse: !city.hide && city.dest.el !== panel,
      at: step.at,
      dur: step.dur,
      layer,
      ...(release ? { cleanup: release } : {})
    });
  }

  #prepareDestroy(
    step: DestroyStep,
    i: number,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const source = cap.sources.get(i);
    if (!source) return null;
    let clone = source.clone;
    let box = source.box;
    let release: (() => void) | undefined;
    if (!clone) {
      // From the band, which has no card to copy: the card itself, mounted.
      const card = this.#mountCard(step.card, MOUNT_W);
      const c = centreOf(source.box);
      clone = card.el;
      box = boxAround(c.x, c.y, card.box.w, card.box.h);
      release = card.release;
    }
    // A discard pile on the board, if one is ever drawn, takes it in.
    const pile = findShown(root, ['[data-fx-discard]']);
    if (pile) {
      return this.#flight(cap, {
        kind: 'destroy',
        clone,
        from: box,
        dest: pile,
        pulse: true,
        at: step.at,
        dur: step.dur,
        layer,
        ...(release ? { cleanup: release } : {})
      });
    }
    // Else it is thrown out: it rises off the city, tips and falls away.
    place(clone, box);
    clone.classList.add('fx-flight', 'fx-destroyed');
    const where = (dx: number, dy: number, turn: number, s: number) =>
      `translate(${box.x + dx}px, ${box.y + dy}px) rotate(${turn}deg) scale(${s})`;
    const keyframes: Keyframe[] = [
      { transform: where(0, 8, 0, 0.6), opacity: 0, offset: 0 },
      { transform: where(0, -18, -3, 1.05), opacity: 1, offset: 0.3 },
      { transform: where(4, -14, 2, 1), opacity: 1, offset: 0.5 },
      { transform: where(22, 48, 16, 0.82), opacity: 0, offset: 1 }
    ];
    const el = clone;
    return () => {
      layer.append(el);
      const anim = el.animate(keyframes, {
        duration: step.dur,
        delay: step.at,
        easing: 'ease-in-out',
        fill: 'backwards'
      });
      this.#runs.track(cap.version, 'destroy', [anim], () => {
        el.remove();
        release?.();
      });
    };
  }

  #prepareToken(
    step: TokenStep,
    i: number,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const from = cap.sources.get(i);
    const panel = this.#panel(root, step.seat);
    if (!from?.clone || !panel) return null;
    const token = cssEscape(step.token);
    const panelBox = boxOf(panel);
    const owned = findShown(panel, [`[data-fx-token="${token}"]`]);
    const dest =
      owned ??
      findShown(panel, ['[data-fx-tokens]', '[data-fx-city]']) ??
      (shown(panelBox) ? { el: panel, box: panelBox } : null);
    if (!dest) return null;
    return this.#flight(cap, {
      kind: 'token',
      clone: from.clone,
      from: from.box,
      dest,
      clamp: panelBox,
      hide: !!owned,
      pulse: !owned && dest.el !== panel,
      // Into the city's own medallion it stays whole: it becomes it.
      ...(owned ? { fade: 1 } : {}),
      at: step.at,
      dur: step.dur,
      layer
    });
  }

  #prepareDraft(
    step: DraftStep,
    i: number,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const from = cap.sources.get(i);
    if (!from?.clone) return null;
    const w = cssEscape(step.wonder);
    const picked = findShown(root, [`[data-draft-picks="${step.seat}"] [data-picked="${w}"]`]);
    const panel = this.#panel(root, step.seat);
    // The last pick of the draft lands on the board that replaced it.
    const dest =
      picked ??
      (panel
        ? findShown(panel, [`[data-fx-wonder="${w}"]`, '[data-fx-wonders]', '[data-fx-city]'])
        : null);
    if (!dest) return null;
    return this.#flight(cap, {
      kind: 'draft',
      clone: from.clone,
      from: from.box,
      dest,
      hide: !!picked,
      // The tile stays whole until it sinks into the chip, which pops as it shows.
      pulse: true,
      fadeFrom: 0.75,
      at: step.at,
      dur: step.dur,
      layer
    });
  }

  #prepareFlip(
    step: FlipStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const back = cap.flips.get(step.slot);
    const slot = root.querySelector<HTMLElement>(`.card-structure [data-slot="${step.slot}"]`);
    if (!back?.clone || !slot) return null;
    const box = boxOf(slot);
    if (!shown(box)) return null;
    const holder = this.#flipper(box, back.clone, inertClone(slot));
    return () => {
      layer.append(holder.el);
      slot.setAttribute(HIDDEN_ATTR, '');
      const anim = holder.spinner.animate(
        [{ transform: 'rotateY(0deg)' }, { transform: 'rotateY(180deg)' }],
        { duration: step.dur, delay: step.at, easing: EASE_OUT, fill: 'backwards' }
      );
      this.#runs.track(cap.version, 'flip', [anim], () => {
        holder.el.remove();
        slot.removeAttribute(HIDDEN_ATTR);
      });
    };
  }

  /** A two-faced card at `box`: `back` up, `front` on its other side. */
  #flipper(
    box: Box,
    back: HTMLElement,
    front: HTMLElement
  ): { el: HTMLElement; spinner: HTMLElement } {
    const holder = drawn('div', 'fx-flip');
    place(holder, box);
    holder.style.perspective = `${Math.round(Math.max(box.w, 24) * 6)}px`;
    const spinner = document.createElement('div');
    spinner.className = 'fx-flip-inner';
    for (const [face, turn] of [
      [back, 'none'],
      [front, 'rotateY(180deg)']
    ] as const) {
      place(face, { x: 0, y: 0, w: box.w, h: box.h });
      face.style.transform = turn;
      face.style.backfaceVisibility = 'hidden';
      face.classList.add('fx-face');
      spinner.append(face);
    }
    holder.append(spinner);
    return { el: holder, spinner };
  }

  /** The coin counter of a seat: its tally, and its tween for this frame. */
  #counter(cap: Captured, panel: HTMLElement): { box: Box; tween: CounterTween | null } | null {
    const tally = findShown(panel, ['[data-tally="coins"]']);
    if (!tally) return null;
    const count = tally.el.querySelector<HTMLElement>('[data-fx-count]');
    const text = count?.firstChild instanceof Text ? count.firstChild : null;
    if (!text || text.nodeValue === null) return { box: tally.box, tween: null };
    const known = cap.counters.get(text);
    if (known) return { box: tally.box, tween: known };
    // The counter is a run of its own: it outlives one float when two steps
    // move it (the coins of a build, then a looting).
    const holder: { run: ReturnType<Runs['track']> | null } = { run: null };
    const tween = new CounterTween(text, text.nodeValue, this.#now, () => {
      if (holder.run) this.#runs.end(holder.run, false);
    });
    holder.run = this.#runs.track(cap.version, 'coins', [], () => tween.stop(), false);
    cap.counters.set(text, tween);
    return { box: tally.box, tween };
  }

  /** A "+3" or "−2" float rising off a tally. */
  #float(
    cap: Captured,
    layer: HTMLElement,
    kind: StepKind,
    box: Box,
    delta: number,
    delay: number,
    dur: number
  ): () => void {
    const float = drawn('span', `fx-float ${delta < 0 ? 'fx-float-loss' : 'fx-float-gain'}`);
    float.textContent = formatSigned(delta, { signDisplay: 'exceptZero' });
    const x = box.x + box.w / 2;
    const y = box.y;
    const at = (dy: number) => `translate(${x}px, ${y + dy}px) translate(-50%, -60%)`;
    return () => {
      layer.append(float);
      const anim = float.animate(
        [
          { transform: at(0), opacity: 0, offset: 0 },
          { transform: at(-6), opacity: 1, offset: 0.15 },
          { transform: at(-26), opacity: 1, offset: 0.65 },
          { transform: at(-34), opacity: 0, offset: 1 }
        ],
        { duration: dur, delay, easing: 'linear', fill: 'backwards' }
      );
      this.#runs.track(cap.version, kind, [anim], () => float.remove());
    };
  }

  #prepareCoins(
    step: CoinStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const panel = this.#panel(root, step.seat);
    const counter = panel ? this.#counter(cap, panel) : null;
    if (!counter) return null;
    const float = this.#float(
      cap,
      layer,
      'coins',
      counter.box,
      step.to - step.from,
      step.at,
      step.dur
    );
    return () => {
      float();
      // The counter: the old number until the float starts, then a tween
      // to the new one, then the server's own text back.
      if (step.tween > 0) counter.tween?.add(step.at, step.tween, step.from, step.to);
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

  #prepareLoot(
    step: LootStep,
    i: number,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const starts: (() => void)[] = [];
    // The emptied spot under the token stays hidden until the token is gone.
    const spots =
      step.cell !== undefined && cap.loots.has(i)
        ? [...root.querySelectorAll<HTMLElement>(`[data-loot-at="${step.cell}"] .loot-token`)]
        : [];
    // The token flips (the coins it costs, then a blank coin), pops and fades.
    for (const token of cap.loots.get(i) ?? []) {
      if (!token.clone) continue;
      const blank = token.clone.cloneNode(true) as HTMLElement;
      blank.textContent = '';
      const flipper = this.#flipper(token.box, token.clone, blank);
      const holder = flipper.el;
      holder.style.transformOrigin = '50% 50%';
      const base = holder.style.transform;
      const total = step.flip + step.fade;
      const pop: Keyframe[] = [
        { transform: `${base} scale(1)`, opacity: 1, offset: 0 },
        { transform: `${base} scale(1.35)`, opacity: 1, offset: step.flip / total / 2 },
        { transform: `${base} scale(1.15)`, opacity: 1, offset: step.flip / total },
        { transform: `${base} translateY(-4px) scale(1.4)`, opacity: 0, offset: 1 }
      ];
      starts.push(() => {
        layer.append(holder);
        for (const spot of spots) spot.setAttribute(HIDDEN_ATTR, '');
        const spin = flipper.spinner.animate(
          [{ transform: 'rotateY(0deg)' }, { transform: 'rotateY(180deg)' }],
          { duration: step.flip, delay: step.at, easing: EASE_OUT, fill: 'backwards' }
        );
        const anim = holder.animate(pop, {
          duration: total,
          delay: step.at,
          easing: 'ease-out',
          fill: 'backwards'
        });
        this.#runs.track(cap.version, 'loot', [spin, anim], () => {
          holder.remove();
          for (const spot of spots) spot.removeAttribute(HIDDEN_ATTR);
        });
      });
    }
    // The victim's coins drop, with the loss float, as the token flips.
    const panel = this.#panel(root, step.seat);
    const counter = panel ? this.#counter(cap, panel) : null;
    if (counter) {
      const float = this.#float(cap, layer, 'loot', counter.box, -step.amount, step.at, step.dur);
      starts.push(() => {
        float();
        if (step.tween > 0) counter.tween?.add(step.at, step.tween, step.from, step.to);
      });
    }
    if (starts.length === 0) return null;
    return () => {
      for (const start of starts) start();
    };
  }

  #prepareGlow(
    step: GlowStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const panel = this.#panel(root, step.seat);
    if (!panel) return null;
    const symbol = cssEscape(step.symbol);
    // The pair's symbol in the tracker (or the phone's pill), and the band
    // its two green cards are in.
    const targets = [
      ...panel.querySelectorAll<HTMLElement>(
        `[data-fx-science="${symbol}"], [data-fx-science-pill], [data-fx-band="green"]`
      )
    ]
      .map(boxOf)
      .filter(shown);
    if (targets.length === 0) return null;
    const pad = 4;
    const glows = targets.map((box) => {
      const el = drawn('span', 'fx-glow');
      place(el, { x: box.x - pad, y: box.y - pad, w: box.w + pad * 2, h: box.h + pad * 2 });
      el.style.transformOrigin = '50% 50%';
      return { el, base: el.style.transform };
    });
    return () => {
      // Over a dialog's backdrop when the dialog leaves the symbol in view
      // (the desktop's centred token choice), under a sheet that covers it.
      const dialogs = [...document.querySelectorAll('dialog[open]')].map(boxOf).filter(shown);
      const clear =
        dialogs.length > 0 && !dialogs.some((d) => targets.some((box) => overlaps(d, box)));
      const top = clear ? this.#useTop() : null;
      const into = top?.layer ?? layer;
      const anims = glows.map(({ el, base }) => {
        into.append(el);
        return el.animate(
          [
            { transform: `${base} scale(0.85)`, opacity: 0, offset: 0 },
            { transform: `${base} scale(1.05)`, opacity: 1, offset: 0.3 },
            { transform: `${base} scale(1.1)`, opacity: 0.85, offset: 0.6 },
            { transform: `${base} scale(1.3)`, opacity: 0, offset: 1 }
          ],
          { duration: step.dur, delay: step.at, easing: 'ease-out', fill: 'backwards' }
        );
      });
      this.#runs.track(cap.version, 'glow', anims, () => {
        for (const { el } of glows) el.remove();
        top?.release();
      });
    };
  }

  #prepareDraw(
    step: DrawStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const panel = this.#panel(root, step.seat);
    if (!panel) return null;
    const panelBox = boxOf(panel);
    const source =
      findShown(panel, [
        '[data-fx-wonder="great-library"]',
        '[data-fx-wonders]',
        '[data-fx-city]'
      ]) ?? (shown(panelBox) ? { el: panel, box: panelBox } : null);
    if (!source) return null;
    const from = centreOf(source.box);

    // The drawer sees the token itself land in its choice, over the dialog.
    const face = step.token
      ? findShown(root, [`dialog[open] [data-fx-token="${cssEscape(step.token)}"]`])
      : null;
    if (face) {
      const clone = inertClone(face.el);
      // The choice may still be sliding in: aim where its token comes to rest.
      const rest = { el: face.el, box: settled(face.el, face.box) };
      const start = boxAround(from.x, from.y, rest.box.w, rest.box.h);
      return () => {
        const top = this.#useTop();
        this.#flight(cap, {
          kind: 'draw',
          clone,
          from: start,
          dest: rest,
          hide: true,
          fade: 1,
          fadeIn: true,
          at: step.at,
          dur: step.each,
          layer: top?.layer ?? layer,
          ...(top ? { cleanup: top.release } : {})
        })();
      };
    }

    // Everyone else (and a drawer whose choice is not open): a back, dealt
    // to a fan over the board, held a moment, then gone. Nothing in it
    // names the token.
    const area = findShown(root, ['.card-structure > div', '.card-structure']) ?? {
      el: panel,
      box: panelBox
    };
    const c = centreOf(area.box);
    const fanX = c.x + (step.place - 1) * BACK_PX * 1.3;
    const fanY = c.y;
    const back = drawn('span', 'fx-token-back');
    const svg = document.createElementNS(SVG_NS, 'svg');
    const use = document.createElementNS(SVG_NS, 'use');
    use.setAttribute('href', '#i-token-frame');
    svg.append(use);
    back.append(svg);
    place(back, { x: 0, y: 0, w: BACK_PX, h: BACK_PX });
    const total = step.each + step.hold + step.fade;
    const pos = (x: number, y: number, s: number, turn = 0) =>
      `translate(${x - BACK_PX / 2}px, ${y - BACK_PX / 2}px) rotate(${turn}deg) scale(${s})`;
    const keyframes: Keyframe[] = [
      { transform: pos(from.x, from.y, 0.4, -40), opacity: 0, offset: 0 },
      { transform: pos(from.x, from.y - 12, 0.7, -20), opacity: 1, offset: 0.08 },
      { transform: pos(fanX, fanY, 1), opacity: 1, offset: step.each / total },
      { transform: pos(fanX, fanY, 1), opacity: 1, offset: (step.each + step.hold) / total },
      { transform: pos(fanX, fanY - 10, 0.9), opacity: 0, offset: 1 }
    ];
    return () => {
      layer.append(back);
      const anim = back.animate(keyframes, {
        duration: total,
        delay: step.at,
        easing: EASE_FLIGHT,
        fill: 'backwards'
      });
      this.#runs.track(cap.version, 'draw', [anim], () => back.remove());
    };
  }

  #prepareDeal(
    step: DealStep,
    cap: Captured,
    root: ParentNode,
    layer: HTMLElement
  ): (() => void) | null {
    const structure = root.querySelector<HTMLElement>('.card-structure');
    const old = cap.structure;
    if (!structure && !old) return null;
    let started = 0;

    const fadeOld = () => {
      if (!old?.clone) return;
      const el = old.clone;
      place(el, old.box);
      el.classList.add('fx-old-structure');
      layer.append(el);
      const anim = el.animate([{ opacity: 1 }, { opacity: 0 }], {
        duration: step.fade,
        delay: step.at,
        easing: 'ease-out',
        fill: 'backwards'
      });
      this.#runs.track(cap.version, 'deal', [anim], () => el.remove());
    };

    // Each card of the new structure, from a deck above it, top row first.
    const deal = (slots: HTMLElement[]) => {
      const cards = slots
        .map((el) => ({ el, box: boxOf(el) }))
        .filter((card) => shown(card.box))
        .sort((a, b) => Math.round(a.box.y) - Math.round(b.box.y) || a.box.x - b.box.x);
      const first = cards[0];
      if (!first || !structure) return;
      const area = boxOf(structure.firstElementChild ?? structure);
      const deckX = area.x + area.w / 2 - first.box.w / 2;
      const deckY = Math.max(0, area.y - first.box.h * 0.35);
      const late = this.#now() - started;
      cards.forEach(({ el, box }, n) => {
        const clone = inertClone(el);
        clone.classList.remove('selected', 'pending');
        place(clone, box);
        clone.classList.add('fx-flight', 'fx-dealt');
        layer.append(clone);
        el.setAttribute(HIDDEN_ATTR, '');
        const anim = clone.animate(
          [
            {
              transform: `translate(${deckX}px, ${deckY}px) rotate(-7deg) scale(0.7)`,
              opacity: 0,
              offset: 0
            },
            { opacity: 1, offset: 0.25 },
            {
              transform: `translate(${box.x}px, ${box.y}px) rotate(0deg) scale(1)`,
              opacity: 1,
              offset: 1
            }
          ],
          {
            duration: step.each,
            delay: Math.max(0, step.at + step.lead + n * step.stagger - late),
            easing: EASE_FLIGHT,
            fill: 'backwards'
          }
        );
        this.#runs.track(cap.version, 'deal', [anim], () => {
          clone.remove();
          el.removeAttribute(HIDDEN_ATTR);
        });
      });
    };
    const slotsOf = () =>
      structure
        ? [...structure.querySelectorAll<HTMLElement>('[data-slot]:not([data-state="empty"])')]
        : [];

    return () => {
      started = this.#now();
      fadeOld();
      const slots = slotsOf();
      if (slots.length > 0 || !structure) {
        deal(slots);
        return;
      }
      // A structure mounted with this frame lays out its cards a moment
      // later (it measures its box first): deal them as they appear, in the
      // same task that adds them, so none is ever painted before its turn.
      const watch = new MutationObserver(() => {
        const now = slotsOf();
        if (now.length === 0) return;
        watch.disconnect();
        deal(now);
      });
      watch.observe(structure, { childList: true, subtree: true });
      const run = this.#runs.track(cap.version, 'deal', [], () => watch.disconnect(), false);
      setTimeout(() => this.#runs.end(run, false), Math.max(400, step.lead + step.each));
    };
  }

  #prepareVictory(step: VictoryStep, cap: Captured, root: ParentNode): (() => void) | null {
    if (typeof window === 'undefined') return null;
    const banner = victoryBanner(step.victory, step.winner);
    const dialog = findShown(root, ['[data-game-over] dialog[open]']);
    const vw = window.innerWidth;
    const y = dialog ? Math.max(56, settled(dialog.el, dialog.box).y) : window.innerHeight * 0.3;
    banner.el.style.transform = `translate(${vw / 2}px, ${y}px)`;
    const reach = Math.min(vw, 480);
    const d = step.dur;
    const ribbon: Keyframe[] = [
      { transform: 'translate(-50%, -50%) scale(0.6)', opacity: 0, offset: 0 },
      { transform: 'translate(-50%, -50%) scale(1.08)', opacity: 1, offset: 0.12 },
      { transform: 'translate(-50%, -50%) scale(1)', opacity: 1, offset: 0.2 },
      { transform: 'translate(-50%, -50%) scale(1)', opacity: 1, offset: 0.8 },
      { transform: 'translate(-50%, -62%) scale(0.98)', opacity: 0, offset: 1 }
    ];
    return () => {
      const top = this.#useTop();
      const layer = top?.layer ?? this.#layer();
      if (!layer) return;
      layer.append(banner.el);
      const anims = [
        banner.ribbon.animate(ribbon, {
          duration: d,
          delay: step.at,
          easing: EASE_OUT,
          fill: 'backwards'
        }),
        ...banner.particles.map((p, i) => {
          const o = particleOffset(i, reach);
          return p.animate(
            [
              { transform: 'translate(-50%, -50%) scale(0.3)', opacity: 0, offset: 0 },
              { opacity: 1, offset: 0.12 },
              { opacity: 1, offset: 0.6 },
              {
                transform: `translate(-50%, -50%) translate(${o.x}px, ${o.y}px) rotate(${o.turn}deg) scale(1)`,
                opacity: 0,
                offset: 1
              }
            ],
            {
              duration: Math.round(d * 0.6),
              delay: step.at + Math.round(d * 0.06) + (i % 3) * Math.round(d * 0.02),
              easing: 'cubic-bezier(0.1, 0.7, 0.3, 1)',
              fill: 'backwards'
            }
          );
        })
      ];
      this.#runs.track(cap.version, 'victory', anims, () => {
        banner.el.remove();
        top?.release();
      });
    };
  }
}
