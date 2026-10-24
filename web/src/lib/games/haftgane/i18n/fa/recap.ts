// Persian messages for the reveal and the turn recap (../en/recap.ts). A
// coin line stands under its seat's name for any seat, the viewer's own
// included, so it is a noun of action («پرداخت ... سکه») that needs no
// person. See ./index.ts.

import type { Translation } from '$lib/i18n/types';
import type { recap as en } from '../en/recap';

export const recap = {
  'hg.recap.title': 'نوبت گذشته',
  'hg.recap.title.turn': 'نوبت {turn}، دوران {age}',
  'hg.recap.title.ageEnd': 'نوبت آخر دوران {age}',
  'hg.recap.title.sides': 'طرف شگفتی‌ها',
  'hg.recap.close': 'برای بستن هر جایی را بزنید',
  'hg.recap.open': 'نمایش نوبت گذشته',
  'hg.recap.open.unseen': 'نمایش نوبت گذشته، هنوز دیده نشده',
  'hg.recap.none': 'هنوز هیچ نوبتی بازی نشده است.',

  'hg.recap.build': '{name} {card} را ساخت',
  'hg.recap.you.build': '{card} را ساختید',
  'hg.recap.buildDiscarded': '{name} {card} را از کارت‌های کنارگذاشته ساخت',
  'hg.recap.you.buildDiscarded': '{card} را از کارت‌های کنارگذاشته ساختید',
  'hg.recap.stage': '{name} مرحله‌ی {stage} شگفتی را ساخت',
  'hg.recap.you.stage': 'مرحله‌ی {stage} شگفتی را ساختید',
  'hg.recap.you.stageWith': 'مرحله‌ی {stage} شگفتی را با {card} ساختید',
  'hg.recap.sell': '{name} کارتی فروخت',
  'hg.recap.you.sell': '{card} را فروختید',
  'hg.recap.you.sellHidden': 'کارتی فروختید',
  'hg.recap.discardNothing': '{name} در کارت‌های کنارگذاشته چیزی برای ساختن نیافت',
  'hg.recap.you.discardNothing': 'در کارت‌های کنارگذاشته چیزی برای ساختن نیافتید',
  'hg.recap.side': '{name} طرف {side} را برگزید',
  'hg.recap.you.side': 'طرف {side} را برگزیدید',
  'hg.recap.idle': '{name} از خرید دیگران سکه گرفت',
  'hg.recap.you.idle': 'از خرید دیگران سکه گرفتید',
  'hg.recap.kept': '{name} کارت آخر را برای نوبتی اضافه نگه می‌دارد',
  'hg.recap.you.kept': '{card} را برای نوبتی اضافه نگه می‌دارید',
  'hg.recap.you.leftover': 'کارت آخر شما، {card}، به کارت‌های کنارگذاشته رفت',

  'hg.recap.tag.stage': 'مرحله‌ی {stage}',
  'hg.recap.tag.sold': 'فروخته شد',
  'hg.recap.tag.soldFor': 'فروش {coins}',
  'hg.recap.tag.nothing': 'هیچ',

  'hg.recap.coin.bank': { other: 'پرداخت {count} سکه به بانک' },
  'hg.recap.coin.paid': { other: 'پرداخت {count} سکه به {name}' },
  'hg.recap.coin.received': { other: 'دریافت {count} سکه از {name}' },
  'hg.recap.coin.sale': { other: 'دریافت {count} سکه برای فروش' },
  'hg.recap.coin.card': { other: 'دریافت {count} سکه از {card}' },
  'hg.recap.coin.stage': { other: 'دریافت {count} سکه از مرحله‌ی {stage} شگفتی' },

  'hg.recap.pass.left': 'دست‌ها به چپ می‌روند.',
  'hg.recap.pass.right': 'دست‌ها به راست می‌روند.',
  'hg.recap.pass.hold': { other: 'اکنون {count} کارتی را دارید که {name} به شما داد.' },
  'hg.recap.ageEnd': 'دوران {age} به پایان می‌رسد: نبردهای نظامی در پیش است.',
  'hg.recap.ageStart': 'دوران {age} با دست‌های تازه آغاز می‌شود.',
  'hg.recap.gameOver': 'بازی تمام شد: امتیازها در پیش است.',

  'hg.recap.change.seated': '{by} {user} را در جایگاه {seat} نشاند',
  'hg.recap.change.toBot': '{by} جایگاه {seat} را به یک ربات سپرد',
  'hg.recap.change.resign': '{by} کناره گرفت و یک ربات در جایگاه {seat} بازی می‌کند',
  'hg.recap.change.promote': '{by} {user} را مدیر کرد',
  'hg.recap.change.demote': '{by} نقش مدیر را از {user} گرفت',
  'hg.recap.change.stepDown': '{user} از مدیری کناره گرفت',
  'hg.recap.change.temporary': 'تا وقتی مدیران غایب‌اند، {user} مدیر موقت است',
  'hg.recap.someone': 'بازیکنی که رفت'
} as const satisfies Translation<typeof en>;
