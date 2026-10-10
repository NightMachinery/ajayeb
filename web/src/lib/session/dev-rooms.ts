// POST /api/dev/rooms (docs/protocol.md, "Development endpoints"): a room
// at a chosen seed or scenario. Only a server started with --dev
// --dev-endpoints answers it, and only to a client on the same machine.

import type { BotLevel, Mode } from '$lib/wire';

export interface DevRoomRequest {
  mode: Mode;
  /** The game's 64-bit seed, as a decimal string. */
  seed?: string;
  scenario?: string;
  actions?: string[];
  bot?: BotLevel;
  /** The human's seat in an ai room. */
  seat?: 0 | 1;
}

export interface DevRoomCreated {
  code: string;
  path: string;
}

export class DevRoomError extends Error {
  constructor(
    readonly code: string,
    readonly status: number
  ) {
    super(code);
    this.name = 'DevRoomError';
  }
}

export async function createDevRoom(
  req: DevRoomRequest,
  fetcher: typeof fetch = fetch
): Promise<DevRoomCreated> {
  const res = await fetcher('/api/dev/rooms', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req)
  });
  const body: unknown = await res.json().catch(() => null);
  const data = (body && typeof body === 'object' ? body : {}) as Record<string, unknown>;
  if (!res.ok || data['ok'] !== true || typeof data['code'] !== 'string') {
    throw new DevRoomError(
      typeof data['code'] === 'string' ? data['code'] : 'http-' + res.status,
      res.status
    );
  }
  const path = typeof data['path'] === 'string' ? data['path'] : `/play/${data['code']}`;
  return { code: data['code'], path };
}
