<script lang="ts">
  // A coin with its value printed on it: a gold disc with the numeral, the
  // way printed costs and coin effects show money. Sized in em, so it
  // follows the card's font size. Decorative: the card's ARIA name says the
  // amount in words.
  import { formatNumber } from '$lib/i18n';

  interface Props {
    n: number;
    /** A minus sign before the numeral (coins lost). */
    loss?: boolean;
    class?: string;
  }

  let { n, loss = false, class: className = '' }: Props = $props();
</script>

<span
  class="inline-grid size-[1.25em] shrink-0 place-items-center rounded-full border border-(color:--gold-deep) bg-gold
    leading-none font-bold text-card-ink tabular-nums {className}"
  data-min-font="10"
  aria-hidden="true">{loss ? '−' : ''}{formatNumber(n)}</span
>
