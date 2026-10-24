<script lang="ts">
  // One printed effect as a compact glyph group for a card's colour band,
  // with a tooltip that says what it does. Drawn from the catalog's codes.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { HgEffect } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import ResourceTile from './ResourceTile.svelte';

  interface Props {
    effect: HgEffect;
  }

  let { effect }: Props = $props();

  const res = (r: string) => ht(`hg.resource.${r}` as 'hg.resource.wood');
  function orList(rs: readonly string[] | undefined): string {
    const names = (rs ?? []).map(res);
    return names.length === 0 ? '' : names.reduce((a, b) => ht('hg.screen.trade.or', { a, b }));
  }
  function where(scope: readonly string[] | undefined): string {
    const s = scope ?? [];
    const self = s.includes('self');
    const others = s.includes('left') || s.includes('right');
    return self && others
      ? ht('hg.screen.effect.where.all')
      : others
        ? ht('hg.screen.effect.where.neighbours')
        : ht('hg.screen.effect.where.self');
  }
  function colours(cs: readonly string[] | undefined): string {
    const names = (cs ?? []).map((c) =>
      ht(`hg.screen.colourName.${c}` as 'hg.screen.colourName.red')
    );
    return names.length === 0 ? '' : names.reduce((a, b) => ht('hg.screen.trade.or', { a, b }));
  }

  const n = $derived(effect.n ?? 1);
  /** Up to three copies of a repeated symbol. */
  const copies = $derived(Array.from({ length: Math.min(n, 3) }, (_, i) => i));
  const tip = $derived.by(() => {
    switch (effect.kind) {
      case 'produce':
        return ht('hg.screen.effect.produce', { count: n, resource: res(effect.res?.[0] ?? '') });
      case 'produce-choice':
        return ht('hg.screen.effect.produce-choice', { options: orList(effect.res) });
      case 'trade-discount': {
        const s = effect.scope ?? [];
        const who =
          s.includes('left') && s.includes('right')
            ? ht('hg.screen.effect.who.both')
            : s.includes('left')
              ? ht('hg.screen.effect.who.left')
              : ht('hg.screen.effect.who.right');
        return ht('hg.screen.effect.trade-discount', { options: orList(effect.res), who });
      }
      case 'shields':
        return ht('hg.screen.effect.shields', { count: n });
      case 'science':
        return ht('hg.screen.effect.science', {
          symbol: ht(`hg.science.${effect.symbol ?? 'compass'}`)
        });
      case 'science-choice':
        return ht('hg.screen.effect.science-choice');
      case 'coins':
        return ht('hg.screen.effect.coins', { count: n });
      case 'coins-per-card':
        return ht('hg.screen.effect.coins-per-card', {
          count: n,
          colour: colours(effect.colours),
          where: where(effect.scope)
        });
      case 'coins-per-stage':
        return ht('hg.screen.effect.coins-per-stage', { count: n, where: where(effect.scope) });
      case 'points':
        return ht('hg.screen.effect.points', { count: n });
      case 'points-per-card':
        return ht('hg.screen.effect.points-per-card', {
          count: n,
          colour: colours(effect.colours),
          where: where(effect.scope)
        });
      case 'points-per-stage':
        return ht('hg.screen.effect.points-per-stage', { count: n, where: where(effect.scope) });
      case 'points-if-wonder-complete':
        return ht('hg.screen.effect.points-if-wonder-complete', { count: n });
      case 'play-last-card':
        return ht('hg.screen.effect.play-last-card');
      case 'build-from-discard':
        return ht('hg.screen.effect.build-from-discard');
      case 'free-build':
        return effect.when === 'first-colour'
          ? ht('hg.screen.effect.free-build.first-colour')
          : effect.when === 'first-turn'
            ? ht('hg.screen.effect.free-build.first-turn')
            : ht('hg.screen.effect.free-build.last-turn');
    }
    return '';
  });
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
