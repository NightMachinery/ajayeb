<script lang="ts">
  // A read-only summary of a game's hint settings (docs/dev/decisions.md
  // D23): one line ("All hints on except Advisor") and a chip per family,
  // on or off. The online lobby shows it, so both players see the hints
  // before the game starts. A leaf: the caller passes the settings.
  import Check from '@lucide/svelte/icons/check';
  import X from '@lucide/svelte/icons/x';
  import { t } from '$lib/i18n';
  import { HINT_FAMILIES } from '$lib/hints/families';
  import { familyName, summaryText } from '$lib/hints/text';
  import type { HintSettings } from '$lib/wire';

  interface Props {
    hints: HintSettings;
    /** The heading; none when the caller has its own. */
    title?: string;
  }

  let { hints, title }: Props = $props();

  const uid = $props.id();
  const titleId = `${uid}-title`;
</script>

<section
  class="flex flex-col gap-2"
  aria-labelledby={title ? titleId : undefined}
  data-hints-summary
>
  {#if title}
    <h2 id={titleId} class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
      {title}
    </h2>
  {/if}
  <div class="flex flex-col gap-2 rounded-control border border-rule bg-panel px-3 py-2.5">
    <p class="text-sm font-semibold text-fg" data-hints-summary-text>{summaryText(hints)}</p>
    <ul class="flex flex-wrap gap-1.5">
      {#each HINT_FAMILIES as family (family)}
        {@const on = hints[family]}
        <li
          class="flex items-center gap-1 rounded-full px-2 py-0.5 text-xs
            {on
            ? 'bg-accent-soft text-accent-soft-fg'
            : 'bg-panel-sunken text-fg-muted line-through'}"
          data-hint-family={family}
          data-on={on ? '' : undefined}
        >
          {#if on}
            <Check class="size-3.5 shrink-0" aria-hidden="true" />
          {:else}
            <X class="size-3.5 shrink-0" aria-hidden="true" />
          {/if}
          <span>{familyName(family)}</span>
          <span class="sr-only">{on ? t('hints.on') : t('hints.off')}</span>
        </li>
      {/each}
    </ul>
    <p class="text-xs text-fg-muted">{t('hints.sameForBoth')}</p>
  </div>
</section>
