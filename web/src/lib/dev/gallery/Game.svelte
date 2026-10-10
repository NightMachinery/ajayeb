<script lang="ts">
  // M5.S04a: the game screen on recorded game states. Each scene is the
  // /dev/fixture page of a recorded view in an iframe, so the screen gets the
  // viewport size it would have on that device: its layout follows the
  // viewport class (lib/layout/breakpoints), which a box inside this page
  // could not change. The frames are scaled down to fit the gallery.
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';

  type SceneKey =
    | 'game.gallery.scene.draft'
    | 'game.gallery.scene.ageOne'
    | 'game.gallery.scene.ageTwo'
    | 'game.gallery.scene.ageTwoLate'
    | 'game.gallery.scene.ageThree'
    | 'game.gallery.scene.pending'
    | 'game.gallery.scene.gameOver'
    | 'game.gallery.scene.lobby'
    | 'game.gallery.scene.away'
    | 'game.gallery.scene.watching';

  interface Scene {
    fixture: string;
    label: SceneKey;
  }

  interface Frame {
    width: number;
    height: number;
    scale: number;
  }

  const SCENES: readonly Scene[] = [
    { fixture: 'draft-round-1-seat1', label: 'game.gallery.scene.draft' },
    { fixture: 'age-1-start-seat0', label: 'game.gallery.scene.ageOne' },
    { fixture: 'appian-economy-seat0', label: 'game.gallery.scene.ageTwo' },
    { fixture: 'costs-seat0', label: 'game.gallery.scene.ageTwoLate' },
    { fixture: 'looting-seat1', label: 'game.gallery.scene.ageThree' },
    { fixture: 'destroy-card-seat0', label: 'game.gallery.scene.pending' },
    { fixture: 'game-over-civilian-seat0', label: 'game.gallery.scene.gameOver' }
  ];

  // M7.S01: online rooms (session/fixtures.ts makes them from recorded views).
  const ONLINE_SCENES: readonly Scene[] = [
    { fixture: 'online-lobby-seat0', label: 'game.gallery.scene.lobby' },
    { fixture: 'online-away-seat0', label: 'game.gallery.scene.away' },
    { fixture: 'online-watching-spectator', label: 'game.gallery.scene.watching' }
  ];

  const PHONE: Frame = { width: 390, height: 844, scale: 0.55 };
  const LANDSCAPE: Frame = { width: 844, height: 390, scale: 0.55 };
  const DESKTOP: Frame = { width: 1280, height: 800, scale: 0.36 };
</script>

{#snippet framed(scene: Scene, frame: Frame)}
  {@const title = t('game.gallery.frame', {
    scene: t(scene.label),
    width: String(frame.width),
    height: String(frame.height)
  })}
  <figure class="flex flex-col gap-1" data-game-scene={scene.fixture}>
    <div
      class="overflow-hidden rounded-control border border-rule bg-table"
      style:width="{Math.round(frame.width * frame.scale)}px"
      style:height="{Math.round(frame.height * frame.scale)}px"
      dir="ltr"
    >
      <iframe
        class="scaled border-0"
        src={resolve('/dev/fixture/[name]', { name: scene.fixture })}
        {title}
        loading="lazy"
        width={frame.width}
        height={frame.height}
        style:transform="scale({frame.scale})"
      ></iframe>
    </div>
    <figcaption class="text-xs text-fg-muted">
      {title} ·
      <a
        class="font-semibold text-fg underline decoration-gild"
        href={resolve('/dev/fixture/[name]', { name: scene.fixture })}
      >
        {t('game.gallery.open')}
      </a>
    </figcaption>
  </figure>
{/snippet}

<div class="flex flex-col gap-4">
  <p class="max-w-prose text-sm text-fg-muted">{t('game.gallery.intro')}</p>
  <div class="flex flex-wrap gap-4">
    {#each SCENES as scene (scene.fixture)}
      {@render framed(scene, PHONE)}
    {/each}
  </div>
  <div class="flex flex-wrap gap-4">
    {@render framed(SCENES[3]!, LANDSCAPE)}
    {@render framed(SCENES[3]!, DESKTOP)}
    {@render framed(SCENES[4]!, DESKTOP)}
  </div>
  <div class="flex flex-wrap gap-4">
    {#each ONLINE_SCENES as scene (scene.fixture)}
      {@render framed(scene, PHONE)}
    {/each}
  </div>
  <div class="flex flex-wrap gap-4">
    {#each ONLINE_SCENES as scene (scene.fixture)}
      {@render framed(scene, DESKTOP)}
    {/each}
  </div>
</div>

<style>
  .scaled {
    transform-origin: 0 0;
    max-width: none;
  }
</style>
