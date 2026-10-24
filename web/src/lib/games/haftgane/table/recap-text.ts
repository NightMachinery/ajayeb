// The turn recap in words: each seat's sentence, each coin change, the
// seat and role changes, where hands went, and all of it as one text for
// the screen reader's live region. Every number is an event's own; the
// names come from the caller, who knows the seats and the people.

import { ageLabel, formatNumber } from '$lib/i18n';
import type { PlatformLogEntry } from '$lib/wire/platform.gen';
import { ht } from '../i18n/messages.svelte';
import { cardName } from './labels';
import type { CoinChange, Recap, RecapAct, RecapLine } from './recap';

const MINUS = '−';

/** A signed coin amount, "+3" or "−2", for a left-to-right isolate. */
export function signedCoins(n: number): string {
  return n > 0 ? `+${formatNumber(n)}` : n < 0 ? MINUS + formatNumber(-n) : formatNumber(0);
}

/** The coins a change adds to its seat (negative for a payment). */
export function coinAmount(c: CoinChange): number {
  return c.kind === 'bank' || c.kind === 'paid' ? -c.amount : c.amount;
}

function stageName(stage: number): string {
  return ht(
    `hg.screen.stageNumeral.${Math.min(Math.max(stage, 1), 4)}` as 'hg.screen.stageNumeral.1'
  );
}

function sideName(side: string): string {
  return side === 'night' ? ht('hg.screen.side.night') : ht('hg.screen.side.day');
}

/** What the seat did, as one sentence; `you` when it is the viewer's seat. */
export function actText(line: RecapLine, name: string, you: boolean): string {
  const a: RecapAct | null = line.act;
  if (!a) return you ? ht('hg.recap.you.idle') : ht('hg.recap.idle', { name });
  switch (a.kind) {
    case 'build':
      return you
        ? ht('hg.recap.you.build', { card: cardName(a.design) })
        : ht('hg.recap.build', { name, card: cardName(a.design) });
    case 'build-discarded':
      return you
        ? ht('hg.recap.you.buildDiscarded', { card: cardName(a.design) })
        : ht('hg.recap.buildDiscarded', { name, card: cardName(a.design) });
    case 'stage':
      return you && a.design
        ? ht('hg.recap.you.stageWith', { stage: stageName(a.stage), card: cardName(a.design) })
        : you
          ? ht('hg.recap.you.stage', { stage: stageName(a.stage) })
          : ht('hg.recap.stage', { name, stage: stageName(a.stage) });
    case 'sell':
      return you && a.design
        ? ht('hg.recap.you.sell', { card: cardName(a.design) })
        : you
          ? ht('hg.recap.you.sellHidden')
          : ht('hg.recap.sell', { name });
    case 'discard-nothing':
      return you ? ht('hg.recap.you.discardNothing') : ht('hg.recap.discardNothing', { name });
    case 'side':
      return you
        ? ht('hg.recap.you.side', { side: sideName(a.side) })
        : ht('hg.recap.side', { name, side: sideName(a.side) });
  }
}

/** The short label of a seat's reveal tag: a card's name, "Stage II", "Sold +3". */
export function tagText(line: RecapLine, sale: number | null): string {
  const a = line.act;
  if (!a) return '';
  switch (a.kind) {
    case 'build':
    case 'build-discarded':
      return cardName(a.design);
    case 'stage':
      return ht('hg.recap.tag.stage', { stage: stageName(a.stage) });
    case 'sell':
      return sale === null
        ? ht('hg.recap.tag.sold')
        : ht('hg.recap.tag.soldFor', { coins: signedCoins(sale) });
    case 'discard-nothing':
      return ht('hg.recap.tag.nothing');
    case 'side':
      return sideName(a.side);
  }
}

/** One coin change in words, from its seat's side ("paid Lena 2 coins"). */
export function coinText(c: CoinChange, nameOf: (seat: number) => string): string {
  switch (c.kind) {
    case 'bank':
      return ht('hg.recap.coin.bank', { count: c.amount });
    case 'paid':
      return ht('hg.recap.coin.paid', { count: c.amount, name: nameOf(c.seat) });
    case 'received':
      return ht('hg.recap.coin.received', { count: c.amount, name: nameOf(c.seat) });
    case 'sale':
      return ht('hg.recap.coin.sale', { count: c.amount });
    case 'card':
      return ht('hg.recap.coin.card', { count: c.amount, card: cardName(c.design) });
    case 'stage':
      return ht('hg.recap.coin.stage', { count: c.amount, stage: stageName(c.stage) });
  }
}

/** The extra lines of a seat at the end of an Age: a kept or discarded last card. */
export function lastCardText(line: RecapLine, name: string, you: boolean): string[] {
  const out: string[] = [];
  if (line.kept !== null) {
    out.push(
      you && line.kept
        ? ht('hg.recap.you.kept', { card: cardName(line.kept) })
        : ht('hg.recap.kept', { name })
    );
  }
  if (line.leftover) out.push(ht('hg.recap.you.leftover', { card: cardName(line.leftover) }));
  return out;
}

/** The recap's heading. */
export function recapTitle(r: Recap): string {
  if (r.turn) return ht('hg.recap.title.turn', { turn: r.turn.turn, age: ageLabel(r.turn.age) });
  if (r.ageEnd !== null) return ht('hg.recap.title.ageEnd', { age: ageLabel(r.ageEnd) });
  if (r.lines.some((l) => l.act?.kind === 'side')) return ht('hg.recap.title.sides');
  return ht('hg.recap.title');
}

/** A seat or role change in words; `userName` names the people the entry names. */
export function changeText(
  e: PlatformLogEntry,
  userName: (userId: string | undefined) => string
): string {
  const by = userName(e.by);
  const user = userName(e.userId);
  const seat = (e.seat ?? 0) + 1;
  switch (e.kind) {
    case 'seat':
      return e.control === 'human'
        ? ht('hg.recap.change.seated', { by, user, seat })
        : ht('hg.recap.change.toBot', { by, seat });
    case 'resign':
      return ht('hg.recap.change.resign', { by, seat });
    case 'role':
      if (e.role === 'admin') return ht('hg.recap.change.promote', { by, user });
      if (e.role === 'temporary') return ht('hg.recap.change.temporary', { user });
      return e.by === e.userId
        ? ht('hg.recap.change.stepDown', { user })
        : ht('hg.recap.change.demote', { by, user });
  }
  return '';
}

/** The closing lines: where hands went (and whose the viewer holds now), and what follows. */
export function closingText(r: Recap, held: { name: string; count: number } | null): string[] {
  const out: string[] = [];
  if (r.pass) {
    out.push(r.pass === 'left' ? ht('hg.recap.pass.left') : ht('hg.recap.pass.right'));
    if (held) out.push(ht('hg.recap.pass.hold', { name: held.name, count: held.count }));
  }
  if (r.ageEnd !== null) out.push(ht('hg.recap.ageEnd', { age: ageLabel(r.ageEnd) }));
  if (r.ageStart !== null) out.push(ht('hg.recap.ageStart', { age: ageLabel(r.ageStart) }));
  if (r.gameOver) out.push(ht('hg.recap.gameOver'));
  return out;
}

/** The whole recap as text, for the live region. */
export function recapSpeech(
  r: Recap,
  nameOf: (seat: number) => string,
  you: number | null,
  userName: (userId: string | undefined) => string,
  held: { name: string; count: number } | null
): string {
  const parts = [recapTitle(r) + '.'];
  for (const line of r.lines) {
    const mine = line.seat === you;
    const coins = line.coins.map((c) => coinText(c, nameOf));
    parts.push(
      [actText(line, nameOf(line.seat), mine), ...coins].join(', ') + '.',
      ...lastCardText(line, nameOf(line.seat), mine).map((s) => s + '.')
    );
  }
  for (const e of r.changes) parts.push(changeText(e, userName) + '.');
  parts.push(...closingText(r, held));
  return parts.join(' ');
}
