<script lang="ts">
  // The sheet of a seat's device link (docs/protocol.md, "Device links"):
  // the link in a read-only field, Copy, and Reset, which stops every
  // earlier link to the seat. Opening the link on another device plays the
  // seat there, so the text says plainly that it works like a password. A
  // leaf: the caller fetches the link, copies it and resets it, and passes
  // what happened back in (status, copy, reset).
  import Copy from '@lucide/svelte/icons/copy';
  import RotateCw from '@lucide/svelte/icons/rotate-cw';
  import { t } from '$lib/i18n';
  import Button from '../shell/Button.svelte';
  import ConfirmButton from '../shell/ConfirmButton.svelte';
  import Sheet from '../shell/Sheet.svelte';

  interface Props {
    open?: boolean;
    seat: 0 | 1;
    /** The seat's display name. */
    name: string;
    /** The seat is the player's own (else the admin is looking at another's). */
    own: boolean;
    /** The link, once it is known. */
    link: string | null;
    status: 'loading' | 'ready' | 'failed';
    /** The last copy: copied, or failed (the field's text is then selected). */
    copy?: 'idle' | 'copied' | 'failed';
    /** The link was just reset. */
    reset?: boolean;
    oncopy: () => void;
    onreset: () => void;
    onretry: () => void;
  }

  let {
    open = $bindable(false),
    seat,
    name,
    own,
    link,
    status,
    copy = 'idle',
    reset = false,
    oncopy,
    onreset,
    onretry
  }: Props = $props();

  const uid = $props.id();
  const titleId = `${uid}-title`;
  const fieldId = `${uid}-field`;
  let field: HTMLInputElement | undefined = $state();

  // A copy that failed leaves the link selected, to copy by hand. DOM only.
  $effect(() => {
    if (copy === 'failed' && field && link) {
      field.focus();
      field.select();
    }
  });

  const message = $derived.by(() => {
    if (status === 'failed') return t('online.link.failed');
    if (copy === 'failed') return t('online.link.copyFailed');
    if (reset) return t('online.link.resetDone');
    if (copy === 'copied') return t('online.link.copied');
    return '';
  });

  function selectAll(event: Event): void {
    (event.currentTarget as HTMLInputElement).select();
  }
</script>

<Sheet bind:open {titleId}>
  <div class="flex flex-col gap-3 px-4 pt-1 pb-5 split:w-[26rem]" data-device-link-sheet={seat}>
    <h2 id={titleId} class="font-display text-xl text-fg" dir="auto">
      {own ? t('online.link.title') : t('online.link.titleOther', { name })}
    </h2>
    <p class="text-sm text-fg-muted" dir="auto">
      {own ? t('online.link.intro') : t('online.link.introOther', { name })}
    </p>

    <div class="flex flex-col gap-1">
      <label for={fieldId} class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
        {t('online.link.field')}
      </label>
      <input
        bind:this={field}
        id={fieldId}
        class="min-h-11 w-full rounded-control border border-rule bg-panel-sunken px-3 text-sm text-fg"
        type="text"
        readonly
        value={link ?? ''}
        placeholder={status === 'loading' ? t('online.link.loading') : ''}
        dir="ltr"
        data-device-link={link ?? undefined}
        onfocus={selectAll}
      />
    </div>

    <p
      class="min-h-5 text-sm {status === 'failed' || copy === 'failed'
        ? 'text-cinnabar-ink'
        : 'text-fg'}"
      role="status"
      data-device-link-status={status === 'failed'
        ? 'failed'
        : reset
          ? 'reset'
          : copy !== 'idle'
            ? copy
            : undefined}
    >
      {message}
    </p>

    {#if status === 'failed'}
      <Button variant="primary" full onclick={onretry} data-control="device-link-retry">
        <RotateCw class="size-4" aria-hidden="true" />
        {t('online.link.retry')}
      </Button>
    {:else}
      <Button
        variant="primary"
        full
        disabled={status !== 'ready'}
        onclick={oncopy}
        data-control="device-link-copy"
      >
        <Copy class="size-4" aria-hidden="true" />
        {t('online.link.copy')}
      </Button>
    {/if}

    <div class="flex flex-col gap-1 border-t border-rule pt-3">
      <ConfirmButton
        variant="secondary"
        class="w-full"
        label={t('online.link.reset')}
        disabled={status === 'loading'}
        onConfirm={onreset}
        data-control="device-link-reset"
      />
      <p class="text-xs text-fg-muted">{t('online.link.resetHint')}</p>
    </div>
  </div>
</Sheet>
