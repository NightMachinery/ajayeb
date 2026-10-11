<script lang="ts">
  // "On my screen": what this player hides on their own screen (the local
  // hide, lib/hints/local.svelte.ts): every hint mark on the board, and
  // whole families. Display only, for this device: it never shows a hint
  // the game's settings turned off, and the other player is not told. A
  // leaf: the caller passes the current hide and gets each change back.
  import EyeOff from '@lucide/svelte/icons/eye-off';
  import { t } from '$lib/i18n';
  import { HINT_FAMILIES, familyText, type LocalHide } from '$lib/hints/families';
  import type { HintFamily } from '$lib/wire';

  interface Props {
    hide: LocalHide;
    onfamily: (family: HintFamily, hidden: boolean) => void;
    onboardmarks: (hidden: boolean) => void;
  }

  let { hide, onfamily, onboardmarks }: Props = $props();

  const uid = $props.id();
  const titleId = `${uid}-title`;
</script>

<section class="flex flex-col gap-2" aria-labelledby={titleId} data-hint-local-hide>
  <h3 id={titleId} class="text-xs font-semibold tracking-wide text-fg-muted uppercase">
    {t('hints.local.title')}
  </h3>
  <p class="text-sm text-fg-muted">{t('hints.local.intro')}</p>

  <label class="flex min-h-11 cursor-pointer items-center justify-between gap-3">
    <span class="flex flex-col">
      <span class="text-sm font-medium text-fg">{t('hints.local.boardMarks')}</span>
      <span class="text-xs text-fg-muted">{t('hints.local.boardMarksHint')}</span>
    </span>
    <input
      type="checkbox"
      role="switch"
      class="size-5 shrink-0 accent-accent"
      data-hide-board-marks
      checked={hide.boardMarks}
      onchange={(e) => onboardmarks(e.currentTarget.checked)}
    />
  </label>

  <ul class="flex flex-wrap gap-2">
    {#each HINT_FAMILIES as family (family)}
      {@const hidden = hide.families.includes(family)}
      {@const name = t(familyText(family).name)}
      <li>
        <button
          type="button"
          class="flex min-h-9 items-center gap-1.5 rounded-full border px-3 text-sm transition-colors duration-(--dur-fast)
            {hidden
            ? 'border-rule-strong bg-panel-sunken text-fg-muted line-through'
            : 'border-rule bg-panel text-fg hover:bg-panel-sunken'}"
          aria-pressed={hidden}
          aria-label={t('hints.local.hideFamily', { family: name })}
          data-hide-family={family}
          onclick={() => onfamily(family, !hidden)}
        >
          {#if hidden}
            <EyeOff class="size-4 shrink-0" aria-hidden="true" />
          {/if}
          {name}
        </button>
      </li>
    {/each}
  </ul>
</section>
