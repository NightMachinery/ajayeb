<script lang="ts">
  import { t } from './i18n/messages';
  import { createLobby } from '$lib/games/haftgane/session/create';
  import { saveName, cleanName } from '$lib/session/identity';
  import Button from '$lib/components/shell/Button.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  let {
    open = $bindable(false),
    name,
    suggestion,
    onStart
  }: {
    open?: boolean;
    name: string;
    suggestion: string;
    onStart: (code: string) => Promise<void>;
  } = $props();
  let typed = $state<string | null>(null);
  let starting = $state(false);
  let failed = $state(false);
  const uid = $props.id();
  async function start() {
    if (starting) return;
    starting = true;
    failed = false;
    try {
      const chosenName = saveName(cleanName(typed ?? name) || suggestion);
      const code = await createLobby(4, {}, { name: chosenName });
      await onStart(code);
    } catch {
      failed = true;
    } finally {
      starting = false;
    }
  }
</script>

<Sheet bind:open titleId={`${uid}-title`}>
  <form
    class="flex flex-col gap-4 p-5"
    data-new-table
    onsubmit={(e) => {
      e.preventDefault();
      void start();
    }}
  >
    <h2 id={`${uid}-title`} class="font-display text-xl">{t('lobby.new')}</h2>
    <p class="text-sm text-fg-muted">{t('lobby.newHint')}</p>
    <label for={`${uid}-name`} class="font-semibold">{t('lobby.name')}</label>
    <input
      id={`${uid}-name`}
      class="min-h-12 rounded-control border border-rule bg-panel-sunken px-3 text-fg"
      name="name"
      dir="auto"
      autocomplete="nickname"
      maxlength="20"
      placeholder={suggestion}
      value={typed ?? name}
      oninput={(e) => (typed = e.currentTarget.value)}
      data-name-input
    />
    {#if failed}<p role="alert">{t('lobby.failed')}</p>{/if}
    <Button type="submit" disabled={starting} data-act="lobby-create">{t('lobby.create')}</Button>
  </form>
</Sheet>
