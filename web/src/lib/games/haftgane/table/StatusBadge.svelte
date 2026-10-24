<script lang="ts">
  // Where a seat stands on the current decision, told apart by shape as well
  // as colour: a tick (confirmed), three dots (a bot choosing), a dashed
  // ring (a human choosing) and the word "away". Nothing while the seat has
  // nothing to decide.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import type { SeatStatus } from './model';

  interface Props {
    status: SeatStatus;
    /** The seat's display name. */
    name: string;
  }

  let { status, name }: Props = $props();

  const tip = $derived(
    status === 'confirmed'
      ? ht('hg.screen.status.confirmed', { name })
      : status === 'bot'
        ? ht('hg.screen.status.bot', { name })
        : status === 'choosing'
          ? ht('hg.screen.status.choosing', { name })
          : status === 'away'
            ? ht('hg.screen.status.away', { name })
            : ''
  );
</script>

{#if status !== 'idle'}
  <span
    class="badge {status}"
    data-status={status}
    role="img"
    aria-label={tip}
    {@attach tooltip({ text: tip })}
  >
    {#if status === 'confirmed'}
      <HgGlyph name="badge-check" />
    {:else if status === 'bot'}
      <span class="dots" aria-hidden="true"><i></i><i></i><i></i></span>
    {:else if status === 'away'}
      <span class="word">{ht('hg.screen.status.awayWord')}</span>
    {/if}
  </span>
{/if}

<style>
  .badge {
    display: inline-grid;
    flex-shrink: 0;
    place-items: center;
    min-width: 1.5rem;
    height: 1.5rem;
    border-radius: 9999px;
    font-size: 0.95rem;
    line-height: 1;
  }
  .confirmed {
    background: var(--afford-ok);
    color: var(--paper-bright);
  }
  .bot {
    padding-inline: 0.3rem;
    background: var(--accent-soft);
    color: var(--accent-soft-fg);
  }
  .choosing {
    border: 2px dashed var(--fg-muted);
  }
  .away {
    padding-inline: 0.4rem;
    border: 1px solid var(--rule-strong);
    background: var(--panel-sunken);
    color: var(--fg-muted);
    font-size: 0.6875rem;
    font-weight: 600;
  }
  .dots {
    display: inline-flex;
    gap: 0.15rem;
  }
  .dots i {
    width: 0.3rem;
    height: 0.3rem;
    border-radius: 9999px;
    background: currentColor;
  }
</style>
