<script lang="ts">
  // The top bar's pass chip: which way hands travel this Age, with the
  // drawn pass-hand arrow mirrored for the left, and the receiving
  // neighbour named in its tooltip. On a narrow phone it keeps the arrow
  // alone, since its label and tooltip say the direction in words. When
  // this page sees a new Age dealt (`fresh`), the arrow sweeps once the way
  // hands now travel; with Animation off it simply stands there.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    pass: string;
    /** The display name of the neighbour who receives the viewer's hand. */
    to: string;
    /** A new Age was just dealt: sweep the arrow once. */
    fresh?: boolean;
  }

  let { pass, to, fresh = false }: Props = $props();
  const tip = $derived(
    pass === 'left'
      ? ht('hg.screen.pass.tip.left', { name: to })
      : ht('hg.screen.pass.tip.right', { name: to })
  );
</script>

{#if pass === 'left' || pass === 'right'}
  <span
    class="chip"
    data-pass={pass}
    data-fresh={fresh ? '' : undefined}
    role="img"
    aria-label={tip}
    {@attach tooltip({ text: tip })}
  >
    <span class="arrow" dir="ltr"><HgGlyph name="pass-hand" flip={pass === 'left'} /></span>
    <span class="word"
      >{pass === 'left' ? ht('hg.screen.pass.left') : ht('hg.screen.pass.right')}</span
    >
  </span>
{/if}

<style>
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    height: 2rem;
    padding-inline: 0.65rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: 9999px;
    color: var(--lapis);
    font-weight: 600;
    white-space: nowrap;
  }
  .chip :global(svg) {
    font-size: 1.15rem;
  }
  .arrow {
    display: inline-flex;
  }
  /* The sweep starts on the side hands come from. */
  .chip[data-fresh][data-pass='right'] .arrow {
    animation: pass-sweep-right calc(var(--dur-slow) * 2) ease-out both;
  }
  .chip[data-fresh][data-pass='left'] .arrow {
    animation: pass-sweep-left calc(var(--dur-slow) * 2) ease-out both;
  }
  @keyframes pass-sweep-right {
    from {
      opacity: 0;
      transform: translateX(-0.6rem);
    }
  }
  @keyframes pass-sweep-left {
    from {
      opacity: 0;
      transform: translateX(0.6rem);
    }
  }
  @media (max-width: 26.99rem) {
    .chip {
      padding-inline: 0.5rem;
    }
    .word {
      display: none;
    }
  }
</style>
