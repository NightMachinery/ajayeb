<script lang="ts">
  // A plain arrow between two parts of a concept diagram. It points to the
  // inline end, so it follows the reading direction of its row; the
  // diagrams whose rows are spatial set dir="ltr" on them.
  interface Props {
    class?: string;
  }

  let { class: className = '' }: Props = $props();
</script>

<svg
  viewBox="0 0 16 12"
  class="h-[0.75em] w-[1em] shrink-0 rtl:-scale-x-100 {className}"
  aria-hidden="true"
>
  <path
    d="M1 6H13M9 2L14 6L9 10"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
  />
</svg>
