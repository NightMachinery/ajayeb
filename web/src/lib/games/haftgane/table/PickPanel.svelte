<script lang="ts">
  // The minimal pick list: the server's choices for the selected hand card
  // (or, for a decision with no hand card, such as the wonder side, every
  // offered choice), one button per key, each labelled from that choice's
  // own fields. The card sheet with the payment list replaces it in
  // M18.card-sheet-and-payments. A pick is sent through the Session; it
  // stays changeable until confirmed.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { Choice } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import { cardName, choiceAction, choiceLabel, choiceSplit } from './labels';

  interface Props {
    /** The selected card, or null for a decision without one. */
    card: string | null;
    choices: readonly Choice[];
    pendingKey: string | null;
    locked: boolean;
    onpick: (key: string) => void;
    onclose?: () => void;
  }

  let { card, choices, pendingKey, locked, onpick, onclose }: Props = $props();

  // Without a selected card, each label names its card.
  const named = $derived(card === null);
</script>

<div
  class="panel"
  role="group"
  aria-label={card
    ? ht('hg.screen.pick.title', { card: cardName(card) })
    : ht('hg.screen.bar.choose')}
  data-pick-panel={card ?? ''}
>
  {#if card}
    <span class="title">{ht('hg.screen.pick.title', { card: cardName(card) })}</span>
  {/if}
  <div class="row">
    <ul class="list">
      {#each choices as c (c.key)}
        {@const sub = choiceSplit(c)}
        <li>
          <button
            type="button"
            class="option"
            class:on={pendingKey === c.key}
            aria-pressed={pendingKey === c.key}
            disabled={locked}
            data-act={c.key}
            data-kind={c.kind}
            onclick={() => onpick(c.key)}
          >
            {#if pendingKey === c.key}<HgGlyph name="badge-check" />{/if}
            <span class="label">
              {named ? choiceLabel(c) : choiceAction(c)}
              {#if sub}<span class="sub" dir="auto">{sub}</span>{/if}
            </span>
          </button>
        </li>
      {/each}
    </ul>
    {#if card && onclose}
      <button
        type="button"
        class="close"
        aria-label={ht('hg.screen.pick.close')}
        data-act="pick-close"
        onclick={onclose}
        {@attach tooltip({ text: ht('hg.screen.pick.close') })}
      >
        <span aria-hidden="true">×</span>
      </button>
    {/if}
  </div>
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    max-height: 40dvh;
    overflow-y: auto;
    padding: 0.3rem 0.4rem 0.4rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel);
  }
  .title {
    color: var(--fg-muted);
    font-size: 0.75rem;
    font-weight: 600;
  }
  .row {
    display: flex;
    align-items: flex-start;
    gap: 0.3rem;
  }
  .list {
    display: flex;
    flex: 1 1 auto;
    flex-wrap: wrap;
    gap: 0.3rem;
    min-width: 0;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .list li {
    display: flex;
    flex: 1 1 7rem;
  }
  .option {
    display: flex;
    flex: 1 1 auto;
    align-items: center;
    gap: 0.3rem;
    min-height: 2.75rem;
    padding: 0.2rem 0.5rem;
    border: 1.5px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel-sunken);
    color: var(--fg);
    font: inherit;
    font-size: 0.875rem;
    font-weight: 600;
    line-height: 1.15;
    text-align: start;
    cursor: pointer;
  }
  .option:disabled {
    cursor: default;
    opacity: 0.6;
  }
  .on {
    border-color: var(--lapis);
    background: var(--accent-soft);
    color: var(--accent-soft-fg);
  }
  .label {
    display: flex;
    flex-direction: column;
  }
  .sub {
    color: var(--fg-muted);
    font-size: 0.75rem;
    font-weight: 400;
  }
  .close {
    display: grid;
    flex-shrink: 0;
    place-items: center;
    width: 2.75rem;
    height: 2.75rem;
    border: 0;
    border-radius: var(--shape-control);
    background: transparent;
    color: var(--fg);
    font-size: 1.4rem;
    cursor: pointer;
  }
</style>
