<script lang="ts">
  // Who starts the next age: why this player chooses, and one button per
  // seat, each carrying the server's own starter key. When this client is
  // not the chooser it shows a waiting note instead. Display only: the
  // reason line reads the pawn's position from the view, the options are
  // the view's choices.
  //
  // Hints (M10.HS3), from the page's game context: under the buttons, the
  // wins the starting seat would have at once (starter-threat, the server's
  // Choice.outcome.threatens), and the cards the age opens with
  // (starter-preview, the view's structure).
  import { cardName, joinList } from '$lib/components/card/effect-text';
  import { hintHide } from '$lib/hints/local.svelte';
  import { t } from '$lib/i18n';
  import { sessionHintScreen } from '$lib/view/hints';
  import { openingCards, starterThreatLines } from '$lib/view/outcome';
  import StarterDiagram from '../rules/StarterDiagram.svelte';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import Icon from '$lib/icons/Icon.svelte';
  import { militaryEndangered } from '$lib/view/threats';
  import type { Choice, MilitaryView } from '$lib/wire';
  import { actButtonState } from '../game/act-button';
  import ConfirmButton from '../shell/ConfirmButton.svelte';
  import { asSeat, tryGameContext, type Seat, type SeatNames } from './room';

  interface Props {
    /** The age about to start. */
    age: number;
    /** Split details and controls between the sheet body and its footer. */
    part?: 'all' | 'details' | 'actions';
    /** The seat that chooses (pending.seat). */
    chooser: Seat;
    names: SeatNames;
    /** The seat that reads as "you", if any. */
    you?: Seat;
    military: MilitaryView;
    /** A centred actual choice proves this room uses the diagram’s default ruling. */
    showDiagram?: boolean;
    /** The view's starter choices; empty when this client is not the chooser. */
    choices: readonly Choice[];
    canAct: boolean;
    /** Ask for a second tap before sending (the act-button rule). */
    confirm?: boolean;
    /** The connection is up. */
    connected?: boolean;
    onact: (key: string) => void;
  }

  let {
    age,
    part = 'all',
    chooser,
    names,
    you,
    military,
    showDiagram = false,
    choices,
    canAct,
    confirm = false,
    connected = true,
    onact
  }: Props = $props();

  const chooserIsYou = $derived(you === chooser);

  const ctx = tryGameContext();
  const screen = $derived(sessionHintScreen(ctx?.session, hintHide.value));
  const threatLines = $derived(
    screen ? starterThreatLines(choices, screen.families, screen.names) : []
  );
  const opening = $derived.by(() => {
    const structure = ctx?.session.view?.structure;
    if (!screen?.families.has('tempo') || !structure) return undefined;
    const cards = openingCards(structure);
    return cards.length > 0
      ? t('hints.starter.opening', { list: joinList(cards.map(cardName), 'conjunction') })
      : undefined;
  });

  const why = $derived.by(() => {
    const name = names[chooser];
    if (military.position === 0) {
      return chooserIsYou
        ? t('choices.starter.whyEvenYou')
        : t('choices.starter.whyEven', { name });
    }
    if (militaryEndangered(military) === chooser) {
      return chooserIsYou
        ? t('choices.starter.whyBehindYou')
        : t('choices.starter.whyBehind', { name });
    }
    return chooserIsYou
      ? t('choices.starter.whyOtherYou')
      : t('choices.starter.whyOther', { name });
  });

  function optionLabel(choice: Choice): string {
    const seat = asSeat(choice.starter ?? 0);
    if (seat === you) return t('choices.starter.optionYou');
    return t('choices.starter.option', { name: names[seat] });
  }

  // Seat order on screen, whatever order the choices come in.
  const options = $derived(
    [...choices]
      .filter((c) => c.kind === 'starter')
      .sort((a, b) => (a.starter ?? 0) - (b.starter ?? 0))
  );
</script>

{#snippet starterDiagram()}<StarterDiagram compact />{/snippet}

<div class="flex flex-col gap-3" data-starter-choice={part !== 'actions' ? '' : undefined}>
  {#if part !== 'actions'}
    <h3 class="text-center font-display text-lg text-fg">{t('choices.starter.title', { age })}</h3>
    <p
      class="text-center text-sm text-fg"
      {@attach tooltip({
        text: why,
        rulesId: 'age-end',
        diagram: showDiagram ? starterDiagram : undefined
      })}
    >
      {why}
    </p>
  {/if}
  {#if options.length > 0}
    {#if part !== 'details'}
      <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
        {#each options as choice (choice.key)}
          {@const seat = asSeat(choice.starter ?? 0)}
          {@const rule = actButtonState({
            label: optionLabel(choice),
            canAct,
            connected,
            confirm
          })}
          {#if rule.confirm}
            <!-- The confirm step (online rooms): the crest colour stays on the
               border; the button's label names the player. -->
            <ConfirmButton
              label={rule.label}
              variant="secondary"
              class="h-auto! min-h-14 min-w-0 border-2! px-3 py-2 leading-tight
              {seat === 0 ? 'border-sun!' : 'border-moon!'}"
              data-act={choice.key}
              disabled={rule.disabled}
              onConfirm={() => onact(choice.key)}
            />
          {:else}
            <button
              type="button"
              class="flex min-h-14 min-w-0 items-center justify-center gap-2 rounded-control border-2 bg-panel px-3 py-2
              font-semibold text-fg transition-colors duration-(--dur-fast) hover:bg-panel-sunken
              disabled:cursor-not-allowed disabled:opacity-50
              {seat === 0 ? 'border-sun' : 'border-moon'}"
              data-act={choice.key}
              disabled={rule.disabled}
              onclick={() => onact(choice.key)}
            >
              <Icon
                name={seat === 0 ? 'crest-sun' : 'crest-moon'}
                class="shrink-0 text-xl {seat === 0 ? 'text-sun-ink' : 'text-moon-ink'}"
              />
              <span class="min-w-0 truncate">{rule.label}</span>
            </button>
          {/if}
        {/each}
      </div>
    {/if}
    {#if part !== 'actions'}
      <p class="text-center text-xs text-fg-muted">{t('choices.starter.hint')}</p>
      {#if threatLines.length > 0}
        <ul class="flex flex-col gap-1" data-starter-threats>
          {#each threatLines as line (line.seat)}
            <li
              class="threat-line border-s-[3px] ps-2 text-sm font-semibold text-fg"
              data-starter-threat={line.seat}
            >
              {line.text}
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
  {:else if part !== 'actions'}
    <p
      class="rounded-control bg-panel-sunken px-3 py-2 text-center text-sm text-fg-muted"
      role="status"
    >
      {t('choices.waiting.chooseStarter', { name: names[chooser], age })}
    </p>
  {/if}
  {#if opening && part !== 'actions'}
    <p class="text-center text-xs text-fg-muted" data-starter-preview>{opening}</p>
  {/if}
</div>

<style>
  .threat-line {
    border-color: var(--gold-deep);
  }
</style>
