<script lang="ts">
  // The discard pile as medium cards with colour filter chips, latest age
  // first. The view sends the pile as a set in card id order, and ids follow
  // the ages, so "latest age first" is as close to "newest first" as the
  // view allows. Used read-only by the discard sheet, and selectable by the
  // Mausoleum's build-from-discard choice.
  //
  // Hints (M10.HS4, mausoleum-pool-hint): a card that a Mausoleum could
  // rebuild to some effect carries its pip, as on the board (view/hints.ts
  // discardPips over the server's View.hints.discard), from the page's game
  // context unless `pips` is given.
  import { cardBySlug } from '$lib/catalog';
  import { hintHide } from '$lib/hints/local.svelte';
  import { discardPips, sessionHintScreen, type CardPip } from '$lib/view/hints';
  import { tryGameContext } from './room';
  import Icon from '$lib/icons/Icon.svelte';
  import { formatNumber, t } from '$lib/i18n';
  import type { Color } from '$lib/wire';
  import { categoryIcon } from '../card/card-style';
  import CardPickGrid from './CardPickGrid.svelte';

  interface Props {
    cards: readonly string[];
    keys?: ReadonlyMap<string, string>;
    selected?: string | null;
    onselect?: (key: string) => void;
    disabled?: boolean;
    /** Hint pips by card slug; else from the game context. */
    pips?: ReadonlyMap<string, CardPip>;
  }

  let { cards, keys, selected = null, onselect, disabled = false, pips }: Props = $props();

  const ctx = tryGameContext();
  const cardPips = $derived.by((): ReadonlyMap<string, CardPip> | undefined => {
    if (pips) return pips;
    const view = ctx?.session.view;
    return view ? discardPips(view, sessionHintScreen(ctx?.session, hintHide.value)) : undefined;
  });
  const COLOR_ORDER: readonly Color[] = [
    'brown',
    'grey',
    'blue',
    'green',
    'yellow',
    'red',
    'purple'
  ];

  let filter = $state<Color | 'all'>('all');

  const sorted = $derived(
    [...cards].sort((a, b) => {
      const ca = cardBySlug(a);
      const cb = cardBySlug(b);
      return (cb?.age ?? 0) - (ca?.age ?? 0) || (ca?.id ?? 0) - (cb?.id ?? 0);
    })
  );

  // Cards per colour present in the pile, in colour order.
  const counts = $derived(
    COLOR_ORDER.map(
      (color) => [color, cards.filter((slug) => cardBySlug(slug)?.color === color).length] as const
    ).filter(([, n]) => n > 0)
  );
  const colors = $derived(counts.map(([color]) => color));

  function countOf(color: Color): number {
    return counts.find(([c]) => c === color)?.[1] ?? 0;
  }

  // A filter whose colour left the pile falls back to everything.
  const active = $derived(filter !== 'all' && colors.includes(filter) ? filter : 'all');
  const shown = $derived(
    active === 'all' ? sorted : sorted.filter((slug) => cardBySlug(slug)?.color === active)
  );
</script>

{#if cards.length === 0}
  <p class="py-6 text-center text-sm text-fg-muted">{t('choices.discard.empty')}</p>
{:else}
  <div class="flex flex-col gap-3">
    {#if colors.length > 1}
      <div class="flex flex-wrap gap-1.5" role="group" aria-label={t('choices.discard.filter')}>
        <button
          type="button"
          class="chip {active === 'all' ? 'chip-on' : ''}"
          aria-pressed={active === 'all'}
          onclick={() => (filter = 'all')}
        >
          {t('choices.discard.all')}
          <span class="tabular-nums opacity-75">{formatNumber(cards.length)}</span>
        </button>
        {#each colors as color (color)}
          <button
            type="button"
            class="chip {active === color ? 'chip-on' : ''}"
            aria-pressed={active === color}
            onclick={() => (filter = color)}
          >
            <span class="inline-flex" style:color="var(--card-{color}-ink)"
              ><Icon name={categoryIcon(color)} class="text-[15px]" /></span
            >
            {t(`board.color.${color}`)}
            <span class="tabular-nums opacity-75">{formatNumber(countOf(color))}</span>
          </button>
        {/each}
      </div>
    {/if}
    <p class="text-xs text-fg-muted">
      {t('choices.discard.count', { count: shown.length })} · {t('choices.discard.order')}
    </p>
    <CardPickGrid
      cards={shown}
      {keys}
      {selected}
      {onselect}
      {disabled}
      pips={cardPips}
      label={t('choices.discard.sheetTitle')}
    />
  </div>
{/if}

<style>
  .chip {
    display: inline-flex;
    min-height: 2.75rem;
    align-items: center;
    gap: 0.35rem;
    border-radius: 999px;
    border: 1px solid var(--rule);
    background: var(--panel);
    padding-inline: 0.75rem;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--fg);
  }
  .chip:hover {
    background: var(--panel-sunken);
  }
  .chip-on {
    border-color: var(--accent);
    background: var(--accent-soft);
    color: var(--accent-soft-fg);
  }
</style>
