import { afterEach, expect, it, vi } from 'vitest';
import { createLobby } from '$lib/games/haftgane/session/create';
import type { SocketLike } from '$lib/session/ws-transport';
class Socket implements SocketLike {
  readonly readyState = 1;
  onopen: SocketLike['onopen'] = null;
  onmessage: SocketLike['onmessage'] = null;
  onerror: SocketLike['onerror'] = null;
  onclose: SocketLike['onclose'] = null;
  sent: Record<string, unknown>[] = [];
  closed = false;
  send(frame: string) {
    this.sent.push(JSON.parse(frame));
  }
  close() {
    this.closed = true;
  }
  message(frame: object) {
    this.onmessage?.({ data: JSON.stringify(frame) } as MessageEvent);
  }
}
afterEach(() => vi.unstubAllGlobals());
it('puts the submitted name in hello before creating, even when storage refuses access', async () => {
  vi.stubGlobal('localStorage', {
    getItem() {
      throw Error('blocked');
    },
    setItem() {
      throw Error('blocked');
    }
  });
  const socket = new Socket();
  const created = createLobby(4, {}, { socket: () => socket, name: 'Invited' });
  socket.onopen?.({} as Event);
  expect(socket.sent).toHaveLength(1);
  expect(socket.sent[0]).toMatchObject({ t: 'hello', v: 2, name: 'Invited' });
  socket.message({ t: 'welcome', v: 2 });
  expect(socket.sent[1]).toEqual({ t: 'lobby-create', seatCount: 4, options: {} });
  socket.message({ t: 'created', code: 'NEWAAA' });
  await expect(created).resolves.toBe('NEWAAA');
  expect(socket.closed).toBe(true);
  expect(socket.onmessage).toBeNull();
});
it('refuses an incompatible welcome before sending lobby-create', async () => {
  const socket = new Socket();
  const created = createLobby(4, {}, { socket: () => socket });
  socket.onopen?.({} as Event);
  socket.message({ t: 'welcome', v: 1 });
  await expect(created).rejects.toMatchObject({ code: 'protocol' });
  expect(socket.sent.map((frame) => frame.t)).toEqual(['hello']);
  expect(socket.closed).toBe(true);
});
