<script lang="ts">
  // The frame every Haftgane concept diagram shares. Full: a sunken panel
  // with the caption sentence under the picture, for the rules reference.
  // Compact: headed "For example:" when the picture holds numbers, with the
  // caption only for screen readers, small enough (about 220 px wide) for a
  // tooltip bubble whose own text says the rule. Duel's frame looks the
  // same (components/rules/DiagramFrame.svelte); it is not imported, so
  // that Duel's rules chunks stay as they are.
  import type { Snippet } from 'svelte';
  import { ht } from '../i18n/messages.svelte';

  interface Props {
    name: string;
    caption: string;
    compact?: boolean;
    /** Head the compact form with "For example:" (its numbers are not the game's). */
    example?: boolean;
    children: Snippet;
  }

  let { name, caption, compact = false, example = false, children }: Props = $props();
</script>

<figure
  class="flex flex-col rounded-panel border border-rule bg-panel-sunken text-fg {compact
    ? 'max-w-[13.5rem] gap-1 p-1.5'
    : 'gap-2 p-3'}"
  data-rules-diagram={name}
  data-compact={compact ? '' : undefined}
>
  {#if compact && example}
    <span class="text-[11px] leading-tight font-semibold text-fg-muted">
      {ht('hg.rules.diagram.example')}
    </span>
  {/if}
  {@render children()}
  <figcaption class={compact ? 'sr-only' : 'text-xs leading-snug text-fg-muted'}>
    {caption}
  </figcaption>
</figure>
