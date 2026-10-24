// The table screen's model over the engine's own view fixtures
// (server/internal/haftgane/rules/testdata/views): the seat ring for a
// seat and for a spectator, the far columns, the badges a hand card earns
// from the server's choices, and the printed facts the catalog adds.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import type { Seat, Table } from '$lib/wire/haftgane.gen';
import { hgBoards, hgDesigns } from '$lib/wire/haftgane-catalog.gen';
import type { PlatformHuman, TableState } from '$lib/wire/platform.gen';
import { tableFrame } from '../session/test-frames';
import {
  afford,
  cardChoices,
  chosenCount,
  colourCounts,
  COLOURS,
  farColumns,
  makings,
  previousSeatOf,
  ring,
  roleActions,
  roleOf,
  seatStatus,
  spectatorsOf,
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

  it('centres a spectator on the seat they follow', () => {
    const seats = fixture('turn-spectator').seats;
    const r = ring(seats, 4)!;
    expect(r.own).toBe(false);
    expect(r.anchor.seat).toBe(4);
    expect([r.left.seat, r.right.seat]).toEqual([5, 3]);
    expect(seatNumbers(r.far)).toEqual([6, 0, 1, 2]);
    // A seat the table does not have falls back to seat 0.
    expect(ring(seats, 9)!.anchor.seat).toBe(0);
  });

  it('ignores the followed seat for a seated viewer', () => {
    expect(ring(fixture('discard-seat').seats, 5)!.anchor.seat).toBe(2);
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

describe('roles and spectators', () => {
  const human = (userId: string, over: Partial<PlatformHuman> = {}): PlatformHuman => ({
    userId,
    name: userId,
    nameNo: 1,
    connected: true,
    creator: false,
    admin: false,
    temporary: false,
    ...over
  });

  it('tags the creator, an admin and the temporary admin', () => {
    expect(roleOf(human('a', { creator: true, admin: true }))).toBe('creator');
    expect(roleOf(human('a', { admin: true }))).toBe('admin');
    expect(roleOf(human('a', { temporary: true }))).toBe('temporary');
    // A creator who stepped down holds no role.
    expect(roleOf(human('a', { creator: true }))).toBeNull();
    expect(roleOf(human('a'))).toBeNull();
    expect(roleOf(undefined)).toBeNull();
  });

  it('lists the humans who hold no seat, away or not', () => {
    const f = tableFrame({}, 0);
    expect(spectatorsOf(f).map((h) => h.userId)).toEqual(['w']);
    const away = tableFrame({ humans: [...f.humans, human('x', { connected: false })] }, 0);
    expect(spectatorsOf(away).map((h) => h.userId)).toEqual(['w', 'x']);
  });

  it('reads the previous seat only from the frame that names it', () => {
    const f = tableFrame({}, 0);
    expect(previousSeatOf(f, 'w')).toBeNull();
    const admin = { ...f, you: { ...f.you, previousSeats: [{ userId: 'w', seat: 2 }] } };
    expect(previousSeatOf(admin, 'w')).toBe(2);
    expect(previousSeatOf(admin, 'u1')).toBeNull();
  });

  it('offers role buttons as the roles allow', () => {
    // u0 is the creator and an admin; u1 a player; w a spectator. The
    // server tells each real admin whether the admin cap has room.
    const f = tableFrame({}, 0);
    const creator = { ...f, you: { ...f.you, canPromote: true } };
    expect(roleActions(creator, 'u1')).toEqual({ promote: true, demote: false });
    expect(roleActions(creator, 'u0')).toEqual({ promote: false, demote: false });
    // A full cap, or a frame that does not say, offers no promotion.
    const full = { ...f, you: { ...f.you, canPromote: false } };
    expect(roleActions(full, 'u1')).toEqual({ promote: false, demote: false });
    expect(roleActions(f, 'u1')).toEqual({ promote: false, demote: false });
    const promoted = tableFrame(
      {
        humans: creator.humans.map((h) =>
          h.userId === 'u1' ? { ...h, admin: true, promotedBy: 'u0' } : h
        )
      },
      0
    );
    expect(roleActions(promoted, 'u1')).toEqual({ promote: false, demote: true });
    // u1, promoted by u0, may not remove the creator, and may remove only
    // the admins u1 promoted.
    const asU1 = { ...promoted, you: { userId: 'u1', seat: 1, canPromote: true } };
    expect(roleActions(asU1, 'u0')).toEqual({ promote: false, demote: false });
    expect(roleActions(asU1, 'w')).toEqual({ promote: true, demote: false });
    const third = {
      ...asU1,
      humans: asU1.humans.map((h) =>
        h.userId === 'w' ? { ...h, admin: true, promotedBy: 'u0' } : h
      )
    };
    expect(roleActions(third, 'w')).toEqual({ promote: false, demote: false });
    // A temporary admin, a player and a finished game get none.
    const temporary = {
      ...creator,
      you: { userId: 'u1', seat: 1 },
      humans: creator.humans.map((h) => (h.userId === 'u1' ? { ...h, temporary: true } : h))
    };
    expect(roleActions(temporary, 'w')).toEqual({ promote: false, demote: false });
    expect(roleActions(tableFrame({}, 1), 'w')).toEqual({ promote: false, demote: false });
    expect(roleActions({ ...creator, status: 'finished' }, 'u1')).toEqual({
      promote: false,
      demote: false
    });
  });
});
