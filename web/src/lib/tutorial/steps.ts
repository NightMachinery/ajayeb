// The tutorial's step machine (the tutorial design, section 3.2): from the
// server's tutorial object (State.tutorial of the frame on screen), the
// info lessons acknowledged on this device and a snapshot of the UI, the
// one coach step to show now, or none. Pure, so it is unit-tested in
// steps.test.ts.
//
// The server derives which lessons to show and the one action key the
// guided turn asks for; this module only expands that into UI steps:
//
// - Info lessons come first, in order: the first one not acknowledged is
//   the step (kind 'info', ended by Next).
// - Then the action lesson, by the key's verb:
//   - construct:X and discard:X: the card sheet open on X is an 'act' step
//     on the action button (a note in the sheet); otherwise a 'tap' step on
//     the structure's card X (a bubble on the board).
//   - wonder:W:X: the sheet on X in its wonder step is an 'act' step on the
//     button; the sheet on X in its card step a 'tap' step on "Build a
//     wonder" (a note); otherwise a 'tap' step on card X.
//   - draft:W: the phone's wonder sheet open on W is an 'act' step on Take
//     (a note); where the page shows every tile's Take button (the split
//     layout) an 'act' step on it (a bubble); otherwise a 'tap' step on the
//     tile.
// - A sheet open on another card (or wonder) is a 'note' step that names
//   the expected one, with no target.
// - The wonder-first pick bar is up: no step (the action, if it is the
//   guided one, still completes the turn).
//
// Nothing here decides what is legal: the target is always the server's
// action key, and the step only says where its button is now.

import type { TutorialLesson, TutorialState } from '$lib/wire';

/** The lesson ids this client words (TutorialLessonID is a plain string
 *  on the wire: tygo cannot make it a union). */
export const LESSON_IDS = [
  'draftIntro',
  'draftRound2',
  'draftPick',
  'draftPickAgain',
  'ageOne',
  'turn',
  'buildFree',
  'reveal',
  'buildOwn',
  'military',
  'trade',
  'coins',
  'discard',
  'wonders',
  'buildWonder',
  'science',
  'tokens',
  'chain',
  'playOn',
  'leftPath'
] as const;
export type LessonId = (typeof LESSON_IDS)[number];

export function isLessonId(value: unknown): value is LessonId {
  return (LESSON_IDS as readonly unknown[]).includes(value);
}

/** What a step asks of the player: read and press Next, tap a thing on the
 *  way to the move, make the move, or only read a note. */
export type CoachKind = 'info' | 'tap' | 'act' | 'note';

/** Where the step shows: a bubble over the board, or a note inside the
 *  card sheet or the draft's wonder sheet. */
export type CoachSurface = 'board' | 'card-sheet' | 'draft-sheet';

/** Which words a step shows (lib/tutorial/text.ts turns them into text). */
export type CoachWords =
  /** The lesson's own text (tutorial.lesson.<id> or tutorial.action.<id>). */
  | 'lesson'
  /** "Tap {card}." */
  | 'tapCard'
  /** "Choose Build a wonder." */
  | 'openWonders'
  /** "Tap {wonder}, then Take." */
  | 'takeWonder'
  /** "This lesson takes {card}. Close this and tap it." */
  | 'otherCard'
  /** The same for a wonder in the draft. */
  | 'otherWonder'
  /** The card's sheet is in its wonder step, but the lesson builds the card. */
  | 'back';

export interface CoachStep {
  /** A stable id of this step: the room, ply, lesson and UI step. A new id
   *  is a new step (focus, the live region). */
  readonly key: string;
  /** The lesson id (data-coach-step). */
  readonly id: string;
  /** Whether the lesson is an info lesson or the turn's action. */
  readonly lesson: 'info' | 'action';
  readonly kind: CoachKind;
  readonly surface: CoachSurface;
  readonly words: CoachWords;
  /** The anchor's selectors, the first visible match wins (anchors.ts);
   *  null for a note with no target. */
  readonly target: readonly string[] | null;
  /** The target is a region (the structure, the draft offer): the bubble
   *  may cover a part of it. */
  readonly region: boolean;
  /** The card and wonder the lesson names (catalog slugs). */
  readonly card?: string;
  readonly wonder?: string;
  /** The guided turn, 1-based, and the number of guided turns; 0 outside
   *  the guided part. */
  readonly turn: number;
  readonly turns: number;
  /** The acknowledgement key of an info step (its Next). */
  readonly ack?: string;
  /** The guided action key, on an action step. */
  readonly actionKey?: string;
}

/** The UI around the board, as the step machine needs it. */
export interface CoachUi {
  /** The open card sheet: its card's slug (null for a face-down card) and
   *  its step; null while it is closed. */
  readonly sheet: { readonly card: string | null; readonly step: 'card' | 'wonders' } | null;
  /** The phone draft's open wonder sheet: its wonder; null while closed. */
  readonly draftSheet: string | null;
  /** The draft shows every tile's Take button on the page (split layout). */
  readonly split: boolean;
  /** The wonder-first pick bar is up. */
  readonly pickBar: boolean;
}

export const CLOSED_UI: CoachUi = { sheet: null, draftSheet: null, split: false, pickBar: false };

export interface StepInput {
  /** The presented frame's tutorial object; absent outside a tutorial room. */
  readonly tutorial: TutorialState | undefined;
  readonly code: string;
  readonly ply: number;
  /** Whether the player paused the tutorial in this room. */
  readonly paused: boolean;
  /** Whether an info lesson's acknowledgement key was acknowledged. */
  readonly acked: (key: string) => boolean;
  /** Info lessons the client adds after the server's (the leftPath shown
   *  on resuming after leaving the path). */
  readonly extra?: readonly TutorialLesson[];
  readonly ui: CoachUi;
}

/** The acknowledgement key of info lesson `index` at a position. */
export function ackKey(code: string, ply: number, index: number, id: string): string {
  return `${code}:${ply}:${index}:${id}`;
}

/** Where an info lesson points. */
function infoTarget(lesson: TutorialLesson): { target: string[] | null; region: boolean } {
  switch (lesson.id) {
    case 'draftIntro':
    case 'draftRound2':
      return { target: ['[data-tut="draft-offer"]'], region: true };
    case 'ageOne':
      return { target: ['[data-tut="structure"]'], region: true };
    case 'turn':
    case 'playOn':
    case 'leftPath':
      return { target: ['[data-tut="turn-bar"]'], region: false };
    case 'reveal':
      return lesson.card
        ? { target: [cardTarget(lesson.card)], region: false }
        : { target: ['[data-tut="structure"]'], region: true };
    case 'military':
      return { target: ['[data-tut="track"] [data-pawn]'], region: false };
    case 'coins':
      return { target: ['[data-tut="home"] [data-tally="coins"]'], region: false };
    case 'wonders':
      return { target: ['[data-tut="home"] [data-tut="wonders"]'], region: false };
    case 'tokens':
      return { target: ['[data-tut="tokens"]'], region: false };
    default:
      return { target: null, region: false };
  }
}

/** The structure's card with this slug. */
export function cardTarget(card: string): string {
  return `[data-tut="structure"] [data-card="${card}"]`;
}

/** An action button. */
export function actTarget(key: string): string {
  return `[data-act="${key}"]`;
}

/** The parts of an action key: its verb and slugs (wonder:W:X, draft:W,
 *  construct:X, discard:X). */
export function parseKey(key: string): { verb: string; card?: string; wonder?: string } {
  const [verb = '', a, b] = key.split(':');
  switch (verb) {
    case 'construct':
    case 'discard':
      return a ? { verb, card: a } : { verb };
    case 'wonder':
      return a && b ? { verb, wonder: a, card: b } : { verb };
    case 'draft':
      return a ? { verb, wonder: a } : { verb };
    default:
      return { verb };
  }
}

/** The coach step to show now, or null (none due, paused, the free part). */
export function coachStep(input: StepInput): CoachStep | null {
  const tut = input.tutorial;
  if (!tut || input.paused) return null;
  const base = `${input.code}:${input.ply}`;
  const turn = tut.turn;
  const turns = tut.turns;

  const lessons = [...tut.lessons, ...(input.extra ?? [])];
  for (const [i, lesson] of lessons.entries()) {
    const ack = ackKey(input.code, input.ply, i, lesson.id);
    if (input.acked(ack)) continue;
    const { target, region } = infoTarget(lesson);
    return {
      key: `${base}:info:${i}:${lesson.id}`,
      id: lesson.id,
      lesson: 'info',
      kind: 'info',
      surface: 'board',
      words: 'lesson',
      target,
      region,
      ...(lesson.card ? { card: lesson.card } : {}),
      ...(lesson.wonder ? { wonder: lesson.wonder } : {}),
      turn,
      turns,
      ack
    };
  }

  const action = tut.status === 'guided' ? tut.action : undefined;
  if (!action) return null;
  const ui = input.ui;
  if (ui.pickBar) return null;
  const parts = parseKey(action.key);
  const card = action.card ?? parts.card;
  const wonder = action.wonder ?? parts.wonder;
  const step = (
    kind: CoachKind,
    surface: CoachSurface,
    words: CoachWords,
    target: readonly string[] | null
  ): CoachStep => ({
    key: `${base}:action:${action.id}:${kind}:${surface}:${words}`,
    id: action.id,
    lesson: 'action',
    kind,
    surface,
    words,
    target,
    region: false,
    ...(card ? { card } : {}),
    ...(wonder ? { wonder } : {}),
    turn,
    turns,
    actionKey: action.key
  });

  switch (parts.verb) {
    case 'construct':
    case 'discard': {
      if (!card) return null;
      if (ui.sheet) {
        if (ui.sheet.card !== card) return step('note', 'card-sheet', 'otherCard', null);
        if (ui.sheet.step === 'wonders') return step('note', 'card-sheet', 'back', null);
        return step('act', 'card-sheet', 'lesson', [actTarget(action.key)]);
      }
      return step('tap', 'board', 'tapCard', [cardTarget(card)]);
    }
    case 'wonder': {
      if (!card) return null;
      if (ui.sheet) {
        if (ui.sheet.card !== card) return step('note', 'card-sheet', 'otherCard', null);
        if (ui.sheet.step === 'wonders') {
          return step('act', 'card-sheet', 'lesson', [actTarget(action.key)]);
        }
        return step('tap', 'card-sheet', 'openWonders', ['[data-control="wonder-step"]']);
      }
      return step('tap', 'board', 'tapCard', [cardTarget(card)]);
    }
    case 'draft': {
      if (!wonder) return null;
      if (ui.draftSheet !== null) {
        if (ui.draftSheet !== wonder) return step('note', 'draft-sheet', 'otherWonder', null);
        return step('act', 'draft-sheet', 'lesson', [actTarget(action.key)]);
      }
      if (ui.split) return step('act', 'board', 'lesson', [actTarget(action.key)]);
      return step('tap', 'board', 'takeWonder', [`[data-pick="${action.key}"]`]);
    }
    default:
      return null;
  }
}
