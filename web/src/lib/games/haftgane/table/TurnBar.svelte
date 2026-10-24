<script lang="ts">
  // The turn bar: what the table waits for, the seat's pending pick with
  // Confirm or Change, and the chosen count (one dot per seat the point
  // asks, filled once it confirmed). The dots and count come from the
  // frame's point. A confirmed pick shows a check that pops in, and a dot
  // fills smoothly as its seat confirms; with Animation off both are
  // simply there.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';

  interface Props {
    /** The status line, already translated. */
    text: string;
    chosen: { done: number; of: number };
    pending: { key: string; confirm: boolean } | null;
    /** The pending pick in words, already translated. */
    pendingLabel?: string;
    canAct: boolean;
    watching?: boolean;
    onconfirm: () => void;
    onwithdraw: () => void;
  }

  let {
    text,
    chosen,
    pending,
    pendingLabel = '',
    canAct,
    watching = false,
    onconfirm,
    onwithdraw
  }: Props = $props();

  const chosenTip = $derived(ht('hg.screen.bar.chosen.tip', { done: chosen.done, of: chosen.of }));
  const dots = $derived(Array.from({ length: chosen.of }, (_, i) => i < chosen.done));
</script>

<div class="bar" data-turn-bar>
  <div class="text">
    <span class="line" role="status" data-watching={watching ? '' : undefined}>{text}</span>
    {#if pending}
      <span
        class="pending"
        data-pending={pending.key}
        data-pending-confirm={String(pending.confirm)}
        dir="auto"
        >{#if pending.confirm}<span class="check" aria-hidden="true">✓</span
          >{/if}{pendingLabel}</span
      >
    {/if}
  </div>
  {#if pending && !pending.confirm}
    <button
      type="button"
      class="act primary"
      disabled={!canAct}
      onclick={onconfirm}
      data-act="confirm"
    >
      {ht('hg.screen.confirm')}
    </button>
  {:else if pending?.confirm}
    <button type="button" class="act" disabled={!canAct} onclick={onwithdraw} data-act="withdraw">
      {ht('hg.screen.change')}
    </button>
  {/if}
  {#if chosen.of > 0}
    <span
      class="chosen"
      data-chosen={chosen.done}
      role="img"
      aria-label={chosenTip}
      {@attach tooltip({ text: chosenTip })}
    >
      <span class="dots" aria-hidden="true">
        {#each dots as on, i (i)}<i class:on></i>{/each}
      </span>
      <span dir="ltr"
        >{ht('hg.screen.bar.chosen', {
          done: formatNumber(chosen.done),
          of: formatNumber(chosen.of)
        })}</span
      >
    </span>
  {/if}
</div>

<style>
  .bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-height: 3rem;
    padding: 0.25rem 0.75rem;
    border-block-start: 1px solid var(--rule);
    background: var(--panel);
  }
  .text {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-width: 0;
  }
  .line {
    overflow: hidden;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pending {
    overflow: hidden;
    color: var(--fg-muted);
    font-size: 0.8125rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .act {
    flex-shrink: 0;
    min-width: 2.75rem;
    min-height: 2.75rem;
    padding-inline: 1rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel-sunken);
    color: var(--fg);
    font: inherit;
    font-weight: 700;
    cursor: pointer;
  }
  .primary {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
  }
  .act:disabled {
    cursor: default;
    opacity: 0.6;
  }
  .chosen {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    gap: 0.3rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .dots {
    display: inline-flex;
    gap: 0.15rem;
  }
  .dots i {
    width: 0.55rem;
    height: 0.55rem;
    border: 1.5px solid var(--afford-ok);
    border-radius: 9999px;
    transition: background-color var(--dur-base) ease-out;
  }
  .dots i.on {
    background: var(--afford-ok);
  }
  .check {
    display: inline-block;
    margin-inline-end: 0.25rem;
    color: var(--afford-ok);
    font-weight: 700;
    animation: check-pop var(--dur-slow) cubic-bezier(0.2, 1.4, 0.4, 1) both;
  }
  @keyframes check-pop {
    from {
      opacity: 0;
      scale: 0.4;
    }
  }
</style>
