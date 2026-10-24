<script lang="ts">
  // A city's buildings by colour, one row per colour in the order the table
  // screen counts them: the category glyph, the card names and the count,
  // with an empty colour dimmed and uncounted. A city with no building yet
  // says so in one line instead. Each name carries the card's tooltip (what
  // the card does), so a row can be read without opening anything.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { cityRows } from './city';
  import { effectText } from './effect-text';
  import { cardName } from './labels';
  import { design } from './model';

  interface Props {
    built: readonly string[];
  }

  let { built }: Props = $props();
  const rows = $derived(cityRows(built));
  const cardTip = (slug: string) =>
    ht('hg.seen.card.tip', {
      card: cardName(slug),
      effect: (design(slug)?.effects ?? []).map(effectText).join(' ')
    });
</script>

<ul class="rows" data-city-rows>
  {#if built.length === 0}<li class="nothing" data-city-empty>{ht('hg.city.row.nothing')}</li>{/if}
  {#each built.length === 0 ? [] : rows as row (row.colour)}
    {@const tip = `${ht(`hg.screen.colourName.${row.colour}`)}: ${ht(`hg.screen.colour.${row.colour}`, { count: row.cards.length })}`}
    <li
      class="row"
      class:empty={row.cards.length === 0}
      style:--ink="var(--card-{row.colour})"
      data-city-row={row.colour}
    >
      <span class="glyph" role="img" aria-label={tip} {@attach tooltip({ text: tip })}>
        <HgGlyph name="category-{row.colour}" />
      </span>
      <span class="names">
        {#if row.cards.length === 0}
          {ht('hg.city.row.none')}
        {:else}
          {#each row.cards as slug, i (i)}<span
              class="card"
              data-card={slug}
              {@attach tooltip({ text: cardTip(slug) })}>{cardName(slug)}</span
            >{#if i < row.cards.length - 1}<span class="sep">{ht('hg.city.row.sep')}</span
              >{/if}{/each}
        {/if}
      </span>
      {#if row.cards.length > 0}<span class="n">{formatNumber(row.cards.length)}</span>{/if}
    </li>
  {/each}
</ul>

<style>
  .rows {
    display: grid;
    gap: 0.35rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    min-width: 0;
    min-height: 2.75rem;
    padding-block: 0.3rem;
    padding-inline: 0.3rem 0.6rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel);
  }
  .glyph {
    display: grid;
    flex: 0 0 2rem;
    place-items: center;
    height: 2rem;
    border-radius: 0.4rem;
    background: var(--ink);
    color: var(--paper-bright);
    font-size: 1.15rem;
  }
  .names {
    flex: 1;
    min-width: 0;
    line-height: 1.35;
    overflow-wrap: anywhere;
  }
  .card {
    text-decoration: underline dotted color-mix(in oklch, var(--fg) 40%, transparent);
    text-underline-offset: 0.2em;
  }
  .n {
    flex-shrink: 0;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .empty {
    border-color: var(--rule);
    color: var(--fg-muted);
  }
  .nothing {
    padding: 0.5rem 0.3rem;
    color: var(--fg-muted);
  }
  .empty .glyph {
    opacity: 0.45;
  }
</style>
