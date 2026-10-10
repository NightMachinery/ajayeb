<script lang="ts">
  // The result screen once the game is over (phase gameOver): the victory
  // kind and the winner, the final score sheet, the tie-break note when the
  // totals tie, and Rematch, View final board and Home. It is a centred
  // dialog over the board; "View final board" (or Esc, or a tap outside)
  // collapses it to a small bar that brings it back.
  //
  // Everything shown is the server's result: the winner it named and the
  // score sheet it computed (lib/view/score.ts only re-shapes them).
  import House from '@lucide/svelte/icons/house';
  import RotateCw from '@lucide/svelte/icons/rotate-cw';
  import Eye from '@lucide/svelte/icons/eye';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { finalScores, outcomeOf, scoreRows, type Outcome } from '$lib/view/score';
  import type { View } from '$lib/wire';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import PeekBar from '../choices/PeekBar.svelte';
  import { asSeat, isYou, seatNames, tryGameContext, type SeatNames } from '../choices/room';
  import ScoreTable from './ScoreTable.svelte';

  interface Props {
    view: View;
    /** Asks the server for a rematch; absent when the room cannot offer one. */
    onrematch?: () => void;
    /** Leaves the room for the home screen. */
    onhome: () => void;
    /** Display names by seat; read from the room when not given. */
    names?: SeatNames;
  }

  let { view, onrematch, onhome, names: namesProp }: Props = $props();

  const ctx = tryGameContext();
  const names = $derived(namesProp ?? seatNames(ctx?.presenter.frame?.room));
  const mode = $derived(ctx?.presenter.frame?.room.mode);

  const scores = $derived(finalScores(view));
  const rows = $derived(scoreRows(scores));
  // A gameOver view always carries its result; the fallback only keeps the
  // screen whole if one ever does not (the preview's leader, no kind).
  const outcome: Outcome | null = $derived(view.result ? outcomeOf(view.result) : null);
  const winner = $derived(outcome?.winner ?? null);
  const youSeat = $derived(
    view.viewer !== undefined && isYou(view.viewer, view.viewer, mode)
      ? asSeat(view.viewer)
      : undefined
  );

  let collapsed = $state(false);

  const headline = $derived.by(() => {
    if (!outcome || outcome.kind === 'shared' || winner === null) return t('choices.over.bothWin');
    if (youSeat === winner) return t('choices.over.youWin');
    return t('choices.over.wins', { name: names[winner] });
  });

  const kindText = $derived(outcome ? t(`choices.over.kind.${outcome.kind}`) : '');

  const reason = $derived.by(() => {
    if (!outcome) return '';
    switch (outcome.kind) {
      case 'civilian':
        return t('choices.over.reason.civilian');
      case 'shared':
        return t('choices.over.reason.shared');
      case 'military':
      case 'science':
        return winner === null
          ? ''
          : t(`choices.over.reason.${outcome.kind}`, { name: names[winner] });
    }
  });

  const crest = $derived(winner === 1 ? 'crest-moon' : 'crest-sun');
  const titleId = 'game-over-title';
</script>

<div
  data-game-over
  data-winner={view.result?.winner}
  data-victory={outcome?.kind}
  data-collapsed={collapsed}
>
  <Sheet
    bind:open={() => !collapsed, (open) => (collapsed = !open)}
    centered
    {titleId}
    class="game-over"
  >
    <div class="flex flex-col gap-3 px-5 pt-4 pb-1">
      <header class="flex flex-col items-center gap-1 text-center">
        <span class="flex items-center gap-1 text-3xl" aria-hidden="true">
          {#if winner === null}
            <Icon name="crest-sun" class="text-sun" />
            <Icon name="crest-moon" class="text-moon" />
          {:else}
            <Icon name={crest} class={winner === 0 ? 'text-sun' : 'text-moon'} />
          {/if}
        </span>
        <p class="text-xs font-semibold tracking-wider text-fg-muted uppercase">{kindText}</p>
        <h2 id={titleId} class="font-display text-2xl leading-tight text-fg" dir="auto">
          {headline}
        </h2>
        {#if reason}
          <p class="text-sm text-fg-muted" dir="auto">{reason}</p>
        {/if}
      </header>

      <ScoreTable {rows} {names} {winner} you={youSeat} />

      {#if outcome?.tieBreak}
        <p
          class="rounded-control border border-rule bg-panel-sunken px-3 py-2 text-sm text-fg"
          data-tie-break
        >
          <Icon name="category-blue" class="me-1 text-card-blue" />
          {t('choices.over.tieBreak')}
        </p>
      {/if}

      <div class="grid grid-cols-2 gap-2 pb-2">
        {#if onrematch}
          <Button variant="primary" class="col-span-2" onclick={onrematch} data-rematch>
            <RotateCw class="size-4" aria-hidden="true" />
            {t('choices.over.rematch')}
          </Button>
        {/if}
        <Button
          variant="secondary"
          class="px-2 text-[14px]"
          onclick={() => (collapsed = true)}
          data-view-board
        >
          <Eye class="hidden size-4 shrink-0 sm:block" aria-hidden="true" />
          {t('choices.over.viewBoard')}
        </Button>
        <Button variant="secondary" class="px-2 text-[14px]" onclick={onhome} data-home>
          <House class="hidden size-4 shrink-0 sm:block" aria-hidden="true" />
          {t('choices.over.home')}
        </Button>
      </div>
    </div>
  </Sheet>

  {#if collapsed}
    <PeekBar
      text="{t('choices.over.bar')} · {headline}"
      openLabel={t('choices.over.results')}
      onopen={() => (collapsed = false)}
    />
  {/if}
</div>
