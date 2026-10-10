<script lang="ts">
  // The name prompt (UX design section 2, "NamePrompt"): a first visit
  // through an invite link asks for a display name before joining, since
  // the other player sees it. One field (dir="auto", so a Persian name
  // reads right to left) with a suggested name as its placeholder, and a
  // "Play as …" button that takes the typed name or the suggestion. A leaf:
  // the caller stores the name and joins the room.
  import { t } from '$lib/i18n';
  import { cleanName } from '$lib/session/identity';
  import Button from '../shell/Button.svelte';

  interface Props {
    /** The suggested name (the placeholder, and the name when the field is left empty). */
    suggestion: string;
    /** The room the player is about to join, shown for orientation. */
    code?: string;
    onsubmit: (name: string) => void;
  }

  let { suggestion, code, onsubmit }: Props = $props();

  /** The UX design's limit; the server takes up to 64 characters. */
  const MAX_LENGTH = 20;
  /** A div and label[for], not a wrapping label: the app's ring for an
   *  input inside a label is meant for hidden radio inputs. */
  const uid = $props.id();
  const inputId = `${uid}-input`;

  let value = $state('');
  const name = $derived(cleanName(value) || suggestion);
</script>

<svelte:head>
  <title>{t('online.prompt.title')} · {t('app.name')}</title>
</svelte:head>

<main class="flex h-full items-center justify-center overflow-y-auto px-4 py-6" data-name-prompt>
  <form
    class="flex w-full max-w-sm flex-col gap-4 rounded-panel border border-rule bg-panel p-5 shadow-panel"
    onsubmit={(event) => {
      event.preventDefault();
      onsubmit(name);
    }}
  >
    <div class="flex flex-col gap-1">
      <h1 class="font-display text-2xl text-fg">{t('online.prompt.title')}</h1>
      <p class="text-sm text-fg-muted">{t('online.prompt.intro')}</p>
    </div>

    <div class="flex flex-col gap-1.5">
      <label for={inputId} class="text-sm font-semibold text-fg">{t('online.name.label')}</label>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="min-h-12 rounded-control border border-rule-strong bg-panel-sunken px-3 text-base text-fg placeholder:text-fg-muted"
        type="text"
        id={inputId}
        name="name"
        autocomplete="nickname"
        maxlength={MAX_LENGTH}
        dir="auto"
        placeholder={suggestion}
        autofocus
        bind:value
        data-name-input
      />
    </div>

    <Button type="submit" variant="primary" full data-control="name-submit">
      <span class="truncate" dir="auto">{t('online.prompt.submit', { name })}</span>
    </Button>

    {#if code}
      <p class="text-center text-xs text-fg-muted" dir="ltr">
        {t('game.title', { code })}
      </p>
    {/if}
  </form>
</main>
