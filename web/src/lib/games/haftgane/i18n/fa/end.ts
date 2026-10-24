// Persian messages for the end of an Age and of the game (../en/end.ts): the
// military dialog, the score sheet and the revealed cards. See ./index.ts.

import type { Translation } from '$lib/i18n/types';
import type { end as en } from '../en/end';

export const end = {
  'hg.end.you': 'شما',

  'hg.end.military.title': 'نبردهای دوران {age}',
  'hg.end.military.ages': 'نبردهای هر دوران',
  'hg.end.military.yours': 'نبردهای شما',
  'hg.end.military.left': 'همسایه‌ی چپ',
  'hg.end.military.right': 'همسایه‌ی راست',
  'hg.end.military.shields': 'سپرها: {you} در برابر {them}',
  'hg.end.military.all': 'همه‌ی نبردهای دور میز',
  'hg.end.military.points': 'امتیاز نظامی تا اینجا',
  'hg.end.military.pointsTip': '{name}: {points} امتیاز نظامی تا اینجا',
  'hg.end.military.yourPoints': 'امتیاز نظامی شما تا اینجا: {points}',
  'hg.end.military.close': 'ادامه',
  'hg.end.result.victory': 'پیروزی',
  'hg.end.result.defeat': 'شکست',
  'hg.end.result.tie': 'برابر',
  'hg.end.token.victory': {
    other: '{name} نشان پیروزی دوران {age} را می‌گیرد، به ارزش {count} امتیاز.'
  },
  'hg.end.token.defeat': '{name} نشان شکست می‌گیرد.',
  'hg.end.token.tie': 'سپرهای {a} و {b} برابر است: نشانی در کار نیست.',
  'hg.end.token.youWin': {
    other: 'بر {name} پیروز می‌شوید و نشان پیروزی دوران {age} را می‌گیرید، به ارزش {count} امتیاز.'
  },
  'hg.end.token.youLose': '{name} بر شما پیروز می‌شود: نشان شکست می‌گیرید.',
  'hg.end.token.youTie': 'سپرهای شما و {name} برابر است: نشانی در کار نیست.',
  'hg.end.pair.shields': 'سپرها در پایان دوران {age}: {a} {aCount}، {b} {bCount}.',

  'hg.end.verdict.you': 'شما بردید',
  'hg.end.verdict.one': '{name} برد',
  'hg.end.verdict.sharedYou': 'پیروزی میان شما و {names} مشترک است',
  'hg.end.verdict.shared': 'پیروزی میان {names} مشترک است',
  'hg.end.verdict.none': 'پایان بازی',
  'hg.end.tie.coins':
    '{names} هر کدام {total} امتیاز گرفتند. بیشترین سکه برابری را می‌شکند: {counts}.',
  'hg.end.tie.count': '{name} {coins}',
  'hg.end.tie.separator': '، ',
  'hg.end.tie.shared':
    '{names} هر کدام {total} امتیاز و {coins} سکه دارند، پس پیروزی میانشان مشترک است.',
  'hg.end.score.player': 'بازیکن',
  'hg.end.score.col.wonder': 'شگفتی',
  'hg.end.score.col.treasure': 'سکه‌ها: هر سه سکه یک امتیاز',
  'hg.end.score.col.military': 'نظامی',
  'hg.end.score.col.blue': 'ساختمان‌های مدنی',
  'hg.end.score.col.yellow': 'ساختمان‌های تجاری',
  'hg.end.score.col.green': 'علم',
  'hg.end.score.col.purple': 'انجمن‌ها',
  'hg.end.score.col.total': 'جمع',
  'hg.end.score.rank': 'رتبه‌ی {rank}',
  'hg.end.score.totalTip': '{name}: {total} امتیاز، {coins} سکه در پایان',
  'hg.end.score.city': 'باز کردن شهر {name}',
  'hg.end.download': 'دریافت گزارش',
  'hg.end.download.hidden':
    'این میز پایان بازی را پنهان نگه می‌دارد، پس گزارشش را نمی‌توان دریافت کرد.',
  'hg.end.board': 'دیدن میز پایانی',
  'hg.end.home': 'خانه',

  'hg.end.open.scores': 'امتیازها',
  'hg.end.open.military': 'نبردها',
  'hg.end.open.reveal': 'کارت‌های رو شده',

  'hg.end.reveal.title': 'رو شده در پایان',
  'hg.end.reveal.tucked': 'کارت‌های زیر مرحله‌های شگفتی',
  'hg.end.reveal.leftovers': 'کارت‌های آخر دوران {age}',
  'hg.end.reveal.kept': 'نگه‌داشته برای نوبت آخر',
  'hg.end.reveal.pile': { other: 'کارت‌های کنارگذاشته: {count} کارت' },
  'hg.end.reveal.box': 'انجمن‌هایی که در جعبه ماندند',
  'hg.end.reveal.none': 'هیچ',
  'hg.end.reveal.close': 'بستن'
} as const satisfies Translation<typeof en>;
