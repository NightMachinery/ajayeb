import type { State } from '$lib/wire';

/** Move focus before an authoritative frame removes its focused structure card. */
export function moveRemovedCardFocus(next: State, prev: State | null): void {
  if (!prev || prev.code !== next.code || typeof document === 'undefined') return;
  const active = document.activeElement;
  const card = active?.closest<HTMLButtonElement>('.card-structure button[data-slot]');
  if (!card) return;
  const index = Number(card.dataset.slot);
  const oldSlot = prev.view.structure.find((slot) => slot.index === index);
  const newSlot = next.view.structure.find((slot) => slot.index === index);
  if (!oldSlot || oldSlot.state === 'empty') return;
  if (prev.view.age === next.view.age && newSlot && newSlot.state !== 'empty') return;
  // The named region survives age changes without choosing the next card.
  // Move focus while the old DOM is clean, before removal can force style.
  const board = card.closest<HTMLElement>('.game-screen[role="region"]');
  board?.focus({ preventScroll: true });
}
