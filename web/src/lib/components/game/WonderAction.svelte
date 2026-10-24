<script lang="ts">
  // The card sheet's "Build a wonder" button, which opens the wonder step
  // (M18.card-sheet-signals). When the card can go under a wonder now, the
  // button stands out: a soft accent fill, an accent edge and the wonder
  // glyph, with the wonder named under the label (or how many, when there
  // are several). `ready` is the wonders the view's own choices offer for
  // this card (card-actions.ts readyWonders); the client decides nothing.
  // Otherwise it is the plain secondary button, and the wonder step says
  // why each wonder is out of reach. A leaf with plain props.
  import Icon from '$lib/icons/Icon.svelte';
  import { t } from '$lib/i18n';
  import { wonderName } from '../card/effect-text';
  import Button from '../shell/Button.svelte';

  interface Props {
    /** The wonders this card can go under now, by slug. */
    ready: readonly string[];
    onclick: () => void;
  }

  let { ready, onclick }: Props = $props();

  const note = $derived(
    ready.length === 1
      ? t('game.action.wonderReadyOne', { wonder: wonderName(ready[0]!) })
      : ready.length > 1
        ? t('game.action.wonderReadyMany', { count: ready.length })
        : ''
  );
</script>

<Button
  variant="secondary"
  full
  class="h-auto! min-h-12 py-2 leading-tight {ready.length > 0
    ? 'border-2! border-accent! bg-accent-soft! text-accent-soft-fg!'
    : ''}"
  data-control="wonder-step"
  data-wonder-ready={ready.length > 0 ? ready.join(' ') : undefined}
  {onclick}
>
  {#if ready.length > 0}<Icon name="wonder" class="shrink-0 text-xl text-accent" />{/if}
  <span class="flex min-w-0 flex-col items-center">
    <span>{t('game.action.wonder')}</span>
    {#if note}<span class="text-xs font-normal" data-wonder-ready-note>{note}</span>{/if}
  </span>
</Button>
