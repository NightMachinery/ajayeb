import type { lobby as en } from './picker-en';
import type { Translation } from '$lib/i18n/types';

export const lobby = {
  'lobby.duel': 'دوئل',
  'lobby.haftgane': 'هفتگانه',
  'lobby.chooseGame': 'بازی را انتخاب کنید',
  'lobby.duelHint': 'دو بازیکن',
  'lobby.haftganeHint': 'سه تا هفت بازیکن',
  'lobby.new': 'میز جدید',
  'lobby.newHint': 'دوستان را دعوت کنید، ربات اضافه کنید یا هر دو.'
} as const satisfies Translation<typeof en>;
