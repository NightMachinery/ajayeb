<script lang="ts">
  // Home's first-launch tutorial offer (the tutorial design, section 2.5):
  // an inline panel above the mode buttons, so it swallows nothing. No
  // thanks is the "never show again"; the tutorial stays one tap away in
  // Home's Learn to play row and in the vs-AI sheet. A leaf with plain props.
  import GraduationCap from '@lucide/svelte/icons/graduation-cap';
  import { t } from '$lib/i18n';
  import Button from '../shell/Button.svelte';

  interface Props {
    /** The bot's display name, already translated. */
    botName: string;
    onstart: () => void;
    ondecline: () => void;
  }

  let { botName, onstart, ondecline }: Props = $props();
  const uid = $props.id();
</script>

<section
  class="flex flex-col gap-3 rounded-control border border-accent bg-accent-soft p-4 text-accent-soft-fg shadow-panel short:gap-2 short:p-3"
  aria-labelledby="{uid}-title"
  data-tutorial-offer
>
  <div class="flex items-start gap-3">
    <GraduationCap class="mt-0.5 size-6 shrink-0 text-accent" aria-hidden="true" />
    <div class="flex min-w-0 flex-col gap-1">
      <h2 id="{uid}-title" class="text-base font-semibold text-fg">{t('tutorial.offer.title')}</h2>
      <p class="text-sm">{t('tutorial.offer.text', { name: botName })}</p>
    </div>
  </div>
  <div class="flex flex-wrap items-center justify-end gap-2">
    <Button variant="secondary" data-control="tutorial-decline" onclick={ondecline}>
      {t('tutorial.offer.decline')}
    </Button>
    <Button variant="primary" data-control="tutorial-start" onclick={onstart}>
      {t('tutorial.offer.start')}
    </Button>
  </div>
</section>
