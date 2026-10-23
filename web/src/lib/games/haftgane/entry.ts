import { isLobbyState, isTableState } from './session/frames';
export { default as Component } from './Entry.svelte';
/** A table or lobby snapshot of this room, in the shape the Session reads. */
export function accepts(snapshot: unknown, code: string): boolean {
  return (isTableState(snapshot) || isLobbyState(snapshot)) && snapshot.code === code;
}
