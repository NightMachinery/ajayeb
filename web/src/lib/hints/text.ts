// Display text for hint settings: a family's name, and the one-line
// summary of a game's settings (the lobby, the collapsed new-game section,
// the game menu). Pure display mapping over lib/hints/families.ts.

import { joinList } from '$lib/components/card/effect-text';
import { t } from '$lib/i18n';
import type { HintFamily, HintSettings } from '$lib/wire';
import { familyText, summarizeHints } from './families';

/** A family's name, in the current locale. */
export function familyName(family: HintFamily): string {
  return t(familyText(family).name);
}

/** The families' names as one list: "Prices, Threats and Advisor". */
export function familyList(families: readonly HintFamily[]): string {
  return joinList(families.map(familyName), 'conjunction');
}

/** One line that says which hints a game has: "All hints on except Advisor". */
export function summaryText(h: HintSettings): string {
  const s = summarizeHints(h);
  switch (s.kind) {
    case 'all':
      return t('hints.summary.all');
    case 'none':
      return t('hints.summary.none');
    case 'allBut':
      return t('hints.summary.allBut', { families: familyList(s.off) });
    case 'only':
      return t('hints.summary.only', { families: familyList(s.on) });
  }
}
