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
  'shell.demo.announced': 'Vizier built Lumber Yard. Your turn.',

  // /dev/gallery/tooltips: the tooltip layer, the stacked tooltip and the
  // dismiss layer (TooltipLayer, TooltipBubble, TooltipStack, DismissLayer).
  'dev.tooltips.title': 'Tooltips',
  'dev.tooltips.intro':
    'Hover a symbol (after a short delay) or focus it with the keyboard; on touch, tap a symbol, or long-press an actionable one. Any tap closes an open tooltip without acting.',
  'dev.tooltips.back': 'Back to the gallery',
  'dev.tooltips.text': 'Text',
  'dev.tooltips.params': 'With parameters',
  'dev.tooltips.diagram': 'With a diagram',
  'dev.tooltips.stack': 'Stacked: chain symbols',
  'dev.tooltips.inCard': 'A chip inside an actionable card',
  'dev.tooltips.button': 'Long-press a button',
  'dev.tooltips.specimens': 'Bubble and stack, standalone',
  'dev.tooltips.dismiss': 'Dismiss layer',
  'dev.tooltips.openDismiss': 'Open a floating panel',
  'dev.tooltips.dismissBody': 'Any tap closes this panel, and the tap does nothing else.',
  'dev.tooltips.cardOpened': 'Card sheet opened {count} times',
  'dev.tooltips.pressed': 'Pressed {count} times',
  'dev.tooltips.pressMe': 'Build',
  'dev.tooltips.pressTip': 'Builds the card for its cost.',
  'dev.tooltips.laurelTip': 'Victory points: counted at the end of the game.',
  'dev.tooltips.diagramTip':
    'The conflict pawn moves one space toward the rival capital for each shield.',
  'dev.tooltips.diagramYou': 'Your capital',
  'dev.tooltips.diagramRival': 'Rival capital',
  'dev.tooltips.sheetTitle': 'Card sheet',
  'dev.tooltips.sheetBody': 'The card sheet would open here.'
} as const satisfies Messages;
