// The FX layer: the one element flying clones, flips and coin floats are
// drawn in (components/shell/FxLayer.svelte registers it while mounted).
// A module-level slot rather than a context, because the runner lives in
// the app's one Presenter, outside every component.
//
// Beside it, the top part: a manual popover the runner shows only while it
// draws over a dialog (the Great Library's tokens dealt into the drawer's
// choice, the victory banner over the result screen), because a dialog
// sits in the top layer, above everything else on the page.

let current: HTMLElement | null = null;
let top: HTMLElement | null = null;

/** The FX layer on the page, or null when none is mounted. */
export function fxLayer(): HTMLElement | null {
  return current;
}

/** The FX layer's top-layer part, or null when none is mounted. */
export function fxTop(): HTMLElement | null {
  return top;
}

/** Registers `el` as the FX layer; the returned function unregisters it. */
export function registerFxLayer(el: HTMLElement): () => void {
  current = el;
  return () => {
    if (current === el) current = null;
  };
}

/** Registers `el` as the FX layer's top-layer part. */
export function registerFxTop(el: HTMLElement): () => void {
  top = el;
  return () => {
    if (top === el) top = null;
  };
}
