// The table screen's model over the engine's own view fixtures
// (server/internal/haftgane/rules/testdata/views): the seat ring for a
// seat and for a spectator, the far columns, the badges a hand card earns
// from the server's choices, and the printed facts the catalog adds.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import type { Seat, Table } from '$lib/wire/haftgane.gen';
import { hgBoards, hgDesigns } from '$lib/wire/haftgane-catalog.gen';
import type { TableState } from '$lib/wire/platform.gen';
import {
  afford,
  cardChoices,
  chosenCount,
  colourCounts,
  COLOURS,
  farColumns,
  makings,
  ring,
  seatStatus,
  stageOpen,
  stagesOf
} from './model';

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

const seatNumbers = (seats: readonly Seat[]) => seats.map((s) => s.seat);

describe('ring', () => {
  it('puts the next seat on the left and the previous on the right', () => {
    const r = ring(fixture('turn-seat').seats)!;
    expect(r.own).toBe(true);
    expect(r.anchor.seat).toBe(0);
    expect(r.left.relation).toBe('left');
    expect(r.right.relation).toBe('right');
    expect(seatNumbers(r.far)).toEqual([2, 3, 4, 5]);
  });

  it('follows the viewer around the table', () => {
    const r = ring(fixture('discard-seat').seats)!;
    expect(r.anchor.seat).toBe(2);
    expect([r.left.seat, r.right.seat]).toEqual([3, 1]);
    expect(seatNumbers(r.far)).toEqual([4, 5, 6, 0]);
  });

  it('anchors a spectator on seat 0', () => {
    const r = ring(fixture('turn-spectator').seats)!;
    expect(r.own).toBe(false);
    expect(r.anchor.seat).toBe(0);
    expect([r.left.seat, r.right.seat]).toEqual([1, 6]);
    expect(seatNumbers(r.far)).toEqual([2, 3, 4, 5]);
  });

  it('has no far seats at three', () => {
    const seats = fixture('turn-seat')
      .seats.slice(0, 3)
      .map((s, i) => ({
        ...s,
        relative: i,
        relation: ['self', 'left', 'right'][i]!
      }));
    const r = ring(seats)!;
    expect([r.left.seat, r.right.seat]).toEqual([1, 2]);
    expect(r.far).toEqual([]);
  });
});

describe('farColumns', () => {
  it('splits the far seats in ring order, the larger half on the start side', () => {
    const far = ring(fixture('turn-seat').seats)!.far;
    expect(seatNumbers(farColumns(far).start)).toEqual([2, 3]);
    expect(seatNumbers(farColumns(far).end)).toEqual([4, 5]);
    expect(seatNumbers(farColumns(far.slice(0, 3)).start)).toEqual([2, 3]);
    expect(seatNumbers(farColumns(far.slice(0, 3)).end)).toEqual([4]);
    expect(farColumns([])).toEqual({ start: [], end: [] });
  });
});

describe('afford', () => {
  const view = fixture('turn-seat');
  const you = view.you!;
  const of = (slug: string) => afford(cardChoices(slug, you.choices, you.disabled));

  it('shows the server total and whether it buys from a neighbour', () => {
    const baths = you.choices.find((c) => c.key.startsWith('build:baths'))!;
    expect(of('baths')).toEqual({ kind: 'paid', total: baths.total, trade: true });
    const pit = you.choices.find((c) => c.key.startsWith('build:clay-pit'))!;
    expect(of('clay-pit')).toEqual({ kind: 'paid', total: pit.total, trade: false });
    expect(of('theater')).toEqual({ kind: 'none' });
  });

  it('names a free build and a refused one', () => {
    const discard = fixture('discard-seat').you!;
    expect(afford(cardChoices('baths', discard.choices, discard.disabled))).toEqual({
      kind: 'free',
      via: 'discard'
    });
    const last = fixture('last-seat').you!;
    expect(afford(cardChoices('dispensary', last.choices, last.disabled))).toEqual({
      kind: 'blocked',
      reason: 'cannot-cover',
      shortBy: 0
    });
  });

  it('groups a card choices by kind', () => {
    const cc = cardChoices('baths', you.choices, you.disabled);
    expect(cc.builds.map((c) => c.kind)).toEqual(['build']);
    expect(cc.sells.map((c) => c.key)).toEqual(['sell:baths']);
    expect(cc.stages).toEqual([]);
    expect(stageOpen(you.choices)).toBe(false);
  });
});

describe('catalog facts', () => {
  it('counts a city by colour in the layout order', () => {
    const city = fixture('discard-seat').seats[0]!;
    const counts = colourCounts(city.built);
    expect(counts.map((c) => c.colour)).toEqual([...COLOURS]);
    expect(counts.reduce((a, c) => a + c.count, 0)).toBe(city.built.length);
    for (const { colour, count } of counts) {
      expect(count).toBe(
        city.built.filter((s) => hgDesigns[s as keyof typeof hgDesigns].colour === colour).length
      );
    }
  });

  it('lists the board resource first, then each printed producer', () => {
    const city = fixture('discard-seat').seats[1]!;
    const made = makings(city);
    const board = hgBoards[city.board as keyof typeof hgBoards];
    expect(made[0]).toEqual({ res: [board.start], n: 1, choice: false });
    const producers = city.built.filter((s) =>
      hgDesigns[s as keyof typeof hgDesigns].effects.some(
        (e) => e.kind === 'produce' || e.kind === 'produce-choice'
      )
    );
    expect(made.length).toBeGreaterThanOrEqual(1 + producers.length);
  });

  it('has no stages before a side is chosen', () => {
    const seat = fixture('sides').seats[0]!;
    expect(stagesOf(seat)).toEqual([]);
    expect(stagesOf({ board: seat.board, side: 'day' }).length).toBeGreaterThan(0);
  });
});

describe('seat status', () => {
  const frame = (over: Partial<TableState>): TableState =>
    ({
      seats: [
        { seat: 0, kind: 'human', userId: 'a' },
        { seat: 1, kind: 'bot', bot: 'easy' },
        { seat: 2, kind: 'human', userId: 'c' }
      ],
      humans: [
        { userId: 'a', connected: true },
        { userId: 'c', connected: false }
      ],
      ...over
    }) as unknown as TableState;
  const point = (required: number[], confirmed: number[]) => ({
    ref: { id: 'a1t1', revision: 1 },
    kind: 'turn',
    required,
    confirmed
  });

  it('marks confirmed, choosing, a bot and an away human', () => {
    const f = frame({ point: point([0, 1, 2], [0]) });
    expect(seatStatus(f, 0)).toBe('confirmed');
    expect(seatStatus(f, 1)).toBe('bot');
    expect(seatStatus(f, 2)).toBe('away');
    expect(chosenCount(f)).toEqual({ done: 1, of: 3 });
  });

  it('shows nothing for a seat the point does not ask', () => {
    const f = frame({ point: point([2], []) });
    expect(seatStatus(f, 0)).toBe('idle');
    expect(seatStatus(frame({}), 0)).toBe('idle');
    expect(chosenCount(frame({}))).toEqual({ done: 0, of: 0 });
  });
});
