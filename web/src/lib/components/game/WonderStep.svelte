<script lang="ts">
  // The card sheet's second step (card first): every wonder this card could
  // go under, each with the server's cost and either a build button (its
  // wonder:<wonder>:<card> key) or the reason it cannot be built. Plain
  // props: the options come from card-actions.ts, which only sorts the
  // view's own choices.
  //
  // Hints (M10.HS3 and HS4): under each wonder's build button, its outcome
  // line (the server's Choice.outcome for that key) with the confirm
  // step's warning, as in the card sheet; and on each wonder, the hourglass
  // of the race for the last wonder. The outcome context comes from the
  // page's game context unless `octx` is given.
  //
  // The tutorial's coach step in this step (`coach`) shows as a note above
  // the list (M10.tutorial-client), and the rails (`allow`) lock every
  // build button but the guided one, with `lockedReason` once under the list.
  import ArrowLeft from '@lucide/svelte/icons/arrow-left';
  import { hintHide } from '$lib/hints/local.svelte';
  import { wonderHint, wonderOwner } from '$lib/view/board-hints';
  import { sessionHintScreen } from '$lib/view/hints';
  import {
    blunderLine,
    outcomeContext,
    outcomeTerms,
    type OutcomeContext
  } from '$lib/view/outcome';
  import { tryGameContext } from '../choices/room';
  import OutcomeLine from './OutcomeLine.svelte';
  import { cardName, wonderName } from '$lib/components/card/effect-text';
  import { t } from '$lib/i18n';
  import { affordOf } from '$lib/view/afford';
  import WonderTile from '../wonder/WonderTile.svelte';
  import ActionButton from './ActionButton.svelte';
  import type { WonderOption } from './card-actions';
  import CostBreakdown from './CostBreakdown.svelte';
  import { reasonText } from './reasons';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import CoachNote from '../tutorial/CoachNote.svelte';
  import type { CoachNoteView } from '$lib/tutorial/coach.svelte';
  import { ALLOW_ALL, type Allow } from '$lib/tutorial/gate';

  interface Props {
    card: string;
    options: readonly WonderOption[];
    /** Wonders built by both players (View.wondersBuilt). */
    wondersBuilt: number;
    /** The paying seat's coins, for the cost lines. */
    coins?: number;
    canAct: boolean;
    confirm: boolean;
    /** The connection is up; while it is down the buttons say they wait for it. */
    connected?: boolean;
    titleId: string;
    onact: (key: string) => void;
    onback: () => void;
    /** The outcome line's context; else from the game context. */
    octx?: OutcomeContext;
    /** The tutorial's coach step here, as a note; null for none. */
    coach?: CoachNoteView | null;
    /** The tutorial's rails: whether a move key may be sent now. */
    allow?: Allow;
    /** Why a move the rails refuse is locked. */
    lockedReason?: string;
  }

  let {
    card,
    options,
    wondersBuilt,
    coins,
    canAct,
    confirm,
    connected = true,
    titleId,
    onact,
    onback,
    octx: octxProp,
    coach = null,
    allow = ALLOW_ALL,
    lockedReason = ''
  }: Props = $props();

  const ctx = tryGameContext();
  const screen = $derived(sessionHintScreen(ctx?.session, hintHide.value));
  const octx = $derived.by((): OutcomeContext | undefined => {
    if (octxProp) return octxProp;
    const view = ctx?.session.view;
    return view && screen ? outcomeContext(view, screen.families, screen.names) : undefined;
  });
  function termsOf(option: WonderOption) {
    return octx && option.choice ? outcomeTerms(option.choice.outcome, octx) : [];
  }
  function blunderOf(option: WonderOption) {
    const view = ctx?.session.view;
    return octx && confirm && option.choice && view
      ? blunderLine(option.choice, view.choices, octx)
      : undefined;
  }
  function hintOf(slug: string) {
    const view = ctx?.session.view;
    if (!view || !screen) return undefined;
    const seat = wonderOwner(view, slug);
    const w =
      seat === undefined ? undefined : view.players[seat].wonders.find((x) => x.wonder === slug);
    if (seat === undefined || !w) return undefined;
    const h = wonderHint(view, w, seat, screen, true);
    // The outcome line under the button says what it would do.
    return h?.hourglass ? { terms: [], hourglass: h.hourglass } : undefined;
  }

  // Legal wonders first, in the order the server listed them.
  const sorted = $derived([
    ...options.filter((o) => o.choice),
    ...options.filter((o) => !o.choice)
  ]);
</script>

<div class="flex flex-col gap-3 px-4 pt-1 pb-2 split:pt-4" data-wonder-step>
  <div class="flex items-center gap-2">
    <button
      type="button"
      class="grid size-11 shrink-0 place-items-center rounded-control text-fg hover:bg-panel-sunken"
      aria-label={t('game.wonders.back')}
      {@attach tooltip({ text: t('game.wonders.back') })}
      data-control="wonder-back"
      onclick={onback}
    >
      <ArrowLeft class="size-5 rtl:-scale-x-100" aria-hidden="true" />
    </button>
    <h2 id={titleId} class="min-w-0 font-display text-lg leading-tight text-fg">
      {t('game.wonders.title', { card: cardName(card) })}
    </h2>
  </div>

  {#if coach}
    <CoachNote
      title={coach.title}
      progress={coach.progress}
      text={coach.text}
      step={coach.step}
      kind={coach.kind}
      textId={coach.textId}
      onpause={coach.onpause}
    />
  {/if}

  {#if sorted.length === 0}
    <p class="text-sm text-fg-muted">{t('game.wonders.none')}</p>
  {/if}

  <ul class="flex flex-col gap-3">
    {#each sorted as option (option.key)}
      <li
        class="flex flex-col gap-2 rounded-panel border border-rule p-2"
        data-wonder-option={option.wonder}
      >
        <WonderTile
          slug={option.wonder}
          size="city"
          afford={option.cost ? affordOf(option.cost) : undefined}
          hint={hintOf(option.wonder)}
        />
        {#if option.cost}
          <details class="text-sm">
            <summary class="min-h-11 cursor-pointer py-3 text-fg-muted"
              >{t('game.cost.title')}</summary
            >
            <CostBreakdown cost={option.cost} {coins} class="pt-1" />
          </details>
        {/if}
        {#if option.choice}
          <div
            class="flex flex-col gap-1"
            data-act-group
            data-tutorial-locked={allow(option.key) ? undefined : ''}
          >
            <ActionButton
              key={option.key}
              label={t('game.action.buildWonder', {
                wonder: wonderName(option.wonder),
                count: option.cost?.total ?? 0
              })}
              {confirm}
              {connected}
              disabled={!canAct || !allow(option.key)}
              {onact}
            />
            <OutcomeLine terms={termsOf(option)} actKey={option.key} blunder={blunderOf(option)} />
          </div>
        {:else if option.reason}
          <p class="text-sm text-fg-muted">{reasonText(option.reason, option.cost)}</p>
        {/if}
      </li>
    {/each}
  </ul>
  {#if canAct && options.some((o) => o.choice && !allow(o.key))}
    <p class="text-xs text-fg-muted" data-tutorial-reason>{lockedReason}</p>
  {/if}

  <p class="text-xs text-fg-muted">{t('game.wonders.tuckNote')}</p>
  <p class="text-xs text-fg-muted">{t('game.wonders.sevenNote', { count: wondersBuilt })}</p>
</div>
