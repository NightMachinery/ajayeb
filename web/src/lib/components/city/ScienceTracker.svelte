<script lang="ts">
  // The seven science symbol slots of one seat, with its distinct count out
  // of 6 and a danger ring at 5 or more (lib/view/threats.scienceDanger).
  // Pairs (count >= 2 of the same symbol) get a small ring of their own,
  // since a pair is what lets the player take a progress token.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import { scienceDanger } from '$lib/view/threats';
  import type { ScienceSymbol, ScienceView } from '$lib/wire';

  interface Props {
    science: ScienceView;
    compact?: boolean;
    class?: string;
  }

  let { science, compact = false, class: className = '' }: Props = $props();

  const SYMBOLS = [
    'globe',
    'scales',
    'sundial',
    'mortar',
    'set-square',
    'quill',
    'wheel'
  ] as const satisfies readonly ScienceSymbol[];

  const SYMBOL_ICON: Record<ScienceSymbol, IconName> = {
    globe: 'science-globe',
    scales: 'science-scales',
    sundial: 'science-sundial',
    mortar: 'science-mortar',
    'set-square': 'science-set-square',
    quill: 'science-quill',
    wheel: 'science-wheel'
  };

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

  const danger = $derived(scienceDanger(science));
</script>

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
        <Icon name={SYMBOL_ICON[symbol]} class={compact ? 'text-xs' : 'text-sm'} />
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
