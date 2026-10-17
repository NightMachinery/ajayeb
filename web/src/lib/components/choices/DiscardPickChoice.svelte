<script lang="ts">
  import { t } from '$lib/i18n';
  import { hintHide } from '$lib/hints/local.svelte';
  import { discardSections, sessionHintScreen, type HintScreen } from '$lib/view/hints';
  import type { View } from '$lib/wire';
  import HintLines from '../game/HintLines.svelte';
  import { tryGameContext } from './room';
  import { cardName } from '../card/effect-text';
  import CommitButton from './CommitButton.svelte';
  import DiscardPile from './DiscardPile.svelte';

  interface Props {
    /** The discard pile (pending.cards). */
    cards: readonly string[];
    /** Card slug to its revive key, from the view's choices. */
    keys: ReadonlyMap<string, string>;
    canAct: boolean;
    /** Ask for a second tap before sending (the act-button rule). */
    confirm?: boolean;
    /** The connection is up. */
    connected?: boolean;
    /** The view and hint display, when there is no game context (the gallery). */
    hintView?: Pick<View, 'hints'>;
    hints?: HintScreen;
    onact: (key: string) => void;
  }

  let {
    cards,
    keys,
    canAct,
    confirm = false,
    connected = true,
    hintView,
    hints,
    onact
  }: Props = $props();

  const ctx = tryGameContext();
  const screen = $derived(hints ?? sessionHintScreen(ctx?.session, hintHide.value));
  const source = $derived(hintView ?? ctx?.session.view ?? undefined);
  const sections = $derived(
    screen && source ? discardSections(source, cards, screen.perspective.seat, screen) : []
  );

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
  <HintLines {sections} class="max-h-48 overflow-y-auto" />
  <CommitButton
    label={selectedCard
      ? t('choices.discard.confirm', { card: cardName(selectedCard) })
      : t('choices.selectCard')}
    actKey={selected}
    disabled={!canAct}
    {confirm}
    {connected}
    {onact}
  />
</div>
