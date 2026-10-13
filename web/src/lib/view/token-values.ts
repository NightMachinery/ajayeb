// The token pick's values (the tokens hint family, M10.F3): under each
// token of a pending pick, what taking it would do now, as the server
// computed it (M10.HS4). A board pick reads HintsView.tokens, which holds
// both seats' values, so the decider sees its own line and the other
// seat's; a Great Library pick reads PendingView.tokenValues, which only
// the decider's view carries, for the decider alone (the offer is secret).
//
// Display only: the words come from board-hints.ts (tokenHint and
// tokenValueText, the same lines as the board token's tooltip and sheet).

import { t } from '$lib/i18n';
import type { View } from '$lib/wire';
import { tokenHint, tokenValueText } from './board-hints';
import type { HintScreen } from './hints';

/**
 * The value lines of token `slug` in an open token pick, decider first:
 * one line per seat the view has a value for, while the tokens family
 * shows. Empty without one.
 */
export function tokenPickLines(
  view: Pick<View, 'players' | 'hints' | 'pending'>,
  slug: string,
  library: boolean,
  screen: HintScreen | undefined
): string[] {
  if (!screen || !screen.families.has('tokens')) return [];
  if (!library) return tokenHint(view, slug, screen)?.lines.slice() ?? [];
  const pending = view.pending;
  if (!pending || pending.kind !== 'pickLibraryToken') return [];
  const out: string[] = [];
  for (const v of pending.tokenValues ?? []) {
    if (v.token !== slug) continue;
    const seat = v.seat === 1 ? 1 : 0;
    const value = tokenValueText(v, screen.names);
    out.push(
      screen.names.you === seat
        ? t('hints.token.forYou', { value })
        : t('hints.token.forNamed', { name: screen.names.names[seat], value })
    );
  }
  return out;
}
