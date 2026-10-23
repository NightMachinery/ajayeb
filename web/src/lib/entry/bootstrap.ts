// Discovery owns an unknown snapshot, not a Duel view or an event stream.
// Once a game's entry has validated that snapshot, the entry's Session
// attaches: the bootstrap hands it the newest snapshot, keeps nothing more,
// and passes it every later frame of the room, while it goes on saying
// hello and joining on every new socket.
import { backoffDelay, DEFAULT_BACKOFF } from '$lib/session/backoff';
import type { Identity } from '$lib/session/identity';
import { defaultSocketUrl, fatalClose, type SocketLike } from '$lib/session/ws-transport';
import type { TransportStatus } from '$lib/session/transport';

export function object(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}
export function snapshotKind(value: unknown): 'duel' | 'haftgane' | 'lobby' | null {
  const raw = object(value);
  if (raw?.['t'] === 'state') return 'duel';
  if (raw?.['t'] === 'table-state' && raw['game'] === 'haftgane') return 'haftgane';
  if (raw?.['t'] === 'lobby-state' && raw['game'] === 'lobby') return 'lobby';
  return null;
}
/** A game Session that took over the connection (attach). */
export interface RoomListener {
  /** Every frame after the handover except welcome and error. */
  message(message: unknown): void;
  /** The link's state, as the bootstrap reports it. */
  status(status: TransportStatus): void;
  /** A new socket was welcomed and the room's join sent on it. */
  joined(): void;
}
export interface BootstrapOptions {
  code: string;
  claim: string | null;
  identity: Identity;
  snapshot: () => void;
  error: (code: string) => void;
  status: (status: TransportStatus) => void;
  socket?: () => SocketLike;
  network?: Pick<Window, 'addEventListener' | 'removeEventListener'> | null;
}
export class RoomBootstrap {
  latest: unknown;
  #options: BootstrapOptions;
  #socket: SocketLike | null = null;
  #timer: ReturnType<typeof setTimeout> | undefined;
  #deadline: ReturnType<typeof setTimeout> | undefined;
  #closed = false;
  #failures = 0;
  #welcomed = false;
  #listener: RoomListener | null = null;
  #last: TransportStatus = { state: 'connecting', attempt: 0 };
  #network: BootstrapOptions['network'];
  constructor(options: BootstrapOptions) {
    this.#options = options;
    this.#network = options.network === undefined ? globalThis.window : options.network;
    this.#network?.addEventListener('online', this.#online);
    this.#network?.addEventListener('offline', this.#offline);
    this.#connect();
  }
  #status(status: TransportStatus): void {
    this.#last = status;
    this.#options.status(status);
    this.#listener?.status(status);
  }
  #online = (): void => {
    if (!this.#closed && !this.#socket) {
      clearTimeout(this.#timer);
      this.#connect();
    }
  };
  #offline = (): void => {
    if (!this.#closed) {
      this.#detach();
      this.#retry();
    }
  };
  #connect(): void {
    if (this.#closed) return;
    if (
      this.#options.network === undefined &&
      typeof navigator !== 'undefined' &&
      navigator.onLine === false
    ) {
      this.#status({ state: 'offline', attempt: this.#failures });
      return;
    }
    this.#status({
      state: this.#failures ? 'reconnecting' : 'connecting',
      attempt: this.#failures
    });
    let socket: SocketLike;
    try {
      socket = this.#options.socket?.() ?? new WebSocket(defaultSocketUrl());
    } catch {
      this.#retry();
      return;
    }
    this.#socket = socket;
    this.#welcomed = false;
    socket.onopen = () =>
      socket.send(JSON.stringify({ t: 'hello', v: 2, ...this.#options.identity }));
    socket.onmessage = (event) => {
      if (typeof event.data !== 'string') return;
      let message: unknown;
      try {
        message = JSON.parse(event.data);
      } catch {
        return;
      }
      const raw = object(message);
      if (!raw) return;
      if (raw['t'] === 'welcome') {
        if (this.#welcomed) return;
        if (raw['v'] !== 2) {
          this.#fail('protocol');
          return;
        }
        this.#welcomed = true;
        this.#failures = 0;
        this.#status({ state: 'open', attempt: 0 });
        const { code, claim } = this.#options;
        socket.send(JSON.stringify({ t: 'join', code, ...(claim ? { claim } : {}) }));
        this.#listener?.joined();
      } else if (raw['t'] === 'error' && typeof raw['code'] === 'string') {
        this.#fail(raw['code']);
      } else if (this.#listener) {
        if (raw['code'] === this.#options.code && snapshotKind(message))
          clearTimeout(this.#deadline);
        if (this.#welcomed) this.#listener.message(message);
      } else if (raw['t'] === 'reject' && typeof raw['code'] === 'string') {
        this.#fail(raw['code']);
      } else if (this.#welcomed && raw['code'] === this.#options.code && snapshotKind(message)) {
        clearTimeout(this.#deadline);
        this.latest = message; // exactly one raw snapshot, replaced whole
        this.#options.snapshot();
      }
    };
    socket.onerror = () => {}; // close follows
    socket.onclose = (event) => {
      if (this.#socket !== socket) return;
      this.#detach();
      const fatal = fatalClose(event.code, event.reason);
      if (fatal) this.#fail(fatal);
      else this.#retry();
    };
    this.#deadline = setTimeout(() => {
      this.#detach();
      this.#retry();
    }, 10_000);
  }
  #detach(): void {
    clearTimeout(this.#deadline);
    const socket = this.#socket;
    this.#socket = null;
    if (socket) {
      socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null;
      try {
        socket.close(1000);
      } catch {
        /* closing a connecting socket can throw */
      }
    }
  }
  #retry(): void {
    if (this.#closed) return;
    clearTimeout(this.#timer);
    const attempt = this.#failures++;
    this.#status({
      state: this.#failures >= 4 ? 'offline' : 'reconnecting',
      attempt: this.#failures
    });
    this.#timer = setTimeout(
      () => this.#connect(),
      backoffDelay(attempt, Math.random, DEFAULT_BACKOFF)
    );
  }
  #fail(code: string): void {
    this.close();
    this.#options.error(code);
  }
  /** The link's latest state. */
  get status(): TransportStatus {
    return this.#last;
  }
  /**
   * Hands the connection to a game's Session: it receives the newest
   * snapshot now (the return value, which the bootstrap stops holding) and
   * every later frame, status and rejoin. A refused request stays the
   * Session's to handle; an `error` frame still ends the entry.
   */
  attach(listener: RoomListener): unknown {
    const first = this.latest;
    this.latest = undefined;
    this.#listener = listener;
    return first;
  }
  /** Sends a request on the welcomed socket. Returns false, sending nothing, otherwise. */
  send(message: object): boolean {
    if (!this.#welcomed || this.#socket?.readyState !== 1) return false;
    this.#socket.send(JSON.stringify(message));
    return true;
  }
  /** The claim every later join carries (a relink of this device's own seat). */
  useClaim(claim: string | null): void {
    this.#options = { ...this.#options, claim };
  }
  /** Ends the entry, as an `error` frame with this code would. */
  fail(code: string): void {
    this.#fail(code);
  }
  resync(): void {
    if (this.#welcomed && this.#socket?.readyState === 1)
      this.#socket.send(JSON.stringify({ t: 'resync', code: this.#options.code }));
  }
  /** Close without leave. The server retains the seat for this same identity. */
  close(): void {
    if (this.#closed) return;
    this.#closed = true;
    clearTimeout(this.#timer);
    this.#detach();
    this.latest = undefined;
    this.#listener = null;
    this.#network?.removeEventListener('online', this.#online);
    this.#network?.removeEventListener('offline', this.#offline);
  }
}
