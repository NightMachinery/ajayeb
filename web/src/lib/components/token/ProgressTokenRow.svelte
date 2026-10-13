<script lang="ts">
  // The progress tokens still on the board (view.boardTokens), as a row of
  // ProgressToken medallions. When a pickBoardToken/pickLibraryToken choice
  // is open, the caller passes the pickable slugs (straight from
  // view.choices, filtered to kind "token", mapped to each choice's own
  // key) and each becomes a real button carrying that key; otherwise the
  // row is purely informational. This never decides which slugs are legal
  // and never builds a key itself.
  //
  // Board hints (M10.HS4): each token's value now and the Law laurel
  // (view/board-hints.ts tokenHint), from the page's game context.
  //
  // `onopen` (M10.F3): a token that is not a legal pick opens its sheet on
  // a tap (the game screen passes it while no token pick is pending).
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import { tokenHint } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import { tryGameContext } from '../choices/room';
  import ProgressToken from './ProgressToken.svelte';

  interface Props {
    tokens: readonly string[];
    /** Legal picks right now: slug to the choice's action key
     *  (view.choices of kind "token"). */
    pickable?: ReadonlyMap<string, string>;
    /** The slug chosen so far in an open pick, if any, for the selected ring. */
    selectedSlug?: string;
    compact?: boolean;
    /** Opens a token's sheet (TokenSheet) on a tap. */
    onopen?: (slug: string) => void;
    class?: string;
  }

  let {
    tokens,
    pickable,
    selectedSlug,
    compact = false,
    onopen,
    class: className = ''
  }: Props = $props();

  const ctx = tryGameContext();
  const screen = $derived(sessionHintScreen(ctx?.session, hintHide.value));
  function hintOf(slug: string) {
    const view = ctx?.session.view;
    return view && screen ? tokenHint(view, slug, screen) : undefined;
  }
</script>

<div
  role="group"
  aria-label={t('board.tokens.boardLabel')}
  class="progress-token-row flex flex-wrap items-center {compact
    ? 'gap-1.5'
    : 'gap-2.5'} {className}"
>
  {#if tokens.length === 0}
    <p class="text-xs text-fg-muted">{t('board.tokens.empty')}</p>
  {/if}
  {#each tokens as slug (slug)}
    <ProgressToken
      {slug}
      size={compact ? 'sm' : 'md'}
      selected={selectedSlug === slug}
      selectable={pickable?.has(slug) ?? false}
      actionKey={pickable?.get(slug)}
      hint={hintOf(slug)}
      onopen={onopen && !pickable?.has(slug) ? () => onopen(slug) : undefined}
    />
  {/each}
</div>
