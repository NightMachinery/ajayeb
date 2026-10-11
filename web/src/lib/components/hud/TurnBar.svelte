<script lang="ts">
  // "What now": the TurnBar renders one TurnState (lib/view/turn.ts) as the
  // single line the UX design calls the source of truth for what the
  // player should do next. Connection wording reuses session.conn.* (the
  // session layer already owns those three messages); everything else is
  // this file's own board.turn.* keys.
  //
  // It is one slim line (40 px plus the safe area) so the board keeps its
  // height on a phone. A seat's own turn shows that seat's crest; the Retry
  // button, when it shows, keeps a 44 px tap height and the bar grows to
  // hold it.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { TurnState } from '$lib/view/turn';
  import Loader2 from '@lucide/svelte/icons/loader-2';
  import RotateCw from '@lucide/svelte/icons/rotate-cw';
  import { wonderName } from '$lib/components/card/effect-text';
  import ThinkingDots from './ThinkingDots.svelte';

  interface Props {
    state: TurnState;
    /** Display names by seat (RoomInfo, not the view), for messages that
     *  name a player. Falls back to "Seat N" when not given, e.g. in the
     *  gallery. */
    names?: readonly [string, string];
    onRetry?: () => void;
    class?: string;
  }

  let { state, names, onRetry, class: className = '' }: Props = $props();

  function nameOf(seat: 0 | 1): string {
    return names?.[seat] ?? t('debug.seat', { seat });
  }

  function pendingText(kind: TurnState & { kind: 'pendingChoice' }): string {
    switch (kind.pendingKind) {
      case 'chooseStarter':
        return t('board.turn.pending.chooseStarter');
      case 'destroyCard':
        return t('board.turn.pending.destroyCard');
      case 'buildFromDiscard':
        return t('board.turn.pending.buildFromDiscard');
      case 'pickBoardToken':
        return t('board.turn.pending.pickBoardToken');
      case 'pickLibraryToken':
        return t('board.turn.pending.pickLibraryToken');
    }
  }

  const text = $derived.by(() => {
    switch (state.kind) {
      case 'connecting':
        return t('session.conn.connecting');
      case 'reconnecting':
        return t('session.conn.reconnecting');
      case 'offline':
        return t('session.conn.offline');
      case 'sent':
        return t(state.slow ? 'board.turn.sentSlow' : 'board.turn.sent');
      case 'gameOver':
        return t('board.turn.gameOver');
      case 'spectating':
        return t('board.turn.spectating');
      case 'aiThinking':
        return t('board.turn.aiThinking', { name: nameOf(state.seat) });
      case 'waitingOpponent':
        return state.away
          ? t('board.turn.waitingAway', { name: nameOf(state.seat) })
          : t('board.turn.waitingFor', { name: nameOf(state.seat) });
      case 'pendingChoice':
        return pendingText(state);
      case 'playAgain': {
        const wonder = state.wonder ? wonderName(state.wonder) : undefined;
        if (state.hotseat) {
          const name = nameOf(state.seat);
          return wonder
            ? t('board.turn.playAgainNameWonder', { name, wonder })
            : t('board.turn.playAgainName', { name });
        }
        return wonder ? t('board.turn.playAgainWonder', { wonder }) : t('board.turn.playAgain');
      }
      case 'hotseatTurn':
        return t('turn.player', { name: nameOf(state.seat) });
      case 'yourTurn':
        return t('board.turn.yourTurn');
    }
  });

  const spinning = $derived(
    state.kind === 'sent' || state.kind === 'reconnecting' || state.kind === 'connecting'
  );
  const thinking = $derived(state.kind === 'aiThinking');
  // The seat whose own turn it is, for its crest.
  const crestSeat = $derived(
    state.kind === 'yourTurn' || state.kind === 'hotseatTurn' || state.kind === 'pendingChoice'
      ? state.seat
      : undefined
  );
  const offerRetry = $derived(state.kind === 'sent' && state.slow && !!onRetry);
</script>

<div
  class="turn-bar flex min-h-10 items-center justify-between gap-2 border-t border-rule bg-panel px-3
    py-1 {className}"
  role="status"
  aria-live="polite"
  data-turn={state.kind}
  data-away={state.kind === 'waitingOpponent' && state.away ? '' : undefined}
>
  <span class="flex min-w-0 items-center gap-2 text-sm font-semibold text-fg" data-must-fit>
    {#if spinning}
      <Loader2 class="size-4 shrink-0 animate-spin text-fg-muted" aria-hidden="true" />
    {:else if state.kind === 'playAgain'}
      <RotateCw
        class="size-4 shrink-0 {state.seat === 1 ? 'text-moon-ink' : 'text-sun-ink'}"
        aria-hidden="true"
      />
    {:else if crestSeat !== undefined}
      <Icon
        name={crestSeat === 1 ? 'crest-moon' : 'crest-sun'}
        class="shrink-0 text-base {crestSeat === 1 ? 'text-moon-ink' : 'text-sun-ink'}"
      />
    {:else if thinking}
      <ThinkingDots
        class="shrink-0 {state.kind === 'aiThinking' && state.seat === 1
          ? 'text-moon-ink'
          : 'text-sun-ink'}"
      />
    {/if}
    <span class="min-w-0" dir="auto">{text}</span>
  </span>
  {#if offerRetry}
    <button
      type="button"
      onclick={onRetry}
      class="min-h-11 rounded-control border border-rule px-3 text-sm font-semibold text-fg hover:bg-panel-sunken"
    >
      {t('board.turn.retry')}
    </button>
  {/if}
</div>
