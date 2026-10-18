<script lang="ts">
  // /dev/gallery, the sound board (M10.S04): one file and synth button per cue
  // (fx/sfx.ts) for auditioning, a volume step, and one button that plays a
  // busy frame's cues the way the Presenter does (each kind once, spaced).
  // Plays whether or not the Sound setting is on: this page is for listening.
  import { t, type MessageKey } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import Segmented from '$lib/components/shell/Segmented.svelte';
  import { CUES, play, unlockAudio, type Cue } from '$lib/fx';
  import { preloadAudio } from '$lib/fx/sfx';
  import { planCues } from '$lib/fx/cues';
  import { SOUND_VOLUMES, type SoundVolume } from '$lib/ui/settings.svelte';
  import type { GameEvent } from '$lib/wire';

  const CUE_LABEL = {
    slide: 'dev.gallery.sound.cue.slide',
    coin: 'dev.gallery.sound.cue.coin',
    flip: 'dev.gallery.sound.cue.flip',
    drum: 'dev.gallery.sound.cue.drum',
    chime: 'dev.gallery.sound.cue.chime',
    fanfare: 'dev.gallery.sound.cue.fanfare',
    tick: 'dev.gallery.sound.cue.tick',
    wonder: 'dev.gallery.sound.cue.wonder',
    token: 'dev.gallery.sound.cue.token',
    loot: 'dev.gallery.sound.cue.loot',
    ageStart: 'dev.gallery.sound.cue.ageStart',
    defeat: 'dev.gallery.sound.cue.defeat',
    yourTurn: 'dev.gallery.sound.cue.yourTurn',
    destroy: 'dev.gallery.sound.cue.destroy',
    rebuild: 'dev.gallery.sound.cue.rebuild',
    libraryDraw: 'dev.gallery.sound.cue.libraryDraw',
    extraTurn: 'dev.gallery.sound.cue.extraTurn'
  } as const satisfies Record<Cue, MessageKey>;
  const VOLUME_LABEL = {
    quiet: 'home.settings.volumeQuiet',
    medium: 'home.settings.volumeMedium',
    loud: 'home.settings.volumeLoud'
  } as const satisfies Record<SoundVolume, MessageKey>;

  let volume = $state<SoundVolume>('medium');
  const volumeOptions = $derived(
    SOUND_VOLUMES.map((v) => ({ value: v, label: t(VOLUME_LABEL[v]) }))
  );

  let loading = $state(false);
  let unavailable = $state<Cue | null>(null);

  async function audition(cue: Cue, synth = false): Promise<void> {
    unlockAudio();
    if (!synth) {
      loading = true;
      try {
        await preloadAudio();
      } finally {
        loading = false;
      }
    }
    unavailable = play(cue, { volume, source: synth ? 'synth' : 'file' }) ? null : cue;
  }

  // The events of the looting-double scenario's Arsenal: a build that pays,
  // pushes the pawn and takes both looting tokens.
  const BUSY: GameEvent[] = [
    { kind: 'build', private: false, seat: 0, card: 'arsenal' },
    { kind: 'pay', private: false, seat: 0, amount: 3 },
    { kind: 'pawn', private: false, seat: 0, from: 2, to: 6 },
    { kind: 'loot', private: false, seat: 1, amount: 2 },
    { kind: 'loot', private: false, seat: 1, amount: 5 }
  ];

  function playBusy(): void {
    unlockAudio();
    const plan = planCues(
      { code: 'X', version: 2, events: BUSY },
      { code: 'X', version: 1, events: [] }
    );
    for (const { cue, delayMs } of plan.cues) play(cue, { delayMs, volume });
  }
</script>

<div class="flex flex-col gap-4 rounded-panel border border-rule bg-panel p-4" data-sound-board>
  <p class="max-w-prose text-sm text-fg-muted">{t('dev.gallery.sound.intro')}</p>
  <Segmented
    name="gallery-volume"
    legend={t('dev.gallery.sound.volume')}
    options={volumeOptions}
    bind:value={() => volume, (v) => (volume = v as SoundVolume)}
  />
  <div class="grid gap-3 sm:grid-cols-2">
    {#each CUES as cue (cue)}
      <div class="flex flex-col gap-1" data-sound-cue={cue}>
        <span class="text-sm font-medium">{t(CUE_LABEL[cue])}</span>
        <div class="grid grid-cols-2 gap-2">
          <Button
            variant="secondary"
            data-cue={cue}
            aria-label={t('dev.gallery.sound.fileFor', { cue: t(CUE_LABEL[cue]) })}
            disabled={loading}
            onclick={() => audition(cue)}
          >
            {t('dev.gallery.sound.file')}
          </Button>
          <Button
            variant="secondary"
            data-synth={cue}
            aria-label={t('dev.gallery.sound.synthFor', { cue: t(CUE_LABEL[cue]) })}
            onclick={() => audition(cue, true)}
          >
            {t('dev.gallery.sound.synth')}
          </Button>
        </div>
        {#if unavailable === cue}
          <span class="text-xs text-fg-muted" role="status">
            {t('dev.gallery.sound.unavailable')}
          </span>
        {/if}
      </div>
    {/each}
  </div>
  <div class="flex flex-col gap-1">
    <Button variant="primary" data-cue="busy" onclick={playBusy}>
      {t('dev.gallery.sound.all')}
    </Button>
    <span class="text-xs text-fg-muted">{t('dev.gallery.sound.allHint')}</span>
  </div>
</div>
