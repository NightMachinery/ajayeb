<script lang="ts">
  // One conflict token as the end-of-Age dialog draws it: the victory
  // pennant with its Age's numeral, the broken defeat pennant, or "=" for a
  // tie, which gives no token. A token's points show only when the server
  // sends them (`value`); the view does not carry them yet, so the Age
  // numeral names the token instead. Shape, never colour alone, tells them
  // apart.
  import { ageLabel, formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    kind: 'victory' | 'defeat' | 'tie';
    age: number;
    /** The spoken form, already translated. */
    tip: string;
    /** The token's points, when the server states them. */
    value?: number;
  }

  let { kind, age, tip, value }: Props = $props();

  const MINUS = '−';
  const shown = $derived(
    value === undefined
      ? ''
      : value > 0
        ? `+${formatNumber(value)}`
        : value < 0
          ? MINUS + formatNumber(-value)
          : formatNumber(0)
  );
</script>

<span
  class="token {kind}"
  data-token={kind}
  role="img"
  aria-label={tip}
  {@attach tooltip({ text: tip })}
>
  {#if kind === 'tie'}
    <span class="eq" aria-hidden="true">=</span>
  {:else}
    <HgGlyph name={kind === 'victory' ? 'conflict-victory' : 'conflict-defeat'} />
    {#if shown}
      <span class="value" dir="ltr">{shown}</span>
    {:else if kind === 'victory'}
      <span class="age">{ageLabel(age)}</span>
    {/if}
  {/if}
</span>

<style>
  .token {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.15rem;
    min-width: 1.9rem;
    height: 1.6rem;
    padding-inline: 0.3rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: 0.4rem;
    background: var(--panel);
    font-size: 0.75rem;
    font-weight: 700;
    line-height: 1;
    white-space: nowrap;
  }
  .token :global(svg) {
    font-size: 1.05rem;
  }
  .victory {
    border-color: var(--afford-ok-edge);
    background: var(--afford-ok);
    color: var(--paper-bright);
  }
  .defeat {
    border-color: var(--cinnabar);
    background: var(--afford-short-bg);
    color: var(--cinnabar-ink);
  }
  .tie {
    border-style: dashed;
    color: var(--fg-muted);
  }
  .eq {
    font-size: 0.95rem;
  }
  .age {
    font-family: var(--type-display);
  }
</style>
