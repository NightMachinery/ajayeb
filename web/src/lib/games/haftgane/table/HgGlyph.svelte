<script lang="ts">
  // One drawing from glyphs.ts, sized in em by the caller's font size like
  // $lib/icons/Icon.svelte. Decorative: the element around it carries the
  // tooltip or the label. `flip` mirrors it (the pass arrow pointing left).
  import { glyphMarkup, glyphSource } from './glyphs';

  interface Props {
    name: string;
    flip?: boolean;
    class?: string;
  }

  let { name, flip = false, class: className = '' }: Props = $props();
  const uid = $props.id();
  const source = $derived(glyphSource(name));
</script>

{#if source}
  <svg
    viewBox={source.viewBox}
    class="hg-glyph {className}"
    class:flip
    aria-hidden="true"
    data-glyph={name}
  >
    <!-- Only our build-time vector sources enter markup (glyphs.ts). -->
    <!-- eslint-disable-next-line svelte/no-at-html-tags -->
    {@html glyphMarkup(source, uid)}
  </svg>
{/if}

<style>
  /* Scoped, not Tailwind: a utility used nowhere else would add a rule to
     the global stylesheet that Home and Duel load. */
  .hg-glyph {
    display: inline-block;
    width: 1em;
    height: 1em;
    flex-shrink: 0;
    vertical-align: -0.125em;
  }
  .flip {
    transform: scaleX(-1);
  }
</style>
