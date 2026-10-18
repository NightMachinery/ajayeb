# Persian (fa) glossary

The Persian locale was drafted by an agent and waits for a check by a native speaker. This page is for that check. It lists the house terms that every Persian string must use, the name of every card, wonder and progress token with its English source, the reason behind each choice where there was a real choice, and the open questions.

The tables live in `web/src/lib/i18n/fa/`, one file per area, next to the English ones in `web/src/lib/i18n/en/`. A key a Persian table leaves out falls back to English (see `web/src/lib/i18n/index.ts`). To change a term, change it here first, then in every file that uses it: `command grep -rn 'term' web/src/lib/i18n/fa/` finds them.

These are our own translations, made from our English text. Nothing was taken from a published Persian edition of the game ([../legal.md](../legal.md)).

## Writing conventions

- **Register.** The polite plural, as in a modern Persian board game or app: "روی کارت بزنید", "شما برنده شدید". Second person throughout, never the familiar "تو". The one exception is the invite a player sends a friend (`online.lobby.shareText`), which is informal on purpose.
- **Zero-width non-joiner (ZWNJ, U+200C).** Used where Persian needs it: the verb prefix می‌ (می‌سازید), the plural ‌ها (کارت‌ها), compound words (کنارگذاشته, پشت‌ورو), and the ezafe after a silent ه, written ‌ی (کتابخانه‌ی بزرگ, صفحه‌ی بازی). Never a plain space in those places.
- **Numbers.** A number that varies is a `{count}` or `{n}` placeholder, so `Intl` writes it in Persian digits. A fixed number that is part of a rule's wording is written as a word: "دو سکه", "شش نماد علمی متفاوت". No digits are typed by hand, Persian or Latin.
- **Signed changes.** Use `formatSigned()` for coin and point changes, preserving Persian digits and direction marks. Summary gains request a plus sign; losses use the typographic minus sign −. Price-change plurals follow the destination price.
- **Plurals.** Persian keeps the noun singular after a number ("۳ سکه"), so a plural message usually has only the `other` form. A `=0` form is added where zero reads better as words ("کارتی نمانده").
- **Quotation marks.** Guillemets: «ساخت یک شگفتی».
- **Names.** Player names, card names and the like arrive as `{placeholders}`, and the i18n code isolates them, so a Latin player name never reorders the Persian around it.
- **Possessives.** "{name}'s city" becomes "شهر {name}"; "your city" becomes "شهر شما".

## House terms

These are fixed. Every string in the locale uses them, and only them, for these concepts.

- **coin**: سکه. "1 coin per unit": "هر واحد ۱ سکه" (through `{count}`). The bank: بانک.
- **resource**: منبع (plural منابع). A resource unit: واحد.
  - wood: چوب
  - clay: خاک رس (in card names, رس alone)
  - stone: سنگ
  - glass: شیشه
  - papyrus: پاپیروس
  - raw materials (the brown category): مواد خام
  - manufactured goods (the grey category): کالاهای ساخته‌شده
- **card**: کارت. A building (a card in a city): ساختمان. To build: ساختن. Free: رایگان.
- **wonder**: شگفتی (plural شگفتی‌ها). The wonder draft: گزینش شگفتی‌ها.
- **progress token**: نشان پیشرفت. A token alone: نشان. A looting token: نشان غارت. Because نشان means token everywhere, the player's seat emblem is نماد (see below), never نشان.
- **the military track**: مسیر نظامی. The conflict pawn: مهره (in full مهره‌ی نظامی). A space on the track: خانه. A shield: سپر. A capital: پایتخت. Looting: غارت.
- **science symbol**: نماد علمی. A pair of the same symbol: جفت. "Six different science symbols": شش نماد علمی متفاوت.
- **chain**: زنجیره. A chain symbol: نماد زنجیره‌ای. "Free through a chain": رایگان از راه زنجیره.
- **age**: دوران. Every displayed age uses an ordinal word: یکم, دوم, سوم ("دوران یکم"), in place of the English Roman numerals I, II and III. The shared `ageLabel()` helper reads `card.ageNumeral.*` for bars, dialogs, summaries, logs, hints, card sheets and accessible labels. Server age values remain numeric.
- **the card structure**: ساختار (the layout of an age's cards). Face up: رو. Face down: پشت‌ورو. To uncover: آزاد کردن. Covered: پوشیده.
- **discard**: کنار گذاشتن (for coins). The discard pile: کارت‌های کنارگذاشته.
- **tuck a card under a wonder**: زیر شگفتی گذاشتن.
- **victory point (VP)**: امتیاز. "Blue points" (the tie-break): امتیاز کارت‌های آبی.
- **the three victories**: military supremacy برتری نظامی, scientific supremacy برتری علمی, civilian victory پیروزی مدنی. A shared victory: پیروزی مشترک.
- **turn**: نوبت. An extra turn (play again): نوبت دوباره; "Play again" as a card effect: "بی‌درنگ یک نوبت دیگر بازی کنید"; the turn bar's state: "دوباره نوبت شماست".
- **trade** (buying a resource from the bank): خرید از بانک; the trading price: قیمت خرید.
- **guild**: انجمن (the purple cards).
- **player**: بازیکن. Opponent: حریف. Spectator: تماشاگر. The player who starts an age: آغازکننده.
- **seat emblem** (Sun, Moon): نماد. Sun: خورشید. Moon: ماه.
- **hint**: راهنما. A first-time tip: نکته. The advisor: مشاور.
- **undo**: برگرداندن. Forfeit: واگذاری (to forfeit: واگذار کردن). Replay (a game record): بازپخش. Log: گزارش.
- **game code** (online): کد بازی. Invite link: پیوند دعوت.
- **bot difficulty**: Apprentice شاگرد, Scholar دانشمند, Master استاد.
- **bot**: ربات.
- **destroy** (a card, by a wonder): نابود کردن.
- **cancel**: لغو.
- **the game screen** (as a whole): نمای بازی; the board itself: صفحه‌ی بازی.
- **cannot afford**: توان پرداختش را ندارید.
- **at once** (an immediate win or effect): بی‌درنگ.
- **accessible card** (one no other card covers): کارت در دسترس. To cover: پوشاندن.
- **the box** (unused tokens and wonders): جعبه.
- **cards set aside unseen** at the start of an age: بیرون می‌ماند (not کنار گذاشتن, which is the discard action).
- **the weaker player** (who chooses the next starter): بازیکن ضعیف‌تر.
- **a zone of the military track**: منطقه.
- **the draft rounds**: دور یکم, دور دوم.
- **the game board** (where the progress tokens lie): صفحه‌ی بازی.
- **scoring**: امتیازشماری.
- **seat**: جایگاه. **sheet** (a panel that slides in): برگه. **toast**: اعلان. **room** (online, debug only): اتاق.
- **copy** (a link): رونوشت گرفتن. **reset**: بازنشانی.
- **tooltip**: توضیح شناور. A hint mark or glyph on a card: نشانه (never نشان). A badge: برچسب.
- **in view** (visible on the board): نمایان; unseen: ندیده.
- **a later age's deck**: دسته‌ی کارت‌های دوران {age}.
- **trademark**: علامت تجاری (not نشان تجاری, so that نشان only means a token).

## Card colours

- brown: قهوه‌ای
- grey: خاکستری
- blue: آبی
- green: سبز
- yellow: زرد
- red: قرمز
- purple: بنفش

Categories: raw materials مواد خام, manufactured goods کالاهای ساخته‌شده, civilian building ساختمان مدنی, scientific building ساختمان علمی, commercial building ساختمان تجاری, military building ساختمان نظامی, guild انجمن.

## Science symbols

- globe: کره
- scales: ترازو
- sundial: ساعت آفتابی
- mortar and pestle: هاون
- set square: گونیا
- quill: قلم پر
- wheel: چرخ

## Chain symbols

- horseshoe: نعل
- sword: شمشیر
- tower: برج
- book: کتاب
- gear: چرخ‌دنده
- amphora: کوزه
- mask: نقاب
- moon: ماه
- drop: قطره
- target: هدف
- helmet: کلاه‌خود
- lyre: چنگ
- lamp: چراغ
- barrel: بشکه
- sun: خورشید
- column: ستون
- temple: معبد

## Card names

Names are short, because a card face is small, and name the building in plain Persian. Where a classical Persian word fits (گرمابه, عطاری, رصدخانه, کاروانسرا) it is preferred to a loan word.

Age I:

- Lumber Yard: انبار الوار
- Stone Pit: گودال سنگ
- Clay Pool: برکه‌ی رس
- Logging Camp: اردوی هیزم‌شکنان
- Quarry: معدن سنگ
- Clay Pit: گودال رس
- Glassworks: شیشه‌گری
- Press: پاپیروس‌سازی
- Stable: اصطبل
- Garrison: پادگان
- Palisade: حصار چوبی
- Guard Tower: برج نگهبانی
- Scriptorium: کتابت‌خانه
- Workshop: کارگاه
- Pharmacist: داروساز
- Apothecary: عطاری
- Tavern: مهمان‌خانه
- Stone Reserve: ذخیره‌ی سنگ
- Clay Reserve: ذخیره‌ی رس
- Wood Reserve: ذخیره‌ی چوب
- Theater: تماشاخانه
- Altar: قربانگاه
- Baths: گرمابه

Age II:

- Sawmill: چوب‌بری
- Shelf Quarry: معدن پلکانی
- Brickyard: آجرپزی
- Glass-Blower: کوره‌ی شیشه
- Drying Room: خشک‌خانه
- Horse Breeders: اسب‌پروری
- Barracks: سربازخانه
- Walls: بارو
- Archery Range: میدان تیراندازی
- Parade Ground: میدان رژه
- School: مدرسه
- Laboratory: آزمایشگاه
- Dispensary: داروخانه
- Library: کتابخانه
- Brewery: آبجوسازی
- Forum: میدان شهر
- Caravansery: کاروانسرا
- Customs House: گمرک
- Temple: معبد
- Statue: تندیس
- Courthouse: دادگاه
- Aqueduct: آبراهه
- Rostrum: سکوی خطابه

Age III:

- Circus: سیرک
- Arsenal: زرادخانه
- Siege Workshop: کارگاه منجنیق
- Fortifications: استحکامات
- Pretorium: قرارگاه
- Academy: آکادمی
- University: دانشگاه
- Study: اتاق مطالعه
- Observatory: رصدخانه
- Arena: آمفی‌تئاتر
- Chamber of Commerce: اتاق بازرگانی
- Port: بندر
- Lighthouse: فانوس دریایی
- Armory: اسلحه‌خانه
- Palace: کاخ
- Gardens: باغ‌ها
- Pantheon: پانتئون
- Town Hall: تالار شهر
- Senate: مجلس سنا
- Obelisk: ستون یادبود

Guilds:

- Merchants Guild: انجمن بازرگانان
- Shipowners Guild: انجمن کشتی‌داران
- Builders Guild: انجمن سازندگان
- Magistrates Guild: انجمن دادرسان
- Scientists Guild: انجمن دانشمندان
- Moneylenders Guild: انجمن صرافان
- Tacticians Guild: انجمن سرداران

Why, where there was a choice:

- **Clay Pool, Clay Pit, Clay Reserve** use رس, the short form of the resource خاک رس.
- **Glassworks and Glass-Blower** are both glass producers (Age I and Age II), so their names must not look alike: شیشه‌گری (the craft and its shop) and کوره‌ی شیشه (the glass furnace). شیشه‌گر was rejected as too close to شیشه‌گری.
- **Press** makes papyrus, so پاپیروس‌سازی says what it does. منگنه or پرس would read as a modern machine.
- **Lumber Yard and Wood Reserve** both hold wood: انبار الوار for the brown producer and ذخیره‌ی چوب for the yellow trading card, so the three Reserves share one pattern (ذخیره‌ی ...).
- **Apothecary** is عطاری, the traditional Persian herbalist's shop; **Pharmacist** داروساز and **Dispensary** داروخانه stay distinct from it.
- **Tavern** is مهمان‌خانه (an inn), not میخانه, to keep the game family-friendly.
- **Altar** is قربانگاه; محراب was rejected because it names the niche of a mosque.
- **Pretorium** is قرارگاه, a commander's headquarters, which is what a praetorium was.
- **Circus** is سیرک, the usual Persian name for a Roman circus (as in سیرک ماکسیموس), though in modern Persian سیرک also means a travelling circus.
- **Arena** is آمفی‌تئاتر, distinct from تماشاخانه (Theater).
- **Guild** is انجمن throughout, already used by the score table. صنف is more exact for a trade guild but reads oddly for scientists or tacticians.

## Wonder names

A historical wonder takes its usual Persian name.

- Pyramids: هرم‌های بزرگ
- Colossus: غول‌پیکر رودس
- Great Lighthouse: فانوس دریایی اسکندریه
- Temple of Artemis: معبد آرتمیس
- Mausoleum: آرامگاه هالیکارناسوس
- Great Library: کتابخانه‌ی بزرگ
- Piraeus: پیرئوس
- Hanging Gardens: باغ‌های معلق بابل
- Statue of Zeus: تندیس زئوس
- Sphinx: ابوالهول
- Appian Way: راه آپیا
- Circus Maximus: سیرک ماکسیموس

Why, where there was a choice:

- **Sphinx** is ابوالهول, the name Persian uses for the Sphinx of Giza.
- **Great Library** keeps کتابخانه‌ی بزرگ, as the English keeps "Great Library" rather than "Library of Alexandria"; the Great Lighthouse keeps اسکندریه because فانوس دریایی alone is the Age III card Lighthouse.
- **Mausoleum** is آرامگاه هالیکارناسوس, its usual Persian name. A hint that says "your Mausoleum" says "آرامگاه شما".
- **Statue of Zeus** uses تندیس, the same word as the Age II card Statue (تندیس), which is the usual word for a monumental statue.

## Progress tokens

- Agriculture: کشاورزی
- Architecture: معماری
- Economy: اقتصاد
- Law: قانون
- Masonry: بنایی
- Mathematics: ریاضیات
- Philosophy: فلسفه
- Strategy: راهبرد
- Theology: الهیات
- Urbanism: شهرسازی

## Kept in English on purpose

No key falls back to English. Within the Persian table:

- `app.name` stays "Ajayeb": Home shows `app.nameNative` (عجایب) right above it. `home.settings.languageEn` stays "English", because each language is named in itself.
- Inside translated strings, proper nouns and codes stay in Latin script: the game's own name and its publishers in `app.legal` (7 Wonders Duel, Repos Production, Asmodee), GitHub, MIT and QR, the key name Esc, the letters and digits a game code never uses (`online.join.invalid`), and the developer-only paths and URL parameters (`dev.gallery.intro`, `game.gallery.intro`, `dev.new.badMode`). A Latin run that starts with a digit needs a left-to-right isolate (U+2066 ... U+2069) around it, or the bidi algorithm moves the digit to its other end: `app.legal` wraps "7 Wonders Duel" that way.
- Strings with no words keep their English form: `card.cost.more` and `hints.badge.bonus` ("+{n}", "+{count}"), `hints.cardWith`, `hints.after.byMove`, `hints.science.symbol`, `hints.science.card` and the diagram signs `rules.diagram.discard.plus` and `.equals`. They are still listed in the Persian table, so they do not fall back.

Everything else is translated, the `/dev/gallery` and demo strings included.

## Player-facing wording

- Pawn event lines use the destination player's display name in both locales (`board.log.pawn`). The Persian sentence is «مهره‌ی نظامی به سوی {name} رفت.»; the seat emblem is only a fallback when that seat has no name.
- Outcome score changes are **total VP**, امتیاز کل, including coin points: «{delta} امتیاز کل (با احتساب سکه‌ها)». The phone form is «{delta} امتیاز کل». The named opponent form uses the same explanation after «{name}:». The trading lesson explains «پیش‌نمایش امتیاز کل، امتیاز سکه‌ها را هم حساب می‌کند، پس خرج کردن سکه می‌تواند آن را کم کند.» No card-versus-coin split is inferred by the client.
- A crown means a win **if the card is built**, «اگر آن را بسازید، بازی را می‌برید», without اکنون: a covered card can carry the same mark.
- A shortfall note says **another card**, کارتی دیگر, and makes the later build conditional on availability and unchanged prices: «با کنار گذاشتن کارتی دیگر (+{count} سکه)، نوبت بعد می‌توانید این را بسازید، اگر هنوز اینجا باشد و قیمت‌ها عوض نشوند.»
- Masonry's savings name **blue cards in view**, کارت‌های آبی نمایان. Architecture's name **wonders in view**, شگفتی‌های نمایان. Both positive and zero-saving messages keep this scope. The card producer's broader savings message still covers cards and wonders.
- Resource-chip prices reuse the pluralized coin message (سکه), separately from the production count: «{resource}: تولید {count}، خرید هر واحد {coins}». The English price is therefore “one coin” or “two coins” according to `Intl.PluralRules`.
- The reveal lesson says **the cards it was covering**, «کارت‌هایی را که می‌پوشاند», rather than referring to a direction on screen.
- A zero-coin looting result is omitted from card pips, card details and move outcome lines. The pawn movement remains visible.

## Open questions for the native check

- **Copy**: رونوشت is formal; کپی is what most apps say.
- **Tooltip** (developer pages only): راهنمای شناور collides a little with راهنما (hint).
- **"at once"**: بی‌درنگ is used everywhere; همان دم was dropped for consistency. Is بی‌درنگ too literary for a game?
- **Clay**: خاک رس is exact but two words. گِل is shorter, but without its vowel mark it reads as گُل (flower). خشت (mud brick) is crisp, but it is a made thing, not the raw material. Which reads best on a resource chip?
- **Guild**: انجمن or صنف?
- **Strategy**: راهبرد or the loan word استراتژی, which many players use?
- **Academy**: آکادمی (loan) or دانش‌سرا (which today means a teacher-training college)?
- **Aqueduct**: آبراهه, or قنات (the Persian underground channel, a different structure but a nice local touch)?
- **Logging Camp**: اردوی هیزم‌شکنان is long for a card face. Is there a shorter natural name?
- **Forum**: میدان شهر is plain; is there a better word for a Roman forum?
- **Colossus**: غول‌پیکر رودس or مجسمه‌ی رودس?
- **Appian Way**: راه آپیا or جاده‌ی آپیان?
- **Seat emblem**: نماد now names both the seat emblem and the science and chain symbols. The context always tells them apart, but a native reader may prefer another word for the seat (for example پرچم, flag).
- **Ezafe after a silent ه**: the tables write it ‌ی after a zero-width non-joiner (کتابخانه‌ی بزرگ, صفحه‌ی بازی), as many apps do. The Academy's style writes ـهٔ (کتابخانهٔ بزرگ) or leaves the ezafe unwritten. Which should the locale use?
- **Waiting lines**: the lines shown while the other player chooses read «منتظر بمانید تا {name} ... برگزیند…». Is a shorter form («{name} در حال انتخاب است…») more natural on a small screen?
- **Draft round**: `choices.draft.round` reads «دور {round} از دو», a Persian digit followed by a word, because no digit is typed by hand. Is «دور یکم» / «دور دوم» alone better there?

## Player-flow messages (M11.player-flow)

- “While you were away” is «وقتی نبودید», the short label above opponent moves
  received while the page was away (`game.summary.away`). It describes the
  missing interval without implying a network failure.
- “Undid your last move.” is «آخرین حرکت شما برگردانده شد.» (`game.undo.done`).
  The same toast provides the visible confirmation and polite announcement.
- The Great Lighthouse lesson says «این شگفتی در هر نوبت، به انتخاب شما چوب،
  خاک رس یا سنگ تولید می‌کند تا هزینه‌ی ساخت‌های بعدی را آسان‌تر بپردازید.»
  (`tutorial.action.buildWonder`). It explains choosing one raw material for
  each turn, following the house terms for production, wood, clay and stone.


## Completed-age controls (M11.age-summary-stays)

- Continue is «ادامه» (`choices.age.continue`), the local dismissal of the
  completed-age summary while another player may still be choosing. It sends
  no game action. Once the starter has been chosen, the existing Begin Age N
  wording names the age explicitly in both locales.
- Result actions retain their existing full Persian labels in the reserved
  footer. They may wrap within their buttons on a short screen rather than
  being shortened or moved below the score details.
