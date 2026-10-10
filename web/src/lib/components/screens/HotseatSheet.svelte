<script lang="ts">
  // The hot-seat ("pass and play") setup sheet (UX design §2, §6). Start
  // sends a real `create` frame (docs/protocol.md) through the Session the
  // caller supplies, so this component stays gallery-safe (no live
  // connection needed to render it): the caller passes onStart and reports
  // the outcome back through the open/busy/error props.
  //
  // "First to play" only decides which entered name lands in Create.names[0]
  // (seat 0) versus names[1] (seat 1): a live room's actual Options.FirstPlayer
  // is always "random" (internal/room/manager.go's Create path uses
  // engine.Options{}, its zero value), and Create carries no field to change
  // that. So this control cannot make a specific person's turn come first;
  // it only picks which of the two seats (and so which crest colour) each
  // name is assigned to. "Random" is therefore given its own real 50/50
  // client-side coin flip at Start, rather than silently behaving like
  // "First" would.
  import { MediaQuery } from 'svelte/reactivity';
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import Segmented from '$lib/components/shell/Segmented.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import { cleanName } from '$lib/session/identity';
  import { SessionError } from '$lib/session/session.svelte';

  interface Props {
    open?: boolean;
    onStart: (opts: { names: [string, string] }) => Promise<void> | void;
  }

  let { open = $bindable(false), onStart }: Props = $props();

  const titleId = 'hotseat-sheet-title';

  let name1 = $state('');
  let name2 = $state('');
  let firstToPlay = $state<'random' | '0' | '1'>('random');
  const coarsePointer = new MediaQuery('(pointer: coarse)', false);
  const narrowScreen = new MediaQuery('(max-width: 700px)', false);
  let passDevice = $state(coarsePointer.current);
  let orientation = $state<'face' | 'fixed'>(narrowScreen.current ? 'face' : 'fixed');
  let starting = $state(false);
  let failure = $state<string | null>(null);

  const firstOptions = $derived([
    { value: 'random', label: t('home.hotseat.firstRandom') },
    { value: '0', label: t('home.hotseat.firstOption') },
    { value: '1', label: t('home.hotseat.secondOption') }
  ]);
  const orientationOptions = [
    { value: 'face', label: t('home.hotseat.orientationFace') },
    { value: 'fixed', label: t('home.hotseat.orientationFixed') }
  ];

  async function start(): Promise<void> {
    if (starting) return;
    const cleaned: [string, string] = [cleanName(name1), cleanName(name2)];
    const swap = firstToPlay === '1' || (firstToPlay === 'random' && Math.random() < 0.5);
    const names: [string, string] = swap ? [cleaned[1], cleaned[0]] : cleaned;
    starting = true;
    failure = null;
    try {
      await onStart({ names });
    } catch (err) {
      failure = err instanceof SessionError ? t(err.messageKey) : t('session.error.unknown');
    } finally {
      starting = false;
    }
  }
</script>

<Sheet bind:open {titleId}>
  <form
    class="flex flex-col gap-5 px-5 pt-1 pb-5"
    onsubmit={(event) => {
      event.preventDefault();
      void start();
    }}
  >
    <h2 id={titleId} class="font-display text-xl text-fg">{t('home.hotseat.title')}</h2>

    <div class="flex flex-col gap-3">
      <label class="flex flex-col gap-1">
        <span class="text-sm font-medium text-fg-muted">{t('home.hotseat.player1Label')}</span>
        <input
          class="min-h-11 rounded-control border border-rule bg-panel px-3 text-fg"
          type="text"
          dir="auto"
          maxlength="20"
          placeholder={t('home.hotseat.player1Default')}
          bind:value={name1}
        />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm font-medium text-fg-muted">{t('home.hotseat.player2Label')}</span>
        <input
          class="min-h-11 rounded-control border border-rule bg-panel px-3 text-fg"
          type="text"
          dir="auto"
          maxlength="20"
          placeholder={t('home.hotseat.player2Default')}
          bind:value={name2}
        />
      </label>
    </div>

    <div class="flex flex-col gap-1">
      <Segmented
        name="hotseat-first"
        legend={t('home.hotseat.firstToPlayLegend')}
        options={firstOptions}
        bind:value={firstToPlay}
      />
    </div>

    <label class="flex min-h-11 items-center justify-between gap-3">
      <span class="flex flex-col">
        <span class="text-sm font-medium text-fg">{t('home.hotseat.passDeviceToggle')}</span>
        <span class="text-xs text-fg-muted">{t('home.hotseat.passDeviceHint')}</span>
      </span>
      <input type="checkbox" class="size-5 accent-accent" bind:checked={passDevice} />
    </label>

    <Segmented
      name="hotseat-orientation"
      legend={t('home.hotseat.orientationLegend')}
      options={orientationOptions}
      bind:value={orientation}
    />

    {#if failure}
      <p class="text-sm text-cinnabar" role="alert">{failure}</p>
    {/if}

    <Button type="submit" variant="primary" full disabled={starting}>
      {t('home.hotseat.start')}
    </Button>
  </form>
</Sheet>
