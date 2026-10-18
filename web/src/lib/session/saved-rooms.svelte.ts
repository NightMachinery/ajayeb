// Home and the live game share this small entry point, one store and one
// storage listener. Importing it does not load the session, presenter or FX.
import { ROOMS_KEY, SavedRooms } from './rooms.svelte';

let rooms: SavedRooms | undefined;

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
  if (!rooms) {
    rooms = detached(() => new SavedRooms());
    // localStorage's storage event reaches the other tabs, including Home
    // tabs with no socket. No connection or polling is needed there.
    if (typeof window !== 'undefined') {
      window.addEventListener('storage', (event) => {
        if (
          event.storageArea === window.localStorage &&
          (event.key === ROOMS_KEY || event.key === null)
        ) {
          rooms?.refresh(event.key === null ? null : event.newValue);
        }
      });
    }
  }
  return rooms;
}
