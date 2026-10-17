// Persian messages for the guided first game: the coach chrome, the rails'
// reasons, pause and resume, Home's offer and the completion card's
// buttons. Partial: the lesson texts fall back to English. See ./index.ts.

import type { tutorial as en } from '../en/tutorial';
import type { Translation } from '../types';

export const tutorial = {
  'tutorial.coach.title': 'آموزش',
  'tutorial.coach.progress': 'درس {n} از {total}',
  'tutorial.coach.next': 'بعدی',
  'tutorial.coach.pause': 'خودم بازی می‌کنم',
  'tutorial.coach.region': 'گام آموزش',
  'tutorial.coach.tapCard': 'روی {card} بزنید.',
  'tutorial.coach.openWonders': '«ساختن شگفتی» را انتخاب کنید.',
  'tutorial.coach.takeWonder': 'روی {wonder} بزنید، سپس «برداشتن».',
  'tutorial.coach.otherCard': 'این درس {card} را برمی‌دارد. این را ببندید و روی آن بزنید.',
  'tutorial.coach.otherWonder': 'این درس {wonder} را برمی‌دارد. این را ببندید و روی آن بزنید.',
  'tutorial.locked': 'آموزش: این درس {card} را می‌سازد.',
  'tutorial.lockedDiscard': 'آموزش: این درس {card} را دور می‌اندازد.',
  'tutorial.lockedBuildWonder': 'آموزش: این درس {wonder} را با {card} می‌سازد.',
  'tutorial.lockedWonder': 'آموزش: این درس {wonder} را برمی‌دارد.',
  'tutorial.paused.toast': 'آموزش متوقف شد. از منو ادامه‌اش دهید.',
  'tutorial.menu.pause': 'توقف آموزش',
  'tutorial.menu.resume': 'ادامه‌ی آموزش',
  'tutorial.offer.title': 'تازه با بازی آشنا می‌شوید؟',
  'tutorial.offer.text': 'با یک بازی راهنمایی‌شده در برابر {name} یاد بگیرید.',
  'tutorial.offer.start': 'بازی آموزشی',
  'tutorial.offer.decline': 'نه، ممنون',
  'tutorial.done.title': 'آموزش تمام شد',
  'tutorial.done.keep': 'ادامه‌ی بازی',
  'tutorial.done.new': 'بازی تازه',
  'tutorial.done.home': 'خانه'
} as const satisfies Translation<typeof en>;
