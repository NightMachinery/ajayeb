import { describe, expect, it } from 'vitest';
import type { SeatInfo, State } from '$lib/wire';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import {
  CODE_ALPHABET,
  inviteUrl,
  isRoomCode,
  needsName,
  normalizeCodeInput,
  presenceChanges,
  qrPath,
  seatPresence,
  suggestName
} from './online';

/** The recorded play frame as an online room seen from seat 0. */
function onlineFrame(seats: Partial<SeatInfo>[], code = 'K7QX2M'): State {
  const i = (fixtureNames as readonly string[]).indexOf('state-play');
  const frame = structuredClone(fixtures[i]) as State;
  frame.code = code;
  frame.room = {
    ...frame.room,
    mode: 'online',
    seats: [0, 1].map((seat) => ({
      seat,
      name: seat === 0 ? 'Sara' : 'Omid',
      kind: 'human' as const,
      connected: true,
      you: seat === 0,
      ...seats[seat]
    }))
  };
  return frame;
}

describe('normalizeCodeInput', () => {
  it('upper-cases and drops spaces and punctuation', () => {
    expect(normalizeCodeInput(' k7q-x2m ')).toBe('K7QX2M');
    expect(normalizeCodeInput('k7q x2m')).toBe('K7QX2M');
  });

  it('cuts to six characters', () => {
    expect(normalizeCodeInput('K7QX2MZZ')).toBe('K7QX2M');
  });

  it('takes the code out of a pasted invite link', () => {
    expect(normalizeCodeInput('https://example.org/play/k7qx2m')).toBe('K7QX2M');
    expect(normalizeCodeInput('https://example.org/play/K7QX2M/?x=1#y')).toBe('K7QX2M');
    expect(normalizeCodeInput('example.org/play/K7QX2M?ref=chat')).toBe('K7QX2M');
  });

  it('keeps letters outside the alphabet so the field shows what was typed', () => {
    expect(normalizeCodeInput('o0il1a')).toBe('O0IL1A');
  });

  it('drops letters outside ASCII', () => {
    expect(normalizeCodeInput('k7é')).toBe('K7');
  });
});

describe('isRoomCode', () => {
  it('accepts six characters of the alphabet', () => {
    expect(isRoomCode('K7QX2M')).toBe(true);
    expect(isRoomCode(CODE_ALPHABET.slice(0, 6))).toBe(true);
  });

  it('refuses other lengths, lower case and the confusable characters', () => {
    expect(isRoomCode('K7QX2')).toBe(false);
    expect(isRoomCode('K7QX2MM')).toBe(false);
    expect(isRoomCode('k7qx2m')).toBe(false);
    for (const ch of ['I', 'L', 'O', '0', '1']) expect(isRoomCode(`K7QX2${ch}`)).toBe(false);
  });
});

describe('inviteUrl', () => {
  it('joins the origin and the room path', () => {
    expect(inviteUrl('https://example.org', '/play/K7QX2M')).toBe(
      'https://example.org/play/K7QX2M'
    );
  });
});

describe('seatPresence', () => {
  it('reads each human seat of an online room', () => {
    const f = onlineFrame([{}, { connected: false }]);
    expect(seatPresence(f.room)).toEqual(['connected', 'away']);
  });

  it('says nothing about an open seat, a bot, or another mode', () => {
    expect(seatPresence(onlineFrame([{}, { kind: 'open', connected: false }]).room)).toEqual([
      'connected',
      undefined
    ]);
    const ai = onlineFrame([{}, { kind: 'bot', bot: 'easy' }]).room;
    expect(seatPresence({ ...ai, mode: 'ai' })).toEqual([undefined, undefined]);
    expect(seatPresence(null)).toEqual([undefined, undefined]);
  });
});

describe('presenceChanges', () => {
  it('announces the friend taking the open seat', () => {
    const lobby = onlineFrame([{}, { kind: 'open', connected: false, name: '' }]);
    lobby.room.status = 'lobby';
    const playing = onlineFrame([{}, {}]);
    expect(presenceChanges(lobby, playing)).toEqual([{ kind: 'joined', seat: 1 }]);
  });

  it('announces the other player leaving and coming back', () => {
    const here = onlineFrame([{}, {}]);
    const gone = onlineFrame([{}, { connected: false }]);
    expect(presenceChanges(here, gone)).toEqual([{ kind: 'left', seat: 1 }]);
    expect(presenceChanges(gone, here)).toEqual([{ kind: 'back', seat: 1 }]);
    expect(presenceChanges(here, here)).toEqual([]);
  });

  it('never announces this connection itself', () => {
    const a = onlineFrame([{ connected: false }, {}]);
    const b = onlineFrame([{}, {}]);
    expect(presenceChanges(a, b)).toEqual([]);
  });

  it('announces both seats to a spectator', () => {
    const a = onlineFrame([{ you: false }, { you: false }]);
    const b = onlineFrame([
      { you: false, connected: false },
      { you: false, connected: false }
    ]);
    expect(presenceChanges(a, b)).toEqual([
      { kind: 'left', seat: 0 },
      { kind: 'left', seat: 1 }
    ]);
  });

  it('says nothing on the first frame, across rooms or outside online rooms', () => {
    const here = onlineFrame([{}, {}]);
    const gone = onlineFrame([{}, { connected: false }]);
    expect(presenceChanges(null, gone)).toEqual([]);
    expect(presenceChanges(onlineFrame([{}, {}], 'AAAAAA'), gone)).toEqual([]);
    expect(
      presenceChanges(
        { ...here, room: { ...here.room, mode: 'hotseat' } },
        { ...gone, room: { ...gone.room, mode: 'hotseat' } }
      )
    ).toEqual([]);
  });
});

describe('qrPath', () => {
  it('draws each horizontal run of dark modules as one rectangle', () => {
    expect(
      qrPath([
        [true, true, false, true],
        [false, false, false, false],
        [false, true, true, true]
      ])
    ).toBe('M0 0h2v1h-2zM3 0h1v1h-1zM1 2h3v1h-3z');
  });

  it('draws nothing for an empty code', () => {
    expect(qrPath([])).toBe('');
    expect(qrPath([[false, false]])).toBe('');
  });
});

describe('suggestName', () => {
  it('picks one name of the list by the random number', () => {
    expect(suggestName('Kaveh, Roxana,Darius', 0)).toBe('Kaveh');
    expect(suggestName('Kaveh, Roxana,Darius', 0.5)).toBe('Roxana');
    expect(suggestName('Kaveh, Roxana,Darius', 0.999)).toBe('Darius');
    expect(suggestName('Kaveh,Roxana', 1)).toBe('Roxana');
  });

  it('is empty for an empty list', () => {
    expect(suggestName('', 0.3)).toBe('');
    expect(suggestName(' , ', 0.3)).toBe('');
  });
});

describe('needsName', () => {
  const rooms = [{ code: 'K7QX2M' }, { code: 'H8TQ4R' }];

  it('asks a first-time visitor for a name before an unknown room', () => {
    expect(needsName(false, 'B4TQ7N', rooms)).toBe(true);
    expect(needsName(false, 'B4TQ7N', [])).toBe(true);
  });

  it('never asks for a room this browser made or played', () => {
    expect(needsName(false, 'K7QX2M', rooms)).toBe(false);
  });

  it('never asks a player who has a name', () => {
    expect(needsName(true, 'B4TQ7N', [])).toBe(false);
  });
});
