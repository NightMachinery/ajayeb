<script lang="ts">
  // The game screen's first-time tips (lib/ui/tips.svelte.ts, M10.HS5): the
  // first time a hint mark appears on this device, a callout beside it says
  // what it means (FirstTip.svelte), once per tip and at most one per turn.
  // Only on the player's own turn, with nothing else open over the board
  // (the caller's `canShow`), so it never lands on the other player's turn
  // on a shared hot-seat device. A tip counts as seen once it is closed.
  // Besides the pips (M10.F3): the Law laurel, a looting token within
  // reach, the wonder hourglass, the chain badge and the guild back; and
  // the concept tips (M10.tutorial-client): the pawn off centre, the first
  // science symbol, choosing who starts the next age and a new age.
  import { t } from '$lib/i18n';
  import { openRules } from '../rules/rules-overlay.svelte';
  import { dueTip, TIPS, tips, tipSelector, tipsInView, type TipId } from '$lib/ui/tips.svelte';
  import type { HintFamily, View } from '$lib/wire';
  import FirstTip from './FirstTip.svelte';

  interface Props {
    view: View | null;
    /** The screen's own seat (the seat to move in hot-seat); null for a spectator. */
    seat: 0 | 1 | null;
    families: ReadonlySet<HintFamily>;
    /** Whether hint marks show on the board (the local hide). */
    boardMarks: boolean;
    /** Display names by seat. */
    names: readonly [string, string];
    /** The player's own turn, with no sheet, popup or pass screen open. */
    canShow: boolean;
  }

  let { view, seat, families, boardMarks, names, canShow }: Props = $props();

  // The ply a tip was last closed at: one tip per turn.
  let closedAt = $state<number | null>(null);

  const due = $derived.by((): TipId | null => {
    if (!view || seat === null || !canShow || closedAt === view.ply) return null;
    return dueTip(tipsInView(view, seat, families, boardMarks), tips.value);
  });

  function text(id: TipId): string {
    const name = seat === null ? '' : names[seat === 0 ? 1 : 0];
    switch (id) {
      case 'pair':
        return t('hints.tip.pair');
      case 'denyPair':
        return t('hints.tip.denyPair', { name });
      case 'win':
        return t('hints.tip.win');
      case 'winOpp':
        return t('hints.tip.winOpp', { name });
      case 'loot':
        return t('hints.tip.loot');
      case 'lawLaurel':
        return t('hints.tip.lawLaurel');
      case 'ghostLoot':
        return t('hints.tip.ghostLoot');
      case 'hourglass':
        return t('hints.tip.hourglass');
      case 'freeChain':
        return t('hints.tip.freeChain');
      case 'guildBack':
        return t('hints.tip.guildBack');
      case 'track':
        return t('hints.tip.track');
      case 'science':
        return t('hints.tip.science');
      case 'newAge':
        return t('hints.tip.newAge');
    }
  }

  function close(id: TipId): void {
    tips.markSeen(id);
    closedAt = view?.ply ?? null;
  }
</script>

{#if due}
  {@const id = due}
  <FirstTip
    open
    {id}
    text={text(id)}
    anchor={tipSelector(id, view ?? undefined, seat ?? 0)}
    onclose={() => close(id)}
    onmore={() => void openRules(TIPS[id].rules)}
  />
{/if}
