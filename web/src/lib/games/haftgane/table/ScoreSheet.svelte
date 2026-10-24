<script lang="ts">
  // The final scores (docs/haftgane/phone-layout.md, section 6): the
  // verdict first, from the server's winners, then the tie-break sentence
  // when one applied, then the score table, and the actions: Rematch (the
  // entry's own control), Download, Final board and Home. Every number is
  // the server's result; the client adds nothing up and decides no winner.
  // Download shows only when the view allows it: with the end kept hidden,
  // a note says why instead. A phone in portrait gets a bottom sheet the
  // full width of the screen, so the names keep room beside eight number
  // columns; a short landscape screen gives the table the start column and
  // the verdict with the actions the end column. A tapped row opens that
  // seat's city when the screen offers it.
  import type { Snippet } from 'svelte';
  import { formatNumber, i18n } from '$lib/i18n';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import type { Result, Seat } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import ScoreTable from './ScoreTable.svelte';
  import { coinsOf, scoreRows, verdictOf, type TieBreak } from './end';

  interface Props {
    result: Result;
    seats: readonly Seat[];
    names: Readonly<Record<number, string>>;
    /** The viewer's seat, or null for a spectator. */
    you: number | null;
    /** The tie at the top, once the view states it (requested as
     *  `result.tieBreak`); absent today. */
    tieBreak?: TieBreak;
    canDownload: boolean;
    ondownload: () => void;
    onhome: () => void;
    /** Closes the sheet: the final board behind it. */
    onclose: () => void;
    /** The rematch control, which the entry owns. */
    rematch?: Snippet;
    /** Opens a seat's city from its row. */
    onseat?: (seat: number) => void;
  }

  let {
    result,
    seats,
    names,
    you,
    tieBreak,
    canDownload,
    ondownload,
    onhome,
    onclose,
    rematch,
    onseat
  }: Props = $props();

  let open = $state(true);
  const rows = $derived(scoreRows(result));
  const verdict = $derived(verdictOf(result, you));
  const coins = $derived(Object.fromEntries(seats.map((s) => [s.seat, s.coins])));
  const list = (items: string[]) =>
    new Intl.ListFormat(i18n.locale, { type: 'conjunction' }).format(items);
  // Names for a sentence: the others in seat order, the viewer last as
  // "you", so a sentence never starts with the lower-case word.
  function named(seatList: readonly number[]): string[] {
    const others = seatList.filter((s) => s !== you).map((s) => names[s] ?? '');
    return seatList.includes(you ?? -1) ? [...others, ht('hg.end.you')] : others;
  }
  const title = $derived.by(() => {
    if (!verdict) return ht('hg.end.verdict.none');
    switch (verdict.kind) {
      case 'you':
        return ht('hg.end.verdict.you');
      case 'one':
        return ht('hg.end.verdict.one', { name: names[verdict.seat] ?? '' });
      case 'shared':
        return verdict.you
          ? ht('hg.end.verdict.sharedYou', { names: list(named(verdict.seats)) })
          : ht('hg.end.verdict.shared', { names: list(named(verdict.seats)) });
    }
    return '';
  });
  const tieText = $derived.by(() => {
    if (!tieBreak || tieBreak.seats.length < 2) return '';
    const total = result.scores[tieBreak.seats[0]!]?.total ?? 0;
    const who = list(named(tieBreak.seats));
    if (tieBreak.kind === 'shared')
      return ht('hg.end.tie.shared', {
        names: who,
        total,
        coins: coinsOf(seats, tieBreak.seats[0]!)
      });
    const counts = tieBreak.seats
      .map((s) =>
        ht('hg.end.tie.count', {
          name: s === you ? ht('hg.end.you') : (names[s] ?? ''),
          coins: formatNumber(coinsOf(seats, s))
        })
      )
      .join(ht('hg.end.tie.separator'));
    return ht('hg.end.tie.coins', { names: who, total, counts });
  });
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-score-title"
  class="hg-scores"
  panelData={{ 'data-score-sheet': '' }}
>
  <div class="body">
    <header class="verdict">
      <h2 id="hg-score-title" data-verdict={verdict?.kind ?? 'none'}>{title}</h2>
      {#if tieText}
        <p class="tie" data-tie-break={tieBreak?.kind}>{tieText}</p>
      {/if}
    </header>
    <div class="table">
      <ScoreTable {rows} {names} {you} {coins} {onseat} />
    </div>
    <div class="actions">
      {#if rematch}
        <div class="rematch">{@render rematch()}</div>
      {/if}
      <div class="row">
        {#if canDownload}
          <Button variant="secondary" onclick={ondownload} data-act="download-log">
            {ht('hg.end.download')}
          </Button>
        {/if}
        <Button variant="secondary" onclick={() => (open = false)} data-act="final-board">
          {ht('hg.end.board')}
        </Button>
        <Button variant="secondary" onclick={onhome} data-act="home">
          {ht('hg.end.home')}
        </Button>
      </div>
      {#if !canDownload}
        <p class="note" data-log-hidden>{ht('hg.end.download.hidden')}</p>
      {/if}
    </div>
  </div>
</Sheet>

<style>
  :global(dialog.sheet.hg-scores) {
    width: min(100%, 40rem);
  }
  .body {
    display: grid;
    grid-template-areas:
      'verdict'
      'table'
      'actions';
    gap: 0.75rem;
    padding-block: 0.25rem 0;
    padding-inline: 0.75rem;
    color: var(--fg);
  }
  .verdict {
    grid-area: verdict;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }
  .table {
    grid-area: table;
    min-width: 0;
  }
  h2 {
    margin: 0;
    font: 600 1.6rem/1.15 var(--type-display);
    overflow-wrap: anywhere;
  }
  .tie {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
    line-height: 1.4;
  }
  .actions {
    grid-area: actions;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .rematch :global(button) {
    width: 100%;
  }
  .row {
    display: grid;
    grid-auto-columns: minmax(0, 1fr);
    grid-auto-flow: column;
    gap: 0.4rem;
  }
  .row :global(button) {
    min-width: 0;
    padding-inline: 0.4rem;
    font-size: 0.875rem;
  }
  .note {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.8125rem;
    text-align: center;
  }
  /* A short landscape screen: the table at the start, the verdict at the end. */
  @media (orientation: landscape) and (max-height: 32rem) {
    :global(dialog.sheet.hg-scores) {
      width: min(100%, 52rem);
    }
    .body {
      grid-template-columns: minmax(0, 1.7fr) minmax(0, 1fr);
      grid-template-rows: auto 1fr;
      grid-template-areas:
        'table verdict'
        'table actions';
      align-items: start;
      padding-block-start: 0.75rem;
    }
    h2 {
      font-size: 1.35rem;
    }
    .row {
      grid-auto-flow: row;
    }
  }
</style>
