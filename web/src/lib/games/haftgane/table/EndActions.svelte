<script lang="ts">
  // The finished table's own controls, under the turn bar of the final
  // board: reopen the score sheet, the last conflicts and, when the table
  // revealed the end, the cards it turned face up.
  import Button from '$lib/components/shell/Button.svelte';
  import { ht } from '../i18n/messages.svelte';

  interface Props {
    onscores: () => void;
    onconflicts?: () => void;
    /** Absent when the end stays hidden. */
    onreveal?: () => void;
  }

  let { onscores, onconflicts, onreveal }: Props = $props();
</script>

<div class="end" data-end-actions>
  <Button variant="secondary" onclick={onscores} data-act="open-scores">
    {ht('hg.end.open.scores')}
  </Button>
  {#if onconflicts}
    <Button variant="secondary" onclick={onconflicts} data-act="open-military">
      {ht('hg.end.open.military')}
    </Button>
  {/if}
  {#if onreveal}
    <Button variant="secondary" onclick={onreveal} data-act="open-reveal">
      {ht('hg.end.open.reveal')}
    </Button>
  {/if}
</div>

<style>
  .end {
    display: grid;
    grid-auto-columns: minmax(0, 1fr);
    grid-auto-flow: column;
    gap: 0.4rem;
    padding-inline: 0.4rem;
  }
  .end :global(button) {
    min-width: 0;
    padding-inline: 0.4rem;
    font-size: 0.875rem;
  }
</style>
