<script lang="ts">
  // The "vs AI" new-game sheet (UX design §2). Start sends a real `create`
  // frame through the Session the caller supplies via onStart, so this
  // component needs no live connection to render (gallery-safe).
  import BotAvatar from '$lib/art/BotAvatar.svelte';
  import { t, type MessageKey } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import Segmented from '$lib/components/shell/Segmented.svelte';
  import Sheet from '$lib/components/shell/Sheet.svelte';
  import { SessionError } from '$lib/session/session.svelte';
  import type { BotLevel, SeatChoice } from '$lib/wire';

  interface Props {
    open?: boolean;
    onStart: (opts: { bot: BotLevel; seat: SeatChoice }) => Promise<void> | void;
  }

  let { open = $bindable(false), onStart }: Props = $props();

  const titleId = 'newgame-sheet-title';

  // as const + satisfies (not a type annotation) so each name/hint keeps its
  // own literal key type; t() needs that to know no params are required
  // (see $lib/i18n's MessageParams: it cannot tell for the widened MessageKey).
  const LEVELS = [
    { level: 'easy', name: 'home.ai.apprentice', hint: 'home.ai.apprenticeHint' },
    { level: 'normal', name: 'home.ai.scholar', hint: 'home.ai.scholarHint' },
    { level: 'hard', name: 'home.ai.master', hint: 'home.ai.masterHint' }
  ] as const satisfies readonly { level: BotLevel; name: MessageKey; hint: MessageKey }[];

  let level = $state<BotLevel>('normal');
  let seatValue = $state<'random' | '0' | '1'>('random');
  let starting = $state(false);
  let failure = $state<string | null>(null);

  const seatOptions = $derived([
    { value: 'random', label: t('home.ai.seatRandom') },
    { value: '0', label: t('home.ai.seatSun') },
    { value: '1', label: t('home.ai.seatMoon') }
  ]);

  async function start(): Promise<void> {
    if (starting) return;
    const seat: SeatChoice = seatValue === 'random' ? 'random' : seatValue === '0' ? 0 : 1;
    starting = true;
    failure = null;
    try {
      await onStart({ bot: level, seat });
    } catch (err) {
      failure = err instanceof SessionError ? t(err.messageKey) : t('session.error.unknown');
    } finally {
      starting = false;
    }
  }
</script>

<Sheet bind:open {titleId}>
  <form
    class="flex flex-col gap-5 px-5 pt-1 pb-5 split:pt-5"
    onsubmit={(event) => {
      event.preventDefault();
      void start();
    }}
  >
    <h2 id={titleId} class="font-display text-xl text-fg">{t('home.ai.title')}</h2>

    <fieldset class="flex flex-col gap-2">
      <legend class="sr-only">{t('home.ai.levelLegend')}</legend>
      {#each LEVELS as option (option.level)}
        <label
          class="flex min-h-14 cursor-pointer items-center gap-3 rounded-control border px-4 py-2 transition-colors duration-(--dur-fast)
            {level === option.level
            ? 'border-accent bg-accent-soft'
            : 'border-rule bg-panel hover:bg-panel-sunken'}"
        >
          <input
            type="radio"
            name="ai-level"
            value={option.level}
            checked={level === option.level}
            onchange={() => (level = option.level)}
            class="size-4 shrink-0 accent-accent"
          />
          <BotAvatar level={option.level} class="size-10 short:size-9" />
          <span class="flex min-w-0 flex-col">
            <span class="text-base font-semibold text-fg">{t(option.name)}</span>
            <span class="text-sm text-fg-muted">{t(option.hint)}</span>
          </span>
        </label>
      {/each}
    </fieldset>

    <div class="flex flex-col gap-1.5">
      <span class="text-sm font-medium text-fg-muted" aria-hidden="true">
        {t('home.ai.seatLegend')}
      </span>
      <Segmented
        name="ai-seat"
        legend={t('home.ai.seatLegend')}
        options={seatOptions}
        bind:value={seatValue}
      />
      <p class="text-xs text-fg-muted">{t('home.ai.firstRandomNote')}</p>
    </div>

    {#if failure}
      <p class="text-sm text-cinnabar" role="alert">{failure}</p>
    {/if}

    <Button type="submit" variant="primary" full disabled={starting}>
      {t('home.ai.start')}
    </Button>
  </form>
</Sheet>
