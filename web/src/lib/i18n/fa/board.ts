// Persian messages for the board: the turn bar and the top bar. Partial;
// the rest of the board falls back to English. See ./index.ts.

import type { board as en } from '../en/board';
import type { Translation } from '../types';

export const board = {
  'board.turn.yourTurn': 'نوبت شما',
  'board.turn.playAgain': 'دوباره بازی کنید',
  'board.turn.playAgainWonder': 'دوباره بازی کنید ({wonder})',
  'board.turn.playAgainName': '{name} دوباره بازی می‌کند',
  'board.turn.playAgainNameWonder': '{name} دوباره بازی می‌کند ({wonder})',
  'board.turn.aiThinking': '{name} در حال فکر کردن است…',
  'board.turn.waitingFor': 'در انتظار {name}…',
  'board.turn.waitingAway': '{name} حاضر نیست. منتظر بازگشتش می‌مانیم…',
  'board.turn.sent': 'در حال فرستادن…',
  'board.turn.sentSlow': 'هنوز در انتظار سرور…',
  'board.turn.retry': 'تلاش دوباره',
  'board.turn.gameOver': 'پایان بازی',
  'board.turn.spectating': 'در حال تماشا',
  'board.turn.connecting': 'در حال اتصال…',
  'board.turn.pending.chooseStarter': 'انتخاب کنید چه کسی دوران بعد را آغاز کند',
  'board.turn.pending.destroyCard': 'کارتی را برای نابودی انتخاب کنید',
  'board.turn.pending.buildFromDiscard': 'کارتی از کارت‌های کنارگذاشته برای ساخت انتخاب کنید',
  'board.turn.pending.pickBoardToken': 'یک نشان پیشرفت انتخاب کنید',
  'board.turn.pending.pickLibraryToken': 'یک نشان از کتابخانه‌ی بزرگ انتخاب کنید',

  'board.age': 'دوران {age}',
  'board.age.draft': 'گزینش شگفتی‌ها',
  'board.topbar.undo': 'برگرداندن',
  'board.topbar.log': 'گزارش',
  'board.topbar.settings': 'تنظیمات',

  'board.wonders.tally': '{built} از {total} ساخته شده',
  'board.wonders.back': 'همه‌ی شگفتی‌ها',
  'board.player.points': { other: '{count} امتیاز' },
  'board.player.thinking': 'در حال فکر'
} as const satisfies Translation<typeof en>;
