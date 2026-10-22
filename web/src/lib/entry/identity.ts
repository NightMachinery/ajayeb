// One page identity across discovery and the unchanged Duel Session, including
// storage-blocked browsers. Bootstrap has no seq-bearing requests to restart.
import { loadIdentity, loadName, type Identity } from '$lib/session/identity';
let identity: Identity | undefined;
let submittedName = '';
/** Keep the submitted name even when localStorage writes are blocked. */
export function rememberEntryName(name: string): void {
  submittedName = name;
}
const knownDuel = new Set<string>();
let duelCode: () => string | null = () => null;
export function entryIdentity(): Identity {
  identity ??= loadIdentity();
  return { ...identity, name: loadName() || submittedName };
}
export function registerDuelCode(current: () => string | null): void {
  duelCode = current;
}
/** Only the existing live Duel Session, never a persisted hint, bypasses discovery. */
export function rememberDuelCode(code: string): void {
  knownDuel.add(code);
}
export function isDirectDuel(code: string): boolean {
  if (duelCode() === code) knownDuel.add(code);
  return knownDuel.has(code);
}
