// The Persian (fa) table of Haftgane's messages. It is empty until
// M18.persian-messages translates them, with the house terms of
// docs/i18n/persian.md; until then ht() falls back to English per key
// (../messages.svelte.ts). Translations go in one file per area, beside the
// English ones in ../en/, spread in here.

import type { HgPartialTable } from '../messages.svelte';

export const fa: HgPartialTable = {};
