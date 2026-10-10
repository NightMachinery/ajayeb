// English messages for the shell primitives (components/shell/*): Sheet,
// Toasts, Segmented, ConfirmButton, LiveRegion. See ../index.ts for the
// conventions. Other areas get their own file (card.ts, board.ts, home.ts,
// game.ts), merged in ../index.ts; keep new keys here to this area.

import type { Messages } from '../types';

export const shell = {
  'shell.close': 'Close',
  'shell.dismiss': 'Dismiss',
  // {label} is the action's own text, e.g. "Build for 4 coins".
  'shell.confirmArmed': 'Confirm: {label}',
  'shell.tapAgainToConfirm': 'Tap again to confirm',

  // /dev/gallery demo strings for the primitives in this file's folder.
  'shell.demo.openSheet': 'Open a sheet',
  'shell.demo.sheetTitle': 'Example sheet',
  'shell.demo.sheetBody':
    'A native dialog: Esc, a backdrop tap, or the drag handle all close it, and focus returns to the opener.',
  'shell.demo.showToast': 'Show a toast',
  'shell.demo.toastMessage': 'Built Lumber Yard',
  'shell.demo.toastUndo': 'Undo',
  'shell.demo.segmentedLegend': 'First player',
  'shell.demo.optionRandom': 'Random',
  'shell.demo.optionMe': 'Me',
  'shell.demo.optionRival': 'Rival',
  'shell.demo.confirmLabel': 'Build for 4 coins',
  'shell.demo.announce': 'Announce an event',
  'shell.demo.announced': 'Vizier built Lumber Yard. Your turn.'
} as const satisfies Messages;
