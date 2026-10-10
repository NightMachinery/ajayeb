<script lang="ts">
  // The wonder draft (phase wonderDraft). PLACEHOLDER: it lists the draft
  // choices as plain buttons so a game stays playable; M5.S04b replaces the
  // body with the real draft screen and keeps these props.
  import type { View } from '$lib/wire';

  interface Props {
    view: View;
    /** Whether this client may act now (presenter idle and session.canAct). */
    canAct: boolean;
    /** Sends one of the view's choice keys. */
    onact: (key: string) => void;
  }

  let { view, canAct, onact }: Props = $props();

  const choices = $derived(view.choices.filter((c) => c.kind === 'draft'));
</script>

<div class="flex flex-wrap gap-2 p-2" data-draft>
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
