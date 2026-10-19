<script lang="ts">
  // The Ajayeb wordmark (logo-wordmark.svg): the Nastaliq عجایب over the
  // Latin "Ajayeb", inline so it follows the theme. The Persian word takes
  // the text colour (--i1, currentColor) and the Latin line the gild
  // accent. Sized by its width in the caller's class; the height follows.
  //
  // Decorative by default, beside a visible title; pass `label` (through
  // t()) where it stands for the name itself.
  import { WORDMARK } from '$lib/ornaments/art';

  let { label, class: className = '' }: { label?: string; class?: string } = $props();
</script>

<svg
  class="block h-auto {className}"
  viewBox={WORDMARK.viewBox}
  role={label ? 'img' : undefined}
  aria-label={label}
  aria-hidden={label ? undefined : 'true'}
  data-wordmark
>
  <!-- Our own wordmark file, imported at build time (ornaments/art.ts). -->
  <!-- eslint-disable-next-line svelte/no-at-html-tags -->
  {@html WORDMARK.inner}
</svg>
