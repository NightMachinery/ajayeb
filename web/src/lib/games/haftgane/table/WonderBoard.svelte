<script lang="ts">
  // Seven intact paintings, shared by both sides. Rules stay in live text
  // and stage cells; the scene is never stretched or cropped into a panorama.
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { HgStage } from '$lib/wire/haftgane-catalog.gen';
  import { ht } from '../i18n/messages.svelte';
  import Painting from '../art/Painting.svelte';
  import { wonderArtId, WONDER_ART_SIZES } from '../art/catalog';
  import { board } from './model';
  import HgGlyph from './HgGlyph.svelte';
  import ResourceTile from './ResourceTile.svelte';
  import CityStages from './CityStages.svelte';

  let {
    boardId,
    side = '',
    stages = [],
    built = 0,
    under = null
  }: {
    boardId: string;
    side?: string;
    stages?: readonly HgStage[];
    built?: number;
    under?: { cards: readonly string[]; own: boolean } | null;
  } = $props();
  const b = $derived(board(boardId));
  const name = $derived(ht(`hg.board.${boardId}` as 'hg.board.gizah'));
  const sideLabel = $derived(
    side === 'day' || side === 'night' ? ht(`hg.screen.side.${side}`) : ''
  );
</script>

<div class="board" data-wonder-board={boardId} data-wonder-side={side || undefined}>
  <div class="heading">
    <span class="name" data-wonder-name>{name}</span>
    {#if sideLabel}
      <span class="side" {@attach tooltip({ text: sideLabel })}>
        <HgGlyph name={side === 'day' ? 'crest-sun' : 'crest-moon'} />{sideLabel}
      </span>
    {/if}
    {#if b}
      <span
        class="start"
        role="img"
        aria-label={ht('hg.special.side.start') + ' ' + ht(`hg.resource.${b.start}`)}
        {@attach tooltip({
          text: ht('hg.special.side.start'),
          lines: [ht(`hg.resource.${b.start}`)]
        })}
      >
        <ResourceTile res={b.start} />
      </span>
    {/if}
  </div>
  <div class="scene">
    <span class="painting" role="img" aria-label={name} {@attach tooltip({ text: name })}>
      <Painting id={wonderArtId(boardId)} sizes={WONDER_ART_SIZES}>
        {#snippet fallback()}<HgGlyph name="wonder" />{/snippet}
      </Painting>
    </span>
  </div>
  {#if stages.length}<CityStages {stages} {built} {under} compact />{/if}
</div>

<style>
  .board {
    overflow: hidden;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel-sunken);
  }
  .heading {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem 0.6rem;
    padding: 0.6rem 0.75rem 0;
  }
  .name {
    flex: 1 0 auto;
    white-space: nowrap;
    overflow-wrap: normal;
    font: 600 1.1rem/1.2 var(--type-display);
  }
  .scene {
    display: grid;
    place-items: center;
    padding: 0.4rem;
  }
  .side {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.3rem;
    font-size: 0.85rem;
  }
  .painting {
    display: grid;
    place-items: center;
    aspect-ratio: 3 / 2;
    width: 50%;
    max-width: 360px;
    justify-self: center;
  }
  .start {
    justify-self: center;
    font-size: 1.5rem;
  }
  @media (max-width: 26rem) {
    .name {
      font-size: 0.9rem;
    }
    .side {
      font-size: 0.75rem;
    }
  }
</style>
