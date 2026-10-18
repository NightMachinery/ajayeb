<script lang="ts">
  // The advisor's strength (HintSettings.advisorStrength): how carefully the
  // move review checks a move, shown under the advisor's switch while it is
  // on. The admin picks one of three segments; everyone else reads the one
  // the game uses. Each segment's tooltip says what it checks and that a
  // deeper check is slower, and the line under the segments says it for the
  // one chosen, so a phone needs no long-press to learn it. A leaf: the
  // caller passes the strength and gets the new one back.
  import { t } from '$lib/i18n';
  import { ADVISOR_STRENGTHS, strengthText } from '$lib/hints/families';
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import type { AdvisorStrength } from '$lib/wire';

  interface Props {
    strength: AdvisorStrength;
    /** Show the strength, without the segments. */
    readonly?: boolean;
    /** A change is on its way to the server: the segments wait. */
    busy?: boolean;
    onchange?: (next: AdvisorStrength) => void;
  }

  let { strength, readonly = false, busy = false, onchange }: Props = $props();

  const uid = $props.id();
  const titleId = `${uid}-advisor-strength`;
  const chosen = $derived(strengthText(strength));

  // As with the switches, the segments show what the caller passes: the
  // radio is put back at once, and moves when the caller's strength does.
  function pick(input: HTMLInputElement, next: AdvisorStrength): void {
    const group = input.closest('[role="radiogroup"]');
    for (const radio of group?.querySelectorAll<HTMLInputElement>('input[type="radio"]') ?? []) {
      radio.checked = radio.value === strength;
    }
    if (busy || next === strength) return;
    onchange?.(next);
  }

  // Waiting segments are only aria-disabled, never disabled (see the
  // switches in HintSettingsEditor): a click on one does nothing.
  function hold(event: MouseEvent): void {
    if (busy) event.preventDefault();
  }
</script>

<div class="flex flex-col gap-1.5 px-3 pt-0.5 pb-2.5" data-advisor-strength={strength}>
  {#if readonly}
    <p class="flex flex-wrap items-baseline gap-x-1.5 text-xs">
      <span class="text-fg-muted">{t('hints.advisorStrength.title')}</span>
      <span
        class="font-semibold text-fg"
        data-strength={strength}
        {@attach tooltip({ text: t(chosen.tip) })}
      >
        {t(chosen.name)}
      </span>
    </p>
  {:else}
    <span id={titleId} class="text-xs font-semibold text-fg-muted">
      {t('hints.advisorStrength.title')}
    </span>
    <div
      role="radiogroup"
      aria-labelledby={titleId}
      class="grid grid-cols-3 gap-1 rounded-control border border-rule bg-panel-sunken p-1"
    >
      {#each ADVISOR_STRENGTHS as s (s)}
        <label
          class="flex min-h-11 items-center justify-center rounded-control px-1.5 text-center text-sm leading-tight font-semibold
            has-focus-visible:outline-2 has-focus-visible:outline-accent
            {s === strength ? 'bg-accent text-on-accent' : 'text-fg'}
            {busy ? 'cursor-wait' : 'cursor-pointer'}"
          data-strength={s}
          data-on={s === strength ? '' : undefined}
          {@attach tooltip({ text: t(strengthText(s).tip), actionable: true })}
        >
          <input
            type="radio"
            class="sr-only"
            name={titleId}
            value={s}
            checked={s === strength}
            aria-disabled={busy ? 'true' : undefined}
            onclick={hold}
            onchange={(e) => pick(e.currentTarget, s)}
          />
          {t(strengthText(s).name)}
        </label>
      {/each}
    </div>
    <p class="text-xs text-fg-muted" data-strength-hint>{t(chosen.tip)}</p>
  {/if}
</div>
