// The app's one live connection. The WebSocket opens the first time a page
// asks for it (Home never does), and stays open across page changes, so
// creating a room on one page and playing it on the next reuses the socket.

import { AnimRunner } from '$lib/fx/anim/runner';
import { hintDefaults } from '$lib/hints/local.svelte';
import { botDwellMs, botPickMs, motion } from '$lib/ui/motion.svelte';
import { loadIdentity } from './identity';
import { cuePrev, Presenter } from './presenter.svelte';
import { SavedRooms } from './rooms.svelte';
import { Session } from './session.svelte';
import { installTestHook } from './test-hook';
import { undoDefault } from './undo-default.svelte';
import { WsTransport } from './ws-transport';

let rooms: SavedRooms | undefined;
let live: { session: Session; presenter: Presenter } | undefined;

/**
 * Runs `make` outside every component. The $derived fields of a class
 * belong to the effect that was running when the object was made, so an
 * object made while a page starts would read stale values once that page
 * is gone (Svelte warns "derived_inert"). An effect root that is never
 * destroyed owns them instead, for the life of the app.
 */
export function detached<T>(make: () => T): T {
  const box: { made?: { value: T } } = {};
  $effect.root(() => {
    box.made = { value: make() };
  });
  // Svelte's server build (the unit tests run on it) never runs the root.
  return box.made ? box.made.value : make();
}

/** The rooms this browser has played in. */
export function savedRooms(): SavedRooms {
  rooms ??= detached(() => new SavedRooms());
  return rooms;
}

/** The live Session and its Presenter, created on first use. */
export function liveGame(): { session: Session; presenter: Presenter } {
  if (!live) {
    live = detached(() => {
      const session = new Session({
        transport: new WsTransport(),
        identity: loadIdentity(),
        rooms: savedRooms(),
        // A new game starts from the hint settings saved on this device.
        defaultHints: (mode) => hintDefaults.for(mode),
        // ...and from the "Allow undo" choice saved next to them.
        defaultAllowUndo: (mode) => undoDefault.for(mode)
      });
      // A bot's move waits out its thinking dwell, then its pick is
      // highlighted, both scaled by the animation speed (read at each move).
      // Each frame's events then animate (fx/anim): the runner reads the
      // old DOM before the swap and plays after it, at the same speed; a
      // frame with more queued behind it shows at once.
      const anim: AnimRunner = new AnimRunner({
        speed: () => motion.effective,
        // Read only while a frame shows, long after `presenter` exists.
        backlog: (): number => presenter.queued
      });
      const presenter: Presenter = new Presenter(session, {
        minThinkingMs: () => botDwellMs(motion.effective),
        preRollMs: () => botPickMs(motion.effective),
        hooks: {
          capture: (next, prev) => anim.capture(next, cuePrev(prev, next)),
          play: (_next, _prev, captured) => anim.play(captured),
          finish: () => anim.finish()
        }
      });
      return { session, presenter };
    });
    installTestHook(live);
  }
  return live;
}
