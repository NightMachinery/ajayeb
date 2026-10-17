// Persian messages for the card components: card and wonder names, the
// effect sentences, costs, affordability badges, tooltips and the ARIA
// descriptions. Complete, except the Roman age numerals, which stay as
// they are on purpose. The names and the reasons behind them are listed in
// docs/i18n/persian.md. See ./index.ts.

import type { card as en } from '../en/card';
import type { Translation } from '../types';

export const card = {
  'card.name.lumber-yard': 'انبار الوار',
  'card.name.stone-pit': 'گودال سنگ',
  'card.name.clay-pool': 'برکه‌ی رس',
  'card.name.logging-camp': 'اردوی هیزم‌شکنان',
  'card.name.quarry': 'معدن سنگ',
  'card.name.clay-pit': 'گودال رس',
  'card.name.glassworks': 'شیشه‌گری',
  'card.name.press': 'پاپیروس‌سازی',
  'card.name.stable': 'اصطبل',
  'card.name.garrison': 'پادگان',
  'card.name.palisade': 'حصار چوبی',
  'card.name.guard-tower': 'برج نگهبانی',
  'card.name.scriptorium': 'کتابت‌خانه',
  'card.name.workshop': 'کارگاه',
  'card.name.pharmacist': 'داروساز',
  'card.name.apothecary': 'عطاری',
  'card.name.tavern': 'مهمان‌خانه',
  'card.name.stone-reserve': 'ذخیره‌ی سنگ',
  'card.name.clay-reserve': 'ذخیره‌ی رس',
  'card.name.wood-reserve': 'ذخیره‌ی چوب',
  'card.name.theater': 'تماشاخانه',
  'card.name.altar': 'قربانگاه',
  'card.name.baths': 'گرمابه',
  'card.name.sawmill': 'چوب‌بری',
  'card.name.shelf-quarry': 'معدن پلکانی',
  'card.name.brickyard': 'آجرپزی',
  'card.name.glass-blower': 'کوره‌ی شیشه',
  'card.name.drying-room': 'خشک‌خانه',
  'card.name.horse-breeders': 'اسب‌پروری',
  'card.name.barracks': 'سربازخانه',
  'card.name.walls': 'بارو',
  'card.name.archery-range': 'میدان تیراندازی',
  'card.name.parade-ground': 'میدان رژه',
  'card.name.school': 'مدرسه',
  'card.name.laboratory': 'آزمایشگاه',
  'card.name.dispensary': 'داروخانه',
  'card.name.library': 'کتابخانه',
  'card.name.brewery': 'آبجوسازی',
  'card.name.forum': 'میدان شهر',
  'card.name.caravansery': 'کاروانسرا',
  'card.name.customs-house': 'گمرک',
  'card.name.temple': 'معبد',
  'card.name.statue': 'تندیس',
  'card.name.courthouse': 'دادگاه',
  'card.name.aqueduct': 'آبراهه',
  'card.name.rostrum': 'سکوی خطابه',
  'card.name.circus': 'سیرک',
  'card.name.arsenal': 'زرادخانه',
  'card.name.siege-workshop': 'کارگاه منجنیق',
  'card.name.fortifications': 'استحکامات',
  'card.name.pretorium': 'قرارگاه',
  'card.name.academy': 'آکادمی',
  'card.name.university': 'دانشگاه',
  'card.name.study': 'اتاق مطالعه',
  'card.name.observatory': 'رصدخانه',
  'card.name.arena': 'آمفی‌تئاتر',
  'card.name.chamber-of-commerce': 'اتاق بازرگانی',
  'card.name.port': 'بندر',
  'card.name.lighthouse': 'فانوس دریایی',
  'card.name.armory': 'اسلحه‌خانه',
  'card.name.palace': 'کاخ',
  'card.name.gardens': 'باغ‌ها',
  'card.name.pantheon': 'پانتئون',
  'card.name.town-hall': 'تالار شهر',
  'card.name.senate': 'مجلس سنا',
  'card.name.obelisk': 'ستون یادبود',
  'card.name.merchants-guild': 'انجمن بازرگانان',
  'card.name.shipowners-guild': 'انجمن کشتی‌داران',
  'card.name.builders-guild': 'انجمن سازندگان',
  'card.name.magistrates-guild': 'انجمن دادرسان',
  'card.name.scientists-guild': 'انجمن دانشمندان',
  'card.name.moneylenders-guild': 'انجمن صرافان',
  'card.name.tacticians-guild': 'انجمن سرداران',
  'wonder.name.pyramids': 'هرم‌های بزرگ',
  'wonder.name.colossus': 'غول‌پیکر رودس',
  'wonder.name.great-lighthouse': 'فانوس دریایی اسکندریه',
  'wonder.name.temple-of-artemis': 'معبد آرتمیس',
  'wonder.name.mausoleum': 'آرامگاه هالیکارناسوس',
  'wonder.name.great-library': 'کتابخانه‌ی بزرگ',
  'wonder.name.piraeus': 'پیرئوس',
  'wonder.name.hanging-gardens': 'باغ‌های معلق بابل',
  'wonder.name.statue-of-zeus': 'تندیس زئوس',
  'wonder.name.sphinx': 'ابوالهول',
  'wonder.name.appian-way': 'راه آپیا',
  'wonder.name.circus-maximus': 'سیرک ماکسیموس',

  // Colours, as adjectives for ARIA and effect sentences, and as categories.
  'card.color.brown': 'قهوه‌ای',
  'card.color.grey': 'خاکستری',
  'card.color.blue': 'آبی',
  'card.color.green': 'سبز',
  'card.color.yellow': 'زرد',
  'card.color.red': 'قرمز',
  'card.color.purple': 'بنفش',
  'card.category.brown': 'مواد خام',
  'card.category.grey': 'کالاهای ساخته‌شده',
  'card.category.blue': 'ساختمان مدنی',
  'card.category.green': 'ساختمان علمی',
  'card.category.yellow': 'ساختمان تجاری',
  'card.category.red': 'ساختمان نظامی',
  'card.category.purple': 'انجمن',
  'card.ageLabel': 'دوران {age}',

  // Resources, science symbols and chain symbols.
  'card.resource.wood': 'چوب',
  'card.resource.clay': 'خاک رس',
  'card.resource.stone': 'سنگ',
  'card.resource.glass': 'شیشه',
  'card.resource.papyrus': 'پاپیروس',
  'card.science.globe': 'کره',
  'card.science.scales': 'ترازو',
  'card.science.sundial': 'ساعت آفتابی',
  'card.science.mortar': 'هاون',
  'card.science.set-square': 'گونیا',
  'card.science.quill': 'قلم پر',
  'card.science.wheel': 'چرخ',
  'card.chain.horseshoe': 'نعل',
  'card.chain.sword': 'شمشیر',
  'card.chain.tower': 'برج',
  'card.chain.book': 'کتاب',
  'card.chain.gear': 'چرخ‌دنده',
  'card.chain.amphora': 'کوزه',
  'card.chain.mask': 'نقاب',
  'card.chain.moon': 'ماه',
  'card.chain.drop': 'قطره',
  'card.chain.target': 'هدف',
  'card.chain.helmet': 'کلاه‌خود',
  'card.chain.lyre': 'چنگ',
  'card.chain.lamp': 'چراغ',
  'card.chain.barrel': 'بشکه',
  'card.chain.sun': 'خورشید',
  'card.chain.column': 'ستون',
  'card.chain.temple-facade': 'معبد',

  // Effect sentences, one per effect kind of the catalog.
  'card.effect.produce': 'هر نوبت {n} {resource} تولید می‌کند',
  'card.effect.produceChoice': 'هر نوبت یک واحد {resources} تولید می‌کند',
  'card.effect.tradeFix': '{resources} را از بانک هر واحد یک سکه می‌خرید',
  'card.effect.shields': { other: '{count} سپر: مهره را {count} خانه پیش می‌برد' },
  'card.effect.science': 'نماد علمی: {symbol}',
  'card.effect.points': { other: '{count} امتیاز' },
  'card.effect.coins': { other: '{count} سکه بگیرید' },
  'card.effect.coinsPerCard': {
    other: 'برای هر کارت {colors} در شهرتان {count} سکه بگیرید'
  },
  'card.effect.coinsPerCardMax': {
    other: 'برای هر کارت {colors} در شهری که بیشترینش را دارد {count} سکه بگیرید'
  },
  'card.effect.coinsPerWonder': { other: 'برای هر شگفتی که ساخته‌اید {count} سکه بگیرید' },
  'card.effect.pointsPerCard': {
    other: 'در پایان، برای هر کارت {colors} در شهرتان {count} امتیاز'
  },
  'card.effect.pointsPerCardMax': {
    other: 'در پایان، برای هر کارت {colors} در شهری که بیشترینش را دارد {count} امتیاز'
  },
  'card.effect.pointsPerWonder': {
    other: 'در پایان، برای هر شگفتی که ساخته‌اید {count} امتیاز'
  },
  'card.effect.pointsPerWonderMax': {
    other: 'در پایان، برای هر شگفتی ساخته‌شده در شهری که شگفتی بیشتری دارد {count} امتیاز'
  },
  'card.effect.pointsPerCoins': {
    one: 'در پایان، برای هر سکه در ثروتمندترین شهر یک امتیاز',
    other: 'در پایان، برای هر {count} سکه در ثروتمندترین شهر یک امتیاز'
  },
  'card.effect.pointsPerToken': {
    other: 'در پایان، برای هر نشان پیشرفتی که دارید {count} امتیاز'
  },
  'card.effect.playAgain': 'بی‌درنگ یک نوبت دیگر بازی کنید',
  'card.effect.destroy': 'یکی از کارت‌های {colors} حریف را نابود کنید',
  'card.effect.buildFromDiscard': 'یکی از کارت‌های کنارگذاشته را رایگان بسازید',
  'card.effect.drawTokens': {
    one: '{count} نشان پیشرفت از جعبه بکشید و نگهش دارید',
    other: '{count} نشان پیشرفت از جعبه بکشید و یکی را نگه دارید'
  },
  'card.effect.opponentLosesCoins': { other: 'حریف {count} سکه از دست می‌دهد' },
  'card.effect.discountBlue': {
    other: 'ساختمان‌های آبی {count} واحد منبع کمتر می‌خواهند، به انتخاب شما'
  },
  'card.effect.discountWonder': {
    other: 'شگفتی‌ها {count} واحد منبع کمتر می‌خواهند، به انتخاب شما'
  },
  'card.effect.tradeIncome': 'سکه‌هایی که حریف برای خرید از بانک می‌پردازد به شما می‌رسد',
  'card.effect.extraShield': { other: 'هر ساختمان نظامی {count} سپر بیشتر می‌دهد' },
  'card.effect.wondersPlayAgain': 'هر شگفتی که می‌سازید یک نوبت دوباره به شما می‌دهد',
  'card.effect.chainBonus': {
    other: 'هر بار که از راه زنجیره می‌سازید، {count} سکه بگیرید'
  },
  'card.effect.none': 'بی‌اثر',
  'card.chainOut': 'نماد زنجیره‌ای {symbol}: {card} را رایگان می‌کند',
  'card.chainIn': 'رایگان با نماد زنجیره‌ای {symbol} از {card}',
  'card.chainInShort': 'رایگان با {symbol}',
  // Chain chip tooltips; the stack caption heads the linked card's preview.
  'card.tip.chainOut': 'نماد زنجیره‌ای {symbol}: ساختن این کارت {card} را رایگان می‌کند.',
  'card.tip.chainOutBare': 'نماد زنجیره‌ای {symbol}.',
  'card.tip.chainOutStack': 'رایگان از راه این زنجیره',
  'card.tip.chainIn': 'رایگان است اگر {card} در شهرتان باشد، که نماد زنجیره‌ای {symbol} را دارد.',
  'card.tip.chainInStack': 'نماد زنجیره‌ای را می‌دهد',
  // Tooltips of a card's micro-elements (view/tips.ts).
  'card.tip.card': '{card}. {category}.',
  'card.tip.costCoins': { other: '{count} سکه هزینه دارد که به بانک پرداخت می‌شود.' },
  'card.tip.costUnits': '{n} {resource} لازم است.',
  'card.tip.costMore': '{items} هم لازم است.',
  'card.tip.payOwn': 'تولید خود شهر: {n}.',
  'card.tip.payFlex': 'از یک کارت زرد یا یک شگفتی: {n}.',
  'card.tip.payDiscount': 'بخشوده به لطف {token}: {n}.',
  'card.tip.payTrade': { other: 'خرید از بانک: {n} واحد، هر واحد {count} سکه.' },
  'card.tip.badgeChain': 'رایگان: نماد زنجیره‌ای {card} در شهر هست.',
  'card.tip.badgeChainBare': 'رایگان از راه یک نماد زنجیره‌ای.',
  'card.tip.badgeFree': 'ساختنش رایگان است: چیزی نمی‌پردازید.',
  'card.tip.badgeOwn': {
    other: 'شهر همه‌ی منابع لازم را تولید می‌کند. {count} سکه بپردازید.'
  },
  'card.tip.badgeTrade': {
    other: 'منابعی را که کم دارید از بانک بخرید. روی هم {count} سکه بپردازید.'
  },
  'card.tip.tradeLine': { other: '{n} {resource}، هر واحد {count} سکه' },
  // Heads the trade rule's worked example in the badge's bubble.
  'card.tip.example': 'برای نمونه:',
  'card.tip.badgeShort': { other: '{coins} سکه هزینه دارد: {count} سکه کم دارید.' },
  'card.tip.backAge': 'یک کارت پشت‌وروی {back}.',
  'card.tip.backGuild': 'یک کارت انجمن پشت‌ورو.',

  // Costs.
  'card.cost.free': 'بی‌هزینه',
  'card.cost.label': 'هزینه: {items}',
  'card.cost.coins': { other: '{count} سکه' },
  'card.cost.units': '{n} {resource}',
  'card.cost.more': '+{n}',

  // Affordability badges: visible words, then the spoken forms.
  'card.badge.free': 'رایگان',
  'card.afford.chain': 'رایگان از راه {card}',
  'card.afford.chainNoCard': 'رایگان از راه زنجیره',
  'card.afford.free': 'ساخت رایگان',
  'card.afford.own': { other: 'ساخت با {count} سکه' },
  'card.afford.trade': { other: 'ساخت با {count} سکه، با خرید از بانک' },
  'card.afford.short': {
    other: 'ساختنی نیست: {count} سکه هزینه دارد، {shortfall} سکه بیش از دارایی شما'
  },
  'card.afford.waiting': 'نوبت شما نیست',

  // ARIA descriptions (view/describe.ts) and the structure.
  'card.aria.card': '{name}، کارت {color}. {effects}',
  'card.aria.available': 'در دسترس',
  'card.aria.covered': { other: 'پوشیده زیر {count} کارت' },
  'card.aria.faceDown': 'کارت پشت‌وروی {back}، که با آزاد شدن رو می‌شود',
  'card.aria.empty': 'جای خالی',
  'card.aria.selected': 'انتخاب‌شده',
  'card.structure.label': 'کارت‌های دوران {age}',
  'card.back.age1': 'دوران یکم',
  'card.back.age2': 'دوران دوم',
  'card.back.age3': 'دوران سوم',
  'card.back.guild': 'انجمن',

  // Wonders.
  'wonder.aria': '{name}، شگفتی. {effects}',
  'wonder.built': 'ساخته شده',
  'wonder.builtWith': 'ساخته شده با {card}',
  'wonder.removed': 'بیرون از بازی',
  'wonder.open': 'هنوز ساخته نشده',
  'wonder.notBuilt': 'ساخته نشده',
  'wonder.canBuild': 'اکنون ساختنی است',
  'wonder.takenBy': '{name} آن را برداشت',

  // The /dev/gallery cards section.
  'dev.cards.intro':
    'همه‌ی کارت‌ها، شگفتی‌ها و پشت کارت‌های فهرست در هر اندازه‌ی نمایش، پنج برچسب توان پرداخت از نماهای ضبط‌شده، و سه ساختار دوران‌ها از نماهای ضبط‌شده در اندازه‌ی گوشی و رایانه.',
  'dev.cards.tier': 'اندازه‌ی {tier}',
  'dev.cards.badges': 'برچسب‌های توان پرداخت',
  'dev.cards.badgesDim': 'نوبت شما نیست (کم‌رنگ)',
  'dev.cards.art': 'قاب‌های تصویر',
  'dev.cards.backs': 'پشت کارت‌ها',
  'dev.cards.wonders': 'شگفتی‌ها',
  'dev.cards.structures': 'ساختار دوران‌ها',
  'dev.cards.structureBox': '{fixture} در کادری {width}×{height}',
  'dev.cards.loading': 'در حال بارگیری نماهای ضبط‌شده…',
  'dev.cards.show': 'نمایش'
} as const satisfies Translation<typeof en>;
