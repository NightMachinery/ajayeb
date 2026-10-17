<script lang="ts">
  // The overlay that carries what the animation runner draws on top of the
  // board (lib/fx/anim/runner.ts): a card flying from its slot, a card
  // flipping face up, a coin float, a looting token's flip, a progress
  // token's flight, a science pair's glow, the Great Library's backs, the
  // age deal, a draft tile's flight, a destroyed card thrown out. Fixed over
  // the whole screen, never clickable, above the board and its bars and
  // below the pass-device screen, the toasts, the sheets (native dialogs, in
  // the top layer) and the tooltips. It is empty whenever nothing is
  // animating, and with Animation off it is never used.
  //
  // Beside it, its top part (`fx-top`): a manual popover the runner shows
  // only while it draws over a dialog (the Great Library's tokens dealt into
  // the drawer's choice, the victory banner over the result screen), hidden
  // again when that ends. It is just as unclickable, so the dialog under it
  // keeps every tap.
  //
  // Its stylesheet is also where data-fx-hidden lives: the runner sets it on
  // an element a flying card is about to land on, which keeps its box (so
  // nothing moves) and hides it until the card lands.
  //
  // dir="ltr": the runner places everything with physical screen
  // coordinates, whatever the reading direction.
  import { registerFxLayer, registerFxTop } from '$lib/fx/anim/layer';
</script>

<div
  class="fx-layer pointer-events-none fixed inset-0 z-30 overflow-hidden"
  dir="ltr"
  aria-hidden="true"
  {@attach registerFxLayer}
></div>
<div class="fx-top" popover="manual" dir="ltr" aria-hidden="true" {@attach registerFxTop}></div>

<style>
  :global([data-fx-hidden]) {
    visibility: hidden !important;
  }
  .fx-top {
    position: fixed;
    inset: 0;
    width: 100%;
    height: 100%;
    max-width: none;
    max-height: none;
    margin: 0;
    padding: 0;
    border: 0;
    background: transparent;
    overflow: hidden;
    pointer-events: none;
  }
  :global(:is(.fx-layer, .fx-top) .fx-flight) {
    will-change: transform, opacity;
    transform-origin: 0 0;
    box-shadow: 0 10px 22px -8px var(--card-raised-shadow);
  }
  /* A token flies as the round medallion it is, not as a card. */
  :global(:is(.fx-layer, .fx-top) .fx-flight.token-tile) {
    box-shadow: none;
    border-radius: 999px;
  }
  :global(:is(.fx-layer, .fx-top) .fx-flip) {
    transform-origin: 0 0;
  }
  :global(:is(.fx-layer, .fx-top) .fx-flip-inner) {
    position: absolute;
    inset: 0;
    transform-style: preserve-3d;
    will-change: transform;
  }
  :global(:is(.fx-layer, .fx-top) .fx-face) {
    -webkit-backface-visibility: hidden;
    backface-visibility: hidden;
  }
  :global(:is(.fx-layer, .fx-top) .fx-float) {
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
  :global(:is(.fx-layer, .fx-top) .fx-float-gain) {
    background: var(--gold);
    color: var(--card-ink);
    border: 1px solid var(--gold-deep);
  }
  :global(:is(.fx-layer, .fx-top) .fx-float-loss) {
    background: var(--cinnabar);
    color: var(--paper-bright);
  }
  /* The old structure of an age deal, fading out as it was. */
  :global(:is(.fx-layer, .fx-top) .fx-old-structure) {
    will-change: opacity;
  }
  /* A science pair's glow: a soft gold and green halo over the symbol. */
  :global(:is(.fx-layer, .fx-top) .fx-glow) {
    border-radius: 0.6rem;
    background: radial-gradient(
      closest-side,
      color-mix(in oklch, var(--gold) 55%, transparent),
      color-mix(in oklch, var(--card-green) 35%, transparent) 70%,
      transparent
    );
    box-shadow:
      0 0 0 2px color-mix(in oklch, var(--gold) 80%, transparent),
      0 0 14px 4px color-mix(in oklch, var(--card-green) 45%, transparent);
    will-change: transform, opacity;
  }
  /* The back of a progress token, as the Great Library deals it to a viewer
     who may not see it: the token frame alone, no letter, no art. */
  :global(:is(.fx-layer, .fx-top) .fx-token-back) {
    display: grid;
    place-items: center;
    border-radius: 999px;
    border: 2px solid var(--gold-deep);
    background: var(--card-paper);
    color: var(--gild);
    box-shadow: 0 4px 10px -4px var(--card-raised-shadow);
    will-change: transform, opacity;
  }
  :global(:is(.fx-layer, .fx-top) .fx-token-back svg) {
    width: 80%;
    height: 80%;
    fill: currentColor;
  }
  /* A card mounted for its flight (one destroyed out of a city band). */
  :global(:is(.fx-layer, .fx-top) .fx-card) {
    border-radius: 7% / 4.5%;
  }
  /* The victory banner: a ribbon tinted by the victory kind, with its burst. */
  :global(:is(.fx-layer, .fx-top) .fx-banner) {
    position: absolute;
    inset-block-start: 0;
    inset-inline-start: 0;
    width: 0;
    height: 0;
    --banner: var(--gold);
    --banner-ink: var(--card-ink);
    --banner-edge: var(--gold-deep);
  }
  :global(:is(.fx-layer, .fx-top) .fx-banner-military) {
    --banner: var(--cinnabar);
    --banner-ink: var(--paper-bright);
    --banner-edge: color-mix(in oklch, var(--cinnabar) 70%, black);
  }
  :global(:is(.fx-layer, .fx-top) .fx-banner-science) {
    --banner: var(--card-green);
    --banner-ink: var(--paper-bright);
    --banner-edge: color-mix(in oklch, var(--card-green) 65%, black);
  }
  :global(:is(.fx-layer, .fx-top) :is(.fx-banner-shared, .fx-banner-forfeit)) {
    --banner: var(--paper-bright);
    --banner-ink: var(--card-ink);
    --banner-edge: var(--gold-deep);
  }
  :global(:is(.fx-layer, .fx-top) .fx-banner-ribbon) {
    position: absolute;
    inset-block-start: 0;
    inset-inline-start: 0;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.45rem 1.1rem;
    border-radius: 999px;
    border: 2px solid var(--banner-edge);
    background: var(--banner);
    color: var(--banner-ink);
    font-family: var(--font-display, inherit);
    font-size: 1.25rem;
    line-height: 1.2;
    white-space: nowrap;
    box-shadow:
      0 0 0 3px color-mix(in oklch, var(--gold) 45%, transparent),
      0 12px 28px -10px var(--card-raised-shadow);
    will-change: transform, opacity;
  }
  :global(:is(.fx-layer, .fx-top) .fx-banner-crest) {
    width: 1.4rem;
    height: 1.4rem;
    fill: currentColor;
  }
  :global(:is(.fx-layer, .fx-top) .fx-particle) {
    position: absolute;
    inset-block-start: 0;
    inset-inline-start: 0;
    width: 0.6rem;
    height: 0.6rem;
    background: var(--gold);
    will-change: transform, opacity;
  }
  :global(:is(.fx-layer, .fx-top) .fx-particle-gem) {
    clip-path: polygon(50% 0, 100% 50%, 50% 100%, 0 50%);
  }
  :global(:is(.fx-layer, .fx-top) .fx-particle-dot) {
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 999px;
  }
  :global(:is(.fx-layer, .fx-top) .fx-hue-1) {
    background: var(--sun);
  }
  :global(:is(.fx-layer, .fx-top) .fx-hue-2) {
    background: var(--moon);
  }
  :global(:is(.fx-layer, .fx-top) .fx-hue-3) {
    background: var(--banner);
  }
</style>
