// One page identity across discovery and the unchanged Duel Session, including
// storage-blocked browsers. Bootstrap has no seq-bearing requests to restart.
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
