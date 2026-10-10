<script lang="ts">
  // The discard pile, read-only (the board's discard button opens it): every
  // discarded card as a medium card, latest age first, with colour filter
  // chips. The Mausoleum's choice uses the same pile, selectable, in its own
  // panel (choices/DiscardPickChoice.svelte).
  import { t } from '$lib/i18n';
  import Sheet from '../shell/Sheet.svelte';
  import DiscardPile from '../choices/DiscardPile.svelte';

  interface Props {
    open?: boolean;
    /** The discard pile (view.discard). */
    cards: readonly string[];
    titleId?: string;
  }

  let { open = $bindable(false), cards, titleId = 'discard-sheet-title' }: Props = $props();
</script>

<Sheet bind:open {titleId}>
  <div class="flex flex-col gap-3 px-4 pt-1 pb-2" data-discard-sheet>
    <h2 id={titleId} class="pt-1 font-display text-xl text-fg">
      {t('choices.discard.sheetTitle')}
    </h2>
    <DiscardPile {cards} />
  </div>
</Sheet>
