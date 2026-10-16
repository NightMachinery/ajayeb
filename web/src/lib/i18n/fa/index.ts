// The Persian (fa) message table: a stub, not a full translation. Each area
// file here translates a subset of its English area (../en/<area>.ts), typed
// by Translation so a misspelt or stale key is a type error; every key left
// out falls back to English in t(), per key. Card, wonder and token names
// are keyed by engine id and stay English for now.

import type { PartialLocaleTable } from '../index';
import { board } from './board';
import { common } from './common';
import { game } from './game';
import { hints } from './hints';
import { home } from './home';
import { rules } from './rules';
import { session } from './session';
import { shell } from './shell';

export const fa: PartialLocaleTable = {
  ...common,
  ...session,
  ...shell,
  ...board,
  ...home,
  ...game,
  ...hints,
  ...rules
};
