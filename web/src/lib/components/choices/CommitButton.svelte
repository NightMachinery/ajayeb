<script lang="ts">
  // The one button that sends a choice panel's move, pinned to the bottom of
  // the panel. It carries data-act with the selected choice's key from the
  // view (the end-to-end contract), and no key at all until something is
  // selected. The danger style is for destroying a card.
  //
  // It follows the same act-button rule as the card sheet
  // (game/act-button.ts): with the confirm step (online rooms by default)
  // the first tap arms it and a second one sends, and while the connection
  // is down it is disabled and says it waits for the connection.
  import { actButtonState } from '../game/act-button';
  import Button from '../shell/Button.svelte';
  import ConfirmButton from '../shell/ConfirmButton.svelte';

  interface Props {
    /** Already translated. */
    label: string;
    /** The selected choice's key, or null before a selection. */
    actKey: string | null;
    danger?: boolean;
    disabled?: boolean;
    /** Ask for a second tap before sending. */
    confirm?: boolean;
    /** The connection is up. */
    connected?: boolean;
    onact: (key: string) => void;
  }

  let {
    label,
    actKey,
    danger = false,
    disabled = false,
    confirm = false,
    connected = true,
    onact
  }: Props = $props();

  const rule = $derived(
    actButtonState({ label, canAct: !disabled, connected, confirm, ready: actKey !== null })
  );
  const cls = 'commit h-auto! min-h-[52px] w-full py-2 leading-tight';

  function send(): void {
    if (actKey !== null) onact(actKey);
  }
</script>

<div
  class="sticky bottom-0 -mx-4 mt-1 border-t border-rule bg-panel px-4 pt-3"
  class:danger
  data-commit
>
  {#if rule.confirm}
    <!-- A new selection starts unarmed: the armed tap was for the old one. -->
    {#key actKey}
      <ConfirmButton
        label={rule.label}
        class={cls}
        data-act={actKey ?? undefined}
        disabled={rule.disabled}
        onConfirm={send}
      />
    {/key}
  {:else}
    <Button full class={cls} data-act={actKey ?? undefined} disabled={rule.disabled} onclick={send}>
      <span class="min-w-0" dir="auto">{rule.label}</span>
    </Button>
  {/if}
</div>

<style>
  /* A deep cinnabar (--danger), dark enough for white text in both themes.
     Unlayered, so it wins over the button's accent utilities. */
  .danger :global(.commit) {
    background: var(--danger);
    color: var(--on-danger);
  }
</style>
