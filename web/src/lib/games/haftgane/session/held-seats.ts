// Where a page keeps the game seat it last held, by room code, so a former
// player who reloads still reads who plays their seat now. Only an admin's
// frame names a former seat (you.previousSeats), so for everyone else the
// page's own memory is all there is. sessionStorage keeps it for this tab
// across reloads and shares it with no other tab.

export interface HeldSeats {
  get(code: string): number | null;
  set(code: string, seat: number): void;
}

const PREFIX = 'ajayeb:hg-held:';

/** The tab's store, or null where there is no sessionStorage. */
export function sessionHeldSeats(): HeldSeats | null {
  if (typeof sessionStorage === 'undefined') return null;
  return {
    get(code) {
      try {
        const n = Number(sessionStorage.getItem(PREFIX + code) ?? NaN);
        return Number.isInteger(n) && n >= 0 ? n : null;
      } catch {
        return null;
      }
    },
    set(code, seat) {
      try {
        sessionStorage.setItem(PREFIX + code, String(seat));
      } catch {
        // Storage full or refused: the memory lasts until a reload.
      }
    }
  };
}
