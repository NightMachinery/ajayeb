<script lang="ts">
  // The body of Haftgane's in-game rules overlay, the same in the side panel
  // and in the bottom sheet: a sticky header (the title, Close, and the
  // contents as a row of chips), then the whole rules reference, and a link
  // to the full page. It scrolls inside its container (the panel or the
  // sheet), never the page. Duel's is components/rules/RulesPanel.svelte.
  //
  // `section` and `request` come from openHaftganeRules(): each new request
  // scrolls to its section. Opening with no section keeps the place the
  // reader left. A leaf with plain props, so the gallery renders it
  // standalone. Its link has no icon, unlike Duel's: importing the same
  // icon would make the bundler serve it from Duel's panel chunk.
  import { tick } from 'svelte';
  import { resolve } from '$app/paths';
  import X from '@lucide/svelte/icons/x';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import RulesContent from './RulesContent.svelte';
  import RulesToc from './RulesToc.svelte';
  import type { HgRuleSectionId } from './sections';

  interface Props {
    /** The heading's id, for the container's aria-labelledby. */
    titleId: string;
    onclose?: () => void;
    section?: HgRuleSectionId;
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
  function scrollToSection(id: HgRuleSectionId, behavior: ScrollBehavior = 'auto'): void {
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

<div bind:this={root} class="rules-panel flex flex-col" data-rules-panel="haftgane">
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
        {ht('hg.rules.title')}
      </h2>
      {#if onclose}
        <button
          type="button"
          class="-me-2 grid size-11 shrink-0 place-items-center rounded-control text-fg hover:bg-panel-sunken"
          aria-label={ht('hg.rules.close')}
          data-control="rules-close"
          onclick={onclose}
          {@attach tooltip({ text: ht('hg.rules.close') })}
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
      href={resolve('/rules/haftgane')}
      target="_blank"
      rel="noopener"
      class="inline-flex min-h-11 w-fit items-center text-sm font-semibold text-fg underline decoration-gild underline-offset-2"
      data-control="rules-open-page"
    >
      {ht('hg.rules.openPage')}
    </a>
  </div>
</div>
