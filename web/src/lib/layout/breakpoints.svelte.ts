// Viewport classes as MediaQuery runes (UX design, "Viewport classes"). They
// are chosen by the viewport's size and pointer, never by user agent:
//
//   phone-portrait    360×740, 390×844, 430×932 (stacked board, bottom sheet)
//   phone-landscape   844×390, 740×360 (three columns, short)
//   tablet            768×1024 (stacked board with city panels)
//   desktop           1024 px and wider (three columns, popovers)
//
// The queries match the `short:` and `split:` variants in app.css, so CSS
// and script agree on the class. Each `.current` is reactive; read it in a
// $derived or in markup. MediaQuery answers false before hydration (the app
// renders on the client only, so that never shows).

import { MediaQuery } from 'svelte/reactivity';

export type ViewportClass = 'phone-portrait' | 'phone-landscape' | 'tablet' | 'desktop';

/** The query strings, shared with tests and with code that builds its own MediaQuery. */
export const QUERIES = {
  /** A phone held sideways: little height (the `short:` variant). */
  phoneLandscape: '(orientation: landscape) and (max-height: 32rem)',
  /** Room for side columns (the `split:` variant minus phone landscape). */
  wide: '(min-width: 64rem)',
  /** Wider than a phone in portrait. */
  tablet: '(min-width: 40rem)',
  /** A mouse or trackpad on a wide screen: popovers instead of sheets. */
  finePointerWide: '(pointer: fine) and (min-width: 64rem)',
  /** Too little height for the stacked board's full rows, such as a phone
   *  sharing its screen with a video call: the stacked layouts tighten
   *  (M18.short-screen-status). */
  shortStack: '(max-height: 40rem)'
} as const;

class Breakpoints {
  #phoneLandscape = new MediaQuery(QUERIES.phoneLandscape, false);
  #wide = new MediaQuery(QUERIES.wide, false);
  #tablet = new MediaQuery(QUERIES.tablet, false);
  #finePointerWide = new MediaQuery(QUERIES.finePointerWide, false);
  #shortStack = new MediaQuery(QUERIES.shortStack, false);

  get phoneLandscape(): boolean {
    return this.#phoneLandscape.current;
  }

  get finePointerWide(): boolean {
    return this.#finePointerWide.current;
  }

  /** A stacked layout (phone portrait or tablet) with too little height for
   *  its full rows (QUERIES.shortStack). */
  get shortStack(): boolean {
    const v = this.viewport;
    return this.#shortStack.current && (v === 'phone-portrait' || v === 'tablet');
  }

  /** The viewport class right now. */
  get viewport(): ViewportClass {
    return viewportClass({
      phoneLandscape: this.#phoneLandscape.current,
      wide: this.#wide.current,
      tablet: this.#tablet.current
    });
  }

  get phonePortrait(): boolean {
    return this.viewport === 'phone-portrait';
  }

  get desktop(): boolean {
    return this.viewport === 'desktop';
  }

  /** The gap between structure cards for this viewport (4 / 6 / 8 px). */
  get structureGap(): number {
    return structureGap(this.viewport);
  }
}

/** The viewport class from the answers to the three size queries. */
export function viewportClass(matches: {
  phoneLandscape: boolean;
  wide: boolean;
  tablet: boolean;
}): ViewportClass {
  if (matches.phoneLandscape) return 'phone-landscape';
  if (matches.wide) return 'desktop';
  if (matches.tablet) return 'tablet';
  return 'phone-portrait';
}

/** The gap between structure cards: 4 px on phones, 6 on tablets, 8 on desktops. */
export function structureGap(viewport: ViewportClass): number {
  if (viewport === 'desktop') return 8;
  if (viewport === 'tablet') return 6;
  return 4;
}

/** The app's one set of breakpoint runes. */
export const breakpoints = new Breakpoints();
