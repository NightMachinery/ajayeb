<script lang="ts">
  import { i18n, formatNumber } from '$lib/i18n';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import Button from '$lib/components/shell/Button.svelte';
  import type { Choice } from '$lib/wire/haftgane.gen';
  import { ht } from '../i18n/messages.svelte';
  import { board } from './model';
  import { effectText } from './effect-text';
  import CostPips from './CostPips.svelte';
  import ResourceTile from './ResourceTile.svelte';

  interface Props {
    boardId: string;
    choices: readonly Choice[];
    pendingKey: string | null;
    locked: boolean;
    ondraft: (key: string) => void;
    onconfirm: (key: string) => void;
    onclose: () => void;
  }
  let { boardId, choices, pendingKey, locked, ondraft, onconfirm, onclose }: Props = $props();
  let open = $state(true);
  const b = $derived(board(boardId));
  const sides = $derived(choices.filter((c) => c.kind === 'side'));
  const selected = $derived(sides.find((c) => c.key === pendingKey));
</script>

<Sheet
  bind:open={
    () => open,
    (value) => {
      open = value;
      if (!value) onclose();
    }
  }
  titleId="hg-side-title"
  class="hg-side-picker"
  panelData={{ 'data-side-picker': boardId }}
>
  <div class="content" dir={i18n.dir}>
    <h2 id="hg-side-title">
      {ht('hg.special.side.title', { wonder: ht(`hg.board.${boardId}` as 'hg.board.babylon') })}
    </h2>
    <p>{ht('hg.special.side.private')}</p>
    {#if b}<p class="start">
        {ht('hg.special.side.start')}
        <span
          role="img"
          aria-label={ht(`hg.resource.${b.start}`)}
          {@attach tooltip({ text: ht(`hg.resource.${b.start}`) })}
          ><ResourceTile res={b.start} /></span
        >
      </p>{/if}
    <div class="sides">
      {#each sides as c (c.key)}
        <section class:selected={selected?.key === c.key}>
          <button
            type="button"
            data-act={c.key}
            data-pick={c.key}
            aria-pressed={selected?.key === c.key}
            disabled={locked}
            onclick={() => ondraft(c.key)}
            >{ht(`hg.screen.side.${c.side}` as 'hg.screen.side.day')}</button
          >
          <ol>
            {#each (c.side === 'night' ? b?.night : b?.day) ?? [] as stage, i (i)}
              <li>
                <h3>{ht('hg.special.side.stage', { stage: formatNumber(i + 1) })}</h3>
                <CostPips cost={stage.cost} />
                <p>{stage.effects.map(effectText).join(' ')}</p>
              </li>
            {/each}
          </ol>
        </section>
      {/each}
    </div>
    <Button variant="secondary" data-act="side-close" onclick={onclose}
      >{ht('hg.screen.pick.close')}</Button
    >
  </div>
  {#snippet footer()}<div dir={i18n.dir}>
      <Button
        class="confirm"
        data-act={selected?.key ?? 'side-confirm'}
        data-special-confirm
        disabled={locked || !selected}
        onclick={() => {
          if (selected) onconfirm(selected.key);
        }}
        >{selected
          ? ht('hg.special.side.confirm', {
              side: ht(`hg.screen.side.${selected.side}` as 'hg.screen.side.day')
            })
          : ht('hg.screen.bar.side')}</Button
      >
    </div>{/snippet}
</Sheet>

<style>
  :global(dialog.sheet.hg-side-picker) {
    width: min(100%, 52rem);
    --sheet-max-height: calc(100dvh - 0.5rem);
  }
  .content {
    padding: 0.6rem 1rem 1rem;
    color: var(--fg);
  }
  h2 {
    margin: 0;
    font: 600 1.5rem/1.15 var(--type-display);
  }
  p {
    line-height: 1.4;
  }
  .start {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .sides {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.6rem;
    margin-block: 1rem;
  }
  section {
    min-width: 0;
    padding: 0.5rem;
    border: 2px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel-sunken);
  }
  section.selected {
    border-color: var(--accent);
    background: var(--panel);
  }
  section button {
    width: 100%;
    min-height: 44px;
    min-width: 44px;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-control);
    background: var(--panel);
    color: var(--fg);
    font: 600 1.1rem var(--type-display);
    cursor: pointer;
  }
  ol {
    padding: 0;
    list-style: none;
  }
  li + li {
    border-block-start: 1px solid var(--rule);
    padding-block-start: 0.5rem;
  }
  h3 {
    margin: 0 0 0.4rem;
    font: 600 0.95rem var(--type-display);
  }
  li p {
    font-size: 0.85rem;
    overflow-wrap: anywhere;
  }
  :global(.hg-side-picker .confirm) {
    width: 100%;
    height: auto;
    min-height: 44px;
    white-space: normal;
  }
</style>
