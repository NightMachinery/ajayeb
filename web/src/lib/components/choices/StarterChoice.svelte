<script lang="ts">
  // Who starts the next age: why this player chooses, and one button per
  // seat, each carrying the server's own starter key. When this client is
  // not the chooser it shows a waiting note instead. Display only: the
  // reason line reads the pawn's position from the view, the options are
  // the view's choices.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';
  import { militaryEndangered } from '$lib/view/threats';
  import type { Choice, MilitaryView } from '$lib/wire';
  import { actButtonState } from '../game/act-button';
  import ConfirmButton from '../shell/ConfirmButton.svelte';
  import { asSeat, type Seat, type SeatNames } from './room';

  interface Props {
    /** The age about to start. */
    age: number;
    /** The seat that chooses (pending.seat). */
    chooser: Seat;
    names: SeatNames;
    /** The seat that reads as "you", if any. */
    you?: Seat;
    military: MilitaryView;
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
    chooser,
    names,
    you,
    military,
    choices,
    canAct,
    confirm = false,
    connected = true,
    onact
  }: Props = $props();

  const chooserIsYou = $derived(you === chooser);

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

<div class="flex flex-col gap-3" data-starter-choice>
  <h3 class="text-center font-display text-lg text-fg">{t('choices.starter.title', { age })}</h3>
  <p class="text-center text-sm text-fg" dir="auto">{why}</p>

  {#if options.length > 0}
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
            <span class="min-w-0 truncate" dir="auto">{rule.label}</span>
          </button>
        {/if}
      {/each}
    </div>
    <p class="text-center text-xs text-fg-muted">{t('choices.starter.hint')}</p>
  {:else}
    <p
      class="rounded-control bg-panel-sunken px-3 py-2 text-center text-sm text-fg-muted"
      role="status"
      dir="auto"
    >
      {t('choices.waiting.chooseStarter', { name: names[chooser], age })}
    </p>
  {/if}
</div>
