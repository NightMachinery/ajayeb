// Persian messages for the wonder draft, the end-of-age screen, the
// starter choice and the game-over screen. Partial; the other choice panels
// fall back to English. See ./index.ts.

import type { choices as en } from '../en/choices';
import type { Translation } from '../types';

export const choices = {
  'choices.starter.hint': 'آغازکننده نخستین کارت دوران تازه را برمی‌دارد.',
  'choices.starter.option': '{name} آغاز کند',
  'choices.starter.optionYou': 'من آغاز می‌کنم',

  'choices.age.complete': 'دوران {age} به پایان رسید',
  'choices.age.summary': 'هر دو بازیکن پس از دوران {age}',
  'choices.age.coins': 'سکه‌ها',
  'choices.age.points': 'امتیازها',
  'choices.age.you': '(شما)',
  'choices.age.militaryEven': 'مسیر نظامی برابر است.',
  'choices.age.starts': '{name} دوران {age} را آغاز می‌کند.',
  'choices.age.startsYou': 'شما دوران {age} را آغاز می‌کنید.',
  'choices.age.begin': 'آغاز دوران {age}',
  'choices.age.summaryOpen': 'خلاصه',

  'choices.draft.title': 'گزینش شگفتی‌ها',
  'choices.draft.round': 'دور {round} از ۲',
  'choices.draft.order': 'ترتیب انتخاب',
  'choices.draft.yourPick': 'نوبت انتخاب شما: یک شگفتی برگزینید',
  'choices.draft.namePick': 'نوبت انتخاب {name}: یک شگفتی برگزینید',
  'choices.draft.waiting': 'در انتظار انتخاب {name}…',
  'choices.draft.offer': 'شگفتی‌های پیش رو',
  'choices.draft.take': 'برداشتن {wonder}',
  'choices.draft.cost': 'هزینه: {cost}',
  'choices.draft.picked': 'شگفتی‌های {name}',
  'choices.draft.emptySlot': 'هنوز انتخاب نشده',

  'choices.over.kind.civilian': 'پیروزی مدنی',
  'choices.over.kind.military': 'برتری نظامی',
  'choices.over.kind.science': 'برتری علمی',
  'choices.over.kind.shared': 'پیروزی مشترک',
  'choices.over.kind.forfeit': 'پیروزی با واگذاری',
  'choices.over.wins': '{name} برنده شد',
  'choices.over.youWin': 'شما برنده شدید',
  'choices.over.bothWin': 'هر دو بازیکن برنده شدند',
  'choices.over.reason.civilian': 'بیشترین امتیاز پس از دوران سوم.',
  'choices.over.reason.military': 'سپاه {name} به پایتخت حریف رسید.',
  'choices.over.reason.science': '{name} شش نماد علمی گوناگون گرد آورد.',
  'choices.over.reason.shared': 'امتیازها برابر، و امتیاز کارت‌های آبی هم برابر.',
  'choices.over.reason.forfeit': '{name} بازی را واگذار کرد.',
  'choices.over.tieBreak': 'امتیازها برابر بود، پس بیشترین امتیاز کارت‌های آبی برنده است.',
  'choices.over.rematch': 'بازی دوباره',
  'choices.over.viewBoard': 'دیدن صفحه‌ی پایانی',
  'choices.over.home': 'خانه',
  'choices.over.results': 'نتیجه‌ها',
  'choices.over.bar': 'پایان بازی',

  'choices.score.caption': 'امتیاز نهایی',
  'choices.score.category': 'دسته',
  'choices.score.blue': 'کارت‌های آبی',
  'choices.score.green': 'کارت‌های سبز',
  'choices.score.yellow': 'کارت‌های زرد',
  'choices.score.purple': 'انجمن‌ها',
  'choices.score.wonders': 'شگفتی‌ها',
  'choices.score.tokens': 'نشان‌های پیشرفت',
  'choices.score.coins': 'سکه‌ها',
  'choices.score.military': 'نظامی',
  'choices.score.total': 'مجموع',
  'choices.score.winner': 'برنده'
} as const satisfies Translation<typeof en>;
