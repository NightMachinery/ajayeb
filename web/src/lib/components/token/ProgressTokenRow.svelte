<script lang="ts">
  // The progress tokens still on the board (view.boardTokens), as a row of
  // ProgressToken medallions. When a pickBoardToken/pickLibraryToken choice
  // is open, the caller passes the pickable slugs (straight from
  // view.choices, filtered to kind "token", mapped to each choice's own
  // key) and each becomes a real button carrying that key; otherwise the
  // row is purely informational. This never decides which slugs are legal
  // and never builds a key itself. With `caption` a short visible label
  // ("Progress tokens") leads the row, so a player new to the game knows
  // what the medallions above the cards are. In the compact row (a phone's
  // landscape top bar) it shows only from 52rem wide, where the track keeps
  // room for its spaces; the row is still labelled by it below that.
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
    /** Lead the row with its visible name. */
    caption?: boolean;
    /** The medallions' size, when not the default of the compact flag. */
    size?: 'sm' | 'md';
    class?: string;
  }

  let {
    tokens,
    pickable,
    selectedSlug,
    compact = false,
    caption = false,
    size,
    class: className = ''
  }: Props = $props();

  const tokenSize = $derived(size ?? (compact ? 'sm' : 'md'));
  const captionId = $props.id();
</script>

<div
  role="group"
  aria-label={caption ? undefined : t('board.tokens.boardLabel')}
  aria-labelledby={caption ? captionId : undefined}
  class="progress-token-row flex flex-wrap items-center {compact
    ? 'gap-1.5'
    : 'gap-2.5'} {className}"
>
  {#if caption}
    <span
      id={captionId}
      class="max-w-[5.5rem] text-[11px] leading-tight font-semibold tracking-wide text-fg-muted uppercase {compact
        ? 'hidden min-[52rem]:block'
        : ''}"
      data-tokens-caption
    >
      {t('board.tokens.caption')}
    </span>
  {/if}
  {#if tokens.length === 0}
    <p class="text-xs text-fg-muted">{t('board.tokens.empty')}</p>
  {/if}
  {#each tokens as slug (slug)}
    <ProgressToken
      {slug}
      size={tokenSize}
      selected={selectedSlug === slug}
      selectable={pickable?.has(slug) ?? false}
      actionKey={pickable?.get(slug)}
    />
  {/each}
</div>
