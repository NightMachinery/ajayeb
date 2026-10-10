<script lang="ts">
  // The hot-seat pass-device interstitial (UX design §6): not there to hide
  // information, but to orient the next player and say what just happened
  // before they pick up the device. A pure leaf: plain props, no session or
  // rules code, so the gallery and a future game screen (M5.S04 wires this
  // into real play) can both render it standalone. Fills whatever box its
  // parent gives it (the real game screen positions that box full-screen).
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import Icon from '$lib/icons/Icon.svelte';

  interface Props {
    /** The player about to take the device, for the crest colour and title. */
    nextName: string;
    nextSeat: 0 | 1;
    /** A few already-translated summary lines of what just happened. */
    summary: readonly string[];
    canUndo?: boolean;
    /** The player whose move would be undone; shown only while canUndo. */
    previousName?: string;
    onReady: () => void;
    onUndo?: () => void;
  }

  let {
    nextName,
    nextSeat,
    summary,
    canUndo = false,
    previousName,
    onReady,
    onUndo
  }: Props = $props();

  const crest = $derived(nextSeat === 0 ? 'crest-sun' : 'crest-moon');
</script>

<div
  class="flex h-full flex-col items-center justify-center gap-6 px-6 py-10 text-center"
  class:bg-sun={nextSeat === 0}
  class:text-sun-ink={nextSeat === 0}
  class:bg-moon={nextSeat === 1}
  class:text-moon-ink={nextSeat === 1}
>
  <Icon name={crest} class="text-6xl" />
  <h1 class="font-display text-3xl" dir="auto">{t('home.pass.title', { name: nextName })}</h1>
  {#if summary.length > 0}
    <ul class="flex flex-col gap-1 text-base opacity-90">
      {#each summary as line, i (i)}
        <li dir="auto">{line}</li>
      {/each}
    </ul>
  {/if}
  <div class="mt-4 flex w-full max-w-xs flex-col items-center gap-3">
    <Button variant="primary" full onclick={onReady}>{t('home.pass.ready')}</Button>
    {#if canUndo && previousName}
      <button
        type="button"
        class="min-h-11 text-sm font-semibold underline decoration-current/60 underline-offset-2"
        onclick={onUndo}
      >
        {t('home.pass.undo', { name: previousName })}
      </button>
    {/if}
  </div>
</div>
