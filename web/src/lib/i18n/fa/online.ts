// Persian messages for online play: the create and join sheets, the name
// prompt, the lobby with its invite, presence, spectators, the online
// rematch, the room errors, device links and the names of the players.
// Complete: every key of ../en/online.ts. Kept in Latin script on purpose:
// the letters and digits a game code never uses (I, L, O, 0, 1) in
// 'online.join.invalid', and QR. The invite text a player sends a friend
// ('online.lobby.shareText') is informal on purpose. See ./index.ts.

import type { online as en } from '../en/online';
import type { Translation } from '../types';

export const online = {
  // The create sheet (Home, Play online).
  'online.create.title': 'بازی آنلاین',
  'online.create.intro': 'بازی‌ای بسازید، سپس پیوندش را برای دوستتان بفرستید.',
  'online.create.submit': 'ساختن بازی',
  'online.create.haveCode': 'کد دارید؟ به بازی بپیوندید',
  'online.name.label': 'نام شما',
  'online.name.hint': 'حریفتان این نام را می‌بیند.',
  // Suggested names, comma separated (an ASCII comma, which the code splits
  // on); one is offered when no name is stored.
  'online.name.suggestions': 'کاوه,رکسانا,داریوش,شیرین,بابک,پریسا,کوروش,میترا',
  'online.crest.legend': 'نماد شما',
  'online.crest.random': 'تصادفی',
  'online.crest.sun': 'خورشید',
  'online.crest.moon': 'ماه',
  'online.crest.note': 'آغازکننده به قید قرعه تعیین می‌شود.',

  // The join sheet (Home, Join with a code).
  'online.join.title': 'پیوستن به بازی',
  'online.join.label': 'کد بازی',
  'online.join.hint': 'شش حرف و رقم پایان پیوند دوستتان.',
  'online.join.invalid': 'کد بازی شش حرف و رقم است و هرگز I، L، O، 0 یا 1 ندارد.',
  'online.join.submit': 'پیوستن',

  // The name prompt (a first visit through an invite link).
  'online.prompt.title': 'شما را چه صدا کنیم؟',
  'online.prompt.intro': 'دوستتان در طول بازی این نام را می‌بیند.',
  'online.prompt.submit': 'بازی با نام {name}',

  // The lobby: the room before the second player arrives.
  'online.lobby.title': 'دوستتان را دعوت کنید',
  'online.lobby.intro': 'پیوند یا کد را برایش بفرستید. بازی با پیوستن او آغاز می‌شود.',
  'online.lobby.code': 'کد بازی',
  'online.lobby.link': 'پیوند دعوت',
  'online.lobby.copy': 'رونوشت پیوند',
  'online.lobby.copied': 'پیوند رونوشت شد.',
  'online.lobby.copyFailed': 'رونوشت پیوند ممکن نشد. آن را انتخاب کنید و خودتان رونوشت کنید.',
  'online.lobby.share': 'هم‌رسانی',
  'online.lobby.shareTitle': 'یک دست عجایب',
  'online.lobby.shareText': 'به بازی عجایب من بپیوند. کد بازی {code} است.',
  'online.lobby.qr': 'رمزینه‌ی QR پیوند دعوت',
  'online.lobby.scan': 'یا این را با گوشی دیگر اسکن کنید.',
  'online.lobby.players': 'بازیکنان',
  'online.lobby.waiting': 'در انتظار دوستتان…',
  'online.lobby.home': 'خانه',
  'online.lobby.homeHint': 'بازی باز می‌ماند: از خانه ادامه‌اش دهید.',

  // Presence on a player's strip.
  'online.presence.connected': 'آنلاین',
  'online.presence.away': 'غایب',
  'online.presence.connectedLabel': '{name} آنلاین است',
  'online.presence.awayLabel': '{name} غایب است',

  // Toasts in the game screen.
  'online.toast.joined': '{name} پیوست. گزینش شگفتی‌ها آغاز می‌شود.',
  'online.toast.left': '{name} رفت. بازی منتظرش می‌ماند.',
  'online.toast.back': '{name} برگشت.',
  'online.toast.rematch': '{name} بازی دوباره می‌خواهد.',

  // Spectators. A label with the number at the end, so one form serves all.
  'online.spectatorsLabel': { other: 'شمار تماشاگران این بازی: {count}' },
  'online.watching': 'شما تماشاگرید: فقط بازیکنان حرکت می‌کنند.',

  // The connection, on move buttons.
  'online.waitingConnection': 'در انتظار اتصال',

  // The rematch, on the result screen.
  'online.rematch.join': 'پیوستن به بازی دوباره',
  'online.rematch.watch': 'تماشای بازی دوباره',

  // A room that cannot be opened.
  'online.error.title': 'این بازی باز نمی‌شود',
  'online.error.notFoundHint':
    'کد یا پیوند را بررسی کنید. هر بازی چندی پس از پایانش هم بسته می‌شود.',
  'online.error.retryHint': 'لحظه‌ای دیگر دوباره تلاش کنید، یا خودتان بازی تازه‌ای شروع کنید.',
  // A pass-and-play or bot game opens only in the browser that started it.
  'online.error.forbidden': 'این بازی مال مرورگر یا دستگاه دیگری است.',
  'online.error.forbiddenHint':
    'بازی دست‌به‌دست یا بازی با هوش مصنوعی فقط در مرورگری باز می‌شود که آن را شروع کرده است. برای بازی در اینجا، آن را در همان مرورگر باز کنید و پیوند دستگاهش را با دکمه‌ی پیوند کنار نامتان رونوشت کنید.',
  'online.error.home': 'خانه',

  // Who is who (components/identity, lib/view/display-name): a name that
  // equals one already in the game gets a number after it, and the
  // player's own name a small mark.
  'online.identity.numbered': '{name} {number}',
  'online.identity.you': 'شما',
  'online.identity.youLabel': 'این شمایید',

  // Device links (components/identity/DeviceLink): a link that plays one
  // seat of this game on another device.
  'online.link.button': 'پیوند دستگاه برای {name}',
  'online.link.buttonOwn': 'بازی روی دستگاهی دیگر',
  'online.link.title': 'بازی روی دستگاهی دیگر',
  'online.link.titleOther': 'پیوند دستگاه برای {name}',
  'online.link.intro':
    'این پیوند را روی دستگاه دیگرتان باز کنید تا جایگاهتان را آنجا بازی کنید. هر کس آن را داشته باشد می‌تواند در این بازی به جای شما بازی کند، پس آن را پیش خودتان نگه دارید.',
  'online.link.introOther':
    'این پیوند را برای {name} بفرستید: با آن جایگاهش را روی دستگاهی دیگر بازی می‌کند. هر کس آن را داشته باشد می‌تواند در این بازی آن جایگاه را بازی کند.',
  'online.link.field': 'پیوند دستگاه',
  'online.link.loading': 'در حال گرفتن پیوند…',
  'online.link.failed': 'گرفتن پیوند ممکن نشد. دوباره تلاش کنید.',
  'online.link.retry': 'تلاش دوباره',
  'online.link.copy': 'رونوشت پیوند',
  'online.link.copied': 'پیوند رونوشت شد.',
  'online.link.copyFailed': 'رونوشت پیوند ممکن نشد. آن را انتخاب کنید و خودتان رونوشت کنید.',
  'online.link.reset': 'بازنشانی پیوند',
  'online.link.resetHint': 'بازنشانی همه‌ی پیوندهای پیشین این جایگاه را بی‌درنگ از کار می‌اندازد.',
  'online.link.resetDone': 'پیوند بازنشانی شد. پیوندهای پیشین دیگر کار نمی‌کنند.',

  // The game log: a reset of a device link.
  'online.log.linkResetOwn': '{name} پیوند دستگاه خود را بازنشانی کرد.',
  'online.log.linkResetOther': '{name} پیوند دستگاه {target} را بازنشانی کرد.',
  'online.log.linkReset': 'پیوند دستگاه {target} بازنشانی شد.'
} as const satisfies Translation<typeof en>;
