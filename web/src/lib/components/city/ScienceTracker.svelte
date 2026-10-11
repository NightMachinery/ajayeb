<script lang="ts">
  // The seven science symbol slots of one seat, with its distinct count out
  // of 6 and a danger ring at 5 (lib/view/threats.scienceDanger; none once
  // the game has ended). Pairs (count >= 2 of the same symbol) get a small
  // ring of their own, since a pair is what lets the player take a progress
  // token. The `pill` form is the phone strip's: the green building's
  // glyph (the science category) and the count out of 6, nothing more; the
  // city sheet has the slots.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import { scienceDanger } from '$lib/view/threats';
  import type { ScienceSymbol, ScienceView } from '$lib/wire';

  interface Props {
    science: ScienceView;
    compact?: boolean;
    /** The phone strip's one-chip form. */
    pill?: boolean;
    /** The game has ended: no danger tint. */
    ended?: boolean;
    class?: string;
  }

  let {
    science,
    compact = false,
    pill = false,
    ended = false,
    class: className = ''
  }: Props = $props();

  const SYMBOLS = [
    'globe',
    'scales',
    'sundial',
    'mortar',
    'set-square',
    'quill',
    'wheel'
  ] as const satisfies readonly ScienceSymbol[];

  // `as const satisfies` keeps each literal name, so `${SYMBOL_ICON[s]}-sm`
  // types as the seven -sm names rather than as every icon name plus "-sm".
  const SYMBOL_ICON = {
    globe: 'science-globe',
    scales: 'science-scales',
    sundial: 'science-sundial',
    mortar: 'science-mortar',
    'set-square': 'science-set-square',
    quill: 'science-quill',
    wheel: 'science-wheel'
  } as const satisfies Record<ScienceSymbol, IconName>;

  function labelKey(symbol: ScienceSymbol) {
    switch (symbol) {
      case 'globe':
        return 'board.science.globe' as const;
      case 'scales':
        return 'board.science.scales' as const;
      case 'sundial':
        return 'board.science.sundial' as const;
      case 'mortar':
        return 'board.science.mortar' as const;
      case 'set-square':
        return 'board.science.set-square' as const;
      case 'quill':
        return 'board.science.quill' as const;
      case 'wheel':
        return 'board.science.wheel' as const;
    }
  }

  function countOf(symbol: ScienceSymbol): number {
    return science.symbols.find((s) => s.symbol === symbol)?.count ?? 0;
  }

  const danger = $derived(scienceDanger(science, ended));
  const label = $derived(
    [t('board.science.label', { count: science.distinct }), danger ? t('board.science.danger') : '']
      .filter(Boolean)
      .join('. ')
  );
</script>

{#if pill}
  <span
    class="science-pill inline-flex h-5 shrink-0 items-center gap-0.5 rounded-control border px-1 text-[11px] font-semibold tabular-nums
      {danger
      ? 'border-cinnabar bg-cinnabar/15 text-cinnabar'
      : 'border-rule bg-panel text-fg'} {className}"
    role="img"
    aria-label={label}
    data-science={science.distinct}
    data-danger={danger ? '' : undefined}
  >
    <Icon name="category-green-sm" class="text-xs {danger ? '' : 'text-card-green-ink'}" />
    <span data-must-fit>{science.distinct}/6</span>
  </span>
{:else}
  <div
    class="science-tracker flex flex-wrap items-center {compact ? 'gap-1' : 'gap-2'} {className}"
    role="group"
    aria-label={t('board.science.label', { count: science.distinct })}
  >
    <ul class="flex {compact ? 'gap-0.5' : 'gap-1'}">
      {#each SYMBOLS as symbol (symbol)}
        {@const count = countOf(symbol)}
        <li
          class="grid place-items-center rounded-control border {compact
            ? 'size-5'
            : 'size-7'} {count >= 2
            ? 'border-gild bg-accent-soft text-accent-soft-fg'
            : count === 1
              ? 'border-rule-strong text-fg'
              : 'border-rule/40 text-fg-muted/40'}"
          title={t(labelKey(symbol))}
        >
          <!-- The glyph is 12 px (compact) or 14 px: both at or under the
             14 px where the small-grid (-sm) variant reads better than the
             scaled-down md one (art/vector/STYLE.md). -->
          <Icon name={`${SYMBOL_ICON[symbol]}-sm`} class={compact ? 'text-xs' : 'text-sm'} />
        </li>
      {/each}
    </ul>
    <span
      class="rounded-control px-1.5 py-0.5 {compact ? 'text-[11px]' : 'text-xs'} font-semibold
      tabular-nums {danger ? 'bg-cinnabar/15 text-cinnabar' : 'text-fg-muted'}"
      data-must-fit
      title={danger ? t('board.science.danger') : undefined}
    >
      {science.distinct}/6
    </span>
  </div>
{/if}
