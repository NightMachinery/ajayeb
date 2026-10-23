import { afterEach, describe, expect, it } from 'vitest';
import { SeqCounters } from '$lib/entry/identity';
import { FakeConnection } from '$lib/games/haftgane/session/fake-connection';
import { TableSession } from '$lib/games/haftgane/session/session.svelte';
import { CODE, lobbyFrame, tableFrame } from '$lib/games/haftgane/session/test-frames';
const active: TableSession[] = [];
afterEach(() => active.splice(0).forEach((s) => s.close()));
function setup() {
  const connection = new FakeConnection();
  connection.latest = lobbyFrame();
  const session = new TableSession({
    connection,
    code: CODE,
    claim: null,
    clientId: 'tab',
    seqs: new SeqCounters()
  });
  active.push(session);
  return { connection, session };
}
describe('production lobby requests', () => {
  it('serializes edits and roles, preserves their payload over reconnect, then starts', () => {
    const { connection, session } = setup();
    expect(session.editLobby({ op: 'add-bot', seat: 2, bot: 'normal' })).toBe(true);
    const request = connection.last();
    expect(request).toEqual({
      t: 'lobby-edit',
      code: CODE,
      seq: 1,
      op: 'add-bot',
      seat: 2,
      bot: 'normal'
    });
    expect(session.lobbyRole('step-down')).toBe(false);
    connection.drop();
    connection.rejoin();
    connection.frame(lobbyFrame());
    expect(connection.last()).toEqual(request);
    connection.frame(lobbyFrame({ ack: { clientId: 'tab', seq: 1 } }));
    expect(session.lobbyRole('promote', 'u1')).toBe(true);
    expect(connection.last()).toEqual({
      t: 'lobby-role',
      code: CODE,
      seq: 2,
      op: 'promote',
      userId: 'u1'
    });
    connection.frame({ t: 'lobby-reject', seq: 2, code: 'forbidden' });
    expect(session.notice?.code).toBe('forbidden');
    expect(session.start(true)).toBe(true);
    connection.frame(tableFrame());
    expect(session.lobby).toBeNull();
    expect(session.pending).toBeNull();
  });
  it('lets the actor distinguish former participants from spectators, and shows refusal', () => {
    const { connection, session } = setup();
    connection.frame(tableFrame({ status: 'finished', point: undefined }, null));
    expect(session.rematch()).toBe(true);
    expect(connection.last()).toEqual({ t: 'rematch', code: CODE });
    connection.frame({ t: 'table-reject', seq: 0, code: 'forbidden' });
    expect(session.rematchAsked).toBe(false);
    expect(session.rematchError).toBe('forbidden');
  });
  it('keeps a returned rematch claim private and follows only after an explicit request', () => {
    const { connection, session } = setup();
    expect(session.rematch()).toBe(false);
    connection.frame(tableFrame({ status: 'finished', point: undefined }));
    connection.frame({ t: 'created', code: 'NEWAAA', claim: 'a'.repeat(22) });
    expect(session.rematchAsked).toBe(false);
    expect(session.rematchCode).toBe('NEWAAA');
    expect(session.rematchClaim).toBe('a'.repeat(22));
    expect(session.rematch()).toBe(true);
    expect(session.rematchAsked).toBe(true);
  });
});
