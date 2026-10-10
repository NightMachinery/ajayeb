<script lang="ts">
  // Build from the discard pile (the Mausoleum): the pile, latest age first,
  // with colour filter chips. Select a card, then "Build <card> for free"
  // sends its revive key. Every card in the pile the view offers a key for
  // is selectable; the rest are shown but not picked.
  import { t } from '$lib/i18n';
  import { cardName } from '../card/effect-text';
  import CommitButton from './CommitButton.svelte';
  import DiscardPile from './DiscardPile.svelte';

  interface Props {
    /** The discard pile (pending.cards). */
    cards: readonly string[];
    /** Card slug to its revive key, from the view's choices. */
    keys: ReadonlyMap<string, string>;
    canAct: boolean;
    onact: (key: string) => void;
  }

  let { cards, keys, canAct, onact }: Props = $props();

  let picked = $state<string | null>(null);
  const only = $derived(keys.size === 1 ? ([...keys.values()][0] ?? null) : null);
  const selected = $derived(picked !== null && [...keys.values()].includes(picked) ? picked : only);
  const selectedCard = $derived(
    selected === null
      ? null
      : ([...keys.keys()].find((slug) => keys.get(slug) === selected) ?? null)
  );
</script>

<div class="flex flex-col gap-3" data-discard-choice>
  <p class="text-sm text-fg">{t('choices.discard.intro')}</p>
  <DiscardPile {cards} {keys} {selected} onselect={(key) => (picked = key)} disabled={!canAct} />
  <CommitButton
    label={selectedCard
      ? t('choices.discard.confirm', { card: cardName(selectedCard) })
      : t('choices.selectCard')}
    actKey={selected}
    disabled={!canAct}
    {onact}
  />
</div>
