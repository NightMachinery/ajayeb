// Spoken descriptions (ARIA names) of cards, structure slots and wonders,
// built from the catalog's printed data and the server's view through the
// i18n templates. For example:
//
//   "Lumber Yard, brown. Produces 1 wood each turn. Available, build for 2
//    coins with trade"
//
// Display only: availability and cost come from the view (accessible,
// coveredBy, the cost breakdown), never from rules code.

import { cardBySlug, wonderBySlug } from '$lib/catalog';
import {
  backName,
  cardName,
  chainName,
  colorName,
  effectsText,
  wonderName
} from '$lib/components/card/effect-text';
import { t } from '$lib/i18n';
import type { SlotView, WonderView } from '$lib/wire';
import { affordOf, type Afford } from './afford';

/** The spoken form of an affordability badge. */
export function describeAfford(afford: Afford): string {
  switch (afford.state) {
    case 'chain':
      return afford.chainFrom
        ? t('card.afford.chain', { card: cardName(afford.chainFrom) })
        : t('card.afford.chainNoCard');
    case 'free':
      return t('card.afford.free');
    case 'own':
      return t('card.afford.own', { count: afford.coins });
    case 'trade':
      return t('card.afford.trade', { count: afford.coins });
    case 'short':
      return t('card.afford.short', { count: afford.coins, shortfall: afford.shortfall });
  }
}

/** Upper-cases the first letter, for a phrase that starts a sentence. */
function capitalize(text: string): string {
  return text.charAt(0).toLocaleUpperCase() + text.slice(1);
}

/** A building's name, colour and printed effects (and chain), as one description. */
export function describeCard(slug: string): string {
  const card = cardBySlug(slug);
  if (!card) return slug;
  const parts = [effectsText(card.effects)];
  if (card.chainIn && card.chainFrom) {
    parts.push(
      t('card.chainIn', { symbol: chainName(card.chainIn), card: cardName(card.chainFrom) })
    );
  }
  if (card.chainOut && card.chainTo) {
    parts.push(
      t('card.chainOut', { symbol: chainName(card.chainOut), card: cardName(card.chainTo) })
    );
  }
  return t('card.aria.card', {
    name: cardName(slug),
    color: colorName(card.color),
    effects: parts.join('. ')
  });
}

export interface DescribeSlotOptions {
  /** The seat whose cost to read, if any (usually the viewer). */
  seat?: number;
  /** It is not that seat's turn: the cost is for planning only. */
  waiting?: boolean;
  selected?: boolean;
}

/** The ARIA name of one structure slot. */
export function describeSlot(slot: SlotView, options: DescribeSlotOptions = {}): string {
  if (slot.state === 'empty') return t('card.aria.empty');
  if (slot.state === 'faceDown' || !slot.card) {
    return t('card.aria.faceDown', { back: backName(slot.back ?? 'age1') });
  }
  const parts = [describeCard(slot.card)];
  if (slot.accessible) {
    const cost = options.seat === undefined ? undefined : slot.costs?.[options.seat];
    const status = [t('card.aria.available')];
    if (cost) status.push(describeAfford(affordOf(cost)));
    if (options.waiting) status.push(t('card.afford.waiting'));
    parts.push(capitalize(status.join(', ')));
  } else {
    parts.push(capitalize(t('card.aria.covered', { count: slot.coveredBy.length })));
  }
  if (options.selected) parts.push(capitalize(t('card.aria.selected')));
  return parts.join('. ');
}

/** The ARIA name of a wonder, with its state when a WonderView is given. */
export function describeWonder(slug: string, state?: WonderView): string {
  const wonder = wonderBySlug(slug);
  if (!wonder) return slug;
  const parts = [
    t('wonder.aria', { name: wonderName(slug), effects: effectsText(wonder.effects) })
  ];
  if (state?.built) {
    parts.push(
      state.tucked ? t('wonder.builtWith', { card: cardName(state.tucked) }) : t('wonder.built')
    );
  } else if (state?.removed) {
    parts.push(t('wonder.removed'));
  } else if (state?.cost) {
    parts.push(capitalize(describeAfford(affordOf(state.cost))));
  }
  return parts.join('. ');
}
