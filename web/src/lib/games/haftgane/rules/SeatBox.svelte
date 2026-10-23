<script lang="ts">
  // One player at the table in a concept diagram: a small panel with the
  // seat's label ("You", "Left neighbour") and whatever the diagram puts
  // under it. "You" is drawn in the accent outline so the reader finds
  // their own seat first.
  import type { Snippet } from 'svelte';

  interface Props {
    label: string;
    you?: boolean;
    compact?: boolean;
    children?: Snippet;
  }

  let { label, you = false, compact = false, children }: Props = $props();
</script>

<div
  class="flex min-w-0 flex-col items-center gap-1 rounded-control border bg-panel text-center {you
    ? 'border-accent'
    : 'border-rule'} {compact ? 'px-1 py-0.5' : 'px-1.5 py-1.5'}"
  data-seat={you ? 'you' : 'neighbour'}
>
  <span
    class="label leading-tight font-semibold text-fg {compact ? 'text-[11px]' : 'text-xs'}"
    data-min-font={compact ? '10' : undefined}>{label}</span
  >
  {@render children?.()}
</div>

<style>
  /* Layout that no Duel screen uses lives here, not in Tailwind classes:
     a class would add its rule to the global stylesheet that Home and
     Duel load, and this one loads only with Haftgane's rules. */
  .label {
    overflow-wrap: break-word;
    hyphens: auto;
  }
</style>
