// The FX layer: the one element flying clones, flips and coin floats are
// drawn in (components/shell/FxLayer.svelte registers it while mounted).
// A module-level slot rather than a context, because the runner lives in
// the app's one Presenter, outside every component.

let current: HTMLElement | null = null;

/** The FX layer on the page, or null when none is mounted. */
export function fxLayer(): HTMLElement | null {
  return current;
}

/** Registers `el` as the FX layer; the returned function unregisters it. */
export function registerFxLayer(el: HTMLElement): () => void {
  current = el;
  return () => {
    if (current === el) current = null;
  };
}
