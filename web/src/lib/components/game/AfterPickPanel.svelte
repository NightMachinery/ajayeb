<script lang="ts">
  // The card sheet's "After this pick" panel (the hints design, M10.HS3):
  // what taking the card uncovers, who sees a turned card first, what a
  // freed card does for the opponent, and each move's threats one turn
  // ahead (view/outcome.ts afterPickLines). A disclosure: open on wider
  // screens, closed on a phone in portrait unless a line is a threat. Once
  // the player opens or closes it, this device remembers that choice (a
  // threat still opens it). A plain leaf.
  import { t } from '$lib/i18n';
  import { breakpoints } from '$lib/layout/breakpoints.svelte';
  import type { AfterPickLine } from '$lib/view/outcome';

  interface Props {
    lines: readonly AfterPickLine[];
    class?: string;
  }

  let { lines, class: className = '' }: Props = $props();

  const KEY = 'ajayeb:afterPickOpen';

  function readSaved(): boolean | null {
    try {
      const raw = globalThis.localStorage?.getItem(KEY);
      return raw === 'true' ? true : raw === 'false' ? false : null;
    } catch {
      return null;
    }
  }

  let saved = $state<boolean | null>(readSaved());

  const threat = $derived(lines.some((l) => l.threat));
  const open = $derived(threat || (saved ?? !breakpoints.phonePortrait));

  let details: HTMLDetailsElement | undefined = $state();

  // A tap or a key on the summary toggles the panel: remember where the
  // player left it (the toggle event also fires when `open` changes by
  // itself, so it cannot tell a choice).
  function remember() {
    const next = !(details?.open ?? open);
    saved = next;
    try {
      globalThis.localStorage?.setItem(KEY, String(next));
    } catch {
      // Blocked or full: the choice holds for this page.
    }
  }
</script>

{#if lines.length > 0}
  <details
    bind:this={details}
    {open}
    class="after-pick rounded-md border border-rule px-3 py-2 {className}"
    data-after-pick
    data-threat={threat ? '' : undefined}
  >
    <summary
      class="cursor-pointer text-xs font-semibold tracking-wide text-fg-muted uppercase"
      onclick={remember}
    >
      {t('hints.after.title')}
    </summary>
    <ul class="mt-1.5 flex flex-col gap-1">
      {#each lines as line (line.id)}
        <li
          class="text-sm text-fg {line.threat ? 'threat ps-2' : ''}"
          dir="auto"
          data-after-line={line.id}
        >
          {line.text}
        </li>
      {/each}
    </ul>
  </details>
{/if}

<style>
  .threat {
    border-inline-start: 3px solid var(--gold-deep);
  }
</style>
