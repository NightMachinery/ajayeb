// The Transport carries frames between the Session and a server. Two
// implementations exist: WsTransport (the real WebSocket to /ws, with
// reconnects) and FixtureTransport (recorded frames, for dev pages and
// tests). The Session does not know which one it has.

import type { ClientMessage, ServerMessage } from '$lib/wire';

/**
 * The link's state:
 *   - connecting: the first connection is being opened;
 *   - open: frames flow; the Session must say hello now;
 *   - reconnecting: the link dropped and a new socket is on its way;
 *   - offline: the browser is offline, or several tries in a row failed;
 *     the transport keeps trying (at the capped delay) unless it is fatal;
 *   - closed: close() was called, or the server ended the connection for a
 *     reason another try cannot fix (see TransportStatus.fatal).
 */
export type LinkState = 'connecting' | 'open' | 'reconnecting' | 'offline' | 'closed';

/** Why the server ended the connection for good: protocol.md, close codes. */
export type FatalReason = 'protocol' | 'bad-hello';

export interface TransportStatus {
  state: LinkState;
  /** Set with state 'closed' when the server refused this client for good. */
  fatal?: FatalReason;
  /** Consecutive failed tries since the link was last open. */
  attempt: number;
}

export type Unsubscribe = () => void;

export interface Transport {
  /** Sends a frame now. Returns false, sending nothing, unless the link is open. */
  send(msg: ClientMessage): boolean;
  /** Calls fn with every frame from the server. */
  onMessage(fn: (msg: ServerMessage) => void): Unsubscribe;
  /** Calls fn at once with the current status, then on every change. */
  onState(fn: (status: TransportStatus) => void): Unsubscribe;
  /** Closes the link for good. */
  close(): void;
  /**
   * Drops the current connection and opens a new one at once, so the
   * Session says hello again (with a new display name). Optional: a
   * transport without it keeps its connection.
   */
  reconnect?(): void;
}

/**
 * A minimal set of listeners, shared by the transports. Listeners run in the
 * order they subscribed; one that throws does not stop the others.
 */
export class Listeners<T> {
  #fns = new Set<(value: T) => void>();

  add(fn: (value: T) => void): Unsubscribe {
    this.#fns.add(fn);
    return () => {
      this.#fns.delete(fn);
    };
  }

  emit(value: T): void {
    for (const fn of [...this.#fns]) {
      try {
        fn(value);
      } catch (err) {
        // A broken listener must not take the connection down with it.
        console.error(err);
      }
    }
  }

  clear(): void {
    this.#fns.clear();
  }
}
