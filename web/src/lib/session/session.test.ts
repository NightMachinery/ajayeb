import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { GameEvent, HistoryEntry, State } from '$lib/wire';
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

/** The recorded play frame as a room of another mode, seen from `seats`. */
function modeFrame(
  mode: 'ai' | 'online',
  opts: { code?: string; version?: number; seats?: number[]; canUndo?: boolean } = {}
): State {
  const f = playFrame(opts.code ?? 'H8TQ4R', opts.version ?? 10);
  const seats = opts.seats ?? [0];
  f.room = {
    ...f.room,
    mode,
    seats: f.room.seats.map((s) => ({
      ...s,
      you: seats.includes(s.seat),
      ...(mode === 'ai' && s.seat === 1 ? { kind: 'bot' as const, bot: 'easy' as const } : {})
    }))
  };
  f.you = { seats, canAct: seats.includes(f.view.toMove ?? -1), admin: seats.includes(0) };
  f.canUndo = opts.canUndo ?? mode === 'ai';
  return f;
}

/** A session joined to a room of another mode at version 10. */
function joinedAs(mode: 'ai' | 'online', seats: number[] = [0]) {
  const s = setup();
  connect(s.transport);
  s.session.join('H8TQ4R');
  s.transport.receive(modeFrame(mode, { seats }));
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
    g.you = { seats: [1], canAct: true, admin: false }; // toMove is 0
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

  it('forgets the room once its game is finished, and never remembers a finished one', () => {
    const rooms = new SavedRooms(memoryStorage());
    const { transport } = joined({ rooms });
    const over = playFrame('H8TQ4R', 11);
    over.room = { ...over.room, status: 'finished' };
    transport.receive(over);
    expect(rooms.list).toEqual([]);

    const later = setup({ rooms });
    connect(later.transport);
    later.session.join('H8TQ4R');
    later.transport.receive(over);
    expect(later.session.view).not.toBeNull();
    expect(rooms.list).toEqual([]);
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
    f.you = { seats: [0], canAct: true, admin: true };
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

  it('remembers a room it creates before the create resolves, so the room page asks no name', async () => {
    for (const mode of ['ai', 'hotseat', 'online'] as const) {
      const rooms = new SavedRooms(memoryStorage());
      const { session, transport } = setup({ rooms });
      connect(transport);
      const created = session.create(mode === 'ai' ? { mode, bot: 'easy' } : { mode });
      transport.receive({ t: 'created', code: 'K7QX2M' });
      // Home navigates the moment the promise resolves, before any state.
      await created;
      expect(rooms.list.map((r) => [r.code, r.mode])).toEqual([['K7QX2M', mode]]);
    }
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

  it('reports an archived room distinctly, ignores private diagnostics and forgets it', () => {
    const rooms = new SavedRooms(memoryStorage());
    rooms.remember('OLDGAM', 'online', 1);
    const { session, transport } = setup({ rooms });
    connect(transport);
    session.join('OLDGAM');
    transport.receive({ t: 'error', code: 'room-archived', msg: 'private details' });
    expect(session.roomError).toBe('room-archived');
    expect(session.notice?.message).toBe('session.error.roomArchived');
    expect(rooms.list).toEqual([]);
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

  it('takes a created frame outside a create as the rematch room, and remembers it', () => {
    const rooms = new SavedRooms(memoryStorage());
    const { session, transport } = joined({ rooms });
    session.rematch();
    expect(transport.last()).toEqual({ t: 'rematch', code: 'H8TQ4R' });
    transport.receive({ t: 'created', code: 'NEXT22' });
    expect(session.rematchCode).toBe('NEXT22');
    expect(session.code).toBe('H8TQ4R');
    expect(rooms.list.map((r) => [r.code, r.mode])).toContainEqual(['NEXT22', 'hotseat']);
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

describe('Session: an undo that crosses the bot answer', () => {
  it('sends a bot-room undo refused as stale once more with the fresh version', () => {
    const { session, transport } = joinedAs('ai');
    expect(session.undo()).toBe(true);
    expect(transport.last()).toEqual({ t: 'undo', code: 'H8TQ4R', seq: 1, base: 10 });
    // The bot's answer reached the room first.
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: 'x' });
    expect(session.pending).not.toBeNull();
    expect(session.notice).toBeNull();
    transport.receive(modeFrame('ai', { version: 11 }));
    expect(transport.last()).toEqual({ t: 'undo', code: 'H8TQ4R', seq: 1, base: 11 });
    expect(session.pending?.frame).toEqual({ t: 'undo', code: 'H8TQ4R', seq: 1, base: 11 });
    expect(session.notice).toBeNull();
    transport.receive({ ...modeFrame('ai', { version: 12 }), ack: { clientId: 'tab-1', seq: 1 } });
    expect(session.pending).toBeNull();
    expect(session.seq).toBe(1);
  });

  it('sends it again only once, then shows the refusal', () => {
    const { session, transport } = joinedAs('ai');
    session.undo();
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: 'x' });
    transport.receive(modeFrame('ai', { version: 11 }));
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: 'x' });
    expect(session.pending).toBeNull();
    expect(session.notice?.message).toBe('session.reject.stale');
    transport.receive(modeFrame('ai', { version: 12 }));
    expect(transport.sentOf('undo')).toHaveLength(2);
  });

  it('shows the refusal when the fresh state takes no undo', () => {
    const { session, transport } = joinedAs('ai');
    session.undo();
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: 'x' });
    transport.receive(modeFrame('ai', { version: 11, canUndo: false }));
    expect(transport.sentOf('undo')).toHaveLength(1);
    expect(session.pending).toBeNull();
    expect(session.notice?.code).toBe('stale');
  });

  it('never sends again outside a bot room', () => {
    const { session, transport } = joined();
    session.undo();
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: 'x' });
    expect(session.pending).toBeNull();
    expect(session.notice?.message).toBe('session.reject.stale');
    transport.receive(playFrame('H8TQ4R', 11));
    expect(transport.sentOf('undo')).toHaveLength(1);
  });

  it('a stale act is never sent again', () => {
    const { session, transport } = joinedAs('ai');
    const key = session.view!.choices[0]!.key;
    session.act(key);
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: 'x' });
    transport.receive(modeFrame('ai', { version: 11 }));
    expect(transport.sentOf('act')).toHaveLength(1);
    expect(session.pending).toBeNull();
  });
});

describe('Session: the display name', () => {
  it('says hello with a name set before the socket opened, without a new connection', () => {
    const { session, transport } = setup();
    session.setName('  Roxana ');
    expect(session.name).toBe('Roxana');
    transport.setState('open');
    expect(transport.sentOf('hello')[0]?.name).toBe('Roxana');
    expect(transport.reconnects).toBe(0);
  });

  it('asks for a fresh connection when hello already went out under another name', () => {
    const { session, transport } = joined();
    session.setName('Sara');
    expect(transport.reconnects).toBe(0);
    session.setName('Roxana');
    expect(transport.reconnects).toBe(1);
    expect(session.conn).toBe('reconnecting');
    transport.sent.length = 0;
    connect(transport);
    expect(transport.sent).toEqual([
      { t: 'hello', v: ProtocolVersion, token: identity.token, name: 'Roxana', clientId: 'tab-1' },
      { t: 'join', code: 'H8TQ4R' }
    ]);
  });
});

describe('Session: online rooms', () => {
  it('joins by code, case-insensitively, and plays from its seat', () => {
    const { session, transport } = setup();
    connect(transport);
    session.join('h8tq4r');
    expect(transport.last()).toEqual({ t: 'join', code: 'H8TQ4R' });
    transport.receive(modeFrame('online', { seats: [0] }));
    expect(session.room?.mode).toBe('online');
    expect(session.you?.seats).toEqual([0]);
  });

  it('a spectator never acts, never undoes and never asks for a rematch', () => {
    const { session, transport } = joinedAs('online', []);
    expect(session.canAct).toBe(false);
    expect(session.canUndo).toBe(false);
    expect(session.act(session.view!.choices[0]?.key ?? 'x')).toBe(false);
    session.rematch();
    expect(transport.sentOf('rematch')).toEqual([]);
    expect(session.rematchAsked).toBe(false);
  });

  it('a spectator follows a rematch the players made, without asking the server', () => {
    const { session, transport } = joinedAs('online', []);
    transport.receive({ t: 'created', code: 'NEXT22' });
    expect(session.rematchCode).toBe('NEXT22');
    expect(session.rematchAsked).toBe(false);
    session.rematch();
    expect(session.rematchAsked).toBe(true);
    expect(transport.sentOf('rematch')).toEqual([]);
  });

  it('a rematch code the other player asked for waits until this player asks too', () => {
    const { session, transport } = joinedAs('online');
    transport.receive({ t: 'created', code: 'NEXT22' });
    expect(session.rematchCode).toBe('NEXT22');
    expect(session.rematchAsked).toBe(false);
    session.rematch();
    expect(session.rematchAsked).toBe(true);
    expect(transport.last()).toEqual({ t: 'rematch', code: 'H8TQ4R' });
  });

  it('asks again after a reconnect when the rematch answer was lost', () => {
    const { session, transport } = joinedAs('online');
    transport.setState('reconnecting', { attempt: 1 });
    session.rematch();
    expect(session.rematchAsked).toBe(true);
    expect(transport.sentOf('rematch')).toEqual([]);
    connect(transport);
    expect(transport.sent.slice(-2)).toEqual([
      { t: 'join', code: 'H8TQ4R' },
      { t: 'rematch', code: 'H8TQ4R' }
    ]);
  });

  it('joining the rematch room forgets the old rematch', () => {
    const { session, transport } = joinedAs('online');
    session.rematch();
    transport.receive({ t: 'created', code: 'NEXT22' });
    session.join('NEXT22');
    expect(session.rematchCode).toBeNull();
    expect(session.rematchAsked).toBe(false);
    expect(transport.last()).toEqual({ t: 'join', code: 'NEXT22' });
  });
});

/** A frame of the room at `version` and `ply`, with these events and history. */
function plyFrame(
  version: number,
  ply: number,
  opts: { events?: GameEvent[]; history?: HistoryEntry[]; toMove?: 0 | 1 } = {}
): State {
  const f = playFrame('H8TQ4R', version);
  f.view = { ...f.view, ply, toMove: opts.toMove ?? (ply % 2 === 0 ? 0 : 1) };
  f.events = opts.events ?? [];
  if (opts.history) f.history = opts.history;
  else delete f.history;
  return f;
}

function built(seat: 0 | 1, card: string): GameEvent {
  return { kind: 'build', private: false, seat, card };
}

describe('Session: the history', () => {
  function joinedWithHistory(n = 2) {
    const s = setup();
    connect(s.transport);
    s.session.join('H8TQ4R');
    const history: HistoryEntry[] = Array.from({ length: n }, (_, i) => ({
      ply: i,
      seat: (i % 2) as 0 | 1,
      events: [built((i % 2) as 0 | 1, `card-${i}`)]
    }));
    s.transport.receive(plyFrame(n, n, { history }));
    return s;
  }

  it('takes the history of a join frame whole, and its log is every event in order', () => {
    const { session } = joinedWithHistory(3);
    expect(session.history.map((e) => e.ply)).toEqual([0, 1, 2]);
    expect(session.log.map((e) => e.card)).toEqual(['card-0', 'card-1', 'card-2']);
  });

  it('adds one entry per live frame, by the seat that was to move', () => {
    const { session, transport } = joinedWithHistory(2);
    transport.receive(plyFrame(3, 3, { events: [built(0, 'next')] }));
    expect(session.history.at(-1)).toEqual({ ply: 2, seat: 0, events: [built(0, 'next')] });
    // A presence frame (no events, same ply) adds nothing.
    transport.receive(plyFrame(4, 3));
    expect(session.history).toHaveLength(3);
    expect(transport.sentOf('resync')).toEqual([]);
  });

  it('names the forfeiting seat, even when it was not to move', () => {
    const { session, transport } = joinedWithHistory(2);
    transport.receive(
      plyFrame(3, 3, {
        events: [
          { kind: 'resign', private: false, seat: 1 },
          { kind: 'gameOver', private: false, victory: 'forfeit', winner: 0 }
        ]
      })
    );
    expect(session.history.at(-1)?.seat).toBe(1);
  });

  it('replaces the history after an undo, dropping the undone moves', () => {
    const { session, transport } = joinedWithHistory(4);
    transport.receive(plyFrame(5, 2, { history: session.history.slice(0, 2) }));
    expect(session.log.map((e) => e.card)).toEqual(['card-0', 'card-1']);
    transport.receive(plyFrame(6, 3, { events: [built(0, 'again')] }));
    expect(session.history.map((e) => e.ply)).toEqual([0, 1, 2]);
  });

  it('repairs a gap with one resync: merged frames, or a move without its events', () => {
    const { session, transport } = joinedWithHistory(2);
    transport.receive(plyFrame(4, 4, { events: [built(0, 'a'), built(1, 'b')] }));
    expect(session.log.map((e) => e.card).slice(-2)).toEqual(['a', 'b']);
    expect(transport.sentOf('resync')).toEqual([{ t: 'resync', code: 'H8TQ4R' }]);
    // The same version again asks no more.
    transport.receive(plyFrame(4, 4));
    expect(transport.sentOf('resync')).toHaveLength(1);
    transport.receive(plyFrame(5, 6));
    expect(transport.sentOf('resync')).toHaveLength(2);
  });

  it('starts a history from live frames when the server sent none', () => {
    const { session } = joined();
    expect(session.history).toHaveLength(1);
    expect(session.log).toEqual(playFrame().events);
  });

  it('forgets the history when the room is left', () => {
    const { session } = joinedWithHistory(3);
    session.leave();
    expect(session.history).toEqual([]);
    expect(session.log).toEqual([]);
  });
});

describe('Session: the forfeit', () => {
  it('forfeits the seat to move in hot-seat, naming it', () => {
    const { session, transport } = joined();
    expect(session.resignSeat).toBe(0);
    expect(session.resign()).toBe(true);
    expect(transport.last()).toEqual({
      t: 'act',
      code: 'H8TQ4R',
      seq: 1,
      base: 10,
      key: 'resign:0',
      seat: 0
    });
    // One request at a time.
    expect(session.resignSeat).toBeNull();
    expect(session.resign()).toBe(false);
  });

  it("forfeits the player's own seat online, whoever is to move", () => {
    const { session, transport } = joinedAs('online', [1]);
    expect(session.canAct).toBe(false);
    expect(session.resign()).toBe(true);
    expect(transport.last()).toEqual({
      t: 'act',
      code: 'H8TQ4R',
      seq: 1,
      base: 10,
      key: 'resign:1'
    });
  });

  it('is not offered to a spectator, in the lobby, or after the end', () => {
    expect(joinedAs('online', []).session.resignSeat).toBeNull();
    const lobby = joinedAs('online');
    const f = modeFrame('online', { version: 11 });
    lobby.transport.receive({ ...f, room: { ...f.room, status: 'lobby' } });
    expect(lobby.session.resignSeat).toBeNull();
    const over = joinedAs('ai');
    const g = modeFrame('ai', { version: 11 });
    over.transport.receive({ ...g, room: { ...g.room, status: 'finished' } });
    expect(over.session.resign()).toBe(false);
  });

  it('sends a forfeit refused as stale once more with the fresh version', () => {
    const { session, transport } = joinedAs('ai');
    session.resign();
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: '' });
    expect(session.notice).toBeNull();
    transport.receive(modeFrame('ai', { version: 11 }));
    expect(transport.last()).toEqual({
      t: 'act',
      code: 'H8TQ4R',
      seq: 1,
      base: 11,
      key: 'resign:0'
    });
    // A second stale refusal shows.
    transport.receive({ t: 'reject', seq: 1, code: 'stale', msg: '' });
    expect(session.notice?.code).toBe('stale');
    expect(session.pending).toBeNull();
  });
});

describe('Session: hint settings', () => {
  const hints = recorded('state-play').room.hints;

  /** The recorded play frame of an online room, with this user as its admin or not. */
  function onlineFrame(admin: boolean, version = 10): State {
    const f = modeFrame('online', { version });
    f.you = { ...f.you, admin };
    return f;
  }

  it('reads the settings, the admin flag and the room log from the frame', () => {
    const { session, transport } = setup();
    connect(transport);
    session.join('H8TQ4R');
    expect(session.hints).toBeNull();
    expect(session.roomLog).toEqual([]);
    const f = recorded('state-hints-changed');
    delete f.ack;
    transport.receive({ ...f, code: 'H8TQ4R' });
    expect(session.hints).toEqual(f.room.hints);
    // The recorded frame is seat 1's; seat 0 created the room.
    expect(session.admin).toBe(false);
    expect(session.canSetHints).toBe(false);
    expect(session.roomLog).toEqual([
      { kind: 'hints', at: 4, seat: 0, on: ['advisor'], off: ['prices', 'threats'] }
    ]);
  });

  it('sends the admin’s settings with the next seq, and settles on the state that acks it', () => {
    const { session, transport } = setup();
    connect(transport);
    session.join('H8TQ4R');
    transport.receive(onlineFrame(true));
    expect(session.canSetHints).toBe(true);
    const next = { ...hints, prices: false };
    expect(session.setHints(next)).toBe(true);
    expect(transport.last()).toEqual({ t: 'hints', code: 'H8TQ4R', seq: 1, hints: next });
    expect(session.pending?.frame.t).toBe('hints');
    expect(session.canSetHints).toBe(false);
    expect(session.canAct).toBe(false);
    expect(session.setHints(hints)).toBe(false);

    const after = onlineFrame(true);
    after.room = { ...after.room, hints: next };
    transport.receive({ ...after, ack: { clientId: 'tab-1', seq: 1 } });
    expect(session.pending).toBeNull();
    expect(session.hints).toEqual(next);
    // The seq counter is shared with acts.
    session.act('construct:stone-reserve');
    expect(transport.last()).toMatchObject({ t: 'act', seq: 2 });
  });

  it('sends nothing for a player who is not the admin, or after the end', () => {
    const { session, transport } = setup();
    connect(transport);
    session.join('H8TQ4R');
    transport.receive(onlineFrame(false));
    expect(session.admin).toBe(false);
    expect(session.canSetHints).toBe(false);
    expect(session.setHints(hints)).toBe(false);
    expect(transport.sentOf('hints')).toEqual([]);

    const over = recorded('state-game-over');
    delete over.ack;
    transport.receive({ ...over, code: 'H8TQ4R', version: 99, you: { ...over.you, admin: true } });
    expect(session.admin).toBe(true);
    expect(session.canSetHints).toBe(false);
  });

  it('shows the refusal of a change the server refused', () => {
    const { session, transport } = setup();
    connect(transport);
    session.join('H8TQ4R');
    transport.receive(onlineFrame(true));
    session.setHints({ ...hints, recap: false });
    transport.receive({ t: 'reject', seq: 1, code: 'not-admin', msg: '' });
    expect(session.pending).toBeNull();
    expect(session.notice?.message).toBe('session.reject.notAdmin');
    transport.receive({ t: 'reject', seq: 0, code: 'game-over', msg: '' });
    expect(session.notice?.message).toBe('session.reject.gameOver');
  });

  it('re-sends a pending change unchanged after a reconnect', () => {
    const { session, transport } = setup();
    connect(transport);
    session.join('H8TQ4R');
    transport.receive(onlineFrame(true));
    session.setHints({ ...hints, tempo: false });
    transport.setState('reconnecting', { attempt: 1 });
    connect(transport);
    expect(transport.sentOf('hints')).toHaveLength(2);
    expect(transport.sentOf('hints')[1]).toEqual(transport.sentOf('hints')[0]);
  });

  it('creates a game with the settings given, else the device default, else none', () => {
    const a = setup();
    connect(a.transport);
    void a.session.create({ mode: 'online', hints: { ...hints, science: false } });
    expect(a.transport.last()).toMatchObject({ t: 'create', hints: { ...hints, science: false } });

    const transport = new ScriptedTransport();
    const asked: string[] = [];
    const session = new Session({
      transport,
      identity,
      rooms: null,
      defaultHints: (mode) => {
        asked.push(mode);
        return { ...hints, counting: false };
      }
    });
    connect(transport);
    void session.create({ mode: 'hotseat', names: ['A', 'B'] });
    expect(asked).toEqual(['hotseat']);
    expect(transport.last()).toMatchObject({ t: 'create', hints: { ...hints, counting: false } });

    const c = setup();
    connect(c.transport);
    void c.session.create({ mode: 'ai', bot: 'easy', seat: 'random' });
    expect(c.transport.last()).not.toHaveProperty('hints');
  });
});

describe('Session: device links', () => {
  const CLAIM = 'q3Vb8Kx0Zr7TnW2mYc5JhA';
  const NEW = 'Hw4pZ9cN1sKe6uTb0LgQvY';

  function savedRooms(): SavedRooms {
    return new SavedRooms(memoryStorage());
  }

  it('joins with a claim, again after a reconnect, and saves the room with it', () => {
    const rooms = savedRooms();
    const { transport, session } = setup({ rooms });
    connect(transport);
    session.join('H8TQ4R', CLAIM);
    expect(transport.sentOf('join')).toEqual([{ t: 'join', code: 'H8TQ4R', claim: CLAIM }]);
    expect(session.claim).toBe(CLAIM);
    transport.receive(modeFrame('online', { seats: [0] }));
    expect(rooms.claimFor('H8TQ4R')).toBe(CLAIM);
    transport.setState('reconnecting');
    connect(transport);
    expect(transport.sentOf('join').at(-1)).toEqual({ t: 'join', code: 'H8TQ4R', claim: CLAIM });
  });

  it('joins without a claim as before', () => {
    const { transport, session } = setup();
    connect(transport);
    session.join('H8TQ4R');
    expect(transport.sentOf('join')).toEqual([{ t: 'join', code: 'H8TQ4R' }]);
    expect(session.claim).toBeNull();
  });

  it('asks for a seat’s link and resolves with the claim', async () => {
    const { transport, session } = joinedAs('online', [0]);
    const got = session.requestLink(1);
    expect(transport.last()).toEqual({ t: 'link', code: 'H8TQ4R', seat: 1 });
    transport.receive({ t: 'linked', code: 'H8TQ4R', seat: 1, claim: CLAIM });
    await expect(got).resolves.toBe(CLAIM);
  });

  it('rejects a refused link request, and one that gets no answer', async () => {
    const { transport, session } = joinedAs('online', [1]);
    const refused = session.requestLink(0);
    transport.receive({ t: 'reject', seq: 0, code: 'not-admin', msg: '' });
    await expect(refused).rejects.toBeInstanceOf(SessionError);
    const silent = session.requestLink(0);
    vi.advanceTimersByTime(10_000);
    await expect(silent).rejects.toBeInstanceOf(SessionError);
  });

  it('a reset sends relink; a device on a claim that resets its own seat moves to the new claim', async () => {
    const rooms = savedRooms();
    const { transport, session } = setup({ rooms });
    connect(transport);
    session.join('H8TQ4R', CLAIM);
    transport.receive(modeFrame('online', { seats: [0] }));
    const got = session.resetLink(0);
    expect(transport.last()).toEqual({ t: 'relink', code: 'H8TQ4R', seat: 0 });
    transport.receive({ t: 'linked', code: 'H8TQ4R', seat: 0, claim: NEW });
    await expect(got).resolves.toBe(NEW);
    expect(session.claim).toBe(NEW);
    expect(rooms.claimFor('H8TQ4R')).toBe(NEW);
    transport.setState('reconnecting');
    connect(transport);
    expect(transport.sentOf('join').at(-1)).toEqual({ t: 'join', code: 'H8TQ4R', claim: NEW });
  });

  it('a reset of another seat leaves this device’s claim alone', async () => {
    const { transport, session } = setup();
    connect(transport);
    session.join('H8TQ4R', CLAIM);
    transport.receive(modeFrame('online', { seats: [0] }));
    const got = session.resetLink(1);
    transport.receive({ t: 'linked', code: 'H8TQ4R', seat: 1, claim: NEW });
    await got;
    expect(session.claim).toBe(CLAIM);
  });

  it('a bad claim shows the room error and drops the saved claim, even mid-game', () => {
    const rooms = savedRooms();
    const { transport, session } = setup({ rooms });
    connect(transport);
    session.join('H8TQ4R', CLAIM);
    transport.receive(modeFrame('online', { seats: [0] }));
    expect(rooms.claimFor('H8TQ4R')).toBe(CLAIM);
    transport.receive({ t: 'error', code: 'bad-claim', msg: '' });
    expect(session.roomError).toBe('bad-claim');
    expect(rooms.claimFor('H8TQ4R')).toBeNull();
    expect(rooms.list.map((r) => r.code)).toEqual(['H8TQ4R']);
  });

  it('a rematch announced with a claim keeps it for the new room', () => {
    const rooms = savedRooms();
    const { transport, session } = setup({ rooms });
    connect(transport);
    session.join('H8TQ4R', CLAIM);
    transport.receive(modeFrame('online', { seats: [0] }));
    transport.receive({ t: 'created', code: 'R7MQ3X', claim: NEW });
    expect(session.rematchCode).toBe('R7MQ3X');
    expect(session.rematchClaim).toBe(NEW);
    expect(rooms.claimFor('R7MQ3X')).toBe(NEW);
  });
});

describe('Session: allowUndo', () => {
  /** A create a later one supersedes rejects; these tests only read what was sent. */
  function quiet(p: Promise<string>): void {
    p.catch(() => {});
  }

  it('creates a hot-seat or ai game with the choice given, else the device default, else none', () => {
    const a = setup();
    connect(a.transport);
    quiet(a.session.create({ mode: 'hotseat', allowUndo: false }));
    expect(a.transport.last()).toMatchObject({ t: 'create', mode: 'hotseat', allowUndo: false });

    const transport = new ScriptedTransport();
    const asked: string[] = [];
    const session = new Session({
      transport,
      identity,
      rooms: null,
      defaultAllowUndo: (mode) => {
        asked.push(mode);
        return mode === 'ai' ? false : null;
      }
    });
    connect(transport);
    quiet(session.create({ mode: 'ai', bot: 'easy' }));
    expect(transport.last()).toMatchObject({ t: 'create', allowUndo: false });
    quiet(session.create({ mode: 'hotseat' }));
    expect(transport.last()).not.toHaveProperty('allowUndo');
    // An online game has no undo: the field is never sent, not even when asked.
    quiet(session.create({ mode: 'online', allowUndo: true }));
    expect(transport.last()).not.toHaveProperty('allowUndo');
    expect(asked).toEqual(['ai', 'hotseat']);
  });

  it("reads the room's allowUndo, and shows undo-off as its own refusal", () => {
    const { session, transport } = joined();
    expect(session.allowUndo).toBe(true);
    const off = playFrame('H8TQ4R', 11);
    off.room = { ...off.room, allowUndo: false };
    off.canUndo = false;
    transport.receive(off);
    expect(session.allowUndo).toBe(false);
    expect(session.canUndo).toBe(false);
    expect(session.undo()).toBe(false);
    transport.receive({ t: 'reject', seq: 0, code: 'undo-off', msg: '' });
    expect(session.notice).toMatchObject({ code: 'undo-off', message: 'session.reject.undoOff' });
  });
});
