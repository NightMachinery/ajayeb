<script lang="ts">
  // The table's top bar: the game menu, the Age and turn (or the side
  // point, or the end), the pass chip, and Undo while the table allows it
  // (it goes for good once a second human sits down). The recap and log
  // buttons come with their slices (M18.reveal-and-recap). Undo is
  // Lucide's `undo`, not Duel's `undo-2`, so the two games share no icon.
  import { ageLabel } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import EllipsisVertical from '@lucide/svelte/icons/ellipsis-vertical';
  import Undo from '@lucide/svelte/icons/undo';
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
    finished: boolean;
    onmenu: () => void;
    /** Takes back the viewer's last turn; absent while undo is not allowed. */
    onundo?: (() => void) | null;
  }

  let { phase, age, turn, turns, pass, passTo, finished, onmenu, onundo = null }: Props = $props();

  const turnTip = $derived(ht('hg.screen.turn.tip', { turn, of: turns }));
</script>

<header class="top">
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
    <PassChip {pass} to={passTo} />
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
  .where {
    display: flex;
    flex: 1 1 auto;
    align-items: baseline;
    gap: 0.4rem;
    min-width: 0;
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
  }
</style>
