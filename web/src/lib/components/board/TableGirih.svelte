<script lang="ts">
  // The table's girih (art/vector/specs/ornaments-track-logo.md,
  // ornament-girih): the eight-fold tile repeated as a CSS mask over the
  // theme's rule colour, so the colour follows the theme and is never baked
  // into the file. It is a faint inlay behind the card structure, faded out
  // toward its edges so it never frames or competes with the cards, and it
  // takes no pointer events. One static layer: nothing in it animates.
  // It is a span, not a div: the animations find the dealt structure as
  // `.card-structure > div` (fx/anim/runner.ts).
  //
  // The tile's URL comes from Vite, which inlines a file this small or else
  // names it by its content hash, so a changed tile always reaches the page.
  import girihUrl from '$lib/ornaments/svg/ornament-girih.svg?url';

  let { class: className = '' }: { class?: string } = $props();
</script>

<span
  class="table-girih pointer-events-none absolute inset-0 block {className}"
  style:--girih-url={`url("${girihUrl}")`}
  aria-hidden="true"
  data-table-girih
></span>

<style>
  .table-girih {
    background: var(--rule-strong);
    opacity: var(--girih-alpha, 0.28);
    /* The tile, intersected with an oval that fades it out before the
       edges of the table. */
    -webkit-mask:
      var(--girih-url) center / 112px 112px repeat,
      radial-gradient(farthest-side, #000 60%, transparent) center / 100% 100% no-repeat;
    -webkit-mask-composite: source-in;
    mask:
      var(--girih-url) center / 112px 112px repeat,
      radial-gradient(farthest-side, #000 60%, transparent) center / 100% 100% no-repeat;
    mask-composite: intersect;
  }
</style>
