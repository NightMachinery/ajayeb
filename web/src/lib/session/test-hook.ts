// window.__ajayeb: a read-only window onto the game this page shows, for
// end-to-end drivers (docs/testing.md, "The client test hook"). It is in
// every build, because it exposes only what this client already holds: the
// redacted view the server sent this viewer, never the identity token.
//
// A driver waits for ready(), reads choices(), and clicks the button whose
// data-act is the chosen key.

import { updated } from '$app/state';
import { coachShown, type CoachShown } from '$lib/tutorial/coach.svelte';
import type { Choice, HistoryEntry, RoomInfo, TutorialState, View, YouInfo } from '$lib/wire';
import type { Presenter } from './presenter.svelte';
import type { ConnState, Session } from './session.svelte';

export interface TestHookRoom extends RoomInfo {
  code: string;
  version: number;
  canUndo: boolean;
  thinking: number | null;
}

export interface AjayebTestHook {
  /** Checks SvelteKit's deployed build version without changing game state. */
  checkForUpdate(): Promise<boolean>;
  /** The view on screen, or null before the first state. */
  view(): View | null;
  /** The legal choices of the view on screen; each key is a [data-act] button. */
  choices(): Choice[];
  /** The room on screen, with its code and version. */
  room(): TestHookRoom | null;
  /** What this connection may do (seats, canAct). */
  you(): YouInfo | null;
  /** The connection state. */
  conn(): ConnState;
  /** The game's history as the Session holds it: one entry per action. */
  history(): HistoryEntry[];
  /**
   * True when the page shows the latest state and takes input: welcomed,
   * a state on screen, nothing queued in the Presenter, no request pending.
   */
  ready(): boolean;
  /** The tutorial object of the frame on screen, or null outside a tutorial room. */
  tutorial(): TutorialState | null;
  /** The coach step on screen (its lesson id, kind, surface and target
   *  selector), or null. */
  coach(): CoachShown | null;
}

declare global {
  interface Window {
    __ajayeb?: AjayebTestHook;
  }
}

interface Game {
  session: Session;
  presenter: Presenter;
}

let active: Game | null = null;
// The game whose screen is mounted. The live game exists before its screen
// (Duel's screen loads lazily after the route), and a driver must not act
// before the buttons it clicks are on the page.
let screen: Game | null = null;

// Copies, so a driver cannot change what the page renders.
function copy<T>(value: T): T {
  return value === null || value === undefined ? value : (JSON.parse(JSON.stringify(value)) as T);
}

const hook: AjayebTestHook = {
  checkForUpdate: () => updated.check(),
  view: () => copy(active?.presenter.view ?? null),
  choices: () => copy(active?.presenter.view?.choices ?? []),
  room: () => {
    const frame = active?.presenter.frame;
    if (!frame) return null;
    return copy({
      ...frame.room,
      code: frame.code,
      version: frame.version,
      canUndo: frame.canUndo,
      thinking: frame.thinking ?? null
    });
  },
  you: () => copy(active?.presenter.frame?.you ?? null),
  conn: () => active?.session.conn ?? 'connecting',
  history: () => copy([...(active?.session.history ?? [])]),
  tutorial: () => copy(active?.presenter.frame?.tutorial ?? null),
  coach: () => copy(coachShown()),
  ready: () => {
    if (!active || active !== screen) return false;
    const { session, presenter } = active;
    return (
      session.conn === 'open' &&
      presenter.frame !== null &&
      presenter.frame === session.frame &&
      presenter.idle &&
      session.pending === null
    );
  }
};

/** Points window.__ajayeb at this game (the page's live one). */
export function installTestHook(game: Game): void {
  active = game;
  if (typeof window !== 'undefined' && !window.__ajayeb) {
    Object.defineProperty(window, '__ajayeb', { value: hook, configurable: true });
  }
}

/** Stops pointing window.__ajayeb at this game, if it still does. */
export function uninstallTestHook(game: Game): void {
  if (active === game) active = null;
}

/** Records whether this game's screen is mounted; ready() needs it. */
export function markTestHookScreen(game: Game, mounted: boolean): void {
  if (mounted) screen = game;
  else if (screen === game) screen = null;
}
