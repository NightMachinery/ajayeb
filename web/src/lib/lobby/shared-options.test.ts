import { expect, it } from 'vitest';
import { defaultHints } from '$lib/hints/families';
import { isLobbyState } from '$lib/games/haftgane/session/frames';
import { lobbyFrame } from '$lib/games/haftgane/session/test-frames';
it('validates Duel lobby hints and starter values without interpreting game rules', () => {
  const frame = lobbyFrame();
  frame.seatCount = 2;
  frame.options = { firstPlayer: 1, hints: defaultHints('online') };
  expect(isLobbyState(frame)).toBe(true);
  for (const invalid of [
    null,
    { ...frame.options.hints, prices: 'yes' },
    { ...frame.options.hints, advisorStrength: 'unbounded' }
  ])
    expect(isLobbyState({ ...frame, options: { ...frame.options, hints: invalid } })).toBe(false);
  expect(isLobbyState({ ...frame, options: { ...frame.options, firstPlayer: 2 } })).toBe(false);
});
