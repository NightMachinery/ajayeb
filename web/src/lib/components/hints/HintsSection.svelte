<script lang="ts">
  // The Hints section of a new-game sheet: collapsed to one summary line
  // ("All hints on except Advisor"), opened to the settings editor. The
  // sheet starts it from this device's local default and saves it there
  // when the game is created (lib/hints/local.svelte.ts). A leaf: the
  // settings are a bindable value.
  import ChevronDown from '@lucide/svelte/icons/chevron-down';
  import { t } from '$lib/i18n';
  import { summaryText } from '$lib/hints/text';
  import type { HintSettings } from '$lib/wire';
  import HintSettingsEditor from './HintSettingsEditor.svelte';

  interface Props {
    value: HintSettings;
    /** Opened at first (the gallery). */
    open?: boolean;
  }

  let { value = $bindable(), open = false }: Props = $props();
</script>

<details
  class="hints-section group rounded-control border border-rule bg-panel-sunken/40"
  {open}
  data-hints-section
>
  <summary
    class="flex min-h-12 cursor-pointer list-none items-center justify-between gap-3 rounded-control px-3 py-2"
    data-control="hints-section"
  >
    <span class="flex min-w-0 flex-col">
      <span class="text-sm font-semibold text-fg">{t('hints.title')}</span>
      <span class="text-xs text-fg-muted" data-hints-summary-text>{summaryText(value)}</span>
    </span>
    <ChevronDown
      class="size-5 shrink-0 text-fg-muted transition-transform duration-(--dur-fast) group-open:rotate-180"
      aria-hidden="true"
    />
  </summary>
  <div class="px-3 pt-1 pb-3">
    <HintSettingsEditor hints={value} onchange={(next) => (value = next)} />
  </div>
</details>

<style>
  .hints-section summary::-webkit-details-marker {
    display: none;
  }
</style>
