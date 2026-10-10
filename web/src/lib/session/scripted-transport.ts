// A transport driven by hand, for unit tests: the test decides when the link
// opens or drops and which frames the server sends, and reads every frame
// the client sent. Nothing here is asynchronous.

import type { ClientMessage, ServerMessage } from '$lib/wire';
import {
  Listeners,
  type LinkState,
  type Transport,
  type TransportStatus,
  type Unsubscribe
} from './transport';

export class ScriptedTransport implements Transport {
  /** Every frame the client sent while the link was open, in order. */
  readonly sent: ClientMessage[] = [];
  /** Frames the client tried to send while the link was down. */
  readonly dropped: ClientMessage[] = [];
  closed = false;
  #status: TransportStatus = { state: 'connecting', attempt: 0 };
  #messages = new Listeners<ServerMessage>();
  #states = new Listeners<TransportStatus>();

  send(msg: ClientMessage): boolean {
    if (this.#status.state !== 'open') {
      this.dropped.push(msg);
      return false;
    }
    this.sent.push(msg);
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
    this.closed = true;
    this.setState('closed');
  }

  /** Moves the link to a state. */
  setState(state: LinkState, extra: Partial<TransportStatus> = {}): void {
    this.#status = { state, attempt: 0, ...extra };
    this.#states.emit(this.#status);
  }

  /** Delivers a frame from the server. */
  receive(msg: ServerMessage): void {
    this.#messages.emit(msg);
  }

  /** The frames sent so far of one type. */
  sentOf<T extends ClientMessage['t']>(t: T): Extract<ClientMessage, { t: T }>[] {
    return this.sent.filter((m): m is Extract<ClientMessage, { t: T }> => m.t === t);
  }

  /** The last frame sent. */
  last(): ClientMessage | undefined {
    return this.sent[this.sent.length - 1];
  }
}
