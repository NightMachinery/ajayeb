<script lang="ts">
  // Home's "Play online" sheet: your name (prefilled from the stored name,
  // editable, saved by the caller), which crest you take (Sun, Moon or
  // random), and Create. Create sends a real `create` frame with mode
  // online through the caller's onStart, so this sheet needs no live
  // connection to render (the gallery). A link at the bottom switches to
  // joining by code.
  import { t } from '$lib/i18n';
  import { cleanName } from '$lib/session/identity';
  import { SessionError } from '$lib/session/session.svelte';
  import type { SeatChoice } from '$lib/wire';
  import Button from '../shell/Button.svelte';
  import Segmented from '../shell/Segmented.svelte';
  import Sheet from '../shell/Sheet.svelte';
  import HintsSection from '$lib/components/hints/HintsSection.svelte';
  import { hintDefaults } from '$lib/hints/local.svelte';

  interface Props {
    open?: boolean;
    /** The stored display name, if any. */
    name: string;
    /** The suggested name when none is stored (the placeholder). */
    suggestion: string;
    onStart: (opts: { name: string; seat: SeatChoice }) => Promise<void> | void;
    /** Switches to the join sheet. */
    onJoin?: () => void;
  }

  let { open = $bindable(false), name, suggestion, onStart, onJoin }: Props = $props();

  const titleId = 'online-sheet-title';
  const MAX_LENGTH = 20;
  /** A div and label[for], not a wrapping label: the app's ring for an
   *  input inside a label is meant for hidden radio inputs. */
  const uid = $props.id();
  const inputId = `${uid}-input`;

  // The field starts from the stored name, then is the player's to edit.
  let typed = $state<string | null>(null);
  const value = $derived(typed ?? name);
  let seatValue = $state<'random' | '0' | '1'>('random');
  // The game's hint settings, from this device's last saved ones.
  let hints = $state(hintDefaults.for('online'));
  let starting = $state(false);
  let failure = $state<string | null>(null);

  const seatOptions = $derived([
    { value: 'random', label: t('online.crest.random') },
    { value: '0', label: t('online.crest.sun') },
    { value: '1', label: t('online.crest.moon') }
  ]);

  async function start(): Promise<void> {
    if (starting) return;
    const seat: SeatChoice = seatValue === 'random' ? 'random' : seatValue === '0' ? 0 : 1;
    // The game starts from these hints (the Session's create reads the
    // local default), and the next new game on this device will too.
    hintDefaults.save('online', hints);
    starting = true;
    failure = null;
    try {
      await onStart({ name: cleanName(value) || suggestion, seat });
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
    data-online-sheet
    onsubmit={(event) => {
      event.preventDefault();
      void start();
    }}
  >
    <div class="flex flex-col gap-1">
      <h2 id={titleId} class="font-display text-xl text-fg">{t('online.create.title')}</h2>
      <p class="text-sm text-fg-muted">{t('online.create.intro')}</p>
    </div>

    <div class="flex flex-col gap-1.5">
      <label for={inputId} class="text-sm font-semibold text-fg">{t('online.name.label')}</label>
      <input
        class="min-h-12 rounded-control border border-rule-strong bg-panel-sunken px-3 text-base text-fg placeholder:text-fg-muted"
        type="text"
        id={inputId}
        name="name"
        autocomplete="nickname"
        maxlength={MAX_LENGTH}
        dir="auto"
        placeholder={suggestion}
        {value}
        oninput={(event) => (typed = event.currentTarget.value)}
        data-name-input
      />
      <span class="text-xs text-fg-muted">{t('online.name.hint')}</span>
    </div>

    <div class="flex flex-col gap-1.5">
      <span class="text-sm font-semibold text-fg" aria-hidden="true">
        {t('online.crest.legend')}
      </span>
      <Segmented
        name="online-seat"
        legend={t('online.crest.legend')}
        options={seatOptions}
        bind:value={seatValue}
      />
      <p class="text-xs text-fg-muted">{t('online.crest.note')}</p>
    </div>

    <HintsSection bind:value={hints} />

    {#if failure}
      <p class="text-sm text-cinnabar" role="alert">{failure}</p>
    {/if}

    <Button type="submit" variant="primary" full disabled={starting} data-control="online-create">
      {t('online.create.submit')}
    </Button>

    {#if onJoin}
      <button
        type="button"
        class="-my-2 min-h-11 self-center rounded-control px-3 text-sm font-semibold text-fg underline decoration-gild underline-offset-2"
        data-control="online-join"
        onclick={onJoin}
      >
        {t('online.create.haveCode')}
      </button>
    {/if}
  </form>
</Sheet>
