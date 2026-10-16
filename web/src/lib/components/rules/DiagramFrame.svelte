<script lang="ts">
  // The frame every concept diagram shares. Full: a sunken panel with the
  // caption sentence under the picture, for the rules reference. Compact:
  // no frame and the caption only for screen readers, small enough (about
  // 220 by 120 CSS px) for a tooltip bubble, whose own text says the rule.
  import type { Snippet } from 'svelte';

  interface Props {
    name: string;
    caption: string;
    compact?: boolean;
    children: Snippet;
  }

  let { name, caption, compact = false, children }: Props = $props();
</script>

<figure
  class={compact
    ? 'flex max-w-[220px] flex-col gap-1.5 text-fg'
    : 'flex flex-col gap-2 rounded-panel border border-rule bg-panel-sunken p-3 text-fg'}
  data-rules-diagram={name}
  data-compact={compact ? '' : undefined}
>
  {@render children()}
  <figcaption class={compact ? 'sr-only' : 'text-xs leading-snug text-fg-muted'}>
    {caption}
  </figcaption>
</figure>
