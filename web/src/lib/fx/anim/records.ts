// What the animation runner did, for the end-to-end tests (window.__ajayebFx,
// fx/index.ts): one record per frame the runner was handed, and one per
// animation it started. Times are performance.now() in the page.

import type { StepKind, Timeline } from './plan';

export interface FrameRecord {
  /** The frame's version. */
  version: number;
  /** When the Presenter swapped it in (the capture hook, just before). */
  swapAt: number;
  /** When the play hook let input go, or null while it still holds it. */
  releasedAt: number | null;
  /** The planned steps, after the speed (0 for Animation off). */
  steps: number;
}

export interface AnimRecord {
  version: number;
  kind: StepKind;
  startAt: number;
  /** When it ended or was finished early, or null while it runs. */
  endAt: number | null;
  /** Finished early by an interrupt (a new frame, hurry, clear). */
  interrupted: boolean;
}

const LIMIT = 500;
const frames: FrameRecord[] = [];
const anims: AnimRecord[] = [];

function push<T>(list: T[], item: T): T {
  list.push(item);
  if (list.length > LIMIT) list.splice(0, list.length - LIMIT);
  return item;
}

export function recordFrame(rec: FrameRecord): FrameRecord {
  return push(frames, rec);
}

export function recordAnim(rec: AnimRecord): AnimRecord {
  return push(anims, rec);
}

export function frameRecords(): FrameRecord[] {
  return frames.map((f) => ({ ...f }));
}

export function animRecords(): AnimRecord[] {
  return anims.map((a) => ({ ...a }));
}

export function clearRecords(): void {
  frames.length = 0;
  anims.length = 0;
}

// The timeline the runner plays for a frame, at the speed it plays it, for
// the sound cues (fx/index.ts times them by it). Set at each capture; Off,
// a frame with nothing to animate and a fast-forwarded one leave it empty.
let planned: { version: number; timeline: Timeline } | null = null;

export function setPlanned(version: number, timeline: Timeline): void {
  planned = { version, timeline };
}

/** The timeline of the frame the runner saw last (empty when it plays none). */
export function latestPlanned(): Timeline | null {
  return planned?.timeline ?? null;
}

/** The timeline played for `version`, or null when the runner never saw it. */
export function plannedFor(version: number): Timeline | null {
  return planned && planned.version === version ? planned.timeline : null;
}
