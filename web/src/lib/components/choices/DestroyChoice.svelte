<script lang="ts">
  // The destroy choice (the Statue of Zeus, Circus Maximus): the opponent's
  // cards of the named colour that the view offers, as medium cards. Select
  // one, then the danger-styled "Destroy <card>" sends its key. With a
  // single target it is selected already, so one tap does it.
  import { t } from '$lib/i18n';
  import type { Color } from '$lib/wire';
  import { cardName, colorName } from '../card/effect-text';
  import CardPickGrid from './CardPickGrid.svelte';
  import CommitButton from './CommitButton.svelte';

  interface Props {
    /** Card slug to its destroy key, from the view's choices. */
    keys: ReadonlyMap<string, string>;
    /** The colour to destroy (pending.color). */
    color?: Color;
    /** The opponent whose card goes (user text). */
    opponentName: string;
    canAct: boolean;
    /** Ask for a second tap before sending (the act-button rule). */
    confirm?: boolean;
    /** The connection is up. */
    connected?: boolean;
    onact: (key: string) => void;
  }

  let {
    keys,
    color,
    opponentName,
    canAct,
    confirm = false,
    connected = true,
    onact
  }: Props = $props();

  let picked = $state<string | null>(null);
  const cards = $derived([...keys.keys()]);
  const only = $derived(keys.size === 1 ? ([...keys.values()][0] ?? null) : null);
  const selected = $derived(picked !== null && [...keys.values()].includes(picked) ? picked : only);
  const selectedCard = $derived(
    selected === null ? null : (cards.find((slug) => keys.get(slug) === selected) ?? null)
  );
</script>

<div class="flex flex-col gap-3" data-destroy-choice>
  <p class="text-sm text-fg" dir="auto">
    {t('choices.destroy.intro', { name: opponentName, color: color ? colorName(color) : '' })}
  </p>
  <CardPickGrid
    {cards}
    {keys}
    {selected}
    onselect={(key) => (picked = key)}
    danger
    disabled={!canAct}
    label={t('choices.destroy.title')}
  />
  <CommitButton
    label={selectedCard
      ? t('choices.destroy.confirm', { card: cardName(selectedCard) })
      : t('choices.selectCard')}
    actKey={selected}
    danger
    disabled={!canAct}
    {confirm}
    {connected}
    {onact}
  />
</div>
