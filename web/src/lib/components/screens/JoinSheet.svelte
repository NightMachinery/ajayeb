<script lang="ts">
  // Home's "Join with a code" sheet: one field for the six-character room
  // code. What is typed or pasted is normalised as it comes (upper case, no
  // spaces or dashes, the code taken out of a pasted link), and Join stays
  // off until it is a code the server could have made. Join hands the code
  // to the caller, which opens /play/<code>; the room page asks for a name
  // there if none is stored, and shows the server's answer (room not found
  // and so on).
  import { t } from '$lib/i18n';
  import { CODE_LENGTH, isRoomCode, normalizeCodeInput } from '$lib/ui/online';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';

  interface Props {
    open?: boolean;
    /** The code to start from (the gallery). */
    initial?: string;
    onJoin: (code: string) => void;
  }

  let { open = $bindable(false), initial = '', onJoin }: Props = $props();

  const titleId = 'join-sheet-title';
  /** A div and label[for], not a wrapping label: the app's ring for an
   *  input inside a label is meant for hidden radio inputs. */
  const uid = $props.id();
  const inputId = `${uid}-input`;
  const hintId = `${uid}-hint`;

  let typed = $state<string | null>(null);
  const code = $derived(normalizeCodeInput(typed ?? initial));
  const valid = $derived(isRoomCode(code));
  // The hint turns into the error only once a full code is typed.
  const wrong = $derived(code.length === CODE_LENGTH && !valid);
</script>

<Sheet bind:open {titleId}>
  <form
    class="flex flex-col gap-5 px-5 pt-1 pb-5 split:pt-5"
    data-join-sheet
    onsubmit={(event) => {
      event.preventDefault();
      if (valid) onJoin(code);
    }}
  >
    <h2 id={titleId} class="font-display text-xl text-fg">{t('online.join.title')}</h2>

    <div class="flex flex-col gap-1.5">
      <label for={inputId} class="text-sm font-semibold text-fg">{t('online.join.label')}</label>
      <!-- Typing the code is the sheet's whole job, so it opens on the
           field, not on its title (Sheet's focusTitle). -->
      <!-- svelte-ignore a11y_autofocus -->
      <input
        autofocus
        class="min-h-14 rounded-control border bg-panel-sunken px-3 text-center font-display text-3xl tracking-[0.3em] text-fg uppercase
          {wrong ? 'border-cinnabar' : 'border-rule-strong'}"
        type="text"
        id={inputId}
        name="code"
        inputmode="text"
        autocomplete="off"
        autocapitalize="characters"
        spellcheck="false"
        dir="ltr"
        aria-invalid={wrong}
        aria-describedby={hintId}
        value={code}
        oninput={(event) => {
          typed = event.currentTarget.value;
          // Show the normalised code at once, also when it equals the last one.
          event.currentTarget.value = normalizeCodeInput(typed);
        }}
        data-code-input
      />
      <span
        id={hintId}
        class="text-xs {wrong ? 'text-cinnabar-ink' : 'text-fg-muted'}"
        role={wrong ? 'alert' : undefined}
      >
        {wrong ? t('online.join.invalid') : t('online.join.hint')}
      </span>
    </div>

    <Button
      type="submit"
      variant="primary"
      full
      disabled={!valid}
      data-control="join-submit"
      data-sheet-actions
    >
      {t('online.join.submit')}
    </Button>
  </form>
</Sheet>
