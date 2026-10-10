// The Session: the client's side of the protocol (docs/protocol.md). It
// says hello on every new connection, creates and joins rooms, sends acts
// and undos with the tab's seq counter and the current base, and keeps the
// latest authoritative state frame, replaced whole on every update.
//
// It decides nothing about the game. What may be played comes from the
// view's choices; whether this client may act now comes from the frame
// (you.canAct, view.toMove, you.seats); what things cost comes from the
// choices' cost breakdowns.
//
// A request (act or undo) is pending from the moment it is sent until a
// state frame acks its seq or a reject names it. Only one is pending at a
// time. After a reconnect the Session says hello, joins its room again and
// re-sends the pending request unchanged, which the protocol makes safe:
// the room either acks it (it was applied before the drop) or judges it
// afresh. A request the server dropped under its frame rate limit
// (`reject rate` with its seq: not handled, so its seq is not used up) is
// sent again unchanged after a short pause, a few times, before the
// Session gives up and shows the notice.

import { catalogMatches } from '$lib/catalog';
import type {
  Act,
  BotLevel,
  ClientMessage,
  Create,
  Mode,
  RoomInfo,
  SeatChoice,
  ServerInfo,
  ServerMessage,
  State,
  Undo,
  View,
  YouInfo
} from '$lib/wire';
import { ProtocolVersion } from '$lib/wire/types.gen';
import type { Identity } from './identity';
import { errorMessage, rejectMessage, type NoticeKey } from './messages';
import type { SavedRooms } from './rooms.svelte';
import {
  Listeners,
  type FatalReason,
  type Transport,
  type TransportStatus,
  type Unsubscribe
} from './transport';

/** The connection as the player sees it: 'open' once the server has welcomed us. */
export type ConnState = 'connecting' | 'open' | 'reconnecting' | 'offline';

export interface CreateOptions {
  mode: Mode;
  /** The bot's level; required for an 'ai' room and refused for any other. */
  bot?: BotLevel;
  /** The creator's seat in an online or ai room (default 0). */
  seat?: SeatChoice;
  /** Display names by seat; an empty entry means the hello name or a default. */
  names?: string[];
}

export interface PendingRequest {
  readonly frame: Act | Undo;
  /** When it was first sent, in ms (the Session's clock). */
  readonly sentAt: number;
}

/** The last thing the server refused, mapped to a message. */
export interface Notice {
  readonly kind: 'reject' | 'error';
  readonly code: string;
  readonly message: NoticeKey;
}

export interface SessionOptions {
  transport: Transport;
  identity: Identity;
  /** Where rooms seen are remembered for Home; none in tests and fixtures. */
  rooms?: SavedRooms | null;
  /** A pending request turns slow after this many ms. */
  slowMs?: number;
  /** The pause before re-sending a request dropped by the rate limit. */
  rateRetryMs?: number;
  /** How many times such a request is re-sent before the notice shows. */
  rateRetries?: number;
  now?: () => number;
}

type Intent =
  | { kind: 'join'; code: string }
  | {
      kind: 'create';
      frame: Create;
      resolve: (code: string) => void;
      reject: (err: SessionError) => void;
    };

/** A failed create, with the server's code and the message for it. */
export class SessionError extends Error {
  constructor(
    readonly code: string,
    readonly messageKey: NoticeKey
  ) {
    super(code);
    this.name = 'SessionError';
  }
}

/** A room code as the server writes it: upper case, no spaces. */
export function normalizeCode(code: string): string {
  return code.trim().toUpperCase();
}

export class Session {
  /** The connection, as the player sees it. */
  conn = $state<ConnState>('connecting');
  /** Set when the server refused this client for good (reload needed). */
  fatal = $state<FatalReason | null>(null);
  /** The server's build, from welcome. */
  server = $state.raw<ServerInfo | null>(null);
  /** The room this session is in or joining. */
  code = $state<string | null>(null);
  /** The latest state frame of that room, replaced whole. */
  frame = $state.raw<State | null>(null);
  /** The latest view, replaced whole (the frame's view). */
  view = $state.raw<View | null>(null);
  /** The request in flight, if any. */
  pending = $state.raw<PendingRequest | null>(null);
  /** The pending request has waited longer than slowMs. */
  slow = $state(false);
  /** The last refusal, until the next request or dismissNotice(). */
  notice = $state.raw<Notice | null>(null);
  /** The error that stopped the last join (room-not-found and so on). */
  roomError = $state<string | null>(null);
  /** The code of the rematch room, once the server has made one. */
  rematchCode = $state<string | null>(null);

  room: RoomInfo | null = $derived(this.frame?.room ?? null);
  you: YouInfo | null = $derived(this.frame?.you ?? null);
  /** The room version of the latest frame: the base of the next request. */
  version: number | null = $derived(this.frame?.version ?? null);
  /** The seat whose bot is searching, if any. */
  thinking: number | null = $derived(this.frame?.thinking ?? null);
  /** The server plays with other card data than this client's catalog. */
  catalogStale: boolean = $derived(this.view !== null && !catalogMatches(this.view.catalogHash));

  /** Whether this client may send an act now: all of it read from the frame. */
  canAct: boolean = $derived.by(() => {
    const f = this.frame;
    if (!f || this.conn !== 'open' || this.pending || this.fatal) return false;
    const toMove = f.view.toMove;
    return (
      f.you.canAct &&
      toMove !== undefined &&
      f.you.seats.includes(toMove) &&
      f.view.choices.length > 0
    );
  });

  /** Whether an undo would be sent now (the server's canUndo says if it is accepted). */
  canUndo: boolean = $derived(
    !!this.frame?.canUndo && this.conn === 'open' && !this.pending && !this.fatal
  );

  #transport: Transport;
  #identity: Identity;
  #rooms: SavedRooms | null;
  #slowMs: number;
  #now: () => number;
  #seq = 0;
  #welcomed = false;
  #intent: Intent | null = null;
  #remembered: string | null = null;
  #slowTimer: ReturnType<typeof setTimeout> | undefined;
  #rateRetryMs: number;
  #rateRetries: number;
  #rateTries = 0;
  #rateTimer: ReturnType<typeof setTimeout> | undefined;
  #frames = new Listeners<State>();
  #offs: Unsubscribe[] = [];
  #closed = false;

  constructor(opts: SessionOptions) {
    this.#transport = opts.transport;
    this.#identity = opts.identity;
    this.#rooms = opts.rooms ?? null;
    this.#slowMs = opts.slowMs ?? 300;
    // The server refills its frame bucket at 10 frames a second.
    this.#rateRetryMs = opts.rateRetryMs ?? 250;
    this.#rateRetries = opts.rateRetries ?? 3;
    this.#now = opts.now ?? (() => Date.now());
    this.#offs.push(
      this.#transport.onMessage((msg) => this.#onMessage(msg)),
      this.#transport.onState((status) => this.#onStatus(status))
    );
  }

  /** This tab's client id (not secret; the token never leaves the Session). */
  get clientId(): string {
    return this.#identity.clientId;
  }

  /** The seq of the last request sent. */
  get seq(): number {
    return this.#seq;
  }

  /** Calls fn with every accepted state frame (and at once with the current one). */
  subscribe(fn: (frame: State) => void): Unsubscribe {
    const off = this.#frames.add(fn);
    if (this.frame) fn(this.frame);
    return off;
  }

  /** Creates a room and joins it; resolves with its code. */
  create(opts: CreateOptions): Promise<string> {
    const frame: Create = {
      t: 'create',
      mode: opts.mode,
      seat: opts.seat ?? 0,
      names: opts.names ?? []
    };
    if (opts.bot !== undefined) frame.bot = opts.bot;
    if (this.#intent?.kind === 'create') {
      this.#intent.reject(new SessionError('superseded', 'session.error.unknown'));
    }
    this.#leaveRoom();
    return new Promise<string>((resolve, reject) => {
      this.#intent = { kind: 'create', frame, resolve, reject };
      if (this.#welcomed) this.#send(frame);
    });
  }

  /** Joins a room by code (a fresh state follows). */
  join(code: string): void {
    const c = normalizeCode(code);
    if (this.#intent?.kind === 'create') {
      this.#intent.reject(new SessionError('superseded', 'session.error.unknown'));
    }
    if (c !== this.code) this.#leaveRoom();
    this.code = c;
    this.roomError = null;
    this.#intent = { kind: 'join', code: c };
    if (this.#welcomed) this.#send({ t: 'join', code: c });
  }

  /**
   * Sends an act for one of the view's choices. The key must be in the
   * latest view's choices; the Session never makes one up. Returns whether
   * it was sent.
   */
  act(key: string): boolean {
    const f = this.frame;
    if (!this.canAct || !f || !this.code) return false;
    if (!f.view.choices.some((c) => c.key === key)) return false;
    const frame: Act = { t: 'act', code: this.code, seq: this.#seq + 1, base: f.version, key };
    // A hot-seat client names the seat it acts for, so a tap left over from
    // the previous player's turn cannot act for the next one.
    const toMove = f.view.toMove;
    if (f.room.mode === 'hotseat' && (toMove === 0 || toMove === 1)) frame.seat = toMove;
    this.#request(frame);
    return true;
  }

  /** Asks to take back the last human decision. Returns whether it was sent. */
  undo(): boolean {
    const f = this.frame;
    if (!this.canUndo || !f || !this.code) return false;
    this.#request({ t: 'undo', code: this.code, seq: this.#seq + 1, base: f.version });
    return true;
  }

  /** Sends the pending request again, unchanged (same seq and base). */
  retry(): boolean {
    if (!this.pending) return false;
    return this.#send(this.pending.frame);
  }

  /** Asks for a fresh state of the room. */
  resync(): void {
    if (this.code && this.#welcomed) this.#send({ t: 'resync', code: this.code });
  }

  /** Asks for a rematch once the game is over; the code arrives in rematchCode. */
  rematch(): void {
    if (this.code && this.#welcomed) this.#send({ t: 'rematch', code: this.code });
  }

  /**
   * Leaves a room. With a code other than the current room, only tells the
   * server (a page that is going away may call it after the next one joined).
   */
  leave(code: string | null = this.code): void {
    if (!code) return;
    const c = normalizeCode(code);
    if (this.#welcomed) this.#send({ t: 'leave', code: c });
    if (c === this.code) {
      this.#leaveRoom();
      this.#intent = null;
    }
  }

  dismissNotice(): void {
    this.notice = null;
  }

  /** Closes the connection for good. */
  close(): void {
    if (this.#closed) return;
    this.#closed = true;
    clearTimeout(this.#slowTimer);
    clearTimeout(this.#rateTimer);
    for (const off of this.#offs) off();
    this.#frames.clear();
    this.#transport.close();
  }

  // Internals ----------------------------------------------------------------

  #send(msg: ClientMessage): boolean {
    return this.#transport.send(msg);
  }

  #request(frame: Act | Undo): void {
    this.#seq = frame.seq;
    const pending: PendingRequest = { frame, sentAt: this.#now() };
    this.pending = pending;
    this.slow = false;
    this.notice = null;
    this.#rateTries = 0;
    clearTimeout(this.#rateTimer);
    clearTimeout(this.#slowTimer);
    this.#slowTimer = setTimeout(() => {
      if (this.pending === pending) this.slow = true;
    }, this.#slowMs);
    // When the link is down this sends nothing; the request goes out again
    // after the reconnect (#resume).
    this.#send(frame);
  }

  #settle(): void {
    clearTimeout(this.#slowTimer);
    this.#slowTimer = undefined;
    clearTimeout(this.#rateTimer);
    this.#rateTimer = undefined;
    this.pending = null;
    this.slow = false;
  }

  #leaveRoom(): void {
    this.code = null;
    this.frame = null;
    this.view = null;
    this.rematchCode = null;
    this.roomError = null;
    this.notice = null;
    this.#settle();
  }

  #onStatus(status: TransportStatus): void {
    switch (status.state) {
      case 'connecting':
        this.conn = 'connecting';
        break;
      case 'open':
        // Say hello on every new socket; 'open' for the player comes with welcome.
        this.#welcomed = false;
        this.#send({
          t: 'hello',
          v: ProtocolVersion,
          token: this.#identity.token,
          name: this.#identity.name,
          clientId: this.#identity.clientId
        });
        break;
      case 'reconnecting':
      case 'offline':
        this.#welcomed = false;
        this.conn = status.state;
        break;
      case 'closed':
        this.#welcomed = false;
        this.conn = 'offline';
        if (status.fatal) this.#fail(status.fatal);
        break;
    }
  }

  #fail(reason: FatalReason): void {
    this.fatal = reason;
    this.conn = 'offline';
    this.notice = { kind: 'error', code: reason, message: errorMessage(reason) };
    if (this.#intent?.kind === 'create') {
      this.#intent.reject(new SessionError(reason, errorMessage(reason)));
      this.#intent = null;
    }
  }

  #onMessage(msg: ServerMessage): void {
    switch (msg.t) {
      case 'welcome':
        if (msg.v !== ProtocolVersion) {
          this.#fail('protocol');
          this.#transport.close();
          return;
        }
        this.server = msg.server;
        this.#welcomed = true;
        this.conn = 'open';
        this.#resume();
        break;
      case 'created':
        if (this.#intent?.kind === 'create') {
          const { resolve } = this.#intent;
          this.code = msg.code;
          this.#intent = { kind: 'join', code: msg.code };
          resolve(msg.code);
        } else {
          // A rematch room: the client joins it by its code.
          this.rematchCode = msg.code;
        }
        break;
      case 'state':
        this.#onState(msg);
        break;
      case 'reject':
        if (this.pending && msg.seq === this.pending.frame.seq) {
          if (msg.code === 'rate' && this.#rateTries < this.#rateRetries) {
            this.#retryAfterRate(this.pending);
            return;
          }
          this.#settle();
        }
        if (msg.seq === 0 && this.#intent?.kind === 'create') {
          this.#intent.reject(new SessionError(msg.code, rejectMessage(msg.code)));
          this.#intent = null;
        } else if (msg.seq === 0 && this.#intent?.kind === 'join' && !this.frame) {
          this.roomError = msg.code; // a join over the address's limit
        }
        this.notice = { kind: 'reject', code: msg.code, message: rejectMessage(msg.code) };
        break;
      case 'error':
        if (msg.code === 'protocol' || msg.code === 'bad-hello') {
          this.#fail(msg.code);
          return;
        }
        this.notice = { kind: 'error', code: msg.code, message: errorMessage(msg.code) };
        if (this.#intent?.kind === 'create') {
          this.#intent.reject(new SessionError(msg.code, errorMessage(msg.code)));
          this.#intent = null;
        } else if (this.#intent?.kind === 'join' && !this.frame) {
          // The join failed: there is no room to show.
          this.roomError = msg.code;
          if (msg.code === 'room-not-found') this.#rooms?.forget(this.#intent.code);
        }
        break;
    }
  }

  // The request was dropped unhandled: send it again unchanged after a pause.
  #retryAfterRate(pending: PendingRequest): void {
    this.#rateTries++;
    clearTimeout(this.#rateTimer);
    this.#rateTimer = setTimeout(() => {
      this.#rateTimer = undefined;
      if (this.pending === pending && this.#welcomed) this.#send(pending.frame);
    }, this.#rateRetryMs * this.#rateTries);
  }

  #onState(frame: State): void {
    // The outbox may still deliver a state of a room this client has left.
    if (frame.code !== this.code) return;
    this.frame = frame;
    this.view = frame.view;
    this.roomError = null;
    // Home's Continue list offers live games only: a finished room is
    // dropped from it, whether it ended while on screen or before the join.
    if (frame.room.status === 'finished') {
      this.#remembered = frame.code;
      this.#rooms?.forget(frame.code);
    } else if (this.#remembered !== frame.code) {
      this.#remembered = frame.code;
      this.#rooms?.remember(frame.code, frame.room.mode);
    }
    const ack = frame.ack;
    if (
      this.pending &&
      ack &&
      ack.clientId === this.#identity.clientId &&
      ack.seq === this.pending.frame.seq
    ) {
      this.#settle();
    }
    this.#frames.emit(frame);
  }

  // After welcome: carry on with what the session was doing.
  #resume(): void {
    const intent = this.#intent;
    if (intent?.kind === 'create') {
      this.#send(intent.frame);
      return;
    }
    if (intent?.kind === 'join') {
      this.#send({ t: 'join', code: intent.code });
      // The server answers in order: the fresh state of the join, then the
      // ack or the refusal of the re-sent request.
      if (this.pending) this.#send(this.pending.frame);
    }
  }
}
