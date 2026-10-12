<script lang="ts">
  // A game's hint settings (docs/dev/decisions.md D23): one switch per
  // family, each with its one-line description, and a note that both
  // players play with the same hints. Read-only for everyone but the
  // game's admin, who sees the same rows as switches. A leaf: the caller
  // passes the settings and gets the next settings back on a change (a
  // new-game sheet keeps them as its draft; the game sends a hints frame).
  import { t } from '$lib/i18n';
  import { hintRows, withFamily } from '$lib/hints/families';
  import type { HintFamily, HintSettings } from '$lib/wire';

  interface Props {
    hints: HintSettings;
    /** The settings cannot be changed here: the rows show On or Off. */
    readonly?: boolean;
    /** Say who may change them (a player who is not the admin). */
    readonlyNote?: boolean;
    /** A change is on its way to the server: the switches wait. */
    busy?: boolean;
    onchange?: (next: HintSettings) => void;
  }

  let { hints, readonly = false, readonlyNote = false, busy = false, onchange }: Props = $props();

  const rows = $derived(hintRows(hints));

  // The switch shows what the caller passes: it is put back at once, and
  // turns when the caller's settings do (a refused change leaves it as it
  // was).
  function toggle(input: HTMLInputElement, family: HintFamily, on: boolean): void {
    const want = input.checked;
    input.checked = on;
    if (busy) return;
    onchange?.(withFamily(hints, family, want));
  }

  // While a change is on its way the switches wait, but they are only
  // aria-disabled, never disabled: a disabled switch drops the keyboard
  // focus to the page, and the Escape that follows a quick toggle would no
  // longer close the sheet. A click on a waiting switch does nothing.
  function hold(event: MouseEvent): void {
    if (busy) event.preventDefault();
  }
</script>

<div class="flex flex-col gap-2" data-hint-editor={readonly ? 'readonly' : 'edit'}>
  <p class="text-sm text-fg-muted">
    {t('hints.sameForBoth')}
    {#if readonly && readonlyNote}
      {t('hints.readOnly')}
    {/if}
  </p>
  <ul class="flex flex-col divide-y divide-rule rounded-control border border-rule bg-panel">
    {#each rows as row (row.family)}
      <li data-hint-family={row.family} data-on={row.on ? '' : undefined}>
        {#if readonly}
          <div class="flex min-h-12 items-center justify-between gap-3 px-3 py-2">
            <span class="flex min-w-0 flex-col">
              <span class="text-sm font-semibold text-fg">{t(row.name)}</span>
              <span class="text-xs text-fg-muted">{t(row.hint)}</span>
            </span>
            <span
              class="shrink-0 rounded-full px-2.5 py-0.5 text-xs font-semibold
                {row.on ? 'bg-accent-soft text-accent-soft-fg' : 'bg-panel-sunken text-fg-muted'}"
            >
              {row.on ? t('hints.on') : t('hints.off')}
            </span>
          </div>
        {:else}
          <label
            class="flex min-h-12 items-center justify-between gap-3 px-3 py-2
              {busy ? 'cursor-wait' : 'cursor-pointer'}"
          >
            <span class="flex min-w-0 flex-col">
              <span class="text-sm font-semibold text-fg">{t(row.name)}</span>
              <span class="text-xs text-fg-muted">{t(row.hint)}</span>
            </span>
            <input
              type="checkbox"
              role="switch"
              class="size-5 shrink-0 accent-accent aria-disabled:opacity-50"
              checked={row.on}
              aria-disabled={busy ? 'true' : undefined}
              onclick={hold}
              onchange={(e) => toggle(e.currentTarget, row.family, row.on)}
            />
          </label>
        {/if}
      </li>
    {/each}
  </ul>
  {#if busy}
    <p class="text-xs text-fg-muted" role="status">{t('hints.saving')}</p>
  {/if}
</div>
