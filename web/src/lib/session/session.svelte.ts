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
//
// In a bot room an undo can cross the bot's answer on the wire: it names
// the version on screen as its base, and the bot's move reaches the room
// first, so the room refuses it as stale. Only the bot moves there while
// the player's undo is pending, so the Session sends the undo once more
// with the fresh version (same seq, which a refusal does not use up), and
// the player's press is not lost.
//
// The display name travels in hello only. setName() changes it for the
// rest of the page's life; when the socket has already said hello under
// another name it asks the transport for a fresh connection.
//
// A rematch request remembers that this player asked (rematchAsked), so
// the page follows the rematch code to its room. A code that arrives
// without asking (the other player asked) waits for the player to choose.
//
// The game's history (docs/protocol.md, the state frame): the first frame
// of a join, a resync's answer and the frame after an undo carry every
// applied action so far, and replace `history` whole. Every other frame's
// events extend it by one entry, so `log` is the whole game even after a
// reload. A frame whose ply skips ahead of the history (frames merged in
// the outbox, or lost) is appended as one entry and repaired by a resync.
// The history is data for the log and the move summary only: nothing here
// animates it, and a frame that brings one has no live events to animate.
//
// The hint settings (docs/dev/decisions.md D23): the room's settings, the
// admin flag and the room log come with every frame. setHints() sends the
// game's admin's new settings as a request like an act (same seq counter,
// one pending at a time, re-sent after a reconnect). A create without
// settings takes them from the defaultHints option (the device's local
// default), so every way of creating a room starts from it.
//
// Device links (docs/protocol.md, "Device links"; ./claim.ts): join() takes
// an optional claim, sent with every join of that room (also after a
// reconnect), with which this connection plays the seat the link was made
// for. requestLink() and resetLink() ask for a seat's link (the server
// answers this connection alone with a linked frame). A device on a claim
// that resets its own link moves to the new claim; one whose claim stops
// working (error bad-claim) shows the room error and forgets the claim. A
// rematch announced to a device on a claim carries its claim in the new
// room (rematchClaim).
//
// The forfeit: resign() sends an act whose key is resign:<seat>, a key the
// view's choices never hold. It forfeits the seat this connection plays
// (the seat to move in a hot-seat room). Refused as stale because the other
// side moved first, it goes out once more with the fresh version.

import { catalogMatches } from '$lib/catalog';
import type {
  Act,
  BotLevel,
  ClientMessage,
  Create,
  GameEvent,
  HintSettings,
  Hints as HintsFrame,
  HistoryEntry,
  Join,
  Linked,
  Mode,
  RoomInfo,
  RoomLogEntry,
  SeatChoice,
  ServerInfo,
  ServerMessage,
  State,
  Undo,
  View,
  YouInfo
} from '$lib/wire';
import { ProtocolVersion } from '$lib/wire/types.gen';
import { cleanName, type Identity } from './identity';
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
  /** The game's hint settings; absent means the defaultHints option's, else the server's defaults. */
  hints?: HintSettings;
  /**
   * Whether the game allows undo (a hot-seat or ai room; fixed for the
   * game's life). Absent means the defaultAllowUndo option's, else the
   * server's default, on. Never sent for an online room, which has no undo.
   */
  allowUndo?: boolean;
  /** Asks for a tutorial room (an 'ai' room against the Easy bot only). */
  tutorial?: boolean;
}

export interface PendingRequest {
  readonly frame: Act | Undo | HintsFrame;
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
  /** The hint settings a create without any starts from (the local default). */
  defaultHints?: (mode: Mode) => HintSettings | null;
  /** The allowUndo a hot-seat or ai create without one sends (the local default). */
  defaultAllowUndo?: (mode: Exclude<Mode, 'online'>) => boolean | null;
}

type Intent =
  | { kind: 'join'; code: string; claim: string | null }
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

/** How long a link request waits for its answer, in ms. */
const LINK_TIMEOUT_MS = 8000;

interface LinkWaiter {
  seat: 0 | 1;
  reset: boolean;
  resolve: (claim: string) => void;
  reject: (err: SessionError) => void;
  timer: ReturnType<typeof setTimeout>;
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
  /** The device link's claim this session plays the room with, if any. */
  claim = $state<string | null>(null);
  /** The code of the rematch room, once the server has made one. */
  rematchCode = $state<string | null>(null);
  /** This device's claim in the rematch room, when it plays on a claim. */
  rematchClaim = $state<string | null>(null);
  /** This player asked for the rematch (or chose to follow it). */
  rematchAsked = $state(false);
  /** The applied actions of the game so far, oldest first. */
  history = $state.raw<readonly HistoryEntry[]>([]);
  /** Every event of the game so far, in order: the history's, flattened. */
  log: readonly GameEvent[] = $derived(this.history.flatMap((e) => e.events));
  /** The room log (changes of the hint settings), oldest first. */
  roomLog: readonly RoomLogEntry[] = $derived(this.frame?.roomLog ?? []);
  /** The game's hint settings, the same for both players. */
  hints: HintSettings | null = $derived(this.frame?.room.hints ?? null);
  /** This user is the game's admin, who may change its hint settings. */
  admin: boolean = $derived(!!this.frame?.you.admin);
  /** The game allows undo at all (fixed when it was created; never online). */
  allowUndo: boolean = $derived(!!this.frame?.room.allowUndo);

  /** Whether setHints would be sent now: the admin, while the game runs. */
  canSetHints: boolean = $derived.by(() => {
    const f = this.frame;
    if (!f || this.conn !== 'open' || this.pending || this.fatal) return false;
    return f.you.admin && f.room.status !== 'finished' && !f.view.result;
  });

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

  /**
   * The seat this client would forfeit now: its own seat in an online or ai
   * room, the seat to move in a hot-seat room; null when it cannot (a
   * spectator, the lobby, a finished game, a request in flight, no link).
   */
  resignSeat: 0 | 1 | null = $derived.by(() => {
    const f = this.frame;
    if (!f || this.conn !== 'open' || this.pending || this.fatal) return null;
    if (f.room.status !== 'playing' || f.view.result) return null;
    const seat = f.room.mode === 'hotseat' ? f.view.toMove : f.you.seats[0];
    if (f.room.mode !== 'hotseat' && f.you.seats.length !== 1) return null;
    return seat === 0 || seat === 1 ? seat : null;
  });

  #transport: Transport;
  #identity: Identity;
  #rooms: SavedRooms | null;
  #slowMs: number;
  #now: () => number;
  #defaultHints: ((mode: Mode) => HintSettings | null) | null;
  #defaultAllowUndo: ((mode: Exclude<Mode, 'online'>) => boolean | null) | null;
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
  #links: LinkWaiter[] = [];
  #offs: Unsubscribe[] = [];
  #closed = false;
  /** The name the current socket said hello with; null before hello. */
  #helloName: string | null = null;
  /** A bot-room undo refused as stale, waiting for the fresh state. */
  #staleUndo: PendingRequest | null = null;
  /** The pending undo or forfeit was sent again once already. */
  #undoResent = false;
  /** The ply up to which `history` is complete; null before the first frame. */
  #covered: number | null = null;
  /** The version a resync was last asked at, to repair a gap only once. */
  #gapAsked: number | null = null;

  constructor(opts: SessionOptions) {
    this.#transport = opts.transport;
    // A copy: setName changes this page's name, never the caller's object.
    this.#identity = { ...opts.identity };
    this.#rooms = opts.rooms ?? null;
    this.#slowMs = opts.slowMs ?? 300;
    // The server refills its frame bucket at 10 frames a second.
    this.#rateRetryMs = opts.rateRetryMs ?? 250;
    this.#rateRetries = opts.rateRetries ?? 3;
    this.#now = opts.now ?? (() => Date.now());
    this.#defaultHints = opts.defaultHints ?? null;
    this.#defaultAllowUndo = opts.defaultAllowUndo ?? null;
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

  /** The display name this page says hello with. */
  get name(): string {
    return this.#identity.name;
  }

  /**
   * Changes the display name the server gives a seat this user takes. The
   * name goes only in hello, so a socket that already said hello under
   * another name is replaced by a fresh one (the room is joined again).
   */
  setName(name: string): void {
    const clean = cleanName(name);
    if (clean === this.#identity.name) return;
    this.#identity.name = clean;
    if (this.#helloName !== null && this.#helloName !== clean) this.#transport.reconnect?.();
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
    if (opts.tutorial) frame.tutorial = true;
    const hints = opts.hints ?? this.#defaultHints?.(opts.mode) ?? null;
    if (hints) frame.hints = { ...hints };
    if (opts.mode !== 'online') {
      const allow = opts.allowUndo ?? this.#defaultAllowUndo?.(opts.mode) ?? null;
      if (allow !== null) frame.allowUndo = allow;
    }
    if (this.#intent?.kind === 'create') {
      this.#intent.reject(new SessionError('superseded', 'session.error.unknown'));
    }
    this.#leaveRoom();
    return new Promise<string>((resolve, reject) => {
      this.#intent = { kind: 'create', frame, resolve, reject };
      if (this.#welcomed) this.#send(frame);
    });
  }

  /**
   * Joins a room by code (a fresh state follows). With a device link's
   * claim, this connection plays the seat the link was made for.
   */
  join(code: string, claim: string | null = null): void {
    const c = normalizeCode(code);
    if (this.#intent?.kind === 'create') {
      this.#intent.reject(new SessionError('superseded', 'session.error.unknown'));
    }
    if (c !== this.code || claim !== this.claim) this.#leaveRoom();
    this.code = c;
    this.claim = claim;
    this.roomError = null;
    this.#intent = { kind: 'join', code: c, claim };
    if (this.#welcomed) this.#send(this.#joinFrame(c, claim));
  }

  /**
   * Asks for the device link of a seat (the seat's own player or the
   * game's admin may) and resolves with its claim. With reset, the seat's
   * old links stop working first. Rejects when the server refuses, the
   * link is down or no answer comes.
   */
  requestLink(seat: 0 | 1, reset = false): Promise<string> {
    const code = this.code;
    if (!code || !this.#welcomed || !this.frame) {
      return Promise.reject(new SessionError('offline', 'session.error.unknown'));
    }
    return new Promise<string>((resolve, reject) => {
      const waiter: LinkWaiter = {
        seat,
        reset,
        resolve,
        reject,
        timer: setTimeout(() => {
          this.#links = this.#links.filter((w) => w !== waiter);
          reject(new SessionError('timeout', 'session.error.unknown'));
        }, LINK_TIMEOUT_MS)
      };
      this.#links = [...this.#links, waiter];
      this.#send(reset ? { t: 'relink', code, seat } : { t: 'link', code, seat });
    });
  }

  /** Resets a seat's device link: the old one stops working. Resolves with the new claim. */
  resetLink(seat: 0 | 1): Promise<string> {
    return this.requestLink(seat, true);
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

  /**
   * Forfeits the game for resignSeat: the other seat wins at once. The key
   * is not among the choices; the server checks the seat. Returns whether
   * it was sent.
   */
  resign(): boolean {
    const f = this.frame;
    const seat = this.resignSeat;
    if (seat === null || !f || !this.code) return false;
    const frame: Act = {
      t: 'act',
      code: this.code,
      seq: this.#seq + 1,
      base: f.version,
      key: `resign:${seat}`
    };
    if (f.room.mode === 'hotseat') frame.seat = seat;
    this.#request(frame);
    return true;
  }

  /**
   * Replaces the game's hint settings (the admin only, while the game
   * runs; canSetHints says when). Every connection then receives them, with
   * a room log entry. Returns whether it was sent.
   */
  setHints(hints: HintSettings): boolean {
    if (!this.canSetHints || !this.code) return false;
    this.#request({ t: 'hints', code: this.code, seq: this.#seq + 1, hints: { ...hints } });
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

  /**
   * Asks for a rematch once the game is over; the code arrives in
   * rematchCode. Asking again answers the same code, so a player who saw
   * the other's code asks too. A spectator cannot ask: it follows a code
   * the players already made.
   */
  rematch(): void {
    if (!this.code) return;
    const spectator = this.frame !== null && this.frame.you.seats.length === 0;
    if (spectator && !this.rematchCode) return;
    this.rematchAsked = true;
    // Once welcomed again after a drop, #resume asks.
    if (!spectator && this.#welcomed) this.#send({ t: 'rematch', code: this.code });
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
    this.#failLinks('closed');
    for (const off of this.#offs) off();
    this.#frames.clear();
    this.#transport.close();
  }

  // Internals ----------------------------------------------------------------

  #send(msg: ClientMessage): boolean {
    return this.#transport.send(msg);
  }

  #request(frame: Act | Undo | HintsFrame): void {
    this.#seq = frame.seq;
    this.#undoResent = false;
    this.#staleUndo = null;
    this.#track({ frame, sentAt: this.#now() });
    this.slow = false;
    this.notice = null;
    // When the link is down this sends nothing; the request goes out again
    // after the reconnect (#resume).
    this.#send(frame);
  }

  // Makes `pending` the request in flight, with its slow timer.
  #track(pending: PendingRequest): void {
    this.pending = pending;
    this.#rateTries = 0;
    clearTimeout(this.#rateTimer);
    clearTimeout(this.#slowTimer);
    const left = Math.max(0, this.#slowMs - (this.#now() - pending.sentAt));
    this.#slowTimer = setTimeout(() => {
      if (this.pending === pending) this.slow = true;
    }, left);
  }

  #settle(): void {
    clearTimeout(this.#slowTimer);
    this.#slowTimer = undefined;
    clearTimeout(this.#rateTimer);
    this.#rateTimer = undefined;
    this.pending = null;
    this.slow = false;
    this.#staleUndo = null;
  }

  #joinFrame(code: string, claim: string | null): Join {
    return claim ? { t: 'join', code, claim } : { t: 'join', code };
  }

  // Ends every link request still waiting, with a refusal of code.
  #failLinks(code: string, message: NoticeKey = 'session.error.unknown'): void {
    const waiting = this.#links;
    this.#links = [];
    for (const w of waiting) {
      clearTimeout(w.timer);
      w.reject(new SessionError(code, message));
    }
  }

  #leaveRoom(): void {
    this.#failLinks('left');
    this.code = null;
    this.claim = null;
    this.frame = null;
    this.view = null;
    this.rematchCode = null;
    this.rematchClaim = null;
    this.rematchAsked = false;
    this.roomError = null;
    this.notice = null;
    this.history = [];
    this.#covered = null;
    this.#gapAsked = null;
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
        this.#helloName = this.#identity.name;
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
        this.#helloName = null;
        this.conn = status.state;
        this.#failLinks(status.state);
        break;
      case 'closed':
        this.#welcomed = false;
        this.#helloName = null;
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
          const { resolve, frame } = this.#intent;
          this.code = msg.code;
          this.#intent = { kind: 'join', code: msg.code, claim: null };
          // Remembered before the page moves to the room, so the room page
          // knows this browser made it and asks for no name first.
          this.#rooms?.remember(msg.code, frame.mode);
          resolve(msg.code);
        } else {
          // A rematch room: the client joins it by its code. It is this
          // browser's room too, of the same mode as the one on screen. A
          // device on a claim plays it with the claim the server sent.
          this.rematchCode = msg.code;
          this.rematchClaim = msg.claim ?? null;
          const mode = this.frame?.room.mode;
          if (mode) this.#rooms?.remember(msg.code, mode, Date.now(), msg.claim);
        }
        break;
      case 'state':
        this.#onState(msg);
        break;
      case 'linked':
        this.#onLinked(msg);
        break;
      case 'reject':
        if (this.pending && msg.seq === this.pending.frame.seq) {
          if (msg.code === 'rate' && this.#rateTries < this.#rateRetries) {
            this.#retryAfterRate(this.pending);
            return;
          }
          if (msg.code === 'stale' && this.#mayResendUndo(this.pending)) {
            // The fresh state follows at once; #onState sends the undo again.
            this.#staleUndo = this.pending;
            return;
          }
          this.#settle();
        }
        if (msg.seq === 0 && this.#links.length > 0 && msg.code !== 'rate') {
          // A link request refused (not the seat's player nor the admin).
          this.#failLinks(msg.code, rejectMessage(msg.code));
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
        if (msg.code === 'bad-claim' && this.#intent?.kind === 'join') {
          // The device link does not work (any more): the room is closed
          // to this device until it is opened without the claim, or with a
          // fresh link. Home's list forgets the claim, not the room.
          this.roomError = msg.code;
          this.#rooms?.dropClaim(this.#intent.code);
          this.#failLinks(msg.code, errorMessage(msg.code));
          break;
        }
        if (this.#intent?.kind === 'create') {
          this.#intent.reject(new SessionError(msg.code, errorMessage(msg.code)));
          this.#intent = null;
        } else if (this.#intent?.kind === 'join' && !this.frame) {
          // The join failed: there is no room to show.
          this.roomError = msg.code;
          if (msg.code === 'room-not-found' || msg.code === 'room-archived')
            this.#rooms?.forget(this.#intent.code);
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
    this.#takeHistory(frame, this.frame);
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
      this.#rooms?.remember(frame.code, frame.room.mode, Date.now(), this.claim ?? undefined);
    }
    if (frame.room.status !== 'finished') {
      this.#rooms?.updateDetails(frame.code, {
        age: frame.view.age,
        bot: frame.room.seats.find((seat) => seat.kind === 'bot')?.bot,
        ...(frame.room.tutorial ? { tutorial: true } : {})
      });
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
    const stale = this.#staleUndo;
    if (stale && this.pending === stale) {
      this.#staleUndo = null;
      this.#resendUndo(stale, frame);
    }
    this.#frames.emit(frame);
  }

  // The answer to a link request: the waiting requests for that seat
  // resolve. A device on a claim that has just reset its own seat's link
  // plays on under the new claim (the server moved it there).
  #onLinked(msg: Linked): void {
    if (msg.code !== this.code) return;
    const done = this.#links.filter((w) => w.seat === msg.seat);
    this.#links = this.#links.filter((w) => w.seat !== msg.seat);
    const reset = done.some((w) => w.reset);
    const mine = this.frame?.you.seats.includes(msg.seat) ?? false;
    if (reset && mine && this.claim && this.claim !== msg.claim) {
      this.claim = msg.claim;
      if (this.#intent?.kind === 'join') this.#intent = { ...this.#intent, claim: msg.claim };
      const mode = this.frame?.room.mode;
      if (mode) this.#rooms?.remember(msg.code, mode, Date.now(), msg.claim);
    }
    for (const w of done) {
      clearTimeout(w.timer);
      w.resolve(msg.claim);
    }
  }

  // Keeps the history in step with a new frame (see the file comment).
  #takeHistory(frame: State, prev: State | null): void {
    const ply = frame.view.ply;
    let history = this.history;
    let covered = this.#covered;
    if (frame.history) {
      history = frame.history;
      const last = history[history.length - 1];
      // An empty history is a game with no action yet.
      covered = last ? last.ply + 1 : 0;
      prev = null;
    }
    let gap = false;
    if (frame.events.length > 0) {
      const start = covered ?? Math.max(0, ply - 1);
      if (ply > start) {
        // The resign event names the seat that forfeited; otherwise the
        // seat to move before this frame took the action.
        const resign = frame.events.find((e) => e.kind === 'resign');
        const before = prev && prev.view.ply === start ? prev.view.toMove : undefined;
        const seat = resign?.seat ?? before ?? frame.events.find((e) => e.seat !== undefined)?.seat;
        const entry: HistoryEntry = {
          ply: start,
          seat: seat === 1 ? 1 : 0,
          events: frame.events
        };
        history = [...history, entry];
        gap = ply - start > 1;
        covered = ply;
      }
      // Otherwise the events are of actions the history already holds (a
      // frame sent again): nothing to add.
    } else if (covered !== null && ply !== covered) {
      gap = true;
      covered = ply;
    } else if (covered === null) {
      covered = ply;
    }
    if (history !== this.history) this.history = history;
    this.#covered = covered;
    if (gap && this.#gapAsked !== frame.version) {
      this.#gapAsked = frame.version;
      this.resync();
    }
  }

  // Whether a refused request may go out once more with the fresh version:
  // an undo of a bot room (only the bot moves while the player's undo is
  // pending, so a stale base there means the bot's answer came first), or
  // a forfeit (the other side moved first), not sent again yet.
  #mayResendUndo(pending: PendingRequest): boolean {
    if (this.#undoResent) return false;
    if (pending.frame.t === 'act') return pending.frame.key.startsWith('resign:');
    return pending.frame.t === 'undo' && this.frame?.room.mode === 'ai';
  }

  // The fresh state after a stale undo: when the room moved on (the bot's
  // answer to the move being undone) and still takes an undo, the same
  // request goes out with the fresh version as its base; otherwise the
  // refusal shows as any other would.
  #resendUndo(stale: PendingRequest, frame: State): void {
    const undo = stale.frame;
    if (undo.t === 'hints') {
      this.#settle();
      return;
    }
    const resign = undo.t === 'act' && frame.room.status === 'playing' && !frame.view.result;
    if (resign && frame.version > undo.base) {
      this.#undoResent = true;
      const again: Act = { ...undo, base: frame.version };
      this.#track({ frame: again, sentAt: stale.sentAt });
      this.#send(again);
      return;
    }
    if (undo.t === 'undo' && frame.version > undo.base && frame.canUndo) {
      this.#undoResent = true;
      const again: Undo = { ...undo, base: frame.version };
      this.#track({ frame: again, sentAt: stale.sentAt });
      this.#send(again);
      return;
    }
    this.#settle();
    this.notice = { kind: 'reject', code: 'stale', message: rejectMessage('stale') };
  }

  // After welcome: carry on with what the session was doing.
  #resume(): void {
    const intent = this.#intent;
    if (intent?.kind === 'create') {
      this.#send(intent.frame);
      return;
    }
    if (intent?.kind === 'join') {
      this.#send(this.#joinFrame(intent.code, intent.claim));
      // The server answers in order: the fresh state of the join, then the
      // ack or the refusal of the re-sent request.
      if (this.pending) this.#send(this.pending.frame);
      // A rematch asked while the link was down (or whose answer was lost):
      // asking again answers the same code.
      const spectator = this.frame !== null && this.frame.you.seats.length === 0;
      if (this.rematchAsked && !this.rematchCode && !spectator) {
        this.#send({ t: 'rematch', code: intent.code });
      }
    }
  }
}
