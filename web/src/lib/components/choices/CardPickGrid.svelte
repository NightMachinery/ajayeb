<script lang="ts">
  import { joinPhrases } from '$lib/i18n';
  // A grid of medium cards, optionally selectable: the opponent's cards a
  // destroy choice may hit, or the discard pile. Selecting a card only
  // marks it (the caller's commit button sends the move), so a tile carries
  // data-pick with its choice key, never data-act. A plain leaf: the caller
  // passes the cards and, for a choice, each card's key from the view.
  // `pips` puts a hint pip on a card's top-end corner (the Mausoleum's
  // pool, M10.HS4: view/hints.ts discardPips), as on the board.
  import X from '@lucide/svelte/icons/x';
  import Icon from '$lib/icons/Icon.svelte';
  import { cardName } from '$lib/components/card/effect-text';
  import { describeCard } from '$lib/view/describe';
  import { pipLabel, type CardPip } from '$lib/view/hints';
  import CardFace from '../card/CardFace.svelte';
  import HintPip from '../card/HintPip.svelte';

  interface Props {
    cards: readonly string[];
    /** Card slug to its choice key; a card without one is shown, not picked. */
    keys?: ReadonlyMap<string, string>;
    /** The selected choice key. */
    selected?: string | null;
    onselect?: (key: string) => void;
    /** Danger styling for the selection ring (destroy). */
    danger?: boolean;
    disabled?: boolean;
    label: string;
    /** Hint pips by card slug. */
    pips?: ReadonlyMap<string, CardPip>;
    class?: string;
  }

  let {
    cards,
    keys,
    selected = null,
    onselect,
    danger = false,
    disabled = false,
    label,
    pips,
    class: className = ''
  }: Props = $props();

  function nameOf(slug: string): string {
    const pip = pips?.get(slug);
    return pip ? joinPhrases([describeCard(slug), pipLabel(pip)]) : describeCard(slug);
  }

  const CARD_W = 96;
  const CARD_H = 148;
</script>

<ul
  class="grid grid-cols-[repeat(auto-fill,minmax(96px,1fr))] justify-items-center gap-x-2 gap-y-3 {className}"
  aria-label={label}
>
  {#each cards as slug (slug)}
    {@const key = keys?.get(slug)}
    {@const isSelected = key !== undefined && key === selected}
    <li data-card={slug} data-fx-pile-card={slug}>
      {#if key !== undefined && onselect}
        <button
          type="button"
          class="pick relative block rounded-[8px] transition-transform duration-(--dur-fast) ease-(--curve-out)
            disabled:cursor-not-allowed disabled:opacity-60
            {isSelected ? (danger ? 'is-danger' : 'is-selected') : ''}"
          style:width="{CARD_W}px"
          style:height="{CARD_H}px"
          data-pick={key}
          aria-pressed={isSelected}
          aria-label={nameOf(slug)}
          {disabled}
          onclick={() => onselect(key)}
        >
          <CardFace {slug} tier="medium" width={CARD_W} />
          {#if pips?.get(slug) && !isSelected}
            <HintPip pip={pips.get(slug)!} card={cardName(slug)} />
          {/if}
          {#if isSelected}
            <span
              class="absolute -end-1.5 -top-1.5 grid size-6 place-items-center rounded-full shadow-panel
                {danger ? 'danger-badge' : 'bg-accent text-on-accent'}"
              aria-hidden="true"
            >
              {#if danger}
                <X class="size-4" strokeWidth={3} />
              {:else}
                <Icon name="badge-check" class="text-[15px]" />
              {/if}
            </span>
          {/if}
        </button>
      {:else}
        <div
          class="relative"
          style:width="{CARD_W}px"
          style:height="{CARD_H}px"
          role="img"
          aria-label={nameOf(slug)}
        >
          <CardFace {slug} tier="medium" width={CARD_W} />
          {#if pips?.get(slug)}
            <HintPip pip={pips.get(slug)!} card={cardName(slug)} />
          {/if}
        </div>
      {/if}
    </li>
  {/each}
</ul>

<style>
  .pick {
    outline-offset: 3px;
  }
  .pick:not(:disabled):hover {
    translate: 0 -2px;
  }
  .is-selected {
    box-shadow:
      0 0 0 3px var(--panel),
      0 0 0 6px var(--accent);
  }
  .is-danger {
    box-shadow:
      0 0 0 3px var(--panel),
      0 0 0 6px var(--cinnabar);
  }
  .danger-badge {
    background: var(--cinnabar);
    color: white;
  }
</style>
