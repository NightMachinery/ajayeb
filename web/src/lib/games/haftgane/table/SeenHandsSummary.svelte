<script lang="ts">
  // The top of the Seen hands sheet: the statistics the server sends with
  // the notebook, each a figure the player can tap to show the cards it is
  // about. Per colour, how many cards of the hands you passed may still be
  // going round; for a neighbour, the card it took from a hand you passed,
  // once that is certain; and the cards that can still come back to you
  // this Age (the server sends these only at five seats or fewer). Every
  // figure is marked as certain or as an upper bound, as the server bounds
  // it; nothing here infers anything.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Bounds, Statistics } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { pointParts, type SeenFilter } from './city';
  import { cardName } from './labels';
  import { COLOURS, design } from './model';

  interface Props {
    statistics: Statistics;
    names: Readonly<Record<number, string>>;
    filter: SeenFilter | null;
    onfilter: (filter: SeenFilter | null) => void;
  }

  let { statistics, names, filter, onfilter }: Props = $props();

  function bounds(b: Bounds): string {
    if (b.certain === b.atMost) return ht('hg.seen.bounds.exact', { count: b.certain });
    if (b.certain === 0) return ht('hg.seen.bounds.atMost', { count: b.atMost });
    return ht('hg.seen.bounds.range', { certain: b.certain, atMost: b.atMost });
  }
  const circulating = $derived(
    COLOURS.flatMap((colour) => {
      const b = statistics.circulating.find((c) => c.colour === colour)?.bounds;
      return b && b.atMost > 0 ? [{ colour, b }] : [];
    })
  );
  const isColour = (c: string) => !!filter && 'colour' in filter && filter.colour === c;
  const isCard = (slug: string) => !!filter && 'design' in filter && filter.design === slug;
  const toggle = (next: SeenFilter, on: boolean) => onfilter(on ? null : next);
</script>

<section class="summary" aria-labelledby="hg-seen-summary" data-seen-summary>
  <h3 id="hg-seen-summary">{ht('hg.seen.summary')}</h3>

  <h4>{ht('hg.seen.circulating')}</h4>
  {#if circulating.length === 0}
    <p class="none">{ht('hg.seen.circulating.none')}</p>
  {:else}
    <div class="figures">
      {#each circulating as { colour, b } (colour)}
        {@const on = isColour(colour)}
        {@const text = bounds(b)}
        {@const tip = ht('hg.seen.colour.tip', {
          colour: ht(`hg.screen.colourName.${colour}`),
          bounds: text
        })}
        <button
          type="button"
          class="figure"
          style:--ink="var(--card-{colour})"
          aria-pressed={on}
          aria-label={tip}
          data-act="seen-filter:colour:{colour}"
          data-circulating={colour}
          data-certain={b.certain}
          data-at-most={b.atMost}
          onclick={() => toggle({ colour }, on)}
          {@attach tooltip({ text: tip, actionable: true })}
        >
          <span class="band"><HgGlyph name="category-{colour}-sm" /></span>
          <span>{text}</span>
        </button>
      {/each}
    </div>
  {/if}

  {#if statistics.neighbourPicks.length > 0}
    <h4>{ht('hg.seen.picks')}</h4>
    <div class="figures">
      {#each statistics.neighbourPicks as pick, i (i)}
        {@const parts = pointParts(pick.point)}
        {@const on = isCard(pick.design)}
        {@const name = names[pick.seat] ?? ht('hg.screen.seat', { seat: pick.seat + 1 })}
        <button
          type="button"
          class="figure"
          style:--ink="var(--card-{design(pick.design)?.colour ?? 'brown'})"
          aria-pressed={on}
          data-act="seen-filter:card:{pick.design}"
          data-neighbour-pick={pick.seat}
          data-card={pick.design}
          onclick={() => toggle({ design: pick.design }, on)}
          {@attach tooltip({
            text: ht('hg.seen.pick.tip', { name, card: cardName(pick.design) }),
            actionable: true
          })}
        >
          <span class="band"
            ><HgGlyph name="category-{design(pick.design)?.colour ?? 'brown'}-sm" /></span
          >
          <span dir="auto"
            >{ht('hg.seen.pick', {
              name,
              card: cardName(pick.design),
              age: parts?.age ?? 0,
              turn: parts?.turn ?? 0
            })}</span
          >
        </button>
      {/each}
    </div>
  {/if}

  {#if statistics.returning.length > 0}
    <h4>{ht('hg.seen.returning')}</h4>
    <div class="figures">
      {#each statistics.returning as r (r.design)}
        {@const on = isCard(r.design)}
        {@const text = bounds(r.bounds)}
        <button
          type="button"
          class="figure"
          style:--ink="var(--card-{design(r.design)?.colour ?? 'brown'})"
          aria-pressed={on}
          data-act="seen-filter:card:{r.design}"
          data-returning={r.design}
          data-card={r.design}
          onclick={() => toggle({ design: r.design }, on)}
          {@attach tooltip({
            text: ht('hg.seen.returning.tip', { card: cardName(r.design), bounds: text }),
            actionable: true
          })}
        >
          <span class="band"
            ><HgGlyph name="category-{design(r.design)?.colour ?? 'brown'}-sm" /></span
          >
          <span>{cardName(r.design)}{ht('hg.screen.sep')}{text}</span>
        </button>
      {/each}
    </div>
  {/if}
</section>

<style>
  .summary {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    min-width: 0;
    padding: 0.6rem 0.7rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel-sunken);
  }
  h3 {
    margin: 0;
    font: 600 1.05rem var(--type-display);
  }
  h4 {
    margin: 0.2rem 0 0;
    color: var(--fg-muted);
    font-size: 0.85rem;
    font-weight: 600;
  }
  .none {
    margin: 0;
    color: var(--fg-muted);
    font-size: 0.85rem;
  }
  .figures {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }
  .figure {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    max-width: 100%;
    min-width: 44px;
    min-height: 44px;
    padding-block: 0.3rem;
    padding-inline: 0.3rem 0.6rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: inherit;
    font-size: 0.875rem;
    text-align: start;
    cursor: pointer;
  }
  .figure[aria-pressed='true'] {
    border: 2px solid var(--ink);
    background: color-mix(in oklch, var(--ink) 14%, var(--panel));
  }
  .band {
    display: grid;
    flex: 0 0 1.5rem;
    place-items: center;
    height: 1.5rem;
    border-radius: 0.3rem;
    background: var(--ink);
    color: var(--paper-bright);
  }
</style>
