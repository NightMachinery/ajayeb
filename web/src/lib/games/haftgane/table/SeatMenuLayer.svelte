<script lang="ts">
  // Connects one seat's admin menu to the table: reads the occupant, the
  // spectators who may take the seat, the role buttons and the viewer's
  // powers from the frame, and sends the chosen action through the
  // Session. The menu itself (AdminSeatMenu) takes plain props.
  import type { TableState } from '$lib/wire/platform.gen';
  import { ht } from '../i18n/messages.svelte';
  import type { TableSession } from '../session/session.svelte';
  import AdminSeatMenu from './AdminSeatMenu.svelte';
  import { botLevels, isTemporary, occupant, seatable, type SeatAction } from './admin';
  import { roleActions } from './model';
  import { seatName, where } from './seat-names';

  interface Props {
    session: TableSession;
    frame: TableState;
    ready: boolean;
    seat: number;
    onclose: () => void;
  }

  let { session, frame, ready, seat, onclose }: Props = $props();

  const name = $derived(seatName(frame, seat));
  const place = $derived(frame.view.seats.find((v) => v.seat === seat));
  const who = $derived(occupant(frame, seat));
  const levels = $derived(botLevels(frame.botLevels));

  function act(action: SeatAction): void {
    switch (action.kind) {
      case 'bot':
        session.switchSeat(seat, { bot: action.level });
        break;
      case 'human':
        session.switchSeat(seat, { userId: action.userId });
        break;
      case 'promote':
      case 'demote':
        session.role(action.kind, action.userId);
        break;
    }
  }
</script>

<AdminSeatMenu
  {seat}
  {name}
  crestTip={place
    ? ht('hg.screen.crest.tip', { name, where: where(place, frame.seats.length) })
    : name}
  occupant={who}
  {levels}
  bot={frame.seats.find((s) => s.seat === seat && s.kind === 'bot')?.bot ?? null}
  spectators={seatable(frame)}
  roles={who ? roleActions(frame, who.userId) : { promote: false, demote: false }}
  temporary={isTemporary(frame)}
  disabled={!ready || !session.canControl}
  onaction={act}
  {onclose}
/>
