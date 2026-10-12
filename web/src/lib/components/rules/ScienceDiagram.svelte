<script lang="ts">
  // The science symbols, for the rules reference: the six of the green cards
  // and the scales of the Law token, each drawn from the icon sprite with its
  // translated name beside it.
  import { t } from '$lib/i18n';
  import Icon from '$lib/icons/Icon.svelte';

  const SIX = ['globe', 'sundial', 'mortar', 'set-square', 'quill', 'wheel'] as const;
</script>

<figure
  class="flex flex-col gap-2 rounded-panel border border-rule bg-panel-sunken p-3"
  data-rules-diagram="science"
>
  <figcaption class="text-xs font-semibold text-fg-muted">
    {t('rules.diagram.science.six')}
  </figcaption>
  <ul class="grid grid-cols-2 gap-x-3 gap-y-1.5 sm:grid-cols-3">
    {#each SIX as symbol (symbol)}
      <li class="flex items-center gap-2 text-sm text-fg">
        <Icon name={`science-${symbol}` as const} class="shrink-0 text-xl text-card-green-ink" />
        {t(`board.science.${symbol}`)}
      </li>
    {/each}
  </ul>
  <p class="flex items-center gap-2 border-t border-rule pt-2 text-sm text-fg">
    <Icon name="science-scales" class="shrink-0 text-xl text-fg-muted" />
    {t('rules.diagram.science.scales', { law: t('token.law') })}
  </p>
</figure>
