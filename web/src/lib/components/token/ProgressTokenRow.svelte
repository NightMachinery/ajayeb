<script lang="ts">
  // The progress tokens still on the board (view.boardTokens), as a row of
  // ProgressToken medallions. When a pickBoardToken/pickLibraryToken choice
  // is open, the caller passes the pickable slugs (straight from
  // view.choices, filtered to kind "token", mapped to each choice's own
  // key) and each becomes a real button carrying that key; otherwise the
  // row is purely informational. This never decides which slugs are legal
  // and never builds a key itself.
  import { t } from '$lib/i18n';
  import ProgressToken from './ProgressToken.svelte';

  interface Props {
    tokens: readonly string[];
    /** Legal picks right now: slug to the choice's action key
     *  (view.choices of kind "token"). */
    pickable?: ReadonlyMap<string, string>;
    /** The slug chosen so far in an open pick, if any, for the selected ring. */
    selectedSlug?: string;
    compact?: boolean;
    class?: string;
  }

  let { tokens, pickable, selectedSlug, compact = false, class: className = '' }: Props = $props();
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
    />
  {/each}
</div>
