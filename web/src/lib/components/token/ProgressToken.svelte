<script lang="ts">
  // One progress token medallion: a placeholder for the real vector art
  // (M9) -- the hand-authored token-frame icon plus the token's own first
  // letter as a simple glyph, and its name through t() (each of the 10
  // tokens is its own message key; see i18n/en/board.ts). A leaf
  // component, plain props: the caller decides whether it is selectable
  // (the pending pick choice) and whether it should act as a real button.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';

  interface Props {
    /** A TokenSlug, e.g. "agriculture". Untrusted-width string so this also
     *  renders a slug this build does not recognise, falling back to it
     *  verbatim rather than throwing. */
    slug: string;
    /** Already taken (by either player): shown ghosted, never a button. */
    taken?: boolean;
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
    size?: 'sm' | 'md';
    class?: string;
  }

  let {
    slug,
    taken = false,
    selected = false,
    actionKey,
    selectable = true,
    size = 'md',
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
  const initial = $derived(label.charAt(0).toUpperCase());
  const sizeClass = $derived(size === 'sm' ? 'size-[26px] text-[11px]' : 'size-9 text-sm');
</script>

{#snippet medallion()}
  <span class="relative grid place-items-center {sizeClass}">
    <Icon name="token-frame" class="absolute inset-0 size-full text-gild" />
    <span class="relative font-display leading-none font-semibold">{initial}</span>
  </span>
{/snippet}

{#if actionKey}
  <button
    type="button"
    data-act={actionKey}
    disabled={!selectable}
    aria-label={t('board.token.take', { name: label })}
    title={label}
    data-tap-ok={size === 'sm' ? '' : undefined}
    class="token-tile grid place-items-center rounded-full transition-transform duration-(--dur-fast)
      {size === 'sm' ? '' : 'min-h-11 min-w-11'}
      {selected ? 'ring-2 ring-focus ring-offset-2 ring-offset-table' : ''}
      {selectable
      ? 'cursor-pointer hover:-translate-y-0.5'
      : 'cursor-not-allowed opacity-40'} {className}"
  >
    {@render medallion()}
  </button>
{:else}
  <div
    class="token-tile grid place-items-center rounded-full {taken
      ? 'opacity-40 grayscale'
      : ''} {className}"
    title={taken ? t('board.token.owned', { name: label }) : label}
    role="img"
    aria-label={taken ? t('board.token.owned', { name: label }) : label}
  >
    {@render medallion()}
  </div>
{/if}
