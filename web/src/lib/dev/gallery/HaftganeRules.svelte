<script lang="ts">
  // Haftgane's rules reference (M18.rules-reference), for a visual check in
  // both themes, as Rules.svelte does for Duel's: the four diagrams, the
  // same in their compact tooltip size, buttons that open the real in-game
  // overlay (no Haftgane screen calls it yet), and the panel body in a
  // phone-wide and a side-panel-wide frame. The headings are dev labels,
  // not shipped copy.
  import Button from '$lib/components/shell/Button.svelte';
  import ConflictDiagram from '$lib/games/haftgane/rules/ConflictDiagram.svelte';
  import PassingDiagram from '$lib/games/haftgane/rules/PassingDiagram.svelte';
  import RulesPanel from '$lib/games/haftgane/rules/RulesPanel.svelte';
  import ScienceDiagram from '$lib/games/haftgane/rules/ScienceDiagram.svelte';
  import TradeDiagram from '$lib/games/haftgane/rules/TradeDiagram.svelte';
  import { openHaftganeRules } from '$lib/games/haftgane/rules/rules-overlay.svelte';

  const DIAGRAMS = [TradeDiagram, PassingDiagram, ConflictDiagram, ScienceDiagram];
</script>

<div class="flex flex-col gap-6" data-testid="gallery-haftgane-rules">
  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Haftgane rules: diagrams <!-- i18n-ok --></h3>
    <div class="grid gap-3 sm:grid-cols-2">
      {#each DIAGRAMS as Diagram, i (i)}
        <Diagram />
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2" data-testid="gallery-haftgane-rules-compact">
    <h3 class="text-sm font-semibold text-fg-muted">
      Haftgane rules: diagrams, compact (220 px wide boxes) <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap items-start gap-3">
      {#each DIAGRAMS as Diagram, i (i)}
        <div class="w-[220px] rounded-control border border-dashed border-rule bg-panel p-1">
          <Diagram compact />
        </div>
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Haftgane rules: the overlay <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap gap-3">
      <Button
        variant="secondary"
        data-testid="gallery-haftgane-rules-open"
        onclick={() => void openHaftganeRules()}
      >
        openHaftganeRules() <!-- i18n-ok -->
      </Button>
      <Button
        variant="secondary"
        data-testid="gallery-haftgane-rules-open-trade"
        onclick={() => void openHaftganeRules('trade')}
      >
        openHaftganeRules('trade') <!-- i18n-ok -->
      </Button>
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Haftgane rules: the panel at 360 and 300 px <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap items-start gap-4">
      <div
        class="h-[560px] w-[360px] max-w-full overflow-y-auto rounded-panel border border-rule bg-panel"
      >
        <RulesPanel titleId="gallery-haftgane-rules-phone" />
      </div>
      <div
        class="h-[560px] w-[300px] max-w-full overflow-y-auto rounded-panel border border-rule bg-panel"
      >
        <RulesPanel titleId="gallery-haftgane-rules-desk" />
      </div>
    </div>
  </section>
</div>
