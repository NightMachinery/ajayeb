<script lang="ts">
  // The host for every pending choice a card, wonder or age end opens:
  // destroying an opponent's card, building from the discard pile, taking a
  // progress token from the board or the Great Library offer, and choosing
  // who starts the next age. PLACEHOLDER: it lists the legal choices as
  // plain buttons so a game stays playable; M5.S04b replaces the body with
  // the real panels and keeps these props.
  import type { View } from '$lib/wire';

  interface Props {
    view: View;
    /** Whether this client may act now (presenter idle and session.canAct). */
    canAct: boolean;
    /** Sends one of the view's choice keys. */
    onact: (key: string) => void;
  }

  let { view, canAct, onact }: Props = $props();

  const PENDING_KINDS = new Set(['destroy', 'revive', 'token', 'starter']);
  const choices = $derived(view.choices.filter((c) => PENDING_KINDS.has(c.kind)));
</script>

{#if view.pending && choices.length > 0}
  <div class="flex flex-wrap gap-2 p-2" data-choice-host>
    {#each choices as choice (choice.key)}
      <button
        type="button"
        class="min-h-11 rounded-control border border-rule bg-panel px-3 text-fg"
        data-act={choice.key}
        disabled={!canAct}
        onclick={() => onact(choice.key)}><code>{choice.key}</code></button
      >
    {/each}
  </div>
{/if}
