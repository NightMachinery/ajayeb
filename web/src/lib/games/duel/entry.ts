import { object } from '$lib/entry/bootstrap';
export { default as Component } from './Entry.svelte';
/** Discovery is discarded; Session's ordinary join obtains its own history. */
export function accepts(snapshot: unknown, code: string): boolean {
  const raw = object(snapshot),
    room = object(raw?.['room']);
  return (
    raw?.['t'] === 'state' &&
    raw['code'] === code &&
    Number.isInteger(raw['version']) &&
    !!object(raw['view']) &&
    ['online', 'hotseat', 'ai'].includes(String(room?.['mode']))
  );
}
