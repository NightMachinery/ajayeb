<script lang="ts">
  // One printed effect as a compact glyph group for a card's colour band,
  // with a tooltip that says what it does. Drawn from the catalog's codes.
  import { formatNumber } from '$lib/i18n';
  import { hgTooltip } from './tooltip';
  import { ht } from '../i18n/messages.svelte';
  import type { HgRuleSectionId } from '../rules/sections';
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
  const rulesId = $derived<HgRuleSectionId>(
    effect.kind.startsWith('produce')
      ? 'resources'
      : effect.kind === 'trade-discount'
        ? 'trade'
        : effect.kind.startsWith('science')
          ? 'science'
          : effect.kind === 'shields'
            ? 'military'
            : effect.kind === 'free-build'
              ? 'costs'
              : ['play-last-card', 'build-from-discard', 'points-if-wonder-complete'].includes(
                    effect.kind
                  )
                ? 'wonders'
                : 'scoring'
  );
  const scienceSymbols = ['compass', 'gear', 'tablet'] as const;
</script>

<span
  class="fx"
  data-effect={effect.kind}
  role="img"
  aria-label={tip}
  {@attach hgTooltip({ text: tip, rulesId })}
>
  {#if effect.kind === 'produce'}
    {#each copies as i (i)}<ResourceTile res={effect.res?.[0] ?? ''} {tip} />{/each}
  {:else if effect.kind === 'produce-choice'}
    {#each effect.res ?? [] as r, i (r)}{#if i > 0}<span class="sep">/</span>{/if}<ResourceTile
        res={r}
        tip={ht('hg.screen.making.option', {
          resource: ht(`hg.resource.${r}` as 'hg.resource.wood')
        })}
      />{/each}
  {:else if effect.kind === 'trade-discount'}
    {#if effect.scope?.includes('left')}<span
        class="trade-direction"
        dir="ltr"
        role="img"
        aria-label={effectText({ ...effect, scope: ['left'] })}
        {@attach hgTooltip({ text: effectText({ ...effect, scope: ['left'] }), rulesId })}>←</span
      >{/if}
    <span data-effect-symbol="trade" {@attach hgTooltip({ text: tip, rulesId })}
      ><HgGlyph name="trade" /></span
    >
    {#each effect.res ?? [] as r (r)}<ResourceTile
        res={r}
        size="xs"
        tip={effectText({ ...effect, res: [r] })}
        {rulesId}
      />{/each}
    {#if effect.scope?.includes('right')}<span
        class="trade-direction"
        dir="ltr"
        role="img"
        aria-label={effectText({ ...effect, scope: ['right'] })}
        {@attach hgTooltip({ text: effectText({ ...effect, scope: ['right'] }), rulesId })}>→</span
      >{/if}
  {:else if effect.kind === 'shields'}
    {#each copies as i (i)}<HgGlyph name="shield" />{/each}
  {:else if effect.kind === 'science'}
    <HgGlyph name="science-{effect.symbol ?? 'compass'}-sm" />
  {:else if effect.kind === 'science-choice'}
    {#each scienceSymbols as symbol, i (symbol)}
      {#if i > 0}<span class="sep">/</span>{/if}
      {@const text = ht('hg.screen.science.option', { symbol: ht(`hg.science.${symbol}`) })}
      <span
        data-effect-symbol={symbol}
        role="img"
        aria-label={text}
        {@attach hgTooltip({ text, rulesId })}
      >
        <HgGlyph name="science-{symbol}-sm" />
      </span>
    {/each}
  {:else if effect.kind === 'coins' || effect.kind === 'coins-per-card' || effect.kind === 'coins-per-stage'}
    {#if effect.kind === 'coins-per-card'}{#each effect.colours ?? [] as c (c)}<span
          data-effect-symbol={c}
          role="img"
          aria-label={effectText({ ...effect, colours: [c] })}
          {@attach hgTooltip({ text: effectText({ ...effect, colours: [c] }), rulesId })}
        >
          <HgGlyph name="category-{c}-sm" />
        </span>{/each}{:else if effect.kind === 'coins-per-stage'}<span
        data-effect-symbol="stage"
        {@attach hgTooltip({ text: tip, rulesId })}><HgGlyph name="wonder-stage" /></span
      >{/if}
    <span class="coin" data-effect-symbol="coins" {@attach hgTooltip({ text: tip, rulesId })}
      >{formatNumber(n)}</span
    >
  {:else if effect.kind === 'points' || effect.kind === 'points-per-card' || effect.kind === 'points-per-stage' || effect.kind === 'points-if-wonder-complete'}
    {#if effect.kind === 'points-per-card'}{#each effect.colours ?? [] as c (c)}<span
          data-effect-symbol={c}
          role="img"
          aria-label={effectText({ ...effect, colours: [c] })}
          {@attach hgTooltip({ text: effectText({ ...effect, colours: [c] }), rulesId })}
        >
          <HgGlyph name="category-{c}-sm" />
        </span>{/each}{:else if effect.kind === 'points-per-stage'}<span
        data-effect-symbol="stage"
        {@attach hgTooltip({ text: tip, rulesId })}><HgGlyph name="wonder-stage" /></span
      >{:else if effect.kind === 'points-if-wonder-complete'}<span
        data-effect-symbol="wonder"
        {@attach hgTooltip({ text: tip, rulesId })}><HgGlyph name="wonder-complete" /></span
      >{/if}
    <span class="vp" data-effect-symbol="points" {@attach hgTooltip({ text: tip, rulesId })}
      >{formatNumber(n)}</span
    >
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
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    gap: 0.1em;
    max-width: 100%;
    min-width: 0;
    padding: 0.1em 0.2em;
    border-radius: 0.3em;
    background: var(--card-paper);
    color: var(--card-ink);
    font-weight: 700;
    line-height: 1;
  }
  /* Card typography follows its container. Fixed rem tiles would spill
     beyond a phone card even when the surrounding effects can wrap. */
  .fx :global([data-res]) {
    width: 1.1em;
    height: 1.1em;
    font-size: inherit;
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
