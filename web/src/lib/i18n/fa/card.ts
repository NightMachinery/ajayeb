// Persian messages for the wonder statuses a player sees on the draft and
// in the city panels. Partial: card, wonder and token names and the effect
// sentences stay English for now (they are keyed by engine id). See
// ./index.ts.

import type { card as en } from '../en/card';
import type { Translation } from '../types';

export const card = {
  'wonder.built': 'ساخته شده',
  'wonder.builtWith': 'ساخته شده با {card}',
  'wonder.removed': 'بیرون از بازی',
  'wonder.open': 'هنوز ساخته نشده',
  'wonder.notBuilt': 'ساخته نشده',
  'wonder.canBuild': 'اکنون ساختنی است',
  'wonder.takenBy': '{name} آن را برداشت'
} as const satisfies Translation<typeof en>;
