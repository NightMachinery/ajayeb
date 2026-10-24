<script lang="ts">
  // The pass cue on a neighbour panel's inner bottom corner: an arrow from
  // that neighbour toward the viewer's strip when hands come from them, or
  // from the strip toward them when hands go to them. The ring is drawn
  // left to right in every locale (D52), so the arrow is spatial.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';

  interface Props {
    /** Which neighbour's panel carries it. */
    side: 'left' | 'right';
    /** The Age's pass direction: hands go to the left or the right neighbour. */
    pass: string;
    /** That neighbour's display name. */
    name: string;
  }

  let { side, pass, name }: Props = $props();

  const outgoing = $derived(pass === side);
  // The arrow is drawn pointing up and to the right; each case turns it.
  const turn = $derived(side === 'left' ? (outgoing ? 270 : 90) : outgoing ? 0 : 180);
  const tip = $derived(
    outgoing ? ht('hg.screen.pass.to', { name }) : ht('hg.screen.pass.from', { name })
  );
</script>

{#if pass === 'left' || pass === 'right'}
  <span
    class="mark {side}"
    data-pass-mark={outgoing ? 'to' : 'from'}
    role="img"
    aria-label={tip}
    {@attach tooltip({ text: tip })}
  >
    <svg viewBox="0 0 16 16" aria-hidden="true" style:transform="rotate({turn}deg)">
      <path d="M4.5 11.5L11.5 4.5M6 4.5H11.5V10" />
    </svg>
  </span>
{/if}

<style>
  .mark {
    position: absolute;
    inset-block-end: -0.7rem;
    display: grid;
    place-items: center;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 9999px;
    background: var(--lapis);
    color: var(--paper-bright);
    box-shadow: var(--shadow);
  }
  /* The inner corner faces the middle of the table; the board is dir=ltr,
     so the inline ends are the screen's sides. */
  .left {
    inset-inline-end: 0.75rem;
  }
  .right {
    inset-inline-start: 0.75rem;
  }
  svg {
    width: 0.95rem;
    height: 0.95rem;
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
</style>
