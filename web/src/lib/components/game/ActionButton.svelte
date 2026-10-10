<script lang="ts">
  // One move button: it sends one of the view's choice keys and carries it
  // as data-act (the end-to-end contract). The act-button rule
  // (act-button.ts) decides the rest: a local room commits on the tap; when
  // the confirm-moves setting asks for it (online rooms by default) the
  // first tap arms the button and a second one sends; while the connection
  // is down it is disabled and says it waits for it.
  import Button from '../shell/Button.svelte';
  import ConfirmButton from '../shell/ConfirmButton.svelte';
  import { actButtonState } from './act-button';

  interface Props {
    /** The choice's key, from the view. */
    key: string;
    /** Already translated. */
    label: string;
    variant?: 'primary' | 'secondary';
    /** Ask for a second tap before sending. */
    confirm?: boolean;
    /** The connection is up. */
    connected?: boolean;
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
    connected = true,
    disabled = false,
    full = true,
    class: className = '',
    onact
  }: Props = $props();

  const rule = $derived(actButtonState({ label, canAct: !disabled, connected, confirm }));

  // Long labels (a wonder's name and its price) may wrap to two lines on a
  // phone, so the height is a minimum rather than fixed.
  const fit = $derived(
    `h-auto! py-2 text-center leading-tight ${variant === 'primary' ? 'min-h-[52px]' : 'min-h-12'}`
  );
</script>

{#if rule.confirm}
  <ConfirmButton
    label={rule.label}
    {variant}
    disabled={rule.disabled}
    class="{full ? 'w-full' : ''} {fit} {className}"
    data-act={key}
    onConfirm={() => onact(key)}
  />
{:else}
  <Button
    {variant}
    disabled={rule.disabled}
    {full}
    class="{fit} {className}"
    data-act={key}
    onclick={() => onact(key)}
  >
    {rule.label}
  </Button>
{/if}
