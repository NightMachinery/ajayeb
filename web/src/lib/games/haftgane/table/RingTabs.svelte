<script lang="ts">
  // The city sheet's tabs: one per seat in ring order (the anchor first, then
  // round the table from its left neighbour to its right), each with the
  // seat's crest and name. They follow the page direction, as the sheet
  // does; the arrow keys move between them, as tabs do.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import SeatCrest from './SeatCrest.svelte';
  import type { RingTab } from './city';

  interface Props {
    tabs: readonly RingTab[];
    active: number;
    /** The tab list's accessible name. */
    label: string;
    botTip: string;
    /** The id of the panel the tabs control. */
    panelId: string;
    onselect: (seat: number) => void;
  }

  let { tabs, active, label, botTip, panelId, onselect }: Props = $props();

  function step(event: KeyboardEvent): void {
    const keys: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1, Home: 0, End: 0 };
    if (!(event.key in keys)) return;
    event.preventDefault();
    const rtl = getComputedStyle(event.currentTarget as Element).direction === 'rtl';
    const at = tabs.findIndex((t) => t.seat === active);
    const n = tabs.length;
    const next =
      event.key === 'Home'
        ? 0
        : event.key === 'End'
          ? n - 1
          : (at + (rtl ? -keys[event.key]! : keys[event.key]!) + n) % n;
    const seat = tabs[next]!.seat;
    onselect(seat);
    const list = event.currentTarget as HTMLElement;
    queueMicrotask(() => list.querySelector<HTMLElement>(`[data-seat-tab="${seat}"]`)?.focus());
  }
</script>

<div class="tabs" role="tablist" aria-label={label} tabindex="-1" onkeydown={step}>
  {#each tabs as tab (tab.seat)}
    {@const selected = tab.seat === active}
    <button
      type="button"
      role="tab"
      class="tab"
      aria-selected={selected}
      aria-controls={panelId}
      tabindex={selected ? 0 : -1}
      data-act="city-tab:{tab.seat}"
      data-seat-tab={tab.seat}
      onclick={() => onselect(tab.seat)}
      {@attach tooltip({ text: tab.tip, actionable: true })}
    >
      <SeatCrest seat={tab.seat} tip={tab.tip} bot={tab.bot} {botTip} />
      <span class="name" dir="auto">{tab.name}</span>
    </button>
  {/each}
</div>

<style>
  .tabs {
    display: flex;
    gap: 0.15rem;
    min-width: 0;
    padding-block-end: 0.4rem;
    border-block-end: 1px solid var(--rule);
  }
  .tab {
    display: flex;
    flex: 1 1 0;
    flex-direction: column;
    align-items: center;
    gap: 0.15rem;
    min-width: 44px;
    min-height: 44px;
    padding: 0.3rem 0.1rem;
    border: 0;
    border-block-end: 3px solid transparent;
    border-radius: var(--shape-control) var(--shape-control) 0 0;
    background: none;
    color: var(--fg-muted);
    font: inherit;
    cursor: pointer;
  }
  .tab[aria-selected='true'] {
    border-block-end-color: var(--lapis);
    background: var(--panel-sunken);
    color: var(--fg);
  }
  .name {
    max-width: 100%;
    overflow: hidden;
    font-size: 0.75rem;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
