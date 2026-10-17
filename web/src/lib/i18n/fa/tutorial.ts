// Persian messages for the guided first game: every key of the English
// table, so a lesson never falls back to English inside a right-to-left
// paragraph. Card, wonder and resource names come from their own tables
// (still English in Persian for now). See ./index.ts.

import type { tutorial as en } from '../en/tutorial';
import type { Translation } from '../types';

export const tutorial = {
  'tutorial.coach.title': 'آموزش',
  'tutorial.coach.progress': 'درس {n} از {total}',
  'tutorial.coach.next': 'بعدی',
  'tutorial.coach.pause': 'خودم بازی می‌کنم',
  'tutorial.coach.region': 'گام آموزش',
  'tutorial.coach.tapCard': 'روی {card} بزنید.',
  'tutorial.coach.openWonders': '«ساخت یک شگفتی» را انتخاب کنید.',
  'tutorial.coach.takeWonder': 'روی {wonder} بزنید، سپس «برداشتن».',
  'tutorial.coach.otherCard': 'این درس {card} را برمی‌دارد. این را ببندید و روی آن بزنید.',
  'tutorial.coach.otherWonder': 'این درس {wonder} را برمی‌دارد. این را ببندید و روی آن بزنید.',
  'tutorial.coach.back': 'این درس خود {card} را می‌سازد. به کارت برگردید.',

  'tutorial.locked': 'آموزش: این درس {card} را می‌سازد.',
  'tutorial.lockedDiscard': 'آموزش: این درس {card} را کنار می‌گذارد.',
  'tutorial.lockedBuildWonder': 'آموزش: این درس {wonder} را با {card} می‌سازد.',
  'tutorial.lockedWonder': 'آموزش: این درس {wonder} را برمی‌دارد.',

  'tutorial.paused.toast': 'آموزش متوقف شد. از منو ادامه‌اش دهید.',
  'tutorial.menu.pause': 'توقف آموزش',
  'tutorial.menu.resume': 'ادامه‌ی آموزش',

  'tutorial.lesson.draftIntro':
    'اول، شما و {name} شگفتی‌ها را برمی‌دارید: در هر دور چهار شگفتی هست و به نوبت انتخاب می‌کنید.',
  'tutorial.lesson.draftRound2':
    'آخرین شگفتی دور ۱ به شما رسید. در دور ۲، {name} اول برمی‌دارد و شما دو تای بعدی را.',
  'tutorial.lesson.ageOne':
    'این ساختار کارت‌های دوران ۱ است. کارتی را می‌توانید بردارید که رو باشد و کارت دیگری رویش نباشد.',
  'tutorial.lesson.turn':
    'هر نوبت یک کارت برمی‌دارید: آن را می‌سازید، برای سکه کنار می‌گذارید، یا زیر یک شگفتی می‌گذارید تا آن شگفتی ساخته شود.',
  'tutorial.lesson.reveal':
    'برداشتن یک کارت، کارت‌های زیرش را آزاد می‌کند. {card} همین حالا آزاد شد و رو شد.',
  'tutorial.lesson.military':
    'سپر شما مهره را یک خانه به سوی پایتخت {name} برد. به پایتخت برسید و همان دم برنده می‌شوید.',
  'tutorial.lesson.coins':
    'سکه بهای کارت‌ها و خرید از بانک را می‌پردازد. در پایان بازی، هر ۳ سکه ۱ امتیاز است.',
  'tutorial.lesson.wonders':
    'این‌ها شگفتی‌های شما هستند. یکی را با پرداخت هزینه‌اش و گذاشتن کارتی از ساختار زیرش می‌سازید.',
  'tutorial.lesson.tokens':
    'این‌ها نشان‌های پیشرفت‌اند. هر بار که دو نماد علمی یکسان داشته باشید، یکی برمی‌دارید.',
  'tutorial.lesson.playOn':
    'این دل بازی است. دوران ۱ را خودتان تمام کنید: چیزهای تازه هنگام پیش آمدن توضیح داده می‌شوند.',
  'tutorial.lesson.leftPath':
    'راه خودتان را رفتید، پس درس‌های راهنما همین‌جا تمام می‌شوند. بازی را ادامه دهید، یا با «برگرداندن» به درس برگردید.',

  'tutorial.action.draftPick':
    '{wonder} را بردارید. شگفتی‌ها را بعداً با پرداخت هزینه‌شان می‌سازید.',
  'tutorial.action.draftPickAgain': 'حالا {wonder} را هم بردارید.',
  'tutorial.action.buildFree':
    '{card} هزینه‌ای ندارد. آن را بسازید: از این پس هر نوبت برایتان {resource} تولید می‌کند.',
  'tutorial.action.buildOwn':
    'هزینه‌ی {card} {resource} است که خودتان تولید می‌کنید، پس سکه‌ای نمی‌خواهد. سطر هزینه این را نشان می‌دهد.',
  'tutorial.action.trade':
    '{resource} تولید نمی‌کنید، پس آن را از بانک می‌خرید. سطر هزینه بها را نشان می‌دهد: ۲ سکه، به‌اضافه‌ی ۱ برای هر {resource} که {name} تولید می‌کند.',
  'tutorial.action.discard':
    '{card} را برای سکه کنار بگذارید: ۲ سکه، به‌اضافه‌ی ۱ برای هر کارت زرد شما. کارت به کارت‌های کنارگذاشته می‌رود.',
  'tutorial.action.buildWonder':
    '{wonder} را با {card} بسازید. کارت پشت‌ورو زیر شگفتی می‌رود و اثر خودش از دست می‌رود.',
  'tutorial.action.chain':
    'روی {card} نماد زنجیره‌ای کشیده شده است. در دورانی بعدتر، کارتی با همین نماد برای شما رایگان است.',
  'tutorial.action.science':
    'روی {card} یک نماد علمی هست. دو نماد یکسان یک نشان پیشرفت می‌دهد و شش نماد متفاوت همان دم برنده می‌شود.',

  'tutorial.offer.title': 'تازه با بازی آشنا می‌شوید؟',
  'tutorial.offer.text': 'با یک بازی راهنمایی‌شده در برابر {name} یاد بگیرید.',
  'tutorial.offer.start': 'بازی آموزشی',
  'tutorial.offer.decline': 'نه، ممنون',

  'tutorial.done.title': 'آموزش تمام شد',
  'tutorial.done.text': 'دوران ۲ آغاز می‌شود و باقی بازی با شماست. سه راه برای بردن هست:',
  'tutorial.done.military': 'نظامی: مهره را به پایتخت حریف برسانید.',
  'tutorial.done.science': 'علمی: شش نماد علمی متفاوت گرد آورید.',
  'tutorial.done.civilian': 'مدنی: پس از دوران ۳ بیشترین امتیاز را داشته باشید.',
  'tutorial.done.keep': 'ادامه‌ی بازی',
  'tutorial.done.new': 'بازی تازه',
  'tutorial.done.home': 'خانه',

  'home.mode.tutorial': 'آموزش',
  'home.tutorial': 'یادگیری بازی',
  'home.tutorialHint': 'نخستین بازی راهنمایی‌شده در برابر {name}',
  'home.tutorialDone': 'آموزش تمام شد: دوباره بازی کنید',
  'home.ai.tutorialLink': 'بازی اول است؟ به جایش بازی آموزشی را انجام دهید.',
  'hints.settings.tutorialReset': 'پیشنهاد آموزش را دوباره نشان بده',
  'hints.settings.tutorialResetDone': 'خانه دوباره آموزش را پیشنهاد می‌دهد.'
} as const satisfies Translation<typeof en>;
