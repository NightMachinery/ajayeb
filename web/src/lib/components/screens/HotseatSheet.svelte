<script lang="ts">
  // The hot-seat ("pass and play") setup sheet (UX design §2, §6). Start
  // sends a real `create` frame (docs/protocol.md) through the onStart
  // callback the caller supplies, so this component stays gallery-safe (no
  // live connection needed to render it).
  //
  // There is no "first player" control: the create frame has no field for
  // it, and the server draws the first player at random for every live room
  // (internal/room/manager.go creates rooms with engine.Options{}). A control
  // here could only reorder the names, which would claim a choice the game
  // does not honour, so the sheet says plainly that the draw is random.
  //
  // The pass-device and orientation choices are not part of the room: they
  // are this device's presentation preferences, so Start stores them in the
  // settings store (hotseatPassDevice, hotseatOrientation), where the game
  // screen reads them. Until the first hot-seat game they default by device:
  // on and "face the active player" for a coarse pointer or a narrow screen.
  import { MediaQuery } from 'svelte/reactivity';
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import Segmented from '$lib/components/shell/Segmented.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import HintsSection from '$lib/components/hints/HintsSection.svelte';
  import { hintDefaults } from '$lib/hints/local.svelte';
  import { cleanName } from '$lib/session/identity';
  import { SessionError } from '$lib/session/session.svelte';
  import { settings, type Orientation } from '$lib/ui/settings.svelte';

  interface Props {
    open?: boolean;
    onStart: (opts: { names: [string, string] }) => Promise<void> | void;
  }

  let { open = $bindable(false), onStart }: Props = $props();

  const titleId = 'hotseat-sheet-title';

  const coarsePointer = new MediaQuery('(pointer: coarse)', false);
  const narrowScreen = new MediaQuery('(max-width: 700px)', false);

  let name1 = $state('');
  let name2 = $state('');
  let passDevice = $state(settings.value.hotseatPassDevice ?? coarsePointer.current);
  let orientation = $state<Orientation>(
    settings.value.hotseatOrientation ?? (narrowScreen.current ? 'face' : 'fixed')
  );
  // The game's hint settings, from this device's last saved ones.
  let hints = $state(hintDefaults.for('hotseat'));
  let starting = $state(false);
  let failure = $state<string | null>(null);

  const orientationOptions = $derived([
    { value: 'face', label: t('home.hotseat.orientationFace') },
    { value: 'fixed', label: t('home.hotseat.orientationFixed') }
  ]);

  async function start(): Promise<void> {
    if (starting) return;
    // An empty field takes the default its placeholder shows, so the game
    // names the players the way the sheet promised.
    const names: [string, string] = [
      cleanName(name1) || t('home.hotseat.player1Default'),
      cleanName(name2) || t('home.hotseat.player2Default')
    ];
    settings.update({ hotseatPassDevice: passDevice, hotseatOrientation: orientation });
    // The game starts from these hints (the Session's create reads the
    // local default), and the next new game on this device will too.
    hintDefaults.save('hotseat', hints);
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
    class="flex flex-col gap-5 px-5 pt-1 pb-5 split:pt-5"
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
          autocomplete="off"
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
          autocomplete="off"
          placeholder={t('home.hotseat.player2Default')}
          bind:value={name2}
        />
      </label>
      <p class="text-xs text-fg-muted">{t('home.hotseat.firstRandomNote')}</p>
    </div>

    <label class="flex min-h-11 items-center justify-between gap-3">
      <span class="flex flex-col">
        <span class="text-sm font-medium text-fg">{t('home.hotseat.passDeviceToggle')}</span>
        <span class="text-xs text-fg-muted">{t('home.hotseat.passDeviceHint')}</span>
      </span>
      <input type="checkbox" class="size-5 shrink-0 accent-accent" bind:checked={passDevice} />
    </label>

    <div class="flex flex-col gap-1.5">
      <span class="text-sm font-medium text-fg-muted" aria-hidden="true">
        {t('home.hotseat.orientationLegend')}
      </span>
      <Segmented
        name="hotseat-orientation"
        legend={t('home.hotseat.orientationLegend')}
        options={orientationOptions}
        bind:value={orientation}
      />
    </div>

    <HintsSection bind:value={hints} />

    {#if failure}
      <p class="text-sm text-cinnabar" role="alert">{failure}</p>
    {/if}

    <Button type="submit" variant="primary" full disabled={starting}>
      {t('home.hotseat.start')}
    </Button>
  </form>
</Sheet>
