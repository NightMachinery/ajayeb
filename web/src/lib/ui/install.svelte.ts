// The Install row of Settings (M18.installable-and-fullscreen). Where the
// browser offers installation (Chrome's beforeinstallprompt) the row is an
// Install button; iOS has no prompt, so it shows the Add to Home Screen
// steps; a page that already runs installed, or a browser with neither,
// shows nothing. The event can fire before this lazy chunk loads, so the
// second inline script in app.html keeps it on window.__ajayebInstall.

/** Chrome's BeforeInstallPromptEvent, the parts we use. */
interface InstallPromptEvent extends Event {
  prompt(): Promise<unknown>;
}

/** What the Install row shows. */
export type InstallOffer = 'prompt' | 'ios' | null;

export function installOffer(installed: boolean, canPrompt: boolean, ios: boolean): InstallOffer {
  if (installed) return null;
  if (canPrompt) return 'prompt';
  return ios ? 'ios' : null;
}

/** The parts of window the rules read. */
export interface InstallWindow {
  matchMedia?: (query: string) => { matches: boolean };
  navigator: { standalone?: boolean; maxTouchPoints?: number };
}

/** Whether the page runs as an installed app: the manifest's standalone
 *  display, or an iOS home screen app (navigator.standalone). */
export function runsInstalled(win: InstallWindow): boolean {
  return (
    win.matchMedia?.('(display-mode: standalone)').matches === true ||
    win.navigator.standalone === true
  );
}

/** iOS and iPadOS: only their WebKit has navigator.standalone, and a Mac
 *  that might have it has no touch screen. */
export function isIos(win: InstallWindow): boolean {
  return 'standalone' in win.navigator && (win.navigator.maxTouchPoints ?? 0) > 0;
}

type Win = Window & InstallWindow & { __ajayebInstall?: InstallPromptEvent | null };

let deferred = $state.raw<InstallPromptEvent | null>(null);
let installed = $state(false);
let ios = false;

if (typeof window !== 'undefined') {
  const win = window as Win;
  deferred = win.__ajayebInstall ?? null;
  installed = runsInstalled(win);
  ios = isIos(win);
  win.addEventListener('beforeinstallprompt', (e) => (deferred = e as InstallPromptEvent));
  win.addEventListener('appinstalled', () => {
    deferred = null;
    installed = true;
  });
}

export const install = {
  get offer(): InstallOffer {
    return installOffer(installed, deferred !== null, ios);
  },
  /** Opens the browser's install dialog. An event prompts once only, so
   *  the button hides until the browser offers again. */
  async prompt(): Promise<void> {
    const e = deferred;
    if (!e) return;
    deferred = null;
    (window as Win).__ajayebInstall = null;
    try {
      await e.prompt();
    } catch {
      // Already used, or refused without a gesture.
    }
  }
};
