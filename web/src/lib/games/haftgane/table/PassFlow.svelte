<script lang="ts">
  // The hand header's pass flow: the crest of the neighbour the hand came
  // from, the hand, and the crest of the neighbour it goes to next, each
  // with its tooltip. Drawn left to right in every locale, like the ring.
  // When this page sees a new Age dealt (`fresh`), its arrows run once
  // from the giver to the receiver; with Animation off they simply stand.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import SeatCrest from './SeatCrest.svelte';

  interface Props {
    from: { seat: number; name: string };
    to: { seat: number; name: string };
    handSize: number;
    /** A new Age was just dealt: run the arrows once. */
    fresh?: boolean;
  }

  let { from, to, handSize, fresh = false }: Props = $props();

  const handTip = $derived(ht('hg.screen.hand.size', { count: handSize }));
</script>

<span class="flow" dir="ltr" data-pass-flow data-fresh={fresh ? '' : undefined}>
  <SeatCrest seat={from.seat} tip={ht('hg.screen.hand.from', { name: from.name })} />
  <span class="arrow" aria-hidden="true">→</span>
  <span class="hand" role="img" aria-label={handTip} {@attach tooltip({ text: handTip })}>
    <HgGlyph name="pass-hand" />
  </span>
  <span class="arrow then" aria-hidden="true">→</span>
  <SeatCrest seat={to.seat} tip={ht('hg.screen.hand.to', { name: to.name })} />
</span>

<style>
  .flow {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
  }
  .arrow {
    display: inline-block;
    color: var(--lapis);
    font-weight: 700;
  }
  .flow[data-fresh] .arrow {
    animation: flow-run calc(var(--dur-slow) * 2) ease-out both;
  }
  .flow[data-fresh] .then {
    animation-delay: var(--dur-base);
  }
  @keyframes flow-run {
    from {
      opacity: 0;
      transform: translateX(-0.5rem);
    }
  }
  .hand {
    display: inline-grid;
    place-items: center;
    color: var(--lapis);
    font-size: 1.2rem;
  }
</style>
