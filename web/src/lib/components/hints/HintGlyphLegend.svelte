<script lang="ts">
  // The rules reference's "Hint glyphs" legend (the hints design,
  // hint-explainer-tooltip, M10.HS5): each board hint mark drawn as the
  // board draws it (HintPip), with what it means, then how its fill tells
  // whose fact it is, the dot for more hints, and the advisor's review.
  // Display only, from fixed sample pips; it states what the marks mean,
  // not any game's facts. A leaf with no props, for the rules sections and
  // the gallery.
  import Lightbulb from '@lucide/svelte/icons/lightbulb';
  import { t, type MessageKey } from '$lib/i18n';
  import type { CardPip, PipEntry } from '$lib/view/hints';
  import HintPip from '../card/HintPip.svelte';

  function pip(e: Omit<PipEntry, 'text'>, more = false): CardPip {
    const top: PipEntry = { ...e, text: t('rules.hints.sampleCard') };
    return { top, entries: more ? [top, { ...top, code: 'loot' }] : [top], more };
  }

  const own = { seat: 0, side: 'own', style: 'solid', tone: 'gold' } as const;
  const opp = { seat: 1, side: 'opp', style: 'outline', tone: 'moon' } as const;

  const marks: readonly { key: MessageKey; pip: CardPip; id: string }[] = [
    { id: 'win', key: 'rules.hints.win', pip: pip({ ...own, code: 'win' }) },
    { id: 'pair', key: 'rules.hints.pair', pip: pip({ ...own, code: 'pair' }) },
    { id: 'loot', key: 'rules.hints.loot', pip: pip({ ...own, code: 'loot' }) }
  ];

  const sides: readonly { key: MessageKey; pips: CardPip[]; id: string }[] = [
    { id: 'own', key: 'rules.hints.own', pips: [pip({ ...own, code: 'pair' })] },
    { id: 'opp', key: 'rules.hints.opp', pips: [pip({ ...opp, code: 'pair' })] },
    {
      id: 'seat',
      key: 'rules.hints.seat',
      pips: [
        pip({ seat: 0, side: 'own', style: 'solid', tone: 'sun', code: 'pair' }),
        pip({ seat: 1, side: 'opp', style: 'tint', tone: 'moon', code: 'pair' })
      ]
    },
    { id: 'more', key: 'rules.hints.more', pips: [pip({ ...own, code: 'win' }, true)] }
  ];

  const card = t('rules.hints.sampleCard');
</script>

<div class="flex flex-col gap-3" data-hint-legend>
  <ul class="flex flex-col divide-y divide-rule rounded-panel border border-rule bg-panel">
    {#each marks as m (m.id)}
      <li class="flex items-start gap-3 px-3 py-2" data-hint-legend-row={m.id}>
        <span class="legend-card relative mt-0.5 shrink-0">
          <HintPip pip={m.pip} {card} class="absolute -end-1.5 -top-1.5" />
        </span>
        <span class="text-sm leading-snug text-fg">{t(m.key as 'rules.hints.win')}</span>
      </li>
    {/each}
  </ul>
  <ul class="flex flex-col divide-y divide-rule rounded-panel border border-rule bg-panel">
    {#each sides as s (s.id)}
      <li class="flex items-start gap-3 px-3 py-2" data-hint-legend-row={s.id}>
        <span class="flex shrink-0 gap-2">
          {#each s.pips as p, i (i)}
            <span class="legend-card relative mt-0.5">
              <HintPip pip={p} {card} class="absolute -end-1.5 -top-1.5" />
            </span>
          {/each}
        </span>
        <span class="text-sm leading-snug text-fg">{t(s.key as 'rules.hints.win')}</span>
      </li>
    {/each}
    <li class="flex items-start gap-3 px-3 py-2" data-hint-legend-row="review">
      <span class="grid size-8 shrink-0 place-items-center text-accent">
        <Lightbulb class="size-5" aria-hidden="true" />
      </span>
      <span class="text-sm leading-snug text-fg">{t('rules.hints.review')}</span>
    </li>
  </ul>
</div>

<style>
  /* A small paper corner standing for the card the mark sits on. */
  .legend-card {
    display: block;
    width: 1.75rem;
    height: 2.25rem;
    border-radius: 0.25rem;
    background: var(--card-paper);
    border: 1px solid var(--rule-strong);
  }
</style>
