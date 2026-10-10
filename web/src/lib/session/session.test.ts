import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { State } from '$lib/wire';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import { ProtocolVersion } from '$lib/wire/types.gen';
import type { Identity } from './identity';
import { SavedRooms } from './rooms.svelte';
import { ScriptedTransport } from './scripted-transport';
import { Session, SessionError } from './session.svelte';

const identity: Identity = { token: 'AAAAAAAAAAAAAAAAAAAAAA', clientId: 'tab-1', name: 'Sara' };

/** A recorded state frame, copied so a test may change it. */
function recorded(name: string): State {
  const i = (fixtureNames as readonly string[]).indexOf(name);
  const frame = fixtures[i];
  if (!frame || frame.t !== 'state') throw new Error(`no recorded state ${name}`);
  return structuredClone(frame) as State;
}

/** The recorded hot-seat frame, in room `code` at `version`, without an ack. */
function playFrame(code = 'H8TQ4R', version = 10): State {
  const f = recorded('state-play');
  delete f.ack;
  return { ...f, code, version };
}

function memoryStorage(): Storage {
  const data = new Map<string, string>();
  return {
    get length() {
      return data.size;
    },
    clear: () => data.clear(),
    getItem: (k) => data.get(k) ?? null,
    key: (i) => [...data.keys()][i] ?? null,
    removeItem: (k) => void data.delete(k),
    setItem: (k, v) => void data.set(k, v)
  };
}

function setup(opts: { rooms?: SavedRooms } = {}) {
  const transport = new ScriptedTransport();
  const session = new Session({ transport, identity, rooms: opts.rooms ?? null });
  return { transport, session };
}

/** Opens the link and welcomes the client. */
function connect(transport: ScriptedTransport): void {
  transport.setState('open');
  transport.receive({ t: 'welcome', v: ProtocolVersion, server: { version: 'test', rules: 1 } });
}

/** A session in room H8TQ4R at version 10, with the recorded hot-seat view. */
function joined(opts: { rooms?: SavedRooms } = {}) {
  const s = setup(opts);
  connect(s.transport);
  s.session.join('H8TQ4R');
  s.transport.receive(playFrame());
  return s;
}

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

describe('Session: connecting', () => {
  it('says hello with the identity on open, and is open after welcome', () => {
    const { transport, session } = setup();
    expect(session.conn).toBe('connecting');
    transport.setState('open');
    expect(transport.sent).toEqual([
      { t: 'hello', v: ProtocolVersion, token: identity.token, name: 'Sara', clientId: 'tab-1' }
    ]);
    expect(session.conn).toBe('connecting');
    transport.receive({ t: 'welcome', v: ProtocolVersion, server: { version: 'test', rules: 1 } });
    expect(session.conn).toBe('open');
    expect(session.server).toEqual({ version: 'test', rules: 1 });
  });

  it('sends a join made before welcome once welcomed, and nothing before', () => {
    const { transport, session } = setup();
    session.join(' h8tq4r ');
    expect(session.code).toBe('H8TQ4R');
    transport.setState('open');
    expect(transport.sentOf('join')).toEqual([]);
    connect(transport);
    expect(transport.sentOf('join')).toEqual([{ t: 'join', code: 'H8TQ4R' }]);
  });

  it('treats a welcome of another protocol version as fatal', () => {
    const { transport, session } = setup();
    transport.setState('open');
    transport.receive({ t: 'welcome', v: ProtocolVersion + 1, server: { version: 'x', rules: 1 } });
    expect(session.fatal).toBe('protocol');
    expect(session.conn).toBe('offline');
    expect(session.notice?.message).toBe('session.error.protocol');
    expect(transport.closed).toBe(true);
  });

  it('treats a bad-hello error and a fatal close as fatal', () => {
    const a = setup();
    a.transport.setState('open');
    a.transport.receive({ t: 'error', code: 'bad-hello', msg: 'x' });
    expect(a.session.fatal).toBe('bad-hello');
    expect(a.session.notice?.message).toBe('session.error.badHello');

    const b = setup();
    connect(b.transport);
    b.transport.setState('closed', { fatal: 'protocol' });
    expect(b.session.fatal).toBe('protocol');
    expect(b.session.conn).toBe('offline');
  });
});

describe('Session: state frames', () => {
  it('keeps the latest frame and view, replaced whole', () => {
    const { session, transport } = joined();
    const first = session.view;
    expect(first?.phase).toBe('play');
    expect(session.version).toBe(10);
    expect(session.room?.mode).toBe('hotseat');
    expect(session.canAct).toBe(true);
    const next = playFrame('H8TQ4R', 11);
    transport.receive(next);
    expect(session.version).toBe(11);
    expect(session.view).toBe(next.view);
    expect(session.view).not.toBe(first);
  });

  it('ignores a state of another room', () => {
    const { session, transport } = joined();
    transport.receive(playFrame('OTHER1', 99));
    expect(session.version).toBe(10);
  });

  it('hands every accepted frame to subscribers, the current one first', () => {
    const { session, transport } = joined();
    const seen: number[] = [];
    session.subscribe((f) => seen.push(f.version));
    transport.receive(playFrame('H8TQ4R', 11));
    transport.receive(playFrame('OTHER1', 12));
    expect(seen).toEqual([10, 11]);
  });

  it('reads canAct from the frame, never from the rules', () => {
    const { session, transport } = joined();
    const f = playFrame('H8TQ4R', 11);
    f.you = { ...f.you, canAct: false };
    transport.receive(f);
    expect(session.canAct).toBe(false);

    const g = playFrame('H8TQ4R', 12);
    g.you = { seats: [1], canAct: true }; // toMove is 0
    transport.receive(g);
    expect(session.canAct).toBe(false);

    transport.receive(playFrame('H8TQ4R', 13));
    expect(session.canAct).toBe(true);
    transport.setState('reconnecting');
    expect(session.canAct).toBe(false);
  });

  it('flags a view whose catalog hash differs from the client catalog', () => {
    const { session, transport } = joined();
    expect(session.catalogStale).toBe(false);
    const f = playFrame('H8TQ4R', 11);
    f.view = { ...f.view, catalogHash: '0000000000000000' };
    transport.receive(f);
    expect(session.catalogStale).toBe(true);
  });

  it('remembers the room once', () => {
    const rooms = new SavedRooms(memoryStorage());
    const { transport } = joined({ rooms });
    transport.receive(playFrame('H8TQ4R', 11));
    expect(rooms.list.map((r) => [r.code, r.mode])).toEqual([['H8TQ4R', 'hotseat']]);
  });
});

describe('Session: acts, seq and base', () => {
  it('sends an act with the next seq, the current version as base, and the hot-seat seat', () => {
    const { session, transport } = joined();
    expect(session.act('construct:stone-reserve')).toBe(true);
    expect(transport.last()).toEqual({
      t: 'act',
      code: 'H8TQ4R',
      seq: 1,
      base: 10,
      key: 'construct:stone-reserve',
      seat: 0
    });
    expect(session.pending?.frame.seq).toBe(1);
    expect(session.canAct).toBe(false);
  });

  it('leaves out the seat outside hot-seat rooms', () => {
    const { session, transport } = joined();
    const f = playFrame('H8TQ4R', 11);
    f.room = { ...f.room, mode: 'online' };
    f.you = { seats: [0], canAct: true };
    transport.receive(f);
    session.act('construct:stone-reserve');
    expect(transport.last()).not.toHaveProperty('seat');
  });

  it('refuses a key that is not among the view choices, and a second act while one is pending', () => {
    const { session, transport } = joined();
    expect(session.act('construct:pyramids')).toBe(false);
    expect(session.act('construct:stone-reserve')).toBe(true);
    expect(session.act('discard:stone-reserve')).toBe(false);
    expect(transport.sentOf('act')).toHaveLength(1);
  });

  it('settles on the state that acks its seq, then counts on', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    // A state without an ack (another connection's update) settles nothing.
    transport.receive(playFrame('H8TQ4R', 11));
    expect(session.pending).not.toBeNull();
    // An ack for another client id settles nothing either.
    transport.receive({ ...playFrame('H8TQ4R', 11), ack: { clientId: 'tab-2', seq: 1 } });
    expect(session.pending).not.toBeNull();
    transport.receive({ ...playFrame('H8TQ4R', 11), ack: { clientId: 'tab-1', seq: 1 } });
    expect(session.pending).toBeNull();
    expect(session.canAct).toBe(true);
    session.act('discard:stone-reserve');
    expect(transport.last()).toMatchObject({ t: 'act', seq: 2, base: 11 });
  });

  it('turns slow after 300 ms and not before', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    vi.advanceTimersByTime(299);
    expect(session.slow).toBe(false);
    vi.advanceTimersByTime(1);
    expect(session.slow).toBe(true);
    transport.receive({ ...playFrame('H8TQ4R', 11), ack: { clientId: 'tab-1', seq: 1 } });
    expect(session.slow).toBe(false);
  });

  it('maps a reject for its seq to a notice and settles; a stale one is followed by a fresh state', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: 'base 10 is behind 12' });
    expect(session.pending).toBeNull();
    expect(session.notice).toEqual({
      kind: 'reject',
      code: 'stale',
      message: 'session.reject.stale'
    });
    transport.receive(playFrame('H8TQ4R', 12));
    expect(session.version).toBe(12);
    expect(session.canAct).toBe(true);
    // The next act is a new seq on the new base; the notice clears.
    session.act('construct:stone-reserve');
    expect(transport.last()).toMatchObject({ seq: 2, base: 12 });
    expect(session.notice).toBeNull();
  });

  it('keeps the pending request when a reject names another seq', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    transport.receive({ t: 'reject', seq: 0, code: 'rate', msg: 'slow down' });
    expect(session.pending).not.toBeNull();
    expect(session.notice?.message).toBe('session.reject.rate');
  });

  it('maps every reject code, and unknown ones to a generic message', () => {
    const { session, transport } = joined();
    const cases = [
      ['illegal', 'session.reject.illegal'],
      ['not-your-turn', 'session.reject.notYourTurn'],
      ['bad-request', 'session.reject.badRequest'],
      ['something-new', 'session.error.unknown']
    ] as const;
    for (const [code, message] of cases) {
      // The union of reject codes does not know 'something-new'; a newer server might.
      transport.receive({ t: 'reject', seq: 0, code: code as 'illegal', msg: '' });
      expect(session.notice?.message).toBe(message);
    }
  });

  it('re-sends a request dropped by the rate limit, then gives up with a notice', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    const first = transport.last();
    const sends = () => transport.sentOf('act').length;
    for (let tries = 1; tries <= 3; tries++) {
      transport.receive({ t: 'reject', seq: 1, code: 'rate', msg: 'slow down' });
      expect(session.pending).not.toBeNull();
      expect(session.notice).toBeNull();
      vi.advanceTimersByTime(250 * tries - 1);
      expect(sends()).toBe(tries);
      vi.advanceTimersByTime(1);
      expect(sends()).toBe(tries + 1);
      expect(transport.last()).toEqual(first);
    }
    transport.receive({ t: 'reject', seq: 1, code: 'rate', msg: 'slow down' });
    expect(session.pending).toBeNull();
    expect(session.notice?.message).toBe('session.reject.rate');
    vi.advanceTimersByTime(5000);
    expect(sends()).toBe(4);
  });

  it('stops a rate retry once the request settles', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    transport.receive({ t: 'reject', seq: 1, code: 'rate', msg: 'slow down' });
    transport.receive({ ...playFrame('H8TQ4R', 11), ack: { clientId: 'tab-1', seq: 1 } });
    expect(session.pending).toBeNull();
    vi.advanceTimersByTime(1000);
    expect(transport.sentOf('act')).toHaveLength(1);
  });

  it('retries the pending request unchanged', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    const first = transport.last();
    expect(session.retry()).toBe(true);
    expect(transport.last()).toEqual(first);
  });
});

describe('Session: undo', () => {
  it('sends an undo with seq and base only while the frame allows it', () => {
    const { session, transport } = joined();
    expect(session.canUndo).toBe(true);
    expect(session.undo()).toBe(true);
    expect(transport.last()).toEqual({ t: 'undo', code: 'H8TQ4R', seq: 1, base: 10 });
    expect(session.undo()).toBe(false); // pending
    transport.receive({
      ...playFrame('H8TQ4R', 11),
      canUndo: false,
      ack: { clientId: 'tab-1', seq: 1 }
    });
    expect(session.canUndo).toBe(false);
    expect(session.undo()).toBe(false);
  });
});

describe('Session: reconnecting', () => {
  it('says hello, joins again and re-sends the pending request unchanged', () => {
    const { session, transport } = joined();
    session.act('construct:stone-reserve');
    const act = transport.last();
    transport.setState('reconnecting', { attempt: 1 });
    expect(session.conn).toBe('reconnecting');
    expect(session.pending).not.toBeNull();
    transport.sent.length = 0;

    transport.setState('open');
    expect(transport.sent.map((m) => m.t)).toEqual(['hello']);
    expect(transport.sent[0]).toMatchObject({ clientId: 'tab-1', token: identity.token });
    transport.receive({ t: 'welcome', v: ProtocolVersion, server: { version: 'test', rules: 1 } });
    expect(session.conn).toBe('open');
    expect(transport.sent.slice(1)).toEqual([{ t: 'join', code: 'H8TQ4R' }, act]);

    // The join's fresh state, then the ack of the re-sent request.
    transport.receive(playFrame('H8TQ4R', 11));
    expect(session.pending).not.toBeNull();
    transport.receive({ ...playFrame('H8TQ4R', 11), ack: { clientId: 'tab-1', seq: 1 } });
    expect(session.pending).toBeNull();
  });

  it('sends nothing while the link is down', () => {
    const { session, transport } = joined();
    transport.setState('offline', { attempt: 5 });
    expect(session.conn).toBe('offline');
    expect(session.act('construct:stone-reserve')).toBe(false);
    session.resync();
    expect(transport.dropped).toEqual([]);
  });
});

describe('Session: rooms', () => {
  it('creates a room and resolves with its code', async () => {
    const { session, transport } = setup();
    connect(transport);
    const created = session.create({ mode: 'ai', bot: 'easy', seat: 'random' });
    expect(transport.last()).toEqual({
      t: 'create',
      mode: 'ai',
      bot: 'easy',
      seat: 'random',
      names: []
    });
    transport.receive({ t: 'created', code: 'B4TQ7N' });
    await expect(created).resolves.toBe('B4TQ7N');
    expect(session.code).toBe('B4TQ7N');
    transport.receive(playFrame('B4TQ7N', 0));
    expect(session.version).toBe(0);
  });

  it('sends a create made before welcome once welcomed', async () => {
    const { session, transport } = setup();
    const created = session.create({ mode: 'hotseat' });
    transport.setState('open');
    expect(transport.sentOf('create')).toEqual([]);
    transport.receive({ t: 'welcome', v: ProtocolVersion, server: { version: 'test', rules: 1 } });
    expect(transport.sentOf('create')).toEqual([
      { t: 'create', mode: 'hotseat', seat: 0, names: [] }
    ]);
    transport.receive({ t: 'created', code: 'K7QX2M' });
    await expect(created).resolves.toBe('K7QX2M');
  });

  it('rejects a create the server refuses', async () => {
    const { session, transport } = setup();
    connect(transport);
    const busy = session.create({ mode: 'hotseat' });
    transport.receive({ t: 'error', code: 'busy', msg: '' });
    await expect(busy).rejects.toBeInstanceOf(SessionError);
    const rate = session.create({ mode: 'hotseat' });
    transport.receive({ t: 'reject', seq: 0, code: 'rate', msg: '' });
    await expect(rate).rejects.toMatchObject({ code: 'rate', messageKey: 'session.reject.rate' });
  });

  it('reports a join that finds no room, and forgets that room', () => {
    const rooms = new SavedRooms(memoryStorage());
    rooms.remember('GONE42', 'online', 1);
    const { session, transport } = setup({ rooms });
    connect(transport);
    session.join('GONE42');
    transport.receive({ t: 'error', code: 'room-not-found', msg: 'no room with that code' });
    expect(session.roomError).toBe('room-not-found');
    expect(session.notice?.message).toBe('session.error.roomNotFound');
    expect(rooms.list).toEqual([]);
  });

  it('takes a created frame outside a create as the rematch room', () => {
    const { session, transport } = joined();
    session.rematch();
    expect(transport.last()).toEqual({ t: 'rematch', code: 'H8TQ4R' });
    transport.receive({ t: 'created', code: 'NEXT22' });
    expect(session.rematchCode).toBe('NEXT22');
    expect(session.code).toBe('H8TQ4R');
  });

  it('leaves the room it is in, and only tells the server about another', () => {
    const { session, transport } = joined();
    session.leave('OTHER1');
    expect(transport.last()).toEqual({ t: 'leave', code: 'OTHER1' });
    expect(session.code).toBe('H8TQ4R');
    session.leave();
    expect(transport.last()).toEqual({ t: 'leave', code: 'H8TQ4R' });
    expect(session.code).toBeNull();
    expect(session.view).toBeNull();
  });

  it('joining another room drops the old room state at once', () => {
    const { session, transport } = joined();
    session.join('NEXT22');
    expect(session.view).toBeNull();
    expect(transport.last()).toEqual({ t: 'join', code: 'NEXT22' });
    transport.receive(playFrame('H8TQ4R', 11)); // late frame of the old room
    expect(session.view).toBeNull();
  });

  it('close() closes the transport and stops listening', () => {
    const { session, transport } = joined();
    session.close();
    expect(transport.closed).toBe(true);
    transport.receive(playFrame('H8TQ4R', 11));
    expect(session.version).toBe(10);
  });
});
