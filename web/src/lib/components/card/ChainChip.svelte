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

<span
  class="inline-grid size-[1.3em] shrink-0 place-items-center rounded-[0.35em] border
    {direction === 'in'
    ? 'border-(color:--gold-deep) bg-gold text-card-ink'
    : 'border-(color:--gold-deep) bg-paper text-card-ink'} {className}"
  style:--i2="var(--gold-deep)"
  {title}
  aria-hidden="true"
>
  <Icon name={`chain-${chain}`} class="text-[0.95em]" />
</span>
