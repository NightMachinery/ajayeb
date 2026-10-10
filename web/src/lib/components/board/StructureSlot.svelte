<script lang="ts">
  // One place of the card structure, absolutely positioned inside
  // CardStructure from its row and half-card column. Lower rows stack over
  // higher ones (z-index = row), and the CSS variables the structure sets
  // (--cw, --ch, --step, --gap) turn the slot's card units into px.
  //
  //   face up    a <button data-card="<slug>"> with the CardFace; covered
  //              cards are aria-disabled, takeable ones lift and carry the
  //              affordability badge
  //   face down  a <button>, aria-disabled, with the CardBack of its age
  //   empty      a faint outline where the card was, not focusable
  //
  // The button only reports the tap (onselect); what a tap opens, and which
  // actions exist, is the parent's business and the view's choices.
  import Icon from '$lib/icons/Icon.svelte';
  import type { CardTier } from '$lib/layout/fit';
  import type { Afford } from '$lib/view/afford';
  import type { SlotPlace } from '$lib/view/geometry';
  import type { SlotView } from '$lib/wire';
  import CardBack from '../card/CardBack.svelte';
  import CardFace from '../card/CardFace.svelte';

  interface Props {
    slot: SlotView;
    place: SlotPlace;
    tier: Exclude<CardTier, 'band'>;
    /** The card width in px (the structure's fit). */
    width: number;
    /** The ARIA name (view/describe.ts describeSlot). */
    label: string;
    afford?: Afford;
    /** Not this player's turn: badges dim and no turn ring shows. */
    dim?: boolean;
    /** The colour of the turn ring on takeable cards: the seat to move. */
    ring?: 'sun' | 'moon';
    selected?: boolean;
    /** A move with this card is on its way to the server. */
    pending?: boolean;
    /** Wonder-first mode: this card can go under the chosen wonder. */
    marked?: boolean;
    onselect?: (slot: SlotView) => void;
  }

  let {
    slot,
    place,
    tier,
    width,
    label,
    afford,
    dim = false,
    ring,
    selected = false,
    pending = false,
    marked = false,
    onselect
  }: Props = $props();

  const takeable = $derived(slot.accessible && slot.state === 'faceUp');
</script>

{#if slot.state === 'empty'}
  <span
    class="slot empty absolute start-0 top-0 rounded-[7%/4.5%] border border-dashed border-rule/60"
    style:--x={place.x}
    style:--y={place.y}
    style:z-index="0"
    data-slot={slot.index}
    data-state="empty"
    aria-hidden="true"
  ></span>
{:else}
  <button
    type="button"
    class="slot absolute start-0 top-0 rounded-[7%/4.5%] p-0 text-start"
    class:takeable
    class:covered={!takeable}
    class:selected
    class:pending
    class:marked
    style:--x={place.x}
    style:--y={place.y}
    style:z-index={selected ? 50 : place.z}
    style:--ring={ring ? `var(--${ring})` : 'transparent'}
    data-slot={slot.index}
    data-state={slot.state}
    data-card={slot.state === 'faceUp' ? slot.card : undefined}
    data-accessible={slot.accessible ? '' : undefined}
    data-marked={marked ? '' : undefined}
    aria-label={label}
    aria-disabled={takeable ? undefined : 'true'}
    aria-pressed={selected ? 'true' : undefined}
    onclick={() => onselect?.(slot)}
  >
    {#if slot.state === 'faceUp' && slot.card}
      <CardFace slug={slot.card} {tier} {width} {afford} {dim} />
    {:else}
      <CardBack back={slot.back ?? 'age1'} class="h-full w-full" />
    {/if}
    {#if marked}
      <span
        class="wonder-mark pointer-events-none absolute -end-1.5 -top-1.5 grid size-6 place-items-center rounded-full border-2 border-(color:--gold-deep) bg-gold text-card-ink shadow-sm"
        aria-hidden="true"
      >
        <Icon name="wonder" class="text-[13px]" />
      </span>
    {/if}
  </button>
{/if}

<style>
  .slot {
    width: var(--cw);
    height: var(--ch);
    transform: translate(
      calc(var(--x) * (var(--cw) + var(--gap))),
      calc(var(--y) * var(--step) * var(--ch))
    );
    transition:
      transform var(--dur-fast) var(--curve-out),
      box-shadow var(--dur-fast) var(--curve-out);
    -webkit-touch-callout: none;
  }
  .covered {
    box-shadow: 0 1px 2px oklch(0.24 0.04 260 / 0.25);
  }
  :global([data-theme='dark']) .covered {
    filter: brightness(0.92);
  }
  /* Takeable: lifted 2 px with a shadow, and a soft ring in the colour of the
     seat to move while it is that seat's turn. */
  .takeable {
    transform: translate(
      calc(var(--x) * (var(--cw) + var(--gap))),
      calc(var(--y) * var(--step) * var(--ch) - 2px)
    );
    box-shadow:
      0 0 0 2px var(--ring),
      0 4px 10px -3px oklch(0.2 0.04 260 / 0.45);
    cursor: pointer;
  }
  @media (hover: hover) {
    .takeable:hover {
      transform: translate(
        calc(var(--x) * (var(--cw) + var(--gap))),
        calc(var(--y) * var(--step) * var(--ch) - 4px)
      );
    }
  }
  .selected {
    box-shadow:
      0 0 0 3px var(--accent),
      0 6px 14px -4px oklch(0.2 0.04 260 / 0.5);
  }
  .pending {
    outline: 2px dashed var(--accent);
    outline-offset: 3px;
  }
  /* Wonder-first mode: a gold ring on the cards that can go under the wonder. */
  .marked {
    box-shadow:
      0 0 0 3px var(--gold),
      0 4px 10px -3px oklch(0.2 0.04 260 / 0.45);
  }
</style>
