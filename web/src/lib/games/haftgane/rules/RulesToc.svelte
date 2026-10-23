<script lang="ts">
  // The table of contents of Haftgane's rules reference, in Duel's two
  // forms (components/rules/RulesToc.svelte):
  //
  // - 'list' (the /rules/haftgane page): a numbered list of links to #<id>
  //   on the page, so each entry is a real deep link that can be copied or
  //   opened anew.
  // - 'chips' (the in-game panel): one row of chips that scrolls sideways;
  //   a chip calls `onpick` with its section id, and the panel scrolls
  //   itself, so the game's own URL never changes.
  //
  // A leaf with plain props, so the gallery renders it standalone.
  import { formatNumber } from '$lib/i18n';
  import { ht } from '../i18n/messages.svelte';
  import {
    HG_RULE_SECTIONS,
    hgPlainText,
    type HgRuleSection,
    type HgRuleSectionId
  } from './sections';

  interface Props {
    variant?: 'list' | 'chips';
    sections?: readonly HgRuleSection[];
    /** For 'chips': called with the picked section. */
    onpick?: (section: HgRuleSectionId) => void;
    class?: string;
  }

  let {
    variant = 'list',
    sections = HG_RULE_SECTIONS,
    onpick,
    class: className = ''
  }: Props = $props();

  const uid = $props.id();
</script>

<nav aria-labelledby="{uid}-toc" class={className} data-rules-toc={variant}>
  {#if variant === 'chips'}
    <h3 id="{uid}-toc" class="sr-only">{ht('hg.rules.toc')}</h3>
    <ul class="chips -mx-4 flex gap-2 overflow-x-auto px-4 py-1">
      {#each sections as section (section.id)}
        <li class="shrink-0">
          <button
            type="button"
            class="flex min-h-11 items-center rounded-full border border-rule bg-panel-sunken px-3 text-sm font-semibold whitespace-nowrap text-fg hover:border-rule-strong"
            data-rules-toc-item={section.id}
            onclick={() => onpick?.(section.id)}
          >
            {hgPlainText(section.title)}
          </button>
        </li>
      {/each}
    </ul>
  {:else}
    <h2 id="{uid}-toc" class="text-sm font-semibold tracking-wide text-fg-muted uppercase">
      {ht('hg.rules.toc')}
    </h2>
    <ol class="mt-2 flex flex-col">
      {#each sections as section, i (section.id)}
        <li>
          <a
            href="#{section.id}"
            class="flex min-h-11 items-center gap-3 rounded-control px-2 text-sm font-semibold text-fg hover:bg-panel-sunken"
            data-rules-toc-item={section.id}
          >
            <span class="w-5 shrink-0 text-end text-fg-muted tabular-nums"
              >{formatNumber(i + 1)}</span
            >
            {hgPlainText(section.title)}
          </a>
        </li>
      {/each}
    </ol>
  {/if}
</nav>

<style>
  /* The chip row scrolls sideways inside the panel, never the page. */
  .chips {
    scrollbar-width: none;
    overscroll-behavior-inline: contain;
  }
  .chips::-webkit-scrollbar {
    display: none;
  }
</style>
