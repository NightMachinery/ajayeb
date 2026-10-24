// Persian messages for the in-room game screen: the card sheet, the cost
// breakdown, the wonder step, the action reasons, the game menu, the forfeit
// step, the opponent move summary and the /dev/gallery game section. The
// table is complete: every key of the English one. See ./index.ts.

import type { game as en } from '../en/game';
import type { Translation } from '../types';

export const game = {
  'game.title': 'بازی {code}',
  'game.loading': 'در انتظار بازی…',
  'game.board': 'صفحه‌ی بازی',
  'game.openCity': 'نمایش شهر {name}',

  // The card sheet.
  'game.card.meta': '{category} · دوران {age}',
  'game.card.covered': {
    other: 'زیر {count} کارت پوشیده است. وقتی هیچ کارتی رویش نماند، می‌توان برش داشت.'
  },
  'game.card.faceDown': 'پشت‌ورو است. وقتی دیگر هیچ کارتی رویش نباشد، رو می‌شود.',
  'game.card.notYourTurn': 'نوبت شما نیست: می‌توانید نگاه کنید، ولی نمی‌توانید بازی کنید.',
  'game.card.locked': 'صبر کنید تا حرکت شما پاسخ بگیرد.',
  'game.card.nothing': 'اکنون با این کارت کاری نمی‌توان کرد.',
  'game.card.opponentPays': {
    '=0': 'برای {name} رایگان است.',
    other: 'برای {name} {count} سکه هزینه دارد.'
  },
  'game.card.opponentShort': {
    other: '{name} توان پرداختش را ندارد ({count} سکه).'
  },
  // A covered card: what each player would pay once it is uncovered.
  'game.card.coveredCostTitle': 'اگر همین حالا آزاد بود',
  'game.card.youPay': {
    '=0': 'برای شما رایگان است.',
    other: 'برای شما {count} سکه هزینه دارد.'
  },
  'game.card.youShort': {
    other: 'توان پرداختش را ندارید ({count} سکه).'
  },

  // Action buttons.
  'game.action.build': { '=0': 'ساخت رایگان', other: 'ساخت با {count} سکه' },
  'game.action.discard': { other: 'کنار گذاشتن برای {count} سکه' },
  'game.action.wonder': 'ساخت یک شگفتی',
  'game.action.wonderReadyOne': 'آماده: {wonder}',
  'game.action.wonderReadyMany': { other: 'آماده: {count} شگفتی' },
  'game.action.buildWonder': {
    '=0': 'ساخت رایگان {wonder}',
    other: 'ساخت {wonder} با {count} سکه'
  },

  // Why an action is not available (the server's Reason codes).
  'game.reason.gameOver': 'بازی تمام شده است.',
  'game.reason.wrongPhase': 'بازی منتظر نوع دیگری از حرکت است.',
  'game.reason.malformed': 'این حرکت ممکن نیست.',
  'game.reason.notInDraft': 'این شگفتی در میان گزینه‌ها نیست.',
  'game.reason.notAccessible': 'اکنون نمی‌توان این کارت را برداشت.',
  'game.reason.wonderUnavailable': 'این شگفتی را نمی‌توان ساخت.',
  'game.reason.unaffordable': {
    other: '{count} سکه کم دارید.'
  },
  'game.reason.unaffordableNoCount': 'توان پرداختش را ندارید.',
  'game.reason.notTarget': 'این کارت را نمی‌توان انتخاب کرد.',
  'game.reason.notInDiscard': 'این کارت در میان کارت‌های کنارگذاشته نیست.',
  'game.reason.tokenUnavailable': 'این نشان در میان گزینه‌ها نیست.',

  // The cost breakdown.
  'game.cost.title': 'هزینه',
  'game.cost.own': '{n} {resource}: تولید خودتان',
  'game.cost.producer': 'یک {resource}: از {source}',
  'game.cost.discount': '{n} {resource}: رایگان با {token}',
  'game.cost.trade': '{n} {resource}: خرید از بانک، هر واحد {price} سکه',
  'game.cost.tradeFixed': '{n} {resource}: خرید با قیمت ثابت {price} سکه',
  'game.cost.printed': 'هزینه‌ی سکه‌ای کارت',
  'game.cost.chain': 'رایگان از راه نماد زنجیره‌ای {card}',
  'game.cost.chainNoCard': 'رایگان از راه نماد زنجیره‌ای',
  'game.cost.urbanism': {
    other: '{count} سکه می‌گیرید (شهرسازی)'
  },
  'game.cost.toOpponent': {
    other: '{count} سکه از آن به حریف شما می‌رسد (اقتصاد)'
  },
  'game.cost.nothing': 'پرداختی ندارد',
  'game.cost.total': 'مجموع',
  'game.cost.short': {
    other: 'شما {have} سکه دارید: {count} سکه کم است'
  },

  // The wonder step of the card sheet (card first).
  'game.wonders.title': 'با {card} یک شگفتی بسازید',
  'game.wonders.back': 'بازگشت به کارت',
  'game.wonders.tuckNote': 'کارت پشت‌ورو زیر شگفتی می‌رود و اثر خودش از دست می‌رود.',
  'game.wonders.sevenNote': {
    other:
      '{count} از هفت شگفتی ساخته شده است. وقتی هفتمی ساخته شود، آخرین شگفتی ناساخته از بازی بیرون می‌رود.'
  },
  'game.wonders.none': 'اکنون هیچ‌یک از شگفتی‌های شما را نمی‌توان با این کارت ساخت.',

  // The wonder sheet (wonder first) and the pick mode it starts.
  'game.wonder.build': 'ساخت این شگفتی',
  'game.wonder.pickHint': 'سپس کارتی از ساختار برگزینید تا زیر آن بگذارید.',
  'game.wonder.cannot': 'اکنون نمی‌توان آن را ساخت.',
  'game.pick.prompt': 'کارتی برای {wonder} برگزینید',
  'game.pick.cancel': 'لغو',
  'game.pick.confirm': '{wonder} را با {card} می‌سازید؟',
  'game.pick.effectLost': 'اثرش از دست می‌رود: کارت پشت‌ورو زیر شگفتی می‌رود.',
  'game.pick.notThisCard': '{card} اکنون نمی‌تواند زیر {wonder} برود.',

  // The bot's persona, by level.
  'game.persona.easy': 'شاگرد',
  'game.persona.normal': 'دانشمند',
  'game.persona.hard': 'استاد',

  // The game menu in the top bar.
  'game.sound.turnOff': 'خاموش کردن صدا',
  'game.sound.turnOn': 'روشن کردن صدا',
  'game.menu.open': 'منوی بازی',
  'game.menu.home': 'خانه',
  'game.menu.homeHint': 'می‌توانید این بازی را از خانه ادامه دهید.',
  'game.menu.settings': 'تنظیمات',
  'game.menu.log': 'گزارش',
  'game.menu.replayRunning': 'پس از پایان این بازی در دسترس است.',
  'game.menu.replayHidden': 'بازپخش این بازی پس از پایان هم پنهان می‌ماند.',
  'game.menu.replay': 'دریافت بازپخش',
  'game.menu.forfeit': 'واگذار کردن بازی',
  // The menu's undo item, and why it is greyed out when it is.
  'game.menu.undo': 'برگرداندن آخرین حرکت',
  'game.menu.undoOnline': 'برگرداندن در بازی‌های آنلاین ممکن نیست.',
  'game.menu.undoDisabled': 'برگرداندن در این بازی خاموش است.',
  'game.menu.undoSpectator': 'تماشاگران نمی‌توانند حرکتی را برگردانند.',
  'game.menu.undoNothing': 'هنوز حرکتی از شما برای برگرداندن نیست.',
  'game.menu.undoOver': 'بازی تمام شده است.',
  'game.menu.undoWaiting': 'در انتظار سرور…',
  'game.menu.undoOffline': 'در انتظار اتصال.',

  // The forfeit's confirm step.
  'game.forfeit.title': 'بازی را واگذار می‌کنید؟',
  'game.forfeit.titleName': 'بازی را از طرف {name} واگذار می‌کنید؟',
  'game.forfeit.consequence': '{name} بی‌درنگ برنده می‌شود.',
  'game.forfeit.final': 'واگذاری بازی را برای همیشه تمام می‌کند و برگشتی ندارد.',
  'game.forfeit.confirm': 'واگذاری',
  'game.forfeit.cancel': 'ادامه‌ی بازی',

  // The settings sheet inside a game.
  'game.settings.title': 'تنظیمات',

  // The /dev/gallery game screen section.
  'game.gallery.title': 'نمای بازی',
  'game.gallery.intro':
    'نمای بازی روی وضعیت‌های ضبط‌شده‌ی بازی، هر کدام در قابی به اندازه‌ی گوشی یا رایانه‌ی رومیزی. هر قاب، صفحه‌ی /dev/fixture همان وضعیت است.',
  'game.gallery.frame': '{scene} در {width}×{height}',
  'game.gallery.open': 'باز کردن در اندازه‌ی کامل',
  'game.gallery.scene.draft': 'گزینش شگفتی‌ها',
  'game.gallery.scene.ageOne': 'آغاز دوران یکم',
  'game.gallery.scene.ageTwo': 'میانه‌ی دوران دوم',
  'game.gallery.scene.ageTwoLate': 'اواخر دوران دوم',
  'game.gallery.scene.ageThree': 'دوران سوم',
  'game.gallery.scene.pending': 'یک انتخاب در انتظار',
  'game.gallery.scene.gameOver': 'پایان بازی',
  'game.gallery.scene.lobby': 'اتاق انتظار آنلاین',
  'game.gallery.scene.away': 'حریف غایب',
  'game.gallery.scene.watching': 'تماشاگر',
  'game.gallery.summary': 'خلاصه‌ی حرکت حریف',
  'game.gallery.summaryIntro':
    'پنجره‌ای که نشان می‌دهد طرف دیگر همین حالا چه کرد، روی حرکت‌های ضبط‌شده. یکی را باز کنید، سپس برای بستنش هر جا را بزنید.',
  'game.gallery.summaryOpen': 'نمایش: {scene}',
  'game.gallery.summary.bot': 'پاسخ ربات با یک جفت نماد علمی و یک نشان',
  'game.gallery.summary.extra': 'یک نوبت دوباره با خرید از بانک، مهره و غارت',
  'game.gallery.summary.special':
    'شگفتی‌ای که نابود می‌کند، شگفتی‌ای که از بازی بیرون می‌رود و تغییر دوران',

  // The opponent move summary: one line per move, the player named in the title.
  'game.summary.title': 'حرکت {name}',
  'game.summary.away': 'وقتی نبودید',
  'game.undo.done': 'آخرین حرکت شما برگردانده شد.',
  'game.summary.open': 'نمایش آخرین حرکت حریف',
  'game.summary.close': 'برای بستن هر جا را بزنید',
  'game.summary.build': '{card} را ساخت',
  'game.summary.discard': {
    other: '{card} را برای {count} سکه کنار گذاشت'
  },
  'game.summary.wonder': 'شگفتی {wonder} را با {card} ساخت',
  'game.summary.wonderPlain': 'شگفتی {wonder} را ساخت',
  'game.summary.draft': '{wonder} را در گزینش برداشت',
  'game.summary.draftAuto': '{wonder}، آخرین شگفتی دور، به {name} رسید',
  'game.summary.token': 'نشان {token} را برداشت',
  'game.summary.tokenLibrary': 'نشان {token} را از کتابخانه‌ی بزرگ برداشت',
  'game.summary.destroy': '{card} را در شهر {name} نابود کرد',
  'game.summary.rebuild': '{card} را از کارت‌های کنارگذاشته ساخت',
  'game.summary.starter': '{name} را برای آغاز دوران بعد برگزید',
  'game.summary.resign': 'بازی را واگذار کرد',
  'game.summary.ageStart': 'دوران {age} آغاز می‌شود',
  'game.summary.extraTurn': 'دوباره بازی می‌کند',
  'game.summary.extraTurnLost': 'نوبت دوباره در پایان دوران از دست می‌رود',
  'game.summary.wonderRemoved': 'شگفتی {wonder} از آن {name} از بازی بیرون می‌رود',
  'game.summary.wonderRemovedTip': 'هفت شگفتی ساخته شده است، پس هشتمی از بازی بیرون می‌رود.',
  'game.summary.wins': '{name} برنده شد: {kind}',
  'game.summary.paid': {
    other: '{count} سکه پرداخت'
  },
  'game.summary.trade': {
    other: '{count} سکه از آن بابت خرید به {name} رسید'
  },
  'game.summary.gains': {
    other: '{name} {count} سکه می‌گیرد'
  },
  'game.summary.loses': {
    other: '{name} {count} سکه از دست می‌دهد'
  },
  'game.summary.pawn': {
    other: 'مهره‌ی نظامی {count} خانه به سوی پایتخت {name} می‌رود'
  },
  'game.summary.pawnShort': '{count} به سوی {name}',
  'game.summary.loot': {
    other: '{name} {count} سکه در غارت از دست می‌دهد'
  },
  'game.summary.pair': 'یک جفت نماد {symbol}: برداشتن یک نشان پیشرفت',
  'game.summary.pairShort': 'جفت'
} as const satisfies Translation<typeof en>;
