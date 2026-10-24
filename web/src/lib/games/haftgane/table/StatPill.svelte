<script lang="ts">
  // One tally pill: points (the laurel tablet), coins, shields, or the sum
  // of a city's conflict tokens, as Duel's Tally draws them. The value is
  // one the server sent; the tooltip is the spoken form.
  import { formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import HgGlyph from './HgGlyph.svelte';

  interface Props {
    kind: 'points' | 'coins' | 'shields' | 'conflict';
    value: number;
    tip: string;
    /** xs drops the pill's chrome: a glyph and a number, for the far chips. */
    size?: 'xs' | 'sm' | 'md';
  }

  let { kind, value, tip, size = 'sm' }: Props = $props();

  const MINUS = '−';
  const shown = $derived(
    kind === 'conflict'
      ? value > 0
        ? `+${formatNumber(value)}`
        : value < 0
          ? MINUS + formatNumber(-value)
          : formatNumber(0)
      : formatNumber(value)
  );
  const glyph = $derived(
    kind === 'points'
      ? 'vp-laurel'
      : kind === 'coins'
        ? 'coin'
        : kind === 'shields'
          ? 'shield'
          : 'conflict-victory'
  );
</script>

<span
  class="pill {kind} {size}"
  data-stat={kind}
  role="img"
  aria-label={tip}
  {@attach tooltip({ text: tip })}
>
  <HgGlyph name={glyph} />
  <span dir="ltr">{shown}</span>
</span>

<style>
  .pill {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    gap: 0.2em;
    border: 1px solid var(--rule-strong);
    border-radius: 9999px;
    font-weight: 700;
    line-height: 1;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .sm {
    height: 1.25rem;
    padding-inline: 0.35rem;
    font-size: 0.8125rem;
  }
  .md {
    height: 1.625rem;
    padding-inline: 0.5rem;
    font-size: 0.9375rem;
  }
  .points {
    border-color: var(--vp-edge);
    border-radius: 0.45em;
    background: var(--vp);
    color: var(--paper-bright);
  }
  .coins {
    border-color: var(--gold-deep);
    background: var(--gold);
    color: var(--card-ink);
  }
  .coins :global(svg) {
    color: var(--gold-deep);
  }
  .shields {
    background: var(--panel);
    color: var(--fg);
  }
  .shields :global(svg) {
    color: var(--cinnabar);
  }
  .conflict {
    border-radius: 0.45em;
    background: var(--panel-sunken);
    color: var(--fg);
  }
  .pill.xs {
    gap: 0.1em;
    height: 1rem;
    padding: 0;
    border: 0;
    background: none;
    color: var(--fg);
    font-size: 0.75rem;
  }
  .xs.points :global(svg) {
    color: var(--vp);
  }
</style>
