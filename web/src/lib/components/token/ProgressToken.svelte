<script lang="ts">
  // One progress token medallion: the token's approved bitmap (a circular
  // emblem, shipped on the paper colour) inside a gold ring when the
  // generated art index has one; otherwise, or if the file fails to load,
  // the placeholder: the hand-authored token-frame icon plus the token's
  // own first letter. Its name goes through t() (each of the 10 tokens is
  // its own message key; see i18n/en/board.ts). A leaf
  // component, plain props: the caller decides whether it is selectable
  // (the pending pick choice) and whether it should act as a real button.
  //
  // Board hints (M10.HS4, `hint`, view/board-hints.ts tokenHint): the
  // tooltip adds what the token is worth now to each seat (token-value-now,
  // token-deny-note), and the Law medallion gets a laurel ring while Law
  // would be a seat's sixth symbol (law-token-marks): solid for this
  // screen's own seat, dashed in the opponent's crest colour.
  //
  // `onopen` (M10.F3) makes a board token a button that opens its sheet
  // (TokenSheet: what it does and what it is worth), the model of a card:
  // a tap opens it, a long-press on touch shows the tooltip. It carries no
  // action key; the pick of a pending choice stays `actionKey`.
  //
  // The tooltip names the token, says what it does (choices/token-text.ts
  // tokenEffect) and adds the hint lines; it links the rules' tokens section.
  import ArtImage from '$lib/art/ArtImage.svelte';
  import { artId, tokenArtSizes } from '$lib/art/lookup';
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { TokenHint } from '$lib/view/board-hints';
  import { tokenEffect } from '../choices/token-text';

  interface Props {
    /** A TokenSlug, e.g. "agriculture". Untrusted-width string so this also
     *  renders a slug this build does not recognise, falling back to it
     *  verbatim rather than throwing. */
    slug: string;
    /** Gone from the board (taken by either player), shown where the board
     *  token used to be: ghosted, never a button. */
    taken?: boolean;
    /** In a player's city: full colour, and the label says it is owned.
     *  Never ghosted, since the token is very much in play. */
    owned?: boolean;
    /** Highlighted as the current pick in an open choice. */
    selected?: boolean;
    /** The action key to send if tapped, e.g. "token:agriculture". Only
     *  when set does this render as a real, actionable button; otherwise
     *  it is a plain, non-interactive medallion (a city's owned tokens). */
    actionKey?: string;
    /** Whether tapping actionKey is legal right now (the caller already
     *  knows this from the view's choices/disabled lists; this component
     *  never decides legality itself). */
    selectable?: boolean;
    /** xs (20 px) is for a phone strip's owned tokens, sm (26 px) for
     *  compact rows, md (36 px) for the board. */
    size?: 'xs' | 'sm' | 'md';
    /** The board hints about it: its value now, and the Law laurel. */
    hint?: TokenHint;
    /** Opens the token's sheet on a tap (a board token with no pick open). */
    onopen?: () => void;
    class?: string;
  }

  let {
    slug,
    taken = false,
    owned = false,
    selected = false,
    actionKey,
    selectable = true,
    size = 'md',
    hint,
    onopen,
    class: className = ''
  }: Props = $props();

  function name(slug: string): string {
    switch (slug) {
      case 'agriculture':
        return t('token.agriculture');
      case 'architecture':
        return t('token.architecture');
      case 'economy':
        return t('token.economy');
      case 'law':
        return t('token.law');
      case 'masonry':
        return t('token.masonry');
      case 'mathematics':
        return t('token.mathematics');
      case 'philosophy':
        return t('token.philosophy');
      case 'strategy':
        return t('token.strategy');
      case 'theology':
        return t('token.theology');
      case 'urbanism':
        return t('token.urbanism');
      default:
        return slug;
    }
  }

  const label = $derived(name(slug));
  const staticLabel = $derived(
    owned
      ? t('board.token.owned', { name: label })
      : taken
        ? t('board.token.taken', { name: label })
        : label
  );
  const initial = $derived(label.charAt(0).toUpperCase());
  const hintText = $derived(
    hint ? [...hint.lines, hint.mark?.tip ?? ''].filter(Boolean).join(' ') : ''
  );
  const tipText = (base: string): string => (hintText ? `${base}. ${hintText}` : base);
  const effect = $derived(tokenEffect(slug));
  /** The tooltip: the name (as the caller words it) and what the token does,
   *  then the hint lines. */
  const tipFor = (base: string) => ({
    text: effect ? `${base}: ${effect}` : base,
    lines: hint ? [...hint.lines, hint.mark?.tip ?? ''].filter(Boolean) : [],
    rulesId: 'tokens' as const
  });
  const sizeClass = $derived(
    size === 'xs'
      ? 'size-5 text-[10px]'
      : size === 'sm'
        ? 'size-[26px] text-[11px]'
        : 'size-9 text-sm'
  );
</script>

{#snippet medallion()}
  <span class="relative grid place-items-center {sizeClass}">
    {#if hint?.mark}
      <span
        class="law-mark pointer-events-none absolute -inset-1 rounded-full"
        data-hint="law"
        data-style={hint.mark.style}
        data-tone={hint.mark.tone}
        aria-hidden="true"
      ></span>
    {/if}
    <ArtImage
      id={artId('token', slug)}
      size="thumb"
      sizes={tokenArtSizes(size)}
      feather="none"
      class="absolute inset-0 size-full rounded-full {size === 'xs'
        ? 'border-[1.5px]'
        : 'border-2'} border-(color:--gold-deep) bg-card-paper"
    >
      {#snippet fallback()}
        <Icon name="token-frame" class="absolute inset-0 size-full text-gild" />
        <span class="relative font-display leading-none font-semibold">{initial}</span>
      {/snippet}
    </ArtImage>
  </span>
{/snippet}

{#if actionKey}
  <button
    type="button"
    data-act={actionKey}
    disabled={!selectable}
    aria-label={tipText(t('board.token.take', { name: label }))}
    {@attach tooltip(tipFor(label))}
    data-tap-ok={size === 'md' ? undefined : ''}
    class="token-tile grid place-items-center rounded-full transition-transform duration-(--dur-fast)
      {size === 'md' ? 'min-h-11 min-w-11' : ''}
      {selected ? 'ring-2 ring-focus ring-offset-2 ring-offset-table' : ''}
      {selectable
      ? 'cursor-pointer hover:-translate-y-0.5'
      : 'cursor-not-allowed opacity-40'} {className}"
  >
    {@render medallion()}
  </button>
{:else if onopen}
  <button
    type="button"
    aria-label={tipText(t('board.token.open', { name: label }))}
    aria-haspopup="dialog"
    {@attach tooltip(tipFor(label))}
    data-token-open={slug}
    data-tap-ok={size === 'md' ? undefined : ''}
    class="token-tile grid cursor-pointer place-items-center rounded-full transition-transform duration-(--dur-fast) hover:-translate-y-0.5
      {size === 'md' ? 'min-h-11 min-w-11' : ''} {className}"
    onclick={onopen}
  >
    {@render medallion()}
  </button>
{:else}
  <div
    class="token-tile grid place-items-center rounded-full {taken && !owned
      ? 'opacity-40 grayscale'
      : ''} {className}"
    {@attach tooltip(tipFor(staticLabel))}
    role="img"
    aria-label={tipText(staticLabel)}
    data-owned={owned ? '' : undefined}
    data-taken={taken && !owned ? '' : undefined}
  >
    {@render medallion()}
  </div>
{/if}

<style>
  /* The Law laurel: the hint pip's colours (card/HintPip.svelte). */
  .law-mark {
    --mark: var(--gold-deep);
    border: 2px solid var(--mark);
  }
  .law-mark[data-tone='sun'] {
    --mark: var(--sun-ink);
  }
  .law-mark[data-tone='moon'] {
    --mark: var(--moon-ink);
  }
  .law-mark[data-style='outline'] {
    border-style: dashed;
  }
</style>
