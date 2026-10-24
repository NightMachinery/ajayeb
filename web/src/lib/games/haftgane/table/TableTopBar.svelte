<script lang="ts">
  // The table's top bar: the game menu, the Age and turn (or the side
  // point, or the end), the pass chip, the turn recap's button (with a dot
  // while the newest recap is unseen), the count of spectators (it opens
  // the game menu, which lists them), and Undo while the table allows it
  // (it goes for good once a second human sits down). The log button comes
  // with its slice. On a narrow phone the gaps tighten and the pass chip
  // keeps its arrow alone, so the Age and turn keep their room; should
  // they still not fit, the turn wraps under the Age rather than run into
  // the chip. Undo is Lucide's `undo`, not Duel's `undo-2`, the
  // spectators' icon is `users-round`, not Home's `users`, and the recap's
  // is `clipboard-list`, not Duel's `eye` or `scroll-text`, so the games
  // share no icon.
  import { ageLabel, formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import ClipboardList from '@lucide/svelte/icons/clipboard-list';
  import EllipsisVertical from '@lucide/svelte/icons/ellipsis-vertical';
  import Undo from '@lucide/svelte/icons/undo';
  import UsersRound from '@lucide/svelte/icons/users-round';
  import { ht } from '../i18n/messages.svelte';
  import PassChip from './PassChip.svelte';

  interface Props {
    phase: string;
    age: number;
    turn: number;
    /** Turns in an Age: six. The view sends the turn; the rules fix the count. */
    turns: number;
    pass: string;
    /** Who receives the viewer's hand this Age. */
    passTo: string;
    /** This page just saw a new Age dealt: the pass chip sweeps once. */
    passFresh?: boolean;
    finished: boolean;
    onmenu: () => void;
    /** Takes back the viewer's last turn; absent while undo is not allowed. */
    onundo?: (() => void) | null;
    /** How many people watch; the count shows when there are any. */
    spectators?: number;
    onspectators?: () => void;
    /** Opens the newest turn recap; absent before the first turn. */
    onrecap?: (() => void) | null;
    /** The newest recap has not been opened yet. */
    recapUnseen?: boolean;
  }

  let {
    phase,
    age,
    turn,
    turns,
    pass,
    passTo,
    passFresh = false,
    finished,
    onmenu,
    onundo = null,
    spectators = 0,
    onspectators,
    onrecap = null,
    recapUnseen = false
  }: Props = $props();

  const turnTip = $derived(ht('hg.screen.turn.tip', { turn, of: turns }));
</script>

<header class="top" data-top-bar>
  <button
    type="button"
    class="icon"
    aria-label={ht('hg.screen.menu')}
    data-act="table-menu"
    onclick={onmenu}
    {@attach tooltip({ text: ht('hg.screen.menu') })}
  >
    <EllipsisVertical aria-hidden="true" />
  </button>
  <div class="where">
    {#if finished || phase === 'over'}
      <span class="age">{ht('hg.screen.over')}</span>
    {:else if phase === 'sides' || age === 0}
      <span class="age" {@attach tooltip({ text: ht('hg.screen.sides.tip') })}
        >{ht('hg.screen.sides')}</span
      >
    {:else}
      <span class="age">{ht('hg.screen.age', { age: ageLabel(age) })}</span>
      {#if phase === 'last-card'}<span class="turn">{ht('hg.special.extra')}</span>{:else}
        <span class="turn" dir="ltr" {@attach tooltip({ text: turnTip })}
          >{ht('hg.screen.turn', { turn, of: turns })}</span
        >
      {/if}
    {/if}
  </div>
  {#if !finished && age > 0}
    <PassChip {pass} to={passTo} fresh={passFresh} />
  {/if}
  {#if onrecap}
    {@const tip = recapUnseen ? ht('hg.recap.open.unseen') : ht('hg.recap.open')}
    <button
      type="button"
      class="icon recap"
      aria-label={tip}
      data-act="recap"
      data-unseen={recapUnseen ? '' : undefined}
      onclick={onrecap}
      {@attach tooltip({ text: tip, actionable: true })}
    >
      <ClipboardList aria-hidden="true" />
      {#if recapUnseen}<span class="dot" aria-hidden="true"></span>{/if}
    </button>
  {/if}
  {#if spectators > 0 && onspectators}
    {@const tip = ht('hg.admin.spectators.open', { count: spectators })}
    <button
      type="button"
      class="icon watchers"
      aria-label={tip}
      data-act="spectators"
      data-spectators={spectators}
      onclick={onspectators}
      {@attach tooltip({ text: tip, actionable: true })}
    >
      <UsersRound aria-hidden="true" />
      <span dir="ltr">{formatNumber(spectators)}</span>
    </button>
  {/if}
  {#if onundo}
    <button
      type="button"
      class="icon"
      aria-label={ht('hg.admin.undo')}
      data-act="undo"
      onclick={onundo}
      {@attach tooltip({ text: ht('hg.admin.undo.tip'), actionable: true })}
    >
      <Undo aria-hidden="true" />
    </button>
  {/if}
</header>

<style>
  .top {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-height: 2.75rem;
    padding-inline: 0.25rem 0.75rem;
    border-block-end: 1px solid var(--rule);
    background: var(--panel);
  }
  .icon {
    display: grid;
    flex-shrink: 0;
    place-items: center;
    width: 2.75rem;
    height: 2.75rem;
    border: 0;
    border-radius: var(--shape-control);
    background: transparent;
    color: var(--fg);
    cursor: pointer;
  }
  .watchers {
    display: inline-flex;
    gap: 0.15rem;
    width: auto;
    min-width: 2.75rem;
    padding-inline: 0.3rem;
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
  }
  .recap {
    position: relative;
  }
  .dot {
    position: absolute;
    inset-block-start: 0.55rem;
    inset-inline-end: 0.45rem;
    width: 0.55rem;
    height: 0.55rem;
    border: 1.5px solid var(--panel);
    border-radius: 9999px;
    background: var(--accent);
  }
  .watchers :global(svg) {
    width: 1.1rem;
    height: 1.1rem;
  }
  .where {
    display: flex;
    flex: 1 1 auto;
    flex-wrap: wrap;
    align-items: baseline;
    column-gap: 0.4rem;
    min-width: 0;
    overflow-x: clip;
    line-height: 1.2;
  }
  .age {
    font-family: var(--type-display);
    font-size: 1.35rem;
    font-weight: 600;
    white-space: nowrap;
  }
  .turn {
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  @media (max-width: 26.99rem) {
    .top {
      gap: 0.25rem;
      padding-inline-end: 0.25rem;
    }
  }
</style>
