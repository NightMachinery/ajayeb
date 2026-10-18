// Persian messages for the guided first game: every key of the English
// table, so a lesson never falls back to English inside a right-to-left
// paragraph. Complete, none left in English. Card, wonder and resource
// names arrive as {card}, {wonder} and {resource} from their own tables;
// {name} is the bot's persona (شاگرد). Fixed rule numbers are words, as
// everywhere in the Persian tables. See ./index.ts.

import type { tutorial as en } from '../en/tutorial';
import type { Translation } from '../types';

export const tutorial = {
  // The coach mark's and the coach note's chrome.
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

  // The rails: why a sheet's move buttons are locked, by the guided move.
  'tutorial.locked': 'آموزش: این درس {card} را می‌سازد.',
  'tutorial.lockedDiscard': 'آموزش: این درس {card} را کنار می‌گذارد.',
  'tutorial.lockedBuildWonder': 'آموزش: این درس {wonder} را با {card} می‌سازد.',
  'tutorial.lockedWonder': 'آموزش: این درس {wonder} را برمی‌دارد.',

  // Pause and resume.
  'tutorial.paused.toast': 'آموزش متوقف شد. از منو ادامه‌اش دهید.',
  'tutorial.menu.pause': 'توقف آموزش',
  'tutorial.menu.resume': 'ادامه‌ی آموزش',

  // The info lessons (Next ends them).
  'tutorial.lesson.draftIntro':
    'اول، شما و {name} شگفتی‌ها را برمی‌دارید: در هر دور چهار شگفتی پیشنهاد می‌شود و به نوبت انتخاب می‌کنید.',
  'tutorial.lesson.draftRound2':
    'آخرین شگفتی دور یکم به شما رسید. در دور دوم، {name} اول برمی‌دارد و شما دو شگفتی بعدی را.',
  'tutorial.lesson.ageOne':
    'این ساختار کارت‌های دوران یکم است. کارت‌های روشن را می‌توانید بردارید: کارت‌هایی که رو هستند و کارت دیگری رویشان نیست.',
  'tutorial.lesson.turn':
    'هر نوبت یک کارت برمی‌دارید: آن را می‌سازید، برای سکه کنار می‌گذارید، یا زیر یک شگفتی می‌گذارید تا آن شگفتی ساخته شود.',
  'tutorial.lesson.reveal':
    'برداشتن یک کارت، کارت‌هایی را که می‌پوشاند آزاد می‌کند. {card} همین حالا آزاد شد و رو شد.',
  'tutorial.lesson.military':
    'سپر شما مهره را یک خانه به سوی پایتخت {name} برد. به پایتخت برسید و بی‌درنگ برنده می‌شوید.',
  'tutorial.lesson.coins':
    'با سکه بهای کارت‌ها و خرید از بانک را می‌پردازید. در پایان بازی، هر سه سکه یک امتیاز می‌ارزد.',
  'tutorial.lesson.wonders':
    'این‌ها شگفتی‌های شما هستند. یکی را با پرداخت هزینه‌اش و گذاشتن کارتی از ساختار زیرش می‌سازید.',
  'tutorial.lesson.tokens':
    'این‌ها نشان‌های پیشرفت‌اند. هر بار که جفتی از یک نماد علمی یکسان داشته باشید، یکی برمی‌دارید.',
  'tutorial.lesson.playOn':
    'این دل بازی است. دوران یکم را خودتان تمام کنید: چیزهای تازه هنگام پیش آمدن توضیح داده می‌شوند.',
  'tutorial.lesson.leftPath':
    'راه خودتان را رفتید، پس درس‌های راهنما همین‌جا تمام می‌شوند. بازی را ادامه دهید، یا با «برگرداندن» به درس برگردید.',

  // The action lessons (the guided move ends them).
  'tutorial.action.draftPick':
    '{wonder} را بردارید. شگفتی‌ها را بعداً با پرداخت هزینه‌شان می‌سازید.',
  'tutorial.action.draftPickAgain': 'حالا {wonder} را هم بردارید.',
  'tutorial.action.buildFree':
    '{card} هزینه‌ای ندارد. آن را بسازید: از این پس هر نوبت برایتان {resource} تولید می‌کند.',
  'tutorial.action.buildOwn':
    'هزینه‌ی {card} {resource} است که خودتان تولید می‌کنید، پس سکه‌ای نمی‌خواهد. سطر هزینه این را نشان می‌دهد.',
  'tutorial.action.trade':
    '{resource} تولید نمی‌کنید، پس آن را از بانک به بهای دو سکه، به‌اضافه‌ی یک سکه برای هر {resource} که {name} تولید می‌کند، بخرید. پیش‌نمایش امتیاز کل، امتیاز سکه‌ها را هم حساب می‌کند، پس خرج کردن سکه می‌تواند آن را کم کند.',
  'tutorial.action.discard':
    '{card} را برای سکه کنار بگذارید: دو سکه، به‌اضافه‌ی یک سکه برای هر کارت زردی که دارید. کارت به کارت‌های کنارگذاشته می‌رود.',
  'tutorial.action.buildWonder':
    '{wonder} را با {card} بسازید؛ اثر خود کارت از دست می‌رود. این شگفتی در هر نوبت، به انتخاب شما چوب، خاک رس یا سنگ تولید می‌کند تا هزینه‌ی ساخت‌های بعدی را آسان‌تر بپردازید.',
  'tutorial.action.chain':
    'روی {card} یک نماد زنجیره‌ای کشیده شده است. در دورانی بعدتر، کارتی که همین نماد را دارد برای شما رایگان است.',
  'tutorial.action.science':
    'روی {card} یک نماد علمی هست. جفتی از یک نماد یک نشان پیشرفت به شما می‌دهد، و شش نماد علمی متفاوت بی‌درنگ شما را برنده می‌کند.',

  // Home's first-launch offer.
  'tutorial.offer.title': 'تازه با بازی آشنا می‌شوید؟',
  'tutorial.offer.text': 'با یک بازی راهنمایی‌شده در برابر {name} بازی را یاد بگیرید.',
  'tutorial.offer.start': 'بازی آموزشی',
  'tutorial.offer.decline': 'نه، ممنون',

  // The completion card (Age II in play).
  'tutorial.done.title': 'آموزش تمام شد',
  'tutorial.done.text': 'دوران دوم آغاز می‌شود و باقی بازی با شماست. سه راه برای بردن هست:',
  'tutorial.done.military': 'نظامی: مهره را به پایتخت حریف برسانید.',
  'tutorial.done.science': 'علمی: شش نماد علمی متفاوت گرد آورید.',
  'tutorial.done.civilian': 'مدنی: پس از دوران سوم بیشترین امتیاز را داشته باشید.',
  'tutorial.done.keep': 'ادامه‌ی بازی',
  'tutorial.done.new': 'بازی تازه',
  'tutorial.done.home': 'خانه',

  // The tutorial on other screens.
  'home.mode.tutorial': 'آموزش',
  'home.tutorial': 'یادگیری بازی',
  'home.tutorialHint': 'نخستین بازی راهنمایی‌شده در برابر {name}',
  'home.tutorialDone': 'آموزش تمام شد: دوباره بازی کنید',
  'home.ai.tutorialLink': 'اولین بازی شماست؟ به جای آن، بازی آموزشی را انجام دهید.',
  'hints.settings.tutorialReset': 'نمایش دوباره‌ی پیشنهاد آموزش',
  'hints.settings.tutorialResetDone': 'خانه دوباره آموزش را پیشنهاد می‌دهد.'
} as const satisfies Translation<typeof en>;
