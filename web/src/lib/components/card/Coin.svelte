<script lang="ts">
  // A coin with its value printed on it: a gold disc with the numeral, the
  // way printed costs and coin effects show money. Sized in em, so it
  // follows the card's font size. Decorative: the card's ARIA name says the
  // amount in words. `tip` gives it a tooltip of its own; inside a cost row
  // or an effect glyph the surrounding group carries the tooltip instead.
  import { formatSigned } from '$lib/i18n';
  import { tooltip, type TooltipContent } from '$lib/ui/tooltip.svelte';

  interface Props {
    n: number;
    /** A minus sign before the numeral (coins lost; U+2212 through formatSigned). */
    loss?: boolean;
    /** Its own tooltip, when no surrounding element has one. */
    tip?: TooltipContent;
    class?: string;
  }

  let { n, loss = false, tip, class: className = '' }: Props = $props();
</script>

<span
  class="inline-grid size-[1.25em] shrink-0 place-items-center rounded-full border border-(color:--gold-deep) bg-gold
    leading-none font-bold text-card-ink tabular-nums {className}"
  data-min-font="10"
  aria-hidden="true"
  {@attach tooltip(tip)}>{formatSigned(loss ? -n : n)}</span
>
