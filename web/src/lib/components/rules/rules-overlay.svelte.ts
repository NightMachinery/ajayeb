// The in-game rules overlay: its state, and openRules(), the one helper the
// rest of the app needs to show the rules without leaving the screen it is
// on. The game menu calls openRules(); a tooltip or a hint can call
// openRules('military') to open the panel at a section ("More in Rules").
//
// The overlay mounts itself. On first use, openRules() loads
// RulesOverlay.svelte (its own chunk, so the rules text is not in the game's
// first load) and mounts it once at the end of <body>. It shows a side panel
// on wide screens and a bottom sheet on phones (RulesOverlay.svelte). No
// screen has to place it, so any component can open the rules.
//
// The overlay belongs to the page it was opened on: it hides itself when the
// path changes (RulesOverlay.svelte compares `openedOn` with the current
// path), so leaving a game for Home never leaves the rules behind.

import { isRuleSectionId, type RuleSectionId } from './sections';

export class RulesOverlayState {
  /** Whether the rules are open. */
  open = $state(false);
  /** The section to show, or undefined for the top. */
  section = $state<RuleSectionId | undefined>(undefined);
  /** Counts every openRules() call, so opening the same section again
   *  scrolls back to it even though `section` did not change. */
  request = $state(0);
  /** The path the overlay was opened on. */
  openedOn = $state('');

  show(section: RuleSectionId | undefined, path: string): void {
    this.section = section;
    this.openedOn = path;
    this.request += 1;
    this.open = true;
  }

  hide(): void {
    this.open = false;
  }
}

/** The app's one rules overlay. */
export const rulesOverlay = new RulesOverlayState();

type Mounter = () => Promise<void>;

let mounted: Promise<void> | undefined;

/** Loads and mounts the overlay component once, at the end of <body>. */
const mountOverlay: Mounter = async () => {
  const [{ mount }, { default: RulesOverlay }] = await Promise.all([
    import('svelte'),
    import('./RulesOverlay.svelte')
  ]);
  const host = document.createElement('div');
  host.dataset.rulesHost = '';
  document.body.append(host);
  mount(RulesOverlay, { target: host });
};

let mounter: Mounter = mountOverlay;

/** For tests: replaces the mounting step (the overlay is not rendered). */
export function setRulesMounter(next: Mounter | undefined): void {
  mounter = next ?? mountOverlay;
  mounted = undefined;
}

/**
 * Opens the rules over the current screen, at a section when one is given
 * (an unknown id opens the top). Resolves once the overlay is mounted.
 */
export async function openRules(section?: RuleSectionId | string): Promise<void> {
  // A failed load (a lost connection) is forgotten, so the next call retries.
  mounted ??= mounter().catch((error: unknown) => {
    mounted = undefined;
    throw error;
  });
  const path = globalThis.location?.pathname ?? '';
  rulesOverlay.show(isRuleSectionId(section) ? section : undefined, path);
  await mounted;
}

/** Closes the rules overlay. */
export function closeRules(): void {
  rulesOverlay.hide();
}
