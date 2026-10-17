// Persian messages for the hints: the families and the settings editor, the
// per-card, outcome and board-level hints, the advisor's move review, the
// recap, the first-time tips, and the rules reference's "Hint glyphs"
// section. Complete: every key of ../en/hints.ts is translated. A few
// patterns hold no words and so read as in English: hints.badge.bonus
// ('+{count}'), hints.cardWith, hints.after.byMove, hints.science.symbol
// and hints.science.card. See ./index.ts.

import type { hints as en } from '../en/hints';
import type { Translation } from '../types';

export const hints = {
  // The families, each with a one-line description (lib/hints/families.ts).
  'hints.family.prices': 'قیمت‌ها',
  'hints.family.prices.hint': 'هزینه‌ی کارت‌ها و شگفتی‌ها، برای هر دو بازیکن.',
  'hints.family.economy': 'اقتصاد',
  'hints.family.economy.hint': 'اثر هر کارت بر تولید و قیمت خرید.',
  'hints.family.science': 'علم',
  'hints.family.science.hint': 'نمادهای علمی، جفت‌ها، و این‌که کدام‌ها هنوز در بازی‌اند.',
  'hints.family.military': 'نظامی',
  'hints.family.military.hint': 'مهره به کجا می‌رود و چه غارت می‌کند.',
  'hints.family.threats': 'تهدیدها',
  'hints.family.threats.hint': 'حرکتی که بازی را می‌برد، یا پیروزی را به حریف می‌دهد.',
  'hints.family.tempo': 'ضرباهنگ',
  'hints.family.tempo.hint':
    'یک حرکت کدام کارت‌ها را آزاد می‌کند، و چه کسی دوران بعد را آغاز می‌کند.',
  'hints.family.wonders': 'شگفتی‌ها',
  'hints.family.wonders.hint': 'هر شگفتی اکنون چه می‌کند، و رقابت بر سر شگفتی هفتم.',
  'hints.family.tokens': 'نشان‌های پیشرفت',
  'hints.family.tokens.hint': 'ارزش کنونی هر نشان پیشرفت.',
  'hints.family.scoring': 'امتیازشماری',
  'hints.family.scoring.hint': 'امتیاز هر بازیکن از کجا می‌آید.',
  'hints.family.counting': 'شمارش کارت',
  'hints.family.counting.hint':
    'کدام کارت‌ها هنوز دیده نشده‌اند، بر پایه‌ی فهرست‌های همگانی کارت‌ها.',
  'hints.family.recap': 'مرور',
  'hints.family.recap.hint': 'آنچه در آخرین حرکت تغییر کرد.',
  'hints.family.advisor': 'مشاور',
  'hints.family.advisor.hint':
    'پس از حرکت شما، این‌که آیا حرکت بهتری در کار بود، و چرا. فقط شما آن را می‌بینید.',

  // The editor (HintSettingsEditor.svelte).
  'hints.title': 'راهنماها',
  'hints.sameForBoth': 'هر دو بازیکن با راهنماهای یکسان بازی می‌کنند.',
  'hints.readOnly': 'فقط بازیکنی که بازی را ساخته می‌تواند آن‌ها را تغییر دهد.',
  'hints.on': 'روشن',
  'hints.off': 'خاموش',
  'hints.saving': 'در حال ذخیره…',

  // The local hide (HintLocalHide.svelte).
  'hints.local.title': 'روی صفحه‌ی من',
  'hints.local.intro':
    'راهنماها را فقط روی همین صفحه پنهان کنید. راهنمایی که بازی خاموش کرده هرگز نمایش داده نمی‌شود.',
  'hints.local.boardMarks': 'پنهان کردن نشانه‌های راهنما روی صفحه‌ی بازی',
  'hints.local.boardMarksHint': 'برگه‌ها همچنان راهنماها را نشان می‌دهند.',
  'hints.local.hideFamily': 'پنهان کردن {family} روی صفحه‌ی من',

  // The read-only summary (HintsSummary.svelte), and the collapsed new-game
  // section. {families} is a list of family names.
  'hints.summary.all': 'همه‌ی راهنماها روشن',
  'hints.summary.none': 'بدون راهنما',
  'hints.summary.allBut': 'همه‌ی راهنماها روشن به جز {families}',
  'hints.summary.only': 'فقط {families}',
  'hints.lobby.title': 'راهنماهای این بازی',

  // The game menu item and its sheet (HintsMenuItem.svelte, GameHintsSheet.svelte).
  'hints.menu': 'راهنماها',
  'hints.sheet.title': 'راهنماهای این بازی',
  // The game's undo setting, read-only under the hints (hot-seat and bot games).
  'hints.sheet.undoTitle': 'برگرداندن',
  'hints.sheet.undoOn': 'در این بازی برگرداندن حرکت مجاز است.',
  'hints.sheet.undoOff': 'در این بازی برگرداندن حرکت خاموش است.',
  'hints.sheet.undoFixed': 'هنگام ساختن بازی تعیین شده و تغییرپذیر نیست.',

  // A change in the game log (lib/hints/log.ts). {families} is a list of
  // family names; {changes} joins the "on" and "off" parts.
  'hints.log.on': '{families} روشن شد',
  'hints.log.off': '{families} خاموش شد',
  'hints.log.separator': '؛ ',
  'hints.log.changedBy': '{name} راهنماها را تغییر داد: {changes}.',
  'hints.log.changed': 'راهنماها تغییر کرد: {changes}.',

  // Per-card hints (lib/view/hints.ts, M10.HS2): the pip on a card, its
  // tooltip lines, and the card sheet's hint sections and notes. {symbol}
  // and {resource} are lower-case names (card.symbol.*, card.resource.*).
  'hints.pip.title': 'راهنماهای {card}',
  'hints.pip.win.you': 'اگر این را بسازید، برنده می‌شوید.',
  'hints.pip.win.named': '{name} با ساختن این برنده می‌شود.',
  'hints.pip.win.lawYou':
    'با آن برنده می‌شوید: جفتی را کامل می‌کند، و قانون ششمین نماد شما می‌شود.',
  'hints.pip.win.lawNamed':
    '{name} با آن برنده می‌شود: جفتی را کامل می‌کند، و قانون ششمین نماد او می‌شود.',
  'hints.pip.pair.you': 'جفت {symbol} شما را کامل می‌کند: یک نشان پیشرفت برمی‌دارید.',
  'hints.pip.pair.named': 'جفت {symbol} را برای {name} کامل می‌کند: او یک نشان پیشرفت برمی‌دارد.',
  'hints.pip.loot.you': {
    other: 'اگر این را بسازید، {name} {count} سکه از دست می‌دهد.'
  },
  'hints.pip.loot.youLose': {
    other: 'اگر {name} این را بسازد، {count} سکه از دست می‌دهید.'
  },
  'hints.pip.loot.named': {
    other: 'اگر {name} این را بسازد، {other} {count} سکه از دست می‌دهد.'
  },

  'hints.victory.military': 'نظامی',
  'hints.victory.science': 'علمی',
  'hints.victory.civilian': 'امتیاز',
  'hints.victory.forfeit': 'واگذاری',

  'hints.section.you': 'برای شما',
  'hints.section.named': 'برای {name}',
  'hints.line.win': 'ساختنش یعنی پیروزی ({kind}).',
  'hints.line.winShort': 'پیروزی می‌آورد ({kind})، ولی فعلاً نمی‌شود هزینه‌اش را پرداخت.',
  'hints.line.winLaw': 'جفتی را کامل می‌کند، و آن‌گاه قانون پیروزی می‌آورد.',
  'hints.line.pair': 'جفت {symbol} را کامل می‌کند و یک نشان پیشرفت می‌دهد.',
  'hints.line.pairNoToken': 'جفت {symbol} را کامل می‌کند، ولی نشانی نمانده است.',
  'hints.line.newSymbol': {
    other: 'نمادی تازه، {symbol}: {count} نماد متفاوت از شش.'
  },
  'hints.line.pawn': {
    other: 'مهره {count} خانه به سوی پایتخت {name} می‌رود.'
  },
  'hints.line.pawnYou': {
    other: 'مهره {count} خانه به سوی پایتخت شما می‌رود.'
  },
  'hints.line.loot': {
    other: 'غارت: {name} {count} سکه از دست می‌دهد.'
  },
  'hints.line.lootYou': {
    other: 'غارت: {count} سکه از دست می‌دهید.'
  },
  'hints.line.price.you': '{resource} برای شما: هر واحد از {from} به {to} سکه.',
  'hints.line.price.named': '{resource} برای {name}: هر واحد از {from} به {to} سکه.',
  'hints.line.saves': {
    other: 'اکنون در کارت‌ها و شگفتی‌های نمایان {count} سکه صرفه‌جویی می‌کند.'
  },
  'hints.line.savesNothing': 'اکنون در آنچه نمایان است صرفه‌جویی نمی‌کند.',
  'hints.line.lastCertain': 'دیگر هیچ تولیدکننده‌ی {resource} دیگری نمی‌تواند بیاید.',
  'hints.line.lastVisible':
    'آخرین تولیدکننده‌ی {resource} که نمایان است؛ کارت‌های ندیده شاید یکی دیگر داشته باشند.',
  'hints.line.worth': {
    other: 'ارزش کنونی: {count} سکه، {vp} امتیاز.'
  },
  'hints.line.worthCoins': {
    other: 'امروز ساختنش {count} سکه می‌ارزد.'
  },
  'hints.line.onlySource': 'تنها منبع {resource} برای {name}.',
  'hints.line.onlySourceYou': 'تنها منبع {resource} شما.',

  'hints.note.deniesChain':
    'آن را از {name} هم دور نگه می‌دارد (برای او از راه زنجیره رایگان است).',
  'hints.note.deniesChainForNow':
    'فعلاً آن را از {name} دور نگه می‌دارد (برای او از راه زنجیره رایگان است): آرامگاه او هنوز می‌تواند دوباره بسازدش.',
  'hints.note.deniesCheap': {
    other: 'آن را از {name} هم دور نگه می‌دارد، که فقط {count} سکه می‌پرداخت.'
  },
  'hints.note.deniesCheapForNow': {
    other:
      'فعلاً آن را از {name} دور نگه می‌دارد (برای او {count} سکه): آرامگاه او هنوز می‌تواند دوباره بسازدش.'
  },
  'hints.note.mausoleumOpp':
    'آرامگاه {name} می‌تواند آن را رایگان دوباره بسازد. اگر زیر شگفتی گذاشته شود، از دسترس بیرون است.',
  'hints.note.mausoleumYou': 'آرامگاه شما می‌تواند بعداً آن را دوباره بسازد.',
  'hints.note.mausoleumNamed': 'آرامگاه {name} می‌تواند بعداً آن را دوباره بسازد.',
  'hints.note.denyBuild': 'ساختنش آن را از {name} می‌گیرد.',
  'hints.note.denyTuck': 'اگر زیر شگفتی گذاشته شود، برای {name} از بازی بیرون است.',
  'hints.note.shortByDiscard': {
    other: 'یک کنار گذاشتن (+{count} سکه) با قیمت‌های امروز هزینه‌اش را جور می‌کند.'
  },

  'hints.discard.poolYou': 'آرامگاه شما می‌تواند دوباره بسازد',
  'hints.discard.poolNamed': 'آرامگاه {name} می‌تواند دوباره بسازد',
  'hints.cardWith': '{card}: {hint}',

  // The decider's hints (lib/view/outcome.ts, M10.HS3): the outcome line
  // under each move, the confirm step's warning, the card sheet's "After
  // this pick" panel and the starter choice. {delta} is a signed number
  // ("+3"), {card} and {list} are names, {age} a number.
  'hints.outcome.separator': '؛ ',
  'hints.outcome.win': 'بازی را می‌برد ({kind}).',
  'hints.outcome.winLaw': 'پیروزی: جفتی را کامل می‌کند، و قانون ششمین نماد می‌شود.',
  'hints.outcome.winVia': 'با گام بعدی می‌برد: {card}.',
  'hints.outcome.winTwoStep': 'در نوبت دوباره، با {card} می‌برد.',
  'hints.outcome.endYou': 'بازی تمام می‌شود: شما با امتیاز می‌برید، {a} به {b}.',
  'hints.outcome.endNamed': 'بازی تمام می‌شود: {name} با امتیاز می‌برد، {a} به {b}.',
  'hints.outcome.endBlueYou':
    'بازی تمام می‌شود: هر کدام {a} امتیاز، و شما با امتیاز کارت‌های آبی می‌برید.',
  'hints.outcome.endBlueNamed':
    'بازی تمام می‌شود: هر کدام {a} امتیاز، و {name} با امتیاز کارت‌های آبی می‌برد.',
  'hints.outcome.endShared': 'بازی با پیروزی مشترک تمام می‌شود، هر کدام {a} امتیاز.',
  'hints.outcome.vp': '{delta} امتیاز',
  'hints.outcome.vpNamed': '{name}: {delta} امتیاز',
  'hints.outcome.pawn': {
    other: 'مهره {count} خانه به سوی پایتخت {name}'
  },
  'hints.outcome.pawnYou': {
    other: 'مهره {count} خانه به سوی پایتخت شما'
  },
  'hints.outcome.loot': {
    other: '{name} با غارت {count} سکه از دست می‌دهد'
  },
  'hints.outcome.lootYou': {
    other: 'با غارت {count} سکه از دست می‌دهید'
  },
  'hints.outcome.coins': {
    other: 'پس از آن {count} سکه ({delta})'
  },
  'hints.outcome.coinsNamed': {
    other: '{name}: پس از آن {count} سکه ({delta})'
  },
  'hints.outcome.distinct': {
    other: '{count} نماد علمی متفاوت'
  },
  'hints.outcome.playAgain': 'نوبت دوباره',
  'hints.outcome.playAgainTheology': 'نوبت دوباره (الهیات)',
  'hints.outcome.extraLost': 'دوران تمام می‌شود: نوبت دوباره‌ای در کار نیست',
  'hints.outcome.starterYou': 'شما انتخاب می‌کنید چه کسی دوران {age} را آغاز کند',
  'hints.outcome.starterNamed': '{name} انتخاب می‌کند چه کسی دوران {age} را آغاز کند',
  'hints.outcome.opens.chooseStarter': 'سپس انتخاب کنید چه کسی آغاز کند',
  'hints.outcome.opens.destroyCard': 'سپس کارتی را برای نابودی انتخاب کنید',
  'hints.outcome.opens.buildFromDiscard': 'سپس یک کارت کنارگذاشته را دوباره بسازید',
  'hints.outcome.opens.pickBoardToken': 'سپس یک نشان پیشرفت بردارید',
  'hints.outcome.opens.pickLibraryToken': 'سپس از میان سه نشان یکی بردارید',
  'hints.outcome.shortWin': {
    other: 'برای پیروزی ({kind}) {count} سکه کم است.'
  },

  'hints.blunder.gives': 'این حرکت پیروزی را به {name} می‌دهد: {list}.',
  'hints.blunder.missedYou': 'به جای آن می‌توانید همین نوبت ببرید: {list}.',
  'hints.blunder.missedNamed': '{name} به جای آن می‌تواند همین نوبت ببرد: {list}.',

  'hints.after.title': 'پس از این انتخاب',
  'hints.after.freesNothing': 'هیچ کارتی را آزاد نمی‌کند.',
  'hints.after.frees': {
    one: '{list} را آزاد می‌کند.',
    other: '{count} کارت را آزاد می‌کند: {list}.'
  },
  'hints.after.faceDown': {
    other: '{count} کارت پشت‌ورو'
  },
  'hints.after.reveal': {
    one: 'نخستین فرصت برداشتن کارتی که رو می‌شود با {name} است.',
    other: 'نخستین فرصت برداشتن کارت‌هایی که رو می‌شوند با {name} است.'
  },
  'hints.after.revealWonderYou':
    'اگر آن را زیر شگفتی‌ای بگذارید که نوبت دوباره می‌دهد، کارتی را که رو می‌شود اول خودتان برمی‌دارید.',
  'hints.after.revealWonderNamed':
    'اگر {name} آن را زیر شگفتی‌ای بگذارد که نوبت دوباره می‌دهد، کارتی را که رو می‌شود اول خودش برمی‌دارد.',
  'hints.after.giftWin': 'آن‌گاه {card} بازی را برای {name} می‌برد.',
  'hints.after.giftPair': 'آن‌گاه {card} به {name} یک جفت علمی می‌دهد.',
  'hints.after.giftChain': 'آن‌گاه {card} برای {name} از راه زنجیره رایگان می‌شود.',
  'hints.after.giftPays': {
    other: 'آن‌گاه {card} برای {name} {count} سکه هزینه دارد.'
  },
  'hints.after.gives': 'آن‌گاه {name} می‌تواند با {list} ببرد.',
  'hints.after.blocks': 'پیروزی {name} با {list} را از او می‌گیرد.',
  'hints.after.threatensYou':
    'نوبت بعد می‌توانید با {list} ببرید، مگر این‌که {name} جلویش را بگیرد.',
  'hints.after.threatensNamed':
    'نوبت بعد {mover} می‌تواند با {list} ببرد، مگر این‌که {name} جلویش را بگیرد.',
  'hints.after.risk': {
    one: 'به احتمال {num} از {den}، کارتی که رو می‌شود به {name} پیروزی می‌دهد.',
    other: 'به احتمال {num} از {den}، یکی از کارت‌هایی که رو می‌شوند به {name} پیروزی می‌دهد.'
  },
  'hints.after.byMove': '{move}: {line}',
  'hints.after.moveBuild': 'ساختن',
  'hints.after.moveDiscard': 'کنار گذاشتن',

  'hints.starter.threatYou': 'اگر شما آغاز کنید، می‌توانید بی‌درنگ با {list} ببرید.',
  'hints.starter.threatNamed': 'اگر {name} آغاز کند، می‌تواند بی‌درنگ با {list} ببرد.',
  'hints.starter.opening': 'دوران با {list} آغاز می‌شود.',
  'hints.ghost.pawn': 'جایی که این حرکت مهره را می‌برد.',
  'hints.ghost.loot': {
    other: 'این حرکت این نشان غارت را برمی‌دارد: {count} سکه از دست می‌رود.'
  },
  'hints.ghost.coins': {
    other: 'پس از این حرکت {count} سکه'
  },
  'hints.ghost.score': {
    other: 'پس از این حرکت {count} امتیاز'
  },

  // Board-level hints (lib/view/board-hints.ts, M10.HS4): where a card is,
  // the science tracker, the military track, the wonders, the progress
  // tokens, the age chip and the age interstitial. {place} is one of the
  // hints.place.* phrases; {symbol} a symbol's name; {age} a number.
  'hints.place.faceUp': 'رو، در صفحه‌ی بازی',
  'hints.place.tucked': 'زیر یک شگفتی، بیرون از بازی',
  'hints.place.cityYou': 'در شهر شما',
  'hints.place.cityNamed': 'در شهر {name}',
  'hints.place.discard': 'در میان کارت‌های کنارگذاشته',
  'hints.place.unseenThisAge': 'هنوز در این دوران دیده نشده',
  'hints.place.unseenThisAgeOdds': 'هنوز در این دوران دیده نشده (احتمال پخش شدنش {num} از {den})',
  'hints.place.laterAge': 'در دسته‌ی کارت‌های دوران {age}',
  'hints.place.laterAgeOdds': 'در دسته‌ی کارت‌های دوران {age} (احتمال پخش شدنش {num} از {den})',
  'hints.place.removed': 'از بازی بیرون گذاشته شده',

  'hints.science.symbol': '{symbol}.',
  'hints.science.card': '{card}: {place}.',
  'hints.science.paired': 'یک جفت.',
  'hints.science.live': 'یک کارت؛ دومی هنوز ممکن است بیاید.',
  'hints.science.owned': 'یک کارت؛ دومی دیگر نمی‌تواند بیاید.',
  'hints.science.open': {
    other: 'هنوز هیچ کارتی؛ {count} کارت ممکن است بیاید.'
  },
  'hints.science.dead': 'هیچ کارتی، و دیگر هیچ کارتی از آن نمی‌تواند بیاید.',
  'hints.science.scalesOwned': 'به دست آمده، از نشان قانون.',
  'hints.science.scalesOpen': 'فقط از نشان قانون، که هنوز ممکن است بیاید.',
  'hints.science.scalesDead': 'فقط از نشان قانون، که دیگر نمی‌تواند بیاید.',
  'hints.science.needYou': {
    other: 'به {count} نماد متفاوت دیگر نیاز دارید؛ هنوز ممکن: {list}.'
  },
  'hints.science.needNamed': {
    other: '{name} به {count} نماد متفاوت دیگر نیاز دارد؛ هنوز ممکن: {list}.'
  },
  'hints.science.unreachableYou': 'دیگر نمی‌توانید با برتری علمی برنده شوید.',
  'hints.science.unreachableNamed': '{name} دیگر نمی‌تواند با برتری علمی برنده شود.',
  'hints.science.lawYou': 'قانون ششمین نماد شما می‌شود: جفتی که اکنون کامل کنید، پیروزی می‌آورد.',
  'hints.science.lawNamed':
    'قانون ششمین نماد {name} می‌شود: جفتی که اکنون کامل کند، پیروزی می‌آورد.',

  'hints.capital.win': 'رسیدن به این خانه همان دم پیروزی می‌آورد.',
  'hints.capital.titleYou': 'پایتخت شما.',
  'hints.capital.titleNamed': 'پایتخت {name}.',
  'hints.capital.yours': 'پایتخت شما',
  'hints.capital.named': 'پایتخت {name}',
  'hints.capital.needYou': {
    other: 'برای گرفتن {capital} به {count} سپر دیگر نیاز دارید.'
  },
  'hints.capital.needNamed': {
    other: '{name} برای گرفتن {capital} به {count} سپر دیگر نیاز دارد.'
  },
  'hints.capital.safeYou': 'دیگر نمی‌توانید این‌قدر سپر گرد آورید.',
  'hints.capital.safeNamed': '{name} دیگر نمی‌تواند این‌قدر سپر گرد آورد.',
  'hints.zone.you': {
    other:
      'اگر مهره در پایان اینجا باشد، {count} امتیاز می‌گیرید. فقط بازیکنی که جلوتر است امتیاز می‌گیرد.'
  },
  'hints.zone.named': {
    other:
      'اگر مهره در پایان اینجا باشد، {name} {count} امتیاز می‌گیرد. فقط بازیکنی که جلوتر است امتیاز می‌گیرد.'
  },

  'hints.loot.taken': 'پیش‌تر برداشته شده: دیگر هرگز به کار نمی‌افتد.',
  'hints.loot.triggerYou': {
    other: 'وقتی مهره وارد این خانه شود، {count} سکه از دست می‌دهید.'
  },
  'hints.loot.triggerNamed': {
    other: 'وقتی مهره وارد این خانه شود، {name} {count} سکه از دست می‌دهد.'
  },
  'hints.loot.clampYou': {
    other: 'اکنون فقط {count} سکه دارید.'
  },
  'hints.loot.clampNamed': {
    other: '{name} اکنون فقط {count} سکه دارد.'
  },

  'hints.glow.militaryWin': '{name} اکنون می‌تواند با {list} این پایتخت را بگیرد.',
  'hints.glow.militaryWinYou': 'اکنون می‌توانید با {list} این پایتخت را بگیرید.',
  'hints.glow.calm': 'نزدیک است، ولی دیگر نمی‌تواند رخ دهد.',

  // The city sheet's threats list (view/threats.ts threatLines): the wins a
  // seat would have with one move on its turn.
  'hints.threats.title': 'تهدیدها',
  'hints.threats.militaryYou': 'در نوبتتان می‌توانید با {list} پایتخت را بگیرید.',
  'hints.threats.military': '{name} در نوبتش می‌تواند با {list} پایتخت را بگیرد.',
  'hints.threats.scienceYou': 'در نوبتتان می‌توانید با {list} به برتری علمی برسید.',
  'hints.threats.science': '{name} در نوبتش می‌تواند با {list} به برتری علمی برسد.',
  'hints.threats.civilianYou': 'در نوبتتان می‌توانید با {list} بازی را با امتیاز بیشتر تمام کنید.',
  'hints.threats.civilian': '{name} در نوبتش می‌تواند با {list} بازی را با امتیاز بیشتر تمام کند.',

  'hints.wonder.library':
    '{count} تا از {total} نشان درون جعبه را بیرون می‌کشد: {list}. فقط صاحبش آن‌ها را می‌بیند.',
  'hints.wonder.lastSlot': 'فقط یک شگفتی دیگر می‌تواند ساخته شود.',
  'hints.wonder.removes': 'ساختن این یکی، {wonder} را از {name} می‌گیرد و به جعبه برمی‌گرداند.',
  'hints.wonder.removesYour': 'ساختن این یکی، {wonder} شما را به جعبه برمی‌گرداند.',
  'hints.wonder.affordYou': 'اکنون از پس هزینه‌اش برمی‌آیید.',
  'hints.wonder.affordNamed': '{name} اکنون از پس هزینه‌اش برمی‌آید.',
  'hints.wonder.readyYou': 'با قیمت‌های امروز، می‌توانید این را در نوبتتان بسازید.',
  'hints.wonder.readyNamed': 'با قیمت‌های امروز، {name} می‌تواند این را در نوبتش بسازد.',
  'hints.wonder.readyCountYou': {
    other: 'با قیمت‌های امروز می‌توانید {count} شگفتی بسازید.'
  },
  'hints.wonder.readyCountNamed': {
    other: 'با قیمت‌های امروز {name} می‌تواند {count} شگفتی بسازد.'
  },
  'hints.wonder.slotsLeft': {
    other: '{count} جای شگفتی مانده، {unbuilt} شگفتی ساخته‌نشده.'
  },

  'hints.token.forYou': 'برای شما: {value}.',
  'hints.token.forNamed': 'برای {name}: {value}.',
  'hints.token.wins': 'بازی را می‌برد',
  'hints.token.coins': { other: '+{count} سکه' },
  'hints.token.vp': '{delta} امتیاز',
  'hints.token.saves': {
    other: '{count} سکه صرفه‌جویی در کارت‌ها و شگفتی‌های نمایان'
  },
  'hints.token.savesNothing': 'هیچ صرفه‌جویی در آنچه اکنون نمایان است',
  'hints.token.economy': {
    other: 'خرید {name} از بانک برای کارت‌هایی که می‌تواند بردارد: {count} سکه'
  },
  'hints.token.chains': {
    other: '{count} زنجیره‌ی هنوز باز'
  },
  'hints.token.theology': {
    other: '{count} شگفتی ساخته‌نشده نوبت دوباره می‌گیرد'
  },
  'hints.token.strategy': 'کارت‌های قرمز بعدی یک سپر بیشتر می‌گیرند',
  'hints.token.nothing': 'فعلاً هیچ',

  'hints.age.cardsLeft': {
    other: 'در این دوران {count} کارت مانده است.'
  },
  'hints.age.lastYou': 'اگر کسی نوبت دوباره نگیرد، آخرین کارت را شما برمی‌دارید.',
  'hints.age.lastNamed': 'اگر کسی نوبت دوباره نگیرد، {name} آخرین کارت را برمی‌دارد.',
  'hints.age.lastLost': 'نوبت دوباره‌ای که با آخرین کارت گرفته شود، از دست می‌رود.',
  'hints.age.chooserYou': 'شما بازیکن ضعیف‌ترید و انتخاب می‌کنید چه کسی دوران {age} را آغاز کند.',
  'hints.age.chooserNamed':
    '{name} بازیکن ضعیف‌تر است و انتخاب می‌کند چه کسی دوران {age} را آغاز کند.',
  'hints.age.lastStarts': 'بازیکنی که آخرین کارت را بردارد، دوران {age} را آغاز می‌کند.',
  'hints.age.lastChooses':
    'بازیکنی که آخرین کارت را بردارد، انتخاب می‌کند چه کسی دوران {age} را آغاز کند.',
  'hints.age.deadPairYou': '{card} از بازی بیرون گذاشته شد: {own} شما هرگز جفت نمی‌شود.',
  'hints.age.deadPairNamed': '{card} از بازی بیرون گذاشته شد: {own} هرگز برای {name} جفت نمی‌شود.',

  'hints.draft.order': '{first} یکی برمی‌دارد، {second} دو تا، و آخری به {first} می‌رسد.',

  // The badge modifiers (card/AffordBadge.svelte).
  'hints.badge.bonus': '+{count}',
  'hints.badge.urbanismTip': {
    other: 'رایگان از راه زنجیره، و شهرسازی {count} سکه به شما می‌دهد.'
  },
  'hints.badge.flowTip': {
    other: '{count} سکه از این خرید به {name} می‌رسد (اقتصاد).'
  },

  // The move review (lib/view/review.ts, components/game/MoveReview.svelte,
  // M10.HS5): inside the summary of the opponent's next move, what the
  // player did and whether a clearly better move was there. {move} is one
  // of the hints.review.move.* phrases.
  'hints.review.title': 'حرکت شما',
  'hints.review.tip': 'مشاور پس از بازی شما، حرکت خودتان را بررسی می‌کند. فقط شما آن را می‌بینید.',
  'hints.review.played.build': 'شما {card} را ساختید.',
  'hints.review.played.discard': 'شما {card} را کنار گذاشتید.',
  'hints.review.played.wonder': 'شما شگفتی {wonder} را ساختید.',
  'hints.review.played.other': 'آخرین حرکت شما.',
  'hints.review.move.build': 'ساختن {card}',
  'hints.review.move.discard': 'کنار گذاشتن {card}',
  'hints.review.move.wonder': 'ساختن {wonder}',
  'hints.review.move.other': 'حرکتی دیگر',
  'hints.review.none': 'حرکت آشکارا بهتری پیدا نشد.',
  'hints.review.wins': 'با {move} بازی را می‌بردید.',
  'hints.review.stopsMilitary': '{move} نمی‌گذاشت {name} پایتخت شما را بگیرد.',
  'hints.review.stopsWinWith': '{move} نمی‌گذاشت {name} با {card} ببرد.',
  'hints.review.stopsWin': '{move} نمی‌گذاشت {name} در نوبت بعد ببرد.',
  'hints.review.pair': 'با {move} جفت {symbol} شما کامل می‌شد.',
  'hints.review.deniesPair': '{move} نمی‌گذاشت {name} به جفت {symbol} برسد.',
  'hints.review.loot': {
    other: 'با {move} {count} سکه از {name} غارت می‌کردید.'
  },
  'hints.review.points': {
    other: '{move} حدود {count} امتیاز بهتر به نظر می‌رسید.'
  },

  // What changed since the player's last move (lib/view/recap.ts): the
  // trade prices the other side's moves changed, shown in the move summary.
  'hints.recap.title': 'قیمت‌های شما',
  'hints.recap.tip': 'قیمت‌های خرید که از آخرین حرکت شما تغییر کرده‌اند.',
  'hints.recap.price': '{resource} برای شما: پیش‌تر {from}، اکنون {to} سکه.',
  'hints.recap.priceBuilt':
    '{resource} برای شما: پیش‌تر {from}، اکنون {to} سکه. {name} کارت {card} را ساخت.',
  // The board's marks of the same changes (M10.F3): the trade-price chip's
  // caret, a card's badge, the emptied slot and a card just turned face up.
  'hints.recap.chip': 'از آخرین حرکت شما: پیش‌تر {from}، اکنون {to}.',
  'hints.recap.chipBuilt':
    'از آخرین حرکت شما: پیش‌تر {from}، اکنون {to}: {name} کارت {card} را ساخت.',
  'hints.recap.cardPrice': 'پیش‌تر {from}، اکنون {to}.',
  'hints.recap.cardPriceBuilt': 'پیش‌تر {from}، اکنون {to}: {name} کارت {card} را ساخت.',
  'hints.recap.emptied': 'از آخرین حرکت شما برداشته شده.',
  'hints.recap.revealed': 'از آخرین حرکت شما رو شده.',

  // First-time tips (lib/ui/tips.svelte.ts, components/hints/FirstTip.svelte):
  // each explains a hint mark once, the first time it appears on this device.
  'hints.tip.label': 'نکته',
  'hints.tip.pair':
    'هر نماد علمی روی دو کارت است. هر دو را داشته باشید تا یک نشان پیشرفت بردارید. این نشانه کارتی را نشان می‌دهد که جفت شما را کامل می‌کند.',
  'hints.tip.denyPair':
    'این کارت جفت علمی {name} را کامل می‌کند و او یک نشان پیشرفت برمی‌دارد. برداشتنش آن را از او دور نگه می‌دارد.',
  'hints.tip.win': 'تاج کارتی را نشان می‌دهد که اگر اکنون بسازید، بازی را می‌برید.',
  'hints.tip.winOpp': 'تاج توخالی کارتی را نشان می‌دهد که برای {name} پیروزی می‌آورد.',
  'hints.tip.loot':
    'سپرهای این کارت مهره را روی یک نشان غارت می‌برند، و بازیکن دیگر سکه از دست می‌دهد.',
  'hints.tip.lawLaurel':
    'حلقه‌ی برگ بو دور قانون: نماد آن ششمین نماد متفاوت آن بازیکن می‌شود، پس جفتی که اکنون کامل کند می‌تواند قانون را بگیرد و بازی را ببرد.',
  'hints.tip.ghostLoot':
    'یک نشان غارت: وقتی مهره وارد خانه‌اش شود، آن طرف بی‌درنگ این سکه‌ها را از دست می‌دهد. وقتی به کارتی با سپر نگاه می‌کنید، حلقه‌ای خط‌چین هر نشانی را که برمی‌دارد نشان می‌دهد.',
  'hints.tip.hourglass':
    'ساعت شنی: در این بازی فقط یک شگفتی دیگر می‌تواند ساخته شود. هر کس زودتر یکی بسازد جای آخر را می‌گیرد، و شگفتی ساخته‌نشده‌ی دیگر به جعبه برمی‌گردد.',
  'hints.tip.freeChain':
    'برچسب زنجیره: کارتی را دارید که نماد زنجیره‌ای این کارت را دارد، پس آن را رایگان می‌سازید.',
  'hints.tip.guildBack':
    'پشت بنفش یک انجمن را پنهان می‌کند. انجمن‌ها در پایان بازی برای آنچه هر دو شهر دارند امتیاز می‌دهند؛ در هر بازی سه تا از هفت انجمن هست.',
  // The concept tips (M10.tutorial-client): a game concept, the first time it comes up.
  'hints.tip.track':
    'مهره از میانه بیرون رفت. سپرها آن را به سوی پایتخت دیگر می‌برند، و رسیدن به پایتخت همان دم پیروزی می‌آورد.',
  'hints.tip.science':
    'نخستین نماد علمی شما. جفتی از یک نماد یک نشان پیشرفت می‌گیرد، و شش نماد متفاوت همان دم پیروزی می‌آورد.',
  'hints.tip.newAge':
    'دورانی تازه: ساختاری تازه از کارت‌های قوی‌تر. بازیکنی که در مسیر نظامی عقب است انتخاب کرد چه کسی آن را آغاز کند، و دوران وقتی تمام می‌شود که آخرین کارتش برداشته شود.',
  'hints.tip.gotIt': 'فهمیدم',
  'hints.tip.more': 'بیشتر در قوانین',

  // The Settings screen's hint section (screens/Settings.svelte).
  'hints.settings.title': 'راهنماها',
  'hints.settings.tips': 'نکته‌های بار اول',
  'hints.settings.tipsHint':
    'هر نشانه‌ی راهنما را نخستین باری که روی این دستگاه ظاهر می‌شود توضیح می‌دهد.',
  'hints.settings.tipsReset': 'نمایش دوباره‌ی نکته‌ها',
  'hints.settings.tipsResetDone': 'نکته‌ها دوباره نمایش داده می‌شوند.',

  // The hint-settings badge (hud/TurnBar.svelte). {summary} is one of the
  // hints.summary.* lines.
  'hints.status.label': 'راهنماهای این بازی',
  'hints.status.tip': 'راهنماهای این بازی: {summary}. {setBy}',
  'hints.status.setByYou': 'می‌توانید از منوی بازی آن‌ها را تغییر دهید.',
  'hints.status.setByCreator': 'بازیکنی که بازی را ساخته آن‌ها را تعیین می‌کند.',

  // The rules reference's "Hint glyphs" section (rules/sections.ts,
  // components/hints/HintGlyphLegend.svelte). Under rules.* so the rules
  // sections can name it; it lives here with the other hint words.
  'rules.hints.title': 'نشانه‌های راهنما',
  'rules.hints.intro':
    'نشانه‌های راهنما واقعیت‌هایی را درباره‌ی یک کارت در وضع کنونی بازی بیان می‌کنند، و هرگز توصیه نیستند. هر کارت حداکثر یک نشانه در گوشه‌ی بالایش دارد.',
  'rules.hints.win': 'تاج: ساختن این کارت بازی را می‌برد.',
  'rules.hints.pair':
    'گل‌نقش نشان پیشرفت با علامت مساوی: ساختن این کارت یک جفت علمی را کامل می‌کند، پس سازنده‌اش یک نشان پیشرفت برمی‌دارد. توضیح شناورش نام نماد را می‌گوید.',
  'rules.hints.loot':
    'از دست رفتن سکه: سپرهای این کارت مهره را روی یک نشان غارت می‌برند، و بازیکن دیگر سکه از دست می‌دهد.',
  'rules.hints.own': 'توپر و طلایی: این واقعیت برای شماست.',
  'rules.hints.opp':
    'توخالی، به رنگ بازیکن دیگر: این واقعیت برای اوست، پس برداشتن یا کنار گذاشتن کارت آن را از او می‌گیرد.',
  'rules.hints.seat':
    'روی دستگاه مشترک، نشانه‌ی توپر رنگ بازیکنی را می‌گیرد که نوبت اوست، و تماشاگر رنگ خود هر بازیکن را می‌بیند.',
  'rules.hints.more':
    'نقطه‌ای کوچک کنار نشانه: کارت بیش از یک راهنما دارد. توضیح شناورش همه را فهرست می‌کند.',
  'rules.hints.review':
    'مشاور، اگر بازی روشنش کرده باشد، پس از بازی شما حرکت خودتان را بررسی می‌کند: خلاصه‌ی حرکت بعدی بازیکن دیگر می‌گوید آیا حرکت آشکارا بهتری در کار بود، و چرا. فقط شما آن را می‌بینید.',
  'rules.hints.sampleCard': 'یک کارت',

  // Board-level facts drawn in tooltips (M10.tooltip-coverage, view/tips.ts).
  // A face-down back (counting): {list} card names; a guild's worth to each
  // seat now, {first} and {second} hints.unseen.guildYou/Named for seats 0 and 1.
  'hints.unseen.pool': {
    one: 'این کارت {list} است.',
    other: 'یکی از {count} کارت: {list}.'
  },
  'hints.unseen.setAside': {
    other: '{count} کارت از آن‌ها در ساختار نیست.'
  },
  'hints.unseen.guild': '{card}: {first}، {second}.',
  'hints.unseen.guildYou': '{count} امتیاز برای شما',
  'hints.unseen.guildNamed': '{count} امتیاز برای {name}',
  // The coin tally (scoring).
  'hints.tally.nextVp': {
    other: '{count} سکه‌ی دیگر تا امتیاز بعدی از سکه‌ها.'
  },
  // The points tally (scoring): who would win on points if the game ended now.
  'hints.preview.leadYou': {
    other: 'اگر بازی اکنون تمام شود، با {count} امتیاز اختلاف می‌برید.'
  },
  'hints.preview.leadNamed': {
    other: 'اگر بازی اکنون تمام شود، {name} با {count} امتیاز اختلاف می‌برد.'
  },
  'hints.preview.blueYou': 'اکنون امتیازها برابر است: شما با امتیاز کارت‌های آبی می‌برید.',
  'hints.preview.blueNamed': 'اکنون امتیازها برابر است: {name} با امتیاز کارت‌های آبی می‌برد.',
  'hints.preview.shared': 'اکنون امتیازها و امتیاز کارت‌های آبی برابر است: پیروزی مشترک.',
  // The chain chips (counting): {place} a hints.place.* phrase.
  'hints.chain.statusYou': 'شما این نماد را دارید: {card} برای شما رایگان است. جایش: {place}.',
  'hints.chain.statusNamed':
    '{name} این نماد را دارد: {card} برای {name} رایگان است. جایش: {place}.',
  'hints.chain.saves': {
    other: 'این زنجیره امروز {count} سکه صرفه‌جویی می‌کند.'
  },
  // An owned guild (scoring): {city} is hints.guild.cityYou or cityNamed.
  'hints.guild.points': {
    other: '{card}: اکنون {count} امتیاز، با شمردن {city}.'
  },
  'hints.guild.cityYou': 'شهر شما',
  'hints.guild.cityNamed': 'شهر {name}'
} as const satisfies Translation<typeof en>;
