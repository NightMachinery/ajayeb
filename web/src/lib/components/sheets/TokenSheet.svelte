<script lang="ts">
  // One progress token in detail: its medallion, name and printed effect,
  // and "Take <token>" when the caller passes the view's choice key for it
  // (a board token tapped during a pick). A plain leaf in the Sheet
  // primitive; it never builds a key.
  //
  // Board hints (M10.HS4): what the token is worth now to each seat, one
  // line per seat, given as `hint` or from the page's game context.
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import { tokenHint, type TokenHint } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import { tryGameContext } from '../choices/room';
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
    /** The board hints about it; else from the game context. */
    hint?: TokenHint;
  }

  let {
    open = $bindable(false),
    slug,
    actKey,
    canAct = false,
    onact,
    titleId = 'token-sheet-title',
    hint
  }: Props = $props();

  const ctx = tryGameContext();
  const th = $derived.by((): TokenHint | undefined => {
    if (hint || !slug) return hint;
    const view = ctx?.session.view;
    const screen = sessionHintScreen(ctx?.session, hintHide.value);
    return view && screen ? tokenHint(view, slug, screen) : undefined;
  });
</script>

<Sheet bind:open {titleId}>
  {#if slug}
    <div class="flex flex-col gap-3 px-5 pt-2 pb-2" data-token-sheet={slug}>
      <div class="flex items-center gap-3">
        <ProgressToken {slug} hint={th} class="shrink-0 scale-125" />
        <h2 id={titleId} class="font-display text-xl text-fg">{tokenName(slug)}</h2>
      </div>
      <p class="text-sm text-fg">{tokenEffect(slug)}</p>
      {#if th && (th.lines.length > 0 || th.mark)}
        <ul class="flex flex-col gap-1 text-sm text-fg-muted" data-token-values>
          {#each th.lines as line (line)}
            <li data-token-value>{line}</li>
          {/each}
          {#if th.mark}
            <li class="font-semibold text-fg" data-hint="law">{th.mark.tip}</li>
          {/if}
        </ul>
      {/if}
      {#if actKey && onact}
        <button
          type="button"
          class="inline-flex h-[52px] w-full items-center justify-center rounded-control bg-accent px-5 text-[15px]
            font-semibold text-on-accent hover:brightness-105 active:brightness-95 disabled:cursor-not-allowed disabled:opacity-50"
          data-act={actKey}
          data-sheet-actions
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
