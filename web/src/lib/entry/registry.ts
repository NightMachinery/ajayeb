import type { snapshotKind } from './bootstrap';
// A closed table also keeps each import in its own function. Rolldown folds
// conditional return branches before Vite computes CSS preloads; a conditional
// import expression can otherwise preload only its last arm's dependencies.
const entries = Object.freeze({
  duel: () => import('$lib/games/duel/entry'),
  haftgane: () => import('$lib/games/haftgane/entry'),
  lobby: () => import('$lib/games/haftgane/entry')
});
export function loadEntry(kind: NonNullable<ReturnType<typeof snapshotKind>>) {
  return entries[kind]();
}
