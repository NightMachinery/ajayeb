<script lang="ts">
  // The two button styles from the UX design: primary (52px, filled accent)
  // and secondary (48px, outlined panel). A plain leaf component: it takes
  // props and forwards the rest to the native <button>, so it works the same
  // standalone (the gallery) or inside ConfirmButton and future game actions.
  // Disabled labels can be instructions, so keep their ink fully opaque.
  // Neutral surfaces and a dashed edge distinguish unavailable actions from
  // accent-coloured selections, including the choice footer's danger style.
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  interface Props extends Omit<HTMLButtonAttributes, 'class' | 'type'> {
    variant?: 'primary' | 'secondary';
    type?: 'button' | 'submit';
    full?: boolean;
    class?: string;
    children: Snippet;
  }

  let {
    variant = 'primary',
    type = 'button',
    full = false,
    class: className = '',
    children,
    ...rest
  }: Props = $props();
</script>

<button
  {type}
  class="inline-flex items-center justify-center gap-2 rounded-control px-5 text-[15px] font-semibold
    transition-colors duration-(--dur-fast) ease-(--curve-out) disabled:cursor-not-allowed disabled:border! disabled:border-dashed! disabled:border-rule! disabled:bg-panel-sunken! disabled:text-fg-muted!
    {variant === 'primary'
    ? 'h-[52px] bg-accent text-on-accent enabled:hover:brightness-105 enabled:active:brightness-95'
    : 'h-[48px] border border-rule bg-panel text-fg enabled:hover:bg-panel-sunken enabled:active:bg-panel-sunken'}
    {full ? 'w-full' : ''} {className}"
  {...rest}
>
  {@render children()}
</button>
