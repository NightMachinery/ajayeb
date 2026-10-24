// Persian messages for the city sheet and the Seen hands sheet
// (../en/city.ts). See ./index.ts.

import type { Translation } from '$lib/i18n/types';
import type { city as en } from '../en/city';

export const city = {
  'hg.city.open': 'باز کردن شهر {name}',
  'hg.city.close': 'بستن شهر',
  'hg.city.tabs': 'شهرهای دور میز',
  'hg.city.wonder': '{wonder}، طرف {side}',
  'hg.city.wonder.noSide': '{wonder}، هنوز طرفی انتخاب نشده',

  'hg.city.score': 'امتیاز تا اینجا',
  'hg.city.score.wonder': 'شگفتی',
  'hg.city.score.treasure': 'سکه‌ها',
  'hg.city.score.military': 'نبردها',
  'hg.city.score.blue': 'مدنی',
  'hg.city.score.yellow': 'تجاری',
  'hg.city.score.green': 'علمی',
  'hg.city.score.purple': 'انجمن‌ها',
  'hg.city.victories.tip': { other: 'دوران {age}: {count} نشان پیروزی' },
  'hg.city.defeats.tip': { other: '{count} نشان شکست' },
  'hg.city.science.tip': {
    other: 'علم: {compass} پرگار، {gear} چرخ‌دنده، {tablet} لوح، به ارزش {count} امتیاز'
  },
  'hg.city.science.chosen': {
    other: '{count} نماد از آن‌ها انتخابی است و جایی شمرده می‌شود که بیشترین امتیاز را بیاورد'
  },

  'hg.city.stages': 'مرحله‌های شگفتی',
  'hg.city.stage.label': 'مرحله‌ی {stage}',
  'hg.city.stage.built': 'ساخته‌شده با کارتی پنهان',
  'hg.city.stage.notBuilt': 'ساخته‌نشده',
  'hg.city.stage.cost': 'هزینه',
  'hg.city.under.own': 'زیر شگفتی شما، که فقط شما می‌بینید: {cards}',
  'hg.city.under.revealed': 'زیر شگفتی، که در پایان رو شد: {cards}',
  'hg.city.kept.own': '{card} را نگه می‌دارید تا در پایان دوران بازی کنید.',
  'hg.city.kept.other': 'کارتی نگه داشته تا در پایان دوران بازی کند.',

  'hg.city.production': 'تولید',
  'hg.city.production.own': 'دو همسایه‌ی شما می‌توانند آن را بخرند',
  'hg.city.production.other': 'فقط دو همسایه‌ی {name} می‌توانند آن را بخرند',
  'hg.city.buildings': { other: 'شهر: {count} ساختمان' },
  'hg.city.row.none': 'هیچ',
  // In place of the colour rows, while a city has no building.
  'hg.city.row.nothing': 'هنوز چیزی ساخته نشده',
  // Between two card names in a list.
  'hg.city.row.sep': '، ',
  'hg.city.last': 'نوبت گذشته: {what}',
  'hg.city.last.build': 'ساختن {card}',
  'hg.city.last.build-discarded': 'ساختن {card} از کارت‌های کنارگذاشته',
  'hg.city.last.stage': 'ساختن مرحله‌ی {stage} شگفتی',
  'hg.city.last.sell': 'فروش یک کارت',

  'hg.seen.open': 'دست‌های دیده‌شده',
  'hg.seen.open.tip': 'دست‌هایی که داشته‌اید و کارت‌های کنارگذاشته‌ای که از نظر گذرانده‌اید',
  'hg.seen.title': 'دست‌های دیده‌شده',
  'hg.seen.close': 'بستن دست‌های دیده‌شده',
  'hg.seen.intro': 'این دفترچه را فقط شما می‌بینید. ربات‌ها هم دقیقاً همین‌قدر به یاد دارند.',
  'hg.seen.ages': 'دوران‌ها',
  'hg.seen.turn': 'نوبت {turn}',
  'hg.seen.lastCard': 'کارت آخر',
  'hg.seen.pile': 'کارت‌های کنارگذاشته‌ای که از نظر گذراندید، نوبت {turn}',
  'hg.seen.played.build': 'ساخته شد',
  'hg.seen.played.stage': 'زیر یک مرحله',
  'hg.seen.played.sell': 'فروخته شد',
  'hg.seen.played.build-discarded': 'ساخته شد',
  'hg.seen.played.tip.build': 'این کارت را ساختید.',
  'hg.seen.played.tip.stage': 'این کارت را زیر یک مرحله‌ی شگفتی گذاشتید.',
  'hg.seen.played.tip.sell': 'این کارت را فروختید.',
  'hg.seen.played.tip.build-discarded': 'این کارت را از کارت‌های کنارگذاشته ساختید.',
  'hg.seen.empty': 'هنوز چیزی نیست. هر دستی که داشته باشید، پس از رو شدن نوبتش اینجا می‌آید.',
  'hg.seen.emptyAge': 'هنوز چیزی از این دوران نیست.',

  'hg.seen.summary': 'آنچه می‌دانید',
  'hg.seen.circulating': 'کارت‌هایی از دست‌هایی که دادید و شاید هنوز در گردش باشند',
  'hg.seen.circulating.none': 'هیچ‌یک از کارت‌هایی که دادید دیگر در گردش نیست.',
  'hg.seen.bounds.exact': '{count} به یقین',
  'hg.seen.bounds.atMost': 'حداکثر {count}',
  'hg.seen.bounds.range': '{certain} به یقین، حداکثر {atMost}',
  'hg.seen.colour.tip': '{colour}: {bounds}. برای دیدن این کارت‌ها بزنید.',
  'hg.seen.picks': 'کارت‌هایی که همسایه‌هایتان از دست‌های شما برداشتند',
  'hg.seen.pick': '{name} {card} را برداشت (دوران {age}، نوبت {turn})',
  'hg.seen.pick.tip': '{name} {card} را برداشت. برای دیدن جایی که آن را دیدید بزنید.',
  'hg.seen.returning': 'کارت‌هایی که هنوز ممکن است در این دوران به شما برگردند',
  'hg.seen.returning.tip': '{card}: {bounds}. برای دیدن جایی که آن را دیدید بزنید.',
  'hg.seen.filter': 'نمایش: {what}',
  'hg.seen.filter.clear': 'نمایش همه',
  'hg.seen.card.tip': '{card}: {effect}'
} as const satisfies Translation<typeof en>;
