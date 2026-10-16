// Words for the catalog's declarative effects: one sentence per effect,
// from the i18n templates in i18n/en/card.ts. Shared by EffectGlyphs (the
// full form), WonderTile and view/describe.ts (ARIA). Display only: it
// restates printed card text and decides nothing.

import type { CatalogCost, CatalogEffect } from '$lib/catalog';
import { cardBySlug, wonderBySlug } from '$lib/catalog';
import { i18n, isFallbackText, t } from '$lib/i18n';
import type { Back, Chain, Color, Resource, ScienceSymbol } from '$lib/wire';

export const RESOURCES = [
  'wood',
  'clay',
  'stone',
  'glass',
  'papyrus'
] as const satisfies readonly Resource[];

const listFormats = new Map<string, Intl.ListFormat>();

/** Joins words for the current locale: "wood, clay and stone" or "glass or papyrus". */
export function joinList(items: readonly string[], type: 'conjunction' | 'disjunction'): string {
  // Words a partial table left in English are joined in English too.
  const locale = items.length > 0 && items.every(isFallbackText) ? 'en' : i18n.locale;
  const cacheKey = `${locale}|${type}`;
  let format = listFormats.get(cacheKey);
  if (!format) {
    format = new Intl.ListFormat(locale, { style: 'long', type });
    listFormats.set(cacheKey, format);
  }
  return format.format(items);
}

export function resourceName(resource: Resource): string {
  return t(`card.resource.${resource}`);
}

export function colorName(color: Color): string {
  return t(`card.color.${color}`);
}

export function scienceName(symbol: ScienceSymbol): string {
  return t(`card.science.${symbol}`);
}

export function chainName(chain: Chain): string {
  return t(`card.chain.${chain}`);
}

/** A building's name, or the slug itself for an unknown one. */
export function cardName(slug: string): string {
  const card = cardBySlug(slug);
  return card ? t(`card.name.${card.slug}`) : slug;
}

/** A wonder's name, or the slug itself for an unknown one. */
export function wonderName(slug: string): string {
  const wonder = wonderBySlug(slug);
  return wonder ? t(`wonder.name.${wonder.slug}`) : slug;
}

export function backName(back: Back): string {
  return t(`card.back.${back}`);
}

function resourcesOf(effect: CatalogEffect, type: 'conjunction' | 'disjunction'): string {
  return joinList((effect.resources ?? []).map(resourceName), type);
}

function colorsOf(effect: CatalogEffect): string {
  return joinList((effect.colors ?? []).map(colorName), 'conjunction');
}

/** One printed effect as a sentence, without a final full stop. */
export function effectSentence(effect: CatalogEffect): string {
  const n = effect.n ?? 1;
  const max = effect.scope === 'max';
  switch (effect.kind) {
    case 'produce':
      return t('card.effect.produce', {
        n,
        resource: effect.resource ? resourceName(effect.resource) : ''
      });
    case 'produce-choice':
      return t('card.effect.produceChoice', { resources: resourcesOf(effect, 'disjunction') });
    case 'trade-fix':
      return t('card.effect.tradeFix', { resources: resourcesOf(effect, 'conjunction') });
    case 'shields':
      return t('card.effect.shields', { count: n });
    case 'science':
      return t('card.effect.science', {
        symbol: effect.symbol ? scienceName(effect.symbol) : ''
      });
    case 'points':
      return t('card.effect.points', { count: n });
    case 'coins':
      return t('card.effect.coins', { count: n });
    case 'coins-per-card':
      return max
        ? t('card.effect.coinsPerCardMax', { count: n, colors: colorsOf(effect) })
        : t('card.effect.coinsPerCard', { count: n, colors: colorsOf(effect) });
    case 'coins-per-wonder':
      return t('card.effect.coinsPerWonder', { count: n });
    case 'points-per-card':
      return max
        ? t('card.effect.pointsPerCardMax', { count: n, colors: colorsOf(effect) })
        : t('card.effect.pointsPerCard', { count: n, colors: colorsOf(effect) });
    case 'points-per-wonder':
      return max
        ? t('card.effect.pointsPerWonderMax', { count: n })
        : t('card.effect.pointsPerWonder', { count: n });
    case 'points-per-coins':
      return t('card.effect.pointsPerCoins', { count: n });
    case 'points-per-token':
      return t('card.effect.pointsPerToken', { count: n });
    case 'play-again':
      return t('card.effect.playAgain');
    case 'destroy':
      return t('card.effect.destroy', { colors: colorsOf(effect) });
    case 'build-from-discard':
      return t('card.effect.buildFromDiscard');
    case 'draw-tokens':
      return t('card.effect.drawTokens', { count: n });
    case 'opponent-loses-coins':
      return t('card.effect.opponentLosesCoins', { count: n });
    case 'discount':
      return effect.target === 'wonder'
        ? t('card.effect.discountWonder', { count: n })
        : t('card.effect.discountBlue', { count: n });
    case 'trade-income':
      return t('card.effect.tradeIncome');
    case 'extra-shield':
      return t('card.effect.extraShield', { count: n });
    case 'wonders-play-again':
      return t('card.effect.wondersPlayAgain');
    case 'chain-bonus':
      return t('card.effect.chainBonus', { count: n });
  }
}

/** All of a card's effects as sentences, joined by full stops. */
export function effectsText(effects: readonly CatalogEffect[]): string {
  if (effects.length === 0) return t('card.effect.none');
  return effects.map(effectSentence).join('. ');
}

/** A printed cost in words: "2 coins, 1 wood and 2 stone", or "No cost". */
export function costText(cost: CatalogCost): string {
  const items: string[] = [];
  if (cost.coins > 0) items.push(t('card.cost.coins', { count: cost.coins }));
  for (const r of RESOURCES) {
    if (cost[r] > 0) items.push(t('card.cost.units', { n: cost[r], resource: resourceName(r) }));
  }
  return items.length === 0 ? t('card.cost.free') : joinList(items, 'conjunction');
}
