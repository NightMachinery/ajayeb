// The turn recap over the engine's own view fixtures
// (server/internal/haftgane/rules/testdata/views): a seven-seat end of
// Age II seen from seat 2, with trades, sales, a kept and a discarded last
// card and a card under a wonder; the game's last batch seen from seat 0
// and by a spectator. Every coin is an event's own number, and a card the
// viewer may not know stays unnamed.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import type { Table } from '$lib/wire/haftgane.gen';
import type { PlatformLogEntry, TableEvent } from '$lib/wire/platform.gen';
import { buildRecap, dealOf, latestRecap, recapWorthy, ringOrder, saleOf } from './recap';
import {
  actText,
  changeText,
  closingText,
  coinText,
  lastCardText,
  recapSpeech,
  recapTitle,
  tagText
} from './recap-text';

function fixture(name: string): Table {
  return JSON.parse(
    readFileSync(
      new URL(
        `../../../../../../server/internal/haftgane/rules/testdata/views/${name}.json`,
        import.meta.url
      ),
      'utf8'
    )
  ) as Table;
}

const asTable = (events: Table['events']): TableEvent[] => events.map((e) => ({ ...e, pass: '' }));
const discard = asTable(fixture('discard-seat').events);
const overSeat = asTable(fixture('over-hidden-seat').events);
const overWatch = asTable(fixture('over-reveal-spectator').events);
const blank: TableEvent = { ...discard[0]!, kind: '', seat: 0, design: '', pass: '' };
const names = ['Ana', 'Bo', 'Cy', 'Dee', 'Eli', 'Fay', 'Gus'];
const nameOf = (seat: number) => names[seat]!;

describe('the recap of a batch', () => {
  it('lists every seat in ring order from the viewer, with what it did', () => {
    const r = buildRecap({ batch: 9, events: discard }, 7, 2, 0, []);
    expect(ringOrder(7, 2)).toEqual([2, 3, 4, 5, 6, 0, 1]);
    expect(r.lines.map((l) => l.seat)).toEqual([2, 3, 4, 5, 6, 0, 1]);
    const at = (seat: number) => r.lines.find((l) => l.seat === seat)!;
    expect(at(2).act).toEqual({ kind: 'stage', stage: 1, design: 'statue' });
    expect(at(1).act).toEqual({ kind: 'build', design: 'dispensary' });
    expect(at(3).act).toEqual({ kind: 'sell', design: '' });
    expect(r).toMatchObject({ batch: 9, turn: null, ageEnd: null, pass: null, gameOver: false });
  });

  it('reads each coin from its own event: payments, receipts by the payer, sales', () => {
    const r = buildRecap({ batch: 9, events: discard }, 7, 2, 0, []);
    const coins = (seat: number) => r.lines.find((l) => l.seat === seat)!.coins;
    // Seat 2 paid its right neighbour, seat 1, for its stage.
    expect(coins(2)).toEqual([{ kind: 'paid', seat: 1, amount: 2 }]);
    // Seat 1 paid seat 0 four and was paid two by seat 2.
    expect(coins(1)).toEqual([
      { kind: 'paid', seat: 0, amount: 4 },
      { kind: 'received', seat: 2, amount: 2 }
    ]);
    expect(coins(0)).toEqual([
      { kind: 'received', seat: 1, amount: 4 },
      { kind: 'sale', amount: 3 }
    ]);
    expect(coins(4)).toEqual([]);
    expect(saleOf(r.lines.find((l) => l.seat === 3)!)).toBe(3);
  });

  it('names a last card for its owner only', () => {
    const r = buildRecap({ batch: 9, events: discard }, 7, 2, 0, []);
    const at = (seat: number) => r.lines.find((l) => l.seat === seat)!;
    expect(at(2).leftover).toBe('vineyard');
    expect(at(3).leftover).toBeNull();
    expect(at(4).kept).toBe('');
    expect(lastCardText(at(4), 'Eli', false)).toEqual([
      'Eli keeps the last card for an extra turn'
    ]);
    expect(lastCardText(at(2), 'Cy', true)).toEqual([
      'Your last card, Vineyard, went to the discard pile'
    ]);
  });

  it('ends the game with its conflicts, and both neighbours of a buyer are paid', () => {
    const r = buildRecap({ batch: 30, events: overSeat }, 7, 0, 0, []);
    expect(r).toMatchObject({ ageEnd: 3, gameOver: true, ageStart: null });
    const coins = (seat: number) => r.lines.find((l) => l.seat === seat)!.coins;
    expect(coins(1)).toEqual([
      { kind: 'paid', seat: 2, amount: 2 },
      { kind: 'paid', seat: 0, amount: 2 }
    ]);
    expect(coins(2)).toEqual([
      { kind: 'received', seat: 1, amount: 2 },
      { kind: 'sale', amount: 3 }
    ]);
    expect(recapTitle(r)).toBe('Last turn of Age III');
    expect(closingText(r, null)).toEqual([
      'Age III ends: the military conflicts follow.',
      'The game is over: the scores follow.'
    ]);
  });

  it('never names a sold card or a card under a wonder to anyone else', () => {
    const seat = buildRecap({ batch: 30, events: overSeat }, 7, 0, 0, []);
    const watch = buildRecap({ batch: 30, events: overWatch }, 7, 0, 0, []);
    expect(seat.lines[0]!.act).toEqual({ kind: 'sell', design: 'haven' });
    expect(watch.lines[0]!.act).toEqual({ kind: 'sell', design: '' });
    expect(actText(seat.lines[0]!, 'Ana', true)).toBe('You sold Haven');
    expect(actText(watch.lines[0]!, 'Ana', false)).toBe('Ana sold a card');
    const stage = buildRecap({ batch: 9, events: discard }, 7, 2, 0, []).lines[0]!;
    expect(actText(stage, 'Cy', true)).toBe('You built wonder stage I with Statue');
    expect(actText({ ...stage, act: { kind: 'stage', stage: 1, design: '' } }, 'Cy', false)).toBe(
      'Cy built wonder stage I'
    );
  });

  it('reads the turn and the pass from hands-passed, and the deal of a new Age', () => {
    const play: TableEvent = { ...blank, kind: 'play', seat: 1, play: 'build', design: 'baths' };
    const passed: TableEvent = { ...blank, kind: 'hands-passed', age: 2, turn: 3, pass: 'right' };
    const r = buildRecap({ batch: 12, events: [play, passed] }, 3, 0, 0, []);
    expect(r.turn).toEqual({ age: 2, turn: 3 });
    expect(r.pass).toBe('right');
    expect(r.lines.map((l) => l.seat)).toEqual([1]);
    expect(recapTitle(r)).toBe('Turn 3, Age II');
    expect(closingText(r, { name: 'Bo', count: 4 })).toEqual([
      'Hands pass right.',
      'You now hold the 4 cards Bo passed you.'
    ]);
    expect(dealOf([play, passed])).toBe('right');
    expect(dealOf([play, { ...blank, kind: 'age-start', age: 2 }])).toBe('deal');
    expect(dealOf([play])).toBeNull();
  });
});

describe('the newest recap and its changes', () => {
  const side: TableEvent = { ...blank, kind: 'side-chosen', seat: 0, side: 'night' };
  const play = (seat: number): TableEvent => ({
    ...blank,
    kind: 'play',
    seat,
    play: 'sell',
    design: ''
  });
  const log: PlatformLogEntry[] = [
    { kind: 'seat', at: 0, by: 'u-a', seat: 1, control: 'human', userId: 'u-b' },
    { kind: 'role', at: 1, by: 'u-a', userId: 'u-b', role: 'admin' },
    { kind: 'link', at: 1, by: 'u-b', seat: 1 },
    { kind: 'resign', at: 2, by: 'u-b', seat: 1, control: 'bot' }
  ];
  const history = [
    { batch: 1, events: [side, { ...blank, kind: 'age-start', age: 1 }] },
    { batch: 2, events: [play(0), play(1), play(2)] },
    { batch: 3, events: [{ ...blank, kind: 'military', age: 1 }] }
  ];

  it('takes the newest batch with plays, and the changes made during that turn', () => {
    expect(recapWorthy(history[2]!.events)).toBe(false);
    const r = latestRecap(history, 3, 0, log)!;
    expect(r.batch).toBe(2);
    expect(r.changes.map((c) => c.kind)).toEqual(['role']);
    const first = latestRecap(history.slice(0, 1), 3, 0, log)!;
    expect(first.batch).toBe(1);
    expect(first.changes.map((c) => c.kind)).toEqual(['seat']);
    expect(recapTitle(first)).toBe('Wonder sides');
    expect(latestRecap([], 3, 0, log)).toBeNull();
  });

  it('words each change with the names it is given', () => {
    const who = (id: string | undefined) =>
      id === 'u-a' ? 'Ana' : id === 'u-b' ? 'Bo' : 'A player who left';
    expect(log.map((e) => changeText(e, who))).toEqual([
      'Ana seated Bo in seat 2',
      'Ana made Bo an admin',
      '',
      'Bo resigned, and a bot plays seat 2'
    ]);
    expect(changeText({ kind: 'role', at: 0, by: 'u-b', userId: 'u-b', role: 'human' }, who)).toBe(
      'Bo stopped being an admin'
    );
    expect(changeText({ kind: 'role', at: 0, by: 'u-a', userId: 'u-b', role: 'human' }, who)).toBe(
      'Ana removed Bo as admin'
    );
    expect(changeText({ kind: 'seat', at: 0, by: 'u-x', seat: 0, control: 'bot' }, who)).toBe(
      'A player who left gave seat 1 to a bot'
    );
  });
});

describe('the recap in words', () => {
  const r = buildRecap({ batch: 9, events: discard }, 7, 2, 0, []);
  const at = (seat: number) => r.lines.find((l) => l.seat === seat)!;

  it('says each coin change from its seat', () => {
    expect(at(1).coins.map((c) => coinText(c, nameOf))).toEqual([
      'paid Ana 4 coins',
      'received 2 coins from Cy'
    ]);
    expect(coinText({ kind: 'bank', amount: 1 }, nameOf)).toBe('paid 1 coin to the bank');
    expect(coinText({ kind: 'card', design: 'vineyard', amount: 5 }, nameOf)).toBe(
      'took 5 coins from Vineyard'
    );
    expect(coinText({ kind: 'stage', stage: 2, amount: 3 }, nameOf)).toBe(
      'took 3 coins from wonder stage II'
    );
  });

  it('tags each seat with its card, its stage or its sale', () => {
    expect(tagText(at(1), null)).toBe('Dispensary');
    expect(tagText(at(2), null)).toBe('Stage I');
    expect(tagText(at(3), saleOf(at(3)))).toBe('Sold +3');
  });

  it('reads the whole recap for the live region', () => {
    const text = recapSpeech(r, nameOf, 2, () => '', null);
    expect(text).toContain('The last turn.');
    expect(text).toContain('You built wonder stage I with Statue, paid Bo 2 coins.');
    expect(text).toContain('Ana sold a card, received 4 coins from Bo, took 3 coins for the sale.');
    expect(text).toContain('Your last card, Vineyard, went to the discard pile.');
    expect(text).not.toContain('Dee sold Vineyard');
  });
});
