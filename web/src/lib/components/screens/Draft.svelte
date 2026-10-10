<script lang="ts">
  // The wonder draft (phase wonderDraft): which round it is, the pick order
  // as chips, the four wonders on offer, both players' picks so far, and a
  // prompt naming whose pick it is.
  //
  // On a phone a wonder tile opens a sheet with the wonder's full text and
  // "Take <wonder>"; where there is room (the split layout) every tile shows
  // its text and its own Take button instead. The Take buttons carry the
  // view's draft keys in data-act; an offered tile carries the same key in
  // data-pick. Nothing here decides what may be taken: a wonder is takeable
  // exactly when the view lists a draft choice for it.
  //
  // The pick order chips restate the printed draft order (in round 1 the
  // first player picks once, the other twice, then the first player gets
  // the last wonder; round 2 is the mirror), from the view's firstPlayer
  // and round, for display only. Whose pick it is now is the view's toMove.
  import { MediaQuery } from 'svelte/reactivity';
  import { wonderBySlug } from '$lib/catalog';
  import { formatNumber, t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { describeWonder } from '$lib/view/describe';
  import type { View } from '$lib/wire';
  import { costText, effectSentence, wonderName } from '../card/effect-text';
  import Sheet from '../shell/Sheet.svelte';
  import WonderTile from '../wonder/WonderTile.svelte';
  import {
    asSeat,
    isYou,
    otherSeat,
    seatNames,
    tryGameContext,
    type SeatNames
  } from '../choices/room';

  interface Props {
    view: View;
    /** Whether this client may act now (presenter idle and session.canAct). */
    canAct: boolean;
    /** Sends one of the view's choice keys. */
    onact: (key: string) => void;
    /** Display names by seat; read from the room when not given. */
    names?: SeatNames;
  }

  let { view, canAct, onact, names: namesProp }: Props = $props();

  const ctx = tryGameContext();
  const names = $derived(namesProp ?? seatNames(ctx?.presenter.frame?.room));
  const mode = $derived(ctx?.presenter.frame?.room.mode);

  // The split layout (app.css @custom-variant split): a desktop width, or a
  // phone held sideways.
  const split = new MediaQuery(
    '(min-width: 64rem), (orientation: landscape) and (max-height: 32rem)'
  );

  const round = $derived(view.draft?.round ?? 1);
  const offer = $derived(view.draft?.offer ?? []);
  const keys = $derived(
    new Map(
      view.choices.flatMap((c) =>
        c.kind === 'draft' && c.wonder !== undefined ? [[c.wonder, c.key] as const] : []
      )
    )
  );

  const firstOfRound = $derived(
    round === 1 ? asSeat(view.firstPlayer) : otherSeat(asSeat(view.firstPlayer))
  );
  const order = $derived([
    firstOfRound,
    otherSeat(firstOfRound),
    otherSeat(firstOfRound),
    firstOfRound
  ]);
  const picksDone = $derived(offer.filter((o) => o.takenBy !== undefined).length);
  const toMove = $derived(view.toMove === undefined ? null : asSeat(view.toMove));
  const youMove = $derived(toMove !== null && isYou(toMove, view.viewer, mode));

  const prompt = $derived.by(() => {
    if (toMove === null) return '';
    if (keys.size > 0) {
      return youMove
        ? t('choices.draft.yourPick')
        : t('choices.draft.namePick', { name: names[toMove] });
    }
    return t('choices.draft.waiting', { name: names[toMove] });
  });

  function chipLabel(i: number): string {
    const seat = order[i] ?? 0;
    const params = { n: i + 1, name: names[seat] };
    if (i < picksDone) return t('choices.draft.pickDone', params);
    if (i === 3) return t('choices.draft.pickAuto', params);
    if (i === picksDone) return t('choices.draft.pickNow', params);
    return t('choices.draft.pick', params);
  }

  // The phone detail sheet: the wonder it shows, or null when closed.
  let detail = $state<string | null>(null);

  function take(key: string): void {
    detail = null;
    onact(key);
  }

  function effectLines(slug: string): string[] {
    return (wonderBySlug(slug)?.effects ?? []).map(effectSentence);
  }

  const seats = [0, 1] as const;
</script>

{#snippet takeButton(slug: string, key: string | undefined)}
  <button
    type="button"
    class="inline-flex h-12 w-full items-center justify-center rounded-control bg-accent px-4 text-[15px] font-semibold
      text-on-accent hover:brightness-105 active:brightness-95 disabled:cursor-not-allowed disabled:opacity-50"
    data-act={key}
    disabled={!canAct || key === undefined}
    onclick={() => key !== undefined && take(key)}
  >
    {t('choices.draft.take', { wonder: wonderName(slug) })}
  </button>
{/snippet}

{#snippet effects(slug: string, cls: string)}
  <ul class="flex flex-col gap-0.5 {cls}">
    {#each effectLines(slug) as line, i (i)}
      <li>{line}.</li>
    {/each}
  </ul>
{/snippet}

<div
  class="draft mx-auto flex h-full w-full max-w-5xl flex-col gap-3 overflow-y-auto px-4 pt-3 pb-4 split:gap-4"
  data-draft
  data-round={round}
>
  <header class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
    <div class="flex items-baseline gap-2">
      <h2 class="font-display text-xl text-fg">{t('choices.draft.title')}</h2>
      <span class="text-sm text-fg-muted">{t('choices.draft.round', { round })}</span>
    </div>
    <ol class="flex items-center gap-1" aria-label={t('choices.draft.order')}>
      {#each order as seat, i (i)}
        {@const done = i < picksDone}
        {@const now = i === picksDone}
        <li
          class="flex h-8 items-center gap-1 rounded-full border-2 px-2 text-[12px] font-semibold
            {seat === 0 ? 'border-sun' : 'border-moon'}
            {done ? (seat === 0 ? 'bg-sun text-ink' : 'bg-moon text-white') : 'bg-panel text-fg'}
            {now ? 'ring-2 ring-focus ring-offset-1 ring-offset-table' : ''}"
          aria-label={chipLabel(i)}
          aria-current={now ? 'step' : undefined}
          data-pick-chip={i}
          data-seat={seat}
          data-done={done}
        >
          <Icon name={seat === 0 ? 'crest-sun' : 'crest-moon'} class="text-[15px]" />
          <span aria-hidden="true">{formatNumber(i + 1)}</span>
        </li>
      {/each}
    </ol>
  </header>

  <p
    class="flex min-h-11 items-center gap-2 rounded-control border border-rule bg-panel px-3 text-sm font-semibold text-fg"
    role="status"
    aria-live="polite"
    data-draft-prompt
  >
    {#if toMove !== null}
      <Icon
        name={toMove === 0 ? 'crest-sun' : 'crest-moon'}
        class="shrink-0 text-lg {toMove === 0 ? 'text-sun-ink' : 'text-moon-ink'}"
      />
    {/if}
    <span dir="auto">{prompt}</span>
  </p>

  <ul
    class="grid grid-cols-2 gap-2.5 sm:gap-3 split:grid-cols-4"
    aria-label={t('choices.draft.offer')}
  >
    {#each offer as item (item.wonder)}
      {@const key = keys.get(item.wonder)}
      {@const takenBy = item.takenBy === undefined ? undefined : names[asSeat(item.takenBy)]}
      <li class="flex min-w-0 flex-col gap-2" data-wonder-offer={item.wonder}>
        {#if split.current}
          <WonderTile slug={item.wonder} size="draft" {takenBy} />
          {#if item.takenBy === undefined}
            {@render effects(item.wonder, 'text-[13px] leading-snug text-fg')}
            {#if keys.size > 0}
              <div class="mt-auto">{@render takeButton(item.wonder, key)}</div>
            {/if}
          {/if}
        {:else}
          <button
            type="button"
            class="tile block w-full rounded-[10px] text-start transition-transform duration-(--dur-fast)
              disabled:cursor-default"
            aria-label={describeWonder(item.wonder)}
            data-pick={key}
            disabled={item.takenBy !== undefined}
            onclick={() => (detail = item.wonder)}
          >
            <WonderTile slug={item.wonder} size="draft" {takenBy} compactArt />
          </button>
        {/if}
      </li>
    {/each}
  </ul>

  <section class="grid gap-2 split:grid-cols-2" aria-label={t('choices.draft.title')}>
    {#each seats as seat (seat)}
      {@const wonders = view.players[seat].wonders}
      <div
        class="flex min-w-0 items-start gap-2 rounded-control border bg-panel px-3 py-2
          {toMove === seat ? (seat === 0 ? 'border-sun' : 'border-moon') : 'border-rule'}"
        data-draft-picks={seat}
      >
        <h3
          class="flex h-7 max-w-[7rem] min-w-0 shrink-0 items-center gap-1.5 font-sans text-sm font-semibold
            {seat === 0 ? 'text-sun-ink' : 'text-moon-ink'}"
        >
          <Icon name={seat === 0 ? 'crest-sun' : 'crest-moon'} class="shrink-0 text-base" />
          <span class="truncate" dir="auto" aria-hidden="true">{names[seat]}</span>
          <span class="sr-only">{t('choices.draft.picked', { name: names[seat] })}</span>
        </h3>
        <ul class="flex min-w-0 flex-1 flex-wrap gap-1.5">
          {#each [0, 1, 2, 3] as i (i)}
            {@const w = wonders[i]}
            {#if w}
              <li
                class="flex h-7 max-w-full min-w-0 items-center gap-1 rounded-[8px] border border-(color:--gold-deep)
                  bg-card-paper px-2 text-[12px] text-card-ink"
                data-picked={w.wonder}
              >
                <Icon name="wonder" class="shrink-0 text-[13px] text-(--lapis)" />
                <span class="truncate font-semibold">{wonderName(w.wonder)}</span>
              </li>
            {:else}
              <li class="size-7 shrink-0 rounded-[8px] border border-dashed border-rule-strong">
                <span class="sr-only">{t('choices.draft.emptySlot')}</span>
              </li>
            {/if}
          {/each}
        </ul>
      </div>
    {/each}
  </section>
</div>

<Sheet
  bind:open={() => detail !== null, (open) => !open && (detail = null)}
  titleId="draft-wonder-title"
>
  {#if detail}
    {@const slug = detail}
    {@const item = offer.find((o) => o.wonder === slug)}
    {@const wonder = wonderBySlug(slug)}
    <div class="flex flex-col gap-3 px-5 pt-1 pb-2">
      <div class="mx-auto w-full max-w-[260px]">
        <WonderTile {slug} size="draft" />
      </div>
      <h2 id="draft-wonder-title" class="font-display text-xl text-fg">{wonderName(slug)}</h2>
      {#if wonder}
        <p class="text-sm text-fg-muted">
          {t('choices.draft.cost', { cost: costText(wonder.cost) })}
        </p>
      {/if}
      {@render effects(slug, 'text-sm text-fg')}
      {#if item?.takenBy === undefined}
        {@render takeButton(slug, keys.get(slug))}
        {#if !keys.has(slug) && toMove !== null}
          <p class="text-center text-xs text-fg-muted" dir="auto">
            {t('choices.draft.waiting', { name: names[toMove] })}
          </p>
        {/if}
      {/if}
    </div>
  {/if}
</Sheet>

<style>
  .tile:not(:disabled):hover {
    translate: 0 -2px;
  }
</style>
