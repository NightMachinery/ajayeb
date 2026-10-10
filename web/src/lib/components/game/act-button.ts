// The one rule for every button that sends a move (an act key): the card
// sheet's Build, Discard and wonder buttons, and every pending choice's
// commit button (destroy, build from the discard pile, the progress tokens,
// who starts the next age). A display rule only: whether input is open is
// the caller's canAct (presenter idle and the session able to act), and the
// key is always one of the view's choices.
//
//  - While the connection is down the button is disabled and reads
//    "Waiting for connection" (UX design section 2, "Connection states").
//  - When the room asks for it (online rooms by default, or "Confirm moves:
//    Always"), the first tap arms the button and a second one sends
//    (section 5, "Commit, confirm, undo, pending").

import { t } from '$lib/i18n';

export interface ActButtonState {
  /** What the button says: the move, or the wait for the connection. */
  label: string;
  disabled: boolean;
  /** Arm on the first tap, send on the second. */
  confirm: boolean;
}

export function actButtonState(opts: {
  /** The move's own label, already translated. */
  label: string;
  /** Input is open now. */
  canAct: boolean;
  /** The connection is up. */
  connected: boolean;
  /** The room asks for the confirm step (needsConfirm). */
  confirm: boolean;
  /** Something is selected to send (a commit button before its pick: false). */
  ready?: boolean;
}): ActButtonState {
  return {
    label: opts.connected ? opts.label : t('online.waitingConnection'),
    disabled: !opts.canAct || !opts.connected || opts.ready === false,
    confirm: opts.confirm
  };
}
