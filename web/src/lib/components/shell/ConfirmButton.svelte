<script lang="ts">
  // Arm-then-confirm, for online moves (the UX design §5 "Commit, confirm,
  // undo, pending"): the first tap arms the button with a 3s countdown, and
  // a second tap within that window confirms, but a 350ms guard after arming
  // ignores an accidental double tap. A local room (AI, hot-seat) does not
  // use this; it commits on the first tap through a plain Button.
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import { t } from '$lib/i18n';
  import Button from './Button.svelte';

  // Other attributes (data-act, aria-*) go to the button itself.
  interface Props extends Omit<
    HTMLButtonAttributes,
    'class' | 'type' | 'onclick' | 'children' | 'disabled'
  > {
    /** Already translated, e.g. t('board.buildFor', { coins: 4 }). */
    label: string;
    onConfirm: () => void;
    variant?: 'primary' | 'secondary';
    disabled?: boolean;
    class?: string;
    /** How long the armed state lasts before it resets, in ms. */
    armMs?: number;
    /** The double-tap guard after arming, in ms. */
    guardMs?: number;
  }

  let {
    label,
    onConfirm,
    variant = 'primary',
    disabled = false,
    class: className = '',
    armMs = 3000,
    guardMs = 350,
    ...rest
  }: Props = $props();

  let armed = $state(false);
  let armedAt = 0;
  let resetTimer: ReturnType<typeof setTimeout> | undefined;

  function disarm(): void {
    clearTimeout(resetTimer);
    resetTimer = undefined;
    armed = false;
  }

  function handleClick(): void {
    const now = Date.now();
    if (!armed) {
      armed = true;
      armedAt = now;
      clearTimeout(resetTimer);
      resetTimer = setTimeout(disarm, armMs);
      return;
    }
    if (now - armedAt < guardMs) return; // the double-tap guard
    disarm();
    onConfirm();
  }

  $effect(() => () => clearTimeout(resetTimer));
</script>

<Button
  {...rest}
  {variant}
  {disabled}
  data-armed={armed ? '' : undefined}
  class="relative isolate overflow-hidden {className}"
  onclick={handleClick}
>
  {#if armed}
    <span
      class="confirm-ring pointer-events-none absolute inset-0 -z-10 bg-[color-mix(in_oklch,var(--on-accent)_20%,transparent)]"
      style:--arm-ms="{armMs}ms"
      aria-hidden="true"
    ></span>
  {/if}
  {armed ? t('shell.confirmArmed', { label }) : label}
</Button>

<style>
  /* The countdown shrinks toward the inline start; transform-origin has no
     logical keyword, so the RTL case is spelled out. */
  .confirm-ring {
    transform-origin: left; /* physical-ok */
    animation: confirm-shrink var(--arm-ms) linear forwards;
  }
  .confirm-ring:dir(rtl) {
    transform-origin: right; /* physical-ok */
  }
  @keyframes confirm-shrink {
    from {
      scale: 1 1;
    }
    to {
      scale: 0 1;
    }
  }
</style>
