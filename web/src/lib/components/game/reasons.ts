// The words for the server's Reason codes: why an action is not available.

import { t } from '$lib/i18n';
import type { CostView, Reason } from '$lib/wire';

/** Why an action is not available, from the server's reason (and its cost, for a shortfall). */
export function reasonText(reason: Reason, cost?: CostView): string {
  switch (reason) {
    case 'gameOver':
      return t('game.reason.gameOver');
    case 'wrongPhase':
      return t('game.reason.wrongPhase');
    case 'malformed':
      return t('game.reason.malformed');
    case 'notInDraft':
      return t('game.reason.notInDraft');
    case 'notAccessible':
      return t('game.reason.notAccessible');
    case 'wonderUnavailable':
      return t('game.reason.wonderUnavailable');
    case 'unaffordable':
      return cost && cost.shortfall > 0
        ? t('game.reason.unaffordable', { count: cost.shortfall })
        : t('game.reason.unaffordableNoCount');
    case 'notTarget':
      return t('game.reason.notTarget');
    case 'notInDiscard':
      return t('game.reason.notInDiscard');
    case 'tokenUnavailable':
      return t('game.reason.tokenUnavailable');
  }
}
