// A choice in words, from its own fields: what the pick list's buttons and
// the turn bar's pending line say. Every number is the server's (the total,
// the sale's gain, the stage); nothing is computed.

import type { Choice } from '$lib/wire/haftgane.gen';
import { ht } from '../i18n/messages.svelte';

export function cardName(slug: string): string {
  return ht(`hg.card.${slug}` as 'hg.card.forum');
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
