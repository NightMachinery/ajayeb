<script lang="ts">
  // One move button: it sends one of the view's choice keys and carries it
  // as data-act (the end-to-end contract). A local room commits on the tap;
  // when the confirm-moves setting asks for it (online rooms by default) the
  // first tap arms the button and a second one sends.
  import Button from '../shell/Button.svelte';
  import ConfirmButton from '../shell/ConfirmButton.svelte';

  interface Props {
    /** The choice's key, from the view. */
    key: string;
    /** Already translated. */
    label: string;
    variant?: 'primary' | 'secondary';
    /** Ask for a second tap before sending. */
    confirm?: boolean;
    disabled?: boolean;
    full?: boolean;
    class?: string;
    onact: (key: string) => void;
  }

  let {
    key,
    label,
    variant = 'primary',
    confirm = false,
    disabled = false,
    full = true,
    class: className = '',
    onact
  }: Props = $props();

  // Long labels (a wonder's name and its price) may wrap to two lines on a
  // phone, so the height is a minimum rather than fixed.
  const fit = $derived(
    `h-auto! py-2 text-center leading-tight ${variant === 'primary' ? 'min-h-[52px]' : 'min-h-12'}`
  );
</script>

{#if confirm}
  <ConfirmButton
    {label}
    {variant}
    {disabled}
    class="{full ? 'w-full' : ''} {fit} {className}"
    data-act={key}
    onConfirm={() => onact(key)}
  />
{:else}
  <Button
    {variant}
    {disabled}
    {full}
    class="{fit} {className}"
    data-act={key}
    onclick={() => onact(key)}
  >
    {label}
  </Button>
{/if}
