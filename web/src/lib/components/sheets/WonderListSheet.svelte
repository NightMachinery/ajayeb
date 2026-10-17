<script lang="ts">
  // One seat's wonders, opened from the wonder button of its phone strip:
  // every wonder as a row with its art, name, effect, cost and one status,
  // and "Can build now" on each wonder a choice of the view can build. For
  // the player's own seat the wonders still to build are buttons that open
  // that wonder's own sheet (WonderSheet: "Build this wonder" and the pick
  // mode); the other seat's list is to read.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { wonderTally } from '$lib/view/wonders';
  import type { WonderView } from '$lib/wire';
  import Sheet from '../shell/Sheet.svelte';
  import WonderList from '../wonder/WonderList.svelte';

  interface Props {
    open: boolean;
    /** Whose wonders (a display name, shown as given). */
    name: string;
    seat: 0 | 1;
    wonders: readonly WonderView[];
    /** Wonders a choice of the view can build now. */
    buildable?: ReadonlySet<string>;
    /** When given, the wonders still to build are buttons (own seat). */
    onselect?: (wonder: WonderView) => void;
    onclose: () => void;
  }

  let { open, name, seat, wonders, buildable, onselect, onclose }: Props = $props();

  const titleId = 'wonder-list-title';
  const tally = $derived(wonderTally({ wonders }));
</script>

<Sheet bind:open={() => open, (v) => !v && onclose()} {titleId}>
  <div class="flex flex-col gap-3 px-4 pt-1 pb-2 split:pt-4" data-wonder-list={seat}>
    <div class="flex items-center gap-2">
      <Icon
        name={seat === 0 ? 'crest-sun' : 'crest-moon'}
        class="shrink-0 text-xl {seat === 0 ? 'text-sun-ink' : 'text-moon-ink'}"
      />
      <h2 id={titleId} class="min-w-0 flex-1 font-display text-xl text-fg">
        <span>{t('board.wonders.title', { name })}</span>
      </h2>
      <span class="shrink-0 text-sm font-semibold text-fg-muted tabular-nums">
        {t('board.wonders.tally', { built: tally.built, total: tally.total })}
      </span>
    </div>
    <WonderList {wonders} {buildable} {onselect} />
  </div>
</Sheet>
