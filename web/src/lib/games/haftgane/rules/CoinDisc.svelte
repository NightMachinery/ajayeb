<script lang="ts">
  // A coin with its value, for the concept diagrams: the gold disc of the
  // board's coins (components/card/Coin.svelte draws the same look; it is
  // not imported, so that Duel's chunks stay as they are). Decorative: the
  // diagram's caption says the amount in words.
  import { formatNumber } from '$lib/i18n';

  interface Props {
    n: number;
    class?: string;
  }

  let { n, class: className = '' }: Props = $props();
</script>

<span
  class="inline-grid h-[1.25em] min-w-[1.25em] shrink-0 place-items-center rounded-full border border-(color:--gold-deep) bg-gold px-[0.15em] leading-none font-bold text-card-ink tabular-nums {className}"
  aria-hidden="true">{formatNumber(n)}</span
>
