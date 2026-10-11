<script lang="ts">
  // The game menu's Hints item and the sheet it opens (GameHintsSheet):
  // the item says in one line which hints the game has, and opens the
  // editor for the admin or the read-only settings for everyone else. It
  // reads the page's game context, and shows nothing without one (the
  // gallery's top bar). The menu is a popover that is hidden once closed,
  // so the sheet lives at the end of the document instead of inside it.
  import Lightbulb from '@lucide/svelte/icons/lightbulb';
  import { t } from '$lib/i18n';
  import { summaryText } from '$lib/hints/text';
  import { tryGameContext } from '../choices/room';
  import GameHintsSheet from './GameHintsSheet.svelte';

  interface Props {
    /** The menu item's classes (the menu's own). */
    class?: string;
    /** Closes the menu before the sheet opens. */
    onclose?: () => void;
  }

  let { class: className = '', onclose }: Props = $props();

  const ctx = tryGameContext();
  const hints = $derived(ctx?.session.hints ?? null);
  let open = $state(false);

  /** Moves the sheet to the end of the body, out of the hidden popover. */
  function toBody(node: HTMLElement): () => void {
    document.body.append(node);
    return () => node.remove();
  }
</script>

{#if ctx && hints}
  <button
    type="button"
    class={className}
    data-control="menu-hints"
    onclick={() => {
      onclose?.();
      open = true;
    }}
  >
    <Lightbulb class="size-5 shrink-0 text-fg-muted" aria-hidden="true" />
    <span class="flex min-w-0 flex-col py-1.5">
      <span>{t('hints.menu')}</span>
      <span class="text-xs font-normal text-fg-muted" data-hints-summary-text>
        {summaryText(hints)}
      </span>
    </span>
  </button>
  <div class="contents" {@attach toBody}>
    <GameHintsSheet bind:open />
  </div>
{/if}
