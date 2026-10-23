// Haftgane's Session: the client's side of protocol 2 for one table or
// shared lobby (docs/protocol.md, "Protocol 2: table and shared lobby").
// It runs over the welcomed v2 connection the entry's bootstrap opened
// (docs/haftgane/platform-architecture.md, "Protocol and lazy client
// entry"): attaching takes the bootstrap's newest snapshot, which is still
// `unknown` until frames.ts accepts it, and every later frame of the room.
// The bootstrap keeps saying hello and joining on every new socket; the
// Session says neither.
//
// It decides nothing about the game. What a seat may choose comes from the
// frame's view.you.choices, and a pick sends one of their keys unchanged;
// whether this tab must decide comes from the frame's point and you.seat.
// No cost, legality or rule is computed here.
//
// The latest frame of each family is kept whole ($state.raw) and replaced
// whole. Nothing else the server sent is kept, except the history the
// frames themselves carry, so nothing here holds more than a frame gave.
// The Session logs no frame.
//
// Requests (docs/protocol.md, "Strict platform requests"): every one
// carries the next seq of this page's client id. The counter lives beside
// the id for the whole page ($lib/entry/identity), not in the Session, so a
// Session mounted again in the same page (Home and back) never repeats a
// seq the table already took; a reload makes a new id, and a new counter
// with it. One is in flight at a time, from the moment it is sent until a
// frame acknowledges its seq or a reject names it. Lobby Start has no ack:
// the table's first frame answers it.
//
// A pick is {point, key, confirm}: the open point's reference as the frame
// gave it, a key of the choices, and whether the seat confirms. Sending the
// chosen key again with confirm false withdraws the confirmation. The
// table answers a duplicate seq with a frame acknowledging it; a refusal
// (`decision-closed` after the last confirmation resolved the point,
// `illegal`, `stale`, `forbidden` when this connection holds no seat) ends
// the request and shows a notice. `rate` (the frame rate limit dropped it,
// or the pick's floor bound refused it) is sent again unchanged a few times
// before the notice shows.
//
// A request the seat itself makes (a pick, an undo, a resignation) names
// no seat: the table applies it to the sender's seat when it arrives. So
// each is bound to the seat it was chosen for, and is never sent again once
// the latest frame shows this connection anywhere else (another seat, or
// watching); a pick, also only at the same point. Reconnecting and a `rate`
// retry ask the same question (#stillValid), and a frame that moves this
// connection to another place drops such a request at once (a pick whose
// point merely closed waits for its answer, decision-closed or the ack).
// A request that names its target
// (a seat switch, a role change) and Start go out again unchanged, which
// their seq makes safe, as it does a same-seat retry.
//
// Reconnecting: after a new socket's join, the Session waits for the join's
// frame before it settles the request in flight: it goes out again (same
// seq) when still valid, else it was applied or overtaken, and the frame's
// own pending pick says which.
//
// Seat changes (former-controller tabs): a switch, a resignation or a
// relink by an admin moves this connection to watching, or to another
// seat. Its next frame carries the whole history again, redacted for its
// new place, which replaces what it held. A revoked device link ends the
// connection with error bad-claim, which the bootstrap turns into the room
// error.

import { entrySeqs, rememberEntryGame, type SeqCounters } from '$lib/entry/identity';
import type { RoomListener } from '$lib/entry/bootstrap';
import { Listeners, type TransportStatus, type Unsubscribe } from '$lib/session/transport';
import type { SavedRooms } from '$lib/session/rooms.svelte';
import type {
  LobbyStart,
  LobbyState,
  PlatformHuman,
  PlatformLink,
  PlatformRelink,
  PointRef,
  TableChoice,
  TableHistoryEntry,
  TablePick,
  TableResign,
  TableRole,
  TableSeat,
  TableState,
  TableUndo
} from '$lib/wire/platform.gen';
import type { HgKey } from '../i18n/messages.svelte';
import { parseFrame, type SessionFrame } from './frames';

/** The connection as the player sees it: 'open' once welcomed. */
export type ConnState = 'connecting' | 'open' | 'reconnecting' | 'offline';

/** The part of the entry's bootstrap the Session uses. */
export interface RoomConnection {
  readonly status: TransportStatus;
  attach(listener: RoomListener): unknown;
  send(message: object): boolean;
  fail(code: string): void;
  resync(): void;
  useClaim(claim: string | null): void;
}

type Request = TablePick | TableUndo | TableSeat | TableRole | TableResign | LobbyStart;

export interface PendingRequest {
  readonly frame: Request;
  /**
   * For a request the seat makes (a pick, an undo, a resignation): the seat
   * it was chosen for. Null for one that names its target, and for Start.
   */
  readonly seat: number | null;
  /** When it was first sent, in ms (the Session's clock). */
  readonly sentAt: number;
}

type RejectKey = Extract<HgKey, `hg.table.reject.${string}`>;

/** The last refusal or seat change, until the next request or dismissNotice(). */
export type Notice =
  | { readonly kind: 'reject'; readonly code: string; readonly message: RejectKey }
  | { readonly kind: 'watching'; readonly code: 'watching' }
  | { readonly kind: 'seated'; readonly code: 'seated'; readonly seat: number };

export class TableSessionError extends Error {
  constructor(readonly code: string) {
    super(code);
    this.name = 'TableSessionError';
  }
}

export interface TableSessionOptions {
  connection: RoomConnection;
  code: string;
  claim: string | null;
  /** This page's client id, the one its hello carried. */
  clientId: string;
  /** Where the client id's seq counter lives: the page's by default. */
  seqs?: SeqCounters;
  /** Where rooms seen are remembered for Home; none in tests. */
  rooms?: Pick<SavedRooms, 'remember' | 'forget'> | null;
  slowMs?: number;
  rateRetryMs?: number;
  rateRetries?: number;
  linkTimeoutMs?: number;
  now?: () => number;
}

const REJECTS: ReadonlySet<string> = new Set([
  'decision-closed',
  'illegal',
  'stale',
  'bad-request',
  'not-admin',
  'forbidden',
  'rate',
  'storage',
  'timeout'
]);

function rejectMessage(code: string): RejectKey {
  return REJECTS.has(code) ? (`hg.table.reject.${code}` as RejectKey) : 'hg.table.reject.unknown';
}

/** A room code as the server writes it (Duel's Session has the same rule). */
function normalizeCode(code: string): string {
  return code.trim().toUpperCase();
}

function sameRef(a: PointRef | undefined, b: PointRef | undefined): boolean {
  return !!a && !!b && a.id === b.id && a.revision === b.revision;
}

interface LinkWaiter {
  seat: number;
  reset: boolean;
  resolve: (claim: string) => void;
  reject: (err: TableSessionError) => void;
  timer: ReturnType<typeof setTimeout>;
}

export class TableSession {
  readonly code: string;
  conn = $state<ConnState>('connecting');
  /** The latest table frame, replaced whole; null in the lobby. */
  table = $state.raw<TableState | null>(null);
  /** The latest lobby frame, replaced whole; null once the table started. */
  lobby = $state.raw<LobbyState | null>(null);
  pending = $state.raw<PendingRequest | null>(null);
  /** The request in flight has waited longer than slowMs. */
  slow = $state(false);
  notice = $state.raw<Notice | null>(null);
  /** The device link's claim this connection plays with, if any. */
  claim = $state<string | null>(null);
  /** One entry per resolved batch, as the frames gave them. */
  history = $state.raw<readonly TableHistoryEntry[]>([]);

  /** This connection's seat, or null while it watches. */
  seat: number | null = $derived(this.table?.you.seat ?? this.lobby?.you.seat ?? null);
  /** This connection's user, as the latest frame lists them. */
  me: PlatformHuman | null = $derived.by(() => {
    const f = this.table ?? this.lobby;
    return f?.humans.find((h) => h.userId === f.you.userId) ?? null;
  });
  /** A real or temporary admin: the seat powers. */
  admin: boolean = $derived(!!this.me && (this.me.admin || this.me.temporary));
  /** The choices of the open point for this seat, as the server sent them. */
  choices: readonly TableChoice[] = $derived(this.table?.view.you?.choices ?? []);
  /** The open point requires this seat. */
  mustDecide: boolean = $derived.by(() => {
    const f = this.table;
    const seat = this.seat;
    return !!f?.point && f.status === 'playing' && seat !== null && f.point.required.includes(seat);
  });
  #ready: boolean = $derived(this.conn === 'open' && !this.pending);
  canPick: boolean = $derived(this.#ready && this.mustDecide && this.choices.length > 0);
  /** An undo would be sent now; the table says whether it is accepted. */
  canUndo: boolean = $derived.by(() => {
    const f = this.table;
    return (
      this.#ready &&
      !!f &&
      f.status === 'playing' &&
      this.seat !== null &&
      f.allowUndo &&
      !f.undoOff &&
      f.startingHumanCount === 1 &&
      f.batch > 0
    );
  });
  canResign: boolean = $derived(
    this.#ready && this.table?.status === 'playing' && this.seat !== null
  );
  canStart: boolean = $derived(this.#ready && !!this.lobby?.canStart);
  /** Admin seat and role changes would be sent now. */
  canControl: boolean = $derived(this.#ready && this.admin && this.table?.status === 'playing');

  #connection: RoomConnection;
  #clientId: string;
  #rooms: Pick<SavedRooms, 'remember' | 'forget'> | null;
  #slowMs: number;
  #rateRetryMs: number;
  #rateRetries: number;
  #linkTimeoutMs: number;
  #now: () => number;
  #seqs: SeqCounters;
  #rateTries = 0;
  /** The undo in flight was sent again once already. */
  #undoResent = false;
  #slowTimer: ReturnType<typeof setTimeout> | undefined;
  #rateTimer: ReturnType<typeof setTimeout> | undefined;
  /** A new socket joined: its first frame settles the request in flight. */
  #rejoined = false;
  /** The batch up to which `history` is complete; null before a table frame. */
  #covered: number | null = null;
  #gapAsked: number | null = null;
  #remembered = false;
  #links: LinkWaiter[] = [];
  #tables = new Listeners<TableState>();
  #closed = false;

  constructor(opts: TableSessionOptions) {
    this.code = normalizeCode(opts.code);
    this.claim = opts.claim;
    this.#connection = opts.connection;
    this.#clientId = opts.clientId;
    this.#seqs = opts.seqs ?? entrySeqs();
    this.#rooms = opts.rooms ?? null;
    this.#slowMs = opts.slowMs ?? 300;
    this.#rateRetryMs = opts.rateRetryMs ?? 250;
    this.#rateRetries = opts.rateRetries ?? 3;
    this.#linkTimeoutMs = opts.linkTimeoutMs ?? 8000;
    this.#now = opts.now ?? (() => Date.now());
    const first = this.#connection.attach({
      message: (m) => this.#onMessage(m),
      status: (s) => this.#onStatus(s),
      joined: () => {
        this.#rejoined = true;
      }
    });
    this.#onStatus(this.#connection.status);
    if (first !== undefined) this.#onMessage(first);
  }

  /** This page's client id: the one its acks name. */
  get clientId(): string {
    return this.#clientId;
  }

  /** Calls fn with every accepted table frame (and at once with the current one). */
  subscribe(fn: (frame: TableState) => void): Unsubscribe {
    const off = this.#tables.add(fn);
    if (this.table) fn(this.table);
    return off;
  }

  /**
   * Picks one of the open point's choices for this seat, confirmed or not.
   * The key must be in the latest frame's choices; the Session never makes
   * one up. Returns whether it was sent.
   */
  pick(key: string, confirm = false): boolean {
    const f = this.table;
    const seat = this.seat;
    if (!this.canPick || !f?.point || seat === null) return false;
    if (!this.choices.some((c) => c.key === key)) return false;
    if (f.pending?.key === key && f.pending.confirm === confirm) return false;
    this.#request(
      {
        t: 'table-pick',
        code: this.code,
        seq: this.#nextSeq(),
        point: { id: f.point.ref.id, revision: f.point.ref.revision },
        key,
        confirm
      },
      seat
    );
    return true;
  }

  /** Confirms the pending pick. */
  confirm(): boolean {
    const p = this.table?.pending;
    return !!p && !p.confirm && this.pick(p.key, true);
  }

  /** Withdraws the pending pick's confirmation, so it can change. */
  withdraw(): boolean {
    const p = this.table?.pending;
    return !!p && p.confirm && this.pick(p.key, false);
  }

  /** Asks to take back the last batch (the sole original human only). */
  undo(): boolean {
    const f = this.table;
    if (!this.canUndo || !f) return false;
    this.#request(
      { t: 'table-undo', code: this.code, seq: this.#nextSeq(), base: f.revision },
      this.seat
    );
    return true;
  }

  /** An admin hands a seat to a bot of a level, or seats a spectator in it. */
  switchSeat(seat: number, to: { bot: string } | { userId: string }): boolean {
    if (!this.canControl) return false;
    const frame: TableSeat =
      'bot' in to
        ? {
            t: 'table-seat',
            code: this.code,
            seq: this.#nextSeq(),
            seat,
            control: 'bot',
            bot: to.bot
          }
        : {
            t: 'table-seat',
            code: this.code,
            seq: this.#nextSeq(),
            seat,
            control: 'human',
            userId: to.userId
          };
    this.#request(frame);
    return true;
  }

  /** Promotes or demotes a user, or steps this user down. */
  role(op: 'promote' | 'demote' | 'step-down', userId?: string): boolean {
    if (!this.#ready || !this.table || this.table.status !== 'playing') return false;
    const frame: TableRole = { t: 'table-role', code: this.code, seq: this.#nextSeq(), op };
    if (op !== 'step-down') {
      if (!userId) return false;
      frame.userId = userId;
    }
    this.#request(frame);
    return true;
  }

  /** Hands this seat to a bot and watches. */
  resign(): boolean {
    if (!this.canResign) return false;
    this.#request(
      { t: 'table-resign', code: this.code, seq: this.#nextSeq(), confirm: true },
      this.seat
    );
    return true;
  }

  /** Starts the lobby's game; open slots get bots when fillBots is set. */
  start(fillBots: boolean): boolean {
    if (!this.canStart) return false;
    this.#request({ t: 'lobby-start', code: this.code, seq: this.#nextSeq(), fillBots });
    return true;
  }

  /**
   * Asks for a seat's device link (its own user or an admin may) and
   * resolves with its claim; with reset, the seat's old links stop working
   * first. Rejects when refused, when the link drops or when no answer comes.
   */
  requestLink(seat: number, reset = false): Promise<string> {
    if (this.conn !== 'open' || !this.table) {
      return Promise.reject(new TableSessionError('offline'));
    }
    const frame: PlatformLink | PlatformRelink = reset
      ? { t: 'relink', code: this.code, seat }
      : { t: 'link', code: this.code, seat };
    return new Promise<string>((resolve, reject) => {
      const waiter: LinkWaiter = {
        seat,
        reset,
        resolve,
        reject,
        timer: setTimeout(() => {
          this.#links = this.#links.filter((w) => w !== waiter);
          reject(new TableSessionError('timeout'));
        }, this.#linkTimeoutMs)
      };
      this.#links = [...this.#links, waiter];
      if (!this.#connection.send(frame)) this.#failLinks('offline');
    });
  }

  /** Asks for the whole frame again, with the history. */
  resync(): void {
    this.#connection.resync();
  }

  dismissNotice(): void {
    this.notice = null;
  }

  /** Stops listening; the entry's bootstrap owns and closes the connection. */
  close(): void {
    if (this.#closed) return;
    this.#closed = true;
    this.#settle();
    this.#failLinks('closed');
    this.#tables.clear();
    this.#connection.attach({ message: () => {}, status: () => {}, joined: () => {} });
  }

  // Internals ----------------------------------------------------------------

  #nextSeq(): number {
    return this.#seqs.next(this.#clientId);
  }

  #request(frame: Request, seat: number | null = null): void {
    this.#seqs.sent(this.#clientId, frame.seq);
    this.#undoResent = false;
    this.#track({ frame, seat, sentAt: this.#now() });
    this.notice = null;
    // With the link down this sends nothing, and after a new socket's join
    // it waits: either way it goes out after the join's frame (#afterJoin).
    if (!this.#rejoined) this.#connection.send(frame);
  }

  #track(pending: PendingRequest): void {
    this.pending = pending;
    this.slow = false;
    this.#rateTries = 0;
    clearTimeout(this.#rateTimer);
    clearTimeout(this.#slowTimer);
    this.#slowTimer = setTimeout(() => {
      if (this.pending === pending) this.slow = true;
    }, this.#slowMs);
  }

  #settle(): void {
    clearTimeout(this.#slowTimer);
    clearTimeout(this.#rateTimer);
    this.#slowTimer = this.#rateTimer = undefined;
    this.pending = null;
    this.slow = false;
  }

  /**
   * Whether the request in flight may go out again under the latest frame,
   * after a reconnect or a `rate` refusal alike: one the seat makes only
   * while the frame still shows this connection in that seat, and a pick
   * only at the same point (see the file comment).
   */
  #stillValid(pending: PendingRequest): boolean {
    if (pending.seat === null) return true;
    const f = this.table;
    if (!f || f.status !== 'playing' || f.you.seat !== pending.seat) return false;
    const frame = pending.frame;
    return frame.t !== 'table-pick' || sameRef(f.point?.ref, frame.point);
  }

  #failLinks(code: string): void {
    const waiting = this.#links;
    this.#links = [];
    for (const w of waiting) {
      clearTimeout(w.timer);
      w.reject(new TableSessionError(code));
    }
  }

  #onStatus(status: TransportStatus): void {
    if (this.#closed) return;
    switch (status.state) {
      case 'connecting':
      case 'open':
      case 'reconnecting':
      case 'offline':
        this.conn = status.state;
        break;
      case 'closed':
        this.conn = 'offline';
        break;
    }
    if (status.state !== 'open') this.#failLinks(status.state);
  }

  #onMessage(message: unknown): void {
    if (this.#closed) return;
    const { frame, invalid } = parseFrame(message);
    if (invalid) {
      // Client and server disagree on a frame's shape: only a reload helps.
      this.#connection.fail('protocol');
      return;
    }
    if (frame) this.#onFrame(frame);
  }

  #onFrame(frame: SessionFrame): void {
    switch (frame.t) {
      case 'table-state':
        if (frame.code === this.code) this.#onTable(frame);
        break;
      case 'lobby-state':
        if (frame.code === this.code) this.#onLobby(frame);
        break;
      case 'table-reject':
      case 'lobby-reject':
      case 'reject':
        this.#onReject(frame.seq, frame.code);
        break;
      case 'linked':
        if (frame.code === this.code) this.#onLinked(frame.seat, frame.claim);
        break;
      case 'created':
        // A rematch's successor; following it belongs to the lobby screens
        // (M18.home-picker-and-lobby).
        break;
    }
  }

  #onTable(frame: TableState): void {
    const prev = this.table;
    this.#takeHistory(frame);
    this.table = frame;
    this.lobby = null;
    if (!this.#remembered || frame.status === 'finished') {
      this.#remembered = true;
      rememberEntryGame(this.code, 'haftgane');
      // Home's Continue list offers live games only.
      if (frame.status === 'finished') this.#rooms?.forget(this.code);
      else this.#rooms?.remember(this.code, 'online', this.#now(), this.claim, 'haftgane');
    }
    const seatNow = frame.you.seat ?? null;
    const seatBefore = prev ? (prev.you.seat ?? null) : seatNow;
    if (prev && seatNow !== seatBefore) {
      this.notice =
        seatNow === null
          ? { kind: 'watching', code: 'watching' }
          : { kind: 'seated', code: 'seated', seat: seatNow };
    }
    this.#answer(frame.ack);
    // Start has no ack: the table's first frame answers it.
    if (this.pending?.frame.t === 'lobby-start') this.#settle();
    if (this.#rejoined) this.#afterJoin();
    // A frame that moved this connection: what its old seat asked for is
    // never sent again (see the file comment).
    else if (this.pending && this.pending.seat !== null && this.pending.seat !== seatNow) {
      this.#settle();
    }
    this.#tables.emit(frame);
  }

  #onLobby(frame: LobbyState): void {
    this.lobby = frame;
    this.#answer(frame.ack);
    this.#afterJoin();
  }

  // Settles the request in flight that this frame acknowledges.
  #answer(ack: { clientId: string; seq: number } | undefined): void {
    if (this.pending && ack?.clientId === this.#clientId && ack.seq === this.pending.frame.seq) {
      this.#settle();
    }
  }

  // The first frame after a new socket's join settles the request in flight:
  // sent again when still valid, else dropped (see the file comment).
  #afterJoin(): void {
    if (!this.#rejoined) return;
    this.#rejoined = false;
    const pending = this.pending;
    if (!pending) return;
    if (this.#stillValid(pending)) this.#connection.send(pending.frame);
    else this.#settle();
  }

  #onReject(seq: number, code: string): void {
    const pending = this.pending;
    if (pending && seq === pending.frame.seq && seq !== 0) {
      if (code === 'rate' && this.#rateTries < this.#rateRetries) {
        this.#retryAfterRate(pending);
        return;
      }
      if (code === 'stale' && pending.frame.t === 'table-undo' && this.#resendUndo(pending)) return;
      this.#settle();
      this.notice = { kind: 'reject', code, message: rejectMessage(code) };
      return;
    }
    if (seq === 0 && this.#links.length > 0) {
      // A link or relink refused, or dropped under the frame rate limit.
      this.#failLinks(code);
      this.notice = { kind: 'reject', code, message: rejectMessage(code) };
    }
  }

  // The request was dropped unhandled: send it again unchanged after a pause,
  // while it is still meant for what the latest frame shows.
  #retryAfterRate(pending: PendingRequest): void {
    this.#rateTries++;
    clearTimeout(this.#rateTimer);
    this.#rateTimer = setTimeout(() => {
      this.#rateTimer = undefined;
      if (this.pending !== pending) return;
      if (!this.#stillValid(pending)) {
        this.#settle();
        return;
      }
      this.#connection.send(pending.frame);
    }, this.#rateRetryMs * this.#rateTries);
  }

  // A sole human's undo crossed a bot's confirmation: the frames that moved
  // the table on came first, so send it once more with their revision.
  #resendUndo(pending: PendingRequest): boolean {
    const undo = pending.frame;
    const f = this.table;
    if (this.#undoResent || undo.t !== 'table-undo' || !f || f.revision <= undo.base) return false;
    if (!this.#undoAllowed(f) || !this.#stillValid(pending)) return false;
    this.#undoResent = true;
    const again: TableUndo = { ...undo, base: f.revision };
    this.#track({ frame: again, seat: pending.seat, sentAt: pending.sentAt });
    this.#connection.send(again);
    return true;
  }

  #undoAllowed(f: TableState): boolean {
    return (
      f.status === 'playing' &&
      f.you.seat !== undefined &&
      f.allowUndo &&
      !f.undoOff &&
      f.startingHumanCount === 1 &&
      f.batch > 0
    );
  }

  #onLinked(seat: number, claim: string): void {
    const done = this.#links.filter((w) => w.seat === seat);
    this.#links = this.#links.filter((w) => w.seat !== seat);
    // A device on a claim that relinked its own seat keeps its place under
    // the new claim (docs/protocol.md, "Device links").
    if (done.some((w) => w.reset) && this.claim && this.seat === seat && this.claim !== claim) {
      this.claim = claim;
      this.#connection.useClaim(claim);
      this.#rooms?.remember(this.code, 'online', this.#now(), claim, 'haftgane');
    }
    for (const w of done) {
      clearTimeout(w.timer);
      w.resolve(claim);
    }
  }

  // Keeps the history in step: a frame with `history` replaces it whole (a
  // join, a resync, an undo, or this connection's seat changing), and a
  // resolution's events add its batch. A batch that skips ahead asks for a
  // resync once.
  #takeHistory(frame: TableState): void {
    if (frame.history) {
      this.history = frame.history;
      this.#covered = frame.batch;
      return;
    }
    const covered = this.#covered;
    let gap = false;
    if (frame.events.length > 0) {
      if (covered === null || frame.batch > covered) {
        this.history = [...this.history, { batch: frame.batch, events: frame.events }];
        gap = covered !== null && frame.batch > covered + 1;
        this.#covered = frame.batch;
      }
    } else if (covered === null) {
      this.#covered = frame.batch;
    } else if (frame.batch !== covered) {
      gap = true;
      this.#covered = frame.batch;
    }
    if (gap && this.#gapAsked !== frame.revision) {
      this.#gapAsked = frame.revision;
      this.resync();
    }
  }
}
