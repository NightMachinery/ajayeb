<script lang="ts">
  // One printed effect as a compact glyph group for a card's colour band,
  // with a tooltip that says what it does. Drawn from the catalog's codes.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { HgEffect } from '$lib/wire/haftgane-catalog.gen';
  import { effectText } from './effect-text';
  import HgGlyph from './HgGlyph.svelte';
  import ResourceTile from './ResourceTile.svelte';

  interface Props {
    effect: HgEffect;
  }

  let { effect }: Props = $props();

  const n = $derived(effect.n ?? 1);
  const copies = $derived(Array.from({ length: Math.min(n, 3) }, (_, i) => i));
  const tip = $derived(effectText(effect));
</script>

<span
  class="fx"
  data-effect={effect.kind}
  role="img"
  aria-label={tip}
  {@attach tooltip({ text: tip })}
>
  {#if effect.kind === 'produce'}
    {#each copies as i (i)}<ResourceTile res={effect.res?.[0] ?? ''} />{/each}
  {:else if effect.kind === 'produce-choice'}
    {#each effect.res ?? [] as r, i (r)}{#if i > 0}<span class="sep">/</span>{/if}<ResourceTile
        res={r}
      />{/each}
  {:else if effect.kind === 'trade-discount'}
    <HgGlyph name="trade" />
    {#each effect.res ?? [] as r (r)}<ResourceTile res={r} size="xs" />{/each}
  {:else if effect.kind === 'shields'}
    {#each copies as i (i)}<HgGlyph name="shield" />{/each}
  {:else if effect.kind === 'science'}
    <HgGlyph name="science-{effect.symbol ?? 'compass'}-sm" />
  {:else if effect.kind === 'science-choice'}
    <HgGlyph name="science-compass-sm" /><span class="sep">/</span><HgGlyph
      name="science-gear-sm"
    /><span class="sep">/</span><HgGlyph name="science-tablet-sm" />
  {:else if effect.kind === 'coins' || effect.kind === 'coins-per-card' || effect.kind === 'coins-per-stage'}
    {#if effect.kind === 'coins-per-card'}{#each effect.colours ?? [] as c (c)}<HgGlyph
          name="category-{c}-sm"
        />{/each}{:else if effect.kind === 'coins-per-stage'}<HgGlyph name="wonder-stage" />{/if}
    <span class="coin">{formatNumber(n)}</span>
  {:else if effect.kind === 'points' || effect.kind === 'points-per-card' || effect.kind === 'points-per-stage' || effect.kind === 'points-if-wonder-complete'}
    {#if effect.kind === 'points-per-card'}{#each effect.colours ?? [] as c (c)}<HgGlyph
          name="category-{c}-sm"
        />{/each}{:else if effect.kind === 'points-per-stage'}<HgGlyph
        name="wonder-stage"
      />{:else if effect.kind === 'points-if-wonder-complete'}<HgGlyph
        name="wonder-complete"
      />{/if}
    <span class="vp">{formatNumber(n)}</span>
  {:else if effect.kind === 'play-last-card'}
    <HgGlyph name="play-last-card" />
  {:else if effect.kind === 'build-from-discard'}
    <HgGlyph name="build-from-discard" />
  {:else if effect.kind === 'free-build'}
    <HgGlyph name={effect.when === 'first-colour' ? 'free-first-colour' : 'free-age-boundary'} />
  {/if}
</span>

<style>
  .fx {
    display: inline-flex;
    align-items: center;
    gap: 0.1em;
    padding: 0.1em 0.2em;
    border-radius: 0.3em;
    background: var(--card-paper);
    color: var(--card-ink);
    font-weight: 700;
    line-height: 1;
  }
  .sep {
    opacity: 0.6;
    font-weight: 400;
  }
  .coin {
    display: inline-grid;
    place-items: center;
    min-width: 1.2em;
    height: 1.2em;
    border: 1px solid var(--gold-deep);
    border-radius: 9999px;
    background: var(--gold);
    color: var(--card-ink);
  }
  .vp {
    display: inline-grid;
    place-items: center;
    min-width: 1.3em;
    height: 1.2em;
    border: 1px solid var(--vp-edge);
    border-radius: 0.3em;
    background: var(--vp);
    color: var(--paper-bright);
  }
</style>
