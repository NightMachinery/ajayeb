<script lang="ts">
  // The result screen once the game is over (phase gameOver). PLACEHOLDER:
  // M5.S04b replaces the body with the victory banner, the score table and
  // the final-board toggle, and keeps these props.
  import { t } from '$lib/i18n';
  import type { View } from '$lib/wire';

  interface Props {
    view: View;
    /** Asks the server for a rematch; absent when the room cannot offer one. */
    onrematch?: () => void;
    /** Leaves the room for the home screen. */
    onhome: () => void;
  }

  let { view, onrematch, onhome }: Props = $props();
</script>

<div class="flex flex-col items-center gap-3 p-4" data-game-over data-winner={view.result?.winner}>
  {#if onrematch}
    <button
      type="button"
      class="min-h-11 rounded-control bg-accent px-4 font-semibold text-on-accent"
      onclick={onrematch}>{t('debug.rematch')}</button
    >
  {/if}
  <button
    type="button"
    class="min-h-11 rounded-control border border-rule px-4 text-fg"
    onclick={onhome}>{t('app.home')}</button
  >
</div>
