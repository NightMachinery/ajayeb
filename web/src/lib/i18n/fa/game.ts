// Persian messages for the game screen: the game menu, the forfeit step,
// the action buttons and the opponent move summary's frame. Partial; the
// card sheet's details fall back to English. See ./index.ts.

import type { game as en } from '../en/game';
import type { Translation } from '../types';

export const game = {
  'game.title': 'بازی {code}',
  'game.loading': 'در انتظار بازی…',
  'game.board': 'صفحه‌ی بازی',
  'game.openCity': 'نمایش شهر {name}',

  'game.action.build': { '=0': 'ساخت رایگان', other: 'ساخت با {count} سکه' },
  'game.action.discard': { other: 'کنار گذاشتن برای {count} سکه' },
  'game.action.wonder': 'ساخت یک شگفتی',
  'game.action.buildWonder': {
    '=0': 'ساخت رایگان {wonder}',
    other: 'ساخت {wonder} با {count} سکه'
  },

  'game.persona.easy': 'شاگرد',
  'game.persona.normal': 'دانشمند',
  'game.persona.hard': 'استاد',

  'game.menu.open': 'منوی بازی',
  'game.menu.home': 'خانه',
  'game.menu.homeHint': 'می‌توانید این بازی را از خانه ادامه دهید.',
  'game.menu.settings': 'تنظیمات',
  'game.menu.log': 'گزارش',
  'game.menu.replay': 'دریافت بازپخش',
  'game.menu.forfeit': 'واگذار کردن بازی',
  'game.menu.undo': 'برگرداندن آخرین حرکت',
  'game.menu.undoNothing': 'هنوز حرکتی از شما برای برگرداندن نیست.',
  'game.menu.undoOver': 'بازی تمام شده است.',
  'game.menu.undoWaiting': 'در انتظار سرور…',
  'game.menu.undoOffline': 'در انتظار اتصال.',

  'game.forfeit.title': 'بازی را واگذار می‌کنید؟',
  'game.forfeit.titleName': 'بازی را از طرف {name} واگذار می‌کنید؟',
  'game.forfeit.consequence': '{name} بی‌درنگ برنده می‌شود.',
  'game.forfeit.final': 'واگذاری بازی را برای همیشه تمام می‌کند و برگشتی ندارد.',
  'game.forfeit.confirm': 'واگذاری',
  'game.forfeit.cancel': 'ادامه‌ی بازی',

  'game.settings.title': 'تنظیمات',

  'game.summary.title': 'حرکت {name}',
  'game.summary.open': 'نمایش آخرین حرکت حریف',
  'game.summary.close': 'برای بستن هر جا را بزنید'
} as const satisfies Translation<typeof en>;
