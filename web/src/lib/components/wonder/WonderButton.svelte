<script lang="ts">
  // The phone strip's one wonder control: the wonder glyph and how many of
  // the seat's wonders are built out of those still in play ("1/4"), from
  // view/wonders.wonderTally. It opens that seat's wonder sheet. When the
  // view's choices let the seat build a wonder now (buildableWonders), it
  // glows in the accent with a dot on its corner, so a player sees the
  // chance without opening anything; the spoken label says so too.
  //
  // A leaf component with plain props. Its real 44 px box gives the strip
  // a clear tap target without invisible margins over nearby chips.
  //
  // Board hints (M10.HS4, `hint` or from the page's game context with
  // `seat`): an hourglass once only one more wonder can be built and this
  // seat still has one to build, and a ring when the seat could afford a
  // wonder at today's prices though no choice of the view builds it now
  // (the opponent's half of the ready mark). Its tooltip is the spoken
  // label: the tally, whether a wonder can be built now (the accent dot), and
  // the hints (the hourglass, the ready ring); it links the rules' wonders
  // section.
  import { hintHide } from '$lib/hints/local.svelte';
  import { joinPhrases, formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { wonderButtonHint, type WonderButtonHint } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import { tryGameContext } from '../choices/room';
  import SeventhWonderDiagram from '../rules/SeventhWonderDiagram.svelte';
  import Hourglass from './Hourglass.svelte';

  interface Props {
    built: number;
    total: number;
    /** At least one wonder can be built now (from the view's choices). */
    buildable?: boolean;
    /** Whose wonders, for the spoken label (a display name). */
    name: string;
    onclick?: () => void;
    /** Whose wonders, for the hints from the game context. */
    seat?: 0 | 1;
    /** The board hints; else from the game context. */
    hint?: WonderButtonHint;
    class?: string;
  }

  let {
    built,
    total,
    buildable = false,
    name,
    onclick,
    seat,
    hint,
    class: className = ''
  }: Props = $props();

  const ctx = tryGameContext();
  const bh = $derived.by((): WonderButtonHint | undefined => {
    if (hint) return hint;
    const view = ctx?.session.view;
    const screen = sessionHintScreen(ctx?.session, hintHide.value);
    if (!view || !screen || seat === undefined) return undefined;
    return wonderButtonHint(view, seat, screen);
  });

  const label = $derived(
    joinPhrases([
      t('board.wonders.button', { name, built, total }),
      buildable ? t('board.wonders.buildableNow') : '',
      bh?.tip ?? ''
    ])
  );
</script>

{#snippet seventhDiagram()}<SeventhWonderDiagram compact />{/snippet}

<button
  type="button"
  class="wonder-button relative flex h-11 min-w-11 shrink-0 items-center gap-1 rounded-control border px-2 text-[13px] font-semibold
    tabular-nums transition-colors short:px-1.5
    {buildable
    ? 'border-accent bg-accent-soft text-accent-soft-fg ring-2 ring-accent/45'
    : 'border-rule-strong bg-panel text-fg hover:border-accent'} {className}"
  aria-label={label}
  data-control="wonders"
  data-tut="wonders"
  data-fx-wonders
  data-buildable={buildable ? '' : undefined}
  data-hourglass={bh?.hourglass ? '' : undefined}
  data-ready={bh && bh.ready > 0 && !buildable ? bh.ready : undefined}
  {@attach tooltip({ text: label, rulesId: 'wonders', diagram: seventhDiagram })}
  {onclick}
>
  <Icon name="wonder" class="text-base" />
  <span data-must-fit>{formatNumber(built)}/{formatNumber(total)}</span>
  {#if bh?.hourglass}
    <Hourglass class="size-3.5 text-cinnabar-ink" />
  {/if}
  {#if bh && bh.ready > 0 && !buildable}
    <span
      class="absolute -end-1 -top-1 size-2.5 rounded-full border-2 border-fg-muted bg-panel"
      aria-hidden="true"
      data-ready-dot
    ></span>
  {/if}
  {#if buildable}
    <span
      class="absolute -end-1 -top-1 size-2.5 rounded-full border border-panel bg-accent"
      aria-hidden="true"
    ></span>
  {/if}
</button>
