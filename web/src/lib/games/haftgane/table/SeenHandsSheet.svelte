<script lang="ts">
  // The Seen hands sheet (phone-layout.md section 8): the viewer's private
  // notebook, opened from the hand header. The statistics come first; then
  // tabs by Age, the current one first, list each turn's hand with the card
  // played from it and what was done with it, and the pile at each look
  // through the discards. A tapped figure filters the list to its cards.
  // Everything shown is the notebook the view sends this viewer.
  import { i18n } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import type { Notebook } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import MiniCard from './MiniCard.svelte';
  import SeenHandsSummary from './SeenHandsSummary.svelte';
  import {
    itemMatches,
    matchesFilter,
    notebookAges,
    type NotebookItem,
    type SeenFilter
  } from './city';
  import { cardName } from './labels';

  interface Props {
    notebook: Notebook;
    /** The current Age, whose tab comes first. */
    age: number;
    names: Readonly<Record<number, string>>;
    onclose: () => void;
  }

  let { notebook, age, names, onclose }: Props = $props();

  let open = $state(true);
  let filter = $state<SeenFilter | null>(null);
  let chosenAge = $state<number | null>(null);
  const ages = $derived(notebookAges(notebook, age));
  const shownAge = $derived(ages.find((a) => a.age === chosenAge) ?? ages[0] ?? { age, items: [] });
  const items = $derived(shownAge.items.filter((i) => itemMatches(i, filter)));
  const empty = $derived(notebook.seen.length === 0 && notebook.pileLooks.length === 0);
  const numeral = (n: number) => ht(`hg.screen.stageNumeral.${n}` as 'hg.screen.stageNumeral.1');
  const filterName = $derived(
    !filter
      ? ''
      : 'colour' in filter
        ? ht(`hg.screen.colourName.${filter.colour}`)
        : cardName(filter.design)
  );
  const heading = (item: NotebookItem) =>
    item.kind === 'pile'
      ? ht('hg.seen.pile', { turn: item.parts.turn })
      : item.parts.kind === 'last-card'
        ? ht('hg.seen.lastCard')
        : ht('hg.seen.turn', { turn: item.parts.turn });
  type Action = 'build' | 'stage' | 'sell' | 'build-discarded';
  const isAction = (a: string): a is Action =>
    a === 'build' || a === 'stage' || a === 'sell' || a === 'build-discarded';
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-seen-title"
  class="hg-seen-sheet"
  panelData={{ 'data-seen-sheet': '' }}
>
  <div class="content" dir={i18n.dir}>
    <header class="head">
      <h2 id="hg-seen-title">{ht('hg.seen.title')}</h2>
      <button
        type="button"
        class="close"
        data-act="seen-close"
        aria-label={ht('hg.seen.close')}
        onclick={onclose}
        {@attach tooltip({ text: ht('hg.seen.close'), actionable: true })}>×</button
      >
    </header>
    <p class="intro">{ht('hg.seen.intro')}</p>

    {#if empty}
      <p class="empty" data-seen-empty>{ht('hg.seen.empty')}</p>
    {:else}
      <SeenHandsSummary
        statistics={notebook.statistics}
        {names}
        {filter}
        onfilter={(f) => (filter = f)}
      />

      {#if filter}
        <div class="filter" data-seen-filter>
          <span>{ht('hg.seen.filter', { what: filterName })}</span>
          <button
            type="button"
            class="clear"
            data-act="seen-filter-clear"
            onclick={() => (filter = null)}>{ht('hg.seen.filter.clear')}</button
          >
        </div>
      {/if}

      <div class="ages" role="tablist" aria-label={ht('hg.seen.ages')}>
        {#each ages as a (a.age)}
          {@const selected = a.age === shownAge.age}
          <button
            type="button"
            role="tab"
            class="age"
            aria-selected={selected}
            aria-controls="hg-seen-panel"
            tabindex={selected ? 0 : -1}
            data-act="seen-age:{a.age}"
            onclick={() => (chosenAge = a.age)}
            >{ht('hg.screen.age', { age: numeral(a.age) })}</button
          >
        {/each}
      </div>

      <div id="hg-seen-panel" role="tabpanel" class="items" data-seen-age={shownAge.age}>
        {#if items.length === 0}
          <p class="empty">{ht('hg.seen.emptyAge')}</p>
        {/if}
        {#each items as item (item.kind + (item.kind === 'hand' ? item.seen.point : item.look.point))}
          <section
            class="item"
            data-seen-point={item.kind === 'hand' ? item.seen.point : item.look.point}
            data-seen-kind={item.kind}
          >
            <h3>{heading(item)}</h3>
            <div class="cards">
              {#if item.kind === 'hand'}
                {@const action = item.seen.action}
                {@const at = item.seen.hand.indexOf(item.seen.played)}
                {#each item.seen.hand as slug, i (i)}
                  <MiniCard
                    {slug}
                    mark={i === at && isAction(action) ? ht(`hg.seen.played.${action}`) : ''}
                    markTip={i === at && isAction(action) ? ht(`hg.seen.played.tip.${action}`) : ''}
                    dim={!matchesFilter(slug, filter)}
                  />
                {/each}
              {:else}
                {#each item.look.cards as c (c.design)}
                  <MiniCard
                    slug={c.design}
                    count={c.count}
                    dim={!matchesFilter(c.design, filter)}
                  />
                {/each}
              {/if}
            </div>
          </section>
        {/each}
      </div>
    {/if}
  </div>
</Sheet>

<style>
  :global(dialog.sheet.hg-seen-sheet) {
    width: min(100%, 46rem);
    --sheet-max-height: calc(100dvh - 0.5rem);
  }
  .content {
    display: flex;
    flex-direction: column;
    gap: 0.7rem;
    padding: 0.4rem 0.75rem 1.2rem;
    color: var(--fg);
  }
  .head {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  h2 {
    flex: 1;
    margin: 0;
    font: 600 1.55rem/1.15 var(--type-display);
  }
  .close {
    display: grid;
    flex: 0 0 44px;
    place-items: center;
    width: 44px;
    height: 44px;
    border: 0;
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 1.5rem;
    cursor: pointer;
  }
  .intro,
  .empty {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.9rem;
    line-height: 1.4;
  }
  .filter {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    font-weight: 600;
  }
  .clear {
    min-width: 44px;
    min-height: 44px;
    padding-inline: 0.7rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    cursor: pointer;
  }
  .ages {
    display: flex;
    gap: 0.25rem;
    border-block-end: 1px solid var(--rule);
  }
  .age {
    min-width: 44px;
    min-height: 44px;
    padding-inline: 0.8rem;
    border: 0;
    border-block-end: 3px solid transparent;
    background: none;
    color: var(--fg-muted);
    font: 600 0.95rem var(--type-display);
    cursor: pointer;
  }
  .age[aria-selected='true'] {
    border-block-end-color: var(--lapis);
    color: var(--fg);
  }
  .items {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }
  .item {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  h3 {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.85rem;
    font-weight: 600;
  }
  .cards {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
  }
</style>
