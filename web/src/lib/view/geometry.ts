// Structure geometry: turns the server's slot positions (SlotView.row, 1 at
// the top, and SlotView.col, in half-card widths, 1 at the left) into the
// shape fitStructure needs and the place of each slot in the drawn
// structure. Display only: which slot covers which, and which can be taken,
// come from the view (coveredBy, accessible).

import type { StructureShape } from '$lib/layout/fit';
import type { SlotView } from '$lib/wire';

/** A structure's shape plus where its columns and rows start. */
export interface StructureLayout extends StructureShape {
  /** The smallest col of any slot (1 for a full-width age). */
  minCol: number;
  /** The smallest row of any slot (1). */
  minRow: number;
}

/** Where one slot sits, in card units that CSS turns into px. */
export interface SlotPlace {
  /** Horizontal offset in card pitches (card width plus gap), from the left edge. */
  x: number;
  /** Vertical offset in row steps, from the top. */
  y: number;
  /** Stacking order: lower rows are drawn over the rows above them. */
  z: number;
}

/**
 * The shape of a structure: how many cards wide its widest row is and how
 * many rows it has. Slots that are already empty still count, so the
 * structure keeps its size (and every card its place) as the age goes on.
 */
export function structureLayout(slots: readonly SlotView[]): StructureLayout {
  if (slots.length === 0) return { cols: 1, rows: 1, minCol: 1, minRow: 1 };
  let minCol = Infinity;
  let maxCol = -Infinity;
  let minRow = Infinity;
  let maxRow = -Infinity;
  for (const s of slots) {
    minCol = Math.min(minCol, s.col);
    maxCol = Math.max(maxCol, s.col);
    minRow = Math.min(minRow, s.row);
    maxRow = Math.max(maxRow, s.row);
  }
  return {
    // Two half-card columns per card: cols 1, 3, 5 ... 11 are six cards.
    cols: Math.floor((maxCol - minCol) / 2) + 1,
    rows: maxRow - minRow + 1,
    minCol,
    minRow
  };
}

/** Where a slot is drawn inside its structure. */
export function slotPlace(slot: SlotView, layout: StructureLayout): SlotPlace {
  return {
    x: (slot.col - layout.minCol) / 2,
    y: slot.row - layout.minRow,
    z: slot.row - layout.minRow + 1
  };
}

/**
 * The slot's box in px for a given card size, the same arithmetic as the
 * CSS in StructureSlot. Used by tests and anything that needs a rectangle
 * without the DOM.
 */
export function slotRect(
  slot: SlotView,
  layout: StructureLayout,
  fit: { w: number; h: number; step: number; gap: number }
): { x: number; y: number; w: number; h: number } {
  const place = slotPlace(slot, layout);
  return {
    x: place.x * (fit.w + fit.gap),
    y: place.y * fit.step * fit.h,
    w: fit.w,
    h: fit.h
  };
}

/** The slots drawn over this one (the lower-row slots that overlap it), by index. */
export function overlappingSlots(slot: SlotView, slots: readonly SlotView[]): number[] {
  return slots
    .filter((s) => s.row === slot.row + 1 && Math.abs(s.col - slot.col) === 1)
    .map((s) => s.index);
}
