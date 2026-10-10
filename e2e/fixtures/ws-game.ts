// A minimal protocol client in Node (docs/protocol.md), for specs that need
// a game position the UI would take minutes to click its way to. It plays a
// seeded hot-seat game over the real WebSocket with a fixed, deterministic
// policy and records the action keys, so a spec can then open the same game
// at any point of it in the browser through POST /api/dev/rooms with that
// seed and a prefix of the keys.
//
// Like every client it decides nothing: each key it sends is copied from
// the latest state's choices. It paces its frames under the server's frame
// rate limit (10 per second, burst 20) and re-sends an act the server
// dropped with `reject rate`.

import { randomBytes, randomUUID } from 'node:crypto';
import type { Choice, State, View } from '../../web/src/lib/wire';
import type { CapturedFrame, ViewerStream } from './leak-check';

/** A seeded room made through the development endpoint. */
export interface DevRoom {
  code: string;
  path: string;
}

/** Positions of a recorded game, as the number of keys applied to reach each. */
export interface GameMarks {
  /** The wonder draft, before any pick. */
  draft: number;
  /** Age I with its structure full, after the draft. */
  age1: number;
  /** Age II with its structure full, after its starter was chosen. */
  age2: number;
  /** Age III, likewise. */
  age3: number;
  /** The end of the game (all keys). */
  gameOver: number;
}

export interface RecordedGame {
  seed: string;
  keys: string[];
  marks: GameMarks;
  result: NonNullable<View['result']>;
}

/** A token of the shape the server and the client accept: 22 base64url characters. */
export function newToken(): string {
  return randomBytes(16).toString('base64url');
}

/** POST /api/dev/rooms (needs a server started with --dev --dev-endpoints). */
export async function createDevRoom(
  origin: string,
  body: { mode: 'hotseat' | 'ai' | 'online'; seed?: string; scenario?: string; actions?: string[] }
): Promise<DevRoom> {
  const res = await fetch(`${origin}/api/dev/rooms`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  const data = (await res.json().catch(() => null)) as {
    ok?: boolean;
    code?: string;
    path?: string;
    msg?: string;
  } | null;
  if (!res.ok || !data?.ok || typeof data.code !== 'string') {
    throw new Error(`POST /api/dev/rooms ${res.status}: ${JSON.stringify(data)}`);
  }
  return { code: data.code, path: data.path ?? `/play/${data.code}` };
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

/** The minimum gap between two frames: under the 10 per second refill. */
const FRAME_GAP_MS = 120;

type Frame = { t: string; seq?: number; code?: string; msg?: string } & Omit<
  Partial<State>,
  't' | 'code'
>;

/** One WebSocket connection that says hello and then answers one request at a time. */
class Connection {
  #ws: WebSocket;
  #frames: Frame[] = [];
  #waiters: Array<() => void> = [];
  #lastSent = 0;
  #closed: string | null = null;
  #started = Date.now();
  /** Every frame received, as text, in order (for the leak check). */
  readonly raw: CapturedFrame[] = [];
  /** The newest state frame received. */
  latest: State | null = null;

  constructor(url: string) {
    this.#ws = new WebSocket(url);
    this.#ws.addEventListener('message', (event) => {
      const text = String(event.data);
      const frame = JSON.parse(text) as Frame;
      this.raw.push({
        text,
        version: frame.t === 'state' && typeof frame.version === 'number' ? frame.version : null,
        at: Date.now() - this.#started,
        socket: 1
      });
      if (frame.t === 'state') this.latest = frame as State;
      this.#frames.push(frame);
      this.#wake();
    });
    this.#ws.addEventListener('close', (event) => {
      this.#closed = `closed ${event.code} ${event.reason}`;
      this.#wake();
    });
    this.#ws.addEventListener('error', () => {
      this.#closed ??= 'socket error';
      this.#wake();
    });
  }

  #wake(): void {
    const waiters = this.#waiters;
    this.#waiters = [];
    for (const w of waiters) w();
  }

  async open(): Promise<void> {
    if (this.#ws.readyState === WebSocket.OPEN) return;
    await new Promise<void>((resolve, reject) => {
      this.#ws.addEventListener('open', () => resolve(), { once: true });
      this.#ws.addEventListener('error', () => reject(new Error('WebSocket failed to open')), {
        once: true
      });
    });
  }

  async send(frame: object): Promise<void> {
    const wait = this.#lastSent + FRAME_GAP_MS - Date.now();
    if (wait > 0) await sleep(wait);
    this.#lastSent = Date.now();
    this.#ws.send(JSON.stringify(frame));
  }

  /** Takes frames until one satisfies `want`; throws on an error frame or a close. */
  async next(want: (f: Frame) => boolean, timeoutMs = 15_000): Promise<Frame> {
    const end = Date.now() + timeoutMs;
    for (;;) {
      while (this.#frames.length > 0) {
        const f = this.#frames.shift()!;
        if (f.t === 'error') throw new Error(`server error frame: ${JSON.stringify(f)}`);
        if (want(f)) return f;
      }
      if (this.#closed) throw new Error(`WebSocket ${this.#closed}`);
      const left = end - Date.now();
      if (left <= 0) throw new Error('timed out waiting for a server frame');
      await new Promise<void>((resolve) => {
        const timer = setTimeout(resolve, left);
        this.#waiters.push(() => {
          clearTimeout(timer);
          resolve();
        });
      });
    }
  }

  /** Takes frames until the newest state is at `version` or later. */
  async reach(version: number, timeoutMs = 15_000): Promise<State> {
    while (!this.latest || this.latest.version < version) {
      await this.next((f) => f.t === 'state', timeoutMs);
    }
    return this.latest;
  }

  close(): void {
    this.#ws.close();
  }
}

/**
 * The recording policy: deterministic, varied enough to give both cities
 * cards, and never a rules decision (it only orders the view's own choices).
 * Odd plies discard when they can, so a game rarely ends early by military
 * or science and usually reaches Age III.
 */
export function recordPolicy(view: View): Choice {
  const choices = view.choices;
  const first = (kind: Choice['kind']) => choices.find((c) => c.kind === kind);
  const pick =
    (view.ply % 2 === 1 ? first('discard') : undefined) ??
    first('construct') ??
    first('discard') ??
    choices[0];
  if (!pick) throw new Error(`no choice at ply ${view.ply} (phase ${view.phase})`);
  return pick;
}

/**
 * Plays a whole seeded hot-seat game through the protocol and returns its
 * keys with the positions a layout check wants. Throws when the game ends
 * before Age III.
 */
export async function recordGame(
  origin: string,
  seed: string,
  policy: (view: View) => Choice = recordPolicy
): Promise<RecordedGame> {
  const room = await createDevRoom(origin, { mode: 'hotseat', seed });
  const conn = new Connection(origin.replace(/^http/, 'ws') + '/ws');
  const clientId = randomUUID();
  try {
    await conn.open();
    await conn.send({ t: 'hello', v: 1, token: newToken(), name: 'Layout', clientId });
    await conn.next((f) => f.t === 'welcome');
    await conn.send({ t: 'join', code: room.code });
    let state = (await conn.next((f) => f.t === 'state')) as State;

    const keys: string[] = [];
    const marks: Partial<GameMarks> = { draft: 0 };
    let seq = 0;
    for (let guard = 0; guard < 400; guard++) {
      const view = state.view;
      const fresh =
        view.phase === 'play' &&
        !view.pending &&
        view.structure.length > 0 &&
        view.structure.every((slot) => slot.state !== 'empty');
      if (fresh && view.age === 1) marks.age1 ??= keys.length;
      if (fresh && view.age === 2) marks.age2 ??= keys.length;
      if (fresh && view.age === 3) marks.age3 ??= keys.length;
      if (view.phase === 'gameOver') {
        marks.gameOver = keys.length;
        if (marks.age1 === undefined || marks.age2 === undefined || marks.age3 === undefined) {
          throw new Error(`seed ${seed}: the game ended before Age III (${view.result?.victory})`);
        }
        return { seed, keys, marks: marks as GameMarks, result: view.result! };
      }
      const choice = policy(view);
      seq += 1;
      const act = {
        t: 'act',
        code: room.code,
        seq,
        base: state.version,
        key: choice.key,
        seat: view.toMove
      };
      for (let attempt = 0; ; attempt++) {
        await conn.send(act);
        const answer = await conn.next(
          (f) =>
            (f.t === 'state' && f.ack?.seq === seq && f.ack.clientId === clientId) ||
            (f.t === 'reject' && f.seq === seq)
        );
        if (answer.t === 'state') {
          state = answer as State;
          break;
        }
        if (answer.code !== 'rate' || attempt >= 3) {
          throw new Error(`act ${choice.key} refused: ${JSON.stringify(answer)}`);
        }
        await sleep(500);
      }
      keys.push(choice.key);
    }
    throw new Error(`seed ${seed}: the game did not end within 400 actions`);
  } finally {
    conn.close();
  }
}

export interface RecordedOnlineGame {
  code: string;
  /** The frames of seat 0, seat 1 and the spectator, named seat0, seat1 and spectator. */
  streams: ViewerStream[];
  keys: string[];
  result: NonNullable<View['result']>;
}

/**
 * Plays a whole online game through the protocol with two players and a
 * spectator, each its own user on its own socket, and returns every frame
 * each of them received. The room comes from POST /api/dev/rooms (an
 * explicit seed, so the server's --seed-base counter is not used up), and
 * the players join in seat order. Each move is the policy's pick from the
 * mover's own latest state; the others wait until they have seen it.
 */
export async function recordOnlineGame(
  origin: string,
  opts: { seed: string; scenario?: string; policy?: (view: View) => Choice; maxActions?: number }
): Promise<RecordedOnlineGame> {
  const policy = opts.policy ?? recordPolicy;
  const room = await createDevRoom(origin, {
    mode: 'online',
    seed: opts.seed,
    ...(opts.scenario ? { scenario: opts.scenario } : {})
  });
  const url = origin.replace(/^http/, 'ws') + '/ws';
  const names = ['seat0', 'seat1', 'spectator'] as const;
  const conns = names.map(() => new Connection(url));
  const ids = names.map(() => randomUUID());
  try {
    // Join one after the other, so the first takes seat 0 and the second seat 1.
    for (const [i, conn] of conns.entries()) {
      await conn.open();
      await conn.send({ t: 'hello', v: 1, token: newToken(), name: names[i], clientId: ids[i] });
      await conn.next((f) => f.t === 'welcome');
      await conn.send({ t: 'join', code: room.code });
      await conn.next((f) => f.t === 'state');
    }
    const [c0, c1] = conns as [Connection, Connection, Connection];
    let version = c0.latest?.version ?? 0;
    const keys: string[] = [];
    const seqs = [0, 0];
    for (let guard = 0; guard <= (opts.maxActions ?? 400); guard++) {
      for (const conn of conns) await conn.reach(version);
      const view = c0.latest!.view;
      if (view.phase === 'gameOver') {
        // Every viewer has its game-over frame before the streams are returned.
        return {
          code: room.code,
          streams: conns.map((c, i) => ({ name: names[i]!, frames: c.raw })),
          keys,
          result: view.result!
        };
      }
      const seat = view.toMove;
      if (seat !== 0 && seat !== 1) throw new Error(`no seat to move at ply ${view.ply}`);
      const mover = seat === 0 ? c0 : c1;
      const choice = policy(mover.latest!.view);
      seqs[seat] = (seqs[seat] ?? 0) + 1;
      const seq = seqs[seat]!;
      const act = { t: 'act', code: room.code, seq, base: version, key: choice.key };
      for (let attempt = 0; ; attempt++) {
        await mover.send(act);
        const answer = await mover.next(
          (f) =>
            (f.t === 'state' && f.ack?.seq === seq && f.ack.clientId === ids[seat]) ||
            (f.t === 'reject' && f.seq === seq)
        );
        if (answer.t === 'state') {
          version = (answer as State).version;
          break;
        }
        if (answer.code !== 'rate' || attempt >= 3) {
          throw new Error(`act ${choice.key} refused: ${JSON.stringify(answer)}`);
        }
        await sleep(500);
      }
      keys.push(choice.key);
    }
    throw new Error(`the online game did not end within ${opts.maxActions ?? 400} actions`);
  } finally {
    for (const conn of conns) conn.close();
  }
}
