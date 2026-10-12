<script lang="ts">
  // The body of the in-game rules overlay, the same in the side panel and in
  // the bottom sheet: a sticky header (the title, Close, and the contents as
  // a row of chips), then the whole rules reference. It scrolls inside its
  // container (the panel or the sheet), never the page.
  //
  // `section` and `request` come from openRules(): each new request scrolls
  // to its section. Opening with no section keeps the place the reader left.
  // A leaf with plain props, so the gallery renders it standalone.
  import { tick } from 'svelte';
  import { resolve } from '$app/paths';
  import { t } from '$lib/i18n';
  import X from '@lucide/svelte/icons/x';
  import ExternalLink from '@lucide/svelte/icons/external-link';
  import RulesContent from './RulesContent.svelte';
  import RulesToc from './RulesToc.svelte';
  import type { RuleSectionId } from './sections';
  import { tooltip } from '$lib/ui/tooltip.svelte';

  interface Props {
    /** The heading's id, for the container's aria-labelledby. */
    titleId: string;
    onclose?: () => void;
    section?: RuleSectionId;
    /** Changes on every open request, so the same section scrolls again. */
    request?: number;
    /** Whether the panel is on screen (it scrolls only when it is). */
    shown?: boolean;
  }

  let { titleId, onclose, section, request = 0, shown = true }: Props = $props();

  let root = $state<HTMLElement | undefined>();

  /** Scrolls the panel's container to a section. DOM only. A chip's jump
   *  follows the container's scroll-behavior (smooth, or instant with
   *  Animation off); an open request jumps at once. */
  function scrollToSection(id: RuleSectionId, behavior: ScrollBehavior = 'auto'): void {
    root
      ?.querySelector<HTMLElement>(`[data-rules-section="${id}"]`)
      ?.scrollIntoView({ block: 'start', inline: 'nearest', behavior });
  }

  // DOM-only effect: scroll to the requested section once the panel shows.
  // It writes no $state.
  $effect(() => {
    void request;
    const id = section;
    if (!shown || !id) return;
    void tick().then(() => requestAnimationFrame(() => scrollToSection(id, 'instant')));
  });
</script>

<div bind:this={root} class="rules-panel flex flex-col" data-rules-panel>
  <header
    class="sticky top-0 z-10 flex flex-col gap-1 border-b border-rule bg-panel px-4 pt-2 pb-2"
  >
    <div class="flex items-center justify-between gap-2">
      <h2
        id={titleId}
        tabindex="-1"
        class="font-display text-2xl leading-tight text-fg outline-none"
        data-rules-title
      >
        {t('rules.title')}
      </h2>
      {#if onclose}
        <button
          type="button"
          class="-me-2 grid size-11 shrink-0 place-items-center rounded-control text-fg hover:bg-panel-sunken"
          aria-label={t('rules.close')}
          data-control="rules-close"
          onclick={onclose}
          {@attach tooltip({ text: t('rules.close') })}
        >
          <X class="size-5" aria-hidden="true" />
        </button>
      {/if}
    </div>
    <RulesToc variant="chips" onpick={(id) => scrollToSection(id)} />
  </header>

  <div class="flex flex-col gap-8 px-4 pt-4 pb-6">
    <RulesContent level={3} sectionClass="scroll-mt-32" />
    <a
      href={resolve('/rules')}
      target="_blank"
      rel="noopener"
      class="inline-flex min-h-11 w-fit items-center gap-2 text-sm font-semibold text-fg underline decoration-gild underline-offset-2"
      data-control="rules-open-page"
    >
      <ExternalLink class="size-4 shrink-0" aria-hidden="true" />
      {t('rules.openPage')}
    </a>
  </div>
</div>
