// Persian messages for the board chrome: the military track, progress
// tokens, science symbols, colours and resources as the board shows them,
// the strips, the turn bar, the top bar, the city and log sheets and the
// board tooltips. Complete. The house terms are in docs/i18n/persian.md.
// See ./index.ts.

import type { board as en } from '../en/board';
import type { Translation } from '../types';

export const board = {
  // Military track. The seat emblem is نماد (never نشان, which is a token).
  'board.crest.sun': 'خورشید',
  'board.crest.moon': 'ماه',
  'board.military.youLead': 'نظامی: شما {margin} خانه جلوترید.',
  'board.military.opponentLeads': 'نظامی: حریف {margin} خانه جلوتر است.',
  'board.military.crestLeads': 'نظامی: {crest} {margin} خانه جلوتر است.',
  'board.military.tied': 'نظامی: برابر.',
  'board.military.nextLootYou': 'غارت بعدی: در خانه‌ی {magnitude}، {value} سکه از دست می‌دهید.',
  'board.military.nextLootOpponent':
    'غارت بعدی: در خانه‌ی {magnitude}، حریف {value} سکه از دست می‌دهد.',
  'board.military.nextLootCrest':
    'غارت بعدی: در خانه‌ی {magnitude}، {crest} {value} سکه از دست می‌دهد.',
  'board.military.noLootLeft': 'نشان غارتی نمانده است؛ نوبت پایتخت است.',
  'board.military.lootCell': '{value} سکه در خانه‌ی {magnitude}',
  'board.military.lootTaken': '{value} سکه در خانه‌ی {magnitude}، پیش‌تر گرفته شده',
  'board.military.capitalYou': 'پایتخت شما',
  'board.military.capitalOpponent': 'پایتخت حریف',
  'board.military.capitalCrest': 'پایتخت {crest}',
  'board.military.legendLoot': 'سکه‌هایی که با ورود مهره به خانه از دست می‌رود',
  'board.military.legendPoints': 'امتیاز بازیکن جلوتر در پایان بازی',

  // Progress tokens.
  'token.agriculture': 'کشاورزی',
  'token.architecture': 'معماری',
  'token.economy': 'اقتصاد',
  'token.law': 'قانون',
  'token.masonry': 'بنایی',
  'token.mathematics': 'ریاضیات',
  'token.philosophy': 'فلسفه',
  'token.strategy': 'راهبرد',
  'token.theology': 'الهیات',
  'token.urbanism': 'شهرسازی',
  'board.token.take': 'برداشتن نشان {name}',
  'board.token.open': 'نشان {name}: چه می‌کند',
  'board.token.owned': '{name} (از آنِ شما)',
  'board.token.taken': '{name} (برداشته شده)',
  'board.tokens.boardLabel': 'نشان‌های پیشرفت روی صفحه',
  'board.tokens.empty': 'هیچ نشان پیشرفتی روی صفحه نمانده است.',

  // Science symbols and the card colours.
  'board.science.globe': 'کره',
  'board.science.scales': 'ترازو',
  'board.science.sundial': 'ساعت آفتابی',
  'board.science.mortar': 'هاون',
  'board.science.set-square': 'گونیا',
  'board.science.quill': 'قلم پر',
  'board.science.wheel': 'چرخ',
  'board.science.label': '{count} از شش نماد علمی متفاوت',
  'board.science.danger': 'نزدیک به برتری علمی',
  'board.color.brown': 'قهوه‌ای',
  'board.color.grey': 'خاکستری',
  'board.color.blue': 'آبی',
  'board.color.green': 'سبز',
  'board.color.yellow': 'زرد',
  'board.color.red': 'قرمز',
  'board.color.purple': 'بنفش',
  'board.city.colorCount': { other: '{count} کارت {color}' },

  // Resources.
  'board.resource.wood': 'چوب',
  'board.resource.clay': 'خاک رس',
  'board.resource.stone': 'سنگ',
  'board.resource.glass': 'شیشه',
  'board.resource.papyrus': 'پاپیروس',
  'board.resource.chip': '{resource}: تولید {count}، خرید هر واحد {coins}',
  'board.resource.choiceHint': 'از یک کارت تولید انتخابی هم به دست می‌آید',

  // A seat's wonders.
  'board.wonders.button': 'شگفتی‌های {name}: {built} از {total} ساخته شده',
  'board.wonders.buildableNow': 'اکنون می‌توان یک شگفتی ساخت',
  'board.wonders.title': 'شگفتی‌های {name}',
  'board.wonders.tally': '{built} از {total} ساخته شده',
  'board.wonders.back': 'همه‌ی شگفتی‌ها',

  // PlayerStrip / CityPanel.
  'board.player.militaryDanger': 'خطر نظامی: مهره به این پایتخت نزدیک است',
  'board.player.scienceDanger': 'نزدیک به برتری علمی',
  'board.player.points': { other: '{count} امتیاز' },
  'board.player.scorePreview': 'اگر بازی اکنون تمام شود: {total} امتیاز',
  'board.player.chains': 'ساخت رایگان: {count}',
  'board.player.thinking': 'در حال فکر کردن',

  // TurnBar.
  'board.turn.yourTurn': 'نوبت شما',
  'board.turn.playAgain': 'دوباره نوبت شماست',
  'board.turn.playAgainWonder': 'دوباره نوبت شماست ({wonder})',
  'board.turn.playAgainName': '{name} دوباره بازی می‌کند',
  'board.turn.playAgainNameWonder': '{name} دوباره بازی می‌کند ({wonder})',
  'board.turn.aiThinking': '{name} در حال فکر کردن است…',
  'board.turn.waitingFor': 'در انتظار {name}…',
  'board.turn.waitingAway': '{name} حاضر نیست. منتظر بازگشتش می‌مانیم…',
  'board.turn.sent': 'در حال فرستادن…',
  'board.turn.sentSlow': 'هنوز در انتظار سرور…',
  'board.turn.retry': 'تلاش دوباره',
  'board.turn.gameOver': 'پایان بازی',
  'board.turn.spectating': 'در حال تماشا',
  'board.turn.connecting': 'در حال اتصال…',
  'board.turn.pending.chooseStarter': 'انتخاب کنید چه کسی دوران بعد را آغاز کند',
  'board.turn.pending.destroyCard': 'کارتی را برای نابودی انتخاب کنید',
  'board.turn.pending.buildFromDiscard': 'کارتی از کارت‌های کنارگذاشته برای ساخت انتخاب کنید',
  'board.turn.pending.pickBoardToken': 'یک نشان پیشرفت انتخاب کنید',
  'board.turn.pending.pickLibraryToken': 'یک نشان از کتابخانه‌ی بزرگ انتخاب کنید',

  // Compact prompts for the short landscape side column.
  'board.turn.short.chooseStarter': 'آغازکننده را برگزینید',
  'board.turn.short.destroyCard': 'کارتی را نابود کنید',
  'board.turn.short.buildFromDiscard': 'کارتی را بازسازی کنید',
  'board.turn.short.pickBoardToken': 'نشان پیشرفت را برگزینید',
  'board.turn.short.pickLibraryToken': 'نشان کتابخانه را برگزینید',
  'board.turn.short.offline': 'آفلاین. اتصال دوباره…',

  // TopBar.
  'board.age': 'دوران {age}',
  'board.age.draft': 'گزینش شگفتی‌ها',
  'board.topbar.undo': 'برگرداندن',
  'board.topbar.log': 'گزارش',
  'board.topbar.settings': 'تنظیمات',

  // CitySheet.
  'board.citySheet.title': 'شهر',
  'board.citySheet.resources': 'منابع',
  'board.citySheet.science': 'علم',
  'board.citySheet.wonders': 'شگفتی‌ها',
  'board.citySheet.tokens': 'نشان‌ها',
  'board.citySheet.city': 'شهر',
  'board.citySheet.chains': 'ساخت رایگان',

  // LogSheet and lib/view/log-format.ts.
  'board.logSheet.title': 'گزارش',
  'board.logSheet.empty': 'هنوز اتفاقی نیفتاده است.',
  'board.logSheet.age': 'دوران {age}',
  'board.logSheet.draftAge': 'گزینش شگفتی‌ها',
  'board.logSheet.replay': 'بازپخش آخرین حرکت',
  'board.logSheet.replayOff': 'برای بازپخش یک حرکت، پویانمایی را در تنظیمات روشن کنید.',
  'board.logSheet.replayNone': 'هنوز حرکتی برای بازپخش نیست.',
  'board.log.draft': '{name} شگفتی {wonder} را برداشت.',
  'board.log.draftRound2': 'دور دوم شگفتی‌ها رو شد.',
  'board.log.ageStart': 'دوران {age} آغاز شد.',
  'board.log.pawn': 'مهره‌ی نظامی به سوی {name} رفت.',
  'board.log.build': '{name} {card} را ساخت.',
  'board.log.discard': '{name} {card} را برای {amount} سکه کنار گذاشت.',
  'board.log.wonder': '{name} شگفتی {wonder} را با {card} ساخت.',
  'board.log.loot': '{name} در غارت {amount} سکه از دست داد.',
  'board.log.pair': '{name} یک جفت نماد {symbol} کامل کرد.',
  'board.log.token': '{name} نشان {token} را برداشت.',
  'board.log.libraryDraw': '{name} از {wonder} نشان کشید.',
  'board.log.destroy': '{name} {card} را نابود کرد.',
  'board.log.rebuild': '{name} {card} را از کارت‌های کنارگذاشته ساخت.',
  'board.log.wonderRemoved': 'شگفتی ساخته‌نشده‌ی {name}، {wonder}، از بازی بیرون رفت.',
  'board.log.effectSkipped': '{name} گزینه‌ای برای این انتخاب ندارد.',
  'board.log.skipped.destroyCard':
    'چیزی برای نابودی با {source} نیست: {opponent} کارتی از آن رنگ ندارد.',
  'board.log.skipped.buildFromDiscard':
    'چیزی برای ساخت با {source} نیست: کارت کنارگذاشته‌ای وجود ندارد.',
  'board.log.skipped.pickBoardToken': 'هیچ نشان پیشرفتی روی صفحه برای برداشتن نمانده است.',
  'board.log.skipped.pickLibraryToken':
    'چیزی برای کشیدن با {source} نیست: هیچ نشان پیشرفتی در جعبه نمانده است.',
  'board.log.extraTurn': '{name} دوباره بازی می‌کند.',
  'board.log.extraTurnLost': 'نوبت دوباره‌ی {name} با پایان دوران از دست رفت.',
  'board.log.starterChosen': '{name} {starter} را برای آغاز دوران بعد برگزید.',
  'board.log.resign': '{name} بازی را واگذار کرد.',

  // Tooltips of board micro-elements.
  'board.slot.wonderMark': 'این کارت می‌تواند زیر شگفتی انتخاب‌شده برود.',
  'board.military.trackTip': 'مسیر نظامی. {lead} {next}',
  'board.crest.tip': 'نماد {crest}.',
  'board.crest.tipSelf': 'نماد {crest}: جایگاه شما.'
} as const satisfies Translation<typeof en>;
