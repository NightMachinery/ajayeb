<script lang="ts" module>
  // Haftgane's three science symbols, drawn from their vector sources
  // (games/haftgane/icons/svg/, art/vector/specs/haftgane-resources-science.md).
  // Until the art wiring gives Haftgane an icon component of its own, the
  // rules diagrams read the sources here, the way $lib/icons/Icon.svelte
  // reads Duel's. They take the small drawings, the only gear Haftgane has
  // of its own: the large gear is Duel's chain-gear, and importing a Duel
  // icon source would move it out of Duel's chunks.
  import compass from '../icons/svg/science-compass-sm.svg?raw';
  import gear from '../icons/svg/science-gear-sm.svg?raw';
  import tablet from '../icons/svg/science-tablet-sm.svg?raw';

  export type HgScienceSymbol = 'compass' | 'gear' | 'tablet';

  /** The viewBox and the shapes between the opening and closing svg tags. */
  function parse(raw: string): { viewBox: string; inner: string } {
    const open = /<svg\b[^>]*>/.exec(raw);
    const viewBox = open ? (/viewBox="([^"]+)"/.exec(open[0])?.[1] ?? '0 0 12 12') : '0 0 12 12';
    const start = open ? open.index + open[0].length : 0;
    return { viewBox, inner: raw.slice(start, raw.lastIndexOf('</svg>')).trim() };
  }

  const SOURCES: Record<HgScienceSymbol, { viewBox: string; inner: string }> = {
    compass: parse(compass),
    gear: parse(gear),
    tablet: parse(tablet)
  };
</script>

<script lang="ts">
  interface Props {
    symbol: HgScienceSymbol;
    class?: string;
  }

  let { symbol, class: className = '' }: Props = $props();
  const source = $derived(SOURCES[symbol]);
</script>

<!-- Sized in em by the caller's font size, like Icon.svelte. No data-icon:
     that attribute names Duel's icons only. -->
<svg viewBox={source.viewBox} class="glyph {className}" aria-hidden="true" data-science={symbol}>
  <!-- Only our build-time vector sources enter markup; they hold no ids. -->
  <!-- eslint-disable-next-line svelte/no-at-html-tags -->
  {@html source.inner}
</svg>

<style>
  /* Layout that no Duel screen uses lives here, not in Tailwind classes:
     a class would add its rule to the global stylesheet that Home and
     Duel load, and this one loads only with Haftgane's rules. */
  .glyph {
    display: inline-block;
    width: 1em;
    height: 1em;
    flex-shrink: 0;
    vertical-align: -0.125em;
  }
</style>
