// The tutorial's rails (the tutorial design, section 3.4): during a guided
// turn, every move button whose key is not the guided key is disabled, with
// a short reason. A display rule only: the server still accepts any legal
// key, and a different one simply leaves the script. Pure, so it is
// unit-tested in gate.test.ts.
//
// Undo, the menus, forfeit, the sheets and the tooltips are never gated:
// only the move buttons of the card sheet, its wonder step and the draft
// read allow().

import type { TutorialState } from '$lib/wire';

/** Whether a move key may be sent now. */
export type Allow = (key: string) => boolean;

/** Every key is allowed (no tutorial, or the rails are off). */
export const ALLOW_ALL: Allow = () => true;

/**
 * The rails of a frame: false only while the status is guided, the room is
 * not paused, the turn has an action and the key differs from its key.
 */
export function tutorialAllow(tutorial: TutorialState | undefined, paused: boolean): Allow {
  const key = railsKey(tutorial, paused);
  return key === null ? ALLOW_ALL : (k) => k === key;
}

/** The one key the rails leave open now, or null while they are off. */
export function railsKey(tutorial: TutorialState | undefined, paused: boolean): string | null {
  if (!tutorial || paused || tutorial.status !== 'guided' || !tutorial.action) return null;
  return tutorial.action.key;
}
