// The tutorial's page state: which info lessons were acknowledged (Next),
// and the coach step on screen now, for the test hook (session/test-hook.ts).
//
// Acknowledgements are kept for the tab (sessionStorage), keyed by room
// code, ply and lesson index (steps.ts ackKey), so a reload mid-lesson
// shows the same step. An undo forgets the room's acknowledgements from the
// ply it went back to, so the guided turn's lessons show again.

import { SvelteSet } from 'svelte/reactivity';

/** The id of the step's text, in the bubble or in a sheet's note (only
 *  one of them shows at a time), for the target's aria-describedby. */
export const COACH_TEXT_ID = 'coach-step-text';

const ACKS_KEY = 'ajayeb:tutorial-acks';
/** The keys kept; the oldest are dropped first. */
const MAX_ACKS = 200;

function storage(): Pick<Storage, 'getItem' | 'setItem'> | null {
  try {
    return globalThis.sessionStorage ?? null;
  } catch {
    return null;
  }
}

function load(): string[] {
  try {
    const raw = storage()?.getItem(ACKS_KEY);
    const data: unknown = raw ? JSON.parse(raw) : [];
    return Array.isArray(data) ? data.filter((k): k is string => typeof k === 'string') : [];
  } catch {
    return [];
  }
}

/** The acknowledged info lessons of this tab. */
export class CoachAcks {
  #keys: SvelteSet<string>;

  constructor(initial: readonly string[] = load()) {
    this.#keys = new SvelteSet(initial);
  }

  has(key: string): boolean {
    return this.#keys.has(key);
  }

  add(key: string): void {
    if (this.#keys.has(key)) return;
    this.#keys.add(key);
    this.#save();
  }

  /** Forgets a room's acknowledgements at `ply` and later (an undo). */
  forgetFrom(code: string, ply: number): void {
    let changed = false;
    for (const key of [...this.#keys]) {
      const [c, p] = key.split(':');
      if (c === code && Number(p) >= ply) {
        this.#keys.delete(key);
        changed = true;
      }
    }
    if (changed) this.#save();
  }

  #save(): void {
    const all = [...this.#keys].slice(-MAX_ACKS);
    try {
      storage()?.setItem(ACKS_KEY, JSON.stringify(all));
    } catch {
      // Blocked or full: the acknowledgement still holds for this page.
    }
  }
}

export const coachAcks = new CoachAcks();

/** The coach step on screen, as the test hook reports it. */
export interface CoachShown {
  /** The lesson id. */
  readonly step: string;
  readonly kind: 'info' | 'tap' | 'act' | 'note';
  /** Where it shows: the bubble, or a note in a sheet. */
  readonly surface: 'board' | 'card-sheet' | 'draft-sheet';
  /** The selector list of its target, joined; null for a note. */
  readonly selector: string | null;
}

let source: (() => CoachShown | null) | null = null;

/** The game screen registers what it shows (a getter over its derived
 *  state, so nothing is written per step); null unregisters it. */
export function setCoachSource(fn: (() => CoachShown | null) | null): void {
  source = fn;
}

/** The coach step on screen now, for window.__ajayeb.coach(). */
export function coachShown(): CoachShown | null {
  return source?.() ?? null;
}

/** A coach step as a sheet's note (CoachNote's props), or null. */
export interface CoachNoteView {
  readonly title: string;
  readonly progress: string | null;
  readonly text: string;
  readonly step: string;
  readonly kind: 'info' | 'tap' | 'act' | 'note';
  readonly textId: string;
  readonly onpause?: () => void;
}
