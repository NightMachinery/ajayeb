<script lang="ts">
  // The hint sections of a sheet (the hints design, M10.HS2): "For you"
  // (or the seat's name), then the opponent's lines, outlined in their
  // crest colour, each line a fact the server computed (view/hints.ts
  // cardSections, discardLines, destroyLines). A line that is a board pip's
  // fact carries the pip's glyph (the hint-crown and hint-pair icons, M10.F2;
  // the line's text names the science symbol). Muted, small, and absent when there is
  // nothing to say. A plain leaf: it takes the sections as props.
  import Icon from '$lib/icons/Icon.svelte';
  import type { HintSection } from '$lib/view/hints';

  interface Props {
    sections: readonly HintSection[];
    class?: string;
  }

  let { sections, class: className = '' }: Props = $props();
</script>

{#if sections.length > 0}
  <div class="flex flex-col gap-2 {className}" data-hint-sections>
    {#each sections as section (`${section.seat}:${section.title}`)}
      <section
        class="hint-section border-s-[3px] ps-2.5"
        data-hint-section={section.side}
        data-hint-seat={section.seat}
        data-style={section.style}
        data-tone={section.tone}
      >
        <h3 class="text-xs font-semibold tracking-wide text-fg-muted uppercase" dir="auto">
          {section.title}
        </h3>
        <ul class="mt-0.5 flex flex-col gap-0.5">
          {#each section.lines as line (line.id)}
            <li class="flex items-start gap-1.5 text-sm text-fg" data-hint-line={line.id}>
              {#if line.code}
                <span
                  class="glyph mt-[0.2em] grid size-[1.1em] shrink-0 place-items-center"
                  aria-hidden="true"
                >
                  {#if line.code === 'win'}
                    <Icon name="hint-crown" class="size-[1.1em]" />
                  {:else if line.code === 'pair'}
                    <Icon name="hint-pair" class="size-[1.1em]" />
                  {:else}
                    <Icon name="coin-loss" class="size-[1.1em]" />
                  {/if}
                </span>
              {/if}
              <span class="min-w-0" dir="auto">{line.text}</span>
            </li>
          {/each}
        </ul>
      </section>
    {/each}
  </div>
{/if}

<style>
  /* The section's start edge carries the side: gold for the viewer's own
     facts, the crest colour for the rest; the opponent's is dashed, the
     outlined form of its pips. */
  .hint-section {
    border-color: var(--gold);
  }
  .hint-section[data-tone='sun'] {
    border-color: var(--sun);
  }
  .hint-section[data-tone='moon'] {
    border-color: var(--moon);
  }
  .hint-section[data-style='outline'] {
    border-inline-start-style: dashed;
  }
  .glyph {
    color: var(--gold-deep);
  }
  [data-tone='sun'] .glyph {
    color: color-mix(in oklch, var(--sun) 70%, var(--fg));
  }
  [data-tone='moon'] .glyph {
    color: color-mix(in oklch, var(--moon) 70%, var(--fg));
  }
</style>
