<script lang="ts">
  // The tutorial's completion card (the tutorial design, section 2.5): a
  // centred DismissLayer when Age II begins. It names the three victories
  // with their icons and offers Keep playing (close; the game goes on
  // against the bot), New game (Home with the vs-AI sheet open) and Home.
  // With keepInside, its buttons work and a tap outside is Keep playing.
  // A leaf with plain props; GameScreen decides when it is due.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import Button from '../shell/Button.svelte';
  import DismissLayer from '../shell/DismissLayer.svelte';

  interface Props {
    open: boolean;
    /** Keep playing, a tap outside or Escape. */
    onkeep: () => void;
    onnew: () => void;
    onhome: () => void;
  }

  let { open, onkeep, onnew, onhome }: Props = $props();
  const uid = $props.id();

  const WINS = [
    { id: 'military', icon: 'military-pawn', text: 'tutorial.done.military' },
    { id: 'science', icon: 'science-wheel', text: 'tutorial.done.science' },
    { id: 'civilian', icon: 'vp-laurel', text: 'tutorial.done.civilian' }
  ] as const;
</script>

<DismissLayer
  bind:open={() => open, (v) => !v && onkeep()}
  keepInside
  labelledby="{uid}-title"
  class="w-[min(24rem,calc(100vw-1.5rem))] p-5 short:p-3"
>
  <div
    class="flex max-h-[calc(100dvh-3rem)] flex-col gap-3 overflow-y-auto short:gap-2"
    data-tutorial-done
  >
    <h2 id="{uid}-title" class="font-display text-xl text-fg">{t('tutorial.done.title')}</h2>
    <p class="text-sm text-fg-muted">{t('tutorial.done.text')}</p>
    <ul class="flex flex-col gap-2 short:gap-1">
      {#each WINS as win (win.id)}
        <li class="flex items-center gap-3 text-sm text-fg" data-victory={win.id}>
          <Icon name={win.icon} class="shrink-0 text-2xl text-gild" />
          <span>{t(win.text)}</span>
        </li>
      {/each}
    </ul>
    <div class="mt-1 flex flex-col gap-2">
      <Button variant="primary" full data-control="tutorial-keep" onclick={onkeep}>
        {t('tutorial.done.keep')}
      </Button>
      <div class="grid grid-cols-2 gap-2">
        <Button variant="secondary" full data-control="tutorial-new" onclick={onnew}>
          {t('tutorial.done.new')}
        </Button>
        <Button variant="secondary" full data-control="tutorial-home" onclick={onhome}>
          {t('tutorial.done.home')}
        </Button>
      </div>
    </div>
  </div>
</DismissLayer>
