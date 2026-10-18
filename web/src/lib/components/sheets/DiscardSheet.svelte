<script lang="ts">
  // The discard pile, read-only (the board's discard button opens it): every
  // discarded card as a medium card, in server arrival order, with colour filter
  // chips. The Mausoleum's choice uses the same pile, selectable, in its own
  // panel (choices/DiscardPickChoice.svelte).
  //
  // Hints (M10.HS2, mausoleum-pool-hint): while a Mausoleum is open, what
  // its owner could rebuild from the pile, one short section per card that
  // has something to say, from the server's view.hints.discard.
  import { t } from '$lib/i18n';
  import { hintHide } from '$lib/hints/local.svelte';
  import {
    discardSeats,
    discardSections,
    poolTitle,
    sessionHintScreen,
    type HintScreen
  } from '$lib/view/hints';
  import type { CardTipContext } from '$lib/view/tips';
  import type { View } from '$lib/wire';
  import HintLines from '../game/HintLines.svelte';
  import { tryGameContext } from '../choices/room';
  import Sheet from '../shell/Sheet.svelte';
  import DiscardPile from '../choices/DiscardPile.svelte';

  interface Props {
    open?: boolean;
    /** The discard pile (view.discard). */
    cards: readonly string[];
    tipContexts?: ReadonlyMap<string, CardTipContext>;
    titleId?: string;
    /** The view and hint display, when there is no game context (the gallery). */
    hintView?: Pick<View, 'hints'>;
    hints?: HintScreen;
  }

  let {
    open = $bindable(false),
    cards,
    tipContexts,
    titleId = 'discard-sheet-title',
    hintView,
    hints
  }: Props = $props();

  const ctx = tryGameContext();
  const screen = $derived(hints ?? sessionHintScreen(ctx?.session, hintHide.value));
  const source = $derived(hintView ?? ctx?.session.view ?? undefined);
  const pools = $derived(
    screen && source
      ? discardSeats(source)
          .map((seat) => ({
            seat,
            title: poolTitle(seat, screen.names),
            sections: discardSections(source, cards, seat, screen)
          }))
          .filter((p) => p.sections.length > 0)
      : []
  );
</script>

<Sheet bind:open {titleId}>
  <div class="flex flex-col gap-3 px-4 pt-1 pb-2" data-discard-sheet>
    <h2 id={titleId} class="pt-1 font-display text-xl text-fg">
      {t('choices.discard.sheetTitle')}
    </h2>
    <DiscardPile {cards} {tipContexts} />
    {#each pools as pool (pool.seat)}
      <section class="flex flex-col gap-1.5" data-hint-pool={pool.seat}>
        <h3 class="font-display text-base text-fg">{pool.title}</h3>
        <HintLines sections={pool.sections} />
      </section>
    {/each}
  </div>
</Sheet>
