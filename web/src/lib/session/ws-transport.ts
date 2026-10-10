// The real transport: one WebSocket to /ws on the page's own origin, with
// reconnects (docs/protocol.md, "Close codes" and "Reconnecting").
//
// After any close it did not ask for, it opens a new socket after a backoff
// delay with jitter (backoff.ts), except after a 1008 close whose reason is
// "protocol" (the server runs another protocol version: reload the page) or
// "bad-hello" (this client is wrong), which another try cannot fix. The
// Session says hello again on every 'open'.

import type { ClientMessage, ServerMessage } from '$lib/wire';
import { backoffDelay, DEFAULT_BACKOFF, type BackoffOptions } from './backoff';
import {
  Listeners,
  type FatalReason,
  type Transport,
  type TransportStatus,
  type Unsubscribe
} from './transport';

/** The part of the browser's WebSocket this transport uses. */
export interface SocketLike {
  readonly readyState: number;
  onopen: ((ev: Event) => void) | null;
  onclose: ((ev: CloseEvent) => void) | null;
  onmessage: ((ev: MessageEvent) => void) | null;
  onerror: ((ev: Event) => void) | null;
  send(data: string): void;
  close(code?: number, reason?: string): void;
}

/** The part of window this transport listens to: online and offline. */
export interface NetworkLike {
  readonly onLine: boolean;
  addEventListener(type: 'online' | 'offline', fn: () => void): void;
  removeEventListener(type: 'online' | 'offline', fn: () => void): void;
}

export interface WsTransportOptions {
  /** The socket URL. Default: /ws on the page's origin (ws: or wss:). */
  url?: string;
  createSocket?: (url: string) => SocketLike;
  /** Online and offline events; default window and navigator when they exist. */
  network?: NetworkLike | null;
  random?: () => number;
  backoff?: BackoffOptions;
  /** Failed tries in a row after which the state reads 'offline'. */
  offlineAfter?: number;
  /** A socket that has not opened after this many ms is dropped and retried. */
  connectTimeoutMs?: number;
}

const OPEN = 1;
const POLICY_VIOLATION = 1008;
const NORMAL_CLOSURE = 1000;

/** /ws on the page's own origin. */
export function defaultSocketUrl(loc: Pick<Location, 'protocol' | 'host'> = location): string {
  return `${loc.protocol === 'https:' ? 'wss:' : 'ws:'}//${loc.host}/ws`;
}

/** The fatal reason of a close, if it is one that no retry can fix. */
export function fatalClose(code: number, reason: string): FatalReason | undefined {
  if (code !== POLICY_VIOLATION) return undefined;
  return reason === 'protocol' || reason === 'bad-hello' ? reason : undefined;
}

function defaultNetwork(): NetworkLike | null {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') return null;
  return {
    get onLine() {
      return navigator.onLine;
    },
    addEventListener: (type, fn) => window.addEventListener(type, fn),
    removeEventListener: (type, fn) => window.removeEventListener(type, fn)
  };
}

export class WsTransport implements Transport {
  #url: string;
  #createSocket: (url: string) => SocketLike;
  #network: NetworkLike | null;
  #random: () => number;
  #backoff: BackoffOptions;
  #offlineAfter: number;
  #connectTimeoutMs: number;

  #socket: SocketLike | null = null;
  #retryTimer: ReturnType<typeof setTimeout> | undefined;
  #connectTimer: ReturnType<typeof setTimeout> | undefined;
  #closed = false;
  #status: TransportStatus = { state: 'connecting', attempt: 0 };

  #messages = new Listeners<ServerMessage>();
  #states = new Listeners<TransportStatus>();

  constructor(opts: WsTransportOptions = {}) {
    this.#url = opts.url ?? defaultSocketUrl();
    this.#createSocket = opts.createSocket ?? ((url) => new WebSocket(url));
    this.#network = opts.network === undefined ? defaultNetwork() : opts.network;
    this.#random = opts.random ?? Math.random;
    this.#backoff = opts.backoff ?? DEFAULT_BACKOFF;
    this.#offlineAfter = opts.offlineAfter ?? 4;
    this.#connectTimeoutMs = opts.connectTimeoutMs ?? 10_000;
    this.#network?.addEventListener('online', this.#onOnline);
    this.#network?.addEventListener('offline', this.#onOffline);
    this.#connect();
  }

  get status(): TransportStatus {
    return this.#status;
  }

  send(msg: ClientMessage): boolean {
    const socket = this.#socket;
    if (this.#closed || !socket || socket.readyState !== OPEN) return false;
    socket.send(JSON.stringify(msg));
    return true;
  }

  onMessage(fn: (msg: ServerMessage) => void): Unsubscribe {
    return this.#messages.add(fn);
  }

  onState(fn: (status: TransportStatus) => void): Unsubscribe {
    const off = this.#states.add(fn);
    fn(this.#status);
    return off;
  }

  close(): void {
    if (this.#closed) return;
    this.#closed = true;
    this.#clearTimers();
    this.#network?.removeEventListener('online', this.#onOnline);
    this.#network?.removeEventListener('offline', this.#onOffline);
    const socket = this.#socket;
    this.#socket = null;
    if (socket) {
      this.#detach(socket);
      try {
        socket.close(NORMAL_CLOSURE);
      } catch {
        // Closing a socket that never opened can throw in some browsers.
      }
    }
    this.#setStatus({ state: 'closed', attempt: this.#status.attempt });
  }

  #setStatus(status: TransportStatus): void {
    const s = this.#status;
    if (s.state === status.state && s.attempt === status.attempt && s.fatal === status.fatal)
      return;
    this.#status = status;
    this.#states.emit(status);
  }

  #clearTimers(): void {
    clearTimeout(this.#retryTimer);
    clearTimeout(this.#connectTimer);
    this.#retryTimer = undefined;
    this.#connectTimer = undefined;
  }

  #detach(socket: SocketLike): void {
    socket.onopen = null;
    socket.onclose = null;
    socket.onmessage = null;
    socket.onerror = null;
  }

  #connect = (): void => {
    this.#retryTimer = undefined;
    if (this.#closed) return;
    let socket: SocketLike;
    try {
      socket = this.#createSocket(this.#url);
    } catch {
      // A malformed URL or a blocked scheme: treat it as a failed try.
      this.#dropped(1006, '');
      return;
    }
    this.#socket = socket;
    socket.onopen = () => {
      clearTimeout(this.#connectTimer);
      this.#connectTimer = undefined;
      this.#setStatus({ state: 'open', attempt: 0 });
    };
    socket.onmessage = (ev) => {
      if (typeof ev.data !== 'string') return; // the server sends text frames only
      let msg: ServerMessage;
      try {
        msg = JSON.parse(ev.data) as ServerMessage;
      } catch {
        return; // not JSON: nothing a client can do with it
      }
      if (msg && typeof msg === 'object' && typeof msg.t === 'string') this.#messages.emit(msg);
    };
    socket.onerror = () => {
      // A close event always follows; the retry is scheduled there.
    };
    socket.onclose = (ev) => {
      this.#detach(socket);
      if (this.#socket === socket) this.#socket = null;
      this.#dropped(ev.code, ev.reason);
    };
    clearTimeout(this.#connectTimer);
    this.#connectTimer = setTimeout(() => {
      this.#connectTimer = undefined;
      if (this.#socket !== socket || socket.readyState === OPEN) return;
      this.#detach(socket);
      this.#socket = null;
      try {
        socket.close();
      } catch {
        // Ignore: the socket is abandoned either way.
      }
      this.#dropped(1006, '');
    }, this.#connectTimeoutMs);
  };

  #dropped(code: number, reason: string): void {
    clearTimeout(this.#connectTimer);
    this.#connectTimer = undefined;
    if (this.#closed) return;
    const fatal = fatalClose(code, reason);
    if (fatal) {
      this.#closed = true;
      this.#clearTimers();
      this.#network?.removeEventListener('online', this.#onOnline);
      this.#network?.removeEventListener('offline', this.#onOffline);
      this.#setStatus({ state: 'closed', fatal, attempt: this.#status.attempt });
      return;
    }
    this.#scheduleRetry();
  }

  #scheduleRetry(): void {
    const attempt = this.#status.state === 'open' ? 0 : this.#status.attempt;
    const offline = attempt >= this.#offlineAfter || this.#network?.onLine === false;
    this.#setStatus({ state: offline ? 'offline' : 'reconnecting', attempt: attempt + 1 });
    clearTimeout(this.#retryTimer);
    this.#retryTimer = setTimeout(
      this.#connect,
      backoffDelay(attempt, this.#random, this.#backoff)
    );
  }

  // Back online: skip the rest of the wait and try now.
  #onOnline = (): void => {
    if (this.#closed || this.#retryTimer === undefined) return;
    clearTimeout(this.#retryTimer);
    this.#connect();
  };

  #onOffline = (): void => {
    if (this.#closed || this.#status.state === 'open' || this.#status.state === 'connecting')
      return;
    this.#setStatus({ state: 'offline', attempt: this.#status.attempt });
  };
}
