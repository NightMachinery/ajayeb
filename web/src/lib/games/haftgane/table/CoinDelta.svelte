<script lang="ts">
  // One coin change of a seat in the turn recap: the coin, the signed
  // amount as the event states it ("−2", "+3"), and for a trade the crest
  // of the neighbour on the other side. Payments and gains differ by sign
  // and by the coin's ring, not by colour alone. The tooltip says it in
  // words ("paid Lena 2 coins").
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import type { CoinChange } from './recap';
  import { coinAmount, signedCoins } from './recap-text';

  interface Props {
    change: CoinChange;
    /** The change in words, already translated. */
    tip: string;
  }

  let { change, tip }: Props = $props();

  const amount = $derived(coinAmount(change));
  const other = $derived(change.kind === 'paid' || change.kind === 'received' ? change.seat : null);
</script>

<span
  class="coin"
  class:loss={amount < 0}
  data-coin={change.kind}
  data-coin-amount={amount}
  data-coin-seat={other ?? undefined}
  {@attach tooltip({ text: tip })}
>
  <span class="disc"><HgGlyph name="coin" /></span>
  <span class="value" dir="ltr">{signedCoins(amount)}</span>
  {#if other !== null}
    <span class="who" style:--seat="var(--seat-{other})"><HgGlyph name="crest-{other}" /></span>
  {/if}
</span>

<style>
  .coin {
    display: inline-flex;
    align-items: center;
    gap: 0.15rem;
    padding-block: 0.05rem;
    padding-inline: 0.15rem 0.3rem;
    border: 1px solid var(--rule);
    border-radius: 9999px;
    background: var(--panel-sunken);
    color: var(--fg);
    font-size: 0.8rem;
    font-variant-numeric: tabular-nums;
    font-weight: 600;
    white-space: nowrap;
  }
  .disc {
    display: grid;
    place-items: center;
    color: var(--coin-ink);
    font-size: 0.9rem;
  }
  .loss {
    border-style: dashed;
  }
  .loss .value {
    color: var(--danger);
  }
  .who {
    display: grid;
    place-items: center;
    width: 1rem;
    height: 1rem;
    border: 1.5px solid var(--seat);
    border-radius: 9999px;
    color: var(--seat);
    font-size: 0.65rem;
  }
</style>
