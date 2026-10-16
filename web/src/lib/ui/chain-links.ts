// The chain chips' tooltips (components/card/ChainChip.svelte): the
// sentence, and a stack with a readable preview of the linked card. A
// display mapper over the printed catalog, which links every chain symbol
// to exactly one card that shows it (chainOut) and one card it makes free
// (chainIn); whether a player actually builds for free is the server's cost
// breakdown, never this. The status lines, when given, are the server's
// facts about the symbol now (view/tips.ts chainLines). Chains are part of
// the rules' costs section.

import { cardBySlug, cardSlugs, type CardSlug } from '$lib/catalog';
import { t } from '$lib/i18n';
import type { Chain } from '$lib/wire';
import { cardName, chainName } from '$lib/components/card/effect-text';
import type { TooltipContent } from './tooltip.svelte';

/** The card a chain symbol makes free (the card whose cost shows it). */
export function chainTarget(chain: Chain): CardSlug | undefined {
  return cardSlugs.find((slug) => cardBySlug(slug)?.chainIn === chain);
}

/** The card that shows a chain symbol (and so makes its target free). */
export function chainSource(chain: Chain): CardSlug | undefined {
  return cardSlugs.find((slug) => cardBySlug(slug)?.chainOut === chain);
}

/**
 * The tooltip of a chain chip. "out" (on the effect band): this card shows
 * the symbol, and the stack previews the card it makes free. "in" (on the
 * cost row): this card is free for a city that shows the symbol, and the
 * stack previews the card that provides it.
 */
export function chainTooltip(
  chain: Chain,
  direction: 'in' | 'out',
  lines: readonly string[] = []
): TooltipContent {
  const symbol = chainName(chain);
  const rulesId = 'costs';
  if (direction === 'out') {
    const target = chainTarget(chain);
    if (!target) return { text: t('card.tip.chainOutBare', { symbol }), lines, rulesId };
    return {
      text: t('card.tip.chainOut', { symbol, card: cardName(target) }),
      lines,
      stack: [{ kind: 'card', slug: target, caption: t('card.tip.chainOutStack') }],
      rulesId
    };
  }
  const source = chainSource(chain);
  if (!source) return { text: t('card.chainInShort', { symbol }), lines, rulesId };
  return {
    text: t('card.tip.chainIn', { symbol, card: cardName(source) }),
    lines,
    stack: [{ kind: 'card', slug: source, caption: t('card.tip.chainInStack') }],
    rulesId
  };
}
