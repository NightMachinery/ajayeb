// Safe-area overrides for fixtures and screenshots.
//
// The shell pads itself with --safe-t, --safe-r, --safe-b and --safe-l,
// which default to the device's env(safe-area-inset-*) values (src/app.css).
// A headless browser has no notch, so a URL can fake one:
//
//   ?safe=47,34        top 47px, bottom 34px (a phone in portrait)
//   ?safe=0,47,21,47   top, right, bottom, left (the same phone in landscape)

export type SafeInsets = { top: number; right: number; bottom: number; left: number };

const VARS = { top: '--safe-t', right: '--safe-r', bottom: '--safe-b', left: '--safe-l' } as const;

/** Parses a ?safe= value. Returns null when it is missing or malformed. */
export function parseSafeParam(raw: string | null | undefined): SafeInsets | null {
  if (!raw) return null;
  const parts = raw.split(',').map((part) => part.trim());
  if (parts.length !== 2 && parts.length !== 4) return null;
  if (!parts.every((part) => /^\d{1,3}(\.\d+)?$/.test(part))) return null;
  const n = parts.map(Number);
  const [top = 0, a = 0, b = 0, c = 0] = n;
  return n.length === 2
    ? { top, right: 0, bottom: a, left: 0 }
    : { top, right: a, bottom: b, left: c };
}

/** Sets the four variables on an element (normally <html>), or clears them. */
export function applySafeInsets(el: HTMLElement, insets: SafeInsets | null): void {
  for (const side of ['top', 'right', 'bottom', 'left'] as const) {
    if (insets) el.style.setProperty(VARS[side], `${insets[side]}px`);
    else el.style.removeProperty(VARS[side]);
  }
}
