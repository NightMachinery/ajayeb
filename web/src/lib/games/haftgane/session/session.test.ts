// The Haftgane Session over synthetic frames (./test-frames.ts) and a fake
// connection standing in for the entry's bootstrap: every submission
// outcome, reconnecting, reload restore, former-controller tabs, the
// lobby's Start, device links and the bootstrap handover.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { entryIdentity, SeqCounters } from '$lib/entry/identity';
import type { TableState } from '$lib/wire/platform.gen';
import { FakeConnection } from './fake-connection';
import { TableSession } from './session.svelte';
import { CHOICES, CODE, lobbyFrame, tableFrame } from './test-frames';

const KEY = CHOICES[0]!.key;
const OTHER = CHOICES[1]!.key;

/** A page's client id and its seq counter; each test is a page of its own. */
interface PageIds {
  clientId: string;
  seqs: SeqCounters;
}

function setup(
  first: unknown = tableFrame(),
  claim: string | null = null,
  page: PageIds = { clientId: 'tab', seqs: new SeqCounters() }
) {
  const connection = new FakeConnection();
  connection.latest = first;
  const rooms = { remember: vi.fn(), forget: vi.fn() };
  const session = new TableSession({
    connection,
    code: CODE.toLowerCase(),
    claim,
    clientId: page.clientId,
    seqs: page.seqs,
    rooms,
    rateRetryMs: 10
  });
  return { connection, session, rooms };
}

/** The frame acknowledging seq, with the given pending pick. */
function acked(seq: number, pending: TableState['pending'], extra: Partial<TableState> = {}) {
  return tableFrame({ ack: { clientId: 'tab', seq }, pending, ...extra });
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('the handover', () => {
  it('takes the bootstrap snapshot as its first frame', () => {
    const { session, connection, rooms } = setup();
    expect(session.code).toBe(CODE);
    expect(session.table?.code).toBe(CODE);
    expect(session.conn).toBe('open');
    expect(session.canPick).toBe(true);
    expect(connection.failed).toEqual([]);
    expect(rooms.remember).toHaveBeenCalledWith(
      CODE,
      'online',
      expect.any(Number),
      null,
      'haftgane'
    );
  });

  it('ends the entry on a frame of a known family with the wrong shape', () => {
    const { session, connection } = setup();
    connection.frame({ ...tableFrame(), view: { phase: 1 } });
    expect(connection.failed).toEqual(['protocol']);
    expect(session.table?.view.phase).toBe('play');
    connection.frame({ t: 'table-reject', seq: 1, code: 'unheard-of' });
    expect(connection.failed).toEqual(['protocol', 'protocol']);
  });

  it('ignores frames of another room and families it does not read', () => {
    const { session, connection } = setup();
    const before = session.table;
    connection.frame(tableFrame({ code: 'OTHERS', revision: 9 << 20 }));
    connection.frame({ t: 'state', code: CODE });
    expect(session.table).toBe(before);
    expect(connection.failed).toEqual([]);
  });

  it('keeps each frame whole: what it holds is what the server sent', () => {
    const { session, connection } = setup();
    const next = acked(9, { key: KEY, confirm: false });
    connection.frame(next);
    expect(session.table).toBe(next);
    const watching = tableFrame({ history: [] }, null);
    connection.frame(watching);
    expect(session.table).toBe(watching);
    expect(session.table?.view.you).toBeUndefined();
    expect(session.table?.pending).toBeUndefined();
  });
});

describe('picks', () => {
  it('sends a key of the choices with the open point and the next seq', () => {
    const { session, connection } = setup();
    expect(session.pick('build:nothing:0:0')).toBe(false);
    expect(session.pick(KEY)).toBe(true);
    expect(connection.last()).toEqual({
      t: 'table-pick',
      code: CODE,
      seq: 1,
      point: { id: 'age-1-turn-1', revision: 1 },
      key: KEY,
      confirm: false,
      seat: 0
    });
    // One request in flight at a time.
    expect(session.canPick).toBe(false);
    expect(session.pick(OTHER)).toBe(false);
    connection.frame(acked(1, { key: KEY, confirm: false }, { revision: (1 << 20) + 1 }));
    expect(session.pending).toBeNull();
    expect(session.pick(KEY)).toBe(false); // already the pending pick
    expect(session.pick(OTHER)).toBe(true);
    expect(connection.last()).toMatchObject({ seq: 2, key: OTHER, confirm: false });
  });

  it('confirms the pending pick and withdraws the confirmation', () => {
    const { session, connection } = setup(tableFrame({ pending: { key: KEY, confirm: false } }));
    expect(session.withdraw()).toBe(false);
    expect(session.confirm()).toBe(true);
    expect(connection.last()).toMatchObject({ t: 'table-pick', seq: 1, key: KEY, confirm: true });
    connection.frame(
      acked(1, { key: KEY, confirm: true }, { point: { ...tableFrame().point!, confirmed: [0] } })
    );
    expect(session.confirm()).toBe(false);
    expect(session.withdraw()).toBe(true);
    expect(connection.last()).toMatchObject({ seq: 2, key: KEY, confirm: false });
  });

  it('settles a duplicate by the frame acknowledging its seq', () => {
    const { session, connection } = setup();
    session.pick(KEY, true);
    // Another seat's frame, a presence refresh: no ack, still in flight.
    connection.frame(tableFrame({ revision: 2 << 20 }));
    expect(session.pending).not.toBeNull();
    connection.frame(acked(1, { key: KEY, confirm: true }, { revision: 2 << 20 }));
    expect(session.pending).toBeNull();
  });

  it('turns a refusal into a notice and frees the next request', () => {
    for (const code of [
      'decision-closed',
      'illegal',
      'stale',
      'forbidden',
      'storage',
      'bad-request'
    ]) {
      const { session, connection } = setup();
      session.pick(KEY);
      connection.frame({ t: 'table-reject', seq: 1, code });
      expect(session.pending, code).toBeNull();
      expect(session.notice).toEqual({
        kind: 'reject',
        code,
        message: `hg.table.reject.${code}`
      });
      expect(session.canPick).toBe(true);
      session.pick(KEY);
      expect(session.notice).toBeNull();
    }
  });

  it('refuses after the last confirmation: the point closed', () => {
    const { session, connection } = setup();
    session.pick(KEY, true);
    // The resolution overtook the pick: a new point opens.
    connection.frame(
      tableFrame({
        revision: 2 << 20,
        batch: 1,
        events: [
          {
            kind: 'build',
            other: 0,
            turn: 1,
            side: '',
            printed: 0,
            delta: 0,
            source: '',
            outcome: '',
            seat: 1,
            play: 'build',
            design: 'basalt',
            stage: 0,
            left: 0,
            right: 0,
            age: 1,
            pass: 'left'
          }
        ],
        point: {
          ref: { id: 'age-1-turn-2', revision: 2 },
          kind: 'play',
          required: [0, 1, 2],
          confirmed: []
        }
      })
    );
    connection.frame({ t: 'table-reject', seq: 1, code: 'decision-closed' });
    expect(session.notice?.code).toBe('decision-closed');
    expect(session.pick(KEY)).toBe(true);
    expect(connection.last()).toMatchObject({ seq: 2, point: { id: 'age-1-turn-2', revision: 2 } });
  });

  it('sends a rate-limited request again a few times, then shows the notice', () => {
    const { session, connection } = setup();
    session.pick(KEY);
    for (let i = 1; i <= 3; i++) {
      connection.frame({ t: 'table-reject', seq: 1, code: 'rate' });
      expect(session.notice).toBeNull();
      vi.advanceTimersByTime(10 * i);
      expect(connection.sent).toHaveLength(1 + i);
      expect(connection.last()).toEqual(connection.sent[0]);
    }
    connection.frame({ t: 'table-reject', seq: 1, code: 'rate' });
    expect(session.notice?.code).toBe('rate');
    expect(session.pending).toBeNull();
  });

  it('drops a rate-limited pick whose seat or point changed before the retry', () => {
    const { session, connection } = setup();
    session.pick(KEY);
    connection.frame({ t: 'table-reject', seq: 1, code: 'rate' });
    connection.frame(tableFrame({ revision: 2 << 20, history: [] }, null));
    vi.advanceTimersByTime(100);
    expect(connection.sent).toHaveLength(1);
    expect(session.pending).toBeNull();
  });

  it('waits for slowMs before calling a request slow', () => {
    const { session } = setup();
    session.pick(KEY);
    expect(session.slow).toBe(false);
    vi.advanceTimersByTime(300);
    expect(session.slow).toBe(true);
  });

  it('never picks for a spectator, a seat not required, or a finished game', () => {
    expect(setup(tableFrame({}, null)).session.pick(KEY)).toBe(false);
    const done = tableFrame({ point: { ...tableFrame().point!, required: [1, 2] } });
    expect(setup(done).session.canPick).toBe(false);
    expect(setup(tableFrame({ status: 'finished' })).session.pick(KEY)).toBe(false);
  });
});

describe('reconnecting', () => {
  it('sends the pick in flight again, unchanged, after the join frame shows the same seat and point', () => {
    const { session, connection } = setup();
    connection.drop();
    expect(session.conn).toBe('reconnecting');
    expect(session.canPick).toBe(false);
    connection.rejoin();
    // A pick made before the join's frame waits for it.
    expect(session.pick(KEY, true)).toBe(true);
    expect(connection.sent).toHaveLength(0);
    connection.frame(tableFrame({ history: [] }));
    expect(connection.sent).toHaveLength(1);
    const pick = connection.last();
    connection.drop();
    connection.rejoin();
    // Nothing goes out before the join's frame.
    expect(connection.sent).toHaveLength(1);
    connection.frame(tableFrame({ history: [] }));
    expect(connection.sent).toHaveLength(2);
    expect(connection.last()).toEqual(pick);
    connection.frame(acked(1, { key: KEY, confirm: true }));
    expect(session.pending).toBeNull();
  });

  it('drops the pick when the join frame shows another point: it was applied or overtaken', () => {
    const { session, connection } = setup();
    session.pick(KEY, true);
    connection.drop();
    connection.rejoin();
    connection.frame(
      tableFrame({
        history: [],
        batch: 1,
        revision: 2 << 20,
        point: {
          ref: { id: 'age-1-turn-2', revision: 2 },
          kind: 'play',
          required: [0, 1, 2],
          confirmed: []
        }
      })
    );
    expect(connection.sent).toHaveLength(1);
    expect(session.pending).toBeNull();
    expect(session.canPick).toBe(true);
  });

  it('never sends a pick for the seat this tab held once the join frame shows another', () => {
    const { session, connection } = setup();
    session.pick(KEY, true);
    connection.drop();
    connection.rejoin();
    // Switched to seat 1 while away; the same key is offered there too.
    connection.frame(tableFrame({ history: [], revision: 2 << 20 }, 1));
    expect(connection.sent).toHaveLength(1);
    expect(session.pending).toBeNull();
    expect(session.notice).toEqual({ kind: 'seated', code: 'seated', seat: 1 });
  });

  it('sends other requests again unchanged after the join frame', () => {
    const { session, connection } = setup();
    session.switchSeat(1, { bot: 'easy' });
    const seat = connection.last();
    connection.drop();
    connection.rejoin();
    connection.frame(tableFrame({ history: [] }));
    expect(connection.last()).toEqual(seat);
  });

  it('restores the pick and its confirmation after a reload: a new page, its own seq', () => {
    const { session, connection } = setup(
      tableFrame({
        history: [],
        pending: { key: OTHER, confirm: true },
        point: { ...tableFrame().point!, confirmed: [0] }
      })
    );
    expect(session.table?.pending).toEqual({ key: OTHER, confirm: true });
    expect(session.confirm()).toBe(false);
    expect(session.withdraw()).toBe(true);
    expect(connection.last()).toMatchObject({ seq: 1, key: OTHER, confirm: false });
  });
});

describe('former-controller tabs', () => {
  it('turns into a spectator view when the seat is switched away, never acting on the old seat', () => {
    const { session, connection } = setup(
      tableFrame({
        history: [{ batch: 1, events: [] }],
        batch: 1,
        pending: { key: KEY, confirm: false }
      })
    );
    session.pick(OTHER);
    // The switch's frame: watching, with the whole history again.
    const history = [{ batch: 1, events: [] }];
    connection.frame(tableFrame({ revision: 2 << 20, batch: 1, history }, null));
    expect(session.seat).toBeNull();
    expect(session.history).toBe(history);
    expect(session.notice).toEqual({ kind: 'watching', code: 'watching' });
    expect(session.canPick).toBe(false);
    expect(session.choices).toEqual([]);
    // The pick that crossed the switch is refused; nothing goes out again.
    connection.frame({ t: 'table-reject', seq: 1, code: 'forbidden' });
    expect(session.pending).toBeNull();
    connection.drop();
    connection.rejoin();
    connection.frame(tableFrame({ revision: 2 << 20, batch: 1, history }, null));
    expect(connection.sent).toHaveLength(1);
    expect(session.pick(KEY)).toBe(false);
    expect(session.resign()).toBe(false);
  });

  it('drops its pick in flight after a reconnect into watching', () => {
    const { session, connection } = setup();
    session.pick(KEY);
    connection.drop();
    connection.rejoin();
    connection.frame(tableFrame({ revision: 2 << 20, history: [] }, null));
    expect(connection.sent).toHaveLength(1);
    expect(session.pending).toBeNull();
  });

  it('resigns, then watches', () => {
    const { session, connection } = setup();
    expect(session.resign()).toBe(true);
    expect(connection.last()).toEqual({
      t: 'table-resign',
      code: CODE,
      seq: 1,
      confirm: true,
      seat: 0
    });
    connection.frame(
      tableFrame({ ack: { clientId: 'tab', seq: 1 }, history: [], revision: 2 << 20 }, null)
    );
    expect(session.pending).toBeNull();
    expect(session.seat).toBeNull();
    expect(session.notice?.code).toBe('watching');
  });
});

/** The frame after an admin moved this tab's user (u0) to seat 1, or to watching. */
function moved(to: 1 | null): TableState {
  return tableFrame(
    {
      revision: 3 << 20,
      history: [],
      you: to === null ? { userId: 'u0' } : { userId: 'u0', seat: 1 },
      seats: [
        { seat: 0, kind: 'bot', bot: 'easy' },
        { seat: 1, kind: 'human', userId: to === null ? 'u1' : 'u0' },
        { seat: 2, kind: 'bot', bot: 'easy' }
      ]
    },
    to
  );
}

describe('requests the seat makes stay with that seat', () => {
  // Undo is offered too: a sole human's table one batch in.
  const solo = { allowUndo: true, startingHumanCount: 1, batch: 1, revision: 2 << 20 };
  const requests: [string, (s: TableSession) => boolean][] = [
    ['resignation', (s) => s.resign()],
    ['undo', (s) => s.undo()],
    ['pick', (s) => s.pick(KEY)]
  ];
  for (const [name, send] of requests) {
    for (const to of [1, null] as const) {
      const where = to === null ? 'watching' : 'another seat';
      it(`never sends a ${name} again after a reconnect into ${where}`, () => {
        const { session, connection } = setup(tableFrame(solo));
        expect(send(session)).toBe(true);
        connection.drop();
        connection.rejoin();
        connection.frame(moved(to));
        expect(session.seat).toBe(to);
        expect(connection.sent).toHaveLength(1);
        expect(session.pending).toBeNull();
      });

      it(`never retries a rate-limited ${name} after a move to ${where}`, () => {
        const { session, connection } = setup(tableFrame(solo));
        expect(send(session)).toBe(true);
        connection.frame({ t: 'table-reject', seq: 1, code: 'rate' });
        connection.frame(moved(to));
        expect(session.pending).toBeNull();
        vi.advanceTimersByTime(100);
        expect(connection.sent).toHaveLength(1);
      });
    }

    it(`sends a ${name} again unchanged while the seat stays, after a reconnect and after rate`, () => {
      const { session, connection } = setup(tableFrame(solo));
      expect(send(session)).toBe(true);
      const first = connection.last();
      connection.drop();
      connection.rejoin();
      connection.frame(tableFrame({ ...solo, history: [] }));
      expect(connection.sent).toEqual([first, first]);
      connection.frame({ t: 'table-reject', seq: 1, code: 'rate' });
      vi.advanceTimersByTime(10);
      expect(connection.sent).toEqual([first, first, first]);
      expect(session.pending?.frame).toEqual(first);
    });
  }

  it('drops a resignation in flight as soon as a frame shows this tab moved', () => {
    const { session, connection } = setup();
    expect(session.resign()).toBe(true);
    connection.frame(moved(1));
    expect(session.pending).toBeNull();
    expect(session.notice).toEqual({ kind: 'seated', code: 'seated', seat: 1 });
    // An answer to it that comes later changes nothing.
    connection.frame({ t: 'table-reject', seq: 1, code: 'forbidden' });
    expect(session.notice).toEqual({ kind: 'seated', code: 'seated', seat: 1 });
    expect(connection.sent).toHaveLength(1);
  });
});

describe('the seq counter', () => {
  it('goes on from the last seq when the Session mounts again in the same page', () => {
    const page = { clientId: 'page', seqs: new SeqCounters() };
    const a = setup(tableFrame(), null, page);
    expect(a.session.pick(KEY)).toBe(true);
    expect(a.connection.last()).toMatchObject({ seq: 1, key: KEY });
    a.connection.frame(
      tableFrame({ ack: { clientId: 'page', seq: 1 }, pending: { key: KEY, confirm: false } })
    );
    a.session.close();
    // Home and back: a new Session over the frame the server restores the
    // pending pick in.
    const b = setup(tableFrame({ history: [], pending: { key: KEY, confirm: false } }), null, page);
    expect(b.session.table?.pending).toEqual({ key: KEY, confirm: false });
    expect(b.session.pick(OTHER)).toBe(true);
    expect(b.connection.last()).toMatchObject({ seq: 2, key: OTHER });
    b.connection.frame(
      tableFrame({ ack: { clientId: 'page', seq: 2 }, pending: { key: OTHER, confirm: false } })
    );
    expect(b.session.pending).toBeNull();
  });

  it('starts again at 1 under the new client id a reload makes', () => {
    const seqs = new SeqCounters();
    const a = setup(tableFrame(), null, { clientId: 'before', seqs });
    a.session.pick(KEY);
    a.session.close();
    const b = setup(tableFrame({ history: [], pending: { key: KEY, confirm: false } }), null, {
      clientId: 'after',
      seqs
    });
    expect(b.session.pick(OTHER)).toBe(true);
    expect(b.connection.last()).toMatchObject({ seq: 1, key: OTHER });
    // Acknowledgements match the id this page's hello carried, not the old one.
    b.connection.frame(
      tableFrame({ ack: { clientId: 'before', seq: 1 }, pending: { key: KEY, confirm: false } })
    );
    expect(b.session.pending).not.toBeNull();
    b.connection.frame(
      tableFrame({ ack: { clientId: 'after', seq: 1 }, pending: { key: OTHER, confirm: false } })
    );
    expect(b.session.pending).toBeNull();
  });

  it('keeps the counter beside the page identity when none is given, as the entry does', () => {
    const clientId = entryIdentity().clientId;
    const mount = (first: TableState) => {
      const connection = new FakeConnection();
      connection.latest = first;
      return {
        connection,
        session: new TableSession({ connection, code: CODE, claim: null, clientId })
      };
    };
    const a = mount(tableFrame());
    expect(a.session.pick(KEY)).toBe(true);
    const seq = a.connection.last()!.seq as number;
    a.session.close();
    const b = mount(tableFrame({ history: [], pending: { key: KEY, confirm: false } }));
    expect(entryIdentity().clientId).toBe(clientId);
    expect(b.session.pick(OTHER)).toBe(true);
    expect(b.connection.last()!.seq).toBe(seq + 1);
    b.session.close();
  });
});

describe('history', () => {
  const event = {
    kind: 'build',
    other: 0,
    turn: 1,
    side: '',
    printed: 0,
    delta: 0,
    source: '',
    outcome: '',
    seat: 1,
    play: 'build',
    design: 'basalt',
    stage: 0,
    left: 0,
    right: 0,
    age: 1,
    pass: 'left'
  };

  it('extends the history by each resolution and replaces it on a history frame', () => {
    const { session, connection } = setup(tableFrame({ history: [] }));
    connection.frame(tableFrame({ batch: 1, revision: 2 << 20, events: [event] }));
    expect(session.history).toEqual([{ batch: 1, events: [event] }]);
    // The same frame again adds nothing.
    connection.frame(tableFrame({ batch: 1, revision: 2 << 20, events: [event] }));
    expect(session.history).toHaveLength(1);
    const undo = [{ batch: 1, events: [] }];
    connection.frame(tableFrame({ batch: 1, revision: 3 << 20, history: undo }));
    expect(session.history).toBe(undo);
    expect(connection.resyncs).toBe(0);
  });

  it('asks for a resync once when a batch skips ahead', () => {
    const { session, connection } = setup(tableFrame({ history: [] }));
    connection.frame(tableFrame({ batch: 3, revision: 4 << 20, events: [event] }));
    expect(connection.resyncs).toBe(1);
    connection.frame(tableFrame({ batch: 3, revision: 4 << 20 }));
    expect(connection.resyncs).toBe(1);
    connection.frame(tableFrame({ batch: 5, revision: 6 << 20 }));
    expect(connection.resyncs).toBe(2);
    expect(session.history).toHaveLength(1);
  });
});

describe('admin requests and undo', () => {
  it('sends seat switches and role changes as their frames say', () => {
    const { session, connection } = setup();
    expect(session.admin).toBe(true);
    session.switchSeat(1, { userId: 'w' });
    expect(connection.last()).toEqual({
      t: 'table-seat',
      code: CODE,
      seq: 1,
      seat: 1,
      control: 'human',
      userId: 'w'
    });
    connection.frame(acked(1, undefined));
    session.switchSeat(2, { bot: 'easy' });
    expect(connection.last()).toEqual({
      t: 'table-seat',
      code: CODE,
      seq: 2,
      seat: 2,
      control: 'bot',
      bot: 'easy'
    });
    connection.frame(acked(2, undefined));
    expect(session.role('promote')).toBe(false);
    session.role('promote', 'u1');
    expect(connection.last()).toEqual({
      t: 'table-role',
      code: CODE,
      seq: 3,
      op: 'promote',
      userId: 'u1'
    });
    connection.frame(acked(3, undefined));
    session.role('step-down', 'u1');
    expect(connection.last()).toEqual({ t: 'table-role', code: CODE, seq: 4, op: 'step-down' });
  });

  it('offers no seat switch to a non-admin', () => {
    const { session } = setup(tableFrame({}, 1));
    expect(session.admin).toBe(false);
    expect(session.switchSeat(0, { bot: 'easy' })).toBe(false);
  });

  it('sends a sole human undo once more after a stale refusal the bots caused', () => {
    const solo = { allowUndo: true, startingHumanCount: 1, batch: 1, revision: 3 << 20 };
    const { session, connection } = setup(tableFrame(solo));
    expect(session.undo()).toBe(true);
    expect(connection.last()).toEqual({
      t: 'table-undo',
      code: CODE,
      seq: 1,
      base: 3 << 20,
      seat: 0
    });
    connection.frame(tableFrame({ ...solo, revision: 4 << 20 }));
    connection.frame({ t: 'table-reject', seq: 1, code: 'stale' });
    expect(connection.last()).toEqual({
      t: 'table-undo',
      code: CODE,
      seq: 1,
      base: 4 << 20,
      seat: 0
    });
    connection.frame({ t: 'table-reject', seq: 1, code: 'stale' });
    expect(session.pending).toBeNull();
    expect(session.notice?.code).toBe('stale');
    expect(setup(tableFrame({ ...solo, undoOff: true })).session.canUndo).toBe(false);
  });

  it('tells the sole human once that undo is gone when a second human sits down', () => {
    const solo = { allowUndo: true, startingHumanCount: 1, batch: 1, revision: 3 << 20 };
    const { session, connection } = setup(tableFrame(solo));
    expect(session.canUndo).toBe(true);
    connection.frame(tableFrame({ ...solo, undoOff: true, revision: 4 << 20 }));
    expect(session.canUndo).toBe(false);
    expect(session.notice).toEqual({ kind: 'undo-off', code: 'undo-off' });
    session.dismissNotice();
    connection.frame(tableFrame({ ...solo, undoOff: true, revision: 5 << 20 }));
    expect(session.notice).toBeNull();
  });

  it('says nothing about undo to a table that never had it, or to a spectator', () => {
    const { session, connection } = setup(tableFrame());
    connection.frame(tableFrame({ undoOff: true, revision: 2 << 20 }));
    expect(session.notice).toBeNull();
    const solo = { allowUndo: true, startingHumanCount: 1, batch: 1 };
    const watcher = setup(tableFrame(solo, null));
    watcher.connection.frame(tableFrame({ ...solo, undoOff: true, revision: 2 << 20 }, null));
    expect(watcher.session.notice).toBeNull();
  });

  it('remembers the seat it held while it watches, and the one it holds after', () => {
    const { session, connection } = setup();
    expect(session.held).toBe(0);
    connection.frame(moved(null));
    expect(session.seat).toBeNull();
    expect(session.held).toBe(0);
    connection.frame(moved(1));
    expect(session.held).toBe(1);
    expect(setup(tableFrame({}, null)).session.held).toBeNull();
  });

  it('keeps the seat it held across a reload, in its held-seat store', () => {
    const kept = new Map<string, number>();
    const heldSeats = {
      get: (code: string) => kept.get(code) ?? null,
      set: (code: string, seat: number) => void kept.set(code, seat)
    };
    const make = (first: TableState) => {
      const connection = new FakeConnection();
      connection.latest = first;
      const session = new TableSession({
        connection,
        code: CODE,
        claim: null,
        clientId: 'tab',
        seqs: new SeqCounters(),
        heldSeats
      });
      return { session, connection };
    };
    const first = make(tableFrame({}, 2));
    expect(kept.get(CODE)).toBe(2);
    first.connection.frame(moved(null));
    // The reload: a new Session whose first frame shows the page watching.
    const again = make(moved(null));
    expect(again.session.seat).toBeNull();
    expect(again.session.held).toBe(2);
  });
});

describe('the lobby', () => {
  it('starts the game, which the table first frame answers', () => {
    const { session, connection, rooms } = setup(lobbyFrame());
    expect(session.lobby?.code).toBe(CODE);
    expect(session.table).toBeNull();
    expect(rooms.remember).toHaveBeenCalledWith(
      CODE,
      'online',
      expect.any(Number),
      null,
      'haftgane'
    );
    expect(session.start(true)).toBe(true);
    expect(connection.last()).toEqual({ t: 'lobby-start', code: CODE, seq: 1, fillBots: true });
    connection.frame(tableFrame());
    expect(session.pending).toBeNull();
    expect(session.lobby).toBeNull();
    expect(session.table?.code).toBe(CODE);
    expect(rooms.remember).toHaveBeenCalledTimes(2);
  });

  it('shows a lobby refusal', () => {
    const { session, connection } = setup(lobbyFrame());
    session.start(false);
    connection.frame({ t: 'lobby-reject', seq: 1, code: 'forbidden' });
    expect(session.notice?.code).toBe('forbidden');
    expect(setup(lobbyFrame({ canStart: false })).session.start(true)).toBe(false);
  });
});

describe('device links and saved rooms', () => {
  it('resolves a link with its claim and moves a relinking device to the new claim', async () => {
    const { session, connection, rooms } = setup(tableFrame(), 'old-claim');
    const link = session.requestLink(1);
    expect(connection.last()).toEqual({ t: 'link', code: CODE, seat: 1 });
    connection.frame({ t: 'linked', code: CODE, seat: 1, claim: 'c1' });
    await expect(link).resolves.toBe('c1');
    expect(session.claim).toBe('old-claim');
    const relink = session.requestLink(0, true);
    connection.frame({ t: 'linked', code: CODE, seat: 0, claim: 'new-claim' });
    await expect(relink).resolves.toBe('new-claim');
    expect(session.claim).toBe('new-claim');
    expect(connection.claims).toEqual(['new-claim']);
    expect(rooms.remember).toHaveBeenLastCalledWith(
      CODE,
      'online',
      expect.any(Number),
      'new-claim',
      'haftgane'
    );
  });

  it('fails a link on its refusal, a drop or no answer', async () => {
    const { session, connection } = setup();
    const refused = session.requestLink(1);
    connection.frame({ t: 'table-reject', seq: 0, code: 'not-admin' });
    await expect(refused).rejects.toMatchObject({ code: 'not-admin' });
    const dropped = session.requestLink(1);
    connection.frame({ t: 'reject', seq: 0, code: 'rate', msg: '' });
    await expect(dropped).rejects.toMatchObject({ code: 'rate' });
    const late = session.requestLink(1);
    vi.advanceTimersByTime(8000);
    await expect(late).rejects.toMatchObject({ code: 'timeout' });
    const offline = session.requestLink(1);
    connection.drop();
    await expect(offline).rejects.toMatchObject({ code: 'reconnecting' });
    expect(connection.failed).toEqual([]);
  });

  it('forgets a finished game from the saved rooms', () => {
    const { connection, rooms } = setup();
    connection.frame(tableFrame({ status: 'finished', revision: 9 << 20 }));
    expect(rooms.forget).toHaveBeenCalledWith(CODE);
  });
});
