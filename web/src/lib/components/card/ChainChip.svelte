<script lang="ts">
  // A chain symbol. "in" (on the cost row): this card is free for a player
  // whose city shows the symbol; a gold chip. "out" (on the effect band):
  // this card shows the symbol and makes a later card free; an outlined
  // chip. The two differ by fill as well as position, so they read in
  // greyscale. Decorative unless given a label.
  import Icon from '$lib/icons/Icon.svelte';
  import { t } from '$lib/i18n';
  import type { Chain } from '$lib/wire';
  import { chainName } from './effect-text';

  interface Props {
    chain: Chain;
    direction: 'in' | 'out';
    /** Give the chip a title (hover text) naming the symbol. */
    titled?: boolean;
    class?: string;
  }

  let { chain, direction, titled = false, class: className = '' }: Props = $props();

  const title = $derived(
    titled
      ? direction === 'in'
        ? t('card.chainInShort', { symbol: chainName(chain) })
        : chainName(chain)
      : undefined
  );
</script>

<!-- The chip is 1.3 em with the glyph at 0.95 em, but never under 14 px
     with a 12 px glyph: on a mini card (a 10 px font) 0.95 em is 9.5 px,
     too small for the mask, lyre, amphora and barrel to read. -->
<span
  class="inline-grid size-[max(1.3em,14px)] shrink-0 place-items-center rounded-[0.35em] border
    {direction === 'in'
    ? 'border-(color:--gold-deep) bg-gold text-card-ink'
    : 'border-(color:--gold-deep) bg-paper text-card-ink'} {className}"
  style:--i2="var(--gold-deep)"
  {title}
  aria-hidden="true"
>
  <Icon name={`chain-${chain}`} class="text-[length:max(0.95em,12px)]" />
</span>
