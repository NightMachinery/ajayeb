<script lang="ts">
  // The rules reference (M10.RR1), for a visual check in both themes: the
  // diagrams alone (the concept diagrams also in their compact tooltip
  // size), the panel body as the in-game overlay shows it
  // (in a phone-wide and a desktop-wide frame), and buttons that open the
  // real overlay at the top and at a deep-linked section. The headings are
  // dev labels, not shipped copy.
  import ChainDiagram from '$lib/components/rules/ChainDiagram.svelte';
  import DiscardDiagram from '$lib/components/rules/DiscardDiagram.svelte';
  import GuildDiagram from '$lib/components/rules/GuildDiagram.svelte';
  import MilitaryDiagram from '$lib/components/rules/MilitaryDiagram.svelte';
  import RulesPanel from '$lib/components/rules/RulesPanel.svelte';
  import ScienceDiagram from '$lib/components/rules/ScienceDiagram.svelte';
  import SeventhWonderDiagram from '$lib/components/rules/SeventhWonderDiagram.svelte';
  import StarterDiagram from '$lib/components/rules/StarterDiagram.svelte';
  import TradeDiagram from '$lib/components/rules/TradeDiagram.svelte';
  import { openRules } from '$lib/components/rules/rules-overlay.svelte';
  import Button from '$lib/components/shell/Button.svelte';
</script>

<div class="flex flex-col gap-6" data-testid="gallery-rules">
  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">Rules: diagrams <!-- i18n-ok --></h3>
    <div class="grid gap-3 sm:grid-cols-2">
      <MilitaryDiagram />
      <TradeDiagram />
      <ScienceDiagram />
      <ChainDiagram />
      <DiscardDiagram />
      <SeventhWonderDiagram />
      <GuildDiagram />
      <StarterDiagram />
    </div>
  </section>

  <section class="flex flex-col gap-2" data-testid="gallery-rules-compact">
    <h3 class="text-sm font-semibold text-fg-muted">
      Rules: concept diagrams, compact (220 by 120 px boxes) <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap items-start gap-3">
      {#each [ChainDiagram, DiscardDiagram, SeventhWonderDiagram, GuildDiagram, StarterDiagram] as Diagram, i (i)}
        <div
          class="h-[120px] w-[220px] rounded-control border border-dashed border-rule bg-panel p-1"
        >
          <Diagram compact />
        </div>
      {/each}
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Rules: the overlay <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap gap-3">
      <Button variant="secondary" onclick={() => void openRules()}>
        openRules() <!-- i18n-ok -->
      </Button>
      <Button variant="secondary" onclick={() => void openRules('military')}>
        openRules('military') <!-- i18n-ok -->
      </Button>
      <Button variant="secondary" onclick={() => void openRules('tokens')}>
        openRules('tokens') <!-- i18n-ok -->
      </Button>
    </div>
  </section>

  <section class="flex flex-col gap-2">
    <h3 class="text-sm font-semibold text-fg-muted">
      Rules: the panel at 360 and 300 px <!-- i18n-ok -->
    </h3>
    <div class="flex flex-wrap items-start gap-4">
      <div
        class="h-[560px] w-[360px] max-w-full overflow-y-auto rounded-panel border border-rule bg-panel"
      >
        <RulesPanel titleId="gallery-rules-phone" />
      </div>
      <div
        class="h-[560px] w-[300px] max-w-full overflow-y-auto rounded-panel border border-rule bg-panel"
      >
        <RulesPanel titleId="gallery-rules-desk" />
      </div>
    </div>
  </section>
</div>
