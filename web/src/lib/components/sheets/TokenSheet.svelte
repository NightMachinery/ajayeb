<script lang="ts">
  // One progress token in detail: its medallion, name and printed effect,
  // and "Take <token>" when the caller passes the view's choice key for it
  // (a board token tapped during a pick). A plain leaf in the Sheet
  // primitive; it never builds a key.
  import { t } from '$lib/i18n';
  import Sheet from '../shell/Sheet.svelte';
  import ProgressToken from '../token/ProgressToken.svelte';
  import { tokenEffect, tokenName } from '../choices/token-text';

  interface Props {
    open?: boolean;
    slug: string | null;
    /** The view's choice key for taking it, when it can be taken now. */
    actKey?: string;
    canAct?: boolean;
    onact?: (key: string) => void;
    titleId?: string;
  }

  let {
    open = $bindable(false),
    slug,
    actKey,
    canAct = false,
    onact,
    titleId = 'token-sheet-title'
  }: Props = $props();
</script>

<Sheet bind:open {titleId}>
  {#if slug}
    <div class="flex flex-col gap-3 px-5 pt-2 pb-2" data-token-sheet={slug}>
      <div class="flex items-center gap-3">
        <ProgressToken {slug} class="shrink-0 scale-125" />
        <h2 id={titleId} class="font-display text-xl text-fg">{tokenName(slug)}</h2>
      </div>
      <p class="text-sm text-fg">{tokenEffect(slug)}</p>
      {#if actKey && onact}
        <button
          type="button"
          class="inline-flex h-[52px] w-full items-center justify-center rounded-control bg-accent px-5 text-[15px]
            font-semibold text-on-accent hover:brightness-105 active:brightness-95 disabled:cursor-not-allowed disabled:opacity-50"
          data-act={actKey}
          disabled={!canAct}
          onclick={() => {
            open = false;
            onact(actKey);
          }}
        >
          {t('choices.token.confirm', { token: tokenName(slug) })}
        </button>
      {/if}
    </div>
  {/if}
</Sheet>
