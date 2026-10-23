// Creates a shared lobby (docs/protocol.md, "lobby-create") over a short v2
// connection of its own, and resolves with its code. The creator holds the
// lobby's first slot by its user, not by this connection, so the page then
// opens /play/<code> and joins it through the ordinary entry. The real
// New table sheet loads this helper only when the player creates a table.

import { entryIdentity, rememberEntryGame, rememberEntryName } from '$lib/entry/identity';
import { defaultSocketUrl, type SocketLike } from '$lib/session/ws-transport';
import type { LobbyCreate, LobbyOptions } from '$lib/wire/platform.gen';

export class CreateLobbyError extends Error {
  constructor(readonly code: string) {
    super(code);
    this.name = 'CreateLobbyError';
  }
}

export interface CreateLobbyOptions {
  /** Submitted before hello, including when storage is unavailable. */
  name?: string;
  socket?: () => SocketLike;
  timeoutMs?: number;
}

export function createLobby(
  seatCount: number,
  options: LobbyOptions = {},
  opts: CreateLobbyOptions = {}
): Promise<string> {
  if (opts.name) rememberEntryName(opts.name);
  return new Promise<string>((resolve, reject) => {
    let socket: SocketLike;
    try {
      socket = opts.socket?.() ?? new WebSocket(defaultSocketUrl());
    } catch {
      reject(new CreateLobbyError('offline'));
      return;
    }
    let done = false;
    const finish = (code: string | null, failure?: string) => {
      if (done) return;
      done = true;
      clearTimeout(timer);
      socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null;
      try {
        socket.close(1000);
      } catch {
        /* closing a connecting socket can throw */
      }
      if (code) {
        // A room this page created is entered as a lobby at once, without
        // the ordinary Duel path first (D61).
        rememberEntryGame(code, 'lobby');
        resolve(code);
      } else reject(new CreateLobbyError(failure ?? 'offline'));
    };
    const timer = setTimeout(() => finish(null, 'timeout'), opts.timeoutMs ?? 10_000);
    socket.onopen = () => socket.send(JSON.stringify({ t: 'hello', v: 2, ...entryIdentity() }));
    socket.onmessage = (event) => {
      let m: Record<string, unknown>;
      try {
        m = JSON.parse(String(event.data)) as Record<string, unknown>;
      } catch {
        return;
      }
      if (m['t'] === 'welcome') {
        if (m['v'] !== 2) return finish(null, 'protocol');
        const frame: LobbyCreate = { t: 'lobby-create', seatCount, options };
        socket.send(JSON.stringify(frame));
      } else if (m['t'] === 'created' && typeof m['code'] === 'string') {
        finish(m['code']);
      } else if (
        (m['t'] === 'lobby-reject' || m['t'] === 'reject' || m['t'] === 'error') &&
        typeof m['code'] === 'string'
      ) {
        finish(null, m['code']);
      }
    };
    socket.onerror = () => {}; // close follows
    socket.onclose = () => finish(null, 'offline');
  });
}
