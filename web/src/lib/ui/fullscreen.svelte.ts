// The Fullscreen toggle of the game menus and Settings
// (M18.installable-and-fullscreen). It puts the whole document in
// fullscreen, so the game's own client-side navigation keeps it; the
// browser's own exit (Escape, the back gesture) or a full page load leaves
// it, and `active` follows through fullscreenchange.

import { createSubscriber } from 'svelte/reactivity';

/** The parts of document the rule reads. */
export interface FullscreenDocument {
  fullscreenEnabled?: boolean;
  documentElement?: { requestFullscreen?: unknown };
}

/** Whether to offer the toggle: the API exists and this page may use it.
 *  iPhone Safari has fullscreen only for video, so it offers none. */
export function fullscreenShown(doc: FullscreenDocument | undefined): boolean {
  return (
    doc?.fullscreenEnabled === true && typeof doc.documentElement?.requestFullscreen === 'function'
  );
}

const subscribe = createSubscriber((update) => {
  document.addEventListener('fullscreenchange', update);
  return () => document.removeEventListener('fullscreenchange', update);
});

export const fullscreen = {
  get shown(): boolean {
    return typeof document !== 'undefined' && fullscreenShown(document);
  },
  get active(): boolean {
    if (typeof document === 'undefined') return false;
    subscribe();
    return document.fullscreenElement != null;
  },
  /** Enters or leaves fullscreen. Call it from the tap itself: entering
   *  needs the user's gesture. A refusal leaves the page as it was. */
  async toggle(): Promise<void> {
    try {
      if (document.fullscreenElement) await document.exitFullscreen();
      else await document.documentElement.requestFullscreen({ navigationUI: 'hide' });
    } catch {
      // Refused (no gesture, a permissions policy) or already left.
    }
  }
};
