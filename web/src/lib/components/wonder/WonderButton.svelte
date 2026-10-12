<script lang="ts">
  // The phone strip's one wonder control: the wonder glyph and how many of
  // the seat's wonders are built out of those still in play ("1/4"), from
  // view/wonders.wonderTally. It opens that seat's wonder sheet. When the
  // view's choices let the seat build a wonder now (buildableWonders), it
  // glows in the accent with a dot on its corner, so a player sees the
  // chance without opening anything; the spoken label says so too.
  //
  // A leaf component with plain props. The visible box is 36 px tall to
  // keep the strip short; its hit area reaches 44 px through an invisible
  // margin (the ::after layer). The margin is 6 px, since an absolute
  // ::after starts inside the 1 px border: 34 + 2 x 6 = 46 px, of which the
  // science pill just below takes back a pixel or two.
  import { formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';

  interface Props {
    built: number;
    total: number;
    /** At least one wonder can be built now (from the view's choices). */
    buildable?: boolean;
    /** Whose wonders, for the spoken label (a display name). */
    name: string;
    onclick?: () => void;
    class?: string;
  }

  let { built, total, buildable = false, name, onclick, class: className = '' }: Props = $props();

  const label = $derived(
    [
      t('board.wonders.button', { name, built, total }),
      buildable ? t('board.wonders.buildableNow') : ''
    ]
      .filter(Boolean)
      .join('. ')
  );
</script>

<button
  type="button"
  class="wonder-button relative flex h-9 shrink-0 items-center gap-1 rounded-control border px-2 text-[13px] font-semibold
    tabular-nums transition-colors after:absolute after:-inset-1.5 after:content-[''] short:px-1.5
    {buildable
    ? 'border-accent bg-accent-soft text-accent-soft-fg ring-2 ring-accent/45'
    : 'border-rule-strong bg-panel text-fg hover:border-accent'} {className}"
  aria-label={label}
  data-control="wonders"
  data-buildable={buildable ? '' : undefined}
  {onclick}
>
  <Icon name="wonder" class="text-base" />
  <span data-must-fit>{formatNumber(built)}/{formatNumber(total)}</span>
  {#if buildable}
    <span
      class="absolute -end-1 -top-1 size-2.5 rounded-full border border-panel bg-accent"
      aria-hidden="true"
    ></span>
  {/if}
</button>
