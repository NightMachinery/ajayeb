<script lang="ts">
  // The seven science symbol slots of one seat, with its distinct count out
  // of 6 and a danger ring at 5 (lib/view/threats.scienceDanger; none once
  // the game has ended). Pairs (count >= 2 of the same symbol) get a small
  // ring of their own, since a pair is what lets the player take a progress
  // token. The `pill` form is the phone strip's: the green building's
  // glyph (the science category) and the count out of 6, nothing more; the
  // city sheet has the slots.
  //
  // Hints (M10.HS4, symbol-status and science-reach): with the science
  // hints, each slot shows where the seat stands on its symbol (the
  // server's PlayerHintsView.symbols): paired (the ring), live (one card,
  // and a dot under it: the other may still come), owned (one card, no
  // dot: the other can no longer come), open (a dotted outline: none yet,
  // and a card may still come) or dead (a hairline strike: none can come).
  // Each slot's tooltip names its two cards and where they are, and the
  // counter's tooltip says what the seat still needs. The glow is the
  // caller's (lib/view/threats.ts scienceGlow): calm turns the 5/6 warning
  // into a hairline when the missing symbols can no longer come, strong
  // means the seat can win by science with one action now. With `seat`
  // and no `hints`, the hints come from the page's game context.
  //
  // The counter (and the pill) always has a tooltip: the count, the warning
  // and the reach, with the rules' science diagram (compact); it and every
  // slot link the rules' science section.
  import { hintHide } from '$lib/hints/local.svelte';
  import { joinPhrases, formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import type { IconName } from '$lib/icons/names';
  import { scienceHints, type ScienceHints } from '$lib/view/board-hints';
  import { sentences } from '$lib/view/tips';
  import { sessionHintScreen } from '$lib/view/hints';
  import { scienceDanger, type Glow } from '$lib/view/threats';
  import type { ScienceSymbol, ScienceView } from '$lib/wire';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { tryGameContext } from '../choices/room';
  import ScienceDiagram from '../rules/ScienceDiagram.svelte';

  interface Props {
    science: ScienceView;
    compact?: boolean;
    /** The phone strip's one-chip form. */
    pill?: boolean;
    /** The game has ended: no danger tint. */
    ended?: boolean;
    /** Whose science this is, for the hints from the game context. */
    seat?: 0 | 1;
    /** The science hints (view/board-hints.ts scienceHints); else from the context. */
    hints?: ScienceHints;
    /** The warning level (view/threats.ts scienceGlow); else the 5/6 rule. */
    glow?: Glow;
    class?: string;
  }

  let {
    science,
    compact = false,
    pill = false,
    ended = false,
    seat,
    hints,
    glow,
    class: className = ''
  }: Props = $props();

  const ctx = tryGameContext();
  const sh = $derived.by((): ScienceHints | undefined => {
    if (hints || ended) return hints;
    const view = ctx?.session.view;
    const screen = sessionHintScreen(ctx?.session, hintHide.value);
    if (!view || !screen || seat === undefined) return undefined;
    return scienceHints(view, seat, screen.families, screen.names);
  });

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

  const level = $derived<Glow>(
    ended ? 'none' : (glow ?? (scienceDanger(science, ended) ? 'soft' : 'none'))
  );
  const danger = $derived(level === 'soft' || level === 'strong');
  const reach = $derived(sh?.reach ?? '');
  const label = $derived(
    joinPhrases([
      t('board.science.label', { count: science.distinct }),
      danger ? t('board.science.danger') : '',
      reach
    ])
  );
  const counterTip = $derived({
    text: sentences([
      t('board.science.label', { count: science.distinct }),
      danger ? t('board.science.danger') : '',
      level === 'calm' ? t('hints.glow.calm') : '',
      reach
    ]),
    diagram,
    rulesId: 'science' as const
  });

  function slotTip(symbol: ScienceSymbol): string {
    return sh?.slots[symbol]?.tip ?? t(labelKey(symbol));
  }
</script>

{#snippet diagram()}
  <ScienceDiagram compact />
{/snippet}

{#if pill}
  <span
    class="science-pill inline-flex h-5 shrink-0 items-center gap-0.5 rounded-control border px-1 text-[11px] font-semibold tabular-nums
      {danger
      ? 'border-cinnabar bg-cinnabar/15 text-cinnabar'
      : level === 'calm'
        ? 'border-dashed border-rule-strong bg-panel text-fg'
        : 'border-rule bg-panel text-fg'} {level === 'strong'
      ? 'ring-2 ring-cinnabar/45'
      : ''} {className}"
    role="img"
    aria-label={label}
    data-science={science.distinct}
    data-fx-science-pill
    data-danger={danger ? '' : undefined}
    data-glow={level}
    {@attach tooltip(counterTip)}
  >
    <Icon name="category-green-sm" class="text-xs {danger ? '' : 'text-card-green-ink'}" />
    <span data-must-fit>{formatNumber(science.distinct)}/{formatNumber(6)}</span>
  </span>
{:else}
  <div
    class="science-tracker flex flex-wrap items-center {compact
      ? 'gap-1'
      : 'gap-2 lg:gap-1.5'} {className}"
    role="group"
    aria-label={label}
  >
    <!-- At lg the desktop side panel is narrow: tighter gaps keep the
         counter on the slots' line. -->
    <ul class="flex {compact ? 'gap-0.5' : 'gap-1 lg:gap-0.5'}">
      {#each SYMBOLS as symbol (symbol)}
        {@const count = countOf(symbol)}
        {@const state = sh?.slots[symbol]?.state}
        <li
          class="slot relative grid place-items-center rounded-control border {compact
            ? 'size-5'
            : 'size-7'} {count >= 2
            ? 'border-gild bg-accent-soft text-accent-soft-fg'
            : count === 1
              ? 'border-rule-strong text-fg'
              : state === 'open'
                ? 'border-dotted border-fg-muted text-fg-muted/70'
                : 'border-rule/40 text-fg-muted/40'}"
          data-symbol={symbol}
          data-fx-science={symbol}
          data-symbol-state={state}
          {@attach tooltip({ text: slotTip(symbol), rulesId: 'science' })}
        >
          <!-- The glyph is 12 px (compact) or 14 px: both at or under the
             14 px where the small-grid (-sm) variant reads better than the
             scaled-down md one (art/vector/STYLE.md). -->
          <Icon name={`${SYMBOL_ICON[symbol]}-sm`} class={compact ? 'text-xs' : 'text-sm'} />
          {#if state === 'live'}
            <span
              class="live-dot absolute bottom-0.5 size-[3px] rounded-full bg-fg"
              aria-hidden="true"
            ></span>
          {:else if state === 'dead' && count === 0}
            <span
              class="dead-strike absolute inset-x-0.5 top-1/2 h-px bg-fg-muted/70"
              aria-hidden="true"
            ></span>
          {/if}
        </li>
      {/each}
    </ul>
    <span
      class="rounded-control px-1.5 py-0.5 {compact ? 'text-[11px]' : 'text-xs'} font-semibold
      tabular-nums {danger
        ? 'bg-cinnabar/15 text-cinnabar'
        : level === 'calm'
          ? 'border border-dashed border-rule-strong text-fg-muted'
          : 'text-fg-muted'} {level === 'strong' ? 'ring-2 ring-cinnabar/45' : ''}"
      data-must-fit
      data-glow={level}
      data-science-counter
      {@attach tooltip(counterTip)}
    >
      {formatNumber(science.distinct)}/{formatNumber(6)}
    </span>
  </div>
{/if}
