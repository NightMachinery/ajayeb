<script lang="ts">
  // The hint settings (docs/dev/decisions.md D23) for a visual check in
  // both themes: the editor as the admin and read-only, the "On my screen"
  // local hide, the collapsible new-game section, and the lobby summary.
  // Demo state only; nothing is sent or saved.
  import HintLocalHide from '$lib/components/hints/HintLocalHide.svelte';
  import HintSettingsEditor from '$lib/components/hints/HintSettingsEditor.svelte';
  import HintsSection from '$lib/components/hints/HintsSection.svelte';
  import HintsSummary from '$lib/components/hints/HintsSummary.svelte';
  import { defaultHints, withFamily, type LocalHide } from '$lib/hints/families';
  import { t } from '$lib/i18n';
  import type { HintSettings } from '$lib/wire';

  let edited = $state<HintSettings>(withFamily(defaultHints('online'), 'prices', false));
  let draft = $state<HintSettings>(defaultHints('ai'));
  let hide = $state.raw<LocalHide>({ families: ['counting'], boardMarks: false });
</script>

<div class="flex flex-col gap-10" data-testid="gallery-hints-section">
  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Hint settings — the admin <!-- i18n-ok --></h3>
    <div class="rounded-panel border border-rule bg-panel p-4">
      <HintSettingsEditor hints={edited} onchange={(next) => (edited = next)} />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Hint settings — read-only, with the local hide <!-- i18n-ok -->
    </h3>
    <div class="flex flex-col gap-5 rounded-panel border border-rule bg-panel p-4">
      <HintSettingsEditor hints={edited} readonly readonlyNote />
      <HintLocalHide
        {hide}
        onfamily={(family, hidden) =>
          (hide = {
            ...hide,
            families: hidden
              ? [...hide.families, family]
              : hide.families.filter((f) => f !== family)
          })}
        onboardmarks={(hidden) => (hide = { ...hide, boardMarks: hidden })}
      />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      New-game Hints section (closed, then open) <!-- i18n-ok -->
    </h3>
    <div class="flex flex-col gap-3 rounded-panel border border-rule bg-panel p-4">
      <HintsSection bind:value={draft} />
      <HintsSection bind:value={draft} open />
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Lobby summary <!-- i18n-ok --></h3>
    <div class="flex flex-col gap-4 rounded-panel border border-rule bg-table p-4">
      <HintsSummary hints={edited} title={t('hints.lobby.title')} />
      <HintsSummary hints={defaultHints('ai')} title={t('hints.lobby.title')} />
    </div>
  </section>
</div>
