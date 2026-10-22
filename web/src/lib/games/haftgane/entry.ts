import { object, snapshotKind } from '$lib/entry/bootstrap';
export { default as Component } from './Entry.svelte';
/** Only the envelope is defined in M14. M15 owns game projection validation. */
export function accepts(snapshot: unknown, code: string): boolean {
  const raw = object(snapshot),
    kind = snapshotKind(snapshot);
  return (
    (kind === 'haftgane' || kind === 'lobby') &&
    raw?.['code'] === code &&
    Number.isInteger(raw['revision']) &&
    Array.isArray(raw['seats']) &&
    Array.isArray(raw['humans']) &&
    !!object(raw['you'])
  );
}
