// Persian messages for the admin controls, role tags, resignation and the
// spectator view (../en/admin.ts). Every confirmation says what changes, as
// the English does. See ./index.ts.

import type { Translation } from '$lib/i18n/types';
import type { admin as en } from '../en/admin';

export const admin = {
  'hg.admin.role.creator': 'سازنده',
  'hg.admin.role.admin': 'مدیر',
  'hg.admin.role.temporary': 'مدیر موقت',
  'hg.admin.role.creator.tip': '{name} این میز را ساخته و مدیر آن است.',
  'hg.admin.role.admin.tip': '{name} مدیر است: می‌تواند بازیکنان و ربات‌ها را بنشاند.',
  'hg.admin.role.temporary.tip':
    'تا وقتی همه‌ی مدیران غایب‌اند، {name} مدیر موقت است. این نقش با بازگشت یکی از مدیران تمام می‌شود.',

  'hg.admin.seat.open': 'مدیریت جایگاه {name}',
  'hg.admin.seat.title': 'جایگاه {seat}: {name}',
  'hg.admin.seat.close': 'بستن فهرست جایگاه',
  'hg.admin.seat.player': 'بازیکن: {name}',
  'hg.admin.seat.bot': 'در دست ربات',
  'hg.admin.seat.botLevel': 'در دست ربات: {bot}',
  'hg.admin.seat.toBot': 'سپردن جایگاه به ربات',
  'hg.admin.seat.changeLevel': 'سپردن جایگاه به ربات دیگر',
  'hg.admin.seat.seatHere': 'نشاندن یک تماشاگر در اینجا',
  'hg.admin.seat.noSpectators': 'هیچ تماشاگری برای نشستن متصل نیست.',
  'hg.admin.seat.hint': 'پیش‌تر در جایگاه {seat}',
  'hg.admin.seat.hint.tip': 'فقط مدیران می‌بینند که یک تماشاگر پیش‌تر در این بازی کجا نشسته بود.',
  'hg.admin.seat.roles': 'نقش مدیر',
  'hg.admin.seat.temporary':
    'به‌عنوان مدیر موقت می‌توانید بازیکنان و ربات‌ها را بنشانید، اما نمی‌توانید مدیران را تغییر دهید.',

  'hg.admin.confirm.toBot':
    'از این پس {bot} در جایگاه {seat} بازی می‌کند. {name} تماشا می‌کند و ربات انتخاب او را، تأییدشده یا نه، بازی می‌کند.',
  'hg.admin.confirm.botToBot': 'از این پس {bot} در جایگاه {seat} بازی می‌کند.',
  'hg.admin.confirm.seat':
    'از این پس {name} در جایگاه {seat} بازی می‌کند. انتخاب ربات می‌ماند تا {name} آن را تغییر دهد.',
  'hg.admin.confirm.seatReplace':
    'از این پس {name} در جایگاه {seat} بازی می‌کند و {old} تماشا می‌کند. انتخاب او می‌ماند تا {name} آن را تغییر دهد.',
  'hg.admin.confirm.promote':
    '{name} مدیر می‌شود: می‌تواند بازیکنان و ربات‌ها را بنشاند و دیگران را مدیر کند.',
  'hg.admin.confirm.demote': '{name} دیگر مدیر نیست.',
  'hg.admin.confirm.stepDown':
    'دیگر مدیر این میز نیستید. فقط یک مدیر دیگر می‌تواند دوباره شما را مدیر کند.',
  'hg.admin.confirm.resign':
    'از این پس یک ربات در جایگاه شما بازی می‌کند و شما بقیه‌ی بازی را تماشا می‌کنید. یک مدیر می‌تواند دوباره شما را بنشاند.',
  'hg.admin.confirm.ok': 'تأیید',
  'hg.admin.confirm.back': 'بازگشت',

  'hg.admin.promote': 'مدیر کردن',
  'hg.admin.demote': 'گرفتن نقش مدیر',
  'hg.admin.stepDown': 'کناره‌گیری از مدیری',
  'hg.admin.resign': 'واگذاری به ربات',
  'hg.admin.undo': 'برگرداندن',
  'hg.admin.undo.tip': 'برگرداندن آخرین نوبت شما',
  'hg.admin.undoOff': 'بازیکن دیگری جایگاهی گرفت، پس برگرداندن تا پایان بازی خاموش است.',

  'hg.admin.seats': 'جایگاه‌ها',
  'hg.admin.seats.manage': 'مدیریت',
  'hg.admin.spectators': { other: '{count} تماشاگر' },
  'hg.admin.spectators.title': 'تماشاگران',
  'hg.admin.spectators.none': 'کسی تماشا نمی‌کند.',
  'hg.admin.spectators.open': { other: '{count} نفر تماشا می‌کنند. نشان دادن آن‌ها.' },
  'hg.admin.you': 'شما',
  'hg.admin.away': 'غایب',
  'hg.admin.away.tip': '{name} متصل نیست.',

  'hg.watch.title': 'شما تماشا می‌کنید',
  'hg.watch.follow': 'دنبال کردن یک جایگاه',
  'hg.watch.follow.tip': 'متمرکز کردن میز روی {name}',
  'hg.watch.last': 'نوبت گذشته',
  'hg.watch.former.bot':
    'یک ربات در جایگاه شما بازی می‌کند. یک مدیر می‌تواند دوباره شما را بنشاند.',
  'hg.watch.former.human':
    'اکنون {name} در جایگاه شما بازی می‌کند. یک مدیر می‌تواند دوباره شما را بنشاند.'
} as const satisfies Translation<typeof en>;
