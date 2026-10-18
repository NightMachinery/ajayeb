// The last history entry actually shown on this device, per room and seat.
// No identity tokens or game information are stored. Hidden tabs do not
// advance this marker. Undo may lower it, so a replayed ply is seen again.
export const SEEN_MOVES_KEY = 'ajayeb:seen-moves';
export const MAX_SEEN_ROOMS = 40;
type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;
type Marker = readonly [string, number];

export function parseSeenMoves(value: unknown): Marker[] {
  if (!Array.isArray(value)) return [];
  const entries = new Map<string, number>();
  for (const row of value) {
    if (!Array.isArray(row) || row.length !== 2) continue;
    const [key, ply] = row as unknown[];
    if (typeof key !== 'string' || !/^[A-Z0-9]{1,32}:(ai|online|hotseat):[01]$/.test(key)) continue;
    if (typeof ply !== 'number' || !Number.isSafeInteger(ply) || ply < -1) continue;
    entries.delete(key);
    entries.set(key, ply);
  }
  return [...entries].slice(-MAX_SEEN_ROOMS);
}

function defaultStorage(): StorageLike | null {
  try {
    return globalThis.localStorage ?? null;
  } catch {
    return null;
  }
}

export class SeenMoves {
  #storage: StorageLike | null;
  #markers: Marker[] = [];
  constructor(storage: StorageLike | null = defaultStorage()) {
    this.#storage = storage;
    try {
      this.#markers = parseSeenMoves(JSON.parse(storage?.getItem(SEEN_MOVES_KEY) ?? 'null'));
    } catch {
      /* Bad or blocked storage starts unseen. */
    }
  }
  get(key: string): number {
    return this.#markers.find(([k]) => k === key)?.[1] ?? -1;
  }
  rewind(code: string, ply: number): void {
    for (const [key, seen] of this.#markers) {
      if (key.startsWith(`${code}:`) && seen > ply) this.remember(key, ply);
    }
  }
  remember(key: string, ply: number): void {
    if (this.get(key) === ply) return;
    this.#markers = parseSeenMoves([...this.#markers.filter(([k]) => k !== key), [key, ply]]);
    try {
      this.#storage?.setItem(SEEN_MOVES_KEY, JSON.stringify(this.#markers));
    } catch {
      /* Still holds for this page. */
    }
  }
}
