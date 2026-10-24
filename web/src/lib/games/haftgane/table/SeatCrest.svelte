<script lang="ts">
  // A seat's crest: its glyph in a ring of the seat colour (theme.css,
  // art/vector/specs/haftgane-seat-crests.md), with a small bot mark when a
  // bot plays the seat. Crests differ by shape, so colour only helps.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    seat: number;
    /** Name and position, already translated. */
    tip: string;
    bot?: boolean;
    botTip?: string;
    size?: 'sm' | 'md';
  }

  let { seat, tip, bot = false, botTip = '', size = 'sm' }: Props = $props();
</script>

<span class="crest {size}" style:--seat="var(--seat-{seat})" data-crest={seat}>
  <span class="ring" role="img" aria-label={tip} {@attach tooltip({ text: tip })}>
    <HgGlyph name="crest-{seat}" />
  </span>
  {#if bot}
    <span class="bot" role="img" aria-label={botTip} {@attach tooltip({ text: botTip })}>
      <svg viewBox="0 0 12 12" aria-hidden="true">
        <rect x="2" y="3.5" width="8" height="6.5" rx="1.5" />
        <circle cx="4.6" cy="6.6" r="0.9" class="eye" />
        <circle cx="7.4" cy="6.6" r="0.9" class="eye" />
        <path d="M6 1.2V3.5" />
      </svg>
    </span>
  {/if}
</span>

<style>
  .crest {
    position: relative;
    display: inline-flex;
    flex-shrink: 0;
  }
  .ring {
    display: grid;
    place-items: center;
    border: 2px solid var(--seat);
    border-radius: 9999px;
    background: color-mix(in oklch, var(--seat) 18%, var(--panel));
    color: var(--seat);
  }
  .sm .ring {
    width: 1.5rem;
    height: 1.5rem;
    font-size: 0.95rem;
  }
  .md .ring {
    width: 2rem;
    height: 2rem;
    font-size: 1.25rem;
  }
  .bot {
    position: absolute;
    inset-block-end: -0.25rem;
    inset-inline-end: -0.3rem;
    display: grid;
    place-items: center;
    width: 0.95rem;
    height: 0.95rem;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    background: var(--panel);
    color: var(--fg-muted);
  }
  .bot svg {
    width: 0.7rem;
    height: 0.7rem;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.2;
  }
  .bot .eye {
    fill: currentColor;
    stroke: none;
  }
</style>
