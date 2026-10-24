// The seat-count choice follows the server's maxSeats (M18.shared-lobby): a
// server without Haftgane tables allows Duel's two seats only, so its lobby
// offers no choice, and a lobby made larger before that keeps its own count.
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import LobbyScreen from './LobbyScreen.svelte';
import { lobbyFrame } from '$lib/games/haftgane/session/test-frames';
import type { TableSession } from '$lib/games/haftgane/session/session.svelte';

function seatChoices(seatCount: number, maxSeats: number): string[] | null {
  const lobby = lobbyFrame({
    seatCount,
    maxSeats,
    seats: Array.from({ length: seatCount }, (_, seat) =>
      seat === 0 ? { seat, kind: 'human' as const, userId: 'u0' } : { seat, kind: 'open' as const }
    )
  });
  const session = {
    lobby,
    conn: 'open',
    pending: null,
    notice: null,
    admin: true,
    canStart: false,
    me: { creator: true, admin: true }
  } as unknown as TableSession;
  const body = render(LobbyScreen, { props: { session } }).body;
  const options = /<select[^>]*data-act="lobby-seats"[^>]*>([\s\S]*?)<\/select>/.exec(body)?.[1];
  return options === undefined
    ? null
    : [...options.matchAll(/<option value="(\d)"/g)].map((m) => m[1] ?? '');
}

describe('lobby seat count', () => {
  it('offers no choice when the server allows two seats only', () => {
    expect(seatChoices(2, 2)).toBeNull();
  });

  it('offers every count the server allows', () => {
    expect(seatChoices(2, 7)).toEqual(['2', '3', '4', '5', '6', '7']);
  });

  it('keeps a larger lobby’s own count among the choices', () => {
    expect(seatChoices(4, 2)).toEqual(['2', '4']);
  });
});
