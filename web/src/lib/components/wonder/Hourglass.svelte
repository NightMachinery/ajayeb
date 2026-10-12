<script lang="ts">
  // The hourglass of the race for the last wonder (the hints design,
  // wonder-race-seventh): drawn on an unbuilt wonder once only one more
  // wonder can be built. Solid when its owner can afford it now, outlined
  // otherwise. A leaf glyph: the caller gives the tooltip.
  interface Props {
    solid?: boolean;
    class?: string;
  }

  let { solid = false, class: className = 'size-3.5' }: Props = $props();
</script>

<svg class={className} viewBox="0 0 24 24" aria-hidden="true">
  <path d="M6 3h12M6 21h12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
  <path
    d="M7 3c0 5 4 6 5 9-1 3-5 4-5 9h10c0-5-4-6-5-9 1-3 5-4 5-9z"
    fill={solid ? 'currentColor' : 'none'}
    stroke="currentColor"
    stroke-width="1.6"
    stroke-linejoin="round"
  />
</svg>
