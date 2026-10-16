// Persian messages for online play: the lobby, presence and the self mark.
// Partial; the online sheets fall back to English. See ./index.ts.

import type { online as en } from '../en/online';
import type { Translation } from '../types';

export const online = {
  'online.lobby.title': 'دوستتان را دعوت کنید',
  'online.lobby.intro': 'پیوند یا کد را برایش بفرستید. بازی با پیوستن او آغاز می‌شود.',
  'online.lobby.code': 'کد بازی',
  'online.lobby.link': 'پیوند دعوت',
  'online.lobby.copy': 'رونوشت پیوند',
  'online.lobby.copied': 'پیوند رونوشت شد.',
  'online.lobby.share': 'هم‌رسانی',
  'online.lobby.shareTitle': 'یک دست عجایب',
  'online.lobby.shareText': 'به بازی عجایب من بپیوند. کد بازی {code} است.',
  'online.lobby.qr': 'رمزینه‌ی QR پیوند دعوت',
  'online.lobby.scan': 'یا این را با گوشی دیگر اسکن کنید.',
  'online.lobby.players': 'بازیکنان',
  'online.lobby.waiting': 'در انتظار دوستتان…',
  'online.lobby.home': 'خانه',
  'online.lobby.homeHint': 'بازی باز می‌ماند: از خانه ادامه‌اش دهید.',

  'online.presence.connected': 'آنلاین',
  'online.presence.away': 'غایب',
  'online.presence.connectedLabel': '{name} آنلاین است',
  'online.presence.awayLabel': '{name} غایب است',

  'online.identity.you': 'شما',
  'online.identity.youLabel': 'این شمایید'
} as const satisfies Translation<typeof en>;
