// Persian messages for the shell primitives (components/shell/*) and the
// /dev/gallery demo and tooltip pages. Complete: every key of
// ../en/shell.ts, none left in English. See ./index.ts.

import type { shell as en } from '../en/shell';
import type { Translation } from '../types';

export const shell = {
  'shell.close': 'بستن',
  'shell.dismiss': 'بستن',
  // The link at the end of a tooltip that opens the rules at its topic.
  'shell.tooltip.moreRules': 'بیشتر در قوانین',
  // {label} is the action's own text, e.g. "Build for 4 coins".
  'shell.confirmArmed': 'تأیید: {label}',
  'shell.tapAgainToConfirm': 'برای تأیید دوباره بزنید',
  // OfflineNotice: the browser is offline on Home, the rules or Settings.
  'shell.offline': 'آفلاین هستید. بازی‌ها به اتصال نیاز دارند.',

  // /dev/gallery demo strings for the primitives in this file's folder.
  'shell.demo.openSheet': 'باز کردن یک برگه',
  'shell.demo.sheetTitle': 'برگه‌ی نمونه',
  'shell.demo.sheetBody':
    'یک پنجره‌ی بومی: کلید Esc، ضربه روی پس‌زمینه یا دستگیره‌ی کشیدن آن را می‌بندد و تمرکز به دکمه‌ای که بازش کرد برمی‌گردد.',
  'shell.demo.showToast': 'نمایش یک اعلان',
  'shell.demo.toastMessage': 'انبار الوار ساخته شد',
  'shell.demo.toastUndo': 'برگرداندن',
  'shell.demo.segmentedLegend': 'آغازکننده',
  'shell.demo.optionRandom': 'تصادفی',
  'shell.demo.optionMe': 'من',
  'shell.demo.optionRival': 'حریف',
  'shell.demo.confirmLabel': 'ساختن با چهار سکه',
  'shell.demo.announce': 'اعلام یک رویداد',
  'shell.demo.announced': 'وزیر انبار الوار را ساخت. نوبت شماست.',

  // /dev/gallery/tooltips: the tooltip layer, the stacked tooltip and the
  // dismiss layer (TooltipLayer, TooltipBubble, TooltipStack, DismissLayer).
  'dev.tooltips.title': 'توضیح‌های شناور',
  'dev.tooltips.intro':
    'اشاره‌گر را روی یک نماد نگه دارید (پس از درنگی کوتاه) یا با صفحه‌کلید به آن بروید. در صفحه‌ی لمسی روی نماد بزنید، یا اگر کاری انجام می‌دهد، انگشتتان را رویش نگه دارید. هر ضربه توضیح شناور باز را می‌بندد و کار دیگری نمی‌کند.',
  'dev.tooltips.back': 'بازگشت به نگارخانه',
  'dev.tooltips.link': 'نگارخانه‌ی توضیح‌های شناور',
  'dev.tooltips.text': 'متن',
  'dev.tooltips.params': 'با پارامتر',
  'dev.tooltips.diagram': 'با نمودار',
  'dev.tooltips.stack': 'روی هم: نمادهای زنجیره‌ای',
  'dev.tooltips.inCard': 'برچسبی درون کارتی که کاری انجام می‌دهد',
  'dev.tooltips.button': 'نگه داشتن انگشت روی دکمه',
  'dev.tooltips.specimens': 'حباب و پشته، به‌تنهایی',
  'dev.tooltips.micro': 'اجزای ریز صفحه‌ی بازی',
  'dev.tooltips.statusLine': 'یک سطر وضعیت: واقعیتی که سرور فرستاده است.',
  'dev.tooltips.dismiss': 'لایه‌ی بستن',
  'dev.tooltips.openDismiss': 'باز کردن یک پنل شناور',
  'dev.tooltips.dismissBody': 'هر ضربه این پنل را می‌بندد و کار دیگری نمی‌کند.',
  'dev.tooltips.cardOpened': 'برگه‌ی کارت {count} بار باز شد',
  'dev.tooltips.pressed': '{count} بار فشرده شد',
  'dev.tooltips.pressMe': 'ساختن',
  'dev.tooltips.pressTip': 'کارت را با پرداخت هزینه‌اش می‌سازد.',
  'dev.tooltips.laurelTip': 'امتیاز: در پایان بازی شمرده می‌شود.',
  'dev.tooltips.diagramTip': 'مهره‌ی نظامی به ازای هر سپر یک خانه به سوی پایتخت حریف می‌رود.',
  'dev.tooltips.diagramYou': 'پایتخت شما',
  'dev.tooltips.diagramRival': 'پایتخت حریف',
  'dev.tooltips.sheetTitle': 'برگه‌ی کارت',
  'dev.tooltips.sheetBody': 'برگه‌ی کارت اینجا باز می‌شد.'
} as const satisfies Translation<typeof en>;
