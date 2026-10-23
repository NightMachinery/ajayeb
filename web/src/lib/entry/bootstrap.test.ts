import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { RoomBootstrap, snapshotKind } from './bootstrap';
import type { SocketLike } from '$lib/session/ws-transport';
class Socket implements SocketLike {
  readyState = 0;
  onopen: SocketLike['onopen'] = null;
  onclose: SocketLike['onclose'] = null;
  onmessage: SocketLike['onmessage'] = null;
  onerror: SocketLike['onerror'] = null;
  sent: Record<string, unknown>[] = [];
  send(data: string) {
    this.sent.push(JSON.parse(data));
  }
  close() {
    this.readyState = 3;
  }
  open() {
    this.readyState = 1;
    this.onopen?.(new Event('open'));
  }
  message(value: unknown) {
    this.onmessage?.({ data: JSON.stringify(value) } as MessageEvent);
  }
  drop() {
    this.onclose?.({ code: 1012, reason: 'restart' } as CloseEvent);
  }
}
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});
function setup() {
  const sockets: Socket[] = [],
    snapshots = vi.fn(),
    error = vi.fn();
  const bootstrap = new RoomBootstrap({
    code: 'REGAAA',
    claim: 'claim',
    identity: { token: 'token', clientId: 'tab', name: 'Name' },
    socket: () => {
      const s = new Socket();
      sockets.push(s);
      return s;
    },
    network: null,
    snapshot: snapshots,
    error,
    status: vi.fn()
  });
  return { bootstrap, sockets, snapshots, error };
}
it('welcomes before join, retains only the newest unknown snapshot and closes without leave', () => {
  const { bootstrap, sockets, snapshots } = setup();
  const socket = sockets[0]!;
  expect(socket.sent).toEqual([]);
  socket.open();
  expect(socket.sent).toEqual([
    { t: 'hello', v: 2, token: 'token', clientId: 'tab', name: 'Name' }
  ]);
  socket.message({ t: 'welcome', v: 2 });
  socket.message({ t: 'welcome', v: 2 });
  expect(socket.sent[1]).toEqual({ t: 'join', code: 'REGAAA', claim: 'claim' });
  expect(socket.sent).toHaveLength(2);
  socket.message({ t: 'state', code: 'OTHER', view: {} });
  expect(snapshots).not.toHaveBeenCalled();
  socket.message({
    t: 'table-state',
    game: 'haftgane',
    code: 'REGAAA',
    view: { private: 'first' }
  });
  socket.message({
    t: 'table-state',
    game: 'haftgane',
    code: 'REGAAA',
    view: { private: 'latest' }
  });
  expect(bootstrap.latest).toEqual({
    t: 'table-state',
    game: 'haftgane',
    code: 'REGAAA',
    view: { private: 'latest' }
  });
  expect(snapshots).toHaveBeenCalledTimes(2);
  bootstrap.close();
  expect(bootstrap.latest).toBeUndefined();
  vi.runAllTimers();
  expect(sockets).toHaveLength(1);
  expect(socket.sent.some((m) => m['t'] === 'leave')).toBe(false);
});
it('reconnects through a new v2 hello and identical identity/claim without replaying buffered events', () => {
  const { bootstrap, sockets } = setup();
  const first = sockets[0]!;
  first.open();
  first.message({ t: 'welcome', v: 2 });
  first.drop();
  vi.advanceTimersByTime(501);
  expect(sockets).toHaveLength(2);
  const next = sockets[1]!;
  next.open();
  next.message({ t: 'welcome', v: 2 });
  expect(next.sent).toEqual(first.sent);
  bootstrap.close();
});
it('stops on an old-server protocol refusal or an invalid welcome', () => {
  for (const reply of [
    { t: 'error', code: 'protocol' },
    { t: 'welcome', v: 1 }
  ]) {
    const { bootstrap, sockets, error } = setup();
    const socket = sockets[0]!;
    socket.open();
    socket.message(reply);
    vi.runAllTimers();
    expect(error).toHaveBeenCalledWith('protocol');
    expect(sockets).toHaveLength(1);
    expect(socket.sent).toHaveLength(1);
    bootstrap.close();
  }
  expect(snapshotKind({ t: 'table-state', game: 'duel' })).toBeNull();
  expect(snapshotKind({ t: 'state' })).toBe('duel');
});

it('shows offline without opening a socket and resumes discovery on the online event', () => {
  const network = new EventTarget();
  const navigatorState = { onLine: false };
  vi.stubGlobal('window', network);
  vi.stubGlobal('navigator', navigatorState);
  const sockets: Socket[] = [],
    status = vi.fn();
  const bootstrap = new RoomBootstrap({
    code: 'REGAAA',
    claim: null,
    identity: { token: 'token', clientId: 'tab', name: 'Name' },
    socket: () => {
      const s = new Socket();
      sockets.push(s);
      return s;
    },
    snapshot: vi.fn(),
    error: vi.fn(),
    status
  });
  expect(sockets).toHaveLength(0);
  expect(status).toHaveBeenLastCalledWith({ state: 'offline', attempt: 0 });
  navigatorState.onLine = true;
  network.dispatchEvent(new Event('online'));
  expect(sockets).toHaveLength(1);
  sockets[0]!.open();
  sockets[0]!.message({ t: 'welcome', v: 2 });
  expect(sockets[0]!.sent.map((m) => m['t'])).toEqual(['hello', 'join']);
  bootstrap.close();
  network.dispatchEvent(new Event('online'));
  expect(sockets).toHaveLength(1);
});

it('hands the newest snapshot and every later frame to an attached session', () => {
  const { bootstrap, sockets, snapshots, error } = setup();
  const socket = sockets[0]!;
  socket.open();
  expect(bootstrap.send({ t: 'resync', code: 'REGAAA' })).toBe(false); // not welcomed yet
  socket.message({ t: 'welcome', v: 2 });
  const first = { t: 'table-state', game: 'haftgane', code: 'REGAAA', revision: 1 };
  socket.message(first);
  const messages: unknown[] = [],
    statuses: string[] = [];
  let joins = 0;
  expect(
    bootstrap.attach({
      message: (m) => messages.push(m),
      status: (s) => statuses.push(s.state),
      joined: () => joins++
    })
  ).toEqual(first);
  expect(bootstrap.latest).toBeUndefined();
  expect(bootstrap.status.state).toBe('open');
  // Later snapshots, private refusals and a dropped resync's Duel-family
  // reject reach the session; none ends the entry or is kept.
  socket.message({ ...first, revision: 2 });
  socket.message({ t: 'table-reject', seq: 3, code: 'stale' });
  socket.message({ t: 'reject', seq: 0, code: 'rate', msg: '' });
  socket.message({ t: 'linked', code: 'REGAAA', seat: 1, claim: 'c' });
  expect(messages).toHaveLength(4);
  expect(bootstrap.latest).toBeUndefined();
  expect(snapshots).toHaveBeenCalledTimes(1);
  expect(error).not.toHaveBeenCalled();
  expect(bootstrap.send({ t: 'table-undo', code: 'REGAAA', seq: 4, base: 2 })).toBe(true);
  expect(socket.sent.at(-1)).toEqual({ t: 'table-undo', code: 'REGAAA', seq: 4, base: 2 });

  // A new socket says hello and joins with the claim in use, then tells the session.
  bootstrap.useClaim('fresh');
  socket.drop();
  expect(statuses).toEqual(['reconnecting']);
  vi.advanceTimersByTime(501);
  const next = sockets[1]!;
  next.open();
  next.message({ t: 'welcome', v: 2 });
  expect(next.sent.map((m) => m['t'])).toEqual(['hello', 'join']);
  expect(next.sent[1]).toEqual({ t: 'join', code: 'REGAAA', claim: 'fresh' });
  expect(statuses.at(-1)).toBe('open');
  expect(joins).toBe(1);

  // An error frame still ends the entry, such as a revoked device link.
  next.message({ t: 'error', code: 'bad-claim' });
  expect(error).toHaveBeenCalledWith('bad-claim');
  expect(bootstrap.send({ t: 'resync', code: 'REGAAA' })).toBe(false);
});

it('ends the entry when the session finds a frame it cannot read', () => {
  const { bootstrap, sockets, error } = setup();
  sockets[0]!.open();
  sockets[0]!.message({ t: 'welcome', v: 2 });
  bootstrap.attach({ message: () => {}, status: () => {}, joined: () => {} });
  bootstrap.fail('protocol');
  expect(error).toHaveBeenCalledWith('protocol');
  vi.runAllTimers();
  expect(sockets).toHaveLength(1);
});
