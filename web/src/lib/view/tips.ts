// Tooltip texts for the board's micro-elements (M10.tooltip-coverage): the
// pips of a printed cost, the effect glyphs, the affordability badge, a
// face-down back, the coin and point tallies, the chain chips' status lines
// and an owned guild's points. Pure display mappers: they word printed
// catalog data and the numbers the server sent (CostView, PlayerHintsView,
// HintsView), and decide nothing. The components draw the result through
// ui/tooltip.svelte.ts; a concept tooltip names its rules section, which the
// bubble turns into a "More in Rules" link.
//
// The hint lines follow the families this screen shows (HintScreen): the
// chain status lines and the face-down candidates are the counting family,
// the next point from coins, the winner on points now and a guild's points
// the scoring family.

import type { CatalogEffect } from '$lib/catalog';
import { cardBySlug } from '$lib/catalog';
import { cardName, effectSentence, joinList, resourceName } from '$lib/components/card/effect-text';
import { tokenName } from '$lib/components/game/names';
import type { RuleSectionId } from '$lib/components/rules/sections';
import { sentences, t } from '$lib/i18n';
export { sentences } from '$lib/i18n';
import type {
  Back,
  Chain,
  CostView,
  GuildPointsView,
  HintsView,
  PlayerHintsView,
  PreviewWinnerView,
  Resource
} from '$lib/wire';
import type { Afford } from './afford';
import { placeText } from './board-hints';
import type { HintNames, HintScreen, Seat } from './hints';

/** A tooltip's words: the text, status lines, and its rules section. */
export interface TipText {
  text: string;
  lines?: readonly string[];
  rulesId?: RuleSectionId;
}

function asSeat(n: number | undefined): Seat {
  return n === 1 ? 1 : 0;
}

// --- A printed cost ----------------------------------------------------------

/** How a payment covers one resource of a cost: made, from a flexible
 *  producer, waived by a token, or bought. Empty without a breakdown. */
export function payLines(resource: Resource, cost: CostView | undefined): string[] {
  if (!cost || cost.chainFree) return [];
  const lines: string[] = [];
  const own = cost.own[resource];
  if (own > 0) lines.push(t('card.tip.payOwn', { n: own }));
  const flex = cost.producers.filter((p) => p.resource === resource).length;
  if (flex > 0) lines.push(t('card.tip.payFlex', { n: flex }));
  const waived = cost.discount?.units[resource] ?? 0;
  if (waived > 0 && cost.discount) {
    lines.push(t('card.tip.payDiscount', { n: waived, token: tokenName(cost.discount.token) }));
  }
  const bought = cost.trade.find((tr) => tr.resource === resource);
  if (bought && bought.units > 0) {
    lines.push(t('card.tip.payTrade', { n: bought.units, count: bought.unitPrice }));
  }
  return lines;
}

/** The tooltip of one pip group of a printed cost: coins, or a resource and
 *  how the payment in view covers it. */
export function costPipTip(
  group: { kind: 'coins'; n: number } | { kind: 'res'; resource: Resource; n: number },
  cost?: CostView
): TipText {
  if (group.kind === 'coins') {
    return { text: t('card.tip.costCoins', { count: group.n }) };
  }
  return {
    text: t('card.tip.costUnits', { n: group.n, resource: resourceName(group.resource) }),
    lines: payLines(group.resource, cost)
  };
}

/** The "+n" of a cost row too narrow for all of it: what it hides. */
export function costMoreTip(
  hidden: readonly ({ kind: 'coins'; n: number } | { kind: 'res'; resource: Resource; n: number })[]
): TipText {
  const items = hidden.map((g) =>
    g.kind === 'coins'
      ? t('card.cost.coins', { count: g.n })
      : t('card.cost.units', { n: g.n, resource: resourceName(g.resource) })
  );
  return { text: t('card.tip.costMore', { items: joinList(items, 'conjunction') }) };
}

// --- Effects -----------------------------------------------------------------

/** The rules section an effect's concept belongs to, for its "More in
 *  Rules" link; none for an effect its sentence says in full (production,
 *  coins, points). */
export function effectRules(effect: CatalogEffect): RuleSectionId | undefined {
  switch (effect.kind) {
    case 'trade-fix':
    case 'trade-income':
      return 'costs';
    case 'shields':
    case 'extra-shield':
      return 'military';
    case 'science':
      return 'science';
    case 'coins-per-card':
    case 'points-per-card':
    case 'points-per-coins':
    case 'points-per-wonder':
      return effect.scope === 'max' ? 'guilds' : 'scoring';
    case 'points-per-token':
      return 'scoring';
    case 'play-again':
    case 'wonders-play-again':
    case 'destroy':
    case 'build-from-discard':
    case 'draw-tokens':
    case 'opponent-loses-coins':
      return 'wonders';
    case 'discount':
      return effect.target === 'wonder' ? 'wonders' : 'tokens';
    case 'chain-bonus':
      return 'tokens';
    case 'produce':
    case 'produce-choice':
    case 'points':
    case 'coins':
    case 'coins-per-wonder':
      return undefined;
  }
}

/** An effect glyph's tooltip: its sentence (and the guild's paired points
 *  sentence when the glyph draws both), and its rules section. */
export function effectTip(effect: CatalogEffect, points?: number): TipText {
  const parts = [effectSentence(effect)];
  if (points !== undefined && effect.kind === 'coins-per-card') {
    parts.push(effectSentence({ ...effect, kind: 'points-per-card', n: points } as CatalogEffect));
  }
  return { text: sentences(parts), rulesId: effectRules(effect) };
}

// --- The affordability badge -------------------------------------------------

/** The badge's tooltip: why the card costs what it costs. With the cost
 *  breakdown, a trade says what it buys and at what price. The chain,
 *  trade and short states link the rules' costs section. */
export function affordTip(afford: Afford, cost?: CostView): TipText {
  switch (afford.state) {
    case 'chain':
      return {
        text: afford.chainFrom
          ? t('card.tip.badgeChain', { card: cardName(afford.chainFrom) })
          : t('card.tip.badgeChainBare'),
        rulesId: 'costs'
      };
    case 'free':
      return { text: t('card.tip.badgeFree') };
    case 'own':
      return { text: t('card.tip.badgeOwn', { count: afford.coins }) };
    case 'trade':
      return {
        text: t('card.tip.badgeTrade', { count: afford.coins }),
        lines: (cost?.trade ?? []).map((tr) =>
          t('card.tip.tradeLine', {
            n: tr.units,
            resource: resourceName(tr.resource),
            count: tr.unitPrice
          })
        ),
        rulesId: 'costs'
      };
    case 'short':
      return {
        text: t('card.tip.badgeShort', { coins: afford.coins, count: afford.shortfall }),
        rulesId: 'costs'
      };
  }
}

// --- A face-down back --------------------------------------------------------

/** The back's own words: which age, or a guild. */
export function backText(back: Back): string {
  return back === 'guild'
    ? t('card.tip.backGuild')
    : t('card.tip.backAge', { back: t(`card.back.${back}`) });
}

function guildValue(g: HintsView['guildPreview'][number], names: HintNames): string {
  const part = (seat: Seat): string => {
    const vp = g.vp[seat];
    return names.you === seat
      ? t('hints.unseen.guildYou', { count: vp })
      : t('hints.unseen.guildNamed', { count: vp, name: names.names[seat] });
  };
  return t('hints.unseen.guild', { card: cardName(g.guild), first: part(0), second: part(1) });
}

/**
 * A face-down back's tooltip: which back it is, and with the counting
 * family the cards that may lie under it (the server's unseen pool) and, in
 * Age III, what each guild among them would be worth to each seat now.
 */
export function backTip(back: Back, hints: HintsView | undefined, screen?: HintScreen): TipText {
  const text = backText(back);
  if (!hints || !screen?.families.has('counting'))
    return { text, rulesId: back === 'guild' ? 'guilds' : 'turn' };
  const pool = hints.unseen.find((u) => u.back === back);
  const lines: string[] = [];
  if (pool && pool.cards.length > 0) {
    lines.push(
      t('hints.unseen.pool', {
        count: pool.cards.length,
        list: joinList(pool.cards.map(cardName), 'conjunction')
      })
    );
    if (pool.setAside > 0) lines.push(t('hints.unseen.setAside', { count: pool.setAside }));
    if (back === 'guild') {
      for (const g of hints.guildPreview) {
        if (pool.cards.includes(g.guild)) lines.push(guildValue(g, screen.names));
      }
    }
  }
  return { text, lines, rulesId: back === 'guild' ? 'guilds' : 'turn' };
}

// --- The tallies -------------------------------------------------------------

/** The coin tally's status line: the coins still missing for the next point
 *  from coins (scoring). */
export function coinsLines(ph: PlayerHintsView | undefined, screen?: HintScreen): string[] {
  if (!ph || !screen?.families.has('scoring') || ph.coinsToNextVp <= 0) return [];
  return [t('hints.tally.nextVp', { count: ph.coinsToNextVp })];
}

/** Who would win on points now, in words. */
export function previewText(p: PreviewWinnerView, names: HintNames): string {
  if (p.winner === undefined || p.by === 'shared') return t('hints.preview.shared');
  const seat = asSeat(p.winner);
  const you = names.you === seat;
  if (p.by === 'blue') {
    return you
      ? t('hints.preview.blueYou')
      : t('hints.preview.blueNamed', { name: names.names[seat] });
  }
  return you
    ? t('hints.preview.leadYou', { count: p.margin })
    : t('hints.preview.leadNamed', { count: p.margin, name: names.names[seat] });
}

/** The points tally's status line: who would win on points now (scoring). */
export function scoreLines(hints: HintsView | undefined, screen?: HintScreen): string[] {
  const p = hints?.previewWinner;
  if (!p || !screen?.families.has('scoring')) return [];
  return [previewText(p, screen.names)];
}

// --- Chains and guilds -------------------------------------------------------

/**
 * The chain chips' status lines (counting): for each seat whose city shows
 * the symbol, the card it makes free for that seat and where that card is,
 * and what the chain would save today while the card may still come.
 */
export function chainLines(
  chain: Chain,
  players: readonly { hints?: PlayerHintsView }[] | undefined,
  screen?: HintScreen
): string[] {
  if (!players || !screen?.families.has('counting')) return [];
  const lines: string[] = [];
  players.forEach((player, i) => {
    const seat = asSeat(i);
    for (const ct of player.hints?.chainTargets ?? []) {
      if (ct.chain !== chain) continue;
      const card = cardName(ct.target);
      const place = placeText(ct.target, ct, screen.names);
      lines.push(
        screen.names.you === seat
          ? t('hints.chain.statusYou', { card, place })
          : t('hints.chain.statusNamed', { card, place, name: screen.names.names[seat] })
      );
      if (ct.worth !== undefined && ct.worth > 0) {
        lines.push(t('hints.chain.saves', { count: ct.worth }));
      }
    }
  });
  return lines;
}

/** An owned guild's points now (scoring), one line per guild. */
export function guildLines(
  guilds: readonly GuildPointsView[] | undefined,
  screen?: HintScreen
): string[] {
  if (!guilds || !screen?.families.has('scoring')) return [];
  return guilds.map((g) => {
    const seat = asSeat(g.countsSeat);
    const city =
      screen.names.you === seat
        ? t('hints.guild.cityYou')
        : t('hints.guild.cityNamed', { name: screen.names.names[seat] });
    return t('hints.guild.points', { card: cardName(g.card), count: g.vp, city });
  });
}

// --- Small words -------------------------------------------------------------

/** A card's name and its category, for a card too small to print its name. */
export function cardTip(slug: string): string {
  const card = cardBySlug(slug);
  if (!card) return slug;
  return t('card.tip.card', { card: cardName(slug), category: t(`card.category.${card.color}`) });
}
