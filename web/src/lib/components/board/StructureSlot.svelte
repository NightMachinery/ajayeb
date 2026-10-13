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
  //
  // Hints (M10.HS2): a face-up card may carry one hint pip on its top-end
  // corner (view/hints.ts picks it from the server's slot.hints), at half
  // strength on a covered card. In wonder-first mode the gold ring alone
  // marks a tuckable card and the corner goes to the opponent's pip, the
  // deny target, when there is one. The pip follows the game's hint
  // families and this screen's local hide; on a room page it reads them
  // from the game context, and the gallery passes `hints` instead.
  //
  // The recap (last-move-trace and price-changed-tick, M10.F3, from the
  // game screen's context or `recap`): until the viewer's move ends, a slot
  // the other side emptied keeps a faint gold trace (data-recap-trace), a
  // card it turned face up carries a small gold dot on its top-start corner
  // (data-recap-revealed), and a card whose price changed says so in its
  // badge's tooltip (data-recap-price). All static.
  import Icon from '$lib/icons/Icon.svelte';
  import type { CardTier } from '$lib/layout/fit';
  import type { Afford } from '$lib/view/afford';
  import type { SlotPlace } from '$lib/view/geometry';
  import type { SlotView } from '$lib/wire';
  import { hintHide } from '$lib/hints/local.svelte';
  import { denyPip, pipLabel, sessionHintScreen, slotPip, type HintScreen } from '$lib/view/hints';
  import { cardName } from '../card/effect-text';
  import CardBack from '../card/CardBack.svelte';
  import CardFace from '../card/CardFace.svelte';
  import HintPip from '../card/HintPip.svelte';
  import { tryGameContext } from '../choices/room';
  import { t } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { getRecapBoard, type RecapBoard } from '$lib/view/recap';

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
    /** The hint display of this screen; on a room page it comes from the game context. */
    hints?: HintScreen;
    /** The recap's board marks; on a room page they come from the game screen. */
    recap?: RecapBoard | null;
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
    hints,
    recap,
    onselect
  }: Props = $props();

  const recapContext = getRecapBoard();
  const board = $derived(recap === undefined ? recapContext() : recap);
  const traced = $derived(!!board?.marks && board.emptied.has(slot.index));
  const fresh = $derived(
    !!board?.marks && slot.state === 'faceUp' && !!slot.card && board.revealed.has(slot.card)
  );
  const priceTip = $derived(
    slot.state === 'faceUp' && slot.card ? board?.cardPrice.get(slot.card) : undefined
  );

  const takeable = $derived(slot.accessible && slot.state === 'faceUp');

  const ctx = tryGameContext();
  const screen = $derived(hints ?? sessionHintScreen(ctx?.session, hintHide.value));
  const pip = $derived(
    screen ? slotPip(slot, screen.perspective, screen.families, screen.names) : undefined
  );
  // Wonder-first mode: only the opponent's pip claims the corner.
  const shownPip = $derived(screen?.boardMarks ? (marked ? denyPip(pip) : pip) : undefined);
  const name = $derived(slot.state === 'faceUp' && slot.card ? cardName(slot.card) : '');
  // The badge's Urbanism bonus and Economy flow (the prices family).
  const badgeCost = $derived.by(() => {
    if (!afford || !screen?.families.has('prices') || !slot.costs) return undefined;
    return slot.costs[screen.perspective.seat];
  });
</script>

{#if slot.state === 'empty' && traced}
  <span
    class="slot empty recap-trace absolute start-0 top-0 rounded-[7%/4.5%] border-2 border-dotted"
    style:--x={place.x}
    style:--y={place.y}
    style:z-index="0"
    data-slot={slot.index}
    data-state="empty"
    data-recap-trace=""
    aria-hidden="true"
    {@attach tooltip({ text: t('hints.recap.emptied') })}
  ></span>
{:else if slot.state === 'empty'}
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
    aria-label={pip ? `${label}. ${pipLabel(pip)}` : label}
    aria-disabled={takeable ? undefined : 'true'}
    aria-pressed={selected ? 'true' : undefined}
    onclick={() => onselect?.(slot)}
  >
    {#if slot.state === 'faceUp' && slot.card}
      <CardFace
        slug={slot.card}
        {tier}
        {width}
        {afford}
        {dim}
        bonus={badgeCost?.chainFree ? badgeCost.urbanismBonus : 0}
        flow={badgeCost?.affordable ? badgeCost.toOpponent : 0}
        flowTo={screen ? screen.names.names[screen.perspective.seat === 0 ? 1 : 0] : ''}
        {priceTip}
      />
    {:else}
      <CardBack back={slot.back ?? 'age1'} class="h-full w-full" />
    {/if}
    {#if fresh}
      <span
        class="recap-spark absolute -start-1 -top-1 size-2.5 rounded-full border-[1.5px] border-(color:--gold-deep) bg-gold"
        data-recap-revealed=""
        {@attach tooltip({ text: t('hints.recap.revealed') })}
      ></span>
    {/if}
    {#if shownPip}
      <HintPip pip={shownPip} card={name} size={tier === 'mini' ? 'sm' : 'md'} faint={!takeable} />
    {:else if marked}
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
    box-shadow: 0 1px 2px var(--card-rest-shadow);
  }
  /* The recap's trace of a slot the other side emptied: a faint gold
     outline, static (the animation wave may animate it). */
  .recap-trace {
    border-color: color-mix(in oklch, var(--gild) 70%, transparent);
    background: color-mix(in oklch, var(--gold) 10%, transparent);
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
      0 4px 10px -3px var(--card-lift-shadow);
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
      0 6px 14px -4px var(--card-raised-shadow);
  }
  .pending {
    outline: 2px dashed var(--accent);
    outline-offset: 3px;
  }
  /* Wonder-first mode: a gold ring on the cards that can go under the wonder. */
  .marked {
    box-shadow:
      0 0 0 3px var(--gold),
      0 4px 10px -3px var(--card-lift-shadow);
  }
</style>
