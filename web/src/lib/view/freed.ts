// Which cards taking one card would uncover (the hints design, frees-preview
// and gift-to-opponent): a display mapper over the structure the server
// sent. A slot becomes accessible once the last card covering it is taken,
// so taking slot i frees exactly the slots whose coveredBy is [i]. No rule
// beyond that reading: whether the freed card is face up, and what it costs
// each seat, are the server's fields on that slot.

import type { SlotView } from '$lib/wire';

/** The slots that taking the card in slot `index` would make accessible, in layout order. */
export function freedBy(index: number, structure: readonly SlotView[]): SlotView[] {
  return structure.filter(
    (s) => s.state !== 'empty' && s.coveredBy.length === 1 && s.coveredBy[0] === index
  );
}

/** What taking one card uncovers: the face-up cards by slug, and how many face-down cards. */
export interface Freed {
  /** Face-up cards that become takeable, in layout order. */
  readonly faceUp: readonly SlotView[];
  /** Face-down cards that the end of the turn will turn over. */
  readonly faceDown: number;
}

/** The freed slots of `index`, split into face-up cards and a face-down count. */
export function freedSplit(index: number, structure: readonly SlotView[]): Freed {
  const freed = freedBy(index, structure);
  return {
    faceUp: freed.filter((s) => s.state === 'faceUp' && s.card !== undefined),
    faceDown: freed.filter((s) => s.state === 'faceDown').length
  };
}
