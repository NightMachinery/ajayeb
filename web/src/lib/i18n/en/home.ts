// English messages for the Home and setup screens (components/screens/Home,
// NewGameSheet, HotseatSheet, NamePrompt...). The existing 'home.*' keys stay
// in common.ts, which the app shell already uses; a later slice (M5.S05)
// moves them here and fills in the rest. See ../index.ts for the conventions.

import type { Messages } from '../types';

export const home = {} as const satisfies Messages;
