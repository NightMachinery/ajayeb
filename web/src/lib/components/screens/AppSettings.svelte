<script lang="ts">
  // Settings' app rows (M18.installable-and-fullscreen): the Fullscreen
  // switch where the Fullscreen API exists, and Install, a button while
  // the browser offers installation or the Add to Home Screen steps on
  // iOS.
  import Share from '@lucide/svelte/icons/share';
  import { t } from '$lib/i18n';
  import { fullscreen } from '$lib/ui/fullscreen.svelte';
  import { install } from '$lib/ui/install.svelte';
</script>

{#if fullscreen.shown}
  <label class="flex min-h-11 items-center justify-between gap-3">
    <span class="flex flex-col">
      <span class="text-sm font-medium text-fg">{t('fullscreen.enter')}</span>
      <span class="text-xs text-fg-muted">{t('home.settings.fullscreenHint')}</span>
    </span>
    <input
      type="checkbox"
      role="switch"
      class="size-5 shrink-0 accent-accent"
      data-setting="fullscreen"
      checked={fullscreen.active}
      onchange={(e) => {
        // A refused request leaves the box as the page really is.
        const box = e.currentTarget;
        void fullscreen.toggle().then(() => (box.checked = fullscreen.active));
      }}
    />
  </label>
{/if}

{#if install.offer === 'prompt'}
  <div class="flex min-h-11 items-center justify-between gap-3" data-setting="install">
    <span class="text-xs text-fg-muted">{t('home.settings.installHint')}</span>
    <button
      type="button"
      class="min-h-9 shrink-0 rounded-control border border-rule bg-panel px-3 text-sm text-fg hover:bg-panel-sunken"
      data-control="install"
      onclick={() => void install.prompt()}
    >
      {t('home.settings.install')}
    </button>
  </div>
{:else if install.offer === 'ios'}
  <div class="flex flex-col gap-1" data-setting="install-ios">
    <span class="text-sm font-medium text-fg">{t('home.settings.install')}</span>
    <span class="text-xs text-fg-muted">{t('home.settings.installHint')}</span>
    <ol class="list-decimal ps-5 text-sm text-fg">
      <li>
        <Share class="inline size-4 align-text-bottom text-fg-muted" aria-hidden="true" />
        {t('home.settings.installIos1')}
      </li>
      <li>{t('home.settings.installIos2')}</li>
    </ol>
  </div>
{/if}
