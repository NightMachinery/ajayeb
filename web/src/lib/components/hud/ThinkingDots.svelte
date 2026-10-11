<script lang="ts">
  // A calm "thinking" indicator: three dots that fade in turn, slowly. With
  // Animation off in Settings, app.css stops the animation and the dots stay
  // as they are drawn at rest: all three, half lit, so the state still shows
  // without movement. Decorative by default; the caller says what is
  // thinking in text.
  interface Props {
    /** The dot size: 'sm' for the turn bar and strips, 'md' for panels. */
    size?: 'sm' | 'md';
    class?: string;
  }

  let { size = 'sm', class: className = '' }: Props = $props();
</script>

<span
  class="thinking-dots inline-flex items-center {size === 'sm' ? 'gap-0.5' : 'gap-1'} {className}"
  aria-hidden="true"
>
  {#each [0, 1, 2] as i (i)}
    <span
      class="dot rounded-full bg-current {size === 'sm' ? 'size-1.5' : 'size-2'}"
      style:animation-delay="{i * 240}ms"
    ></span>
  {/each}
</span>

<style>
  .dot {
    opacity: 0.55;
    animation: thinking-dot 1.44s ease-in-out infinite;
  }
  @keyframes thinking-dot {
    0%,
    100% {
      opacity: 0.25;
    }
    40% {
      opacity: 1;
    }
  }
</style>
