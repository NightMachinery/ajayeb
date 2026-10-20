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
  // Above the score sheet on a phone, beside it in a wider dialog where
  // two columns fit, the turn that ended the game (M11.gameover-last-move): the move summary's own rows (MoveSummaryLine)
  // for the final move, with a wonder's extra turn and the choices inside
  // the turn, so the result needs no popup of its own over it (the summary
  // popup never opens at game over). Under them, the viewer's move review
  // (the advisor family, M10.F3): of that final move when it was theirs
  // (view/end-review.ts), else of their move before it, as the popup
  // carried it (view/review.ts reviewForSummary). A forfeit, which no move
  // ended, has no section.
  //
  // The wordmark (M11.ornament-integration), small and decorative, never
  // costs the score sheet its room: on a phone it heads the dialog, where
  // the sheet has height to spare; from the sm width (a dialog capped at
  // 640 px) it stands at the end of the header row, beside the scene, which
  // is taller than the headline; a short screen (a phone in landscape),
  // whose dialog already scrolls, leaves it out.
  //
  // A short landscape screen (a phone on its side) has the height of about
  // nine score rows under its footer, so there the body is two columns: the
  // scene, the headline and the final move on one side, and the score sheet
  // on the other, from the top and held there while the first column
  // scrolls, so the totals show at once. Its footer is one row, and the
  // score rows are tighter (the style block's short-landscape query).
  import { restoreFocus } from '$lib/ui/dismiss';
  import Download from '@lucide/svelte/icons/download';
  import { downloadReplay } from '$lib/session/replay';
  import House from '@lucide/svelte/icons/house';
  import RotateCw from '@lucide/svelte/icons/rotate-cw';
  import Eye from '@lucide/svelte/icons/eye';
  import ArtImage from '$lib/art/ArtImage.svelte';
  import Wordmark from '$lib/brand/Wordmark.svelte';
  import { victoryArtId } from '$lib/art/lookup';
  import { visibleFamilies } from '$lib/hints/families';
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { endReview } from '$lib/view/end-review';
  import { finalTurn, type MoveSummary } from '$lib/view/move-summary';
  import { reviewForSummary, reviewLines, type ReviewLines } from '$lib/view/review';
  import { finalScores, outcomeOf, scoreRows, type Outcome } from '$lib/view/score';
  import type { View } from '$lib/wire';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import PeekBar from '../choices/PeekBar.svelte';
  import { asSeat, isYou, seatNames, tryGameContext, type SeatNames } from '../choices/room';
  import ScoreTable from './ScoreTable.svelte';
  import MoveReview from '../game/MoveReview.svelte';
  import MoveSummaryLine from '../game/MoveSummaryLine.svelte';

  interface Props {
    view: View;
    /** Wait behind an unread age summary or an open move summary. */
    show?: boolean;
    /** Wait for initial presentation, then keep focus stable across frames. */
    ready?: boolean;
    /** The room whose finished, revealed log may be downloaded. */
    replayCode?: string;
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
    /** The viewer's move review the final move's section carries; else
     *  from the view and the page's game context. */
    review?: ReviewLines;
    /** The turn that ended the game; else from the page's game context. */
    lastMove?: MoveSummary | null;
  }

  let {
    view,
    show = true,
    ready = true,
    replayCode,
    onrematch,
    rematchLabel,
    rematchNote,
    onhome,
    names: namesProp,
    review: reviewProp,
    lastMove: lastMoveProp
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
  let shown = $state(false);

  function setOpen(open: boolean): void {
    if (!show) return;
    collapsed = !open;
    if (!open)
      requestAnimationFrame(() => {
        // Native close may retain focus inside the closed dialog until
        // its exit completes. That element is no longer a usable opener.
        const focused = document.activeElement;
        if (
          !document.querySelector('dialog:modal') &&
          (focused === document.body || focused?.closest('dialog:not([open])'))
        ) {
          const menu = document.querySelector<HTMLButtonElement>(
            '.game-screen [data-control="menu"]'
          );
          if (menu) restoreFocus(menu);
        }
      });
  }

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

  const lastMove = $derived.by((): MoveSummary | null => {
    if (outcome?.kind === 'forfeit') return null;
    if (lastMoveProp !== undefined) return lastMoveProp;
    const history = ctx?.session.history;
    return history ? finalTurn(history) : null;
  });
  // The end of the game is the dialog's own headline, not a row.
  const lastLines = $derived(lastMove?.lines.filter((l) => l.kind !== 'gameOver') ?? []);
  const lastTitle = $derived(
    !lastMove
      ? ''
      : youSeat === lastMove.seat
        ? t('choices.over.lastMoveYou')
        : t('choices.over.lastMove', { name: names[lastMove.seat] })
  );

  const review = $derived.by((): ReviewLines | null => {
    if (reviewProp) return reviewProp;
    const session = ctx?.session;
    if (!session || !visibleFamilies(session.hints, hintHide.value).has('advisor')) return null;
    if (view.viewer === undefined) return null;
    const own =
      endReview(view.review, view.viewer, session.history, outcome?.kind) ??
      (lastMove && lastMove.seat !== view.viewer
        ? reviewForSummary(view.review, lastMove, session.history)
        : null);
    return own ? reviewLines(own, { names, seat: asSeat(view.viewer) }) : null;
  });

  const hasLastMove = $derived(lastLines.length > 0 || review !== null);

  // The footer's buttons, which a short landscape screen puts in one row.
  const actionCount = $derived((onrematch ? 1 : 0) + (replayCode && view.revealed ? 1 : 0) + 2);

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
  onfocusin={() => (shown = true)}
  data-winner={view.result?.winner}
  data-victory={outcome?.kind}
  data-collapsed={collapsed}
>
  <Sheet
    bind:open={() => show && !collapsed && (ready || shown), setOpen}
    centered
    {titleId}
    class="game-over {hasLastMove ? 'has-last-move' : ''}"
  >
    {#snippet footer()}
      <div
        class="over-actions grid gap-2 {replayCode && view.revealed
          ? 'grid-cols-3'
          : 'grid-cols-2'}"
        style:--over-actions={actionCount}
      >
        {#if onrematch}
          <Button
            variant="primary"
            class="over-rematch col-span-full"
            onclick={onrematch}
            data-rematch
          >
            <RotateCw class="size-4" aria-hidden="true" />
            {rematchLabel ?? t('choices.over.rematch')}
          </Button>
        {/if}
        {#if replayCode && view.revealed}
          <Button
            variant="secondary"
            class="h-auto! min-h-12 min-w-0 flex-col gap-0.5! px-1! py-1 text-xs! leading-tight"
            onclick={() => replayCode && downloadReplay(replayCode)}
            data-download-replay
          >
            <Download class="size-4" aria-hidden="true" />
            {t('game.menu.replay')}
          </Button>
        {:else if replayCode}
          <p class="over-note col-span-full text-center text-sm text-fg-muted" data-replay-hidden>
            {t('game.menu.replayHidden')}
          </p>
        {/if}
        <Button
          variant="secondary"
          class="h-auto! min-h-12 min-w-0 flex-col gap-0.5! px-1! py-1 text-xs! leading-tight"
          onclick={() => (collapsed = true)}
          data-view-board
        >
          <Eye class="size-4 shrink-0" aria-hidden="true" />
          {t('choices.over.viewBoard')}
        </Button>
        <Button
          variant="secondary"
          class="h-auto! min-h-12 min-w-0 flex-col gap-0.5! px-1! py-1 text-xs! leading-tight"
          onclick={onhome}
          data-home
        >
          <House class="size-4 shrink-0" aria-hidden="true" />
          {t('choices.over.home')}
        </Button>
      </div>
    {/snippet}
    <div class="over-body flex flex-col gap-3 px-5 pt-4 pb-1">
      <Wordmark class="mx-auto -mb-1 block w-20 text-fg short:hidden {scene ? 'sm:hidden' : ''}" />
      <header
        class={[
          'over-start flex',
          scene
            ? 'flex-row items-center gap-4 text-start'
            : 'flex-col items-center gap-1 text-center'
        ]}
      >
        {#if scene}
          <!-- The scene takes the crest row's place, beside the headline,
               so the dialog grows by no more than a wrapped line: the score
               table and the buttons keep their room in the sheet's height
               on a phone as on a desktop. The crests sit on
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
          <h2 id={titleId} class="over-headline font-display text-2xl leading-tight text-fg">
            {headline}
          </h2>
          {#if reason}
            <p class="text-sm text-fg-muted">{reason}</p>
          {/if}
        </div>
        {#if scene}
          <Wordmark
            class="ms-auto hidden w-20 shrink-0 self-start text-fg sm:block short:hidden!"
          />
        {/if}
      </header>

      <!-- Phone held upright: the final move above the score sheet. With room
           for two columns (split: a desktop or a short landscape screen),
           the dialog widens and puts it beside the sheet, first in reading
           order, so the dialog is no taller than without it and the totals
           stay in view on a laptop's short window. -->
      <div
        class={hasLastMove
          ? 'over-pair flex flex-col gap-3 split:grid split:grid-cols-[minmax(0,17rem)_minmax(0,1fr)] split:items-start split:gap-x-4'
          : 'contents'}
      >
        {#if hasLastMove}
          <section
            class="over-start flex flex-col gap-2 rounded-control border border-rule bg-panel-sunken px-3 py-2"
            aria-labelledby={lastLines.length > 0 ? 'game-over-last-move' : undefined}
            data-last-move={lastMove?.seat}
            data-plies={lastMove?.plies.join(' ')}
          >
            {#if lastMove && lastLines.length > 0}
              <h3
                id="game-over-last-move"
                class="text-xs font-semibold tracking-wide text-fg-muted uppercase"
              >
                {lastTitle}
              </h3>
              <ol class="flex flex-col gap-2">
                {#each lastLines as line, i (`${line.ply}:${i}`)}
                  <MoveSummaryLine {line} seat={lastMove.seat} {names} />
                {/each}
              </ol>
            {/if}
            {#if review}
              <div data-end-review><MoveReview lines={review} /></div>
            {/if}
          </section>
        {/if}

        <div class="over-scores">
          <ScoreTable {rows} {names} {winner} you={youSeat} />
        </div>
      </div>

      {#if outcome?.tieBreak}
        <p
          class="over-start rounded-control border border-rule bg-panel-sunken px-3 py-2 text-sm text-fg"
          data-tie-break
        >
          <Icon name="category-blue" class="me-1 text-card-blue-ink" />
          {t('choices.over.tieBreak')}
        </p>
      {/if}

      {#if rematchNote}
        <p class="over-start text-center text-sm font-semibold text-fg" role="status">
          {rematchNote}
        </p>
      {/if}
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

<style>
  /* Taller than the other centred sheets (640 px at most): on a phone held
     upright the final move sits above the score sheet, and the whole sheet
     still shows without scrolling, as it did before the section came. A
     short screen gets its height less the margin, as before, and scrolls. */
  :global(.sheet.centered.game-over) {
    --sheet-max-height: min(calc(100dvh - 2rem), 52rem);
  }
  /* With room for two columns (the split variant's query), wide enough for
     the final move beside the score sheet. */
  @media (min-width: 64rem), (orientation: landscape) and (max-height: 32rem) {
    :global(.sheet.centered.game-over.has-last-move) {
      width: min(46rem, calc(100% - 2rem));
    }
  }

  /* A short landscape screen (the short variant, landscape): two columns,
     the scene, headline, final move and notes first, the score sheet beside
     them from the top, held in view while the first column scrolls (an
     auto row per item there, and a last one that takes the slack, so the
     sheet's height never spreads them apart). The footer is one row, with a
     smaller bottom margin. */
  @media (max-height: 32rem) and (orientation: landscape) {
    :global(.sheet.centered.game-over) {
      width: min(46rem, calc(100% - 2rem));
    }
    :global(.sheet.centered.game-over .sheet-panel) {
      --sheet-bottom-padding: 0.5rem;
    }
    .over-body {
      display: grid;
      grid-template-columns: minmax(0, 17rem) minmax(0, 1fr);
      grid-template-rows: repeat(4, auto) 1fr;
      align-items: start;
      column-gap: 1rem;
      row-gap: 0.75rem;
      padding-block-start: 0.75rem;
    }
    .over-pair {
      display: contents;
    }
    .over-start {
      grid-column: 1;
    }
    /* The first column is narrow: a smaller scene and headline. */
    header.over-start {
      gap: 0.75rem;
    }
    .victory-scene {
      width: 6rem;
    }
    .over-headline {
      font-size: 1.25rem;
      overflow-wrap: break-word;
    }
    .over-scores {
      position: sticky;
      inset-block-start: 0;
      grid-column: 2;
      grid-row: 1 / -1;
    }
    .over-actions {
      grid-template-columns: repeat(var(--over-actions), minmax(0, 1fr));
    }
    .over-actions :global(.over-rematch) {
      grid-column: auto;
    }
    .over-actions :global(button) {
      height: auto;
      min-height: 2.75rem;
    }
    .over-note {
      order: 1;
    }
  }

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
