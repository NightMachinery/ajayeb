// The Haftgane Session and Presenter over frames the real table actor sent:
// server/internal/tableroom/testdata/client-frames.json, recorded by
// TestRecordClientFrames (three humans and a spectator; a draft, a change,
// a confirmation withdrawn and given again, an illegal key, a pick on a
// decision that just closed, a repeated seq, and a seat given to a bot
// away from its tab). Every frame parses; each client's Session, driven
// by its recorded requests, sends what that client sent, byte for byte;
// and its Presenter plays the resolution and only restates the rest.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import type { TableState } from '$lib/wire/platform.gen';
import { FakeConnection } from './fake-connection';
import { parseFrame } from './frames';
import { TablePresenter } from './presenter.svelte';
import { TableSession, type Notice } from './session.svelte';

interface Entry {
  send?: Record<string, unknown>;
  /** A request the Session never makes itself; only its answer is replayed. */
  raw?: boolean;
  recv?: Record<string, unknown>;
}

interface Recording {
  code: string;
  clients: Record<string, { user: string; log: Entry[] }>;
}

const recording = JSON.parse(
  readFileSync(
    new URL(
      '../../../../../../server/internal/tableroom/testdata/client-frames.json',
      import.meta.url
    ),
    'utf8'
  )
) as Recording;

async function idle(p: TablePresenter): Promise<void> {
  for (let i = 0; i < 20 && !p.idle; i++) await Promise.resolve();
  await new Promise((r) => setTimeout(r, 0));
}

// Makes the Session send `want` through its own API, as the table screen
// would: a pick, a confirmation, a withdrawal or an admin's seat switch.
function request(session: TableSession, want: Record<string, unknown>): boolean {
  if (want.t === 'table-seat') {
    return session.switchSeat(want.seat as number, { bot: want.bot as string });
  }
  const pending = session.table?.pending;
  if (pending && pending.key === want.key) {
    return want.confirm ? session.confirm() : session.withdraw();
  }
  return session.pick(want.key as string, want.confirm as boolean);
}

interface Replay {
  session: TableSession;
  connection: FakeConnection;
  presenter: TablePresenter;
  played: TableState[];
  /** The frames received, in order. */
  frames: TableState[];
  /** The notice after each received message, when it changed. */
  notices: (Notice | null)[];
}

async function replay(id: string): Promise<Replay> {
  const log = recording.clients[id]!.log;
  const connection = new FakeConnection();
  // The join's frame is the bootstrap's handover.
  connection.latest = log[0]!.recv;
  const session = new TableSession({
    connection,
    code: recording.code.toLowerCase(),
    claim: null,
    clientId: id,
    rateRetryMs: 10
  });
  const played: TableState[] = [];
  const presenter = new TablePresenter(session, {
    afterSwap: () => Promise.resolve(),
    hooks: { play: (next) => void played.push(next) }
  });
  const frames: TableState[] = [log[0]!.recv as unknown as TableState];
  const notices: (Notice | null)[] = [];
  await idle(presenter);
  for (const entry of log.slice(1)) {
    if (entry.send) {
      if (entry.raw) continue;
      const before = connection.sent.length;
      expect(request(session, entry.send), JSON.stringify(entry.send)).toBe(true);
      expect(connection.sent.length).toBe(before + 1);
      expect(connection.last()).toEqual(entry.send);
      continue;
    }
    connection.frame(entry.recv);
    if (entry.recv!.t === 'table-state') frames.push(entry.recv as unknown as TableState);
    if (session.notice !== (notices.at(-1) ?? null)) notices.push(session.notice);
    await idle(presenter);
  }
  expect(connection.failed).toEqual([]);
  expect(connection.resyncs).toBe(0);
  expect(connection.sent).toEqual(log.filter((e) => e.send && !e.raw).map((e) => e.send));
  return { session, connection, presenter, played, frames, notices };
}

const resolution = (f: TableState) => f.events.length > 0;

describe('the recorded table', () => {
  it('sends only frames the client accepts', () => {
    for (const { log } of Object.values(recording.clients)) {
      for (const entry of log) {
        if (!entry.recv) continue;
        const { frame, invalid } = parseFrame(entry.recv);
        expect(invalid, JSON.stringify(entry.recv).slice(0, 200)).toBe(false);
        expect(frame).not.toBeNull();
      }
    }
  });

  it('seat 0 drafts, changes, confirms, withdraws, confirms, and gives seat 1 to a bot', async () => {
    const r = await replay('seat-0');
    const picks = r.connection.sent.filter((m) => m.t === 'table-pick');
    expect(picks.map((m) => [m.seq, m.confirm])).toEqual([
      [1, false],
      [2, false],
      [3, true],
      [4, false],
      [5, true]
    ]);
    expect(picks[0]!.key).not.toBe(picks[1]!.key);
    // The repeated seq is answered with an acknowledging frame, no notice.
    expect(r.notices).toEqual([]);
    expect(r.session.pending).toBeNull();
    expect(r.session.seat).toBe(0);
    expect(r.session.table?.seats[1]?.kind).toBe('bot');
    expect(r.played.length).toBe(1);
    expect(resolution(r.played[0]!)).toBe(true);
    expect(r.presenter.frame).toBe(r.session.table);
    r.session.close();
  });

  it('seat 1 is told its decision closed, then watches once its seat is a bot', async () => {
    const r = await replay('seat-1');
    // The illegal key and the pick from the lost seat were not the
    // Session's own requests: their refusals change nothing.
    expect(r.notices.map((n) => n?.code)).toEqual(['decision-closed', 'watching']);
    expect(r.session.seat).toBeNull();
    expect(r.session.canPick).toBe(false);
    expect(r.session.choices).toEqual([]);
    // The frame after the switch restates the game with its history and
    // replaces the old view: shown, never played.
    const last = r.frames.at(-1)!;
    expect(last.history).toBeDefined();
    expect(r.played.every((f) => f.history === undefined && resolution(f))).toBe(true);
    expect(r.played.length).toBe(1);
    expect(r.presenter.frame).toBe(last);
    expect(r.presenter.shownEvents).toEqual([]);
    expect(r.session.history.length).toBe(last.history!.length);
    r.session.close();
  });

  it('seat 2 confirms the pick that resolves the point', async () => {
    const r = await replay('seat-2');
    expect(r.connection.sent.map((m) => m.confirm)).toEqual([true]);
    expect(r.notices).toEqual([]);
    expect(r.played.length).toBe(1);
    r.session.close();
  });

  it('the spectator never holds a pick or a choice', async () => {
    const r = await replay('watcher');
    expect(r.connection.sent).toEqual([]);
    for (const f of r.frames) {
      expect(f.you.seat ?? null).toBeNull();
      expect(f.pending).toBeUndefined();
      expect(f.view.you ?? null).toBeNull();
    }
    expect(r.played.length).toBe(1);
    expect(r.session.history.length).toBe(1);
    r.session.close();
  });
});
