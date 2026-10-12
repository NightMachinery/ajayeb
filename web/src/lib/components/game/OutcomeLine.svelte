<script lang="ts">
  // The outcome line under one move button (the hints design, M10.HS3):
  // what the move would do, as terms the server decided (view/outcome.ts
  // outcomeTerms). On a phone each term is a glyph and a short number; from
  // sm up it is words. Each term has its own tooltip with the words, and a
  // screen reader hears the whole line once.
  //
  // `blunder` is the confirm step's warning (blunder-confirm): the move
  // hands the opponent a win, or misses one. It stays hidden until the move
  // button beside it is armed (the closest [data-act-group] holds a
  // [data-armed] button), so with Confirm moves off it never shows and
  // never adds a step (owner Q3, A); the card sheet's outcome line and its
  // "After this pick" panel already say the same. A plain leaf.
  import Icon from '$lib/icons/Icon.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { outcomeText, type OutcomeTerm } from '$lib/view/outcome';

  interface Props {
    terms: readonly OutcomeTerm[];
    /** The move's key, for data-outcome. */
    actKey?: string;
    /** The confirm step's warning, shown only while the move is armed. */
    blunder?: string;
    class?: string;
  }

  let { terms, actKey, blunder, class: className = '' }: Props = $props();

  const text = $derived(outcomeText(terms));
</script>

{#if terms.length > 0 || blunder}
  <div class="flex flex-col gap-0.5 {className}" data-outcome={actKey ?? ''}>
    {#if terms.length > 0}
      <p class="flex flex-wrap items-center gap-x-2.5 gap-y-0.5 text-xs text-fg-muted">
        <span class="sr-only">{text}</span>
        {#each terms as term (term.id)}
          <span
            class="term inline-flex items-center gap-1"
            aria-hidden="true"
            data-outcome-term={term.id}
            data-tone={term.tone}
            {@attach tooltip({ text: term.text })}
          >
            {#if term.icon === 'crown'}
              <svg class="glyph size-[1.15em] shrink-0" viewBox="0 0 24 24">
                <path
                  d="M3 8 L7.5 12 L12 5 L16.5 12 L21 8 L19 18 H5 Z"
                  fill="currentColor"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linejoin="round"
                />
                <rect x="5" y="19.5" width="14" height="2" rx="1" fill="currentColor" />
              </svg>
            {:else if term.icon}
              <Icon name={term.icon} class="glyph size-[1.15em] shrink-0" />
            {/if}
            {#if term.short !== term.text}
              <span class="sm:hidden" dir="auto">{term.short}</span>
              <span class="hidden sm:inline" dir="auto">{term.text}</span>
            {:else}
              <span dir="auto">{term.text}</span>
            {/if}
          </span>
        {/each}
      </p>
    {/if}
    {#if blunder}
      <p class="blunder text-xs font-semibold" role="status" dir="auto" data-blunder>
        {blunder}
      </p>
    {/if}
  </div>
{/if}

<style>
  .term[data-tone='win'] {
    color: var(--fg);
    font-weight: 600;
  }
  .term[data-tone='win'] :global(.glyph) {
    color: var(--gold-deep);
  }
  .term[data-tone='warn'] {
    color: var(--fg);
  }
  .term[data-tone='gain'] {
    color: var(--fg);
  }
  /* Hidden until the move beside it is armed (see the comment above). */
  .blunder {
    display: none;
    color: var(--fg);
    border-inline-start: 3px solid var(--gold-deep);
    padding-inline-start: 0.5rem;
  }
  :global([data-act-group]:has([data-armed])) .blunder {
    display: block;
  }
</style>
