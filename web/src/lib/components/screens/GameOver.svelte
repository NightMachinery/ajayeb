<script lang="ts">
  // The result screen once the game is over (phase gameOver): the victory
  // kind and the winner, the final score sheet, the tie-break note when the
  // totals tie, and Rematch, View final board and Home. It is a centred
  // dialog over the board; "View final board" (or Esc, or a tap outside)
  // collapses it to a small bar that brings it back.
  //
  // Everything shown is the server's result: the winner it named and the
  // score sheet it computed (lib/view/score.ts only re-shapes them).
  //
  // Above the headline, a scene of how the game ended (victoryArtId: the
  // military, science or civilian painting; a draw shows its own scene once
  // one is approved, and the civilian one until then), framed in paper like
  // the wonder tiles so it feathers the same way in both themes, with the
  // winner's crest on its lower edge. It replaces the crest row and sits
  // beside the headline, so the score table and the buttons keep their room.
  //
  // The move review (the advisor family, M10.F3, view/end-review.ts): when
  // the viewer's own move ended the game, their review of it shows under
  // the score sheet (when the opponent's move ended it, that move's summary
  // carries the review instead).
  import House from '@lucide/svelte/icons/house';
  import RotateCw from '@lucide/svelte/icons/rotate-cw';
  import Eye from '@lucide/svelte/icons/eye';
  import ArtImage from '$lib/art/ArtImage.svelte';
  import { victoryArtId } from '$lib/art/lookup';
  import { visibleFamilies } from '$lib/hints/families';
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { endReview } from '$lib/view/end-review';
  import { reviewLines, type ReviewLines } from '$lib/view/review';
  import { finalScores, outcomeOf, scoreRows, type Outcome } from '$lib/view/score';
  import type { View } from '$lib/wire';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import PeekBar from '../choices/PeekBar.svelte';
  import { asSeat, isYou, seatNames, tryGameContext, type SeatNames } from '../choices/room';
  import ScoreTable from './ScoreTable.svelte';
  import MoveReview from '../game/MoveReview.svelte';

  interface Props {
    view: View;
    /** Asks the server for a rematch; absent when the room cannot offer one. */
    onrematch?: () => void;
    /** The rematch button's label, when it is not a plain Rematch (joining
     *  the one the other player asked for, or watching it). */
    rematchLabel?: string;
    /** A line above the button, such as who asked for the rematch. */
    rematchNote?: string;
    /** Leaves the room for the home screen. */
    onhome: () => void;
    /** Display names by seat; read from the room when not given. */
    names?: SeatNames;
    /** The viewer's review of the move that ended the game; else from the
     *  view and the page's game context. */
    review?: ReviewLines;
  }

  let {
    view,
    onrematch,
    rematchLabel,
    rematchNote,
    onhome,
    names: namesProp,
    review: reviewProp
  }: Props = $props();

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
      case 'forfeit':
        // The loser is named: the seat that is not the winner.
        return winner === null
          ? ''
          : t('choices.over.reason.forfeit', { name: names[winner === 0 ? 1 : 0] });
    }
  });

  const review = $derived.by((): ReviewLines | null => {
    if (reviewProp) return reviewProp;
    const session = ctx?.session;
    if (!session || !visibleFamilies(session.hints, hintHide.value).has('advisor')) return null;
    const own = endReview(view.review, view.viewer, session.history, outcome?.kind);
    return own && view.viewer !== undefined
      ? reviewLines(own, { names, seat: asSeat(view.viewer) })
      : null;
  });

  const crest = $derived(winner === 1 ? 'crest-moon' : 'crest-sun');
  const scene = $derived(outcome ? victoryArtId(outcome.kind) : null);
  const titleId = 'game-over-title';
</script>

{#snippet crests()}
  {#if winner === null}
    <Icon name="crest-sun" class="text-sun-ink" />
    <Icon name="crest-moon" class="text-moon-ink" />
  {:else}
    <Icon name={crest} class={winner === 0 ? 'text-sun-ink' : 'text-moon-ink'} />
  {/if}
{/snippet}

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
      <header
        class={[
          'flex',
          scene
            ? 'flex-row items-center gap-4 text-start'
            : 'flex-col items-center gap-1 text-center'
        ]}
      >
        {#if scene}
          <!-- The scene takes the crest row's place, beside the headline,
               so the dialog grows by no more than a wrapped line: the score
               table and the buttons keep their room in the sheet's height
               (640 px at most) on a phone as on a desktop. The crests sit on
               its lower edge, like a seal. -->
          <div class="relative mb-3.5 shrink-0">
            <div
              class="victory-scene relative aspect-video w-28 overflow-hidden rounded-[10px] border-[1.5px] border-(color:--gold-deep) bg-card-paper sm:w-40"
              data-victory-scene={scene}
            >
              <ArtImage id={scene} feather="soft" class="absolute inset-0 h-full w-full">
                {#snippet fallback()}
                  <span class="victory-placeholder absolute inset-0"></span>
                {/snippet}
              </ArtImage>
            </div>
            <span class="absolute inset-x-0 -bottom-3.5 flex justify-center" aria-hidden="true">
              <span
                class="flex items-center gap-0.5 rounded-full border border-(color:--gold-deep) bg-panel px-1 py-0.5 text-2xl leading-none"
              >
                {@render crests()}
              </span>
            </span>
          </div>
        {:else}
          <span class="flex items-center gap-1 text-3xl" aria-hidden="true">
            {@render crests()}
          </span>
        {/if}
        <div class="flex min-w-0 flex-col gap-1">
          <p class="text-xs font-semibold tracking-wider text-fg-muted uppercase">{kindText}</p>
          <h2 id={titleId} class="font-display text-2xl leading-tight text-fg" dir="auto">
            {headline}
          </h2>
          {#if reason}
            <p class="text-sm text-fg-muted" dir="auto">{reason}</p>
          {/if}
        </div>
      </header>

      <ScoreTable {rows} {names} {winner} you={youSeat} />

      {#if review}
        <div data-end-review><MoveReview lines={review} /></div>
      {/if}

      {#if outcome?.tieBreak}
        <p
          class="rounded-control border border-rule bg-panel-sunken px-3 py-2 text-sm text-fg"
          data-tie-break
        >
          <Icon name="category-blue" class="me-1 text-card-blue-ink" />
          {t('choices.over.tieBreak')}
        </p>
      {/if}

      <div class="grid grid-cols-2 gap-2 pb-2">
        {#if onrematch}
          {#if rematchNote}
            <p
              class="col-span-2 text-center text-sm font-semibold text-fg"
              dir="auto"
              role="status"
            >
              {rematchNote}
            </p>
          {/if}
          <Button variant="primary" class="col-span-2" onclick={onrematch} data-rematch>
            <RotateCw class="size-4" aria-hidden="true" />
            {rematchLabel ?? t('choices.over.rematch')}
          </Button>
        {/if}
        <Button
          variant="secondary"
          class="px-1! text-[13px]! whitespace-nowrap sm:px-2! sm:text-[14px]!"
          onclick={() => (collapsed = true)}
          data-view-board
        >
          <Eye class="hidden size-4 shrink-0 sm:block" aria-hidden="true" />
          {t('choices.over.viewBoard')}
        </Button>
        <Button
          variant="secondary"
          class="px-1! text-[13px]! whitespace-nowrap sm:px-2! sm:text-[14px]!"
          onclick={onhome}
          data-home
        >
          <House class="hidden size-4 shrink-0 sm:block" aria-hidden="true" />
          {t('choices.over.home')}
        </Button>
      </div>
    </div>
  </Sheet>

  {#if collapsed}
    <PeekBar
      inline
      text="{t('choices.over.bar')} · {headline}"
      openLabel={t('choices.over.results')}
      onopen={() => (collapsed = false)}
    />
  {/if}
</div>

<style>
  /* The placeholder: the paper with a faint gold dot grid, like the wonder
     placeholder's sky, until the scene is approved. */
  .victory-placeholder {
    background:
      radial-gradient(circle, var(--gold) 0 1px, transparent 1.5px) 0 0 / 12px 12px,
      linear-gradient(
        to bottom,
        color-mix(in oklch, var(--scene-sky) 60%, transparent),
        transparent 80%
      );
    opacity: 0.6;
  }
</style>
