<script lang="ts">
  // The overlay that carries what the animation runner draws on top of the
  // board (lib/fx/anim/runner.ts): a card flying from its slot, a card
  // flipping face up, a coin float. Fixed over the whole screen, never
  // clickable, above the board and its bars and below the pass-device
  // screen, the toasts, the sheets (native dialogs, in the top layer) and
  // the tooltips. It is empty whenever nothing is animating, and with
  // Animation off it is never used.
  //
  // Its stylesheet is also where data-fx-hidden lives: the runner sets it on
  // an element a flying card is about to land on, which keeps its box (so
  // nothing moves) and hides it until the card lands.
  //
  // dir="ltr": the runner places everything with physical screen
  // coordinates, whatever the reading direction.
  import { registerFxLayer } from '$lib/fx/anim/layer';
</script>

<div
  class="fx-layer pointer-events-none fixed inset-0 z-30 overflow-hidden"
  dir="ltr"
  aria-hidden="true"
  {@attach registerFxLayer}
></div>

<style>
  :global([data-fx-hidden]) {
    visibility: hidden !important;
  }
  .fx-layer :global(.fx-flight) {
    will-change: transform, opacity;
    transform-origin: 0 0;
    box-shadow: 0 10px 22px -8px var(--card-raised-shadow);
  }
  .fx-layer :global(.fx-flip) {
    transform-origin: 0 0;
  }
  .fx-layer :global(.fx-flip-inner) {
    position: absolute;
    inset: 0;
    transform-style: preserve-3d;
    will-change: transform;
  }
  .fx-layer :global(.fx-face) {
    -webkit-backface-visibility: hidden;
    backface-visibility: hidden;
  }
  .fx-layer :global(.fx-float) {
    position: absolute;
    inset-block-start: 0;
    inset-inline-start: 0;
    padding-inline: 0.3em;
    border-radius: 999px;
    font-size: 0.875rem;
    font-weight: 700;
    line-height: 1.25;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    will-change: transform, opacity;
    box-shadow: 0 2px 6px -2px var(--card-raised-shadow);
  }
  .fx-layer :global(.fx-float-gain) {
    background: var(--gold);
    color: var(--card-ink);
    border: 1px solid var(--gold-deep);
  }
  .fx-layer :global(.fx-float-loss) {
    background: var(--cinnabar);
    color: var(--paper-bright);
  }
</style>
