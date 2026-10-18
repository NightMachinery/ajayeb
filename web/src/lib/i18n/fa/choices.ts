// Persian messages for the pending-choice panels (destroy, build from the
// discard pile, progress tokens, the Great Library, the next age's starter),
// the end-of-age screen, the wonder draft, the game-over screen and the
// score table. The table is complete: every key of the English one. See
// ./index.ts.

import type { choices as en } from '../en/choices';
import type { Translation } from '../types';

export const choices = {
  // Shared by every choice panel.
  'choices.peek': 'نگاهی به صفحه‌ی بازی',
  'choices.reopen': 'باز کردن',
  'choices.selectCard': 'اول یک کارت برگزینید',
  'choices.selectToken': 'اول یک نشان برگزینید',
  'choices.waiting.destroyCard': 'منتظر بمانید تا {name} کارتی را برای نابودی برگزیند…',
  'choices.waiting.buildFromDiscard':
    'منتظر بمانید تا {name} کارتی از کارت‌های کنارگذاشته برگزیند…',
  'choices.waiting.pickBoardToken': 'منتظر بمانید تا {name} یک نشان پیشرفت برگزیند…',
  'choices.waiting.pickLibraryToken': {
    other: '{name} از میان {count} نشان پنهان کتابخانه‌ی بزرگ انتخاب می‌کند…'
  },
  'choices.waiting.chooseStarter': 'منتظر بمانید تا {name} آغازکننده‌ی دوران {age} را برگزیند…',

  // Destroy (the Statue of Zeus, Circus Maximus).
  'choices.destroy.title': 'نابود کردن یک کارت',
  'choices.destroy.intro':
    'یکی از کارت‌های {color} شهر {name} را برگزینید. به کارت‌های کنارگذاشته می‌رود.',
  'choices.destroy.confirm': 'نابود کردن {card}',

  // Build from the discard pile (the Mausoleum) and the discard pile sheet.
  'choices.discard.title': 'ساختن از کارت‌های کنارگذاشته',
  'choices.discard.intro': 'یک کارت کنارگذاشته را برگزینید تا رایگان ساخته شود.',
  'choices.discard.confirm': 'ساخت رایگان {card}',
  'choices.discard.sheetTitle': 'کارت‌های کنارگذاشته',
  'choices.discard.order': 'قدیمی‌ترین‌ها اول.',
  'choices.discard.filter': 'نمایش کارت‌ها بر پایه‌ی رنگ',
  'choices.discard.all': 'همه',
  'choices.discard.count': {
    '=0': 'کارتی نیست',
    other: '{count} کارت'
  },
  'choices.discard.empty': 'هیچ کارت کنارگذاشته‌ای نیست.',

  // Progress tokens: a science pair, and the Great Library.
  'choices.token.title': 'انتخاب نشان پیشرفت',
  'choices.token.intro': 'هر جفت نماد علمی یکسان یک نشان پیشرفت از صفحه‌ی بازی به شما می‌دهد.',
  'choices.token.confirm': 'برداشتن {token}',
  'choices.token.list': 'نشان‌های پیشرفت برای انتخاب',
  'choices.library.title': 'کتابخانه‌ی بزرگ',
  'choices.library.intro':
    'سه نشان از جعبه بیرون کشیده شد. یکی را نگه دارید؛ دو تای دیگر نادیده به جعبه برمی‌گردند.',
  'choices.library.secret': 'تنها شما این نشان‌ها را می‌بینید.',

  // Who starts the next age.
  'choices.starter.title': 'چه کسی دوران {age} را آغاز می‌کند؟',
  'choices.starter.whyBehind': '{name} در مسیر نظامی عقب است، پس {name} انتخاب می‌کند.',
  'choices.starter.whyBehindYou': 'شما در مسیر نظامی عقب هستید، پس شما انتخاب می‌کنید.',
  'choices.starter.whyEven':
    'مسیر نظامی برابر است، پس {name} که آخرین حرکت را انجام داده، انتخاب می‌کند.',
  'choices.starter.whyEvenYou':
    'مسیر نظامی برابر است، پس شما که آخرین حرکت را انجام داده‌اید، انتخاب می‌کنید.',
  'choices.starter.whyOther': '{name} آغازکننده را انتخاب می‌کند.',
  'choices.starter.whyOtherYou': 'شما آغازکننده را انتخاب می‌کنید.',
  'choices.starter.hint': 'آغازکننده نخستین کارت دوران تازه را برمی‌دارد.',
  'choices.starter.option': '{name} آغاز کند',
  'choices.starter.optionYou': 'من آغاز می‌کنم',

  // The end-of-age interstitial.
  'choices.age.complete': 'دوران {age} به پایان رسید',
  'choices.age.summary': 'هر دو بازیکن پس از دوران {age}',
  'choices.age.coins': 'سکه‌ها',
  'choices.age.points': 'امتیازها',
  'choices.age.you': '(شما)',
  'choices.age.militaryLead': '{name} در مسیر نظامی {margin} خانه جلوتر است.',
  'choices.age.militaryLeadYou': 'شما در مسیر نظامی {margin} خانه جلوتر هستید.',
  'choices.age.militaryEven': 'مسیر نظامی برابر است.',
  'choices.age.starts': '{name} دوران {age} را آغاز می‌کند.',
  'choices.age.startsYou': 'شما دوران {age} را آغاز می‌کنید.',
  'choices.age.continue': 'ادامه',
  'choices.age.begin': 'آغاز دوران {age}',
  'choices.age.summaryOpen': 'خلاصه',

  // The wonder draft.
  'choices.draft.title': 'گزینش شگفتی‌ها',
  'choices.draft.round': 'دور {round} از دو',
  'choices.draft.order': 'ترتیب انتخاب',
  'choices.draft.pick': 'انتخاب {n}: {name}',
  'choices.draft.pickDone': 'انتخاب {n}: {name}، انجام شد',
  'choices.draft.pickNow': 'انتخاب {n}: {name}، اکنون',
  'choices.draft.pickAuto': 'انتخاب {n}: آخرین شگفتی به {name} می‌رسد',
  'choices.draft.yourPick': 'نوبت انتخاب شما: یک شگفتی برگزینید',
  'choices.draft.namePick': 'نوبت انتخاب {name}: یک شگفتی برگزینید',
  'choices.draft.waiting': 'در انتظار انتخاب {name}…',
  'choices.draft.offer': 'شگفتی‌های پیش رو',
  'choices.draft.take': 'برداشتن {wonder}',
  'choices.draft.cost': 'هزینه: {cost}',
  'choices.draft.picked': 'شگفتی‌های {name}',
  'choices.draft.emptySlot': 'هنوز انتخاب نشده',

  // The game-over screen.
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
  'choices.over.reason.science': '{name} شش نماد علمی متفاوت گرد آورد.',
  'choices.over.reason.shared': 'امتیازها برابر، و امتیاز کارت‌های آبی هم برابر.',
  'choices.over.reason.forfeit': '{name} بازی را واگذار کرد.',
  'choices.over.tieBreak': 'امتیازها برابر بود، پس بیشترین امتیاز کارت‌های آبی برنده است.',
  'choices.over.rematch': 'بازی دوباره',
  'choices.over.viewBoard': 'دیدن صفحه‌ی پایانی',
  'choices.over.home': 'خانه',
  'choices.over.results': 'نتیجه‌ها',
  'choices.over.bar': 'پایان بازی',

  // The score table.
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
  'choices.score.winner': 'برنده',

  // The /dev/gallery section for this area.
  'dev.gallery.choices': 'گزینش، انتخاب‌ها و پایان بازی'
} as const satisfies Translation<typeof en>;
