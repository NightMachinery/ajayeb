import { describe, expect, it } from 'vitest';
import type { ServerMessage, State, View } from '$lib/wire';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import { createDevRoom, DevRoomError } from './dev-rooms';
import { FixtureTransport, stateFrameFromView } from './fixture-transport';
import { fixtureFrames, fixtureList } from './fixtures';
import { Presenter } from './presenter.svelte';
import { Session } from './session.svelte';

const settle = () => new Promise((resolve) => setTimeout(resolve, 0));

function viewNamed(name: string): View {
  const i = (protocolFixtureNames as readonly string[]).indexOf(name);
  return protocolFixtures[i] as View;
}

describe('stateFrameFromView', () => {
  it('wraps a seat view as that seat in an online room', () => {
    const view = viewNamed('age-1-start-seat0');
    const f = stateFrameFromView(view, { code: 'ABCDEF', names: ['Sara', 'Omid'] });
    expect(f).toMatchObject({ t: 'state', code: 'ABCDEF', version: view.ply });
    expect(f.room.mode).toBe('online');
    expect(f.room.seats.map((s) => [s.name, s.you])).toEqual([
      ['Sara', true],
      ['Omid', false]
    ]);
    expect(f.you.seats).toEqual([0]);
    expect(f.view).toBe(view);
  });

  it('wraps a spectator view with no seats and no choices', () => {
    const f = stateFrameFromView(viewNamed('age-1-start-spectator'));
    expect(f.you).toEqual({ seats: [], canAct: false, admin: false });
    expect(f.room.spectators).toBe(1);
  });

  it('marks a finished game', () => {
    expect(stateFrameFromView(viewNamed('game-over-civilian-seat0')).room.status).toBe('finished');
  });
});

describe('the fixture registry', () => {
  it('lists the recorded state frames and every recorded view', () => {
    const list = fixtureList();
    expect(list).toContain('state-play');
    expect(list).not.toContain('welcome');
    expect(list).toContain('pick-library-token-seat1');
    expect(list.length).toBe(new Set(list).size);
    for (const name of list) expect(fixtureFrames(name), name).toHaveLength(1);
    expect(fixtureFrames('nope')).toBeUndefined();
    expect(fixtureFrames('welcome')).toBeUndefined();
  });

  it('makes the online room states from recorded views', () => {
    const ageStart = fixtureFrames('online-age-start-seat0')![0]!;
    expect(ageStart.view.age).toBe(2);
    expect(ageStart.events).toEqual([
      { kind: 'starterChosen', private: false, seat: 1, starter: 0 }
    ]);

    const lobby = fixtureFrames('online-lobby-seat0')![0]!;
    expect(lobby.room.status).toBe('lobby');
    expect(lobby.room.seats[1]?.kind).toBe('open');
    expect(lobby.you).toEqual({ seats: [0], canAct: false, admin: true });

    const away = fixtureFrames('online-away-seat0')![0]!;
    expect(away.room.seats.map((s) => s.connected)).toEqual([true, false]);
    expect(away.view.toMove).toBe(1);

    const watching = fixtureFrames('online-watching-spectator')![0]!;
    expect(watching.you.seats).toEqual([]);
    expect(watching.room.spectators).toBe(3);
  });
});

describe('FixtureTransport', () => {
  it('delivers its snapshot before simulating a dropped link', async () => {
    const frames = fixtureFrames('state-play')!;
    const transport = new FixtureTransport({ frames, linkState: 'offline' });
    const session = new Session({
      transport,
      identity: { token: 'AAAAAAAAAAAAAAAAAAAAAA', clientId: 'fixture-link', name: '' }
    });
    session.join(frames[0]!.code);
    await settle();
    expect(session.view).toEqual(frames[0]!.view);
    expect(session.conn).toBe('offline');
    expect(session.canAct).toBe(false);
    session.close();
  });
  it('opens, welcomes, and answers join with the first frame', async () => {
    const frames = fixtureFrames('state-play')!;
    const t = new FixtureTransport({ frames });
    const got: ServerMessage[] = [];
    t.onMessage((m) => got.push(m));
    expect(t.send({ t: 'resync', code: 'X' })).toBe(false); // not open yet
    await settle();
    t.send({ t: 'hello', v: 1, token: 'AAAAAAAAAAAAAAAAAAAAAA', name: '', clientId: 'c9' });
    t.send({ t: 'join', code: frames[0]!.code });
    await settle();
    expect(got.map((m) => m.t)).toEqual(['welcome', 'state']);
    expect(got[1]).toBe(frames[0]);
  });

  it('steps through a script on acts and on step(), then refuses', async () => {
    const a = stateFrameFromView(viewNamed('draft-round-1-seat0'));
    const b = stateFrameFromView(viewNamed('draft-round-2-seat0'));
    const c = stateFrameFromView(viewNamed('age-1-start-seat0'));
    const t = new FixtureTransport({ frames: [a, b, c] });
    const got: ServerMessage[] = [];
    t.onMessage((m) => got.push(m));
    await settle();
    t.send({ t: 'hello', v: 1, token: 'AAAAAAAAAAAAAAAAAAAAAA', name: '', clientId: 'c9' });
    t.send({ t: 'act', code: a.code, seq: 1, base: a.version, key: 'draft:x' });
    await settle();
    expect(got[1]).toMatchObject({
      t: 'state',
      version: b.version,
      ack: { clientId: 'c9', seq: 1 }
    });
    expect(t.step()).toBe(true);
    await settle();
    expect(got[2]).toMatchObject({ t: 'state', version: c.version });
    expect((got[2] as State).ack).toBeUndefined();
    expect(t.step()).toBe(false);
    t.send({ t: 'act', code: a.code, seq: 2, base: c.version, key: 'draft:y' });
    await settle();
    expect(got[3]).toMatchObject({ t: 'reject', seq: 2, code: 'illegal' });
  });

  it('drives a Session and a Presenter into a recorded state', async () => {
    const frames = fixtureFrames('pick-library-token-seat1')!;
    const session = new Session({
      transport: new FixtureTransport({ frames }),
      identity: { token: 'AAAAAAAAAAAAAAAAAAAAAA', clientId: 'c1', name: '' }
    });
    const presenter = new Presenter(session, { afterSwap: async () => {} });
    session.join(frames[0]!.code);
    await settle();
    await settle();
    expect(session.conn).toBe('open');
    expect(presenter.view?.phase).toBe('pickLibraryToken');
    expect(presenter.idle).toBe(true);
    session.close();
  });
});

describe('createDevRoom', () => {
  it('posts JSON and returns the code and path', async () => {
    const calls: [string, RequestInit][] = [];
    const fetcher = (async (url: string, init: RequestInit) => {
      calls.push([url, init]);
      return new Response(
        JSON.stringify({ ok: true, code: 'K7QX2M', mode: 'hotseat', path: '/play/K7QX2M' }),
        { status: 201 }
      );
    }) as typeof fetch;
    const remembered: [string, string][] = [];
    const room = await createDevRoom(
      { mode: 'hotseat', seed: '1', scenario: 'mausoleum' },
      fetcher,
      {
        remember: (code, mode) => void remembered.push([code, mode])
      }
    );
    expect(room).toEqual({ code: 'K7QX2M', path: '/play/K7QX2M' });
    // The room page then opens it without asking for a name.
    expect(remembered).toEqual([['K7QX2M', 'hotseat']]);
    expect(calls[0]?.[0]).toBe('/api/dev/rooms');
    expect(calls[0]?.[1].method).toBe('POST');
    expect(new Headers(calls[0]?.[1].headers).get('Content-Type')).toBe('application/json');
    expect(JSON.parse(String(calls[0]?.[1].body))).toEqual({
      mode: 'hotseat',
      seed: '1',
      scenario: 'mausoleum'
    });
  });

  it('throws the API error code', async () => {
    const fetcher = (async () =>
      new Response(JSON.stringify({ ok: false, code: 'unknown-scenario', msg: 'x' }), {
        status: 400
      })) as typeof fetch;
    await expect(createDevRoom({ mode: 'hotseat', scenario: 'x' }, fetcher)).rejects.toEqual(
      new DevRoomError('unknown-scenario', 400)
    );
    const notFound = (async () => new Response('nope', { status: 404 })) as typeof fetch;
    await expect(createDevRoom({ mode: 'hotseat' }, notFound)).rejects.toMatchObject({
      code: 'http-404'
    });
  });
});
