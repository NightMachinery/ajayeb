// A choice in words, from its own fields: what the pick list's buttons and
// the turn bar's pending line say. Every number is the server's (the total,
// the sale's gain, the stage); nothing is computed.

import type { Choice, Play } from '$lib/wire/haftgane.gen';
import { ht } from '../i18n/messages.svelte';

export function cardName(slug: string): string {
  return ht(`hg.card.${slug}` as 'hg.card.forum');
}

/** A seat's last play in words ("built Baths", "built wonder stage II",
 *  "sold a card"), or "" for none. */
export function playText(p: Play | undefined): string {
  if (!p) return '';
  switch (p.kind) {
    case 'build':
      return ht('hg.city.last.build', { card: cardName(p.design) });
    case 'build-discarded':
      return ht('hg.city.last.build-discarded', { card: cardName(p.design) });
    case 'stage':
      return ht('hg.city.last.stage', {
        stage: ht(
          `hg.screen.stageNumeral.${Math.min(Math.max(p.stage, 1), 4)}` as 'hg.screen.stageNumeral.1'
        )
      });
    case 'sell':
      return ht('hg.city.last.sell');
  }
  return '';
}

/** The action alone, as a button under the card's own name says it. */
export function choiceAction(c: Choice): string {
  switch (c.kind) {
    case 'build':
    case 'build-discarded':
      if (c.free !== '') return ht('hg.screen.pick.buildFree');
      return c.total > 0
        ? ht('hg.screen.pick.buildFor', { count: c.total })
        : ht('hg.screen.pick.build');
    case 'stage':
      return c.total > 0
        ? ht('hg.screen.pick.stageFor', { stage: c.stage, count: c.total })
        : ht('hg.screen.pick.stage', { stage: c.stage });
    case 'sell':
      return ht('hg.screen.pick.sell', { count: c.gain });
    case 'side':
      return ht('hg.screen.pick.side', {
        side: c.side === 'night' ? ht('hg.screen.side.night') : ht('hg.screen.side.day')
      });
  }
  return c.kind;
}

/** The action with its card's name, where the card is not otherwise shown. */
export function choiceLabel(c: Choice): string {
  const action = choiceAction(c);
  return c.design && c.kind !== 'side'
    ? ht('hg.screen.pick.named', { card: cardName(c.design), action })
    : action;
}

/** What the trade costs on each side, or nothing when it buys nothing. */
export function choiceSplit(c: Choice): string {
  return c.left + c.right > 0 ? ht('hg.screen.pick.split', { left: c.left, right: c.right }) : '';
}
