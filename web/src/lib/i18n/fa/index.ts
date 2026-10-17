// The Persian (fa) message table. Each area file here translates its English
// area (../en/<area>.ts), typed by Translation so a misspelt or stale key is
// a type error. Every key is translated (the glossary is
// docs/i18n/persian.md); t() falls back to English per key only for a key
// added to English later.

import type { PartialLocaleTable } from '../index';
import { board } from './board';
import { card } from './card';
import { choices } from './choices';
import { common } from './common';
import { game } from './game';
import { hints } from './hints';
import { home } from './home';
import { online } from './online';
import { rules } from './rules';
import { session } from './session';
import { shell } from './shell';
import { tutorial } from './tutorial';

export const fa: PartialLocaleTable = {
  ...common,
  ...session,
  ...shell,
  ...card,
  ...board,
  ...home,
  ...game,
  ...choices,
  ...online,
  ...hints,
  ...rules,
  ...tutorial
};
