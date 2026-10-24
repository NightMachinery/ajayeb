// The Persian (fa) table of Haftgane's messages. Each area file here
// translates its English area (../en/<area>.ts), typed by Translation so a
// misspelt or stale key is a type error, in the house terms of
// docs/i18n/persian.md ("Haftgane"). Every key is translated, and
// ../messages.test.ts fails on a key English has and Persian lacks; ht()
// falls back to English per key only until then (../messages.svelte.ts).
// The table loads on first use in Persian, in its own lazy chunk.

import type { HgPartialTable } from '../messages.svelte';
import { admin } from './admin';
import { city } from './city';
import { end } from './end';
import { names } from './names';
import { recap } from './recap';
import { rules } from './rules';
import { screen } from './screen';
import { table } from './table';

export const fa: HgPartialTable = {
  ...admin,
  ...city,
  ...end,
  ...names,
  ...recap,
  ...rules,
  ...screen,
  ...table
};
