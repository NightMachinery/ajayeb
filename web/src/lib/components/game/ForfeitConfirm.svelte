<script lang="ts">
  // The confirm step of a forfeit, opened from the game menu: it names the
  // consequence (the other player wins at once, and it cannot be undone)
  // before anything is sent. Forfeit reports onconfirm, which sends the
  // documented `resign:<seat>` act through the Session (docs/protocol.md,
  // "Forfeit"); Keep playing closes it. A leaf component with plain props:
  // the caller decides whether a forfeit is possible now (Session.resignSeat)
  // and gives the two names. In a hot-seat game the player to move is named,
  // since both players share the device. The dialog opens with the focus on
  // Keep playing, never on the destructive answer.
  import Flag from '@lucide/svelte/icons/flag';
  import { t } from '$lib/i18n';
  import Button from '../shell/Button.svelte';
  import Sheet from '../shell/Sheet.svelte';

  interface Props {
    open: boolean;
    /** Who forfeits, when the device is shared (hot-seat); else "you". */
    name?: string;
    /** Who would win (the other seat's display name). */
    winner: string;
    onconfirm: () => void;
    onclose: () => void;
  }

  let { open, name, winner, onconfirm, onclose }: Props = $props();

  const titleId = 'forfeit-title';
</script>

<Sheet bind:open={() => open, (v) => !v && onclose()} {titleId} centered>
  <div class="flex flex-col gap-3 px-5 pt-5 pb-1" data-forfeit-confirm>
    <div class="flex items-center gap-2">
      <Flag class="size-5 shrink-0 text-cinnabar-ink" aria-hidden="true" />
      <h2 id={titleId} class="font-display text-xl text-fg">
        {name !== undefined ? t('game.forfeit.titleName', { name }) : t('game.forfeit.title')}
      </h2>
    </div>
    <p class="text-sm text-fg">{t('game.forfeit.consequence', { name: winner })}</p>
    <p class="text-sm text-fg-muted">{t('game.forfeit.final')}</p>
    <div class="mt-1 flex flex-col gap-2 split:flex-row-reverse" data-sheet-actions>
      <Button
        variant="primary"
        full
        class="bg-cinnabar! text-white!"
        data-control="forfeit-confirm"
        onclick={onconfirm}
      >
        {t('game.forfeit.confirm')}
      </Button>
      <!-- The dialog opens on the safe answer, so a stray Enter keeps the game going. -->
      <Button variant="secondary" full autofocus data-control="forfeit-cancel" onclick={onclose}>
        {t('game.forfeit.cancel')}
      </Button>
    </div>
  </div>
</Sheet>
