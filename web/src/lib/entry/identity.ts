// One page identity across discovery and the unchanged Duel Session, including
// storage-blocked browsers. Bootstrap has no seq-bearing requests to restart;
// the Haftgane entry's sessions keep their seq counters here, beside the id.
import { loadIdentity, loadName, type Identity } from '$lib/session/identity';
let identity: Identity | undefined;
let submittedName = '';
/** Keep the submitted name even when localStorage writes are blocked. */
export function rememberEntryName(name: string): void {
  submittedName = name;
}
const games = new Map<string, 'duel' | 'haftgane' | 'lobby' | 'other'>();
export function rememberEntryGame(
  code: string,
  game: 'duel' | 'haftgane' | 'lobby' | 'other'
): void {
  games.set(code, game);
}
export function knownEntryGame(code: string) {
  return games.get(code);
}
export function entryIdentity(): Identity {
  identity ??= loadIdentity();
  return { ...identity, name: loadName() || submittedName };
}

/**
 * The last seq sent under each client id. A client id and its seq counter
 * must live exactly as long as each other (docs/protocol.md, "Idempotency"):
 * a counter that started over under the same id would make the room take
 * new requests for old ones.
 */
export class SeqCounters {
  #last = new Map<string, number>();
  /** The next seq to send under clientId. */
  next(clientId: string): number {
    return (this.#last.get(clientId) ?? 0) + 1;
  }
  /** Records that a request under clientId carried seq. */
  sent(clientId: string, seq: number): void {
    if (seq > (this.#last.get(clientId) ?? 0)) this.#last.set(clientId, seq);
  }
}

const pageSeqs = new SeqCounters();
/**
 * This page's counters, which live as long as its client id, so a game
 * session mounted again in the same page (Home and back) goes on from the
 * last seq any earlier one sent. A reload makes a new id that starts at 1.
 */
export function entrySeqs(): SeqCounters {
  return pageSeqs;
}
