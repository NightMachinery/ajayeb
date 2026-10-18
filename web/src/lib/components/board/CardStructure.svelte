<script lang="ts">
  // The current age's card structure, sized to fill its container.
  //
  // A ResizeObserver attachment measures the container; fitStructure picks
  // the card size and row step for that box and the age's shape (from the
  // view's own slots), and the structure sets them as --cw, --ch, --step and
  // --gap for the slots to position themselves. The structure is centred in
  // its box and never wider than it, so there is no horizontal scroll at any
  // width. It is a spatial region, so it is dir="ltr" in every locale.
  //
  // A plain leaf component: it takes the view's slots and the display state
  // and reports taps on a slot; it decides nothing about the game.
  import { t } from '$lib/i18n';
  import { breakpoints } from '$lib/layout/breakpoints.svelte';
  import { fitStructure } from '$lib/layout/fit';
  import { badgesDimmed, slotAfford } from '$lib/view/afford';
  import { describeSlot } from '$lib/view/describe';
  import { slotPlace, structureLayout } from '$lib/view/geometry';
  import type { CardTooltipView, Choice, SlotView } from '$lib/wire';
  import StructureSlot from './StructureSlot.svelte';

  interface Props {
    /** The view's structure (View.structure). */
    slots: readonly SlotView[];
    cardTooltips?: readonly CardTooltipView[];
    choices?: readonly Choice[];
    costSeat?: number;
    names?: readonly [string, string];
    you?: number;
    /** The age, 1 to 3 (View.age). */
    age: number;
    /** The seat whose costs the badges show (afford.ts badgeSeat). */
    seat?: number;
    /** The seat to move (View.toMove); badges dim when it is not `seat`. */
    toMove?: number;
    /** The slot index shown as selected. */
    selected?: number;
    /** The slot index of a move on its way to the server. */
    pending?: number;
    /** Wonder-first mode: the slots whose card can go under the chosen wonder. */
    marked?: ReadonlySet<number>;
    /** The slot whose card the bot is about to take (the Presenter's pre-roll). */
    botPick?: number;
    /** The gap between cards in px; by default 4, 6 or 8 by viewport class. */
    gap?: number;
    onselect?: (slot: SlotView) => void;
    class?: string;
  }

  let {
    slots,
    cardTooltips = [],
    choices = [],
    costSeat,
    names,
    you,
    age,
    seat,
    toMove,
    selected,
    pending,
    marked,
    botPick,
    gap,
    onselect,
    class: className = ''
  }: Props = $props();

  let box = $state({ w: 0, h: 0 });

  const layout = $derived(structureLayout(slots));
  const fit = $derived(
    fitStructure(box.w, box.h, layout, { gap: gap ?? breakpoints.structureGap })
  );
  const dim = $derived(seat === undefined || badgesDimmed({ toMove }, seat));
  const ring = $derived(
    !dim && toMove !== undefined ? (toMove === 0 ? ('sun' as const) : ('moon' as const)) : undefined
  );
  const ageNumeral = $derived(age >= 1 && age <= 3 ? t(`card.ageNumeral.${age as 1 | 2 | 3}`) : '');

  /** Measures the container; the callback runs outside any effect. */
  function measure(el: HTMLElement) {
    const observer = new ResizeObserver((entries) => {
      const size = entries[0]?.contentBoxSize?.[0];
      const w = size ? size.inlineSize : el.clientWidth;
      const h = size ? size.blockSize : el.clientHeight;
      if (w !== box.w || h !== box.h) box = { w, h };
    });
    observer.observe(el);
    return () => observer.disconnect();
  }
</script>

<div
  class="card-structure relative flex h-full w-full min-w-0 items-center justify-center overflow-visible {className}"
  role="group"
  aria-label={t('card.structure.label', { age: ageNumeral })}
  dir="ltr"
  data-age={age}
  data-tut="structure"
  data-card-w={fit.w}
  data-card-h={fit.h}
  data-tier={fit.tier}
  {@attach measure}
>
  {#if fit.w > 0}
    <div
      class="relative shrink-0"
      style:width="{fit.width}px"
      style:height="{fit.height}px"
      style:--cw="{fit.w}px"
      style:--ch="{fit.h}px"
      style:--step={fit.step}
      style:--gap="{fit.gap}px"
    >
      {#each slots as slot (slot.index)}
        {@const afford = slotAfford(slot, seat)}
        <StructureSlot
          {slot}
          place={slotPlace(slot, layout)}
          tier={fit.tier}
          width={fit.w}
          label={describeSlot(slot, { seat, waiting: dim, selected: selected === slot.index })}
          {afford}
          tipContext={{
            info: cardTooltips.find((c) => c.card === slot.card),
            names,
            you,
            costSeat,
            actualCost: choices.find((c) => c.kind === 'construct' && c.card === slot.card)?.cost
          }}
          cost={seat === undefined ? undefined : slot.costs?.[seat]}
          {dim}
          {ring}
          selected={selected === slot.index}
          pending={pending === slot.index}
          marked={marked?.has(slot.index) ?? false}
          botPick={botPick === slot.index}
          {onselect}
        />
      {/each}
    </div>
  {/if}
</div>
