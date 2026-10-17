<script lang="ts">
  // The inline form of a coach step (the tutorial design, section 2.4):
  // inside a modal sheet (the card sheet, its wonder step, the draft's
  // wonder sheet), just above the action group that holds the target. A
  // modal dialog makes the rest of the page inert, so a bubble outside it
  // could be neither tapped nor read by a screen reader; inside the dialog
  // the note is in the keyboard order and announced with the sheet. It has
  // the step's text and "Play on my own". A leaf with plain props.
  import GraduationCap from '@lucide/svelte/icons/graduation-cap';
  import { t } from '$lib/i18n';
  import type { CoachKind } from '$lib/tutorial/steps';

  interface Props {
    title: string;
    progress?: string | null;
    text: string;
    /** The lesson id (data-coach-step). */
    step: string;
    kind: CoachKind;
    onpause?: () => void;
    /** The text's id, for the target's aria-describedby. */
    textId?: string;
    class?: string;
  }

  let {
    title,
    progress = null,
    text,
    step,
    kind,
    onpause,
    textId,
    class: className = ''
  }: Props = $props();
</script>

<div
  role="note"
  class="flex flex-col gap-1.5 rounded-control border border-accent bg-accent-soft px-3 py-2 text-accent-soft-fg {className}"
  data-coach-note
  data-coach-step={step}
  data-coach-kind={kind}
>
  <p class="flex items-center gap-1.5 text-xs font-semibold tracking-wide uppercase">
    <GraduationCap class="size-4 shrink-0" aria-hidden="true" />
    <span>{title}</span>
    {#if progress}
      <span class="font-normal tracking-normal normal-case opacity-80">· {progress}</span>
    {/if}
  </p>
  {#if text}
    <p id={textId} class="text-sm leading-snug" data-coach-text>{text}</p>
  {/if}
  <div class="flex justify-end">
    {#if onpause}
      <button
        type="button"
        class="-me-2 min-h-11 rounded-control px-3 text-sm font-semibold underline underline-offset-2 hover:bg-panel-sunken"
        data-coach-pause
        onclick={onpause}
      >
        {t('tutorial.coach.pause')}
      </button>
    {/if}
  </div>
</div>
