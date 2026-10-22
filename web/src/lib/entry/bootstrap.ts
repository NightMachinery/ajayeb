// Discovery owns an unknown snapshot, not a Duel view or an event stream.
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
  #network: BootstrapOptions['network'];
  constructor(options: BootstrapOptions) {
    this.#options = options;
    this.#network = options.network === undefined ? globalThis.window : options.network;
    this.#network?.addEventListener('online', this.#online);
    this.#network?.addEventListener('offline', this.#offline);
    this.#connect();
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
      this.#options.status({ state: 'offline', attempt: this.#failures });
      return;
    }
    this.#options.status({
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
        this.#options.status({ state: 'open', attempt: 0 });
        const { code, claim } = this.#options;
        socket.send(JSON.stringify({ t: 'join', code, ...(claim ? { claim } : {}) }));
      } else if (raw['t'] === 'error' && typeof raw['code'] === 'string') {
        this.#fail(raw['code']);
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
    this.#options.status({
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
    this.#network?.removeEventListener('online', this.#online);
    this.#network?.removeEventListener('offline', this.#offline);
  }
}
