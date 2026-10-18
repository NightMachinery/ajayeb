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
import type { CardChainTooltipView, Chain } from '$lib/wire';
import type { CardTipContext } from '$lib/view/tips';
import { cardName, chainName, effectSentence } from '$lib/components/card/effect-text';
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
  lines: readonly string[] = [],
  context?: CardTipContext
): TooltipContent {
  const symbol = chainName(chain);
  const rulesId = 'costs';
  const detail = direction === 'out' ? context?.info?.chainOut : context?.info?.chainIn;
  const linked = direction === 'out' ? chainTarget(chain) : chainSource(chain);
  const summary = linked ? linkedCardLines(linked, detail, direction, context) : [];
  // Required public facts replace optional counting-hint status lines.
  const shown = [...summary, ...(detail ? [] : lines)];
  if (direction === 'out') {
    const target = chainTarget(chain);
    if (!target) return { text: t('card.tip.chainOutBare', { symbol }), lines, rulesId };
    return {
      text: t('card.tip.chainOut', { symbol, card: cardName(target) }),
      lines: shown,
      avoidAnchor: true,
      stack: detail
        ? undefined
        : [{ kind: 'card', slug: target, caption: t('card.tip.chainOutStack') }],
      rulesId
    };
  }
  const source = chainSource(chain);
  if (!source) return { text: t('card.chainInShort', { symbol }), lines, rulesId };
  return {
    text: t('card.tip.chainIn', { symbol, card: cardName(source) }),
    lines: shown,
    avoidAnchor: true,
    stack: detail
      ? undefined
      : [{ kind: 'card', slug: source, caption: t('card.tip.chainInStack') }],
    rulesId
  };
}

/** Printed identity/effects plus server-only public whereabouts and quotes.
 * A city card uses its actual owner, even when the other seat is viewing it.
 * The reverse link previews its source without a hypothetical build price. */
function linkedCardLines(
  slug: CardSlug,
  detail: CardChainTooltipView | undefined,
  direction: 'in' | 'out',
  context?: CardTipContext
): string[] {
  const card = cardBySlug(slug);
  if (!card) return [];
  const lines = [
    t('card.tip.card', {
      card: cardName(slug),
      category: t(`card.category.${card.color}`),
      age: t(`card.ageNumeral.${card.age}`)
    }),
    ...card.effects.slice(0, 2).map(effectSentence)
  ];
  if (!detail) return lines;
  const name = (seat: number) =>
    context?.names?.[seat === 1 ? 1 : 0] ?? t(seat === 1 ? 'board.crest.moon' : 'board.crest.sun');
  const seat = context?.linkSeat ?? context?.costSeat ?? context?.you;
  if (direction === 'out' && detail.costs && (seat === 0 || seat === 1)) {
    const cost = detail.costs[seat];
    const coins =
      cost.total === 0 ? t('card.tip.quoteFree') : t('coins.count', { count: cost.total });
    const quote = cost.affordable
      ? coins
      : t('card.tip.quoteShort', { cost: coins, count: cost.shortfall });
    lines.push(
      context?.you === seat
        ? t('card.tip.chainCostYou', { cost: quote })
        : t('card.tip.chainCostNamed', { cost: quote, name: name(seat) }),
      t(cost.needsTrade ? 'card.tip.chainTrade' : 'card.tip.chainNoTrade')
    );
  }
  switch (detail.place) {
    case 'faceUp':
      lines.push(t(detail.accessible ? 'card.tip.chainAvailable' : 'card.tip.chainCovered'));
      break;
    case 'city':
      if (detail.seat !== undefined) lines.push(t('card.tip.builtBy', { name: name(detail.seat) }));
      break;
    case 'tucked':
      if (detail.seat !== undefined)
        lines.push(t('card.tip.chainTucked', { name: name(detail.seat) }));
      break;
    case 'discard':
      lines.push(t('card.tip.chainDiscard'));
      break;
    case 'unseenThisAge':
    case 'unseenLaterAge':
      lines.push(t('card.tip.chainUnseen'));
      break;
    case 'removedKnown':
      lines.push(t('card.tip.chainAbsent'));
      break;
  }
  return lines;
}
