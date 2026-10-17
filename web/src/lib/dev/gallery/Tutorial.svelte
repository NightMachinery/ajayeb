<script lang="ts">
  // The tutorial's components, for a visual check in both themes: Home's
  // offer panel, a coach note as a sheet shows it, the coach layer on a
  // demo target (its bubble is a CoachMark) and the completion card. Plain
  // props and no live connection, like the rest of this page.
  import { t } from '$lib/i18n';
  import Button from '$lib/components/shell/Button.svelte';
  import CoachLayer from '$lib/components/tutorial/CoachLayer.svelte';
  import CoachMark from '$lib/components/tutorial/CoachMark.svelte';
  import CoachNote from '$lib/components/tutorial/CoachNote.svelte';
  import TutorialDone from '$lib/components/tutorial/TutorialDone.svelte';
  import TutorialOffer from '$lib/components/tutorial/TutorialOffer.svelte';
  import type { CoachStep } from '$lib/tutorial/steps';
  import { coachText } from '$lib/tutorial/text';

  const noop = () => {};

  const infoStep: CoachStep = {
    key: 'gallery:ageOne',
    id: 'ageOne',
    lesson: 'info',
    kind: 'info',
    surface: 'board',
    words: 'lesson',
    target: ['[data-gallery-coach-target]'],
    region: true,
    turn: 3,
    turns: 10
  };
  const words = coachText(infoStep, t('game.persona.easy'));

  let layerShown = $state(false);
  let markShown = $state(false);
  let doneShown = $state(false);
  let markAnchor = $state<HTMLElement | null>(null);
</script>

<section class="flex flex-col gap-4" data-testid="gallery-tutorial-section">
  <h3 class="text-sm font-semibold text-fg-muted">Tutorial <!-- i18n-ok --></h3>

  <div class="max-w-md">
    <TutorialOffer botName={t('game.persona.easy')} onstart={noop} ondecline={noop} />
  </div>

  <div class="max-w-md rounded-panel border border-rule bg-panel p-4">
    <CoachNote
      title={words.title}
      progress={words.progress}
      text={t('tutorial.coach.otherCard', { card: 'Lumber Yard' })}
      step="buildFree"
      kind="note"
      onpause={noop}
    />
  </div>

  <div class="flex flex-wrap gap-2">
    <Button variant="secondary" onclick={() => (layerShown = !layerShown)}>
      CoachLayer <!-- i18n-ok -->
    </Button>
    <Button variant="secondary" onclick={() => (markShown = !markShown)}>
      CoachMark <!-- i18n-ok -->
    </Button>
    <Button variant="secondary" onclick={() => (doneShown = true)}>
      TutorialDone <!-- i18n-ok -->
    </Button>
  </div>
  <div
    class="grid h-40 max-w-md place-items-center rounded-panel border border-dashed border-rule-strong text-sm text-fg-muted"
    data-gallery-coach-target
    bind:this={markAnchor}
  >
    {t('tutorial.coach.region')}
  </div>
</section>

<CoachLayer
  step={layerShown ? infoStep : null}
  text={words}
  active={layerShown}
  onnext={() => (layerShown = false)}
  onpause={() => (layerShown = false)}
/>

<CoachMark
  open={markShown}
  anchor={markAnchor}
  title={words.title}
  progress={words.progress}
  text={words.text}
  kind="tap"
  step="buildFree"
  onpause={() => (markShown = false)}
/>

<TutorialDone
  open={doneShown}
  onkeep={() => (doneShown = false)}
  onnew={() => (doneShown = false)}
  onhome={() => (doneShown = false)}
/>
