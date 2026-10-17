// A transport with no server: it answers the Session from recorded frames
// (src/lib/wire/fixtures.gen.ts), so dev pages and tests can show any
// recorded game state.
//
// It plays a script of state frames:
//   - hello gets a welcome;
//   - join and resync get the current frame of the script;
//   - act and undo move to the next frame and send it with an ack for the
//     request, or, at the end of the script, get a reject 'illegal';
//   - create gets a created frame with the first frame's code, then that frame.
// step() moves to the next frame without a request, the way another
// player's move or a bot's would arrive.
//
// Every answer arrives on a microtask, never inside send(), the way a real
// socket's answer never arrives inside send() either.

import type { ClientMessage, ServerMessage, State, View } from '$lib/wire';
import { ProtocolVersion } from '$lib/wire/types.gen';
import { defaultHints } from '$lib/hints/families';
import {
  Listeners,
  type Transport,
  type TransportStatus,
  type LinkState,
  type Unsubscribe
} from './transport';

export interface FixtureTransportOptions {
  /** The state frames, in order. */
  frames: readonly State[];
  /** Show a dropped link after delivering the recorded snapshot. */
  linkState?: LinkState;
  /** The server info in the welcome. */
  server?: { version: string; rules: number };
}

export class FixtureTransport implements Transport {
  #frames: readonly State[];
  #index = 0;
  #linkState: LinkState;
  #server: { version: string; rules: number };
  #status: TransportStatus = { state: 'connecting', attempt: 0 };
  #messages = new Listeners<ServerMessage>();
  #states = new Listeners<TransportStatus>();
  /** Every frame the client sent, for tests. */
  readonly sent: ClientMessage[] = [];

  constructor(opts: FixtureTransportOptions) {
    if (opts.frames.length === 0) throw new Error('FixtureTransport needs at least one frame');
    this.#frames = opts.frames;
    this.#linkState = opts.linkState ?? 'open';
    this.#server = opts.server ?? { version: 'fixture', rules: 1 };
    queueMicrotask(() => {
      if (this.#status.state === 'connecting') this.#setStatus({ state: 'open', attempt: 0 });
    });
  }

  /** The index of the frame the script is at. */
  get index(): number {
    return this.#index;
  }

  get length(): number {
    return this.#frames.length;
  }

  send(msg: ClientMessage): boolean {
    if (this.#status.state !== 'open') return false;
    this.sent.push(msg);
    switch (msg.t) {
      case 'hello':
        this.#later({ t: 'welcome', v: ProtocolVersion, server: this.#server });
        break;
      case 'create':
        this.#later({ t: 'created', code: this.#current().code });
        this.#later(this.#current());
        break;
      case 'join':
      case 'resync':
        this.#later(this.#current());
        break;
      case 'act':
      case 'undo':
        if (this.#index + 1 < this.#frames.length) {
          this.#index++;
          // A recorded ack names the recorder's client; this one names the
          // request it answers, as a real server's would.
          const clientId = this.#clientId();
          this.#later({ ...this.#current(), ack: { clientId, seq: msg.seq } });
        } else {
          this.#later({ t: 'reject', seq: msg.seq, code: 'illegal', msg: 'end of the fixture' });
        }
        break;
      case 'leave':
      case 'rematch':
        break;
    }
    return true;
  }

  /** Sends the next frame unasked. Returns false at the end of the script. */
  step(): boolean {
    if (this.#index + 1 >= this.#frames.length) return false;
    this.#index++;
    const frame: State = { ...this.#current() };
    delete frame.ack; // only the direct answer to a request carries one
    this.#later(frame);
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
    this.#setStatus({ state: 'closed', attempt: 0 });
    this.#messages.clear();
    this.#states.clear();
  }

  #current(): State {
    return this.#frames[this.#index]!;
  }

  #clientId(): string {
    for (let i = this.sent.length - 1; i >= 0; i--) {
      const m = this.sent[i]!;
      if (m.t === 'hello') return m.clientId;
    }
    return '';
  }

  #later(msg: ServerMessage): void {
    queueMicrotask(() => {
      if (this.#status.state === 'open') {
        this.#messages.emit(msg);
        if (msg.t === 'state' && this.#linkState !== 'open') {
          this.#setStatus({ state: this.#linkState, attempt: 1 });
        }
      }
    });
  }

  #setStatus(status: TransportStatus): void {
    this.#status = status;
    this.#states.emit(status);
  }
}

export interface ViewFrameOptions {
  code?: string;
  /** Seat names, by seat. */
  names?: readonly [string, string];
}

/**
 * Wraps a recorded View (protocolFixtures) in the state frame a real server
 * would send its viewer: an online room in which the connection holds the
 * view's seat, or watches when the view is a spectator's.
 */
export function stateFrameFromView(view: View, opts: ViewFrameOptions = {}): State {
  const names = opts.names ?? ['', ''];
  const seats = view.viewer === undefined ? [] : [view.viewer];
  return {
    t: 'state',
    code: opts.code ?? 'FIXTUR',
    version: view.ply,
    room: {
      mode: 'online',
      status: view.phase === 'gameOver' ? 'finished' : 'playing',
      seats: [0, 1].map((seat) => ({
        seat,
        name: names[seat] ?? '',
        kind: 'human' as const,
        connected: true,
        you: seats.includes(seat),
        nameNo: 0
      })),
      spectators: view.viewer === undefined ? 1 : 0,
      hints: defaultHints('online'),
      allowUndo: false
    },
    // Seat 0 created the room, so it is the game's admin.
    you: { seats, canAct: view.choices.length > 0, admin: seats.includes(0) },
    canUndo: false,
    events: [],
    roomLog: [],
    view
  };
}
