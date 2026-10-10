import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ServerMessage } from '$lib/wire';
import { backoffCeiling, backoffDelay } from './backoff';
import type { TransportStatus } from './transport';
import {
  defaultSocketUrl,
  fatalClose,
  WsTransport,
  type NetworkLike,
  type SocketLike
} from './ws-transport';

class FakeSocket implements SocketLike {
  readyState = 0;
  onopen: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  sent: string[] = [];
  closedWith: number | undefined;

  constructor(readonly url: string) {}

  send(data: string): void {
    this.sent.push(data);
  }

  close(code?: number): void {
    this.closedWith = code ?? 1005;
    this.readyState = 3;
  }

  // Test controls.
  open(): void {
    this.readyState = 1;
    this.onopen?.(new Event('open'));
  }

  drop(code = 1006, reason = ''): void {
    this.readyState = 3;
    this.onclose?.({ code, reason } as CloseEvent);
  }

  message(data: unknown): void {
    this.onmessage?.({ data } as MessageEvent);
  }
}

const WELCOME = '{"t":"welcome","v":1,"server":{"version":"test","rules":1}}';

class FakeNetwork implements NetworkLike {
  onLine = true;
  #fns = new Map<string, Set<() => void>>();
  addEventListener(type: 'online' | 'offline', fn: () => void): void {
    if (!this.#fns.has(type)) this.#fns.set(type, new Set());
    this.#fns.get(type)!.add(fn);
  }
  removeEventListener(type: 'online' | 'offline', fn: () => void): void {
    this.#fns.get(type)?.delete(fn);
  }
  fire(type: 'online' | 'offline'): void {
    this.onLine = type === 'online';
    for (const fn of this.#fns.get(type) ?? []) fn();
  }
  listeners(): number {
    let n = 0;
    for (const s of this.#fns.values()) n += s.size;
    return n;
  }
}

function setup(opts: { random?: () => number; offlineAfter?: number } = {}) {
  const sockets: FakeSocket[] = [];
  const network = new FakeNetwork();
  const transport = new WsTransport({
    url: 'ws://example.test/ws',
    createSocket: (url) => {
      const s = new FakeSocket(url);
      sockets.push(s);
      return s;
    },
    network,
    random: opts.random ?? (() => 1),
    offlineAfter: opts.offlineAfter ?? 4,
    connectTimeoutMs: 10_000
  });
  const states: TransportStatus[] = [];
  transport.onState((s) => states.push(s));
  const messages: ServerMessage[] = [];
  transport.onMessage((m) => messages.push(m));
  const current = () => sockets[sockets.length - 1]!;
  return { transport, sockets, network, states, messages, current };
}

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

describe('backoff', () => {
  it('doubles from the base up to the cap', () => {
    const opts = { baseMs: 500, maxMs: 15_000 };
    expect([0, 1, 2, 3, 4, 5, 6, 50].map((n) => backoffCeiling(n, opts))).toEqual([
      500, 1000, 2000, 4000, 8000, 15_000, 15_000, 15_000
    ]);
    expect(backoffCeiling(-3, opts)).toBe(500);
    expect(backoffCeiling(5000, opts)).toBe(15_000);
  });

  it('jitters between half the ceiling and the ceiling', () => {
    expect(backoffDelay(2, () => 0)).toBe(1000);
    expect(backoffDelay(2, () => 0.5)).toBe(1500);
    expect(backoffDelay(2, () => 0.999999)).toBe(2000);
    // Out-of-range randomness is clamped rather than trusted.
    expect(backoffDelay(2, () => 7)).toBe(2000);
    expect(backoffDelay(2, () => -1)).toBe(1000);
    for (let i = 0; i < 200; i++) {
      const d = backoffDelay(30, Math.random);
      expect(d).toBeGreaterThanOrEqual(7500);
      expect(d).toBeLessThanOrEqual(15_000);
    }
  });
});

describe('WsTransport', () => {
  it('connects to the URL and reports open', () => {
    const { sockets, states, current } = setup();
    expect(sockets).toHaveLength(1);
    expect(current().url).toBe('ws://example.test/ws');
    expect(states.map((s) => s.state)).toEqual(['connecting']);
    current().open();
    expect(states.at(-1)).toEqual({ state: 'open', attempt: 0 });
  });

  it('sends JSON only while open', () => {
    const { transport, current } = setup();
    expect(transport.send({ t: 'resync', code: 'K7QX2M' })).toBe(false);
    current().open();
    expect(transport.send({ t: 'resync', code: 'K7QX2M' })).toBe(true);
    expect(current().sent).toEqual(['{"t":"resync","code":"K7QX2M"}']);
  });

  it('delivers JSON text frames and skips anything else', () => {
    const { messages, current } = setup();
    current().open();
    current().message('{"t":"created","code":"K7QX2M"}');
    current().message('not json');
    current().message('{"no":"type"}');
    current().message(new ArrayBuffer(4));
    expect(messages).toEqual([{ t: 'created', code: 'K7QX2M' }]);
  });

  it('reconnects after a drop with exponential backoff and jitter, capped', () => {
    // random() = 1 makes every delay its ceiling: 500, 1000, 2000 ... 15000.
    const { sockets, states, current } = setup({ offlineAfter: 100 });
    current().open();
    const ceilings = [500, 1000, 2000, 4000, 8000, 15_000, 15_000];
    for (const [i, ceiling] of ceilings.entries()) {
      current().drop(1006);
      expect(states.at(-1)).toEqual({ state: 'reconnecting', attempt: i + 1 });
      const before = sockets.length;
      vi.advanceTimersByTime(ceiling - 1);
      expect(sockets.length, `try ${i + 1} fired early`).toBe(before);
      vi.advanceTimersByTime(1);
      expect(sockets.length, `try ${i + 1} did not fire`).toBe(before + 1);
    }
    // A socket the server welcomes resets the count.
    current().open();
    expect(states.at(-1)).toEqual({ state: 'open', attempt: 0 });
    current().message(WELCOME);
    current().drop(1012, 'service restart');
    vi.advanceTimersByTime(500);
    expect(sockets).toHaveLength(ceilings.length + 2);
  });

  it('keeps backing off from a server that accepts and then closes at once', () => {
    // random() = 1 makes every delay its ceiling. Each socket opens and is
    // closed straight away, before any welcome: without the fix the count
    // reset on every open, and the client hammered the server every 500 ms.
    const { sockets, states, current } = setup({ offlineAfter: 100 });
    const ceilings = [500, 1000, 2000, 4000, 8000, 15_000, 15_000];
    for (const [i, ceiling] of ceilings.entries()) {
      current().open();
      current().drop(1006);
      expect(states.at(-1)).toEqual({ state: 'reconnecting', attempt: i + 1 });
      const before = sockets.length;
      vi.advanceTimersByTime(ceiling - 1);
      expect(sockets.length, `try ${i + 1} fired early`).toBe(before);
      vi.advanceTimersByTime(1);
      expect(sockets.length, `try ${i + 1} did not fire`).toBe(before + 1);
    }
    // Other frames before the close do not count as a welcome.
    current().open();
    current().message('{"t":"error","code":"busy","msg":""}');
    current().drop(1006);
    vi.advanceTimersByTime(14_999);
    expect(sockets).toHaveLength(ceilings.length + 1);
    vi.advanceTimersByTime(1);
    expect(sockets).toHaveLength(ceilings.length + 2);
    // A welcomed socket starts the backoff over.
    current().open();
    current().message(WELCOME);
    current().drop(1006);
    vi.advanceTimersByTime(500);
    expect(sockets).toHaveLength(ceilings.length + 3);
  });

  it('ignores a welcome that arrives on an abandoned socket', () => {
    const { sockets, current } = setup({ offlineAfter: 100 });
    current().open();
    current().drop(1006); // try 1: 500 ms
    vi.advanceTimersByTime(500);
    current().open();
    const old = current();
    current().drop(1006); // try 2: 1000 ms
    // The old socket is detached; a stray late frame on it changes nothing.
    old.message(WELCOME);
    vi.advanceTimersByTime(1000);
    current().drop(1006); // try 3: 2000 ms, not 500
    vi.advanceTimersByTime(1999);
    expect(sockets).toHaveLength(3);
    vi.advanceTimersByTime(1);
    expect(sockets).toHaveLength(4);
  });

  it('waits at least half the ceiling with the smallest jitter', () => {
    const { sockets, current } = setup({ random: () => 0 });
    current().drop(1006);
    vi.advanceTimersByTime(249);
    expect(sockets).toHaveLength(1);
    vi.advanceTimersByTime(1);
    expect(sockets).toHaveLength(2);
  });

  it('reads offline after several failed tries in a row, and keeps trying', () => {
    const { sockets, states, current } = setup({ offlineAfter: 2 });
    current().drop(1006);
    expect(states.at(-1)?.state).toBe('reconnecting');
    vi.advanceTimersByTime(500);
    current().drop(1006);
    expect(states.at(-1)?.state).toBe('reconnecting');
    vi.advanceTimersByTime(1000);
    current().drop(1006);
    expect(states.at(-1)).toEqual({ state: 'offline', attempt: 3 });
    vi.advanceTimersByTime(2000);
    expect(sockets).toHaveLength(4);
  });

  it('honours the policy closes another try cannot fix', () => {
    expect(fatalClose(1008, 'protocol')).toBe('protocol');
    expect(fatalClose(1008, 'bad-hello')).toBe('bad-hello');
    expect(fatalClose(1008, 'rate limit')).toBeUndefined();
    expect(fatalClose(1008, 'client too slow')).toBeUndefined();
    expect(fatalClose(1008, 'hello timeout')).toBeUndefined();
    expect(fatalClose(1009, '')).toBeUndefined();

    const { sockets, states, current, network } = setup();
    current().open();
    current().drop(1008, 'protocol');
    expect(states.at(-1)).toEqual({ state: 'closed', fatal: 'protocol', attempt: 0 });
    vi.advanceTimersByTime(60_000);
    expect(sockets).toHaveLength(1);
    expect(network.listeners()).toBe(0);
  });

  it('retries after the other policy closes', () => {
    for (const reason of ['rate limit', 'client too slow', 'hello timeout']) {
      const { sockets, current } = setup();
      current().open();
      current().drop(1008, reason);
      vi.advanceTimersByTime(500);
      expect(sockets, reason).toHaveLength(2);
    }
  });

  it('does not reconnect after close()', () => {
    const { transport, sockets, states, current, network } = setup();
    current().open();
    const socket = current();
    transport.close();
    expect(socket.closedWith).toBe(1000);
    expect(states.at(-1)?.state).toBe('closed');
    expect(transport.send({ t: 'resync', code: 'K7QX2M' })).toBe(false);
    vi.advanceTimersByTime(60_000);
    expect(sockets).toHaveLength(1);
    expect(network.listeners()).toBe(0);
  });

  it('gives up on a socket that does not open in time', () => {
    const { sockets, states, current } = setup();
    const first = current();
    vi.advanceTimersByTime(10_000);
    expect(first.closedWith).toBeDefined();
    expect(states.at(-1)?.state).toBe('reconnecting');
    vi.advanceTimersByTime(500);
    expect(sockets).toHaveLength(2);
    // The abandoned socket's late events change nothing.
    first.drop(1006);
    expect(sockets).toHaveLength(2);
  });

  it('tries at once when the browser comes back online', () => {
    const { sockets, states, current, network } = setup({ offlineAfter: 100 });
    current().open();
    network.fire('offline');
    expect(states.at(-1)?.state).toBe('open'); // the socket may still be alive
    current().drop(1006);
    expect(states.at(-1)?.state).toBe('offline'); // navigator is offline
    for (let i = 0; i < 3; i++) {
      vi.advanceTimersByTime(20_000);
      current().drop(1006);
    }
    const before = sockets.length;
    network.fire('online');
    expect(sockets).toHaveLength(before + 1);
  });

  it('builds the default URL from the page origin', () => {
    expect(defaultSocketUrl({ protocol: 'https:', host: 'example.org' })).toBe(
      'wss://example.org/ws'
    );
    expect(defaultSocketUrl({ protocol: 'http:', host: '127.0.0.1:38951' })).toBe(
      'ws://127.0.0.1:38951/ws'
    );
  });
});
