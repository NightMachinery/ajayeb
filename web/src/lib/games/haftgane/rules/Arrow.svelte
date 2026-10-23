<script lang="ts">
  // A plain arrow between two seats of a diagram. Its direction is spatial,
  // to the physical left or right of the table, and never follows the
  // reading direction: a left neighbour is on the left in every locale
  // (docs/haftgane/rules.md, ruling 30). The diagrams that use it set
  // dir="ltr" on their rows for the same reason.
  interface Props {
    to: 'left' | 'right';
    class?: string;
  }

  let { to, class: className = '' }: Props = $props();
</script>

<svg viewBox="0 0 16 12" class="h-[0.75em] w-[1em] shrink-0 {className}" aria-hidden="true">
  <path
    d={to === 'right' ? 'M1 6H13M9 2L14 6L9 10' : 'M15 6H3M7 2L2 6L7 10'}
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
  />
</svg>
