// Haftgane's in-game rules overlay: its state, and openHaftganeRules(), the
// one helper a Haftgane screen needs to show the rules without leaving the
// table, shaped like Duel's openRules() (components/rules/rules-overlay.svelte.ts).
// The game menu calls openHaftganeRules(); a tooltip or a hint can call
// openHaftganeRules('trade') to open the panel at a section.
//
// The overlay mounts itself. On first use, openHaftganeRules() loads
// RulesOverlay.svelte (its own chunk) and mounts it once at the end of
// <body>. It shows a side panel on wide screens and a bottom sheet on
// phones. No screen has to place it, so any component can open the rules.
//
// The overlay belongs to the page it was opened on: it hides itself when the
// path changes, so leaving a table for Home never leaves the rules behind.
//
// Nothing outside $lib/games/haftgane imports this file, so Home and Duel
// never load it (src/platform-imports.test.ts).

import { isHgRuleSectionId, type HgRuleSectionId } from './sections';

export class HgRulesOverlayState {
  /** Whether the rules are open. */
  open = $state(false);
  /** The section to show, or undefined for the top. */
  section = $state<HgRuleSectionId | undefined>(undefined);
  /** Counts every open request, so opening the same section again scrolls
   *  back to it even though `section` did not change. */
  request = $state(0);
  /** The path the overlay was opened on. */
  openedOn = $state('');

  show(section: HgRuleSectionId | undefined, path: string): void {
    this.section = section;
    this.openedOn = path;
    this.request += 1;
    this.open = true;
  }

  hide(): void {
    this.open = false;
  }
}

/** Haftgane's one rules overlay. */
export const hgRulesOverlay = new HgRulesOverlayState();

type Mounter = () => Promise<void>;

let mounted: Promise<void> | undefined;

/** Loads and mounts the overlay component once, at the end of <body>. */
const mountOverlay: Mounter = async () => {
  const [{ mount }, { default: RulesOverlay }] = await Promise.all([
    import('svelte'),
    import('./RulesOverlay.svelte')
  ]);
  const host = document.createElement('div');
  host.dataset.rulesHost = 'haftgane';
  document.body.append(host);
  mount(RulesOverlay, { target: host });
};

let mounter: Mounter = mountOverlay;

/** For tests: replaces the mounting step (the overlay is not rendered). */
export function setHgRulesMounter(next: Mounter | undefined): void {
  mounter = next ?? mountOverlay;
  mounted = undefined;
}

/**
 * Opens Haftgane's rules over the current screen, at a section when one is
 * given (an unknown id opens the top). Resolves once the overlay is mounted.
 */
export async function openHaftganeRules(section?: HgRuleSectionId | string): Promise<void> {
  // A failed load (a lost connection) is forgotten, so the next call retries.
  mounted ??= mounter().catch((error: unknown) => {
    mounted = undefined;
    throw error;
  });
  const path = globalThis.location?.pathname ?? '';
  hgRulesOverlay.show(isHgRuleSectionId(section) ? section : undefined, path);
  await mounted;
}

/** Closes Haftgane's rules overlay. */
export function closeHaftganeRules(): void {
  hgRulesOverlay.hide();
}
