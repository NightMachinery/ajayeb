<script lang="ts">
  // What building a hand card takes, from the server's choices (model.ts
  // afford()): free with its reason, the cheapest offered total (with the
  // trade arrows when it buys from a neighbour), already built, or blocked
  // with the reason. Nothing shows for a card offered at no cost.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import type { Afford } from './model';

  interface Props {
    afford: Afford;
  }

  let { afford }: Props = $props();

  const tip = $derived.by(() => {
    switch (afford.kind) {
      case 'free':
        return ht(`hg.screen.afford.free.${afford.via}` as 'hg.screen.afford.free.chain');
      case 'paid':
        return afford.trade
          ? ht('hg.screen.afford.trade', { count: afford.total })
          : ht('hg.screen.afford.coins', { count: afford.total });
      case 'built':
        return ht('hg.screen.afford.built');
      case 'blocked':
        return afford.reason === 'insufficient-coins'
          ? ht('hg.screen.afford.insufficient-coins', { count: afford.shortBy })
          : ht('hg.screen.afford.cannot-cover');
    }
    return '';
  });
</script>

{#if afford.kind !== 'none'}
  <span
    class="badge {afford.kind}"
    class:trade={afford.kind === 'paid' && afford.trade}
    data-afford={afford.kind}
    role="img"
    aria-label={tip}
    {@attach tooltip({ text: tip })}
  >
    {#if afford.kind === 'free'}
      <HgGlyph
        name={afford.via === 'chain'
          ? 'badge-chain'
          : afford.via === 'discard'
            ? 'build-from-discard'
            : 'wonder'}
      /><span>{ht('hg.screen.afford.freeWord')}</span>
    {:else if afford.kind === 'paid'}
      {#if afford.trade}<HgGlyph name="trade" />{/if}
      <span class="coin">{formatNumber(afford.total)}</span>
    {:else if afford.kind === 'built'}
      <HgGlyph name="badge-lock" /><span>{ht('hg.screen.afford.builtWord')}</span>
    {:else}
      <HgGlyph name="badge-lock" />
    {/if}
  </span>
{/if}

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 0.2em;
    min-height: 1.6em;
    padding: 0.1em 0.4em;
    border: 1.5px solid var(--rule-strong);
    border-radius: 9999px;
    background: var(--card-paper);
    color: var(--card-ink);
    font-size: 0.85em;
    font-weight: 700;
    line-height: 1;
    white-space: nowrap;
  }
  .free {
    border-color: var(--afford-ok-edge);
    background: var(--afford-ok);
    color: var(--paper-bright);
  }
  .trade {
    border-color: var(--afford-trade-edge);
  }
  .built {
    border-style: dashed;
  }
  .blocked {
    border-style: dashed;
    border-color: var(--cinnabar);
    background: var(--afford-short-bg);
    color: var(--cinnabar-ink);
  }
  .coin {
    display: inline-grid;
    place-items: center;
    min-width: 1.3em;
    height: 1.3em;
    border: 1px solid var(--gold-deep);
    border-radius: 9999px;
    background: var(--gold);
    color: var(--card-ink);
  }
</style>
